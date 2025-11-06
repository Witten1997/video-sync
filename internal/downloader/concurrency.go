package downloader

import (
	"context"
	"sync"
)

// Semaphore 信号量实现
type Semaphore struct {
	ch chan struct{}
}

// NewSemaphore 创建新的信号量
func NewSemaphore(size int) *Semaphore {
	if size <= 0 {
		size = 1
	}
	return &Semaphore{
		ch: make(chan struct{}, size),
	}
}

// Acquire 获取信号量（阻塞）
func (s *Semaphore) Acquire() {
	s.ch <- struct{}{}
}

// TryAcquire 尝试获取信号量（非阻塞）
func (s *Semaphore) TryAcquire() bool {
	select {
	case s.ch <- struct{}{}:
		return true
	default:
		return false
	}
}

// AcquireWithContext 带上下文的获取（可取消）
func (s *Semaphore) AcquireWithContext(ctx context.Context) error {
	select {
	case s.ch <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Release 释放信号量
func (s *Semaphore) Release() {
	<-s.ch
}

// Available 获取可用许可数
func (s *Semaphore) Available() int {
	return cap(s.ch) - len(s.ch)
}

// Used 获取已使用许可数
func (s *Semaphore) Used() int {
	return len(s.ch)
}

// Capacity 获取总容量
func (s *Semaphore) Capacity() int {
	return cap(s.ch)
}

// ConcurrencyController 并发控制器
type ConcurrencyController struct {
	videoSem *Semaphore   // 视频级别信号量
	pageSem  *Semaphore   // 分P级别信号量
	videoMap sync.Map     // 正在处理的视频映射 (videoID -> count)
	mu       sync.RWMutex // 读写锁
}

// NewConcurrencyController 创建新的并发控制器
func NewConcurrencyController(maxVideos, maxPages int) *ConcurrencyController {
	return &ConcurrencyController{
		videoSem: NewSemaphore(maxVideos),
		pageSem:  NewSemaphore(maxPages),
	}
}

// AcquireVideo 获取视频级别许可
func (cc *ConcurrencyController) AcquireVideo(ctx context.Context) error {
	return cc.videoSem.AcquireWithContext(ctx)
}

// ReleaseVideo 释放视频级别许可
func (cc *ConcurrencyController) ReleaseVideo() {
	cc.videoSem.Release()
}

// AcquirePage 获取分P级别许可
func (cc *ConcurrencyController) AcquirePage(ctx context.Context) error {
	return cc.pageSem.AcquireWithContext(ctx)
}

// ReleasePage 释放分P级别许可
func (cc *ConcurrencyController) ReleasePage() {
	cc.pageSem.Release()
}

// CanStartVideo 检查是否可以开始下载视频
func (cc *ConcurrencyController) CanStartVideo() bool {
	return cc.videoSem.Available() > 0
}

// CanStartPage 检查是否可以开始下载分P
func (cc *ConcurrencyController) CanStartPage() bool {
	return cc.pageSem.Available() > 0
}

// GetVideoStats 获取视频并发统计
func (cc *ConcurrencyController) GetVideoStats() (used, available, total int) {
	return cc.videoSem.Used(), cc.videoSem.Available(), cc.videoSem.Capacity()
}

// GetPageStats 获取分P并发统计
func (cc *ConcurrencyController) GetPageStats() (used, available, total int) {
	return cc.pageSem.Used(), cc.pageSem.Available(), cc.pageSem.Capacity()
}

// TrackVideo 跟踪视频（记录正在处理的视频）
func (cc *ConcurrencyController) TrackVideo(videoID uint) {
	val, _ := cc.videoMap.LoadOrStore(videoID, &sync.WaitGroup{})
	wg := val.(*sync.WaitGroup)
	wg.Add(1)
}

// UntrackVideo 取消跟踪视频
func (cc *ConcurrencyController) UntrackVideo(videoID uint) {
	val, ok := cc.videoMap.Load(videoID)
	if !ok {
		return
	}

	wg := val.(*sync.WaitGroup)
	wg.Done()
}

// WaitVideo 等待视频所有分P完成
func (cc *ConcurrencyController) WaitVideo(videoID uint) {
	val, ok := cc.videoMap.Load(videoID)
	if !ok {
		return
	}

	wg := val.(*sync.WaitGroup)
	wg.Wait()
	cc.videoMap.Delete(videoID)
}

// IsVideoProcessing 检查视频是否正在处理
func (cc *ConcurrencyController) IsVideoProcessing(videoID uint) bool {
	_, ok := cc.videoMap.Load(videoID)
	return ok
}

// Stats 并发统计信息
type Stats struct {
	VideoUsed      int `json:"video_used"`
	VideoAvailable int `json:"video_available"`
	VideoTotal     int `json:"video_total"`
	PageUsed       int `json:"page_used"`
	PageAvailable  int `json:"page_available"`
	PageTotal      int `json:"page_total"`
}

// GetStats 获取统计信息
func (cc *ConcurrencyController) GetStats() Stats {
	videoUsed, videoAvail, videoTotal := cc.GetVideoStats()
	pageUsed, pageAvail, pageTotal := cc.GetPageStats()

	return Stats{
		VideoUsed:      videoUsed,
		VideoAvailable: videoAvail,
		VideoTotal:     videoTotal,
		PageUsed:       pageUsed,
		PageAvailable:  pageAvail,
		PageTotal:      pageTotal,
	}
}

// UpdateLimits 更新并发限制
func (cc *ConcurrencyController) UpdateLimits(maxVideos, maxPages int) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	// 重新创建信号量（因为channel容量无法动态修改）
	cc.videoSem = NewSemaphore(maxVideos)
	cc.pageSem = NewSemaphore(maxPages)
}
