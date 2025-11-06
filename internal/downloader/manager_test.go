package downloader

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bili-download/internal/database/models"
)

// TestTaskQueue 测试任务队列
func TestTaskQueue(t *testing.T) {
	queue := NewTaskQueue()

	// 创建测试任务
	video := &models.Video{ID: 1, BVid: "BV1xx411c7mD", Name: "测试视频"}
	task1 := NewDownloadTask(TaskTypeVideo, video, nil, "./downloads")
	task1.Priority = PriorityNormal

	task2 := NewDownloadTask(TaskTypeVideo, video, nil, "./downloads")
	task2.ID = "task-2"
	task2.Priority = PriorityHigh

	task3 := NewDownloadTask(TaskTypeVideo, video, nil, "./downloads")
	task3.ID = "task-3"
	task3.Priority = PriorityLow

	// 测试入队
	queue.Enqueue(task1)
	queue.Enqueue(task2)
	queue.Enqueue(task3)

	if queue.Size() != 3 {
		t.Errorf("队列大小错误，期望 3，得到 %d", queue.Size())
	}

	// 测试优先级出队（应该是 task2, task1, task3）
	dequeued := queue.Dequeue()
	if dequeued.ID != task2.ID {
		t.Errorf("出队顺序错误，期望 %s，得到 %s", task2.ID, dequeued.ID)
	}

	dequeued = queue.Dequeue()
	if dequeued.ID != task1.ID {
		t.Errorf("出队顺序错误，期望 %s，得到 %s", task1.ID, dequeued.ID)
	}

	dequeued = queue.Dequeue()
	if dequeued.ID != task3.ID {
		t.Errorf("出队顺序错误，期望 %s，得到 %s", task3.ID, dequeued.ID)
	}

	if queue.Size() != 0 {
		t.Errorf("队列应该为空，但大小为 %d", queue.Size())
	}
}

// TestTaskQueueContains 测试任务查找
func TestTaskQueueContains(t *testing.T) {
	queue := NewTaskQueue()

	video := &models.Video{ID: 1, BVid: "BV1xx411c7mD", Name: "测试视频"}
	task := NewDownloadTask(TaskTypeVideo, video, nil, "./downloads")

	queue.Enqueue(task)

	if !queue.Contains(task.ID) {
		t.Error("任务应该在队列中")
	}

	if queue.Contains("non-existent") {
		t.Error("不存在的任务不应该被找到")
	}
}

// TestTaskQueueRemove 测试任务移除
func TestTaskQueueRemove(t *testing.T) {
	queue := NewTaskQueue()

	video := &models.Video{ID: 1, BVid: "BV1xx411c7mD", Name: "测试视频"}
	task1 := NewDownloadTask(TaskTypeVideo, video, nil, "./downloads")
	task2 := NewDownloadTask(TaskTypeVideo, video, nil, "./downloads")
	task2.ID = "task-2"

	queue.Enqueue(task1)
	queue.Enqueue(task2)

	removed := queue.Remove(task1.ID)
	if removed == nil || removed.ID != task1.ID {
		t.Error("移除的任务不正确")
	}

	if queue.Size() != 1 {
		t.Errorf("队列大小错误，期望 1，得到 %d", queue.Size())
	}
}

// TestTaskQueueUpdatePriority 测试优先级更新
func TestTaskQueueUpdatePriority(t *testing.T) {
	queue := NewTaskQueue()

	video := &models.Video{ID: 1, BVid: "BV1xx411c7mD", Name: "测试视频"}
	task := NewDownloadTask(TaskTypeVideo, video, nil, "./downloads")
	task.Priority = PriorityLow

	queue.Enqueue(task)

	// 更新优先级
	queue.UpdatePriority(task.ID, PriorityHigh)

	// 验证优先级已更新
	retrieved := queue.Get(task.ID)
	if retrieved.Priority != PriorityHigh {
		t.Errorf("优先级未更新，期望 %d，得到 %d", PriorityHigh, retrieved.Priority)
	}
}

