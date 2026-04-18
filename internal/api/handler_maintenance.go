package api

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"bili-download/internal/database/models"
	"bili-download/internal/downloader"
	"bili-download/internal/nfo"
	"bili-download/internal/utils"

	"github.com/gin-gonic/gin"
)

type backfillQualityQueryParts struct {
	pageTable    string
	videoTable   string
	selectClause string
	joinClause   string
	whereClause  string
}

type pageWithVideo struct {
	models.Page
	VideoName  string
	SinglePage bool
	VideoPath  string
}

func buildBackfillQualityQueryParts() backfillQualityQueryParts {
	pageTable := (models.Page{}).TableName()
	videoTable := (models.Video{}).TableName()

	return backfillQualityQueryParts{
		pageTable:    pageTable,
		videoTable:   videoTable,
		selectClause: fmt.Sprintf("%s.*, %s.name as video_name, %s.single_page as single_page, %s.path as video_path", pageTable, videoTable, videoTable, videoTable),
		joinClause:   fmt.Sprintf("JOIN %s ON %s.id = %s.video_id", videoTable, videoTable, pageTable),
		whereClause:  fmt.Sprintf("%s.download_status = ? AND (%s.quality = 0 OR %s.width = 0)", pageTable, pageTable, pageTable),
	}
}

func (s *Server) queryPagesForMetadataBackfill(whereClause string, args ...interface{}) ([]pageWithVideo, error) {
	queryParts := buildBackfillQualityQueryParts()

	var rows []pageWithVideo
	err := s.db.Table(queryParts.pageTable).
		Select(queryParts.selectClause).
		Joins(queryParts.joinClause).
		Where(whereClause, args...).
		Scan(&rows).Error

	return rows, err
}

func (s *Server) backfillPageMetadata(rows []pageWithVideo, taskName string) {
	videoExts := []string{".mp4", ".mkv", ".webm", ".flv", ".avi", ".m4v"}
	delayRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	updated, skipped, failed := 0, 0, 0

	for _, r := range rows {
		if r.VideoPath == "" {
			skipped++
			continue
		}
		outputDir := r.VideoPath
		if !filepath.IsAbs(outputDir) {
			outputDir = filepath.Join(s.config.Paths.DownloadBase, outputDir)
		}

		var baseName string
		if r.SinglePage {
			baseName = utils.Filenamify(r.VideoName)
		} else {
			baseName = fmt.Sprintf("%s-%s", utils.Filenamify(r.VideoName), utils.Filenamify(r.Page.Name))
		}

		filePath := ""
		entries, _ := os.ReadDir(outputDir)
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if !strings.HasPrefix(name, baseName) {
				continue
			}
			lower := strings.ToLower(name)
			for _, ext := range videoExts {
				if strings.HasSuffix(lower, ext) {
					filePath = filepath.Join(outputDir, name)
					break
				}
			}
			if filePath != "" {
				break
			}
		}
		if filePath == "" {
			utils.Warn("%s: 未找到文件 %s/%s", taskName, outputDir, baseName)
			skipped++
			continue
		}

		probe, err := downloader.ProbeVideo(context.Background(), filePath)
		time.Sleep(500*time.Millisecond + time.Duration(delayRand.Int63n(int64(500*time.Millisecond)+1)))
		if err != nil {
			utils.Warn("%s: ffprobe 失败 %s: %v", taskName, filePath, err)
			failed++
			continue
		}

		updates := map[string]interface{}{
			"width":       probe.Width,
			"height":      probe.Height,
			"frame_rate":  probe.FrameRate,
			"quality":     models.CalcQuality(probe.Height, probe.FrameRate),
			"orientation": models.CalcOrientation(probe.Width, probe.Height),
		}
		if err := s.db.Model(&models.Page{}).Where("id = ?", r.Page.ID).Updates(updates).Error; err != nil {
			utils.Warn("%s: 更新失败 page=%d: %v", taskName, r.Page.ID, err)
			failed++
			continue
		}
		updated++
	}

	utils.Info("%s完成: 共%d个，成功 %d 个，跳过 %d 个，失败 %d 个", taskName, len(rows), updated, skipped, failed)
}

