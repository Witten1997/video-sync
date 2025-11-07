package api

import (
	"strconv"
	"time"

	"bili-download/internal/database/models"
	"bili-download/internal/utils"

	"github.com/gin-gonic/gin"
)

// handleSchedulerStatus 获取调度器状态
func (s *Server) handleSchedulerStatus(c *gin.Context) {
	status := s.scheduler.GetStatus()

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data":    status,
	})
}

// handleSchedulerStart 启动调度器
func (s *Server) handleSchedulerStart(c *gin.Context) {
	if err := s.scheduler.Start(); err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": err.Error(),
		})
		return
	}

	status := s.scheduler.GetStatus()
	c.JSON(200, gin.H{
		"code":    0,
		"message": "调度器已启动",
		"data":    status,
	})
}

// handleSchedulerStop 停止调度器
func (s *Server) handleSchedulerStop(c *gin.Context) {
	if err := s.scheduler.Stop(); err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": err.Error(),
		})
		return
	}

	status := s.scheduler.GetStatus()
	c.JSON(200, gin.H{
		"code":    0,
		"message": "调度器已停止",
		"data":    status,
	})
}

// handleSchedulerTrigger 手动触发同步
func (s *Server) handleSchedulerTrigger(c *gin.Context) {
	syncID, err := s.scheduler.TriggerManual()
	if err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "同步任务已触发",
		"data": gin.H{
			"sync_id":    syncID,
			"started_at": time.Now(),
		},
	})
}

// handleListSyncLogs 获取同步日志列表
func (s *Server) handleListSyncLogs(c *gin.Context) {
	// 获取查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	triggerType := c.Query("trigger_type")
	status := c.Query("status")
	sortBy := c.DefaultQuery("sort_by", "start_at")
	sortOrder := c.DefaultQuery("sort_order", "desc")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 构建查询
	query := s.db.Model(&models.SyncLog{})

	// 过滤条件
	if triggerType != "" && triggerType != "all" {
		query = query.Where("trigger_type = ?", triggerType)
	}
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	// 获取总数
	var total int64
	query.Count(&total)

	// 排序
	orderClause := sortBy
	if sortOrder == "desc" {
		orderClause += " DESC"
	} else {
		orderClause += " ASC"
	}
	query = query.Order(orderClause)

	// 分页
	offset := (page - 1) * pageSize
	query = query.Offset(offset).Limit(pageSize)

	// 查询数据
	var logs []models.SyncLog
	if err := query.Find(&logs).Error; err != nil {
		utils.Error("查询同步日志失败: %v", err)
		c.JSON(500, gin.H{
			"code":    500,
			"message": "查询失败",
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"items":     logs,
		},
	})
}

// handleGetSyncLog 获取同步日志详情
func (s *Server) handleGetSyncLog(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "无效的日志ID",
		})
		return
	}

	var log models.SyncLog
	if err := s.db.Preload("SourceScans").First(&log, id).Error; err != nil {
		c.JSON(404, gin.H{
			"code":    404,
			"message": "日志不存在",
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data":    log,
	})
}

// handleGetSyncStats 获取同步统计信息
func (s *Server) handleGetSyncStats(c *gin.Context) {
	period := c.DefaultQuery("period", "7d")

	// 计算时间范围
	var startTime time.Time
	switch period {
	case "1d":
		startTime = time.Now().AddDate(0, 0, -1)
	case "7d":
		startTime = time.Now().AddDate(0, 0, -7)
	case "30d":
		startTime = time.Now().AddDate(0, 0, -30)
	default:
		startTime = time.Time{} // 所有时间
	}

	// 查询统计数据
	var totalSyncs int64
	var successfulSyncs int64
	var failedSyncs int64

	query := s.db.Model(&models.SyncLog{})
	if !startTime.IsZero() {
		query = query.Where("start_at >= ?", startTime)
	}

	query.Count(&totalSyncs)
	s.db.Model(&models.SyncLog{}).Where("status = ?", "completed").Where("start_at >= ?", startTime).Count(&successfulSyncs)
	s.db.Model(&models.SyncLog{}).Where("status = ?", "failed").Where("start_at >= ?", startTime).Count(&failedSyncs)

	// 计算成功率
	var successRate float64
	if totalSyncs > 0 {
		successRate = float64(successfulSyncs) / float64(totalSyncs) * 100
	}

	// 统计视频数
	var totalVideosFound, totalVideosNew, totalVideosQueued int64
	s.db.Model(&models.SyncLog{}).Select("COALESCE(SUM(videos_found), 0)").Where("start_at >= ?", startTime).Scan(&totalVideosFound)
	s.db.Model(&models.SyncLog{}).Select("COALESCE(SUM(videos_new), 0)").Where("start_at >= ?", startTime).Scan(&totalVideosNew)
	s.db.Model(&models.SyncLog{}).Select("COALESCE(SUM(videos_queued), 0)").Where("start_at >= ?", startTime).Scan(&totalVideosQueued)

	// 平均耗时
	var avgDuration float64
	s.db.Model(&models.SyncLog{}).Select("COALESCE(AVG(duration_ms), 0)").Where("start_at >= ?", startTime).Where("duration_ms > 0").Scan(&avgDuration)

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"period":              period,
			"total_syncs":         totalSyncs,
			"successful_syncs":    successfulSyncs,
			"failed_syncs":        failedSyncs,
			"success_rate":        successRate,
			"total_videos_found":  totalVideosFound,
			"total_videos_new":    totalVideosNew,
			"total_videos_queued": totalVideosQueued,
			"avg_duration_ms":     avgDuration,
		},
	})
}

// handleGetTasksSummary 获取任务统计
func (s *Server) handleGetTasksSummary(c *gin.Context) {
	// 从 DownloadManager 获取任务统计
	stats := s.downloadMgr.GetStats()

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"pending":   0,                    // 暂未实现 pending 状态
			"queued":    stats.QueuedTasks,    // 排队中
			"running":   stats.RunningTasks,   // 运行中
			"completed": stats.CompletedTasks, // 已完成
			"failed":    stats.FailedTasks,    // 失败
			"total":     stats.TotalTasks,     // 总计
		},
	})
}

// registerSchedulerRoutes 注册调度器相关路由
func (s *Server) registerSchedulerRoutes(r *gin.RouterGroup) {
	scheduler := r.Group("/scheduler")
	{
		// 调度器控制
		scheduler.GET("/status", s.handleSchedulerStatus)
		scheduler.POST("/start", s.handleSchedulerStart)
		scheduler.POST("/stop", s.handleSchedulerStop)
		scheduler.POST("/trigger", s.handleSchedulerTrigger)

		// 同步日志
		scheduler.GET("/logs", s.handleListSyncLogs)
		scheduler.GET("/logs/:id", s.handleGetSyncLog)
		scheduler.GET("/stats", s.handleGetSyncStats)

		// 任务管理
		scheduler.GET("/tasks/summary", s.handleGetTasksSummary)
	}
}
