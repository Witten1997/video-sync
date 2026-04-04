# Telegram Bot Integration First Milestone Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver the first complete Telegram milestone by extracting shared URL submission logic, adding private-chat Telegram Long Polling, closing the request-log and notification loop, and exposing Telegram configuration and visibility in the Web UI.

**Architecture:** Keep Telegram as an entry adapter, not a second download implementation. Both Web and Telegram must call a shared `URLDownloadService`, while Telegram-specific idempotency, polling state, logs, and notifications stay isolated inside `internal/telegram` plus dedicated API/UI surfaces.

**Tech Stack:** Go 1.24, Gin, GORM/Postgres, Vue 3, Vite, Element Plus, TypeScript

---

## File Map

### Phase 0: Source Documents

- Modify: `docs/features/telegram-bot-integration.md`
- Reference: `docs/superpowers/specs/2026-04-04-telegram-bot-integration-review-design.md`

### Phase 1: Shared URL Submission Service

- Create: `internal/service/url_download.go`
- Create: `internal/service/url_download_test.go`
- Create: `internal/api/handler_video_test.go`
- Modify: `internal/api/server.go`
- Modify: `internal/api/handler_video.go`
- Modify: `cmd/server/main.go`

### Phase 2: Telegram Minimal Submission Chain

- Create: `internal/telegram/types.go`
- Create: `internal/telegram/client.go`
- Create: `internal/telegram/parser.go`
- Create: `internal/telegram/parser_test.go`
- Create: `internal/telegram/access.go`
- Create: `internal/telegram/access_test.go`
- Create: `internal/telegram/poller.go`
- Create: `internal/telegram/poller_test.go`
- Create: `internal/telegram/service.go`
- Create: `internal/database/models/telegram_runtime_state.go`
- Modify: `internal/database/db.go`
- Modify: `internal/config/config.go`
- Modify: `internal/config/validator.go`
- Modify: `configs/config.example.yaml`
- Modify: `cmd/server/configs/config.yaml`
- Modify: `cmd/server/main.go`

### Phase 3: Telegram Request Logs And Notification Loop

- Create: `internal/database/models/telegram_request_log.go`
- Create: `internal/telegram/notifier.go`
- Create: `internal/telegram/store.go`
- Create: `internal/telegram/notifier_test.go`
- Modify: `internal/database/db.go`
- Modify: `internal/telegram/service.go`
- Modify: `internal/telegram/parser.go`
- Modify: `internal/telegram/client.go`

### Phase 4: Web Management Support

- Create: `internal/api/handler_telegram.go`
- Create: `internal/api/handler_config_test.go`
- Create: `web/src/api/telegram.ts`
- Create: `web/src/views/TelegramRequests.vue`
- Modify: `cmd/server/main.go`
- Modify: `internal/api/server.go`
- Modify: `internal/api/handler_config.go`
- Modify: `web/src/types/index.ts`
- Modify: `web/src/views/Config.vue`
- Modify: `web/src/router/index.ts`

### Deferred After First Milestone

- Later only: Webhook mode, group-chat support, `@botname` mention handling, distributed rate limiting, test-send actions

## Task 1: Sync The Source Proposal Document

**Files:**
- Modify: `docs/features/telegram-bot-integration.md`
- Reference: `docs/superpowers/specs/2026-04-04-telegram-bot-integration-review-design.md`

- [ ] **Step 1: Replace the proposal sections that conflict with the approved review**

Insert or rewrite the following structure in `docs/features/telegram-bot-integration.md` so the source proposal matches the approved design:

````md
## 4. 总体架构设计

```text
Telegram User / Web User
    ->
Entry Adapter
    - Web API handler
    - Telegram update processor
    ->
URLDownloadService
    - normalize request
    - resolve source type
    - apply business idempotency
    - create/reuse video and download record
    - enqueue task
    ->
DownloadManager
    - queue
    - execution
    - events
    ->
Telegram notification + Web visibility
```

## 5. 关键设计原则

- Telegram 适配层负责 `update_id` / `chat_id` / `message_id` 幂等
- `URLDownloadService` 负责下载提交幂等与复用策略
- `telegram_request_logs` 按“每个提取出的 URL 一行”建模
- Polling offset 必须持久化，且定义提交时机
- 首个完整里程碑包含 Web 配置与可观测性，不只是后端接入
- `bot_token` / `webhook_secret` 不得以明文返回给前端
````