// TestConcurrencyController 测试并发控制器
func TestConcurrencyController(t *testing.T) {
	cc := NewConcurrencyController(2, 4)

	// 测试初始状态
	if !cc.CanStartVideo() {
		t.Error("应该可以启动视频下载")
	}

	if !cc.CanStartPage() {
		t.Error("应该可以启动分P下载")
	}

	// 测试获取许可
	ctx := context.Background()

	err := cc.AcquireVideo(ctx)
	if err != nil {
		t.Errorf("获取视频许可失败: %v", err)
	}

	err = cc.AcquireVideo(ctx)
	if err != nil {
		t.Errorf("获取视频许可失败: %v", err)
	}

	// 此时应该没有可用的视频许可
	if cc.CanStartVideo() {
		t.Error("视频许可应该已用完")
	}

	// 释放一个许可
	cc.ReleaseVideo()

	if !cc.CanStartVideo() {
		t.Error("释放后应该有可用的视频许可")
	}

	cc.ReleaseVideo()
}

// TestConcurrencySemaphore 测试信号量
func TestConcurrencySemaphore(t *testing.T) {
	sem := NewSemaphore(2)

	if sem.Available() != 2 {
		t.Errorf("可用许可数错误，期望 2，得到 %d", sem.Available())
	}

	// 获取许可
	sem.Acquire()
	if sem.Available() != 1 {
		t.Errorf("可用许可数错误，期望 1，得到 %d", sem.Available())
	}

	sem.Acquire()
	if sem.Available() != 0 {
		t.Errorf("可用许可数错误，期望 0，得到 %d", sem.Available())
	}

	// 尝试非阻塞获取（应该失败）
	if sem.TryAcquire() {
		t.Error("非阻塞获取应该失败")
	}

	// 释放许可
	sem.Release()
	if sem.Available() != 1 {
		t.Errorf("可用许可数错误，期望 1，得到 %d", sem.Available())
	}

	// 现在应该可以非阻塞获取
	if !sem.TryAcquire() {
		t.Error("非阻塞获取应该成功")
	}

	sem.Release()
	sem.Release()
}

// TestSemaphoreWithContext 测试带上下文的信号量
func TestSemaphoreWithContext(t *testing.T) {
	sem := NewSemaphore(1)
	sem.Acquire()

	// 创建会被取消的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// 尝试获取（应该超时）
	err := sem.AcquireWithContext(ctx)
	if err == nil {
		t.Error("应该返回超时错误")
	}

	if err != context.DeadlineExceeded {
		t.Errorf("错误类型不正确，期望 %v，得到 %v", context.DeadlineExceeded, err)
	}

	sem.Release()
}

// TestDownloadTask 测试下载任务
func TestDownloadTask(t *testing.T) {
	video := &models.Video{ID: 1, BVid: "BV1xx411c7mD", Name: "测试视频"}
	page := &models.Page{ID: 1, PID: 1, CID: 123456, Name: "第一集"}

	task := NewDownloadTask(TaskTypePage, video, page, "./downloads")

	// 测试初始状态
	if task.GetStatus() != TaskStatusPending {
		t.Errorf("初始状态错误，期望 %s，得到 %s", TaskStatusPending, task.GetStatus())
	}

	// 测试状态更新
	task.SetStatus(TaskStatusRunning)
	if task.GetStatus() != TaskStatusRunning {
		t.Errorf("状态未更新，期望 %s，得到 %s", TaskStatusRunning, task.GetStatus())
	}

	// 测试开始时间
	if task.StartedAt.IsZero() {
		t.Error("开始时间应该已设置")
	}

	// 等待一小段时间
	time.Sleep(10 * time.Millisecond)

	// 测试完成
	task.SetStatus(TaskStatusCompleted)
	if task.CompletedAt.IsZero() {
		t.Error("完���时间应该已设置")
	}

	// 测试执行时长
	duration := task.Duration()
	if duration <= 0 {
		t.Errorf("执行时长应该大于0，得到 %v", duration)
	}
}

