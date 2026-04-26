package downloader

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

	notifyStatus := func(taskName string, status DownloadStatus, progress float64, downloaded, total int64) {
		dm.tracker.NotifyProgress(video.ID, 0, taskName, &SubTaskProgress{
			Name:           taskName,
			Status:         status,
			Progress:       progress,
			DownloadedSize: downloaded,
			TotalSize:      total,
		})
	}

	notifyStatus("video", StatusDownloading, 0, 0, 0)

	client := xhs.NewClient(dm.config, task.OutputDir)
	// 直接下载到 task.OutputDir，避免再嵌套作者/笔记目录
	dl := client.Downloader()

	parser := client.Parser()
	note, err := parser.Parse(task.Context, task.URL)
	if err != nil {
		dm.failXHSTask(task, fmt.Errorf("解析笔记失败: %w", err), notifyStatus)
		return
	}
	if len(note.MediaItems) == 0 {
		dm.failXHSTask(task, fmt.Errorf("笔记未发现可下载媒体"), notifyStatus)
		return
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

	totalFiles := len(note.MediaItems)
	completed := 0
	progressCb := func(filename string, downloaded, total int64) {
		var pct float64
		if totalFiles > 0 {
			pct = float64(completed) / float64(totalFiles) * 100
			if total > 0 {
				pct += float64(downloaded) / float64(total) / float64(totalFiles) * 100
			}
		}
		notifyStatus("video", StatusDownloading, pct, downloaded, total)
	}

	result, err := dl.DownloadNote(task.Context, note, task.OutputDir, progressCb)
	if err != nil {
		dm.failXHSTask(task, err, notifyStatus)
		return
	}

	if result.SuccessNum == 0 {
		dm.failXHSTask(task, fmt.Errorf("所有媒体下载失败"), notifyStatus)
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
				dm.db.Model(p).Updates(updates)
				if f.GroupIndex == 1 && firstCoverPath == "" {
					firstCoverPath = f.Path
				}
			}
		}
		videoUpdates := map[string]interface{}{
			"download_status": 1,
		}
		if firstCoverPath != "" {
			videoUpdates["cover"] = firstCoverPath
		}
		dm.db.Model(video).Updates(videoUpdates)
	}

	var totalSize int64
	for _, f := range result.Files {
		totalSize += f.Size
	}
	notifyStatus("video", StatusSucceeded, 100, totalSize, totalSize)

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

func (dm *DownloadManager) failXHSTask(task *DownloadTask, err error, notify func(string, DownloadStatus, float64, int64, int64)) {
	utils.Error("小红书下载失败: %v", err)
	task.SetError(err)
	task.SetStatus(TaskStatusFailed)
	notify("video", StatusFailed, 0, 0, 0)

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

func (dm *DownloadManager) buildXHSFileDetails() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name":   "video",
			"status": "pending",
		},
	}
}