- [ ] **Step 2: Add the delivery phases to the proposal**

Append a new section near the end of the proposal:

```md
## 14. 分阶段实施计划

### Phase 1：抽共享 URL 下载服务
- 目标：Web URL 下载改为调用共享服务，不改变外部 API

### Phase 2：Telegram 最小提交链路
- 目标：私聊白名单 + Long Polling + URL 提交闭环

### Phase 3：请求日志与通知闭环
- 目标：日志、回执、`/status`、`record_id` 关联

### Phase 4：Web 管理端支持
- 目标：配置、运行状态、请求日志页

### Phase 5：后续增强
- 目标：Webhook、群聊、分布式限流等增强项
```

- [ ] **Step 3: Verify the document contains the required corrections**

Run: `rg -n "URLDownloadService|telegram_request_logs|offset|bot_token|Phase 4" docs/features/telegram-bot-integration.md`

Expected: at least one match for each keyword family, proving the proposal has been updated before implementation starts.

- [ ] **Step 4: Commit the proposal sync**

```bash
git add docs/features/telegram-bot-integration.md
git commit -m "docs: align telegram integration proposal with approved review"
```

## Task 2: Phase 1 - Extract The Shared URL Download Service

**Files:**
- Create: `internal/service/url_download.go`
- Create: `internal/service/url_download_test.go`
- Create: `internal/api/handler_video_test.go`
- Modify: `internal/api/server.go`
- Modify: `internal/api/handler_video.go`
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Write a failing handler delegation test**

Create `internal/api/handler_video_test.go` with a fake service so the current handler fails until it delegates correctly:

```go
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bili-download/internal/service"
	"github.com/gin-gonic/gin"
)

type fakeURLDownloadService struct {
	lastReq *service.SubmitURLRequest
	result  *service.SubmitURLResult
	err     error
}

func (f *fakeURLDownloadService) Submit(ctx context.Context, req service.SubmitURLRequest) (*service.SubmitURLResult, error) {
	f.lastReq = &req
	return f.result, f.err
}

func TestHandleDownloadByURLDelegatesToService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fakeSvc := &fakeURLDownloadService{
		result: &service.SubmitURLResult{
			VideoID:         42,
			RecordID:        88,
			TaskID:          "task-1",
			VideoName:       "delegated",
			SourceType:      "ytdlp",
			IsExistingVideo: false,
			IsExistingTask:  false,
		},
	}

	s := &Server{urlDownloadSvc: fakeSvc}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(map[string]string{"url": "https://youtu.be/demo"})
	c.Request = httptest.NewRequest(http.MethodPost, "/api/videos/download-by-url", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	s.handleDownloadByURL(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if fakeSvc.lastReq == nil || fakeSvc.lastReq.URL != "https://youtu.be/demo" {
		t.Fatalf("expected handler to delegate request to shared service")
	}
}
```

- [ ] **Step 2: Run the new handler test and confirm failure**

Run: `go test ./internal/api -run TestHandleDownloadByURLDelegatesToService -v`

Expected: FAIL because `Server` does not yet expose `urlDownloadSvc` and `handleDownloadByURL` still performs inline branching.

- [ ] **Step 3: Implement the shared service contract**

Create `internal/service/url_download.go` with the shared request/result model and implementation entrypoint:

```go
package service

import (
	"context"

	"bili-download/internal/bilibili"
	"bili-download/internal/config"
	"bili-download/internal/database/models"
	"bili-download/internal/downloader"
	"gorm.io/gorm"
)

type SubmitURLRequest struct {
	URL              string
	TriggerChannel   string
	RequesterID      string
	RequesterName    string
	CorrelationID    string
	AllowExistingTask bool
}

type SubmitURLResult struct {
	VideoID         uint
	RecordID        uint
	TaskID          string
	VideoName       string
	SourceType      string
	IsExistingVideo bool
	IsExistingTask  bool
}

type URLDownloadService interface {
	Submit(ctx context.Context, req SubmitURLRequest) (*SubmitURLResult, error)
}

type urlDownloadService struct {
	cfg         *config.Config
	db          *gorm.DB
	biliClient  *bilibili.Client
	downloadMgr *downloader.DownloadManager
}

func NewURLDownloadService(cfg *config.Config, db *gorm.DB, biliClient *bilibili.Client, downloadMgr *downloader.DownloadManager) URLDownloadService {
	return &urlDownloadService{cfg: cfg, db: db, biliClient: biliClient, downloadMgr: downloadMgr}
}

func (s *urlDownloadService) Submit(ctx context.Context, req SubmitURLRequest) (*SubmitURLResult, error) {
	if isBilibiliURL(req.URL) {
		return s.submitBilibili(ctx, req)
	}
	return s.submitYTDLP(ctx, req)
}

func buildExternalVideoKey(extractor, videoID string) string {
	return extractor + "_" + videoID
}

func copyVideoResult(video *models.Video, taskID string, sourceType string, existing bool) *SubmitURLResult {
	return &SubmitURLResult{
		VideoID:         video.ID,
		TaskID:          taskID,
		VideoName:       video.Name,
		SourceType:      sourceType,
		IsExistingVideo: existing,
	}
}
```

