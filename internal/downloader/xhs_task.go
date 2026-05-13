package downloader

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"bili-download/internal/database/models"
	"bili-download/internal/utils"
	"bili-download/internal/xhs"
)

// PrepareAndAddXHSTask 创建小红书笔记下载任务
//
// video: 关联的视频记录（BVid 形如 XHS_xxx，作为标识）
// noteURL: 小红书笔记原始链接
// baseDir: 输出根目录
func (dm *DownloadManager) PrepareAndAddXHSTask(video *models.Video, noteURL string, baseDir string) (*DownloadTask, error) {
	folderName := utils.Filenamify(video.Name)
	if folderName == "" {
		folderName = video.BVid
	}
	outputDir := filepath.Join(baseDir, folderName)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("创建笔记目录失败: %w", err)
	}

	if dm.db != nil {
		video.Path = outputDir
		dm.db.Model(video).Update("path", outputDir)
	}

	task := NewDownloadTask(TaskTypeXHS, video, nil, outputDir)
	task.URL = noteURL
	task.MaxRetries = dm.getMaxRetries()

	if dm.db != nil {
		fileDetails := dm.buildXHSFileDetails()
		detailsJSON, _ := json.Marshal(fileDetails)

		record := &models.DownloadRecord{
			VideoID:     video.ID,
			SourceType:  "xhs",
			SourceURL:   noteURL,
			SourceName:  "小红书",
			Status:      "pending",
			FileDetails: detailsJSON,
		}
		if err := dm.db.Create(record).Error; err != nil {
			utils.Warn("创建小红书下载记录失败: %v", err)
		} else {
			task.RecordID = record.ID
			record.Video = *video
			dm.emitEvent(ManagerEvent{
				Type:      EventRecordCreated,
				Task:      task,
				Record:    record,
				Timestamp: time.Now(),
			})
		}
	}

	if err := dm.AddTask(task); err != nil {
		return nil, err
	}

	utils.Info("已创建小红书下载任务: [%s], URL: %s", video.Name, noteURL)
	return task, nil
}

