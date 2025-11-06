package downloader

import (
	"container/heap"
	"sync"
)

// TaskQueue 任务队列（优先级队列）
type TaskQueue struct {
	items []*DownloadTask
	mu    sync.RWMutex
	index map[string]int // 任务ID到索引的映射，用于快速查找
}

// NewTaskQueue 创建新的任务队列
func NewTaskQueue() *TaskQueue {
	tq := &TaskQueue{
		items: make([]*DownloadTask, 0),
		index: make(map[string]int),
	}
	heap.Init(tq)
	return tq
}

// Len 实现 heap.Interface
func (tq *TaskQueue) Len() int {
	return len(tq.items)
}

// Less 实现 heap.Interface（优先级高的排在前面）
func (tq *TaskQueue) Less(i, j int) bool {
	// 优先级高的排前面，如果优先级相同，创建时间早的排前面
	if tq.items[i].Priority == tq.items[j].Priority {
		return tq.items[i].CreatedAt.Before(tq.items[j].CreatedAt)
	}
	return tq.items[i].Priority > tq.items[j].Priority
}

// Swap 实现 heap.Interface
func (tq *TaskQueue) Swap(i, j int) {
	tq.items[i], tq.items[j] = tq.items[j], tq.items[i]
	tq.index[tq.items[i].ID] = i
	tq.index[tq.items[j].ID] = j
}

// Push 实现 heap.Interface
func (tq *TaskQueue) Push(x interface{}) {
	task := x.(*DownloadTask)
	tq.index[task.ID] = len(tq.items)
	tq.items = append(tq.items, task)
}

// Pop 实现 heap.Interface
func (tq *TaskQueue) Pop() interface{} {
	old := tq.items
	n := len(old)
	task := old[n-1]
	old[n-1] = nil // 避免内存泄漏
	tq.items = old[0 : n-1]
	delete(tq.index, task.ID)
	return task
}

// Enqueue 入队（线程安全）
func (tq *TaskQueue) Enqueue(task *DownloadTask) {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	// 检查是否已存在
	if _, exists := tq.index[task.ID]; exists {
		return
	}

	task.SetStatus(TaskStatusQueued)
	heap.Push(tq, task)
}

// Dequeue 出队（线程安全）
func (tq *TaskQueue) Dequeue() *DownloadTask {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	if tq.Len() == 0 {
		return nil
	}

	task := heap.Pop(tq).(*DownloadTask)
	return task
}

// Peek 查看队首元素（不移除）
func (tq *TaskQueue) Peek() *DownloadTask {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	if tq.Len() == 0 {
		return nil
	}

	return tq.items[0]
}

// Remove 移除指定任务
func (tq *TaskQueue) Remove(taskID string) *DownloadTask {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	idx, exists := tq.index[taskID]
	if !exists {
		return nil
	}

	task := heap.Remove(tq, idx).(*DownloadTask)
	return task
}

// Get 获取指定任务（不移除）
func (tq *TaskQueue) Get(taskID string) *DownloadTask {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	idx, exists := tq.index[taskID]
	if !exists {
		return nil
	}

	return tq.items[idx]
}

// Contains 检查是否包含指定任务
func (tq *TaskQueue) Contains(taskID string) bool {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	_, exists := tq.index[taskID]
	return exists
}

// Size 获取队列大小
func (tq *TaskQueue) Size() int {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	return tq.Len()
}

// Clear 清空队列
func (tq *TaskQueue) Clear() {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	tq.items = make([]*DownloadTask, 0)
	tq.index = make(map[string]int)
	heap.Init(tq)
}

// GetAll 获取所有任务（返回副本）
func (tq *TaskQueue) GetAll() []*DownloadTask {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	tasks := make([]*DownloadTask, len(tq.items))
	copy(tasks, tq.items)
	return tasks
}

// UpdatePriority 更新任务优先级
func (tq *TaskQueue) UpdatePriority(taskID string, priority TaskPriority) bool {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	idx, exists := tq.index[taskID]
	if !exists {
		return false
	}

	tq.items[idx].Priority = priority
	heap.Fix(tq, idx)
	return true
}