// refreshViewCountRunning 防止重复执行
var refreshViewCountRunning atomic.Bool

// handleRefreshViewCounts 刷新已下载视频的播放量（异步）
func (s *Server) handleRefreshViewCounts(c *gin.Context) {
	if !refreshViewCountRunning.CompareAndSwap(false, true) {
		respondError(c, 409, "刷新播放量任务正在执行中，请稍后再试")
		return
	}

	// 查询数量用于前端展示
	var count int64
	s.db.Model(&models.Video{}).Where("valid = ? AND download_status != 0", true).Count(&count)

	// 异步执行
	go s.doRefreshViewCounts()

	respondSuccess(c, gin.H{
		"total":   count,
		"message": fmt.Sprintf("已开始刷新 %d 个视频的播放量，请查看日志了解进度", count),
	})
}

func (s *Server) doRefreshViewCounts() {
	defer refreshViewCountRunning.Store(false)

	var videos []models.Video
	if err := s.db.Preload("Pages").Where("valid = ? AND download_status != 0", true).Find(&videos).Error; err != nil {
		utils.Error("刷新播放量查询失败: %v", err)
		return
	}

	updated := 0
	failed := 0

	for i := range videos {
		video := &videos[i]

		detail, err := s.biliClient.GetVideoDetail(video.BVid)
		if err != nil {
			utils.Warn("获取视频 %s 播放量失败: %v", video.BVid, err)
			failed++
			continue
		}

		newViewCount := detail.Stat.View

		if err := s.db.Model(video).Update("view_count", newViewCount).Error; err != nil {
			utils.Warn("更新视频 %s 播放量失败: %v", video.BVid, err)
			failed++
			continue
		}
		video.ViewCount = newViewCount

		if video.Path != "" {
			videoPath := video.Path
			if !filepath.IsAbs(videoPath) {
				videoPath = filepath.Join(s.config.Paths.DownloadBase, videoPath)
			}
			s.updateNFOViewCount(video, videoPath)
		}

		updated++
		time.Sleep(200 * time.Millisecond)
	}

	utils.Info("刷新播放量完成: 共 %d 个，成功 %d 个，失败 %d 个", len(videos), updated, failed)
}

// refreshUpperFaceRunning 防止重复执行
var refreshUpperFaceRunning atomic.Bool

// handleRefreshUpperFaces 刷新UP主投稿的头像
func (s *Server) handleRefreshUpperFaces(c *gin.Context) {
	if !refreshUpperFaceRunning.CompareAndSwap(false, true) {
		respondError(c, 409, "刷新头像任务正在执行中，请稍后再试")
		return
	}

	var count int64
	s.db.Model(&models.Submission{}).Count(&count)

	go s.doRefreshUpperFaces()

	respondSuccess(c, gin.H{
		"total":   count,
		"message": fmt.Sprintf("已开始刷新 %d 个UP主的头像", count),
	})
}

func (s *Server) doRefreshUpperFaces() {
	defer refreshUpperFaceRunning.Store(false)

	var submissions []models.Submission
	if err := s.db.Find(&submissions).Error; err != nil {
		utils.Error("刷新UP主头像查询失败: %v", err)
		return
	}

	updated := 0
	failed := 0

	for _, sub := range submissions {
		info, err := s.biliClient.GetUpperInfo(sub.UpperID)
		if err != nil {
			utils.Warn("获取UP主 %s (ID:%d) 信息失败: %v", sub.Name, sub.UpperID, err)
			failed++
			continue
		}

		if info.Face != "" && info.Face != sub.UpperFace {
			if err := s.db.Model(&sub).Update("upper_face", info.Face).Error; err != nil {
				utils.Warn("更新UP主 %s 头像失败: %v", sub.Name, err)
				failed++
				continue
			}
			updated++
		}

		time.Sleep(200 * time.Millisecond)
	}

	utils.Info("刷新UP主头像完成: 共 %d 个，更新 %d 个，失败 %d 个", len(submissions), updated, failed)
}

// backfillQualityRunning 防止重复执行
var backfillQualityRunning atomic.Bool
var reparsePageMetadataRunning atomic.Bool

