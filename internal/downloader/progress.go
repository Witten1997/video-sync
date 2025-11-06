package downloader

import (
	"sync"
	"time"
)

// DownloadStatus 下载状态枚举
type DownloadStatus string

const (
	StatusPending     DownloadStatus = "pending"      // 待处理
	StatusDownloading DownloadStatus = "downloading"  // 下载中
	StatusSucceeded   DownloadStatus = "succeeded"    // 成功
	StatusSkipped     DownloadStatus = "skipped"      // 跳过
	StatusFailed      DownloadStatus = "failed"       // 失败（可重试）
	StatusFixedFailed DownloadStatus = "fixed_failed" // 永久失败
	StatusIgnored     DownloadStatus = "ignored"      // 忽略的错误
)

// ProgressInfo yt-dlp 进度信息
type ProgressInfo struct {
	Status          string  `json:"status"`               // downloading, finished, error
	Filename        string  `json:"filename"`             // 文件名
	TmpFilename     string  `json:"tmpfilename"`          // 临时文件名
	DownloadedBytes int64   `json:"downloaded_bytes"`     // 已下载字节
	TotalBytes      int64   `json:"total_bytes"`          // 总字节数
	TotalBytesEst   int64   `json:"total_bytes_estimate"` // 估计总字节数
	Speed           float64 `json:"speed"`                // 下载速度 (bytes/sec)
	ETA             float64 `json:"eta"`                  // 预计剩余时间 (秒)
	Elapsed         float64 `json:"elapsed"`              // 已用时间 (秒)
	Percentage      float64 `json:"percentage"`           // 下载百分比
	FragmentIndex   int     `json:"fragment_index"`       // 分片索引
	FragmentCount   int     `json:"fragment_count"`       // 分片总数
}

// SubTaskProgress 子任务进度
type SubTaskProgress struct {
	Name           string         `json:"name"`            // 子任务名称（video, poster, nfo, danmaku, subtitle, upper）
	Status         DownloadStatus `json:"status"`          // 下载状态
	Progress       float64        `json:"progress"`        // 进度百分比 (0-100)
	Speed          float64        `json:"speed"`           // 下载速度 (bytes/sec)
	DownloadedSize int64          `json:"downloaded_size"` // 已下载大小
	TotalSize      int64          `json:"total_size"`      // 总大小
	ETA            float64        `json:"eta"`             // 预计剩余时间 (秒)
	Error          string         `json:"error,omitempty"` // 错误信息
	RetryCount     int            `json:"retry_count"`     // 重试次数
	StartTime      time.Time      `json:"start_time"`      // 开始时间
	EndTime        time.Time      `json:"end_time"`        // 结束时间
}

// PageProgress 分P进度
type PageProgress struct {
	PageID    uint                        `json:"page_id"`    // 分P ID
	CID       int64                       `json:"cid"`        // CID
	PID       int                         `json:"pid"`        // 分P编号
	Name      string                      `json:"name"`       // 分P名称
	Status    DownloadStatus              `json:"status"`     // 整体状态
	SubTasks  map[string]*SubTaskProgress `json:"sub_tasks"`  // 子任务进度
	StartTime time.Time                   `json:"start_time"` // 开始时间
	EndTime   time.Time                   `json:"end_time"`   // 结束时间
	mu        sync.RWMutex                `json:"-"`          // 读写锁
}

// VideoProgress 视频进度
type VideoProgress struct {
	VideoID    uint                  `json:"video_id"`    // 视频 ID
	BVid       string                `json:"bvid"`        // BV号
	Title      string                `json:"title"`       // 视频标题
	Status     DownloadStatus        `json:"status"`      // 整体状态
	Pages      map[int]*PageProgress `json:"pages"`       // 分P进度（key为PID）
	TotalPages int                   `json:"total_pages"` // 总分P数
	StartTime  time.Time             `json:"start_time"`  // 开始时间
	EndTime    time.Time             `json:"end_time"`    // 结束时间
	mu         sync.RWMutex          `json:"-"`           // 读写锁
}

// NewPageProgress 创建分P进度
func NewPageProgress(pageID uint, cid int64, pid int, name string) *PageProgress {
	return &PageProgress{
		PageID:    pageID,
		CID:       cid,
		PID:       pid,
		Name:      name,
		Status:    StatusPending,
		SubTasks:  make(map[string]*SubTaskProgress),
		StartTime: time.Now(),
	}
}

// NewVideoProgress 创建视频进度
func NewVideoProgress(videoID uint, bvid, title string, totalPages int) *VideoProgress {
	return &VideoProgress{
		VideoID:    videoID,
		BVid:       bvid,
		Title:      title,
		Status:     StatusPending,
		Pages:      make(map[int]*PageProgress),
		TotalPages: totalPages,
		StartTime:  time.Now(),
	}
}

