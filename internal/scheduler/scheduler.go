package scheduler

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"bili-download/internal/config"
	"bili-download/internal/database/models"
	"bili-download/internal/downloader"
	"bili-download/internal/utils"

	"gorm.io/gorm"
)

// Scheduler 调度器
type Scheduler struct {
	config          *config.Config
	db              *gorm.DB
	downloadManager *downloader.DownloadManager

	// 调度控制
	ticker     *time.Ticker
	running    bool
	mu         sync.RWMutex
	ctx        context.Context
	cancelFunc context.CancelFunc

	// 状态信息
	lastRunAt     *time.Time
	nextRunAt     *time.Time
	currentSyncID string

	// 事件处理
	eventHandlers []EventHandler
}

// SchedulerStatus 调度器状态
type SchedulerStatus struct {
	IsRunning     bool       `json:"is_running"`
	LastRunAt     *time.Time `json:"last_run_at"`
	NextRunAt     *time.Time `json:"next_run_at"`
	CurrentSyncID string     `json:"current_sync_id"`
	Interval      int        `json:"interval"` // 秒
}

// EventType 事件类型
type EventType string

const (
	EventSchedulerStarted EventType = "scheduler_started"
	EventSchedulerStopped EventType = "scheduler_stopped"
	EventSyncStarted      EventType = "sync_started"
	EventSyncCompleted    EventType = "sync_completed"
	EventSyncFailed       EventType = "sync_failed"
	EventSourceScanned    EventType = "source_scanned"
	EventError            EventType = "error"
)