- [ ] **Step 4: Move the handler to the shared service**

Modify `internal/api/server.go`, `internal/api/handler_video.go`, and `cmd/server/main.go` so the server owns a shared service instance and the handler only binds HTTP input/output:

```go
// internal/api/server.go
type Server struct {
	config           *config.Config
	configPath       string
	db               *gorm.DB
	biliClient       *bilibili.Client
	downloadMgr      *downloader.DownloadManager
	scheduler        *scheduler.Scheduler
	router           *gin.Engine
	httpServer       *http.Server
	websocketHub     *WebSocketHub
	imageProxyClient *http.Client
	frontendFS       fs.FS
	checkVersion     CheckVersionInfo
	checkVersionMu   sync.RWMutex
	UpgradeSignal    chan string
	urlDownloadSvc service.URLDownloadService
}

func NewServer(cfg *config.Config, configPath string, db *gorm.DB, biliClient *bilibili.Client, downloadMgr *downloader.DownloadManager, urlDownloadSvc service.URLDownloadService, frontendFS fs.FS) (*Server, error) {
	s := &Server{
		config:         cfg,
		configPath:     configPath,
		db:             db,
		biliClient:     biliClient,
		downloadMgr:    downloadMgr,
		urlDownloadSvc: urlDownloadSvc,
		frontendFS:     frontendFS,
	}
	s.scheduler = scheduler.NewScheduler(cfg, db, downloadMgr)
	s.setupRouter()
	s.startVersionChecker()
	return s, nil
}

// internal/api/handler_video.go
func (s *Server) handleDownloadByURL(c *gin.Context) {
	var req DownloadByURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, "缺少 URL 参数")
		return
	}

	result, err := s.urlDownloadSvc.Submit(c.Request.Context(), service.SubmitURLRequest{
		URL:               req.URL,
		TriggerChannel:    "web",
		RequesterID:       "web",
		RequesterName:     "web",
		CorrelationID:     "",
		AllowExistingTask: true,
	})
	if err != nil {
		respondError(c, 500, err.Error())
		return
	}

	respondSuccess(c, gin.H{
		"task_id":   result.TaskID,
		"record_id": result.RecordID,
		"video_id":  result.VideoID,
		"message":   "下载任务已创建",
	})
}

// cmd/server/main.go
urlDownloadSvc := service.NewURLDownloadService(cfg, db, biliClient, downloadMgr)
server, err := api.NewServer(cfg, configPath, db, biliClient, downloadMgr, urlDownloadSvc, frontend.GetFS())
```

- [ ] **Step 5: Add service tests for pure helper behavior**

Create `internal/service/url_download_test.go`:

```go
package service

import "testing"

func TestBuildExternalVideoKey(t *testing.T) {
	got := buildExternalVideoKey("YouTube", "abc123")
	if got != "YouTube_abc123" {
		t.Fatalf("expected YouTube_abc123, got %s", got)
	}
}
```

- [ ] **Step 6: Run focused verification**

Run: `go test ./internal/api ./internal/service -run "TestHandleDownloadByURLDelegatesToService|TestBuildExternalVideoKey" -v`

Expected: PASS

Run: `go test -vet=off ./internal/api ./internal/downloader ./internal/config ./internal/database/models -run ^$`

Expected: package compile succeeds without running long tests.

- [ ] **Step 7: Commit Phase 1**

