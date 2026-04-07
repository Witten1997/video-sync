package service

import (
	"testing"
	"time"

	"bili-download/internal/database/models"
	"bili-download/internal/downloader"
)

func TestBuildExternalVideoKey(t *testing.T) {
	t.Parallel()

	if got := BuildExternalVideoKey("YouTube", "abc123"); got != "YouTube_abc123" {
		t.Fatalf("expected untrimmed key, got %q", got)
	}

	if got := BuildExternalVideoKey("VeryLongExtractor", "video-identifier"); got != "VeryLongExtractor_vi" {
		t.Fatalf("expected trimmed key, got %q", got)
	}
}

func TestIsBilibiliURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		rawURL string
		want   bool
	}{
		{name: "bvid", rawURL: "BV1xx411c7mD", want: true},
		{name: "desktop host", rawURL: "https://www.bilibili.com/video/BV1xx411c7mD", want: true},
		{name: "mobile host", rawURL: "https://m.bilibili.com/video/BV1xx411c7mD", want: true},
		{name: "short host", rawURL: "https://b23.tv/abc123", want: true},
		{name: "non bilibili", rawURL: "https://www.youtube.com/watch?v=test-video", want: false},
		{name: "invalid url", rawURL: "://bad-url", want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := isBilibiliURL(tt.rawURL); got != tt.want {
				t.Fatalf("expected %v for %q, got %v", tt.want, tt.rawURL, got)
			}
		})
	}
}

func TestNewURLDownloadResult(t *testing.T) {
	t.Parallel()

	createdAt := time.Unix(1712198400, 0)
	video := &models.Video{
		ID:             42,
		BVid:           "BV1xx411c7mD",
		Name:           "delegated",
		Intro:          "intro",
		Cover:          "cover.jpg",
		Tags:           []string{"tag-a", "tag-b"},
		UpperID:        99,
		UpperName:      "uploader",
		UpperFace:      "face.jpg",
		ViewCount:      123,
		Category:       7,
		PubTime:        createdAt,
		FavTime:        createdAt,
		CTime:          createdAt,
		SinglePage:     false,
		Valid:          true,
		ShouldDownload: true,
		DownloadStatus: 1,
		Path:           "video/path",
		CreatedAt:      createdAt,
		Pages: []models.Page{
			{
				ID:             8,
				VideoID:        42,
				CID:            1001,
				PID:            1,
				Name:           "P1",
				Duration:       120,
				Width:          1920,
				Height:         1080,
				Image:          "page.jpg",
				DownloadStatus: 1,
				Path:           "page/path",
				CreatedAt:      createdAt,
			},
		},
	}
	task := &downloader.DownloadTask{
		ID:       "task-42",
		Type:     downloader.TaskTypeVideo,
		RecordID: 88,
	}

	result := newURLDownloadResult(task, video, URLDownloadSourceTypeBilibili, URLDownloadOutcomeExistingVideo)

	if result.TaskID != "task-42" {
		t.Fatalf("expected task id task-42, got %q", result.TaskID)
	}
	if result.TaskType != string(downloader.TaskTypeVideo) {
		t.Fatalf("expected task type %q, got %q", downloader.TaskTypeVideo, result.TaskType)
	}
	if result.RecordID != 88 {
		t.Fatalf("expected record id 88, got %d", result.RecordID)
	}
	if result.VideoID != 42 {
		t.Fatalf("expected video id 42, got %d", result.VideoID)
	}
	if result.VideoBVID != "BV1xx411c7mD" {
		t.Fatalf("expected bvid to be copied, got %q", result.VideoBVID)
	}
	if result.Title != "delegated" {
		t.Fatalf("expected title to be copied, got %q", result.Title)
	}
	if result.SourceType != URLDownloadSourceTypeBilibili {
		t.Fatalf("expected bilibili source type, got %q", result.SourceType)
	}
	if result.Outcome != URLDownloadOutcomeExistingVideo {
		t.Fatalf("expected existing outcome, got %q", result.Outcome)
	}
	if got := result.SuccessMessage(); got != "视频已存在，下载任务已创建" {
		t.Fatalf("expected existing-video message, got %q", got)
	}
	if result.Video.ID != 42 || result.Video.BVid != "BV1xx411c7mD" || result.Video.Name != "delegated" {
		t.Fatalf("expected stable video summary to copy identifiers, got %#v", result.Video)
	}
	if len(result.Video.Tags) != 2 || result.Video.Tags[0] != "tag-a" {
		t.Fatalf("expected tags to be copied into stable DTO, got %#v", result.Video.Tags)
	}
	if len(result.Video.Pages) != 1 || result.Video.Pages[0].ID != 8 || result.Video.Pages[0].CID != 1001 {
		t.Fatalf("expected pages to be copied into stable DTO, got %#v", result.Video.Pages)
	}

	apiVideo := result.APIVideo()
	if len(apiVideo.Pages) != 1 {
		t.Fatalf("expected api video to preserve page list, got %#v", apiVideo.Pages)
	}
	if apiVideo.Pages[0].VideoID != 42 || apiVideo.Pages[0].CID != 1001 {
		t.Fatalf("expected api video page fields to match original video, got %#v", apiVideo.Pages[0])
	}
}
