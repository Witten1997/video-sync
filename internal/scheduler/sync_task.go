package scheduler

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"bili-download/internal/adapter"
	"bili-download/internal/bilibili"
	"bili-download/internal/config"
	"bili-download/internal/database/models"
	"bili-download/internal/downloader"
	"bili-download/internal/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SyncTask 同步任务
type SyncTask struct {
	ID          string
	TriggerType string // auto / manual
	Status      string // running / completed / failed / cancelled
	StartAt     time.Time
	EndAt       *time.Time

	// 统计信息
	SourcesTotal   int
	SourcesScanned int
	SourcesFailed  int
	VideosFound    int
	VideosNew      int
	VideosFiltered int
	VideosQueued   int
	TasksCreated   int

	// 详细信息
	SourceScans []*models.VideoSourceScan
	Errors      []TaskError

	// 依赖
	ctx             context.Context
	db              *gorm.DB
	config          *config.Config
	downloadManager *downloader.DownloadManager
	biliClient      *bilibili.Client
}

// TaskError 任务错误
type TaskError struct {
	Time    time.Time `json:"time"`
	Source  string    `json:"source"`
	Message string    `json:"message"`
	Type    string    `json:"type"` // scan_error / download_error / system_error
}

// VideoSourceInfo 视频源信息
type VideoSourceInfo struct {
	ID         string
	Type       string // favorite / submission / collection / watch_later
	Name       string
	Path       string
	Priority   int
	Rule       string
	LastScanAt *time.Time
	Adapter    adapter.VideoSource
}

// NewSyncTask 创建同步任务
func NewSyncTask(ctx context.Context, triggerType string, db *gorm.DB, cfg *config.Config, dm *downloader.DownloadManager) *SyncTask {
	return &SyncTask{
		ID:              fmt.Sprintf("sync-%s-%s", time.Now().Format("20060102-150405"), uuid.New().String()[:8]),
		TriggerType:     triggerType,
		Status:          "running",
		StartAt:         time.Now(),
		SourceScans:     make([]*models.VideoSourceScan, 0),
		Errors:          make([]TaskError, 0),
		ctx:             ctx,
		db:              db,
		config:          cfg,
		downloadManager: dm,
		biliClient:      bilibili.NewClient(cfg),
	}
}

// Execute 执行同步任务
func (st *SyncTask) Execute() error {
	utils.Info("[%s] 开始执行同步任务", st.ID)

	// 1. 加载所有启用的视频源
	sources, err := st.loadVideoSources()
	if err != nil {
		return fmt.Errorf("加载视频源失败: %w", err)
	}

	st.SourcesTotal = len(sources)
	utils.Info("[%s] 找到 %d 个已启用的视频源", st.ID, st.SourcesTotal)

	// 2. 逐个扫描视频源
	for _, source := range sources {
		select {
		case <-st.ctx.Done():
			utils.Warn("[%s] 同步任务被取消", st.ID)
			st.Status = "cancelled"
			return st.saveToDatabase()

		default:
			// 扫描视频源
			scanResult, err := st.scanVideoSource(source)
			if err != nil {
				utils.Error("[%s] 扫描视频源失败: %s - %v", st.ID, source.Name, err)
				st.SourcesFailed++
				st.addError(TaskError{
					Time:    time.Now(),
					Source:  source.Name,
					Message: err.Error(),
					Type:    "scan_error",
				})
				// 更新视频源健康状态
				st.updateSourceHealth(source.ID, source.Type, false, err.Error())
			} else {
				st.SourcesScanned++
				st.SourceScans = append(st.SourceScans, scanResult)
				// 更新视频源健康状态
				st.updateSourceHealth(source.ID, source.Type, true, "")
			}
		}
	}

	// 3. 完成同步
	endTime := time.Now()
	st.EndAt = &endTime
	st.Status = "completed"

	utils.Info("[%s] 同步任务完成 - 扫描: %d/%d, 发现: %d, 新增: %d, 加入队列: %d",
		st.ID, st.SourcesScanned, st.SourcesTotal, st.VideosFound, st.VideosNew, st.VideosQueued)

	// 4. 保存到数据库
	return st.saveToDatabase()
}