```bash
git add internal/api/server.go internal/api/handler_video.go internal/api/handler_video_test.go internal/service/url_download.go internal/service/url_download_test.go cmd/server/main.go
git commit -m "feat: extract shared url download service"
```

## Task 3: Phase 2 - Build The Telegram Minimal Submission Chain

**Files:**
- Create: `internal/telegram/types.go`
- Create: `internal/telegram/client.go`
- Create: `internal/telegram/parser.go`
- Create: `internal/telegram/parser_test.go`
- Create: `internal/telegram/access.go`
- Create: `internal/telegram/access_test.go`
- Create: `internal/telegram/poller.go`
- Create: `internal/telegram/poller_test.go`
- Create: `internal/telegram/service.go`
- Create: `internal/database/models/telegram_runtime_state.go`
- Modify: `internal/database/db.go`
- Modify: `internal/config/config.go`
- Modify: `internal/config/validator.go`
- Modify: `configs/config.example.yaml`
- Modify: `cmd/server/configs/config.yaml`
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Write parser and access tests first**

Create `internal/telegram/parser_test.go` and `internal/telegram/access_test.go`:

```go
package telegram

import "testing"

func TestExtractURLsSupportsDirectMessageAndCommand(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{"direct", "https://youtu.be/demo", []string{"https://youtu.be/demo"}},
		{"command", "/download https://www.bilibili.com/video/BV1xx411c7mD", []string{"https://www.bilibili.com/video/BV1xx411c7mD"}},
	}

	for _, tt := range tests {
		got := ExtractURLs(tt.text, 3)
		if len(got) != len(tt.want) || got[0] != tt.want[0] {
			t.Fatalf("%s: got %v want %v", tt.name, got, tt.want)
		}
	}
}

func TestAllowPrivateChatOnly(t *testing.T) {
	cfg := AccessConfig{
		AllowedChatTypes: []string{"private"},
		AllowedChatIDs:   []int64{1001},
	}

	if err := CheckAccess(cfg, 1001, 2001, "private"); err != nil {
		t.Fatalf("expected private allow, got %v", err)
	}
	if err := CheckAccess(cfg, 9999, 2001, "group"); err == nil {
		t.Fatalf("expected group rejection")
	}
}
```

- [ ] **Step 2: Run parser and access tests and confirm failure**

Run: `go test ./internal/telegram -run "TestExtractURLsSupportsDirectMessageAndCommand|TestAllowPrivateChatOnly" -v`

Expected: FAIL because the package and helpers do not exist yet.

- [ ] **Step 3: Add Telegram config and runtime-state model**

Modify `internal/config/config.go`, `internal/config/validator.go`, and create `internal/database/models/telegram_runtime_state.go`:

```go
// internal/config/config.go
type TelegramConfig struct {
	Enabled            bool    `yaml:"enabled" json:"enabled"`
	BotToken           string  `yaml:"bot_token" json:"-"`
	Mode               string  `yaml:"mode" json:"mode"`
	PollTimeoutSeconds int     `yaml:"poll_timeout_seconds" json:"poll_timeout_seconds"`
	AllowedChatIDs     []int64 `yaml:"allowed_chat_ids" json:"allowed_chat_ids"`
	AllowedUserIDs     []int64 `yaml:"allowed_user_ids" json:"allowed_user_ids"`
	AllowedChatTypes   []string `yaml:"allowed_chat_types" json:"allowed_chat_types"`
	MaxURLsPerMessage  int     `yaml:"max_urls_per_message" json:"max_urls_per_message"`
	NotifyOnAccept     bool    `yaml:"notify_on_accept" json:"notify_on_accept"`
	NotifyOnComplete   bool    `yaml:"notify_on_complete" json:"notify_on_complete"`
	NotifyOnFail       bool    `yaml:"notify_on_fail" json:"notify_on_fail"`
}

type Config struct {
	Server   ServerConfig   `yaml:"server" mapstructure:"server" json:"server"`
	Database DatabaseConfig `yaml:"database" mapstructure:"database" json:"database"`
	Proxy    ProxyConfig    `yaml:"proxy" mapstructure:"proxy" json:"proxy"`
	Sync     SyncConfig     `yaml:"sync" mapstructure:"sync" json:"sync"`
	Paths    PathsConfig    `yaml:"paths" mapstructure:"paths" json:"paths"`
	Template TemplateConfig `yaml:"template" mapstructure:"template" json:"template"`
	Bilibili BilibiliConfig `yaml:"bilibili" mapstructure:"bilibili" json:"bilibili"`
	Quality  QualityConfig  `yaml:"quality" mapstructure:"quality" json:"quality"`
	Download DownloadConfig `yaml:"download" mapstructure:"download" json:"download"`
	Danmaku  DanmakuConfig  `yaml:"danmaku" mapstructure:"danmaku" json:"danmaku"`
	Advanced AdvancedConfig `yaml:"advanced" mapstructure:"advanced" json:"advanced"`
	Logging  LoggingConfig  `yaml:"logging" mapstructure:"logging" json:"logging"`
	Telegram TelegramConfig `yaml:"telegram" mapstructure:"telegram" json:"telegram"`
}

func (c *TelegramConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.BotToken == "" {
		return errors.New("telegram.bot_token 不能为空")
	}
	if c.Mode != "polling" {
		return errors.New("telegram.mode 仅支持 polling")
	}
	if c.PollTimeoutSeconds < 10 || c.PollTimeoutSeconds > 60 {
		return errors.New("telegram.poll_timeout_seconds 必须在 10-60 之间")
	}
	return nil
}

// internal/database/models/telegram_runtime_state.go
type TelegramRuntimeState struct {
	ID            uint      `gorm:"primaryKey"`
	BotName       string    `gorm:"size:128;uniqueIndex"`
	LastUpdateID  int64     `gorm:"not null;default:0"`
	LastPollAt    time.Time
	LastError     string    `gorm:"type:text"`
	LastErrorAt   *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
```

