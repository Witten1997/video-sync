package api

import (
	"bili-download/internal/database/models"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v3/disk"
)

// DashboardStats 仪表盘统计数据
type DashboardStats struct {
	// 视频源统计
	TotalSources    int `json:"total_sources"`
	ActiveSources   int `json:"active_sources"`
	FavoriteCount   int `json:"favorite_count"`
	WatchLaterCount int `json:"watch_later_count"`
	CollectionCount int `json:"collection_count"`
	SubmissionCount int `json:"submission_count"`

	// 视频统计
	TotalVideos      int `json:"total_videos"`
	DownloadedVideos int `json:"downloaded_videos"`
	PendingVideos    int `json:"pending_videos"`

	// 任务统计
	TotalTasks     int `json:"total_tasks"`
	RunningTasks   int `json:"running_tasks"`
	CompletedTasks int `json:"completed_tasks"`
	FailedTasks    int `json:"failed_tasks"`
	PendingTasks   int `json:"pending_tasks"`

	// 存储统计
	TotalSize  int64 `json:"total_size"`  // 总下载大小（字节）
	VideoCount int   `json:"video_count"` // 视频文件数量

	// 磁盘空间信息
	DiskTotal   uint64  `json:"disk_total"`    // 总空间（字节）
	DiskFree    uint64  `json:"disk_free"`     // 可用空间（字节）
	DiskUsed    uint64  `json:"disk_used"`     // 已用空间（字节）
	DiskUsedPct float64 `json:"disk_used_pct"` // 使用百分比
}

// handleDashboard 获取仪表盘统计数据
func (s *Server) handleDashboard(c *gin.Context) {
	stats := DashboardStats{}

	// 统计视频源
	var favoriteCount, watchLaterCount, collectionCount, submissionCount int64
	s.db.Model(&models.Favorite{}).Count(&favoriteCount)
	s.db.Model(&models.WatchLater{}).Count(&watchLaterCount)
	s.db.Model(&models.Collection{}).Count(&collectionCount)
	s.db.Model(&models.Submission{}).Count(&submissionCount)

	stats.FavoriteCount = int(favoriteCount)
	stats.WatchLaterCount = int(watchLaterCount)
	stats.CollectionCount = int(collectionCount)
	stats.SubmissionCount = int(submissionCount)
	stats.TotalSources = stats.FavoriteCount + stats.WatchLaterCount + stats.CollectionCount + stats.SubmissionCount

	// 统计启用的视频源
	var activeFavorite, activeWatchLater, activeCollection, activeSubmission int64
	s.db.Model(&models.Favorite{}).Where("enabled = ?", true).Count(&activeFavorite)
	s.db.Model(&models.WatchLater{}).Where("enabled = ?", true).Count(&activeWatchLater)
	s.db.Model(&models.Collection{}).Where("enabled = ?", true).Count(&activeCollection)
	s.db.Model(&models.Submission{}).Where("enabled = ?", true).Count(&activeSubmission)
	stats.ActiveSources = int(activeFavorite + activeWatchLater + activeCollection + activeSubmission)

	// 统计视频
	var totalVideos int64
	s.db.Model(&models.Video{}).Count(&totalVideos)
	stats.TotalVideos = int(totalVideos)

	// 统计已下载的视频（download_status != 0 表示已开始或已完成下载）
	var downloadedVideos int64
	s.db.Model(&models.Video{}).Where("download_status != ?", 0).Count(&downloadedVideos)
	stats.DownloadedVideos = int(downloadedVideos)

	// 统计待下载的视频
	stats.PendingVideos = stats.TotalVideos - stats.DownloadedVideos

	// 统计任务
	taskStats := s.downloadMgr.GetStats()
	stats.TotalTasks = taskStats.TotalTasks
	stats.RunningTasks = taskStats.RunningTasks
	stats.CompletedTasks = taskStats.CompletedTasks
	stats.FailedTasks = taskStats.FailedTasks
	stats.PendingTasks = taskStats.QueuedTasks

	// TODO: 实现存储统计（需要扫描下载目录）
	stats.TotalSize = 0
	stats.VideoCount = stats.TotalVideos

	// 获取磁盘空间信息
	if s.config.Paths.DownloadBase != "" {
		diskStats, err := disk.Usage(s.config.Paths.DownloadBase)
		if err == nil {
			stats.DiskTotal = diskStats.Total
			stats.DiskFree = diskStats.Free
			stats.DiskUsed = diskStats.Used
			stats.DiskUsedPct = diskStats.UsedPercent
		}
	}

	respondSuccess(c, stats)
}
