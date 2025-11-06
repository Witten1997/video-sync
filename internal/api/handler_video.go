package api

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"bili-download/internal/database/models"
	"bili-download/internal/utils"

	"github.com/gin-gonic/gin"
)

// handleListVideos 列出所有视频
func (s *Server) handleListVideos(c *gin.Context) {
	// 获取查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	sourceType := c.Query("source_type")
	sourceID := c.Query("source_id")
	keyword := c.Query("keyword")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := s.db.Model(&models.Video{})

	// 按视频源类型过滤
	if sourceType != "" {
		if sourceID != "" {
			// 如果同时提供了 source_type 和 source_id，精确匹配
			switch sourceType {
			case "favorite":
				query = query.Where("favorite_id = ?", sourceID)
			case "watch_later":
				query = query.Where("watch_later_id = ?", sourceID)
			case "collection":
				query = query.Where("collection_id = ?", sourceID)
			case "submission":
				query = query.Where("submission_id = ?", sourceID)
			}
		} else {
			// 如果只提供了 source_type，过滤该类型的所有视频
			switch sourceType {
			case "favorite":
				query = query.Where("favorite_id IS NOT NULL")
			case "watch_later":
				query = query.Where("watch_later_id IS NOT NULL")
			case "collection":
				query = query.Where("collection_id IS NOT NULL")
			case "submission":
				query = query.Where("submission_id IS NOT NULL")
			case "url":
				// URL 下载的视频：没有任何视频源关联
				query = query.Where("favorite_id IS NULL AND watch_later_id IS NULL AND collection_id IS NULL AND submission_id IS NULL")
			}
		}
	}

	// 按关键词过滤（搜索标题或BV号）
	if keyword != "" {
		query = query.Where("name LIKE ? OR bvid LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		respondInternalError(c, err)
		return
	}

	// 分页查询
	var videos []models.Video
	offset := (page - 1) * pageSize
	if err := query.Preload("Pages").Offset(offset).Limit(pageSize).Find(&videos).Error; err != nil {
		respondInternalError(c, err)
		return
	}

	// 处理封面路径，优先使用本地封面
	for i := range videos {
		s.resolveVideoCoverPaths(&videos[i])
	}

	respondSuccess(c, gin.H{
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (int(total) + pageSize - 1) / pageSize,
		"items":       videos,
	})
}

// handleGetVideo 获取视频详情
func (s *Server) handleGetVideo(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondValidationError(c, "无效的视频 ID")
		return
	}

	var video models.Video
	if err := s.db.Preload("Pages").First(&video, id).Error; err != nil {
		respondNotFound(c, "视频未找到")
		return
	}

	// 处理封面路径，优先使用本地封面
	s.resolveVideoCoverPaths(&video)

	respondSuccess(c, video)
}

// handleUpdateVideo 更新视频信息
func (s *Server) handleUpdateVideo(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondValidationError(c, "无效的视频 ID")
		return
	}

	var video models.Video
	if err := s.db.First(&video, id).Error; err != nil {
		respondNotFound(c, "视频未找到")
		return
	}

	// 绑定更新数据
	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		respondValidationError(c, err.Error())
		return
	}

	// 更新
	if err := s.db.Model(&video).Updates(updateData).Error; err != nil {
		respondInternalError(c, err)
		return
	}

	respondSuccess(c, video)
}

// handleDeleteVideo 删除视频
func (s *Server) handleDeleteVideo(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondValidationError(c, "无效的视频 ID")
		return
	}

	// 先查询视频信息，获取本地文件路径
	var video models.Video
	if err := s.db.Preload("Pages").First(&video, id).Error; err != nil {
		respondNotFound(c, "视频未找到")
		return
	}

	// 删除本地文件
	if err := s.deleteLocalFiles(&video); err != nil {
		utils.Warn("删除本地文件失败: %v", err)
		// 继续删除数据库记录，即使文件删除失败
	}

	// 删除视频（会级联删除相关的分P）
	if err := s.db.Delete(&models.Video{}, id).Error; err != nil {
		respondInternalError(c, err)
		return
	}

	respondSuccess(c, gin.H{
		"message": "删除成功",
	})
}

// handleDownloadVideo 下载视频
func (s *Server) handleDownloadVideo(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondValidationError(c, "无效的视频 ID")
		return
	}

	var video models.Video
	if err := s.db.Preload("Pages").First(&video, id).Error; err != nil {
		respondNotFound(c, "视频未找到")
		return
	}

	// 为视频创建独立文件夹
	videoFolderName := utils.Filenamify(video.Name)
	outputDir := filepath.Join(s.config.Paths.DownloadBase, videoFolderName)

	// 创建目录
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		respondInternalError(c, fmt.Errorf("创建视频目录失败: %w", err))
		return
	}

	// 保存路径到数据库
	video.Path = outputDir
	s.db.Save(&video)

	// 创建下载任务
	task, err := s.downloadMgr.AddVideoTask(&video, outputDir, 0)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	respondSuccess(c, gin.H{
		"task_id": task.ID,
		"video":   video,
		"message": "下载任务已创建",
	})
}