// loadVideoSources 加载所有启用的视频源
func (st *SyncTask) loadVideoSources() ([]VideoSourceInfo, error) {
	sources := make([]VideoSourceInfo, 0)

	// 1. 加载收藏夹
	var favorites []models.Favorite
	if err := st.db.Where("enabled = ?", true).Order("priority DESC, id ASC").Find(&favorites).Error; err != nil {
		return nil, fmt.Errorf("查询收藏夹失败: %w", err)
	}

	for _, fav := range favorites {
		favConfig := &adapter.FavoriteConfig{
			SourceConfig: adapter.SourceConfig{
				Type:    adapter.SourceTypeFavorite,
				ID:      fmt.Sprintf("fav_%d", fav.FID),
				Name:    fav.Name,
				Enabled: fav.Enabled,
			},
			MediaID: fmt.Sprintf("%d", fav.FID),
		}
		favAdapter := adapter.NewFavoriteAdapter(st.biliClient, favConfig)
		sources = append(sources, VideoSourceInfo{
			ID:         fmt.Sprintf("fav_%d", fav.FID),
			Type:       "favorite",
			Name:       fav.Name,
			Path:       fav.Path,
			Priority:   fav.Priority,
			Rule:       fav.Rule,
			LastScanAt: fav.LastScanAt,
			Adapter:    favAdapter,
		})
	}

	// 2. 加载UP主投稿
	var submissions []models.Submission
	if err := st.db.Where("enabled = ?", true).Order("priority DESC, id ASC").Find(&submissions).Error; err != nil {
		return nil, fmt.Errorf("查询UP主投稿失败: %w", err)
	}

	for _, sub := range submissions {
		subConfig := &adapter.SubmissionConfig{
			SourceConfig: adapter.SourceConfig{
				Type:    adapter.SourceTypeSubmission,
				ID:      fmt.Sprintf("sub_%d", sub.UpperID),
				Name:    sub.Name,
				Enabled: sub.Enabled,
			},
			Mid: fmt.Sprintf("%d", sub.UpperID),
		}
		subAdapter := adapter.NewSubmissionAdapter(st.biliClient, subConfig)
		sources = append(sources, VideoSourceInfo{
			ID:         fmt.Sprintf("sub_%d", sub.UpperID),
			Type:       "submission",
			Name:       sub.Name,
			Path:       sub.Path,
			Priority:   sub.Priority,
			Rule:       sub.Rule,
			LastScanAt: sub.LastScanAt,
			Adapter:    subAdapter,
		})
	}

	// 3. 加载合集
	var collections []models.Collection
	if err := st.db.Where("enabled = ?", true).Order("priority DESC, id ASC").Find(&collections).Error; err != nil {
		return nil, fmt.Errorf("查询合集失败: %w", err)
	}

	for _, col := range collections {
		colConfig := &adapter.CollectionConfig{
			SourceConfig: adapter.SourceConfig{
				Type:    adapter.SourceTypeCollection,
				ID:      fmt.Sprintf("col_%d", col.CID),
				Name:    col.Name,
				Enabled: col.Enabled,
			},
			Mid:            "", // 合集可能不需要 Mid，或者需要从其他地方获取
			SeasonID:       fmt.Sprintf("%d", col.CID),
			CollectionType: col.CType,
		}
		colAdapter := adapter.NewCollectionAdapter(st.biliClient, colConfig)
		sources = append(sources, VideoSourceInfo{
			ID:         fmt.Sprintf("col_%d", col.CID),
			Type:       "collection",
			Name:       col.Name,
			Path:       col.Path,
			Priority:   col.Priority,
			Rule:       col.Rule,
			LastScanAt: col.LastScanAt,
			Adapter:    colAdapter,
		})
	}

	// 4. 加载稍后再看
	var watchLaters []models.WatchLater
	if err := st.db.Where("enabled = ?", true).Find(&watchLaters).Error; err != nil {
		return nil, fmt.Errorf("查询稍后再看失败: %w", err)
	}

	for _, wl := range watchLaters {
		wlConfig := &adapter.WatchLaterConfig{
			SourceConfig: adapter.SourceConfig{
				Type:    adapter.SourceTypeWatchLater,
				ID:      fmt.Sprintf("wl_%d", wl.ID),
				Name:    wl.Name,
				Enabled: wl.Enabled,
			},
		}
		wlAdapter := adapter.NewWatchLaterAdapter(st.biliClient, wlConfig)
		sources = append(sources, VideoSourceInfo{
			ID:         fmt.Sprintf("wl_%d", wl.ID),
			Type:       "watch_later",
			Name:       wl.Name,
			Path:       wl.Path,
			Priority:   wl.Priority,
			Rule:       wl.Rule,
			LastScanAt: wl.LastScanAt,
			Adapter:    wlAdapter,
		})
	}

	return sources, nil
}

