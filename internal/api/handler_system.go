package api

import (
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
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

	respondSuccess(c, gin.H{
		"goroutines": runtime.NumGoroutine(),
		"memory": gin.H{
			"alloc":       m.Alloc,
			"total_alloc": m.TotalAlloc,
			"sys":         m.Sys,
			"num_gc":      m.NumGC,
		},
		"download_manager": gin.H{
			"running": s.downloadMgr.IsRunning(),
			"stats":   s.downloadMgr.GetStats(),
		},
	})
}