// handleGetVideoPages 获取视频的所有分P
func (s *Server) handleGetVideoPages(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		respondValidationError(c, "无效的视频 ID")
		return
	}

	var pages []models.Page
	if err := s.db.Where("video_id = ?", id).Find(&pages).Error; err != nil {
		respondInternalError(c, err)
		return
	}

	respondSuccess(c, pages)
}

// DownloadByURLRequest 通过URL下载视频的请求
type DownloadByURLRequest struct {
	URL string `json:"url" binding:"required"`
}

// handleDownloadByURL 通过B站视频链接下载视频
func (s *Server) handleDownloadByURL(c *gin.Context) {
	var req DownloadByURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, "缺少 URL 参数")
		return
	}

	// 1. 解析URL获取BVID
	bvid, err := s.biliClient.ParseVideoURL(req.URL)
	if err != nil {
		respondValidationError(c, "无效的B站视频链接: "+err.Error())
		return
	}

	// 2. 检查视频是否已存在
	var existingVideo models.Video
	err = s.db.Where("bvid = ?", bvid).Preload("Pages").First(&existingVideo).Error
	if err == nil {
		// 视频已存在,为其创建独立文件夹并下载
		videoFolderName := utils.Filenamify(existingVideo.Name)
		outputDir := filepath.Join(s.config.Paths.DownloadBase, videoFolderName)

		// 创建目录
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			respondInternalError(c, fmt.Errorf("创建视频目录失败: %w", err))
			return
		}

		// 保存路径到数据库
		existingVideo.Path = outputDir
		s.db.Save(&existingVideo)

		task, err := s.downloadMgr.AddVideoTask(&existingVideo, outputDir, 0)
		if err != nil {
			respondInternalError(c, err)
			return
		}

		respondSuccess(c, gin.H{
			"task_id": task.ID,
			"video":   existingVideo,
			"message": "视频已存在，下载任务已创建",
		})
		return
	}

	// 3. 获取视频详细信息
	videoDetail, err := s.biliClient.GetVideoDetail(bvid)
	if err != nil {
		respondError(c, 500, "获取视频信息失败: "+err.Error())
		return
	}

	// 4. 创建视频记录
	video := models.Video{
		BVid:           videoDetail.BVid,
		Name:           videoDetail.Title,
		Intro:          videoDetail.Desc,
		Cover:          videoDetail.Pic,
		UpperID:        videoDetail.Owner.Mid,
		UpperName:      videoDetail.Owner.Name,
		UpperFace:      videoDetail.Owner.Face,
		Category:       videoDetail.Tid,
		PubTime:        time.Unix(videoDetail.PubDate, 0),
		FavTime:        time.Unix(videoDetail.PubDate, 0),
		CTime:          time.Unix(videoDetail.CTime, 0),
		SinglePage:     len(videoDetail.Pages) == 1,
		Valid:          true,
		ShouldDownload: true,
	}

	// 获取并保存标签
	videoTags, err := s.biliClient.GetVideoTags(bvid)
	if err == nil && len(videoTags) > 0 {
		tags := make([]string, len(videoTags))
		for i, tag := range videoTags {
			tags[i] = tag.TagName
		}
		video.Tags = tags
	}

	// 5. 保存视频到数据库
	if err := s.db.Create(&video).Error; err != nil {
		respondInternalError(c, err)
		return
	}

	// 6. 保存分P信息
	for _, page := range videoDetail.Pages {
		dbPage := models.Page{
			VideoID:  video.ID,
			CID:      page.CID,
			PID:      page.Page,
			Name:     page.Part,
			Duration: page.Duration,
			Width:    page.Dimension.Width,
			Height:   page.Dimension.Height,
			Image:    page.FirstFrame,
		}

		if err := s.db.Create(&dbPage).Error; err != nil {
			respondInternalError(c, err)
			return
		}
	}

	// 7. 重新加载视频（包括分P）
	if err := s.db.Preload("Pages").First(&video, video.ID).Error; err != nil {
		respondInternalError(c, err)
		return
	}

	// 8. 为视频创建独立文件夹
	videoFolderName := utils.Filenamify(video.Name)
	outputDir := filepath.Join(s.config.Paths.DownloadBase, videoFolderName)

	// 创建目录
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		respondInternalError(c, fmt.Errorf("创建视频目录失败: %w", err))
		return
	}

	// 保存路径到数据库
	video.Path = outputDir
	s.db.Save(&video)

	// 9. 创建下载任务
	task, err := s.downloadMgr.AddVideoTask(&video, outputDir, 0)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	respondSuccess(c, gin.H{
		"task_id": task.ID,
		"video":   video,
		"message": "视频信息已获取，下载任务已创建",
	})
}

