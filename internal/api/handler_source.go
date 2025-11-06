package api

import (
	"fmt"
	"strconv"

	"bili-download/internal/bilibili"
	"bili-download/internal/database/models"

	"github.com/gin-gonic/gin"
)

// SourceRequest 添加视频源请求
type SourceRequest struct {
	Type string `json:"type" binding:"required"` // favorite, watch_later, collection, submission
	URL  string `json:"url" binding:"required"`
	Name string `json:"name"`
}

// UpdateSourceRequest 更新视频源请求
type UpdateSourceRequest struct {
	Name    *string `json:"name"`    // 名称（可选）
	Path    *string `json:"path"`    // 保存路径（可选）
	Enabled *bool   `json:"enabled"` // 启用状态（可选）
}

// handleListSources 列出所有视频源
func (s *Server) handleListSources(c *gin.Context) {
	var sources []interface{}

	// 收藏夹
	var favorites []models.Favorite
	if err := s.db.Find(&favorites).Error; err != nil {
		respondInternalError(c, err)
		return
	}
	for _, fav := range favorites {
		sources = append(sources, gin.H{
			"id":            fav.ID,
			"type":          "favorite",
			"name":          fav.Name,
			"path":          fav.Path,
			"f_id":          strconv.FormatInt(fav.FID, 10),
			"enabled":       fav.Enabled,
			"latest_row_at": fav.LatestRowAt,
			"video_count":   len(fav.Videos),
			"created_at":    fav.CreatedAt,
		})
	}

	// 稍后再看
	var watchLaters []models.WatchLater
	if err := s.db.Find(&watchLaters).Error; err != nil {
		respondInternalError(c, err)
		return
	}
	for _, wl := range watchLaters {
		sources = append(sources, gin.H{
			"id":            wl.ID,
			"type":          "watch_later",
			"name":          wl.Name,
			"path":          wl.Path,
			"enabled":       wl.Enabled,
			"latest_row_at": wl.LatestRowAt,
			"video_count":   len(wl.Videos),
			"created_at":    wl.CreatedAt,
		})
	}

	// 合集
	var collections []models.Collection
	if err := s.db.Find(&collections).Error; err != nil {
		respondInternalError(c, err)
		return
	}
	for _, col := range collections {
		sources = append(sources, gin.H{
			"id":            col.ID,
			"type":          "collection",
			"name":          col.Name,
			"path":          col.Path,
			"cid":           col.CID,
			"enabled":       col.Enabled,
			"latest_row_at": col.LatestRowAt,
			"video_count":   len(col.Videos),
			"created_at":    col.CreatedAt,
		})
	}

	// UP主投稿
	var submissions []models.Submission
	if err := s.db.Find(&submissions).Error; err != nil {
		respondInternalError(c, err)
		return
	}
	for _, sub := range submissions {
		sources = append(sources, gin.H{
			"id":            sub.ID,
			"type":          "submission",
			"name":          sub.Name,
			"path":          sub.Path,
			"mid":           strconv.FormatInt(sub.UpperID, 10),
			"upper_id":      sub.UpperID,
			"enabled":       sub.Enabled,
			"latest_row_at": sub.LatestRowAt,
			"video_count":   len(sub.Videos),
			"created_at":    sub.CreatedAt,
		})
	}

	respondSuccess(c, gin.H{
		"items": sources,
		"total": len(sources),
	})
}

