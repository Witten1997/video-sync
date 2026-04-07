package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bili-download/internal/config"
	"bili-download/internal/service"

	"github.com/gin-gonic/gin"
)

type stubURLDownloadService struct {
	lastRequest service.URLDownloadRequest
	result      *service.URLDownloadResult
	err         error
}

func (s *stubURLDownloadService) Submit(_ context.Context, req service.URLDownloadRequest) (*service.URLDownloadResult, error) {
	s.lastRequest = req
	return s.result, s.err
}

func TestHandleDownloadByURLDelegatesToService(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/videos/download-by-url", bytes.NewBufferString(`{"url":"https://www.youtube.com/watch?v=test-video"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	svc := &stubURLDownloadService{
		result: &service.URLDownloadResult{
			TaskID:     "task-123",
			VideoID:    42,
			VideoBVID:  "youtube_test_video",
			Title:      "delegated",
			SourceType: service.URLDownloadSourceTypeExternal,
			Outcome:    service.URLDownloadOutcomeCreatedVideo,
			Video: service.URLDownloadVideo{
				ID:   42,
				BVid: "youtube_test_video",
				Name: "delegated",
			},
		},
	}

	server := &Server{urlDownloadService: svc}
	server.handleDownloadByURL(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if svc.lastRequest.URL != "https://www.youtube.com/watch?v=test-video" {
		t.Fatalf("expected delegated url to match request, got %q", svc.lastRequest.URL)
	}
	if svc.lastRequest.Channel != "web" {
		t.Fatalf("expected channel web, got %q", svc.lastRequest.Channel)
	}
	if svc.lastRequest.Requester != "web" {
		t.Fatalf("expected requester web, got %q", svc.lastRequest.Requester)
	}

	var resp Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.Code != 0 {
		t.Fatalf("expected api code 0, got %d", resp.Code)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected response data map, got %T", resp.Data)
	}

	if data["task_id"] != "task-123" {
		t.Fatalf("expected task_id task-123, got %#v", data["task_id"])
	}
	if data["message"] != "视频信息已获取，下载任务已创建" {
		t.Fatalf("expected handler message from structured result, got %#v", data["message"])
	}

	video, ok := data["video"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected video payload map, got %T", data["video"])
	}
	if video["id"] != float64(42) {
		t.Fatalf("expected video id 42, got %#v", video["id"])
	}
	if video["bvid"] != "youtube_test_video" {
		t.Fatalf("expected copied bvid, got %#v", video["bvid"])
	}
	if video["name"] != "delegated" {
		t.Fatalf("expected copied name, got %#v", video["name"])
	}
}

func TestHandleDownloadByURLMapsValidationError(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/videos/download-by-url", bytes.NewBufferString(`{"url":"https://bad.example/video"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	server := &Server{
		urlDownloadService: &stubURLDownloadService{
			err: &service.URLDownloadError{
				Type:    service.URLDownloadErrorTypeValidation,
				Message: "bad url",
			},
		},
	}

	server.handleDownloadByURL(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	var resp Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected api code %d, got %d", http.StatusBadRequest, resp.Code)
	}
	if resp.Message != "bad url" {
		t.Fatalf("expected validation message, got %q", resp.Message)
	}
}

func TestHandleDownloadByURLMapsInternalError(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/videos/download-by-url", bytes.NewBufferString(`{"url":"https://bad.example/video"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	server := &Server{
		urlDownloadService: &stubURLDownloadService{
			err: &service.URLDownloadError{
				Type:    service.URLDownloadErrorTypeInternal,
				Message: "boom",
				Err:     errors.New("boom"),
			},
		},
	}

	server.handleDownloadByURL(ctx)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}

	var resp Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected api code %d, got %d", http.StatusInternalServerError, resp.Code)
	}
	if resp.Message != "boom" {
		t.Fatalf("expected internal message, got %q", resp.Message)
	}
}

func TestNewServerRejectsNilURLDownloadService(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}

	_, err := NewServer(cfg, "", nil, nil, nil, nil, nil)
	if err == nil {
		t.Fatal("expected constructor to reject nil url download service")
	}
	if !strings.Contains(err.Error(), "url download service") {
		t.Fatalf("expected nil-service error message, got %v", err)
	}
}