// scanVideoSource 扫描单个视频源
func (st *SyncTask) scanVideoSource(source VideoSourceInfo) (*models.VideoSourceScan, error) {
	startTime := time.Now()
	utils.Info("[%s] 开始扫描视频源: %s (%s)", st.ID, source.Name, source.Type)

	scanResult := &models.VideoSourceScan{
		SourceID:   source.ID,
		SourceType: source.Type,
		SourceName: source.Name,
		ScannedAt:  startTime,
		Success:    true,
	}

	// 使用适配器扫描视频源
	scanOpts := &adapter.ScanOptions{
		Limit: 0, // 不限制扫描数量
	}

	// 如果有上次扫描时间，使用增量扫描
	if source.LastScanAt != nil {
		scanOpts.OnlyNew = true
		scanOpts.LastScanTime = *source.LastScanAt
	}

	videos, err := source.Adapter.Scan(st.ctx, scanOpts)
	if err != nil {
		scanResult.Success = false
		scanResult.ErrorMessage = err.Error()
		return scanResult, err
	}

	scanResult.VideosFound = len(videos)
	st.VideosFound += len(videos)

	utils.Info("[%s] 视频源 %s 发现 %d 个视频", st.ID, source.Name, len(videos))

	// 处理视频
	newCount, queuedCount, err := st.processVideos(videos, source)
	if err != nil {
		scanResult.Success = false
		scanResult.ErrorMessage = err.Error()
		return scanResult, err
	}

	scanResult.VideosNew = newCount
	scanResult.VideosQueued = queuedCount
	scanResult.DurationMs = int(time.Since(startTime).Milliseconds())

	st.VideosNew += newCount
	st.VideosQueued += queuedCount

	utils.Info("[%s] 视频源 %s 扫描完成 - 新增: %d, 加入队列: %d, 耗时: %dms",
		st.ID, source.Name, newCount, queuedCount, scanResult.DurationMs)

	return scanResult, nil
}

// processVideos 处理视频列表
func (st *SyncTask) processVideos(videos []adapter.VideoInfo, source VideoSourceInfo) (newCount, queuedCount int, err error) {
	// 获取视频源的数据库ID
	sourceDBID := st.getSourceDBID(source)
	if sourceDBID == 0 {
		return 0, 0, fmt.Errorf("无法获取视频源数据库ID: %s", source.ID)
	}

	for _, video := range videos {
		// 检查视频是否已存在于当前视频源
		var existingVideo models.Video
		query := st.db.Where("bvid = ?", video.BVid)

		// 根据视频源类型添加关联条件
		switch source.Type {
		case "favorite":
			query = query.Where("favorite_id = ?", sourceDBID)
		case "submission":
			query = query.Where("submission_id = ?", sourceDBID)
		case "collection":
			query = query.Where("collection_id = ?", sourceDBID)
		case "watch_later":
			query = query.Where("watch_later_id = ?", sourceDBID)
		}

		result := query.First(&existingVideo)

		if result.Error == gorm.ErrRecordNotFound {
			// 当前视频源中不存在此视频
			utils.Info("[%s] 发现新视频: %s (BV%s)", st.ID, video.Title, video.BVid)

			// 判断是否应该下载
			if !st.shouldDownloadVideo(&video) {
				utils.Debug("[%s] 视频被过滤: %s", st.ID, video.Title)
				st.VideosFiltered++
				continue
			}

			// 创建视频记录
			newVideo := st.createVideoModel(video, source)
			utils.Debug("[%s] 创建视频模型: %s, Pages: %d", st.ID, newVideo.Name, len(newVideo.Pages))

			// 使用 FullSaveAssociations 确保 Pages 也被保存
			if err := st.db.Session(&gorm.Session{FullSaveAssociations: true}).Create(&newVideo).Error; err != nil {
				utils.Error("[%s] 创建视频记录失败: %s - %v", st.ID, video.Title, err)
				continue
			}
			utils.Info("[%s] 视频记录创建成功: %s (ID: %d)", st.ID, newVideo.Name, newVideo.ID)
			newCount++

			// 重新从数据库加载视频和它的Pages，确保关联数据完整
			var videoWithPages models.Video
			if err := st.db.Preload("Pages").First(&videoWithPages, newVideo.ID).Error; err != nil {
				utils.Error("[%s] 加载视频Pages失败: %s - %v", st.ID, video.Title, err)
				continue
			}
			utils.Debug("[%s] 加载视频完整数据: %s, Pages: %d", st.ID, videoWithPages.Name, len(videoWithPages.Pages))

			// 创建下载任务
			// 构建完整的基础目录：下载基础路径 + 视频源相对路径
			baseDir := filepath.Join(st.config.Paths.DownloadBase, source.Path)
			utils.Debug("[%s] 下载基础目录: %s", st.ID, baseDir)

			if err := st.createDownloadTask(&videoWithPages, baseDir); err != nil {
				utils.Error("[%s] 创建下载任务失败: %s - %v", st.ID, video.Title, err)
				continue
			}

			queuedCount++
		} else if result.Error != nil {
			utils.Error("[%s] 查询视频失败: %s - %v", st.ID, video.BVid, result.Error)
		}
		// 视频已存在于当前视频源，跳过
	}

	return newCount, queuedCount, nil
}

