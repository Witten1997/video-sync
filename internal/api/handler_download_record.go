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
	if err := s.db.Preload("Video.Pages").First(&record, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			respondNotFound(c, "下载记录未找到")
			return
		}
		respondInternalError(c, err)
		return
	}

	if record.Status != "failed" {
		respondValidationError(c, "只能重试失败的记录")
		return
	}

	// 重置失败的文件状态
	var details models.FileDetailsData
	if err := json.Unmarshal(record.FileDetails, &details); err == nil {
		for i := range details.Files {
			if details.Files[i].Status == "failed" {
				details.Files[i].Status = "pending"
				details.Files[i].Progress = 0
				details.Files[i].Size = 0
			}
		}
		if updatedJSON, err := json.Marshal(details); err == nil {
			record.FileDetails = updatedJSON
		}
	}

	// 更新状态
	record.Status = "pending"
	record.ErrorMessage = ""
	record.CompletedAt = nil
	s.db.Save(&record)

	// 重新创建下载任务
	task, err := s.downloadMgr.PrepareAndAddVideoTask(&record.Video, s.config.Paths.DownloadBase, 0, true)
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
