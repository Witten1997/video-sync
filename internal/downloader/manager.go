package downloader

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"bili-download/internal/bilibili"
	"bili-download/internal/config"
	"bili-download/internal/database/models"
	"bili-download/internal/utils"

	"gorm.io/gorm"
)

// ManagerEventType 管理器事件类型
type ManagerEventType string

const (
	EventTaskAdded     ManagerEventType = "task_added"
	EventTaskStarted   ManagerEventType = "task_started"
	EventTaskCompleted ManagerEventType = "task_completed"
	EventTaskFailed    ManagerEventType = "task_failed"
	EventTaskCancelled ManagerEventType = "task_cancelled"
	EventTaskRetrying  ManagerEventType = "task_retrying"
	EventTaskProgress  ManagerEventType = "task_progress"
	EventRecordCreated ManagerEventType = "download_record_created"
)

// ManagerEvent 管理器事件
type ManagerEvent struct {
	Type      ManagerEventType       `json:"type"`
	Task      *DownloadTask          `json:"task"`
	Progress  *SubTaskProgress       `json:"progress,omitempty"`
	Message   string                 `json:"message,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Record    *models.DownloadRecord `json:"record,omitempty"`
}

// EventHandler 事件处理器
type EventHandler func(event ManagerEvent)

// DownloadManager 下载管理器
type DownloadManager struct {
	config             *config.Config
	db                 *gorm.DB
	biliClient         *bilibili.Client
	downloader         *Downloader
	queue              *TaskQueue
	concurrency        *ConcurrencyController
	tracker            *ProgressTracker
	runningTasks       sync.Map // taskID -> *DownloadTask
	completedTasks     sync.Map // taskID -> *DownloadTask
	eventHandlers      []EventHandler
	ctx                context.Context
	cancel             context.CancelFunc
	wg                 sync.WaitGroup
	mu                 sync.RWMutex
	running            bool
	lastProgressUpdate sync.Map // videoID -> time.Time (进度更新节流)
}

// NewDownloadManager 创建新的下载管理器
func NewDownloadManager(cfg *config.Config, db *gorm.DB, biliClient *bilibili.Client) (*DownloadManager, error) {
	// 创建下载器
	downloader, err := NewDownloader(cfg, biliClient)
	if err != nil {
		return nil, fmt.Errorf("创建下载器失败: %w", err)
	}

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 创建并发控制器
	maxVideos := cfg.Advanced.ConcurrentLimit.Video
	maxPages := cfg.Advanced.ConcurrentLimit.Page
	if maxVideos <= 0 {
		maxVideos = 2
	}
	if maxPages <= 0 {
		maxPages = 4
	}

	manager := &DownloadManager{
		config:        cfg,
		db:            db,
		biliClient:    biliClient,
		downloader:    downloader,
		queue:         NewTaskQueue(),
		concurrency:   NewConcurrencyController(maxVideos, maxPages),
		tracker:       downloader.GetTracker(),
		eventHandlers: make([]EventHandler, 0),
		ctx:           ctx,
		cancel:        cancel,
		running:       false,
	}

	// 设置进度回调
	downloader.SetProgressCallback(func(videoID uint, pid int, taskName string, progress *SubTaskProgress) {
		// 查找对应的任务
		var task *DownloadTask
		manager.runningTasks.Range(func(key, value interface{}) bool {
			t := value.(*DownloadTask)
			if t.Video.ID == videoID {
				task = t
				return false
			}
			return true
		})

		if task != nil {
			manager.emitEvent(ManagerEvent{
				Type:      EventTaskProgress,
				Task:      task,
				Progress:  progress,
				Timestamp: time.Now(),
			})
		}

		// 更新下载记录
		if manager.db != nil {
			manager.updateDownloadRecordProgress(videoID, taskName, progress)
		}
	})

	return manager, nil
}

// Start 启动管理器
func (dm *DownloadManager) Start() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if dm.running {
		return fmt.Errorf("管理器已在运行")
	}

	dm.running = true
	utils.Info("下载管理器已启动")

	// 启动调度器
	dm.wg.Add(1)
	go dm.scheduler()

	return nil
}

// Stop 停止管理器
func (dm *DownloadManager) Stop() error {
	dm.mu.Lock()
	if !dm.running {
		dm.mu.Unlock()
		return fmt.Errorf("管理器未运行")
	}

	dm.running = false
	dm.mu.Unlock()

	utils.Info("正在停止下载管理器...")

	// 取消所有正在运行的任务
	dm.runningTasks.Range(func(key, value interface{}) bool {
		task := value.(*DownloadTask)
		task.Cancel()
		return true
	})

	// 取消上下文
	dm.cancel()

	// 等待所有 goroutine 结束
	dm.wg.Wait()

	// 清理下载��资源
	dm.downloader.Cleanup()

	utils.Info("下载管理器已停止")
	return nil
}

// scheduler 任务调度器
func (dm *DownloadManager) scheduler() {
	defer dm.wg.Done()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-dm.ctx.Done():
			return
		case <-ticker.C:
			dm.scheduleNextTask()
		}
	}
}

// scheduleNextTask 调度下一个任务
func (dm *DownloadManager) scheduleNextTask() {
	// 检查是否可以启动新任务
	if !dm.concurrency.CanStartVideo() && !dm.concurrency.CanStartPage() {
		return
	}

	// 从队列中获取任务
	task := dm.queue.Dequeue()
	if task == nil {
		return
	}

	// 检查任务是否已取消
	if task.IsCancelled() {
		task.SetStatus(TaskStatusCancelled)
		dm.emitEvent(ManagerEvent{
			Type:      EventTaskCancelled,
			Task:      task,
			Timestamp: time.Now(),
		})
		return
	}

	// 根据任务类型选择并发控制
	switch task.Type {
	case TaskTypeVideo:
		if dm.concurrency.CanStartVideo() {
			dm.wg.Add(1)
			go dm.executeVideoTask(task)
		} else {
			// 放回队列
			dm.queue.Enqueue(task)
		}
	case TaskTypePage:
		if dm.concurrency.CanStartPage() {
			dm.wg.Add(1)
			go dm.executePageTask(task)
		} else {
			// 放回队列
			dm.queue.Enqueue(task)
		}
	default:
		utils.Warn("未知任务类型: %s", task.Type)
	}
}

// executeVideoTask 执行视频任务（下载所有分P）
func (dm *DownloadManager) executeVideoTask(task *DownloadTask) {
	defer dm.wg.Done()

	// 获取视频级别许可
	if err := dm.concurrency.AcquireVideo(task.Context); err != nil {
		task.SetError(err)
		task.SetStatus(TaskStatusCancelled)
		return
	}
	defer dm.concurrency.ReleaseVideo()

	// 记录运行中的任务
	dm.runningTasks.Store(task.ID, task)
	defer func() {
		dm.runningTasks.Delete(task.ID)
		dm.completedTasks.Store(task.ID, task)
	}()

	// 更新状态
	task.SetStatus(TaskStatusRunning)
	dm.emitEvent(ManagerEvent{
		Type:      EventTaskStarted,
		Task:      task,
		Timestamp: time.Now(),
	})

	// 下载所有分P
	video := task.Video
	utils.Info("开始下载视频: %s (BV%s), Pages数量: %d", video.Name, video.BVid, len(video.Pages))

	for _, page := range video.Pages {
		utils.Info("准备下载分P: %s - P%d (%s)", video.Name, page.PID, page.Name)
		// 检查是否取消
		if task.IsCancelled() {
			task.SetStatus(TaskStatusCancelled)
			dm.emitEvent(ManagerEvent{
				Type:      EventTaskCancelled,
				Task:      task,
				Timestamp: time.Now(),
			})
			return
		}

		// 获取分P级别许可
		if err := dm.concurrency.AcquirePage(task.Context); err != nil {
			task.SetError(err)
			task.SetStatus(TaskStatusFailed)
			dm.handleTaskFailure(task)
			// 更新下载记录为失败
			if dm.db != nil && task.RecordID > 0 {
				now := time.Now()
				dm.db.Model(&models.DownloadRecord{}).Where("id = ?", task.RecordID).
					Updates(map[string]interface{}{
						"status":        "failed",
						"error_message": err.Error(),
						"completed_at":  now,
					})
			}
			return
		}

		// 下载分P到视频文件夹（task.OutputDir已经是视频专属文件夹）
		err := dm.downloader.DownloadPage(task.Context, video, &page, task.OutputDir)
		dm.concurrency.ReleasePage()

		if err != nil {
			utils.Error("下载分P失败: %v", err)
			task.SetError(err)
			task.SetStatus(TaskStatusFailed)
			dm.handleTaskFailure(task)
			// 更新下载记录为失败
			if dm.db != nil && task.RecordID > 0 {
				now := time.Now()
				dm.db.Model(&models.DownloadRecord{}).Where("id = ?", task.RecordID).
					Updates(map[string]interface{}{
						"status":        "failed",
						"error_message": err.Error(),
						"completed_at":  now,
					})
			}
			return
		}
	}

	// 任务完成
	task.SetStatus(TaskStatusCompleted)

	// 更新下载记录为完成
	finalStatus := "completed"
	if dm.db != nil && task.RecordID > 0 {
		now := time.Now()
		var record models.DownloadRecord
		if dm.db.First(&record, task.RecordID).Error == nil {
			var details models.FileDetailsData
			if json.Unmarshal(record.FileDetails, &details) == nil {
				hasFailed := false
				for i := range details.Files {
					if details.Files[i].Status == "pending" || details.Files[i].Status == "downloading" {
						details.Files[i].Status = "failed"
						hasFailed = true
					}
					if details.Files[i].Status == "failed" {
						hasFailed = true
					}
				}
				if hasFailed {
					finalStatus = "failed"
				}
				if detailsJSON, err := json.Marshal(details); err == nil {
					dm.db.Model(&record).Updates(map[string]interface{}{
						"file_details":  detailsJSON,
						"status":        finalStatus,
						"completed_at":  now,
						"error_message": map[bool]string{true: "部分文件下载失败", false: ""}[hasFailed],
					})
				}
			}
		}
	}

	// 只有全部下载成功才更新视频下载状态
	if dm.db != nil && finalStatus == "completed" {
		if err := dm.db.Model(&models.Video{}).Where("id = ?", video.ID).Update("download_status", 1).Error; err != nil {
			utils.Warn("更新视频下载状态失败: %v", err)
		} else {
			utils.Info("已更新视频 [%s] 的下载状态", video.Name)
		}

		// 更新所有分P的下载状态
		for _, page := range video.Pages {
			if err := dm.db.Model(&models.Page{}).Where("id = ?", page.ID).Update("download_status", 1).Error; err != nil {
				utils.Warn("更新分P下载状态失败: %v", err)
			}
		}
	}

	dm.emitEvent(ManagerEvent{
		Type:      EventTaskCompleted,
		Task:      task,
		Message:   fmt.Sprintf("视频下载完成: %s", video.Name),
		Timestamp: time.Now(),
	})

	utils.Info("视频任务完成: %s [%s]", video.Name, task.ID)
}

// executePageTask 执行分P任务
func (dm *DownloadManager) executePageTask(task *DownloadTask) {
	defer dm.wg.Done()

	// 获取分P级别许可
	if err := dm.concurrency.AcquirePage(task.Context); err != nil {
		task.SetError(err)
		task.SetStatus(TaskStatusCancelled)
		return
	}
	defer dm.concurrency.ReleasePage()

	// 记录运行中的任务
	dm.runningTasks.Store(task.ID, task)
	defer func() {
		dm.runningTasks.Delete(task.ID)
		dm.completedTasks.Store(task.ID, task)
	}()

	// 更新状态
	task.SetStatus(TaskStatusRunning)
	dm.emitEvent(ManagerEvent{
		Type:      EventTaskStarted,
		Task:      task,
		Timestamp: time.Now(),
	})

	// 下载分P到视频文件夹（task.OutputDir已经是视频专属文件夹）
	err := dm.downloader.DownloadPage(task.Context, task.Video, task.Page, task.OutputDir)
	if err != nil {
		utils.Error("下载分P失败: %v", err)
		task.SetError(err)
		task.SetStatus(TaskStatusFailed)
		dm.handleTaskFailure(task)
		return
	}

	// 任务完成
	task.SetStatus(TaskStatusCompleted)
	dm.emitEvent(ManagerEvent{
		Type:      EventTaskCompleted,
		Task:      task,
		Message:   fmt.Sprintf("分P下载完成: %s - P%d", task.Video.Name, task.Page.PID),
		Timestamp: time.Now(),
	})

	utils.Info("分P任务完成: %s P%d [%s]", task.Video.Name, task.Page.PID, task.ID)
}

// handleTaskFailure 处理任务失败
func (dm *DownloadManager) handleTaskFailure(task *DownloadTask) {
	// 检查是否可以重试
	if task.CanRetry() {
		task.IncrementRetry()
		utils.Info("任务将重试: %s (第 %d/%d 次)", task.ID, task.RetryCount, task.MaxRetries)

		// 克隆任务并重新入队
		newTask := task.Clone()
		newTask.SetStatus(TaskStatusPending)
		dm.queue.Enqueue(newTask)

		dm.emitEvent(ManagerEvent{
			Type:      EventTaskRetrying,
			Task:      newTask,
			Message:   fmt.Sprintf("重试 %d/%d", newTask.RetryCount, newTask.MaxRetries),
			Timestamp: time.Now(),
		})
	} else {
		// 无法重试，标记为失败
		dm.emitEvent(ManagerEvent{
			Type:      EventTaskFailed,
			Task:      task,
			Message:   task.GetError().Error(),
			Timestamp: time.Now(),
		})
		utils.Error("任务失败: %s, 错误: %v", task.ID, task.GetError())
	}
}

// AddTask 添加任务
func (dm *DownloadManager) AddTask(task *DownloadTask) error {
	if task == nil {
		return fmt.Errorf("任务不能为空")
	}

	// 检查任务是否已存在
	if dm.queue.Contains(task.ID) {
		return fmt.Errorf("任务已存在: %s", task.ID)
	}

	// 入队
	dm.queue.Enqueue(task)

	dm.emitEvent(ManagerEvent{
		Type:      EventTaskAdded,
		Task:      task,
		Timestamp: time.Now(),
	})

	utils.Info("任务已添加: %s (%s)", task.ID, task.Type)
	return nil
}

// PrepareAndAddVideoTask 准备并添加视频任务（统一的下载方法）
// baseDir: 基础目录（URL下载使用DownloadBase，定时任务使用视频源Path）
// autoCreateFolder: 是否自动为视频创建专属文件夹（通常为true）
func (dm *DownloadManager) PrepareAndAddVideoTask(video *models.Video, baseDir string, priority TaskPriority, autoCreateFolder bool) (*DownloadTask, error) {
	var outputDir string

	if autoCreateFolder {
		// 为视频创建专属文件夹
		videoFolderName := utils.Filenamify(video.Name)
		outputDir = filepath.Join(baseDir, videoFolderName)

		// 创建目录
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return nil, fmt.Errorf("创建视频目录失败: %w", err)
		}

		// 更新数据库中的视频路径
		if dm.db != nil {
			video.Path = outputDir
			if err := dm.db.Model(video).Update("path", outputDir).Error; err != nil {
				utils.Warn("更新视频路径失败: %v", err)
			}
		}
	} else {
		outputDir = baseDir
	}

	// 从数据库重新加载完整的视频数据（包括Pages）
	var videoWithPages models.Video
	if dm.db != nil {
		if err := dm.db.Preload("Pages").First(&videoWithPages, video.ID).Error; err != nil {
			return nil, fmt.Errorf("加载视频数据失败: %w", err)
		}
	} else {
		videoWithPages = *video
	}

	// 创建下载任务
	task := NewDownloadTask(TaskTypeVideo, &videoWithPages, nil, outputDir)
	task.Priority = priority
	task.MaxRetries = dm.getMaxRetries()
	if dm.db != nil {
		fileDetails := dm.buildFileDetails(&videoWithPages)
		detailsJSON, _ := json.Marshal(fileDetails)
		sourceType, sourceID, sourceName := dm.getVideoSourceInfo(&videoWithPages)

		record := &models.DownloadRecord{
			VideoID:     videoWithPages.ID,
			SourceType:  sourceType,
			SourceID:    sourceID,
			SourceName:  sourceName,
			Status:      "pending",
			FileDetails: detailsJSON,
		}
		if err := dm.db.Create(record).Error; err != nil {
			utils.Warn("创建下载记录失败: %v", err)
		} else {
			task.RecordID = record.ID
			record.Video = videoWithPages
			dm.emitEvent(ManagerEvent{
				Type:      EventRecordCreated,
				Task:      task,
				Record:    record,
				Timestamp: time.Now(),
			})
		}
	}

	if err := dm.AddTask(task); err != nil {
		return nil, err
	}

	utils.Info("已为视频 [%s] 创建下载任务，输出目录: %s，Pages数量: %d", videoWithPages.Name, outputDir, len(videoWithPages.Pages))
	utils.Debug("视频Path字段已更新为: %s", video.Path)
	return task, nil
}

// RetryVideoTask 基于已有下载记录重试视频下载（不创建新记录）
func (dm *DownloadManager) RetryVideoTask(recordID uint, video *models.Video, priority TaskPriority) (*DownloadTask, error) {
	// 从数据库重新加载完整的视频数据（包括Pages）
	var videoWithPages models.Video
	if dm.db != nil {
		if err := dm.db.Preload("Pages").First(&videoWithPages, video.ID).Error; err != nil {
			return nil, fmt.Errorf("加载视频数据失败: %w", err)
		}
	} else {
		videoWithPages = *video
	}

	// 如果没有Pages，尝试从B站API重新获取
	if len(videoWithPages.Pages) == 0 {
		pages, err := dm.biliClient.GetVideoPages(videoWithPages.BVid)
		if err != nil {
			return nil, fmt.Errorf("视频没有分P信息且无法从B站获取: %w", err)
		}
		for _, p := range pages {
			page := models.Page{
				VideoID:  videoWithPages.ID,
				CID:      p.CID,
				PID:      p.Page,
				Name:     p.Part,
				Duration: p.Duration,
				Width:    p.Dimension.Width,
				Height:   p.Dimension.Height,
				Image:    p.FirstFrame,
			}
			if err := dm.db.Create(&page).Error; err != nil {
				utils.Warn("创建分P记录失败: %v", err)
				continue
			}
			videoWithPages.Pages = append(videoWithPages.Pages, page)
		}
		if len(videoWithPages.Pages) == 0 {
			return nil, fmt.Errorf("视频没有分P信息")
		}
	}

	outputDir := videoWithPages.Path
	if outputDir == "" {
		return nil, fmt.Errorf("视频下载路径为空")
	}

	// 确保目录存在
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("创建视频目录失败: %w", err)
	}

	task := NewDownloadTask(TaskTypeVideo, &videoWithPages, nil, outputDir)
	task.Priority = priority
	task.RecordID = recordID
	task.MaxRetries = dm.getMaxRetries()

	if err := dm.AddTask(task); err != nil {
		return nil, err
	}

	utils.Info("已为视频 [%s] 创建重试任务，输出目录: %s，Pages数量: %d", videoWithPages.Name, outputDir, len(videoWithPages.Pages))
	return task, nil
}

// AddVideoTask 添加视频任务（保留向后兼容）
func (dm *DownloadManager) AddVideoTask(video *models.Video, outputDir string, priority TaskPriority) (*DownloadTask, error) {
	task := NewDownloadTask(TaskTypeVideo, video, nil, outputDir)
	task.Priority = priority

	if err := dm.AddTask(task); err != nil {
		return nil, err
	}

	return task, nil
}

// AddPageTask 添加分P任务
func (dm *DownloadManager) AddPageTask(video *models.Video, page *models.Page, outputDir string, priority TaskPriority) (*DownloadTask, error) {
	task := NewDownloadTask(TaskTypePage, video, page, outputDir)
	task.Priority = priority

	if err := dm.AddTask(task); err != nil {
		return nil, err
	}

	return task, nil
}

// CancelTask 取消任务
func (dm *DownloadManager) CancelTask(taskID string) error {
	// 先从队列中移除
	task := dm.queue.Remove(taskID)
	if task != nil {
		task.Cancel()
		dm.emitEvent(ManagerEvent{
			Type:      EventTaskCancelled,
			Task:      task,
			Timestamp: time.Now(),
		})
		return nil
	}

	// 检查是否在运行中
	if val, ok := dm.runningTasks.Load(taskID); ok {
		task := val.(*DownloadTask)
		task.Cancel()
		return nil
	}

	return fmt.Errorf("任务未找到: %s", taskID)
}

// PauseTask 暂停任务（暂不支持，预留接口）
func (dm *DownloadManager) PauseTask(taskID string) error {
	return fmt.Errorf("暂停功能暂未实现")
}

// ResumeTask 恢复任务（暂不支持，预留接口）
func (dm *DownloadManager) ResumeTask(taskID string) error {
	return fmt.Errorf("恢复功能暂未实现")
}

// RetryTask 重试任务
func (dm *DownloadManager) RetryTask(taskID string) error {
	// 从已完成任务中查找
	val, ok := dm.completedTasks.Load(taskID)
	if !ok {
		return fmt.Errorf("任务未找到: %s", taskID)
	}

	oldTask := val.(*DownloadTask)
	if oldTask.GetStatus() != TaskStatusFailed {
		return fmt.Errorf("只能重试失败的任务")
	}

	// 创建新任务
	newTask := oldTask.Clone()
	newTask.SetStatus(TaskStatusPending)
	newTask.RetryCount = 0 // 重置重试次数

	return dm.AddTask(newTask)
}

// GetTask 获取任务信息
func (dm *DownloadManager) GetTask(taskID string) *DownloadTask {
	// 先查队列
	if task := dm.queue.Get(taskID); task != nil {
		return task
	}

	// 再查运行中
	if val, ok := dm.runningTasks.Load(taskID); ok {
		return val.(*DownloadTask)
	}

	// 最后查已完成
	if val, ok := dm.completedTasks.Load(taskID); ok {
		return val.(*DownloadTask)
	}

	return nil
}

// GetAllTasks 获取所有任务
func (dm *DownloadManager) GetAllTasks() []*DownloadTask {
	tasks := make([]*DownloadTask, 0)

	// 队列中的任务
	tasks = append(tasks, dm.queue.GetAll()...)

	// 运行中的任务
	dm.runningTasks.Range(func(key, value interface{}) bool {
		tasks = append(tasks, value.(*DownloadTask))
		return true
	})

	// 已完成的任务
	dm.completedTasks.Range(func(key, value interface{}) bool {
		tasks = append(tasks, value.(*DownloadTask))
		return true
	})

	return tasks
}

// GetQueuedTasks 获取队列中的任务
func (dm *DownloadManager) GetQueuedTasks() []*DownloadTask {
	return dm.queue.GetAll()
}

// GetRunningTasks 获取正在运行的任务
func (dm *DownloadManager) GetRunningTasks() []*DownloadTask {
	tasks := make([]*DownloadTask, 0)
	dm.runningTasks.Range(func(key, value interface{}) bool {
		tasks = append(tasks, value.(*DownloadTask))
		return true
	})
	return tasks
}

// GetCompletedTasks 获取已完成的任务
func (dm *DownloadManager) GetCompletedTasks() []*DownloadTask {
	tasks := make([]*DownloadTask, 0)
	dm.completedTasks.Range(func(key, value interface{}) bool {
		tasks = append(tasks, value.(*DownloadTask))
		return true
	})
	return tasks
}

// GetStats 获取统计信息
func (dm *DownloadManager) GetStats() ManagerStats {
	queuedCount := dm.queue.Size()

	runningCount := 0
	dm.runningTasks.Range(func(key, value interface{}) bool {
		runningCount++
		return true
	})

	completedCount := 0
	failedCount := 0
	dm.completedTasks.Range(func(key, value interface{}) bool {
		task := value.(*DownloadTask)
		if task.GetStatus() == TaskStatusCompleted {
			completedCount++
		} else if task.GetStatus() == TaskStatusFailed {
			failedCount++
		}
		return true
	})

	concurrencyStats := dm.concurrency.GetStats()

	return ManagerStats{
		QueuedTasks:    queuedCount,
		RunningTasks:   runningCount,
		CompletedTasks: completedCount,
		FailedTasks:    failedCount,
		TotalTasks:     queuedCount + runningCount + completedCount + failedCount,
		Concurrency:    concurrencyStats,
	}
}

// ManagerStats 管理器统计信息
type ManagerStats struct {
	QueuedTasks    int   `json:"queued_tasks"`
	RunningTasks   int   `json:"running_tasks"`
	CompletedTasks int   `json:"completed_tasks"`
	FailedTasks    int   `json:"failed_tasks"`
	TotalTasks     int   `json:"total_tasks"`
	Concurrency    Stats `json:"concurrency"`
}

// AddEventHandler 添加事件处理器
func (dm *DownloadManager) AddEventHandler(handler EventHandler) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.eventHandlers = append(dm.eventHandlers, handler)
}

// emitEvent 发送事件
func (dm *DownloadManager) emitEvent(event ManagerEvent) {
	dm.mu.RLock()
	handlers := make([]EventHandler, len(dm.eventHandlers))
	copy(handlers, dm.eventHandlers)
	dm.mu.RUnlock()

	for _, handler := range handlers {
		go handler(event)
	}
}

// ClearCompletedTasks 清理已完成的任务
func (dm *DownloadManager) ClearCompletedTasks() int {
	count := 0
	dm.completedTasks.Range(func(key, value interface{}) bool {
		task := value.(*DownloadTask)
		if task.GetStatus() == TaskStatusCompleted {
			dm.completedTasks.Delete(key)
			count++
		}
		return true
	})
	utils.Info("已清理 %d 个已完成任务", count)
	return count
}

// RepairFalseCompletedRecords 修复误标记为完成的下载记录
// 扫描所有 completed 记录，检查磁盘上是否存在视频文件，不存在则标记为 failed 并重置视频下载状态
func (dm *DownloadManager) RepairFalseCompletedRecords() (int, error) {
	if dm.db == nil {
		return 0, fmt.Errorf("数据库未初始化")
	}

	var records []models.DownloadRecord
	if err := dm.db.Preload("Video").Where("status = ?", "completed").Find(&records).Error; err != nil {
		return 0, fmt.Errorf("查询下载记录失败: %w", err)
	}

	videoExts := []string{".mp4", ".mkv", ".webm", ".flv", ".avi", ".m4v"}
	repaired := 0

	for _, record := range records {
		videoDir := record.Video.Path
		if videoDir == "" {
			continue
		}

		// 检查目录是否存在
		if _, err := os.Stat(videoDir); os.IsNotExist(err) {
			dm.markRecordAsFailed(&record, "下载目录不存在")
			repaired++
			continue
		}

		// 检查目录内是否有视频文件
		hasVideo := false
		entries, err := os.ReadDir(videoDir)
		if err != nil {
			dm.markRecordAsFailed(&record, "无法读取下载目录")
			repaired++
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := strings.ToLower(entry.Name())
			for _, ext := range videoExts {
				if strings.HasSuffix(name, ext) {
					info, err := entry.Info()
					if err == nil && info.Size() > 0 {
						hasVideo = true
					}
					break
				}
			}
			if hasVideo {
				break
			}
		}

		if !hasVideo {
			dm.markRecordAsFailed(&record, "视频文件不存在或大小为0，疑似下载未成功")
			repaired++
		}
	}

	if repaired > 0 {
		utils.Info("已修复 %d 条误标记为完成的下载记录", repaired)
	}
	return repaired, nil
}

// markRecordAsFailed 将记录标记为失败并重置视频下载状态
func (dm *DownloadManager) markRecordAsFailed(record *models.DownloadRecord, errMsg string) {
	now := time.Now()
	var details models.FileDetailsData
	if json.Unmarshal(record.FileDetails, &details) == nil {
		for i := range details.Files {
			// 只标记视频文件为失败（nfo/subtitle/danmaku的size在进度追踪中始终为0，不能作为判断依据）
			if details.Files[i].Name == "video" && details.Files[i].Status == "succeeded" && details.Files[i].Size == 0 {
				details.Files[i].Status = "failed"
			}
		}
		if detailsJSON, err := json.Marshal(details); err == nil {
			record.FileDetails = detailsJSON
		}
	}

	dm.db.Model(record).Updates(map[string]interface{}{
		"status":        "failed",
		"error_message": errMsg,
		"file_details":  record.FileDetails,
		"completed_at":  now,
	})

	// 重置视频下载状态
	dm.db.Model(&models.Video{}).Where("id = ?", record.VideoID).Update("download_status", 0)
	// 重置分P下载状态
	dm.db.Model(&models.Page{}).Where("video_id = ?", record.VideoID).Update("download_status", 0)

	utils.Info("已修复记录 ID=%d, 视频ID=%d: %s", record.ID, record.VideoID, errMsg)
}

// getMaxRetries 获取配置的最大重试次数
func (dm *DownloadManager) getMaxRetries() int {
	if dm.config != nil && dm.config.Advanced.MaxRetryCount > 0 {
		return dm.config.Advanced.MaxRetryCount
	}
	return 3
}

// IsRunning 检查管理器是否在运行
func (dm *DownloadManager) IsRunning() bool {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.running
}

// UpdateConfig 更新配置
func (dm *DownloadManager) UpdateConfig(cfg *config.Config) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// 保存旧的下载基础路径
	oldDownloadBase := dm.config.Paths.DownloadBase

	// 更新管理器的配置引用
	dm.config = cfg

	// 更新下载器的配置引用
	if dm.downloader != nil {
		dm.downloader.UpdateConfig(cfg)
	}

	// 更新并发控制器的限制
	maxVideos := cfg.Advanced.ConcurrentLimit.Video
	maxPages := cfg.Advanced.ConcurrentLimit.Page
	if maxVideos <= 0 {
		maxVideos = 2
	}
	if maxPages <= 0 {
		maxPages = 4
	}
	if dm.concurrency != nil {
		dm.concurrency.UpdateLimits(maxVideos, maxPages)
	}

	// 如果下载基础路径发生变化，更新队列中所有待处理任务的路径
	if oldDownloadBase != cfg.Paths.DownloadBase {
		dm.updateQueuedTaskPaths(oldDownloadBase, cfg.Paths.DownloadBase)
	}

	utils.Info("下载管理器配置已更新")
}

// updateQueuedTaskPaths 更新队列中任务的输出路径
func (dm *DownloadManager) updateQueuedTaskPaths(oldBase, newBase string) {
	if dm.queue == nil {
		return
	}

	// 规范化路径（统一路径分隔符）
	oldBase = filepath.Clean(oldBase)
	newBase = filepath.Clean(newBase)

	updatedCount := 0
	tasks := dm.queue.GetAll()

	for _, task := range tasks {
		// 只更新待处理或排队中的任务
		if task.Status == TaskStatusPending || task.Status == TaskStatusQueued {
			task.mu.Lock()

			oldPath := filepath.Clean(task.OutputDir)

			// 使用 strings.HasPrefix 检查路径前缀，并确保后面跟着路径分隔符或结尾
			if strings.HasPrefix(oldPath, oldBase) {
				// 确保匹配的是完整的路径段，而不是部分匹配
				// 例如：/downloads 应该匹配 /downloads/video，但不应该匹配 /downloads2/video
				isValidPrefix := false
				if len(oldPath) == len(oldBase) {
					// 完全相等
					isValidPrefix = true
				} else if len(oldPath) > len(oldBase) {
					// 检查后面是否是路径分隔符
					nextChar := oldPath[len(oldBase)]
					if nextChar == filepath.Separator || nextChar == '/' || nextChar == '\\' {
						isValidPrefix = true
					}
				}

				if isValidPrefix {
					// 提取相对路径部分
					relativePath := ""
					if len(oldPath) > len(oldBase) {
						relativePath = oldPath[len(oldBase):]
						// 去掉开头的路径分隔符
						relativePath = strings.TrimPrefix(relativePath, string(filepath.Separator))
						relativePath = strings.TrimPrefix(relativePath, "/")
						relativePath = strings.TrimPrefix(relativePath, "\\")
					}

					// 构建新路径
					if relativePath != "" {
						task.OutputDir = filepath.Join(newBase, relativePath)
					} else {
						task.OutputDir = newBase
					}
					updatedCount++

					utils.Debug("更新任务 %s 的输出路径: %s -> %s", task.ID, oldPath, task.OutputDir)
				}
			}

			task.mu.Unlock()
		}
	}

	if updatedCount > 0 {
		utils.Info("已更新 %d 个待处理任务的下载路径", updatedCount)
	}
}

// GetDownloader 获取下载器实例
func (dm *DownloadManager) GetDownloader() *Downloader {
	return dm.downloader
}

// buildFileDetails 根据配置构建文件详情列表
func (dm *DownloadManager) buildFileDetails(video *models.Video) models.FileDetailsData {
	files := []models.FileDetail{
		{Name: "video", Label: "视频", Status: "pending"},
	}
	if dm.config != nil {
		if !dm.config.Download.SkipPoster {
			files = append(files, models.FileDetail{Name: "poster", Label: "封面", Status: "pending"})
		}
		if !dm.config.Download.SkipVideoNFO {
			files = append(files, models.FileDetail{Name: "nfo", Label: "NFO", Status: "pending"})
		}
		if !dm.config.Download.SkipDanmaku {
			files = append(files, models.FileDetail{Name: "danmaku", Label: "弹幕", Status: "pending"})
		}
		if !dm.config.Download.SkipSubtitle {
			files = append(files, models.FileDetail{Name: "subtitle", Label: "字幕", Status: "pending"})
		}
	}
	return models.FileDetailsData{Files: files}
}

// getVideoSourceInfo 获取视频的源信息
func (dm *DownloadManager) getVideoSourceInfo(video *models.Video) (sourceType string, sourceID uint, sourceName string) {
	if video.FavoriteID != nil {
		sourceType = "favorite"
		sourceID = *video.FavoriteID
		var fav models.Favorite
		if dm.db.First(&fav, sourceID).Error == nil {
			sourceName = fav.Name
		}
	} else if video.CollectionID != nil {
		sourceType = "collection"
		sourceID = *video.CollectionID
		var col models.Collection
		if dm.db.First(&col, sourceID).Error == nil {
			sourceName = col.Name
		}
	} else if video.SubmissionID != nil {
		sourceType = "submission"
		sourceID = *video.SubmissionID
		var sub models.Submission
		if dm.db.First(&sub, sourceID).Error == nil {
			sourceName = sub.Name
		}
	} else if video.WatchLaterID != nil {
		sourceType = "watch_later"
		sourceID = *video.WatchLaterID
		var wl models.WatchLater
		if dm.db.First(&wl, sourceID).Error == nil {
			sourceName = wl.Name
		}
	}
	return
}

// updateDownloadRecordProgress 更新下载记录中的文件进度
func (dm *DownloadManager) updateDownloadRecordProgress(videoID uint, taskName string, progress *SubTaskProgress) {
	// 节流：每500ms最多更新一次DB，减少大文件下载时的写入压力
	// 终态（succeeded/failed/skipped）不节流，确保最终状态一定写入DB
	isTerminal := progress.Status == StatusSucceeded || progress.Status == StatusFailed || progress.Status == StatusSkipped
	key := fmt.Sprintf("%d-%s", videoID, taskName)
	now := time.Now()
	if !isTerminal {
		if last, ok := dm.lastProgressUpdate.Load(key); ok {
			if now.Sub(last.(time.Time)) < 500*time.Millisecond {
				return
			}
		}
	}
	dm.lastProgressUpdate.Store(key, now)
	var record models.DownloadRecord
	if err := dm.db.Where("video_id = ? AND status IN ?", videoID, []string{"pending", "downloading"}).
		Order("created_at DESC").First(&record).Error; err != nil {
		return
	}

	var details models.FileDetailsData
	if err := json.Unmarshal(record.FileDetails, &details); err != nil {
		return
	}

	updated := false
	for i := range details.Files {
		if details.Files[i].Name == taskName {
			details.Files[i].Status = string(progress.Status)
			details.Files[i].Progress = progress.Progress
			details.Files[i].Size = progress.DownloadedSize
			updated = true
			break
		}
	}

	if !updated {
		return
	}

	detailsJSON, _ := json.Marshal(details)

	updateTime := time.Now()
	updates := map[string]interface{}{
		"file_details": detailsJSON,
		"status":       "downloading",
	}
	if record.StartedAt == nil {
		updates["started_at"] = updateTime
	}

	dm.db.Model(&record).Where("status IN ?", []string{"pending", "downloading"}).Updates(updates)

	// 通过事件推送进度
	dm.emitEvent(ManagerEvent{
		Type:      "download_record_progress",
		Message:   taskName,
		Timestamp: updateTime,
		Task:      &DownloadTask{RecordID: record.ID},
		Progress:  progress,
	})
}
