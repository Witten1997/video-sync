package api

import (
	"encoding/json"
	"strconv"

	"bili-download/internal/database/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// handleListDownloadRecords 获取下载记录列表
func (s *Server) handleListDownloadRecords(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")
	sourceType := c.Query("source_type")
	sourceID := c.Query("source_id")
	syncLogID := c.Query("sync_log_id")
	keyword := c.Query("keyword")

	query := s.db.Model(&models.DownloadRecord{}).Preload("Video")

	if status != "" && status != "all" {
		query = query.Where("download_records.status = ?", status)
	}
	if sourceType != "" {
		query = query.Where("download_records.source_type = ?", sourceType)
	}
	if sourceID != "" {
		query = query.Where("download_records.source_id = ?", sourceID)
	}
	if syncLogID != "" {
		query = query.Where("download_records.sync_log_id = ?", syncLogID)
	}
	if keyword != "" {
		query = query.Joins("JOIN video ON video.id = download_records.video_id").
			Where("video.name ILIKE ?", "%"+keyword+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		respondInternalError(c, err)
		return
	}

	var records []models.DownloadRecord
	if err := query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&records).Error; err != nil {
		respondInternalError(c, err)
		return
	}

	respondSuccess(c, gin.H{
		"items":       records,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// handleGetDownloadRecord 获取单条下载记录
func (s *Server) handleGetDownloadRecord(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		respondValidationError(c, "无效的ID")
		return
	}

	var record models.DownloadRecord
	if err := s.db.Preload("Video").First(&record, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			respondNotFound(c, "下载记录未找到")
			return
		}
		respondInternalError(c, err)
		return
	}

	respondSuccess(c, record)
}

// handleRetryDownloadRecord 重试下载记录
func (s *Server) handleRetryDownloadRecord(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		respondValidationError(c, "无效的ID")
		return
	}

	var record models.DownloadRecord
	if err := s.db.Preload("Video").First(&record, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			respondNotFound(c, "下载记录未找到")
			return
		}
		respondInternalError(c, err)
		return
	}

	if record.Status != "failed" && record.Status != "completed" {
		respondValidationError(c, "只能重试失败或已完成的记录")
		return
	}

	// 重置文件状态
	var details models.FileDetailsData
	if err := json.Unmarshal(record.FileDetails, &details); err == nil {
		for i := range details.Files {
			details.Files[i].Status = "pending"
			details.Files[i].Progress = 0
			details.Files[i].Size = 0
		}
		if updatedJSON, err := json.Marshal(details); err == nil {
			record.FileDetails = updatedJSON
		}
	}

	// 更新状态
	record.Status = "pending"
	record.ErrorMessage = ""
	record.StartedAt = nil
	record.CompletedAt = nil
	s.db.Save(&record)

	// 基于原有记录重试，不创建新记录
	task, err := s.downloadMgr.RetryVideoTask(record.ID, &record.Video, 0)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	respondSuccess(c, gin.H{
		"task_id":   task.ID,
		"record_id": record.ID,
		"message":   "重试任务已创建",
	})
}

// handleRepairDownloadRecords 修复误标记为完成的下载记录
func (s *Server) handleRepairDownloadRecords(c *gin.Context) {
	repaired, err := s.downloadMgr.RepairFalseCompletedRecords()
	if err != nil {
		respondInternalError(c, err)
		return
	}

	respondSuccess(c, gin.H{
		"repaired": repaired,
		"message":  "修复完成",
	})
}

// handleBatchRetryDownloadRecords 批量重试下载记录
func (s *Server) handleBatchRetryDownloadRecords(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 {
		respondValidationError(c, "请提供要重试的记录ID")
		return
	}

	retried := 0
	for _, id := range req.IDs {
		var record models.DownloadRecord
		if err := s.db.Preload("Video").First(&record, id).Error; err != nil {
			continue
		}
		if record.Status != "failed" && record.Status != "completed" {
			continue
		}

		// 重置文件状态
		var details models.FileDetailsData
		if err := json.Unmarshal(record.FileDetails, &details); err == nil {
			for i := range details.Files {
				details.Files[i].Status = "pending"
				details.Files[i].Progress = 0
				details.Files[i].Size = 0
			}
			if updatedJSON, err := json.Marshal(details); err == nil {
				record.FileDetails = updatedJSON
			}
		}

		record.Status = "pending"
		record.ErrorMessage = ""
		record.StartedAt = nil
		record.CompletedAt = nil
		s.db.Save(&record)

		if _, err := s.downloadMgr.RetryVideoTask(record.ID, &record.Video, 0); err == nil {
			retried++
		}
	}

	respondSuccess(c, gin.H{
		"retried": retried,
		"message": "批量重试完成",
	})
}

// handleDeleteDownloadRecord 删除下载记录
func (s *Server) handleDeleteDownloadRecord(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		respondValidationError(c, "无效的ID")
		return
	}

	if err := s.db.Delete(&models.DownloadRecord{}, id).Error; err != nil {
		respondInternalError(c, err)
		return
	}

	respondSuccess(c, gin.H{"message": "删除成功"})
}

// handleBatchDeleteDownloadRecords 批量删除下载记录
func (s *Server) handleBatchDeleteDownloadRecords(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 {
		respondValidationError(c, "请提供要删除的记录ID")
		return
	}

	if err := s.db.Delete(&models.DownloadRecord{}, req.IDs).Error; err != nil {
		respondInternalError(c, err)
		return
	}

	respondSuccess(c, gin.H{"message": "批量删除成功", "count": len(req.IDs)})
}