// UpdateSubTask 更新子任务进度
func (p *PageProgress) UpdateSubTask(name string, update func(*SubTaskProgress)) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.SubTasks[name]; !exists {
		p.SubTasks[name] = &SubTaskProgress{
			Name:      name,
			Status:    StatusPending,
			StartTime: time.Now(),
		}
	}

	update(p.SubTasks[name])
}

// GetSubTask 获取子任务进度
func (p *PageProgress) GetSubTask(name string) *SubTaskProgress {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if task, exists := p.SubTasks[name]; exists {
		// 返回副本以避免并发问题
		taskCopy := *task
		return &taskCopy
	}
	return nil
}

// UpdateStatus 更新状态
func (p *PageProgress) UpdateStatus(status DownloadStatus) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.Status = status
	if status == StatusSucceeded || status == StatusFailed || status == StatusFixedFailed {
		p.EndTime = time.Now()
	}
}

// GetOverallProgress 获取整体进度百分比
func (p *PageProgress) GetOverallProgress() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.SubTasks) == 0 {
		return 0
	}

	var totalProgress float64
	for _, task := range p.SubTasks {
		totalProgress += task.Progress
	}

	return totalProgress / float64(len(p.SubTasks))
}

// AddPage 添加分P进度
func (v *VideoProgress) AddPage(pid int, pageProgress *PageProgress) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.Pages[pid] = pageProgress
}

// GetPage 获取分P进度
func (v *VideoProgress) GetPage(pid int) *PageProgress {
	v.mu.RLock()
	defer v.mu.RUnlock()

	return v.Pages[pid]
}

// UpdateStatus 更新视频状态
func (v *VideoProgress) UpdateStatus(status DownloadStatus) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.Status = status
	if status == StatusSucceeded || status == StatusFailed || status == StatusFixedFailed {
		v.EndTime = time.Now()
	}
}

// GetOverallProgress 获取视频整体进度
func (v *VideoProgress) GetOverallProgress() float64 {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if len(v.Pages) == 0 {
		return 0
	}

	var totalProgress float64
	for _, page := range v.Pages {
		totalProgress += page.GetOverallProgress()
	}

	return totalProgress / float64(len(v.Pages))
}

// IsCompleted 检查是否全部完成
func (v *VideoProgress) IsCompleted() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()

	for _, page := range v.Pages {
		if page.Status != StatusSucceeded && page.Status != StatusSkipped {
			return false
		}
	}

	return len(v.Pages) > 0
}

// HasFailures 检查是否有失败的任务
func (v *VideoProgress) HasFailures() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()

	for _, page := range v.Pages {
		if page.Status == StatusFailed || page.Status == StatusFixedFailed {
			return true
		}
		for _, task := range page.SubTasks {
			if task.Status == StatusFailed || task.Status == StatusFixedFailed {
				return true
			}
		}
	}

	return false
}

// ProgressCallback 进度回调函数类型
type ProgressCallback func(videoID uint, pid int, taskName string, progress *SubTaskProgress)

// ProgressTracker 进度追踪器
type ProgressTracker struct {
	videos   map[uint]*VideoProgress // 视频进度映射
	mu       sync.RWMutex            // 读写锁
	callback ProgressCallback        // 进度回调
}

// NewProgressTracker 创建进度追踪器
func NewProgressTracker() *ProgressTracker {
	return &ProgressTracker{
		videos: make(map[uint]*VideoProgress),
	}
}

// SetCallback 设置进度回调
func (t *ProgressTracker) SetCallback(callback ProgressCallback) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.callback = callback
}

// AddVideo 添加视频进度
func (t *ProgressTracker) AddVideo(videoID uint, bvid, title string, totalPages int) *VideoProgress {
	t.mu.Lock()
	defer t.mu.Unlock()

	progress := NewVideoProgress(videoID, bvid, title, totalPages)
	t.videos[videoID] = progress
	return progress
}

// GetVideo 获取视频进度
func (t *ProgressTracker) GetVideo(videoID uint) *VideoProgress {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.videos[videoID]
}

// RemoveVideo 移除视频进度
func (t *ProgressTracker) RemoveVideo(videoID uint) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.videos, videoID)
}

// NotifyProgress 通知进度更新
func (t *ProgressTracker) NotifyProgress(videoID uint, pid int, taskName string, progress *SubTaskProgress) {
	t.mu.RLock()
	callback := t.callback
	t.mu.RUnlock()

	if callback != nil {
		callback(videoID, pid, taskName, progress)
	}
}

// GetAllVideos 获取所有视频进度
func (t *ProgressTracker) GetAllVideos() []*VideoProgress {
	t.mu.RLock()
	defer t.mu.RUnlock()

	videos := make([]*VideoProgress, 0, len(t.videos))
	for _, v := range t.videos {
		videos = append(videos, v)
	}

	return videos
}