- [ ] **Step 4: Implement the Telegram adapter core**

Create `internal/telegram/types.go`, `client.go`, `parser.go`, `access.go`, `poller.go`, and `service.go` around a single entry service:

```go
package telegram

import (
	"context"

	"bili-download/internal/config"
	"bili-download/internal/service"
	"gorm.io/gorm"
)

type Update struct {
	UpdateID int64   `json:"update_id"`
	Message  *Message `json:"message,omitempty"`
}

type Message struct {
	MessageID int64  `json:"message_id"`
	Text      string `json:"text"`
}

type BotService struct {
	cfg            *config.Config
	db             *gorm.DB
	urlDownloadSvc service.URLDownloadService
	client         *Client
}

func NewBotService(cfg *config.Config, db *gorm.DB, urlDownloadSvc service.URLDownloadService) *BotService {
	return &BotService{cfg: cfg, db: db, urlDownloadSvc: urlDownloadSvc}
}

func (s *BotService) Start(ctx context.Context) error {
	if !s.cfg.Telegram.Enabled {
		return nil
	}
	return s.poll(ctx)
}
```

- [ ] **Step 5: Wire Telegram startup in `main.go`**

Modify `cmd/server/main.go`:

```go
telegramCtx, telegramCancel := context.WithCancel(context.Background())
defer telegramCancel()

telegramSvc := telegram.NewBotService(cfg, db, urlDownloadSvc)

go func() {
	if err := telegramSvc.Start(telegramCtx); err != nil {
		utils.Error("启动 Telegram 服务失败: %v", err)
	}
}()
```

And in the shutdown path:

```go
telegramCancel()
if err := server.Shutdown(ctx); err != nil {
	utils.Error("关闭 HTTP 服务器失败: %v", err)
}
if err := downloadMgr.Stop(); err != nil {
	utils.Error("关闭下载管理器失败: %v", err)
}
```

- [ ] **Step 6: Update config samples**

Append this block to both `configs/config.example.yaml` and `cmd/server/configs/config.yaml`:

```yaml
telegram:
  enabled: false
  bot_token: ""
  mode: "polling"
  poll_timeout_seconds: 30
  allowed_chat_ids: []
  allowed_user_ids: []
  allowed_chat_types:
    - "private"
  max_urls_per_message: 3
  notify_on_accept: true
  notify_on_complete: true
  notify_on_fail: true
```

- [ ] **Step 7: Run focused verification**

Run: `go test ./internal/telegram ./internal/config ./internal/database/models -run "TestExtractURLsSupportsDirectMessageAndCommand|TestAllowPrivateChatOnly" -v`

Expected: PASS

Run: `go test -vet=off ./cmd/server ./internal/telegram ./internal/config ./internal/database/models -run ^$`