// executeXHSTask 执行小红书笔记下载任务
func (dm *DownloadManager) executeXHSTask(task *DownloadTask) {
	defer dm.wg.Done()

	if err := dm.concurrency.AcquireVideo(task.Context); err != nil {
		task.SetError(err)
		task.SetStatus(TaskStatusCancelled)
		return
	}
	defer dm.concurrency.ReleaseVideo()

	dm.runningTasks.Store(task.ID, task)
	defer func() {
		dm.runningTasks.Delete(task.ID)
		dm.completedTasks.Store(task.ID, task)
	}()

	task.SetStatus(TaskStatusRunning)
	dm.emitEvent(ManagerEvent{
		Type:      EventTaskStarted,
		Task:      task,
		Timestamp: time.Now(),
	})

	if dm.db != nil && task.RecordID > 0 {
		now := time.Now()
		dm.db.Model(&models.DownloadRecord{}).Where("id = ?", task.RecordID).
			Updates(map[string]interface{}{"status": "downloading", "started_at": now})
	}

	video := task.Video

	notifyLabeled := func(taskName, label string, status DownloadStatus, progress float64, downloaded, total int64) {
		dm.tracker.NotifyProgress(video.ID, 0, taskName, &SubTaskProgress{
			Name:           taskName,
			Label:          label,
			Status:         status,
			Progress:       progress,
			DownloadedSize: downloaded,
			TotalSize:      total,
		})
	}

	notifyLabeled("video", "", StatusDownloading, 0, 0, 0)

	client := xhs.NewClient(dm.config, task.OutputDir)
	// 直接下载到 task.OutputDir，避免再嵌套作者/笔记目录
	dl := client.Downloader()

	parser := client.Parser()
	note, err := parser.Parse(task.Context, task.URL)
	if err != nil {
		dm.failXHSTask(task, fmt.Errorf("解析笔记失败: %w", err), notifyLabeled)
		return
	}
	if len(note.MediaItems) == 0 {
		dm.failXHSTask(task, fmt.Errorf("笔记未发现可下载媒体"), notifyLabeled)
		return
	}

	imageCount := 0
	videoCount := 0
	for _, m := range note.MediaItems {
		switch m.Type {
		case xhs.MediaTypeImage, xhs.MediaTypeLivePhoto:
			imageCount++
		case xhs.MediaTypeVideo:
			videoCount++
		}
	}
	isVideoNote := note.Type == xhs.NoteTypeVideo

	// 解析后立即用真实媒体类型重建 file_details
	if dm.db != nil && task.RecordID > 0 {
		var initFiles []models.FileDetail
		if isVideoNote {
			initFiles = append(initFiles, models.FileDetail{Name: "video", Label: "视频", Status: "downloading"})
		} else {
			label := "图片"
			if imageCount > 0 {
				label = fmt.Sprintf("图片 (0/%d)", imageCount)
			}
			initFiles = append(initFiles, models.FileDetail{Name: "images", Label: label, Status: "downloading"})
			if videoCount > 0 {
				initFiles = append(initFiles, models.FileDetail{Name: "video", Label: "视频", Status: "downloading"})
			}
		}
		detailsJSON, _ := json.Marshal(models.FileDetailsData{Files: initFiles})
		dm.db.Model(&models.DownloadRecord{}).Where("id = ?", task.RecordID).
			Update("file_details", detailsJSON)
	}

	// 同步视频元信息到数据库
	if dm.db != nil && video != nil {
		updates := map[string]interface{}{}
		if video.Name == "" || video.Name == video.BVid {
			title := note.Title
			if title == "" {
				title = note.Description
			}
			if title != "" {
				updates["name"] = utils.TruncateString(title, 200)
			}
		}
		if video.Intro == "" && note.Description != "" {
			updates["intro"] = note.Description
		}
		if video.UpperName == "" && note.Author.Nickname != "" {
			updates["upper_name"] = note.Author.Nickname
		}
		if len(updates) > 0 {
			dm.db.Model(video).Updates(updates)
		}
	}

	var (
		videoMu        sync.Mutex
		videoTotals    = make(map[string]int64)
		videoDoneBytes = make(map[string]int64)

		imgMu       sync.Mutex
		imgFinished = make(map[string]bool)
	)

	// 提取图片"组键"以去重 LivePhoto 的图+视频两个文件（同一 group 算一张）
	imgGroupKey := func(filename string) string {
		if idx := strings.Index(filename, "_live_src."); idx >= 0 {
			return filename[:idx]
		}
		if dot := strings.LastIndex(filename, "."); dot >= 0 {
			return filename[:dot]
		}
		return filename
	}

	progressCb := func(filename string, downloaded, total int64) {
		lower := strings.ToLower(filename)
		isLiveSrc := strings.Contains(lower, "_live_src.")
		isMP4 := strings.HasSuffix(lower, ".mp4")

		if isVideoNote && isMP4 && !isLiveSrc {
			videoMu.Lock()
			videoTotals[filename] = total
			videoDoneBytes[filename] = downloaded
			var totalAll, downAll int64
			for _, t := range videoTotals {
				totalAll += t
			}
			for _, d := range videoDoneBytes {
				downAll += d
			}
			videoMu.Unlock()
			var pct float64
			if totalAll > 0 {
				pct = float64(downAll) / float64(totalAll) * 100
			}
			notifyLabeled("video", "视频", StatusDownloading, pct, downAll, totalAll)
			return
		}

		// 图集/Live：按文件完成数计
		if imageCount <= 0 || total <= 0 || downloaded < total {
			return
		}
		key := imgGroupKey(filename)
		imgMu.Lock()
		isNew := !imgFinished[key]
		if isNew {
			imgFinished[key] = true
		}
		done := len(imgFinished)
		imgMu.Unlock()
		if !isNew {
			return
		}
		pct := float64(done) / float64(imageCount) * 100
		label := fmt.Sprintf("图片 (%d/%d)", done, imageCount)
		notifyLabeled("images", label, StatusDownloading, pct, int64(done), int64(imageCount))
	}

	result, err := dl.DownloadNote(task.Context, note, task.OutputDir, progressCb)
	if err != nil {
		dm.failXHSTask(task, err, notifyLabeled)
		return
	}

	if result.SuccessNum == 0 {
		dm.failXHSTask(task, fmt.Errorf("所有媒体下载失败"), notifyLabeled)
		return
	}

	// 按 GroupIndex 更新对应 Page 的 file_path / kind / download_status
	if dm.db != nil && video != nil {
		var pages []models.Page
		firstCoverPath := ""
		if err := dm.db.Where("video_id = ?", video.ID).Order("pid asc").Find(&pages).Error; err == nil {
			pageByPID := make(map[int]*models.Page, len(pages))
			for i := range pages {
				pageByPID[pages[i].PID] = &pages[i]
			}
			for _, f := range result.Files {
				if f.GroupIndex <= 0 {
					continue
				}
				p, ok := pageByPID[f.GroupIndex]
				if !ok {
					continue
				}
				updates := map[string]interface{}{
					"file_path":       f.Path,
					"path":            f.Path,
					"download_status": 1,
					"kind":            string(f.MediaType),
				}
				if f.MediaType == xhs.MediaTypeVideo {
					if probe, err := ProbeVideo(task.Context, f.Path); err != nil {
						utils.Warn("小红书视频 ffprobe 探测失败: %s, %v", f.Path, err)
					} else {
						updates["width"] = probe.Width
						updates["height"] = probe.Height
						updates["frame_rate"] = probe.FrameRate
						updates["quality"] = models.CalcQuality(probe.Width, probe.Height, probe.FrameRate)
						updates["orientation"] = models.CalcOrientation(probe.Width, probe.Height)
					}
				}
				dm.db.Model(p).Updates(updates)
				// cover 优先选图片类型（避免视频笔记把 mp4 当封面）
				if firstCoverPath == "" && f.MediaType == xhs.MediaTypeImage {
					firstCoverPath = f.Path
				}
			}
			// 兜底：没有图片则取第一个文件
			if firstCoverPath == "" && len(result.Files) > 0 {
				firstCoverPath = result.Files[0].Path
			}
		}
		videoUpdates := map[string]interface{}{
			"download_status": 1,
		}
		if coverURL := dm.localPathToDownloadURL(firstCoverPath); coverURL != "" {
			videoUpdates["cover"] = coverURL
		}
		dm.db.Model(video).Updates(videoUpdates)
	}

	var totalSize int64
	for _, f := range result.Files {
		totalSize += f.Size
	}
	if isVideoNote {
		notifyLabeled("video", "视频", StatusSucceeded, 100, totalSize, totalSize)
	} else {
		notifyLabeled("images", fmt.Sprintf("图片 (%d/%d)", imageCount, imageCount), StatusSucceeded, 100, int64(imageCount), int64(imageCount))
		if videoCount > 0 {
			notifyLabeled("video", "视频", StatusSucceeded, 100, totalSize, totalSize)
		}
	}

	task.SetStatus(TaskStatusCompleted)
	dm.emitEvent(ManagerEvent{
		Type:      EventTaskCompleted,
		Task:      task,
		Timestamp: time.Now(),
	})

	if dm.db != nil {
		now := time.Now()
		if task.RecordID > 0 {
			dm.db.Model(&models.DownloadRecord{}).Where("id = ?", task.RecordID).
				Updates(map[string]interface{}{
					"status":       "completed",
					"completed_at": now,
				})
		}
	}
	utils.Info("小红书笔记下载完成: [%s], 成功 %d, 失败 %d", video.Name, result.SuccessNum, result.FailedNum)
}