// handleAddSource 添加视频源
func (s *Server) handleAddSource(c *gin.Context) {
	var req SourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, err.Error())
		return
	}

	// 解析 URL
	parser := bilibili.NewURLParser()
	parsed, err := parser.Parse(req.URL)
	if err != nil {
		respondValidationError(c, fmt.Sprintf("URL 解析失败: %v", err))
		return
	}

	// 验证类型匹配
	if req.Type != "" && req.Type != string(parsed.Type) {
		respondValidationError(c, fmt.Sprintf("请求类型 (%s) 与 URL 类型 (%s) 不匹配", req.Type, parsed.Type))
		return
	}

	// 根据类型创建对应的视频源
	switch parsed.Type {
	case bilibili.SourceTypeFavorite:
		var favorite models.Favorite
		// 检查是否已存在
		if err := s.db.Where("f_id = ?", parsed.ID).First(&favorite).Error; err == nil {
			respondValidationError(c, fmt.Sprintf("收藏夹 (FID: %d) 已存在", parsed.ID))
			return
		}

		// 创建新收藏夹
		name := req.Name
		if name == "" {
			name = fmt.Sprintf("收藏夹-%d", parsed.ID)
		}
		favorite = models.Favorite{
			FID:     parsed.ID,
			Name:    name,
			Enabled: true,
		}
		if err := s.db.Create(&favorite).Error; err != nil {
			respondInternalError(c, fmt.Errorf("创建收藏夹失败: %w", err))
			return
		}
		respondSuccess(c, gin.H{
			"message": "添加收藏夹成功",
			"source":  favorite,
		})

	case bilibili.SourceTypeWatchLater:
		var watchLater models.WatchLater
		// 检查是否已存在
		if err := s.db.First(&watchLater).Error; err == nil {
			respondValidationError(c, "稍后再看已存在")
			return
		}

		// 创建稍后再看
		name := req.Name
		if name == "" {
			name = "稍后再看"
		}
		watchLater = models.WatchLater{
			Name:    name,
			Enabled: true,
		}
		if err := s.db.Create(&watchLater).Error; err != nil {
			respondInternalError(c, fmt.Errorf("创建稍后再看失败: %w", err))
			return
		}
		respondSuccess(c, gin.H{
			"message": "添加稍后再看成功",
			"source":  watchLater,
		})

	case bilibili.SourceTypeCollection:
		var collection models.Collection
		// 检查是否已存在
		if err := s.db.Where("c_id = ?", parsed.ID).First(&collection).Error; err == nil {
			respondValidationError(c, fmt.Sprintf("合集 (CID: %d) 已存在", parsed.ID))
			return
		}

		// 创建新合集
		name := req.Name
		if name == "" {
			name = fmt.Sprintf("合集-%d", parsed.ID)
		}
		collection = models.Collection{
			CID:     parsed.ID,
			CType:   parsed.SubType,
			Name:    name,
			Enabled: true,
		}
		if err := s.db.Create(&collection).Error; err != nil {
			respondInternalError(c, fmt.Errorf("创建合集失败: %w", err))
			return
		}
		respondSuccess(c, gin.H{
			"message": "添加合集成功",
			"source":  collection,
		})

	case bilibili.SourceTypeSubmission:
		var submission models.Submission
		// 检查是否已存在
		if err := s.db.Where("upper_id = ?", parsed.ID).First(&submission).Error; err == nil {
			respondValidationError(c, fmt.Sprintf("UP主投稿 (UpperID: %d) 已存在", parsed.ID))
			return
		}

		// 创建新 UP 主投稿
		name := req.Name
		if name == "" {
			name = fmt.Sprintf("UP主-%d", parsed.ID)
		}
		submission = models.Submission{
			UpperID: parsed.ID,
			Name:    name,
			Enabled: true,
		}
		if err := s.db.Create(&submission).Error; err != nil {
			respondInternalError(c, fmt.Errorf("创建UP主投稿失败: %w", err))
			return
		}
		respondSuccess(c, gin.H{
			"message": "添加UP主投稿成功",
			"source":  submission,
		})

	default:
		respondValidationError(c, fmt.Sprintf("不支持的视频源类型: %s", parsed.Type))
	}
}

// handleGetSource 获取视频源详情
func (s *Server) handleGetSource(c *gin.Context) {
	idStr := c.Param("id")
	sourceType := c.Query("type")

	if sourceType == "" {
		respondValidationError(c, "缺少 type 参数")
		return
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondValidationError(c, "无效的 ID")
		return
	}

	switch sourceType {
	case "favorite":
		var favorite models.Favorite
		if err := s.db.Preload("Videos").First(&favorite, id).Error; err != nil {
			respondNotFound(c, "收藏夹未找到")
			return
		}
		respondSuccess(c, favorite)

	case "watch_later":
		var watchLater models.WatchLater
		if err := s.db.Preload("Videos").First(&watchLater, id).Error; err != nil {
			respondNotFound(c, "稍后再看未找到")
			return
		}
		respondSuccess(c, watchLater)

	case "collection":
		var collection models.Collection
		if err := s.db.Preload("Videos").First(&collection, id).Error; err != nil {
			respondNotFound(c, "合集未找到")
			return
		}
		respondSuccess(c, collection)

	case "submission":
		var submission models.Submission
		if err := s.db.Preload("Videos").First(&submission, id).Error; err != nil {
			respondNotFound(c, "UP主投稿未找到")
			return
		}
		respondSuccess(c, submission)

	default:
		respondValidationError(c, fmt.Sprintf("不支持的视频源类型: %s", sourceType))
	}
}