Expected: package compile succeeds.

- [ ] **Step 8: Commit Phase 2**

```bash
git add internal/telegram internal/database/models/telegram_runtime_state.go internal/database/db.go internal/config/config.go internal/config/validator.go configs/config.example.yaml cmd/server/configs/config.yaml cmd/server/main.go
git commit -m "feat: add telegram polling submission adapter"
```

## Task 4: Phase 3 - Add Request Logs, Notifications, And `/status`

**Files:**
- Create: `internal/database/models/telegram_request_log.go`
- Create: `internal/telegram/notifier.go`
- Create: `internal/telegram/store.go`
- Create: `internal/telegram/notifier_test.go`
- Modify: `internal/database/db.go`
- Modify: `internal/telegram/service.go`
- Modify: `internal/telegram/parser.go`
- Modify: `internal/telegram/client.go`

- [ ] **Step 1: Write the notifier test first**

Create `internal/telegram/notifier_test.go`:

```go
package telegram

import "testing"

func TestBuildCompletionReplyUsesSingleMessageUpdate(t *testing.T) {
	reply := BuildStatusReply(StatusReplyInput{
		Stage:      "completed",
		Title:      "demo video",
		TaskID:     "task-1",
		RecordID:   99,
		MessageID:  123,
	})

	if reply.EditMessageID != 123 {
		t.Fatalf("expected edit-in-place reply, got %+v", reply)
	}
	if reply.Text == "" {
		t.Fatalf("expected non-empty completion text")
	}
}
```

- [ ] **Step 2: Run the notifier test and confirm failure**

Run: `go test ./internal/telegram -run TestBuildCompletionReplyUsesSingleMessageUpdate -v`

Expected: FAIL because notifier helpers do not exist yet.

- [ ] **Step 3: Add one-row-per-URL request logs**

Create `internal/database/models/telegram_request_log.go` and `internal/telegram/store.go`:

```go
type TelegramRequestLog struct {
	ID               uint      `gorm:"primaryKey"`
	UpdateID         int64     `gorm:"uniqueIndex:idx_tg_update_url"`
	ChatID           int64     `gorm:"index"`
	MessageID        int64     `gorm:"index"`
	UserID           int64     `gorm:"index"`
	RawText          string    `gorm:"type:text"`
	RawURL           string    `gorm:"size:1000"`
	URLHash          string    `gorm:"size:128;index"`
	Status           string    `gorm:"size:32;index"`
	VideoID          *uint     `gorm:"index"`
	RecordID         *uint     `gorm:"index"`
	TaskID           string    `gorm:"size:128;index"`
	ReplyMessageID   *int64
	ErrorMessage     string    `gorm:"type:text"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type Store struct {
	db *gorm.DB
}

func (s *Store) CreatePendingLog(log *models.TelegramRequestLog) error {
	return s.db.Create(log).Error
}
```

- [ ] **Step 4: Implement notification and `/status` handling**

Modify `internal/telegram/service.go`, `parser.go`, and `client.go` so the bot:

- creates a log row per extracted URL
- stores `record_id` as the primary correlation anchor
- edits the original reply on queue/completion/failure
- answers `/status` and `/status <task_id>`

Use this shape in `internal/telegram/notifier.go`:

```go
type StatusReplyInput struct {
	Stage         string
	Title         string
	TaskID        string
	RecordID      uint
	MessageID     int64
	ErrorMessage  string
}

type StatusReply struct {
	EditMessageID int64
	Text          string
}