// shouldDownloadVideo 判断视频是否应该下载
func (st *SyncTask) shouldDownloadVideo(video *adapter.VideoInfo) bool {
	// 检查是否在配置的扫描模式下
	if st.config.Sync.ScanOnly {
		return false
	}

	// 可以在这里集成过滤引擎
	// TODO: 集成 FilterEngine

	return true
}

// createVideoModel 创建视频模型
func (st *SyncTask) createVideoModel(video adapter.VideoInfo, source VideoSourceInfo) models.Video {
	utils.Info("[%s] createVideoModel: video.Pages from adapter: %d", st.ID, len(video.Pages))

	newVideo := models.Video{
		BVid:           video.BVid,
		Name:           video.Title,
		Intro:          video.Description,
		Cover:          video.Cover,
		UpperID:        video.Owner.Mid,
		UpperName:      video.Owner.Name,
		PubTime:        video.PubDate,
		FavTime:        video.AddTime,
		ViewCount:      video.Stats.View,
		CTime:          time.Now(),
		SinglePage:     len(video.Pages) <= 1,
		Valid:          true, // 默认为有效
		ShouldDownload: true,
		DownloadStatus: 0,
		Path:           "", // 不在这里设置Path，由PrepareAndAddVideoTask统一设置
	}

	// 创建视频的所有分P
	pages := make([]models.Page, 0, len(video.Pages))
	for _, pageInfo := range video.Pages {
		page := models.Page{
			CID:            pageInfo.CID,
			PID:            pageInfo.Page,
			Name:           pageInfo.Part,
			Duration:       pageInfo.Duration,
			Width:          pageInfo.Width,
			Height:         pageInfo.Height,
			DownloadStatus: 0,
		}
		pages = append(pages, page)
	}
	newVideo.Pages = pages

	utils.Info("[%s] createVideoModel: created video with %d pages", st.ID, len(newVideo.Pages))

	// 设置视频源关联
	// 从 source.ID 中提取数据库 ID
	// source.ID 格式为 "fav_12345" 或 "sub_67890"
	switch source.Type {
	case "favorite":
		// 从数据库查询 FID 对应的 ID
		var fav models.Favorite
		if err := st.db.Where("f_id = ?", extractIDFromSourceID(source.ID)).First(&fav).Error; err == nil {
			newVideo.FavoriteID = &fav.ID
		}
	case "submission":
		// 从数据库查询 UpperID 对应的 ID
		var sub models.Submission
		if err := st.db.Where("upper_id = ?", extractIDFromSourceID(source.ID)).First(&sub).Error; err == nil {
			newVideo.SubmissionID = &sub.ID
		}
	case "collection":
		// 从数据库查询 CID 对应的 ID
		var col models.Collection
		if err := st.db.Where("c_id = ?", extractIDFromSourceID(source.ID)).First(&col).Error; err == nil {
			newVideo.CollectionID = &col.ID
		}
	case "watch_later":
		// 从数据库查询 WatchLater 的 ID
		var wl models.WatchLater
		if err := st.db.First(&wl).Error; err == nil {
			newVideo.WatchLaterID = &wl.ID
		}
	}

	return newVideo
}

