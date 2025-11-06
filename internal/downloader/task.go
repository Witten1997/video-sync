package downloader

import (
	"context"
	"fmt"
	"sync"
	"time"

	"bili-download/internal/database/models"
)

// TaskType 任务类型
type TaskType string

const (
	TaskTypeVideo      TaskType = "video"      // 视频任务（单个视频的所有分P）
	TaskTypePage       TaskType = "page"       // 分P任务（单个分P）
	TaskTypeCollection TaskType = "collection" // 合集任务（批量视频）
)

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"   // 待处理
	TaskStatusQueued    TaskStatus = "queued"    // 已入队
	TaskStatusRunning   TaskStatus = "running"   // 运行中
	TaskStatusPaused    TaskStatus = "paused"    // 已暂停
	TaskStatusCompleted TaskStatus = "completed" // 已完成
	TaskStatusFailed    TaskStatus = "failed"    // 失败
	TaskStatusCancelled TaskStatus = "cancelled" // 已取消
)

// TaskPriority 任务优先级
type TaskPriority int

const (
	PriorityLow    TaskPriority = 0
	PriorityNormal TaskPriority = 5
	PriorityHigh   TaskPriority = 10
	PriorityUrgent TaskPriority = 15
)

// DownloadTask 下载任务
type DownloadTask struct {
	ID          string             `json:"id"`           // 任务ID
	Type        TaskType           `json:"type"`         // 任务类型
	Status      TaskStatus         `json:"status"`       // 任务状态
	Priority    TaskPriority       `json:"priority"`     // 优先级
	Video       *models.Video      `json:"video"`        // 视频信息
	Page        *models.Page       `json:"page"`         // 分P信息（仅分P任务）
	OutputDir   string             `json:"output_dir"`   // 输出目录
	RetryCount  int                `json:"retry_count"`  // 重试次数
	MaxRetries  int                `json:"max_retries"`  // 最大重试次数
	Error       error              `json:"-"`            // 错误信息
	ErrorMsg    string             `json:"error_msg"`    // 错误消息（JSON序列化）
	CreatedAt   time.Time          `json:"created_at"`   // 创建时间
	StartedAt   time.Time          `json:"started_at"`   // 开始时间
	CompletedAt time.Time          `json:"completed_at"` // 完成时间
	CancelFunc  context.CancelFunc `json:"-"`            // 取消函数
	Context     context.Context    `json:"-"`            // 任务上下文
	mu          sync.RWMutex       `json:"-"`            // 读写锁
}

// NewDownloadTask 创建新的下载任务
func NewDownloadTask(taskType TaskType, video *models.Video, page *models.Page, outputDir string) *DownloadTask {
	ctx, cancel := context.WithCancel(context.Background())

	taskID := generateTaskID(taskType, video, page)

	return &DownloadTask{
		ID:         taskID,
		Type:       taskType,
		Status:     TaskStatusPending,
		Priority:   PriorityNormal,
		Video:      video,
		Page:       page,
		OutputDir:  outputDir,
		MaxRetries: 3,
		CreatedAt:  time.Now(),
		CancelFunc: cancel,
		Context:    ctx,
	}
}

// generateTaskID 生成任务ID
func generateTaskID(taskType TaskType, video *models.Video, page *models.Page) string {
	switch taskType {
	case TaskTypePage:
		return fmt.Sprintf("page-%d-%d", video.ID, page.ID)
	case TaskTypeVideo:
		return fmt.Sprintf("video-%d", video.ID)
	case TaskTypeCollection:
		return fmt.Sprintf("collection-%d-%d", video.ID, time.Now().Unix())
	default:
		return fmt.Sprintf("task-%d", time.Now().UnixNano())
	}
}

// GetStatus 获取任务状态（线程安全）
func (t *DownloadTask) GetStatus() TaskStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Status
}

// SetStatus 设置任务状态（线程安全）
func (t *DownloadTask) SetStatus(status TaskStatus) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.Status = status

	switch status {
	case TaskStatusRunning:
		if t.StartedAt.IsZero() {
			t.StartedAt = time.Now()
		}
	case TaskStatusCompleted, TaskStatusFailed, TaskStatusCancelled:
		t.CompletedAt = time.Now()
	}
}

// CanRetry 检查是否可以重试
func (t *DownloadTask) CanRetry() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.RetryCount < t.MaxRetries
}

// IncrementRetry 增加重试次数
func (t *DownloadTask) IncrementRetry() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.RetryCount++
}

// Cancel 取消任务
func (t *DownloadTask) Cancel() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.CancelFunc != nil {
		t.CancelFunc()
	}
	t.Status = TaskStatusCancelled
	t.CompletedAt = time.Now()
}

// IsCancelled 检查任务是否已取消
func (t *DownloadTask) IsCancelled() bool {
	select {
	case <-t.Context.Done():
		return true
	default:
		return false
	}
}

// SetError 设置错误信息
func (t *DownloadTask) SetError(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.Error = err
	if err != nil {
		t.ErrorMsg = err.Error()
	}
}

// GetError 获取错误信息
func (t *DownloadTask) GetError() error {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Error
}

// Duration 获取任务执行时长
func (t *DownloadTask) Duration() time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.StartedAt.IsZero() {
		return 0
	}

	if t.CompletedAt.IsZero() {
		return time.Since(t.StartedAt)
	}

	return t.CompletedAt.Sub(t.StartedAt)
}

// IsTerminal 检查是否是终止状态
func (t *DownloadTask) IsTerminal() bool {
	status := t.GetStatus()
	return status == TaskStatusCompleted ||
		status == TaskStatusFailed ||
		status == TaskStatusCancelled
}

// Clone 克隆任务（用于重试）
func (t *DownloadTask) Clone() *DownloadTask {
	t.mu.RLock()
	defer t.mu.RUnlock()

	ctx, cancel := context.WithCancel(context.Background())

	return &DownloadTask{
		ID:         t.ID,
		Type:       t.Type,
		Status:     TaskStatusPending,
		Priority:   t.Priority,
		Video:      t.Video,
		Page:       t.Page,
		OutputDir:  t.OutputDir,
		RetryCount: t.RetryCount,
		MaxRetries: t.MaxRetries,
		CreatedAt:  time.Now(),
		CancelFunc: cancel,
		Context:    ctx,
	}
}