func BuildStatusReply(in StatusReplyInput) StatusReply {
	switch in.Stage {
	case "completed":
		return StatusReply{
			EditMessageID: in.MessageID,
			Text:          "下载完成\n标题: " + in.Title + "\n记录ID: " + fmt.Sprint(in.RecordID),
		}
	case "failed":
		return StatusReply{
			EditMessageID: in.MessageID,
			Text:          "下载失败\n原因: " + in.ErrorMessage,
		}
	default:
		return StatusReply{
			EditMessageID: in.MessageID,
			Text:          "已接收下载请求\n任务ID: " + in.TaskID,
		}
	}
}
```

- [ ] **Step 5: Run focused verification**

Run: `go test ./internal/telegram ./internal/database/models -run "TestBuildCompletionReplyUsesSingleMessageUpdate" -v`

Expected: PASS

Run: `go test -vet=off ./internal/telegram ./internal/database/models ./internal/downloader -run ^$`

Expected: package compile succeeds.

- [ ] **Step 6: Commit Phase 3**

```bash
git add internal/database/models/telegram_request_log.go internal/telegram/notifier.go internal/telegram/notifier_test.go internal/telegram/store.go internal/telegram/service.go internal/telegram/parser.go internal/telegram/client.go internal/database/db.go
git commit -m "feat: add telegram request logs and notifications"
```

## Task 5: Phase 4 - Add Web Configuration, Status, And Request Logs

**Files:**
- Create: `internal/api/handler_telegram.go`
- Create: `internal/api/handler_config_test.go`
- Create: `web/src/api/telegram.ts`
- Create: `web/src/views/TelegramRequests.vue`
- Modify: `cmd/server/main.go`
- Modify: `internal/api/server.go`
- Modify: `internal/api/handler_config.go`
- Modify: `web/src/types/index.ts`
- Modify: `web/src/views/Config.vue`
- Modify: `web/src/router/index.ts`

- [ ] **Step 1: Write the failing config masking test**

Create `internal/api/handler_config_test.go`:

```go
package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bili-download/internal/config"
	"github.com/gin-gonic/gin"
)