// extractIDFromSourceID 从 source ID 中提取数字 ID
// 例如: "fav_12345" -> 12345
func extractIDFromSourceID(sourceID string) string {
	parts := strings.Split(sourceID, "_")
	if len(parts) > 1 {
		return parts[1]
	}
	return sourceID
}

// getSourceDBID 获取视频源在数据库中的主键ID
func (st *SyncTask) getSourceDBID(source VideoSourceInfo) uint {
	numericID := extractIDFromSourceID(source.ID)

	switch source.Type {
	case "favorite":
		var fav models.Favorite
		if err := st.db.Where("f_id = ?", numericID).First(&fav).Error; err == nil {
			return fav.ID
		}
	case "submission":
		var sub models.Submission
		if err := st.db.Where("upper_id = ?", numericID).First(&sub).Error; err == nil {
			return sub.ID
		}
	case "collection":
		var col models.Collection
		if err := st.db.Where("c_id = ?", numericID).First(&col).Error; err == nil {
			return col.ID
		}
	case "watch_later":
		var wl models.WatchLater
		if err := st.db.First(&wl).Error; err == nil {
			return wl.ID
		}
	}

	return 0
}

// createDownloadTask 创建下载任务
func (st *SyncTask) createDownloadTask(video *models.Video, baseDir string) error {
	// 如果baseDir为空，使用默认下载目录
	if baseDir == "" {
		baseDir = st.config.Paths.DownloadBase
	}

	// 根据视频源优先级确定任务优先级
	var priority downloader.TaskPriority
	sourcePriority := st.getSourcePriority(video)
	if sourcePriority >= 8 {
		priority = downloader.PriorityHigh
	} else if sourcePriority >= 5 {
		priority = downloader.PriorityNormal
	} else {
		priority = downloader.PriorityLow
	}

	// 使用统一的下载方法（自动创建视频专属文件夹并更新数据库路径）
	task, err := st.downloadManager.PrepareAndAddVideoTask(video, baseDir, priority, true)
	if err != nil {
		return fmt.Errorf("添加下载任务失败: %w", err)
	}

	utils.Info("[%s] 创建下载任务成功: %s (任务ID: %s)", st.ID, video.Name, task.ID)
	st.TasksCreated++
	return nil
}

// getSourcePriority 获取视频的来源优先级
func (st *SyncTask) getSourcePriority(video *models.Video) int {
	// 从视频关联的视频源获取优先级
	if video.FavoriteID != nil {
		var fav models.Favorite
		if err := st.db.First(&fav, *video.FavoriteID).Error; err == nil {
			return fav.Priority
		}
	}
	if video.SubmissionID != nil {
		var sub models.Submission
		if err := st.db.First(&sub, *video.SubmissionID).Error; err == nil {
			return sub.Priority
		}
	}
	if video.CollectionID != nil {
		var col models.Collection
		if err := st.db.First(&col, *video.CollectionID).Error; err == nil {
			return col.Priority
		}
	}
	if video.WatchLaterID != nil {
		var wl models.WatchLater
		if err := st.db.First(&wl, *video.WatchLaterID).Error; err == nil {
			return wl.Priority
		}
	}
	return 0 // 默认优先级
}

