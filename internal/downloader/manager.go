package downloader

import (
	"context"
	"fmt"
	"path/filepath"
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
)

// ManagerEvent 管理器事件
type ManagerEvent struct {
	Type      ManagerEventType `json:"type"`
	Task      *DownloadTask    `json:"task"`
	Progress  *SubTaskProgress `json:"progress,omitempty"`
	Message   string           `json:"message,omitempty"`
	Timestamp time.Time        `json:"timestamp"`
}

// EventHandler 事件处理器
type EventHandler func(event ManagerEvent)

// DownloadManager 下载管理器
type DownloadManager struct {
	config         *config.Config
	db             *gorm.DB
	biliClient     *bilibili.Client
	downloader     *Downloader
	queue          *TaskQueue
	concurrency    *ConcurrencyController
	tracker        *ProgressTracker
	runningTasks   sync.Map // taskID -> *DownloadTask
	completedTasks sync.Map // taskID -> *DownloadTask
	eventHandlers  []EventHandler
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	mu             sync.RWMutex
	running        bool
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

	// 为每个视频创建专属文件夹
	videoDir := filepath.Join(task.OutputDir, utils.Filenamify(video.Name))
	utils.Debug("视频下载目录: %s", videoDir)

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
			return
		}

		// 下载分P到视频专属文件夹
		err := dm.downloader.DownloadPage(task.Context, video, &page, videoDir)
		dm.concurrency.ReleasePage()

		if err != nil {
			utils.Error("下载分P失败: %v", err)
			task.SetError(err)
			task.SetStatus(TaskStatusFailed)
			dm.handleTaskFailure(task)
			return
		}
	}

	// 任务完成
	task.SetStatus(TaskStatusCompleted)

	// 更新数据库中的视频下载状态
	if dm.db != nil {
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

	// 为每个视频创建专属文件夹
	videoDir := filepath.Join(task.OutputDir, utils.Filenamify(task.Video.Name))
	utils.Debug("分P下载目录: %s", videoDir)

	// 下载分P到视频专属文件夹
	err := dm.downloader.DownloadPage(task.Context, task.Video, task.Page, videoDir)
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

// AddVideoTask 添加视频任务
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

	utils.Info("下载管理器配置已更新")
}

// GetDownloader 获取下载器实例
func (dm *DownloadManager) GetDownloader() *Downloader {
	return dm.downloader
}
