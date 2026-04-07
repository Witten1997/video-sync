package service

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"bili-download/internal/bilibili"
	"bili-download/internal/config"
	"bili-download/internal/database/models"
	"bili-download/internal/downloader"

	"gorm.io/gorm"
)

type URLDownloadErrorType string

const (
	URLDownloadErrorTypeValidation URLDownloadErrorType = "validation"
	URLDownloadErrorTypeInternal   URLDownloadErrorType = "internal"
)

type URLDownloadError struct {
	Type    URLDownloadErrorType
	Message string
	Err     error
}

func (e *URLDownloadError) Error() string {
	return e.Message
}

func (e *URLDownloadError) Unwrap() error {
	return e.Err
}

type URLDownloadRequest struct {
	URL           string
	Channel       string
	Requester     string
	CorrelationID string
}

type URLDownloadSourceType string

const (
	URLDownloadSourceTypeBilibili URLDownloadSourceType = "bilibili"
	URLDownloadSourceTypeExternal URLDownloadSourceType = "external"
)

type URLDownloadOutcome string

const (
	URLDownloadOutcomeCreatedVideo  URLDownloadOutcome = "created_video"
	URLDownloadOutcomeExistingVideo URLDownloadOutcome = "existing_video"
)

type URLDownloadPage struct {
	ID             uint      `json:"id"`
	VideoID        uint      `json:"video_id"`
	CID            int64     `json:"cid"`
	PID            int       `json:"pid"`
	Name           string    `json:"name"`
	Duration       int       `json:"duration"`
	Width          int       `json:"width"`
	Height         int       `json:"height"`
	Image          string    `json:"image"`
	DownloadStatus int       `json:"download_status"`
	Path           string    `json:"path"`
	CreatedAt      time.Time `json:"created_at"`
}