// handleBackfillQuality 扫描已下载文件回填 width/height/frame_rate/quality/orientation
func (s *Server) handleBackfillQuality(c *gin.Context) {
	if !backfillQualityRunning.CompareAndSwap(false, true) {
		respondError(c, 409, "回填画质任务正在执行中，请稍后再试")
		return
	}

	var count int64
	s.db.Model(&models.Page{}).Where("download_status = ? AND (quality = 0 OR width = 0)", 1).Count(&count)

	go s.doBackfillQuality()

	respondSuccess(c, gin.H{
		"total":   count,
		"message": fmt.Sprintf("已开始回填 %d 个分P的画质信息，请查看日志了解进度", count),
	})
}

func (s *Server) doBackfillQuality() {
	defer backfillQualityRunning.Store(false)

	queryParts := buildBackfillQualityQueryParts()

	type pageWithVideo struct {
		models.Page
		VideoName  string
		SinglePage bool
		VideoPath  string
	}
	var rows []pageWithVideo
	err := s.db.Table(queryParts.pageTable).
		Select(queryParts.selectClause).
		Joins(queryParts.joinClause).
		Where(queryParts.whereClause, 1).
		Scan(&rows).Error
	if err != nil {
		utils.Error("回填画质查询失败: %v", err)
		return
	}

	videoExts := []string{".mp4", ".mkv", ".webm", ".flv", ".avi", ".m4v"}
	updated, skipped, failed := 0, 0, 0

	for _, r := range rows {
		if r.VideoPath == "" {
			skipped++
			continue
		}
		outputDir := r.VideoPath
		if !filepath.IsAbs(outputDir) {
			outputDir = filepath.Join(s.config.Paths.DownloadBase, outputDir)
		}

		var baseName string
		if r.SinglePage {
			baseName = utils.Filenamify(r.VideoName)
		} else {
			baseName = fmt.Sprintf("%s-%s", utils.Filenamify(r.VideoName), utils.Filenamify(r.Page.Name))
		}

		filePath := ""
		entries, _ := os.ReadDir(outputDir)
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if !strings.HasPrefix(name, baseName) {
				continue
			}
			lower := strings.ToLower(name)
			for _, ext := range videoExts {
				if strings.HasSuffix(lower, ext) {
					filePath = filepath.Join(outputDir, name)
					break
				}
			}
			if filePath != "" {
				break
			}
		}
		if filePath == "" {
			utils.Warn("回填画质: 未找到文件 %s/%s", outputDir, baseName)
			skipped++
			continue
		}

		probe, err := downloader.ProbeVideo(context.Background(), filePath)
		if err != nil {
			utils.Warn("回填画质: ffprobe 失败 %s: %v", filePath, err)
			failed++
			continue
		}

		updates := map[string]interface{}{
			"width":       probe.Width,
			"height":      probe.Height,
			"frame_rate":  probe.FrameRate,
			"quality":     models.CalcQuality(probe.Height, probe.FrameRate),
			"orientation": models.CalcOrientation(probe.Width, probe.Height),
		}
		if err := s.db.Model(&models.Page{}).Where("id = ?", r.Page.ID).Updates(updates).Error; err != nil {
			utils.Warn("回填画质: 更新失败 page=%d: %v", r.Page.ID, err)
			failed++
			continue
		}
		updated++
	}

	utils.Info("回填画质完成: 共 %d 个，成功 %d 个，跳过 %d 个，失败 %d 个", len(rows), updated, skipped, failed)
}

// handleReparsePageMetadata 重新解析画质/帧率/方向为空的已下载分P
func (s *Server) handleReparsePageMetadata(c *gin.Context) {
	if !reparsePageMetadataRunning.CompareAndSwap(false, true) {
		respondError(c, 409, "重新解析视频信息任务正在执行中，请稍后再试")
		return
	}

	var count int64
	s.db.Model(&models.Page{}).
		Where("download_status = ? AND (quality = 0 OR quality IS NULL) AND (frame_rate = 0 OR frame_rate IS NULL) AND (orientation = 0 OR orientation IS NULL)", 1).
		Count(&count)

	go s.doReparsePageMetadata()

	respondSuccess(c, gin.H{
		"total":   count,
		"message": fmt.Sprintf("已开始重新解析 %d 个分P的视频信息，请查看日志了解进度", count),
	})
}

