package api

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"bili-download/internal/database/models"
	"bili-download/internal/nfo"
	"bili-download/internal/utils"

	"github.com/gin-gonic/gin"
)

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

// updateNFOViewCount 更新视频的NFO文件中的播放量
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