func (dm *DownloadManager) failXHSTask(task *DownloadTask, err error, notify func(string, string, DownloadStatus, float64, int64, int64)) {
	utils.Error("小红书下载失败: %v", err)
	task.SetError(err)
	task.SetStatus(TaskStatusFailed)
	notify("video", "", StatusFailed, 0, 0, 0)

	if dm.db != nil && task.RecordID > 0 {
		now := time.Now()
		dm.db.Model(&models.DownloadRecord{}).Where("id = ?", task.RecordID).
			Updates(map[string]interface{}{
				"status":        "failed",
				"error_message": err.Error(),
				"completed_at":  now,
			})
	}
	dm.handleTaskFailure(task)
}

func (dm *DownloadManager) buildXHSFileDetails() models.FileDetailsData {
	return models.FileDetailsData{
		Files: []models.FileDetail{
			{Name: "video", Label: "视频", Status: "pending"},
		},
	}
}

// localPathToDownloadURL 将下载文件的绝对路径转换为可由前端访问的 /downloads/... URL。
// 文件不在 DownloadBase 下时返回空串。
func (dm *DownloadManager) localPathToDownloadURL(absPath string) string {
	if absPath == "" || dm.config == nil {
		return ""
	}
	base := dm.config.Paths.DownloadBase
	if base == "" {
		return ""
	}
	absBase, err := filepath.Abs(base)
	if err != nil {
		return ""
	}
	absFile, err := filepath.Abs(absPath)
	if err != nil {
		return ""
	}
	rel, err := filepath.Rel(absBase, absFile)
	if err != nil {
		return ""
	}
	rel = filepath.ToSlash(rel)
	if strings.HasPrefix(rel, "../") || rel == ".." {
		return ""
	}
	return "/downloads/" + rel
}