// updateSourceHealth 更新视频源健康状态
func (st *SyncTask) updateSourceHealth(sourceID, sourceType string, success bool, errorMsg string) {
	now := time.Now()
	numericID := extractIDFromSourceID(sourceID)

	if success {
		// 扫描成功：重置失败次数，更新健康状态
		updates := map[string]interface{}{
			"consecutive_failures": 0,
			"health_status":        "healthy",
			"last_scan_at":         now,
			"last_scan_error":      "",
			"last_success_at":      now,
		}

		switch sourceType {
		case "favorite":
			st.db.Model(&models.Favorite{}).Where("f_id = ?", numericID).Updates(updates)
		case "submission":
			st.db.Model(&models.Submission{}).Where("upper_id = ?", numericID).Updates(updates)
		case "collection":
			st.db.Model(&models.Collection{}).Where("c_id = ?", numericID).Updates(updates)
		case "watch_later":
			st.db.Model(&models.WatchLater{}).Updates(updates)
		}

		utils.Debug("[%s] 视频源 %s 健康状态已更新为 healthy", st.ID, sourceID)
	} else {
		// 扫描失败：增加失败次数，可能更新健康状态
		// 先查询当前失败次数
		var currentFailures int
		var healthStatus string

		switch sourceType {
		case "favorite":
			var fav models.Favorite
			if err := st.db.Where("f_id = ?", numericID).First(&fav).Error; err == nil {
				currentFailures = fav.ConsecutiveFailures
			}
		case "submission":
			var sub models.Submission
			if err := st.db.Where("upper_id = ?", numericID).First(&sub).Error; err == nil {
				currentFailures = sub.ConsecutiveFailures
			}
		case "collection":
			var col models.Collection
			if err := st.db.Where("c_id = ?", numericID).First(&col).Error; err == nil {
				currentFailures = col.ConsecutiveFailures
			}
		case "watch_later":
			var wl models.WatchLater
			if err := st.db.First(&wl).Error; err == nil {
				currentFailures = wl.ConsecutiveFailures
			}
		}

		// 增加失败次数
		currentFailures++

		// 根据失败次数确定健康状态
		if currentFailures >= 10 {
			healthStatus = "unhealthy"
			// 自动禁用
			utils.Warn("[%s] 视频源 %s 连续失败 %d 次，自动禁用", st.ID, sourceID, currentFailures)
		} else if currentFailures >= 5 {
			healthStatus = "degraded"
			utils.Warn("[%s] 视频源 %s 连续失败 %d 次，标记为 degraded", st.ID, sourceID, currentFailures)
		} else {
			healthStatus = "healthy"
		}

		updates := map[string]interface{}{
			"consecutive_failures": currentFailures,
			"health_status":        healthStatus,
			"last_scan_at":         now,
			"last_scan_error":      errorMsg,
		}

		// 如果达到失败阈值，禁用视频源
		if currentFailures >= 10 {
			updates["enabled"] = false
		}

		switch sourceType {
		case "favorite":
			st.db.Model(&models.Favorite{}).Where("f_id = ?", numericID).Updates(updates)
		case "submission":
			st.db.Model(&models.Submission{}).Where("upper_id = ?", numericID).Updates(updates)
		case "collection":
			st.db.Model(&models.Collection{}).Where("c_id = ?", numericID).Updates(updates)
		case "watch_later":
			st.db.Model(&models.WatchLater{}).Updates(updates)
		}

		utils.Debug("[%s] 视频源 %s 健康状态已更新为 %s (连续失败: %d)", st.ID, sourceID, healthStatus, currentFailures)
	}
}

// addError 添加错误
func (st *SyncTask) addError(err TaskError) {
	st.Errors = append(st.Errors, err)
}

// saveToDatabase 保存到数据库
func (st *SyncTask) saveToDatabase() error {
	// 计算耗时
	var durationMs int
	if st.EndAt != nil {
		durationMs = int(st.EndAt.Sub(st.StartAt).Milliseconds())
	}

	// 创建同步日志
	syncLog := models.SyncLog{
		TaskID:         st.ID,
		TriggerType:    st.TriggerType,
		Status:         st.Status,
		StartAt:        st.StartAt,
		EndAt:          st.EndAt,
		DurationMs:     durationMs,
		SourcesTotal:   st.SourcesTotal,
		SourcesScanned: st.SourcesScanned,
		SourcesFailed:  st.SourcesFailed,
		VideosFound:    st.VideosFound,
		VideosNew:      st.VideosNew,
		VideosFiltered: st.VideosFiltered,
		VideosQueued:   st.VideosQueued,
		TasksCreated:   st.TasksCreated,
	}

	// 使用事务保存
	return st.db.Transaction(func(tx *gorm.DB) error {
		// 保存同步日志
		if err := tx.Create(&syncLog).Error; err != nil {
			return fmt.Errorf("保存同步日志失败: %w", err)
		}

		// 保存视频源扫描记录
		for _, scan := range st.SourceScans {
			scan.SyncLogID = uint(syncLog.ID)
			if err := tx.Create(scan).Error; err != nil {
				return fmt.Errorf("保存视频源扫描记录失败: %w", err)
			}
		}

		return nil
	})
}