func TestHandleGetConfigMasksTelegramSecrets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := &Server{
		config: &config.Config{
			Telegram: config.TelegramConfig{
				Enabled:  true,
				BotToken: "123:secret",
			},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/config", nil)

	s.handleGetConfig(c)

	if body := w.Body.String(); strings.Contains(body, "123:secret") {
		t.Fatalf("expected config response to hide telegram bot token")
	}
}
```

- [ ] **Step 2: Run the masking test and confirm failure**

Run: `go test ./internal/api -run TestHandleGetConfigMasksTelegramSecrets -v`

Expected: FAIL because `handleGetConfig` still serializes the in-memory config directly.

- [ ] **Step 3: Add backend Telegram status and request-log endpoints**

Create `internal/api/handler_telegram.go` and register routes in `internal/api/server.go`. Add `telegramSvc *telegram.BotService` to the `Server` struct and expose an attachment method:

```go
func (s *Server) AttachTelegramService(bot *telegram.BotService) {
	s.telegramSvc = bot
}

func (s *Server) handleTelegramStatus(c *gin.Context) {
	respondSuccess(c, gin.H{
		"enabled":       s.config.Telegram.Enabled,
		"running":       s.telegramSvc != nil && s.telegramSvc.IsRunning(),
		"mode":          s.config.Telegram.Mode,
		"last_poll_at":  s.telegramSvc.LastPollAt(),
		"last_error":    s.telegramSvc.LastError(),
	})
}

func (s *Server) handleTelegramRequestLogs(c *gin.Context) {
	query := s.db.Model(&models.TelegramRequestLog{}).Order("created_at DESC").Limit(50)
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if chatID := c.Query("chat_id"); chatID != "" {
		query = query.Where("chat_id = ?", chatID)
	}

	var rows []models.TelegramRequestLog
	if err := query.Find(&rows).Error; err != nil {
		respondInternalError(c, err)
		return
	}
	respondSuccess(c, gin.H{"items": rows})
}
```

Register:

```go
telegram := api.Group("/telegram")
{
	telegram.GET("/status", s.handleTelegramStatus)
	telegram.GET("/requests", s.handleTelegramRequestLogs)
}
```

Call the attachment in `cmd/server/main.go` right after both the HTTP server and Telegram service are created:

```go
server.AttachTelegramService(telegramSvc)
```

- [ ] **Step 4: Mask secrets in config read/write**

Modify `internal/api/handler_config.go` so the config response strips secrets and updates preserve existing values when Telegram secrets are omitted:

```go
func (s *Server) handleGetConfig(c *gin.Context) {
	cfg := *s.config
	if cfg.Telegram.BotToken != "" {
		cfg.Telegram.BotToken = ""
	}
	respondSuccess(c, cfg)
}

func mergeConfigFromMap(cfg *config.Config, configMap map[string]interface{}) {
	if proxyMap, ok := configMap["proxy"].(map[string]interface{}); ok {
		if enabled, exists := proxyMap["enabled"]; exists {
			cfg.Proxy.Enabled = enabled.(bool)
		}
	}
	if telegramMap, ok := configMap["telegram"].(map[string]interface{}); ok {
		if enabled, exists := telegramMap["enabled"]; exists {
			cfg.Telegram.Enabled = enabled.(bool)
		}
		if botToken, exists := telegramMap["bot_token"]; exists {
			if token, ok := botToken.(string); ok && token != "" {
				cfg.Telegram.BotToken = token
			}
		}
	}
}
```

- [ ] **Step 5: Add the Web API client and request-log page**

Create `web/src/api/telegram.ts`:

```ts
import { http } from '@/utils/request'

export const getTelegramStatus = () => http.get('/telegram/status')
export const getTelegramRequests = (params?: Record<string, any>) => http.get('/telegram/requests', { params })
```

Create `web/src/views/TelegramRequests.vue` with a simple filter form and table:

```vue
<template>
  <div>
    <el-card>
      <el-form inline>
        <el-form-item label="状态">
          <el-select v-model="filters.status" clearable>
            <el-option label="Queued" value="queued" />
            <el-option label="Completed" value="completed" />
            <el-option label="Failed" value="failed" />
          </el-select>
        </el-form-item>
        <el-button type="primary" @click="loadRows">刷新</el-button>
      </el-form>
      <el-table :data="rows">
        <el-table-column prop="created_at" label="时间" />
        <el-table-column prop="raw_url" label="URL" />
        <el-table-column prop="status" label="状态" />
        <el-table-column prop="task_id" label="任务ID" />
        <el-table-column prop="record_id" label="记录ID" />
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { getTelegramRequests } from '@/api/telegram'

const rows = ref<any[]>([])
const filters = ref({ status: '' })

const loadRows = async () => {
  const res = await getTelegramRequests(filters.value)
  rows.value = res.items || []
}

onMounted(loadRows)
</script>
```

- [ ] **Step 6: Add a Telegram tab to the existing config page**

Modify `web/src/types/index.ts` and `web/src/views/Config.vue`:

```ts
telegram: {
  enabled: boolean
  bot_token: string
  mode: string
  poll_timeout_seconds: number
  allowed_chat_ids: number[]
  allowed_user_ids: number[]
  allowed_chat_types: string[]
  max_urls_per_message: number
  notify_on_accept: boolean
  notify_on_complete: boolean
  notify_on_fail: boolean
}
```

```vue
<el-tab-pane label="Telegram" name="telegram">
  <el-form :model="config.telegram" label-width="180px">
    <el-form-item label="启用 Telegram">
      <el-switch v-model="config.telegram.enabled" />
    </el-form-item>
    <el-form-item label="Bot Token">
      <el-input v-model="config.telegram.bot_token" type="password" show-password placeholder="留空表示不修改已有 token" />
    </el-form-item>
    <el-form-item label="轮询超时（秒）">
      <el-input-number v-model="config.telegram.poll_timeout_seconds" :min="10" :max="60" />
    </el-form-item>
  </el-form>
</el-tab-pane>
```

Also update `getCurrentTabConfig()` so the Telegram tab submits only `telegram: config.value.telegram`.

- [ ] **Step 7: Add the route**

Modify `web/src/router/index.ts`:

```ts
{
  path: '/telegram/requests',
  name: 'TelegramRequests',
  component: () => import('@/views/TelegramRequests.vue')
}
```

- [ ] **Step 8: Run verification**

Run: `go test ./internal/api -run TestHandleGetConfigMasksTelegramSecrets -v`

Expected: PASS

Run: `go test -vet=off ./internal/api ./internal/config ./internal/database/models -run ^$`

Expected: package compile succeeds.

Run: `npm run build`

Workdir: `web`

Expected: Vue type-check and production build succeed.

- [ ] **Step 9: Commit Phase 4**

```bash
git add cmd/server/main.go internal/api/server.go internal/api/handler_config.go internal/api/handler_config_test.go internal/api/handler_telegram.go web/src/api/telegram.ts web/src/views/TelegramRequests.vue web/src/types/index.ts web/src/views/Config.vue web/src/router/index.ts
git commit -m "feat: add telegram web management surfaces"
```

## Out Of Scope For This Plan

Do not implement these in the first complete milestone:

- Telegram Webhook mode
- group-chat support
- `@botname` mention rules
- distributed rate limiting
- bot-side admin actions such as reconnect and test-send

Write a separate plan after Phase 4 is accepted.