type URLDownloadVideo struct {
	ID             uint              `json:"id"`
	BVid           string            `json:"bvid"`
	Name           string            `json:"name"`
	Intro          string            `json:"intro"`
	Cover          string            `json:"cover"`
	Tags           []string          `json:"tags"`
	UpperID        int64             `json:"upper_id"`
	UpperName      string            `json:"upper_name"`
	UpperFace      string            `json:"upper_face"`
	ViewCount      int               `json:"view_count"`
	Category       int               `json:"category"`
	PubTime        time.Time         `json:"pubtime"`
	FavTime        time.Time         `json:"favtime"`
	CTime          time.Time         `json:"ctime"`
	SinglePage     bool              `json:"single_page"`
	Valid          bool              `json:"valid"`
	ShouldDownload bool              `json:"should_download"`
	DownloadStatus int               `json:"download_status"`
	Path           string            `json:"path"`
	FavoriteID     *uint             `json:"favorite_id,omitempty"`
	WatchLaterID   *uint             `json:"watch_later_id,omitempty"`
	CollectionID   *uint             `json:"collection_id,omitempty"`
	SubmissionID   *uint             `json:"submission_id,omitempty"`
	Pages          []URLDownloadPage `json:"pages,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
}

type URLDownloadResult struct {
	TaskID     string                `json:"task_id"`
	TaskType   string                `json:"task_type"`
	RecordID   uint                  `json:"record_id,omitempty"`
	VideoID    uint                  `json:"video_id"`
	VideoBVID  string                `json:"video_bvid"`
	Title      string                `json:"title"`
	SourceType URLDownloadSourceType `json:"source_type"`
	Outcome    URLDownloadOutcome    `json:"outcome"`
	Video      URLDownloadVideo      `json:"video"`
}

type URLDownloadSubmitter interface {
	Submit(ctx context.Context, req URLDownloadRequest) (*URLDownloadResult, error)
}

type URLDownloadService struct {
	config      *config.Config
	db          *gorm.DB
	biliClient  *bilibili.Client
	downloadMgr *downloader.DownloadManager
}

func NewURLDownloadService(cfg *config.Config, db *gorm.DB, biliClient *bilibili.Client, downloadMgr *downloader.DownloadManager) *URLDownloadService {
	return &URLDownloadService{
		config:      cfg,
		db:          db,
		biliClient:  biliClient,
		downloadMgr: downloadMgr,
	}
}

func (s *URLDownloadService) Submit(ctx context.Context, req URLDownloadRequest) (*URLDownloadResult, error) {
	if isBilibiliURL(req.URL) {
		return s.submitBilibili(ctx, req)
	}

	return s.submitYtdlp(ctx, req)
}

func (s *URLDownloadService) submitBilibili(_ context.Context, req URLDownloadRequest) (*URLDownloadResult, error) {
	bvid, err := s.biliClient.ParseVideoURL(req.URL)
	if err != nil {
		return nil, &URLDownloadError{
			Type:    URLDownloadErrorTypeValidation,
			Message: "无效的B站视频链接: " + err.Error(),
			Err:     err,
		}
	}

	var existingVideo models.Video
	err = s.db.Where("bvid = ?", bvid).Preload("Pages").First(&existingVideo).Error
	if err == nil {
		task, taskErr := s.downloadMgr.PrepareAndAddVideoTask(&existingVideo, s.config.Paths.DownloadBase, 0, true)
		if taskErr != nil {
			return nil, &URLDownloadError{
				Type:    URLDownloadErrorTypeInternal,
				Message: taskErr.Error(),
				Err:     taskErr,
			}
		}

		return newURLDownloadResult(task, &existingVideo, URLDownloadSourceTypeBilibili, URLDownloadOutcomeExistingVideo), nil
	}

	videoDetail, err := s.biliClient.GetVideoDetail(bvid)
	if err != nil {
		return nil, &URLDownloadError{
			Type:    URLDownloadErrorTypeInternal,
			Message: "获取视频信息失败: " + err.Error(),
			Err:     err,
		}
	}

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

	videoTags, err := s.biliClient.GetVideoTags(bvid)
	if err == nil && len(videoTags) > 0 {
		tags := make([]string, len(videoTags))
		for i, tag := range videoTags {
			tags[i] = tag.TagName
		}
		video.Tags = tags
	}

	if err := s.db.Create(&video).Error; err != nil {
		return nil, &URLDownloadError{
			Type:    URLDownloadErrorTypeInternal,
			Message: err.Error(),
			Err:     err,
		}
	}

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
			return nil, &URLDownloadError{
				Type:    URLDownloadErrorTypeInternal,
				Message: err.Error(),
				Err:     err,
			}
		}
	}

	if err := s.db.Preload("Pages").First(&video, video.ID).Error; err != nil {
		return nil, &URLDownloadError{
			Type:    URLDownloadErrorTypeInternal,
			Message: err.Error(),
			Err:     err,
		}
	}

	task, err := s.downloadMgr.PrepareAndAddVideoTask(&video, s.config.Paths.DownloadBase, 0, true)
	if err != nil {
		return nil, &URLDownloadError{
			Type:    URLDownloadErrorTypeInternal,
			Message: err.Error(),
			Err:     err,
		}
	}

	return newURLDownloadResult(task, &video, URLDownloadSourceTypeBilibili, URLDownloadOutcomeCreatedVideo), nil
}

func (s *URLDownloadService) submitYtdlp(ctx context.Context, req URLDownloadRequest) (*URLDownloadResult, error) {
	ytdlpDl := downloader.NewYtdlpDownloader(s.config, nil)
	info, err := ytdlpDl.GetVideoInfo(ctx, req.URL, "")
	if err != nil {
		return nil, &URLDownloadError{
			Type:    URLDownloadErrorTypeInternal,
			Message: "获取视频信息失败: " + err.Error(),
			Err:     err,
		}
	}

	title, _ := info["title"].(string)
	if title == "" {
		title = "未知视频"
	}
	description, _ := info["description"].(string)
	thumbnail, _ := info["thumbnail"].(string)
	uploader, _ := info["uploader"].(string)
	videoID, _ := info["id"].(string)
	if videoID == "" {
		videoID = fmt.Sprintf("ytdlp_%d", time.Now().UnixNano())
	}

	extractor, _ := info["extractor_key"].(string)
	bvid := BuildExternalVideoKey(extractor, videoID)

	var existingVideo models.Video
	if s.db.Where("bvid = ?", bvid).First(&existingVideo).Error == nil {
		task, taskErr := s.downloadMgr.PrepareAndAddYtdlpTask(&existingVideo, req.URL, s.config.Paths.URLDownloadBase())
		if taskErr != nil {
			return nil, &URLDownloadError{
				Type:    URLDownloadErrorTypeInternal,
				Message: taskErr.Error(),
				Err:     taskErr,
			}
		}

		return newURLDownloadResult(task, &existingVideo, URLDownloadSourceTypeExternal, URLDownloadOutcomeExistingVideo), nil
	}

	pubTime := time.Now()
	if ts, ok := info["timestamp"].(float64); ok && ts > 0 {
		pubTime = time.Unix(int64(ts), 0)
	} else if uploadDate, ok := info["upload_date"].(string); ok && len(uploadDate) == 8 {
		if parsed, parseErr := time.Parse("20060102", uploadDate); parseErr == nil {
			pubTime = parsed
		}
	}

	now := time.Now()
	video := models.Video{
		BVid:           bvid,
		Name:           title,
		Intro:          description,
		Cover:          thumbnail,
		UpperName:      uploader,
		PubTime:        pubTime,
		FavTime:        now,
		CTime:          pubTime,
		SinglePage:     true,
		Valid:          true,
		ShouldDownload: true,
	}

	if err := s.db.Create(&video).Error; err != nil {
		return nil, &URLDownloadError{
			Type:    URLDownloadErrorTypeInternal,
			Message: err.Error(),
			Err:     err,
		}
	}

	task, err := s.downloadMgr.PrepareAndAddYtdlpTask(&video, req.URL, s.config.Paths.URLDownloadBase())
	if err != nil {
		return nil, &URLDownloadError{
			Type:    URLDownloadErrorTypeInternal,
			Message: err.Error(),
			Err:     err,
		}
	}

	return newURLDownloadResult(task, &video, URLDownloadSourceTypeExternal, URLDownloadOutcomeCreatedVideo), nil
}

func BuildExternalVideoKey(extractorKey, videoID string) string {
	key := fmt.Sprintf("%s_%s", extractorKey, videoID)
	if len(key) > 20 {
		key = key[:20]
	}
	return key
}

func isBilibiliURL(rawURL string) bool {
	if len(rawURL) == 12 && rawURL[:2] == "BV" {
		return true
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	host := parsedURL.Hostname()
	return host == "www.bilibili.com" || host == "bilibili.com" || host == "b23.tv" || host == "m.bilibili.com"
}

func (r *URLDownloadResult) SuccessMessage() string {
	if r != nil && r.Outcome == URLDownloadOutcomeExistingVideo {
		return "视频已存在，下载任务已创建"
	}

	return "视频信息已获取，下载任务已创建"
}

func (r *URLDownloadResult) APIVideo() models.Video {
	if r == nil {
		return models.Video{}
	}

	video := models.Video{
		ID:             r.Video.ID,
		BVid:           r.Video.BVid,
		Name:           r.Video.Name,
		Intro:          r.Video.Intro,
		Cover:          r.Video.Cover,
		Tags:           append(models.Video{}.Tags, r.Video.Tags...),
		UpperID:        r.Video.UpperID,
		UpperName:      r.Video.UpperName,
		UpperFace:      r.Video.UpperFace,
		ViewCount:      r.Video.ViewCount,
		Category:       r.Video.Category,
		PubTime:        r.Video.PubTime,
		FavTime:        r.Video.FavTime,
		CTime:          r.Video.CTime,
		SinglePage:     r.Video.SinglePage,
		Valid:          r.Video.Valid,
		ShouldDownload: r.Video.ShouldDownload,
		DownloadStatus: r.Video.DownloadStatus,
		Path:           r.Video.Path,
		FavoriteID:     r.Video.FavoriteID,
		WatchLaterID:   r.Video.WatchLaterID,
		CollectionID:   r.Video.CollectionID,
		SubmissionID:   r.Video.SubmissionID,
		CreatedAt:      r.Video.CreatedAt,
	}

	if len(r.Video.Pages) == 0 {
		return video
	}

	video.Pages = make([]models.Page, 0, len(r.Video.Pages))
	for _, page := range r.Video.Pages {
		video.Pages = append(video.Pages, models.Page{
			ID:             page.ID,
			VideoID:        page.VideoID,
			CID:            page.CID,
			PID:            page.PID,
			Name:           page.Name,
			Duration:       page.Duration,
			Width:          page.Width,
			Height:         page.Height,
			Image:          page.Image,
			DownloadStatus: page.DownloadStatus,
			Path:           page.Path,
			CreatedAt:      page.CreatedAt,
		})
	}

	return video
}

func newURLDownloadResult(task *downloader.DownloadTask, video *models.Video, sourceType URLDownloadSourceType, outcome URLDownloadOutcome) *URLDownloadResult {
	result := &URLDownloadResult{
		SourceType: sourceType,
		Outcome:    outcome,
	}

	if task != nil {
		result.TaskID = task.ID
		result.TaskType = string(task.Type)
		result.RecordID = task.RecordID
	}

	if video == nil {
		return result
	}

	result.VideoID = video.ID
	result.VideoBVID = video.BVid
	result.Title = video.Name
	result.Video = toURLDownloadVideo(video)
	return result
}

func toURLDownloadVideo(video *models.Video) URLDownloadVideo {
	result := URLDownloadVideo{
		ID:             video.ID,
		BVid:           video.BVid,
		Name:           video.Name,
		Intro:          video.Intro,
		Cover:          video.Cover,
		Tags:           append([]string(nil), video.Tags...),
		UpperID:        video.UpperID,
		UpperName:      video.UpperName,
		UpperFace:      video.UpperFace,
		ViewCount:      video.ViewCount,
		Category:       video.Category,
		PubTime:        video.PubTime,
		FavTime:        video.FavTime,
		CTime:          video.CTime,
		SinglePage:     video.SinglePage,
		Valid:          video.Valid,
		ShouldDownload: video.ShouldDownload,
		DownloadStatus: video.DownloadStatus,
		Path:           video.Path,
		FavoriteID:     video.FavoriteID,
		WatchLaterID:   video.WatchLaterID,
		CollectionID:   video.CollectionID,
		SubmissionID:   video.SubmissionID,
		CreatedAt:      video.CreatedAt,
	}

	if len(video.Pages) == 0 {
		return result
	}

	result.Pages = make([]URLDownloadPage, 0, len(video.Pages))
	for _, page := range video.Pages {
		result.Pages = append(result.Pages, URLDownloadPage{
			ID:             page.ID,
			VideoID:        page.VideoID,
			CID:            page.CID,
			PID:            page.PID,
			Name:           page.Name,
			Duration:       page.Duration,
			Width:          page.Width,
			Height:         page.Height,
			Image:          page.Image,
			DownloadStatus: page.DownloadStatus,
			Path:           page.Path,
			CreatedAt:      page.CreatedAt,
		})
	}

	return result
}