func (s *Server) doReparsePageMetadata() {
	defer reparsePageMetadataRunning.Store(false)

	queryParts := buildBackfillQualityQueryParts()
	whereClause := fmt.Sprintf("%s.download_status = ? AND (%s.quality = 0 OR %s.quality IS NULL) AND (%s.frame_rate = 0 OR %s.frame_rate IS NULL) AND (%s.orientation = 0 OR %s.orientation IS NULL)",
		queryParts.pageTable,
		queryParts.pageTable, queryParts.pageTable,
		queryParts.pageTable, queryParts.pageTable,
		queryParts.pageTable, queryParts.pageTable,
	)

	rows, err := s.queryPagesForMetadataBackfill(whereClause, 1)
	if err != nil {
		utils.Error("重新解析视频信息查询失败: %v", err)
		return
	}

	s.backfillPageMetadata(rows, "重新解析视频信息")
}

func (s *Server) updateNFOViewCount(video *models.Video, outputDir string) {
	for _, page := range video.Pages {
		var nfoFile string
		if video.SinglePage {
			nfoFile = fmt.Sprintf("%s.nfo", utils.Filenamify(video.Name))
		} else {
			nfoFile = fmt.Sprintf("%s-%s.nfo",
				utils.Filenamify(video.Name),
				utils.Filenamify(page.Name))
		}
		nfoPath := filepath.Join(outputDir, nfoFile)

		if _, err := os.Stat(nfoPath); os.IsNotExist(err) {
			continue
		}

		var dateAdded time.Time
		if s.config.Advanced.NFOTimeType == "pubtime" {
			dateAdded = video.PubTime
		} else {
			dateAdded = video.FavTime
		}

		if video.SinglePage {
			generator := nfo.NewMovieGenerator()
			generator.
				SetTitle(video.Name).
				SetOriginalTitle(video.Name).
				SetPlot(video.Intro).
				SetRuntime(page.Duration).
				SetPremiered(video.PubTime).
				SetDateAdded(dateAdded).
				SetStudio("bilibili").
				SetDirector(video.UpperName).
				SetPlayCount(video.ViewCount).
				AddActor(video.UpperName, "UP主", video.UpperFace).
				AddUniqueID("bvid", video.BVid, true).
				AddTags(video.Tags)

			if page.Width > 0 && page.Height > 0 {
				generator.SetVideoInfo("h264", page.Width, page.Height, page.Duration)
			}
			generator.SetAudioInfo("aac", "zh", 2)
			if video.Cover != "" {
				generator.AddThumb(video.Cover, "poster")
			}

			if err := generator.WriteToFile(nfoPath); err != nil {
				utils.Warn("更新NFO文件失败 %s: %v", nfoPath, err)
			}
		} else {
			generator := nfo.NewEpisodeGenerator()
			generator.
				SetTitle(page.Name).
				SetShowTitle(video.Name).
				SetPlot(video.Intro).
				SetRuntime(page.Duration).
				SetSeasonEpisode(1, page.PID).
				SetAired(video.PubTime).
				SetDateAdded(dateAdded).
				SetStudio("bilibili").
				SetDirector(video.UpperName).
				SetPlayCount(video.ViewCount).
				AddActor(video.UpperName, "UP主", video.UpperFace).
				AddUniqueID("bvid", video.BVid, true).
				AddTags(video.Tags)

			if page.Width > 0 && page.Height > 0 {
				generator.SetVideoInfo("h264", page.Width, page.Height, page.Duration)
			}
			generator.SetAudioInfo("aac", "zh", 2)
			if page.Image != "" {
				generator.AddThumb(page.Image, "poster")
			} else if video.Cover != "" {
				generator.AddThumb(video.Cover, "poster")
			}

			if err := generator.WriteToFile(nfoPath); err != nil {
				utils.Warn("更新NFO文件失败 %s: %v", nfoPath, err)
			}
		}
	}
}