// handleUpdateSource 更新视频源
func (s *Server) handleUpdateSource(c *gin.Context) {
	idStr := c.Param("id")
	sourceType := c.Query("type")

	if sourceType == "" {
		respondValidationError(c, "缺少 type 参数")
		return
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondValidationError(c, "无效的 ID")
		return
	}

	var req UpdateSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, err.Error())
		return
	}

	// 构建更新数据
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Path != nil {
		updates["path"] = *req.Path
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}

	// 如果没有任何更新字段，返回错误
	if len(updates) == 0 {
		respondValidationError(c, "没有提供任何更新字段")
		return
	}

	// 根据类型更新对应的视频源
	switch sourceType {
	case "favorite":
		if err := s.db.Model(&models.Favorite{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			respondInternalError(c, err)
			return
		}

	case "watch_later":
		if err := s.db.Model(&models.WatchLater{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			respondInternalError(c, err)
			return
		}

	case "collection":
		if err := s.db.Model(&models.Collection{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			respondInternalError(c, err)
			return
		}

	case "submission":
		if err := s.db.Model(&models.Submission{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			respondInternalError(c, err)
			return
		}

	default:
		respondValidationError(c, fmt.Sprintf("不支持的视频源类型: %s", sourceType))
		return
	}

	respondSuccess(c, gin.H{
		"message": "更新成功",
	})
}

// handleDeleteSource 删除视频源
func (s *Server) handleDeleteSource(c *gin.Context) {
	idStr := c.Param("id")
	sourceType := c.Query("type")

	if sourceType == "" {
		respondValidationError(c, "缺少 type 参数")
		return
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondValidationError(c, "无效的 ID")
		return
	}

	switch sourceType {
	case "favorite":
		if err := s.db.Delete(&models.Favorite{}, id).Error; err != nil {
			respondInternalError(c, err)
			return
		}

	case "watch_later":
		if err := s.db.Delete(&models.WatchLater{}, id).Error; err != nil {
			respondInternalError(c, err)
			return
		}

	case "collection":
		if err := s.db.Delete(&models.Collection{}, id).Error; err != nil {
			respondInternalError(c, err)
			return
		}

	case "submission":
		if err := s.db.Delete(&models.Submission{}, id).Error; err != nil {
			respondInternalError(c, err)
			return
		}

	default:
		respondValidationError(c, fmt.Sprintf("不支持的视频源类型: %s", sourceType))
		return
	}

	respondSuccess(c, gin.H{
		"message": "删除成功",
	})
}

// handleScanSource 扫描视频源
func (s *Server) handleScanSource(c *gin.Context) {
	idStr := c.Param("id")
	sourceType := c.Query("type")

	if sourceType == "" {
		respondValidationError(c, "缺少 type 参数")
		return
	}

	// TODO: 实现视频源扫描逻辑
	// 需要根据 type 从数据库读取对应的源配置，然后创建 adapter 进行扫描
	respondSuccess(c, gin.H{
		"message": "扫描视频源功能待实现",
		"id":      idStr,
		"type":    sourceType,
	})
}

// EnableSourceRequest 启用/禁用视频源请求
type EnableSourceRequest struct {
	Enabled bool `json:"enabled"`
}

// handleEnableSource 启用/禁用视频源
func (s *Server) handleEnableSource(c *gin.Context) {
	idStr := c.Param("id")
	sourceType := c.Query("type")

	if sourceType == "" {
		respondValidationError(c, "缺少 type 参数")
		return
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondValidationError(c, "无效的 ID")
		return
	}

	var req EnableSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, err.Error())
		return
	}

	// 根据类型更新对应的视频源
	switch sourceType {
	case "favorite":
		if err := s.db.Model(&models.Favorite{}).Where("id = ?", id).Update("enabled", req.Enabled).Error; err != nil {
			respondInternalError(c, err)
			return
		}

	case "watch_later":
		if err := s.db.Model(&models.WatchLater{}).Where("id = ?", id).Update("enabled", req.Enabled).Error; err != nil {
			respondInternalError(c, err)
			return
		}

	case "collection":
		if err := s.db.Model(&models.Collection{}).Where("id = ?", id).Update("enabled", req.Enabled).Error; err != nil {
			respondInternalError(c, err)
			return
		}

	case "submission":
		if err := s.db.Model(&models.Submission{}).Where("id = ?", id).Update("enabled", req.Enabled).Error; err != nil {
			respondInternalError(c, err)
			return
		}

	default:
		respondValidationError(c, fmt.Sprintf("不支持的视频源类型: %s", sourceType))
		return
	}

	respondSuccess(c, gin.H{
		"message": "操作成功",
		"id":      id,
		"type":    sourceType,
		"enabled": req.Enabled,
	})
}