// TestTaskCanRetry 测试任务重试
func TestTaskCanRetry(t *testing.T) {
	video := &models.Video{ID: 1, BVid: "BV1xx411c7mD", Name: "测试视频"}
	task := NewDownloadTask(TaskTypeVideo, video, nil, "./downloads")
	task.MaxRetries = 3

	// 初始可以重试
	if !task.CanRetry() {
		t.Error("应该可以重试")
	}

	// 增加重试次数
	task.IncrementRetry()
	task.IncrementRetry()
	task.IncrementRetry()

	// 达到最大重试次数
	if task.CanRetry() {
		t.Error("不应该可以重试")
	}
}

// TestTaskCancel 测试任务取消
func TestTaskCancel(t *testing.T) {
	video := &models.Video{ID: 1, BVid: "BV1xx411c7mD", Name: "测试视频"}
	task := NewDownloadTask(TaskTypeVideo, video, nil, "./downloads")

	// 取消任务
	task.Cancel()

	// 验证任务已取消
	if !task.IsCancelled() {
		t.Error("任务应该已取消")
	}

	if task.GetStatus() != TaskStatusCancelled {
		t.Errorf("任务状态应该是 %s，得到 %s", TaskStatusCancelled, task.GetStatus())
	}
}

// TestTaskClone 测试任务克隆
func TestTaskClone(t *testing.T) {
	video := &models.Video{ID: 1, BVid: "BV1xx411c7mD", Name: "测试视频"}
	task := NewDownloadTask(TaskTypeVideo, video, nil, "./downloads")
	task.Priority = PriorityHigh
	task.RetryCount = 2

	// 克隆任务
	cloned := task.Clone()

	if cloned.ID != task.ID {
		t.Error("克隆任务ID应该相同")
	}

	if cloned.Priority != task.Priority {
		t.Error("克隆任务优先级应该相同")
	}

	if cloned.RetryCount != task.RetryCount {
		t.Error("克隆任务重试次数应该相同")
	}

	if cloned.GetStatus() != TaskStatusPending {
		t.Error("克隆任务状态应该是 pending")
	}
}

// TestGenerateTaskID 测试任务ID生成
func TestGenerateTaskID(t *testing.T) {
	video := &models.Video{ID: 1, BVid: "BV1xx411c7mD", Name: "测试视频"}
	page := &models.Page{ID: 1, PID: 1, CID: 123456, Name: "第一集"}

	// 测试分P任务ID
	pageID := generateTaskID(TaskTypePage, video, page)
	expected := "page-1-1"
	if pageID != expected {
		t.Errorf("分P任务ID错误，期望 %s，得到 %s", expected, pageID)
	}

	// 测试视频任务ID
	videoID := generateTaskID(TaskTypeVideo, video, nil)
	expected = "video-1"
	if videoID != expected {
		t.Errorf("视频任务ID错误，期望 %s，得到 %s", expected, videoID)
	}
}

// BenchmarkQueueEnqueue 队列入队性能测试
func BenchmarkQueueEnqueue(b *testing.B) {
	queue := NewTaskQueue()
	video := &models.Video{ID: 1, BVid: "BV1xx411c7mD", Name: "测试视频"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		task := NewDownloadTask(TaskTypeVideo, video, nil, "./downloads")
		task.ID = fmt.Sprintf("task-%d", i)
		queue.Enqueue(task)
	}
}

// BenchmarkQueueDequeue 队列出队性能测试
func BenchmarkQueueDequeue(b *testing.B) {
	queue := NewTaskQueue()
	video := &models.Video{ID: 1, BVid: "BV1xx411c7mD", Name: "测试视频"}

	// 预先填充队列
	for i := 0; i < b.N; i++ {
		task := NewDownloadTask(TaskTypeVideo, video, nil, "./downloads")
		task.ID = fmt.Sprintf("task-%d", i)
		queue.Enqueue(task)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		queue.Dequeue()
	}
}

// BenchmarkSemaphoreAcquireRelease 信号量获取释放性能测试
func BenchmarkSemaphoreAcquireRelease(b *testing.B) {
	sem := NewSemaphore(10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sem.Acquire()
		sem.Release()
	}
}