// Event 调度器事件
type Event struct {
	Type      EventType   `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// EventHandler 事件处理器
type EventHandler func(event Event)

// NewScheduler 创建调度器
func NewScheduler(cfg *config.Config, db *gorm.DB, dm *downloader.DownloadManager) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())

	s := &Scheduler{
		config:          cfg,
		db:              db,
		downloadManager: dm,
		ctx:             ctx,
		cancelFunc:      cancel,
		eventHandlers:   make([]EventHandler, 0),
	}

	utils.Info("调度器已创建")
	return s
}

// autoMigrate 自动迁移数据库表
func (s *Scheduler) autoMigrate() error {
	utils.Info("开始迁移调度器相关表...")

	// 自动创建或更新表结构
	// 注意：GORM AutoMigrate 可能会产生一些无害的约束相关警告
	if err := s.db.AutoMigrate(&models.SchedulerState{}); err != nil {
		// 检查是否是约束不存在的错误（这是无害的）
		if !isConstraintError(err) {
			return fmt.Errorf("迁移 scheduler_state 表失败: %w", err)
		}
		utils.Debug("scheduler_state 表迁移产生了无害的约束警告")
	}

	if err := s.db.AutoMigrate(&models.SyncLog{}); err != nil {
		if !isConstraintError(err) {
			return fmt.Errorf("迁移 sync_logs 表失败: %w", err)
		}
		utils.Debug("sync_logs 表迁移产生了无害的约束警告")
	}

	if err := s.db.AutoMigrate(&models.VideoSourceScan{}); err != nil {
		if !isConstraintError(err) {
			return fmt.Errorf("迁移 video_source_scans 表失败: %w", err)
		}
		utils.Debug("video_source_scans 表迁移产生了无害的约束警告")
	}

	utils.Info("调度器相关表迁移完成")
	return nil
}

// isConstraintError 检查是否是约束不存在的错误（无害）
func isConstraintError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	// PostgreSQL 的约束不存在错误
	return strings.Contains(errMsg, "does not exist") &&
		(strings.Contains(errMsg, "constraint") || strings.Contains(errMsg, "CONSTRAINT"))
}

// Start 启动调度器
func (s *Scheduler) Start() error {
	utils.Info("[Scheduler] Start() 方法被调用")

	utils.Debug("[Scheduler] 尝试获取锁...")
	s.mu.Lock()
	utils.Debug("[Scheduler] 锁已获取")
	defer func() {
		s.mu.Unlock()
		utils.Debug("[Scheduler] 锁已释放")
	}()

	if s.running {
		utils.Warn("[Scheduler] 调度器已在运行，返回错误")
		return fmt.Errorf("调度器已在运行")
	}

	utils.Info("[Scheduler] 启动调度器，同步间隔：%d 秒", s.config.Sync.Interval)

	// 首次启动时异步迁移表
	utils.Debug("[Scheduler] 启动异步表迁移")
	go func() {
		utils.Debug("[Scheduler] 表迁移 goroutine 开始执行")
		if err := s.autoMigrate(); err != nil {
			utils.Error("[Scheduler] 数据库表迁移失败（这不会影响调度器运行）: %v", err)
		} else {
			utils.Debug("[Scheduler] 表迁移 goroutine 执行完成")
		}
	}()

	// 初始化定时器
	utils.Debug("[Scheduler] 初始化定时器...")
	interval := time.Duration(s.config.Sync.Interval) * time.Second
	s.ticker = time.NewTicker(interval)
	s.running = true
	utils.Debug("[Scheduler] 定时器已初始化，间隔: %v", interval)

	// 计算下次运行时间
	nextRun := time.Now().Add(interval)
	s.nextRunAt = &nextRun
	utils.Debug("[Scheduler] 下次运行时间: %v", nextRun)

	// 异步更新数据库状态，避免阻塞
	utils.Debug("[Scheduler] 启动异步数据库状态更新")
	go func() {
		utils.Debug("[Scheduler] 数据库状态更新 goroutine 开始执行")
		if err := s.updateStateDB(); err != nil {
			utils.Error("[Scheduler] 更新调度器状态失败: %v", err)
		} else {
			utils.Debug("[Scheduler] 数据库状态更新 goroutine 执行完成")
		}
	}()

	// 启动调度循环
	utils.Debug("[Scheduler] 启动调度循环 goroutine")
	go s.runScheduleLoop()

	utils.Info("[Scheduler] 调度器启动成功")
	return nil
}

// startEmitEvent 在 Start 完成后发送事件（需要在调用 Start 后立即调用）
func (s *Scheduler) startEmitEvent() {
	// 发送事件（不持有锁）
	utils.Debug("[Scheduler] 发送启动事件")
	s.emitEvent(Event{
		Type:      EventSchedulerStarted,
		Timestamp: time.Now(),
	})
	utils.Debug("[Scheduler] 启��事件已发送")
}

// Stop 停止调度器
func (s *Scheduler) Stop() error {
	utils.Info("[Scheduler] Stop() 方法被调用")

	utils.Debug("[Scheduler] 尝试获取锁...")
	s.mu.Lock()
	utils.Debug("[Scheduler] 锁已获取")

	if !s.running {
		utils.Warn("[Scheduler] 调度器未运行，返回错误")
		s.mu.Unlock()
		return fmt.Errorf("调度器未运行")
	}

	utils.Info("[Scheduler] 正在停止调度器...")

	// 停止定时器
	if s.ticker != nil {
		utils.Debug("[Scheduler] 停止定时器")
		s.ticker.Stop()
	}

	// 取消当前同步任务（如果有）
	if s.currentSyncID != "" {
		utils.Warn("[Scheduler] 当前有同步任务运行: %s", s.currentSyncID)
		// TODO: 实现取消逻辑
	}

	s.running = false
	s.nextRunAt = nil
	utils.Debug("[Scheduler] 调度器状态已更新")

	// 释放锁
	s.mu.Unlock()
	utils.Debug("[Scheduler] 锁已释放")

	// 异步更新数据库状态，避免阻塞
	utils.Debug("[Scheduler] 启动异步数据库状态更新")
	go func() {
		utils.Debug("[Scheduler] 数据库状态更新 goroutine 开始执行")
		if err := s.updateStateDB(); err != nil {
			utils.Error("[Scheduler] 更新调度器状态失败: %v", err)
		} else {
			utils.Debug("[Scheduler] 数据库状态更新 goroutine 执行完成")
		}
	}()

	// 发送事件（在释放锁之后）
	utils.Debug("[Scheduler] 发送停止事件")
	s.emitEvent(Event{
		Type:      EventSchedulerStopped,
		Timestamp: time.Now(),
	})
	utils.Debug("[Scheduler] 停止事件已发送")

	utils.Info("[Scheduler] 调度器已停止")
	return nil
}

// UpdateConfig 更新配置
func (s *Scheduler) UpdateConfig(cfg *config.Config) {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldInterval := s.config.Sync.Interval
	s.config = cfg

	// 如果调度器正在运行且同步间隔发生变化，需要重启调度循环
	if s.running && oldInterval != cfg.Sync.Interval {
		utils.Info("同步间隔已从 %d 秒更改为 %d 秒，正在重启调度器...", oldInterval, cfg.Sync.Interval)

		// 取消旧的上下文以停止调度循环
		if s.cancelFunc != nil {
			s.cancelFunc()
		}

		// 停止旧的ticker
		if s.ticker != nil {
			s.ticker.Stop()
		}

		// 创建新的上下文
		ctx, cancel := context.WithCancel(context.Background())
		s.ctx = ctx
		s.cancelFunc = cancel

		// 创建新的ticker
		interval := time.Duration(cfg.Sync.Interval) * time.Second
		s.ticker = time.NewTicker(interval)

		// 更新下次运行时间
		nextRun := time.Now().Add(interval)
		s.nextRunAt = &nextRun

		// 启动新的调度循环
		go s.runScheduleLoop()

		utils.Info("调度器已重启，新的同步间隔: %d 秒，下次运行时间: %v", cfg.Sync.Interval, nextRun)
	}

	utils.Info("调度器配置已更新")
}

// TriggerManual 手动触发一次同步
func (s *Scheduler) TriggerManual() (string, error) {
	utils.Info("[TriggerManual] 方法被调用")

	utils.Debug("[TriggerManual] 尝试获取锁...")
	s.mu.Lock()
	utils.Debug("[TriggerManual] 锁已获取")

	// 检查是否已有同步任务在运行
	if s.currentSyncID != "" {
		utils.Warn("[TriggerManual] 已有同步任务在运行: %s", s.currentSyncID)
		s.mu.Unlock()
		return "", fmt.Errorf("已有同步任务在运行: %s", s.currentSyncID)
	}

	// 创建同步任务
	utils.Debug("[TriggerManual] 创建同步任务...")
	syncTask := NewSyncTask(context.Background(), "manual", s.db, s.config, s.downloadManager)
	syncID := syncTask.ID
	utils.Debug("[TriggerManual] 同步任务已创建: %s", syncID)

	// 设置当前同步ID
	s.currentSyncID = syncID
	s.mu.Unlock()
	utils.Debug("[TriggerManual] 锁已释放")

	utils.Info("[TriggerManual] 手动触发同步任务: %s", syncID)

	// 异步执行同步任务
	utils.Debug("[TriggerManual] 启动同步任务 goroutine")
	go func() {
		utils.Debug("[TriggerManual] 同步任务 goroutine 开始执行")

		// 发送同步开始事件
		utils.Debug("[TriggerManual] 发送同步开始事件")
		s.emitEvent(Event{
			Type: EventSyncStarted,
			Data: map[string]interface{}{
				"sync_id":      syncID,
				"trigger_type": "manual",
			},
			Timestamp: time.Now(),
		})
		utils.Debug("[TriggerManual] 同步开始事件已发送")

		// 执行同步
		utils.Debug("[TriggerManual] 开始执行同步任务...")
		err := syncTask.Execute()
		utils.Debug("[TriggerManual] 同步任务执行完成，错误: %v", err)

		// 清除当前同步ID
		utils.Debug("[TriggerManual] 清除当前同步ID")
		s.mu.Lock()
		s.currentSyncID = ""
		now := time.Now()
		s.lastRunAt = &now
		s.mu.Unlock()
		utils.Debug("[TriggerManual] 当前同步ID已清除")

		if err != nil {
			utils.Error("[TriggerManual] 同步任务执行失败: %v", err)
			utils.Debug("[TriggerManual] 发送同步失败事件")
			s.emitEvent(Event{
				Type: EventSyncFailed,
				Data: map[string]interface{}{
					"sync_id": syncID,
					"error":   err.Error(),
				},
				Timestamp: time.Now(),
			})
			utils.Debug("[TriggerManual] 同步失败事件已发送")
			return
		}

		// 发送同步完成事件
		utils.Debug("[TriggerManual] 发送同步完成事件")
		s.emitEvent(Event{
			Type: EventSyncCompleted,
			Data: map[string]interface{}{
				"sync_id":       syncID,
				"videos_found":  syncTask.VideosFound,
				"videos_new":    syncTask.VideosNew,
				"videos_queued": syncTask.VideosQueued,
			},
			Timestamp: time.Now(),
		})
		utils.Debug("[TriggerManual] 同步完成事件已发送")

		utils.Info("[TriggerManual] 同步任务完成: %s", syncID)
	}()

	utils.Info("[TriggerManual] 方法返回，同步任务ID: %s", syncID)
	return syncID, nil
}

// IsRunning 检查调度器是否正在运行
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetStatus 获取调度器状态
func (s *Scheduler) GetStatus() *SchedulerStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &SchedulerStatus{
		IsRunning:     s.running,
		LastRunAt:     s.lastRunAt,
		NextRunAt:     s.nextRunAt,
		CurrentSyncID: s.currentSyncID,
		Interval:      s.config.Sync.Interval,
	}
}

// OnEvent 注册事件处理器
func (s *Scheduler) OnEvent(handler EventHandler) {
	// 不使用锁，因为这通常在初始化时调用
	// 如果需要在运行时添加处理器，可以考虑使用 sync/atomic 或 channel
	s.eventHandlers = append(s.eventHandlers, handler)
}

// runScheduleLoop 调度循环
func (s *Scheduler) runScheduleLoop() {
	utils.Info("[runScheduleLoop] 调度循环已启动")

	for {
		utils.Debug("[runScheduleLoop] 等待事件...")
		select {
		case <-s.ticker.C:
			// 定时触发
			utils.Info("[runScheduleLoop] 定时器触发，开始同步...")
			s.performSync("auto")
			utils.Debug("[runScheduleLoop] 定时同步完成，继续等待...")

		case <-s.ctx.Done():
			// 上下文取消
			utils.Info("[runScheduleLoop] 调度循环收到退出信号")
			return
		}
	}
}

// performSync 执行同步
func (s *Scheduler) performSync(triggerType string) {
	utils.Info("[performSync] 方法被调用，触发类型: %s", triggerType)

	utils.Debug("[performSync] 尝试获取锁...")
	s.mu.Lock()
	utils.Debug("[performSync] 锁已获取")

	// 检查是否已有同步任务在运行
	if s.currentSyncID != "" {
		s.mu.Unlock()
		utils.Warn("[performSync] 同步任务已在运行，跳过本次触发")
		return
	}
	s.mu.Unlock()
	utils.Debug("[performSync] 锁已释放")

	// 执行同步任务
	utils.Debug("[performSync] 创建上下文")
	ctx := context.Background()
	utils.Debug("[performSync] 调用 executeSync")
	syncID, err := s.executeSync(ctx, triggerType)
	utils.Debug("[performSync] executeSync 返回，同步ID: %s, 错误: %v", syncID, err)

	if err != nil {
		utils.Error("[performSync] 同步任务执行失败: %v", err)
		utils.Debug("[performSync] 发送同步失败事件")
		s.emitEvent(Event{
			Type: EventSyncFailed,
			Data: map[string]interface{}{
				"error": err.Error(),
			},
			Timestamp: time.Now(),
		})
	}

	// 更新下次运行时间
	utils.Debug("[performSync] 更新下次运行时间")
	s.mu.Lock()
	interval := time.Duration(s.config.Sync.Interval) * time.Second
	nextRun := time.Now().Add(interval)
	s.nextRunAt = &nextRun
	s.mu.Unlock()
	utils.Debug("[performSync] 下次运行时间已更新: %v", nextRun)

	utils.Info("[performSync] 同步任务完成: %s", syncID)
}

// executeSync 执行同步逻辑
func (s *Scheduler) executeSync(ctx context.Context, triggerType string) (string, error) {
	utils.Info("[executeSync] 方法被调用，触发类型: %s", triggerType)

	// 创建同步任务
	utils.Debug("[executeSync] 创建同步任务...")
	syncTask := NewSyncTask(ctx, triggerType, s.db, s.config, s.downloadManager)
	utils.Debug("[executeSync] 同步任务已创建: %s", syncTask.ID)

	// 设置当前同步ID
	utils.Debug("[executeSync] 尝试获取锁...")
	s.mu.Lock()
	utils.Debug("[executeSync] 锁已获取")
	s.currentSyncID = syncTask.ID
	now := time.Now()
	s.lastRunAt = &now
	s.mu.Unlock()
	utils.Debug("[executeSync] 锁已释放，当前同步ID已设置: %s", syncTask.ID)

	// 发送同步开始事件
	utils.Debug("[executeSync] 发送同步开始事件")
	s.emitEvent(Event{
		Type: EventSyncStarted,
		Data: map[string]interface{}{
			"sync_id":      syncTask.ID,
			"trigger_type": triggerType,
		},
		Timestamp: time.Now(),
	})
	utils.Debug("[executeSync] 同步开始事件已发送")

	// 执行同步
	utils.Debug("[executeSync] 开始执行同步任务...")
	err := syncTask.Execute()
	utils.Debug("[executeSync] 同步任务执行完成，错误: %v", err)

	// 清除当前同步ID
	utils.Debug("[executeSync] 清除当前同步ID")
	s.mu.Lock()
	s.currentSyncID = ""
	s.mu.Unlock()
	utils.Debug("[executeSync] 当前同步ID已清除")

	if err != nil {
		utils.Error("[executeSync] 同步任务执行失败: %v", err)
		return syncTask.ID, err
	}

	// 发送同步完成事件
	utils.Debug("[executeSync] 发送同步完成事件")
	s.emitEvent(Event{
		Type: EventSyncCompleted,
		Data: map[string]interface{}{
			"sync_id":       syncTask.ID,
			"videos_found":  syncTask.VideosFound,
			"videos_new":    syncTask.VideosNew,
			"videos_queued": syncTask.VideosQueued,
		},
		Timestamp: time.Now(),
	})
	utils.Debug("[executeSync] 同步完成事件已发送")

	utils.Info("[executeSync] 同步任务完成: %s", syncTask.ID)
	return syncTask.ID, nil
}

// updateStateDB 更新数据库中的调度器状态
func (s *Scheduler) updateStateDB() error {
	utils.Debug("[updateStateDB] 方法被调用")
	var state models.SchedulerState

	// 添加超时控制，避免无限阻塞
	utils.Debug("[updateStateDB] 创建5秒超时上下文")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 查找现有状态记录
	utils.Debug("[updateStateDB] 查找现有状态记录...")
	result := s.db.WithContext(ctx).First(&state)
	utils.Debug("[updateStateDB] 查询完成，错误: %v", result.Error)

	if result.Error == gorm.ErrRecordNotFound {
		// 创建新记录
		utils.Debug("[updateStateDB] 未找到记录，创建新记录")
		state = models.SchedulerState{
			IsRunning:     s.running,
			LastRunAt:     s.lastRunAt,
			NextRunAt:     s.nextRunAt,
			CurrentSyncID: s.currentSyncID,
		}
		utils.Debug("[updateStateDB] 写入新记录到数据库...")
		err := s.db.WithContext(ctx).Create(&state).Error
		if err != nil {
			utils.Error("[updateStateDB] 创建记录失败: %v", err)
		} else {
			utils.Debug("[updateStateDB] 新记录创建成功")
		}
		return err
	}

	if result.Error != nil {
		utils.Error("[updateStateDB] 查询错误: %v", result.Error)
		return result.Error
	}

	// 更新现有记录
	utils.Debug("[updateStateDB] 找到现有记录，更新记录...")
	err := s.db.WithContext(ctx).Model(&state).Updates(map[string]interface{}{
		"is_running":      s.running,
		"last_run_at":     s.lastRunAt,
		"next_run_at":     s.nextRunAt,
		"current_sync_id": s.currentSyncID,
	}).Error
	if err != nil {
		utils.Error("[updateStateDB] 更新记录失败: %v", err)
	} else {
		utils.Debug("[updateStateDB] 记录更新成功")
	}
	return err
}

// emitEvent 发送事件
func (s *Scheduler) emitEvent(event Event) {
	utils.Debug("[emitEvent] 发送事件: %s", event.Type)

	utils.Debug("[emitEvent] 尝试获取读锁...")
	s.mu.RLock()
	utils.Debug("[emitEvent] 读锁已获取")
	handlers := make([]EventHandler, len(s.eventHandlers))
	copy(handlers, s.eventHandlers)
	s.mu.RUnlock()
	utils.Debug("[emitEvent] 读锁已释放，事件处理器数量: %d", len(handlers))

	for i, handler := range handlers {
		utils.Debug("[emitEvent] 启动事件处理器 goroutine #%d", i)
		go func(h EventHandler, index int) {
			defer func() {
				if r := recover(); r != nil {
					utils.Error("[emitEvent] 事件处理器 #%d 崩溃: %v", index, r)
				}
			}()
			utils.Debug("[emitEvent] 事件处理器 #%d 开始执行", index)
			h(event)
			utils.Debug("[emitEvent] 事件处理器 #%d 执行完成", index)
		}(handler, i)
	}
	utils.Debug("[emitEvent] 所有事件处理器 goroutine 已启动")
}