// resolveVideoCoverPaths 解析视频封面路径，优先使用本地封面
func (s *Server) resolveVideoCoverPaths(video *models.Video) {
	downloadDir := s.config.Paths.DownloadBase
	if downloadDir == "" {
		return
	}

	// 处理视频级别的封面（用于列表显示）
	if video.SinglePage && len(video.Pages) > 0 {
		// 单P视频：检查视频级别的封面文件
		localPosterPath := s.findLocalVideoPoster(downloadDir, video)
		if localPosterPath != "" {
			// 构建 URL 路径，确保正确处理特殊字符
			// localPosterPath 格式: "视频名/视频名-poster.jpg"
			video.Cover = "/downloads/" + localPosterPath
		}
	}

	// 处理每个分P的封面
	for i := range video.Pages {
		page := &video.Pages[i]
		localPosterPath := s.findLocalPoster(downloadDir, video, page)
		if localPosterPath != "" {
			// 构建 URL 路径
			page.Image = "/downloads/" + localPosterPath
		}
		// 如果本地封面不存在，保持使用远程URL
	}
}

// findLocalVideoPoster 查找视频级别的本地封面文件（用于单P视频）
func (s *Server) findLocalVideoPoster(downloadDir string, video *models.Video) string {
	// 可能的图片扩展名
	extensions := []string{".jpg", ".jpeg", ".png", ".webp", ".gif"}

	videoName := utils.Filenamify(video.Name)
	// 视频文件夹路径
	videoFolder := filepath.Join(downloadDir, videoName)

	// 单P视频封面格式：{video_name}-poster.ext
	for _, ext := range extensions {
		posterFile := videoName + "-poster" + ext

		// 检查文件是否存在（在视频文件夹内）
		fullPath := filepath.Join(videoFolder, posterFile)
		if fileExists(fullPath) {
			// 返回相对于下载目录的 URL 路径（使用正斜杠）
			return videoName + "/" + posterFile
		}
	}

	return ""
}

// findLocalPoster 查找本地封面文件
func (s *Server) findLocalPoster(downloadDir string, video *models.Video, page *models.Page) string {
	// 可能的图片扩展名
	extensions := []string{".jpg", ".jpeg", ".png", ".webp", ".gif"}

	videoName := utils.Filenamify(video.Name)
	pageName := utils.Filenamify(page.Name)
	// 视频文件夹路径
	videoFolder := filepath.Join(downloadDir, videoName)

	var posterFile string

	// 根据是否为单P视频构建不同的文件名
	for _, ext := range extensions {
		if video.SinglePage {
			// 单P视频：{video_name}-poster.ext
			posterFile = videoName + "-poster" + ext
		} else {
			// 多P视频：{video_name}-{ptitle}-poster.ext
			posterFile = videoName + "-" + pageName + "-poster" + ext
		}

		// 检查文件是否存在（在视频文件夹内）
		fullPath := filepath.Join(videoFolder, posterFile)
		if fileExists(fullPath) {
			// 返回相对于下载目录的 URL 路径（使用正斜杠）
			return videoName + "/" + posterFile
		}
	}

	return ""
}

// fileExists 检查文件是否存在
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// deleteLocalFiles 删除视频相关的本地文件
func (s *Server) deleteLocalFiles(video *models.Video) error {
	downloadDir := s.config.Paths.DownloadBase
	if downloadDir == "" {
		return nil
	}

	videoName := utils.Filenamify(video.Name)
	// 视频文件夹路径
	videoFolder := filepath.Join(downloadDir, videoName)

	// 检查文件夹是否存在
	info, err := os.Stat(videoFolder)
	if os.IsNotExist(err) {
		utils.Info("视频文件夹不存在: %s", videoFolder)
		return nil
	}
	if err != nil {
		return fmt.Errorf("检查视频文件夹失败: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("路径不是文件夹: %s", videoFolder)
	}

	// 直接删除整个视频文件夹
	if err := os.RemoveAll(videoFolder); err != nil {
		return fmt.Errorf("删除视频文件夹失败: %w", err)
	}

	utils.Info("已删除视频文件夹: %s", videoFolder)
	return nil
}
