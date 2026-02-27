package api

import (
	"bili-download/internal/database/models"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

// SystemInfo 系统信息
type SystemInfo struct {
	Version   string    `json:"version"`
	GoVersion string    `json:"go_version"`
	OS        string    `json:"os"`
	Arch      string    `json:"arch"`
	StartTime time.Time `json:"start_time"`
	Uptime    string    `json:"uptime"`
}

var startTime = time.Now()

// handleHealth 健康检查
func (s *Server) handleHealth(c *gin.Context) {
	respondSuccess(c, gin.H{
		"status": "ok",
		"time":   time.Now(),
	})
}

// handleSystemInfo 获取系统信息
func (s *Server) handleSystemInfo(c *gin.Context) {
	uptime := time.Since(startTime)

	respondSuccess(c, SystemInfo{
		Version:   "1.0.0", // TODO: 从构建信息获取
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		StartTime: startTime,
		Uptime:    uptime.String(),
	})
}

// handleSystemStats 获取系统统计信息
func (s *Server) handleSystemStats(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// CPU信息
	cpuPercent, _ := cpu.Percent(time.Second, false)
	cpuValue := 0.0
	if len(cpuPercent) > 0 {
		cpuValue = cpuPercent[0]
	}
	cpuCores, _ := cpu.Counts(true)

	// 内存信息
	memStats, _ := mem.VirtualMemory()

	// 视频统计
	var totalVideos, downloadedVideos int64
	s.db.Model(&models.Video{}).Count(&totalVideos)
	s.db.Model(&models.Video{}).Where("download_status != ?", 0).Count(&downloadedVideos)

	// 任务统计
	taskStats := s.downloadMgr.GetStats()

	respondSuccess(c, gin.H{
		"cpu": gin.H{
			"percent": cpuValue,
			"cores":   cpuCores,
		},
		"memory": gin.H{
			"total":        memStats.Total,
			"used":         memStats.Used,
			"free":         memStats.Free,
			"used_percent": memStats.UsedPercent,
		},
		"go_runtime": gin.H{
			"goroutines":   runtime.NumGoroutine(),
			"version":      runtime.Version(),
			"heap_alloc":   m.HeapAlloc,
			"heap_sys":     m.HeapSys,
			"heap_objects": m.HeapObjects,
			"alloc":        m.Alloc,
			"total_alloc":  m.TotalAlloc,
			"sys":          m.Sys,
			"num_gc":       m.NumGC,
		},
		"download_manager": gin.H{
			"running": s.downloadMgr.IsRunning(),
			"stats":   s.downloadMgr.GetStats(),
		},
		"videos": gin.H{
			"total":      totalVideos,
			"downloaded": downloadedVideos,
			"pending":    totalVideos - downloadedVideos,
		},
		"tasks": gin.H{
			"total":     taskStats.TotalTasks,
			"running":   taskStats.RunningTasks,
			"completed": taskStats.CompletedTasks,
			"failed":    taskStats.FailedTasks,
			"pending":   taskStats.QueuedTasks,
		},
		"timestamp": time.Now().Unix(),
	})
}
