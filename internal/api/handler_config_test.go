package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bili-download/internal/bilibili"
	"bili-download/internal/config"
	"bili-download/internal/telegram"

	"github.com/gin-gonic/gin"
)

type fakeConfigTelegramService struct {
	applyConfigFn func(*config.Config) (bool, error)
}

func (f *fakeConfigTelegramService) IsRunning() bool {
	return false
}

func (f *fakeConfigTelegramService) Reconnect() error {
	return nil
}

func (f *fakeConfigTelegramService) ApplyConfig(cfg *config.Config) (bool, error) {
	if f.applyConfigFn != nil {
		return f.applyConfigFn(cfg)
	}
	return false, nil
}

func (f *fakeConfigTelegramService) HandleWebhookUpdate(context.Context, telegram.Update) error {
	return nil
}

func loadConfigForUpdateHandlerTest(t *testing.T) (*config.Config, string) {
	t.Helper()

	sourcePath := filepath.Join("..", "..", "cmd", "server", "configs", "config.yaml")
	raw, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read source config: %v", err)
	}

	tempPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(tempPath, raw, 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	cfg, err := config.Load(tempPath)
	if err != nil {
		t.Fatalf("load temp config: %v", err)
	}

	return cfg, tempPath
}

func TestHandleGetConfigMasksTelegramSecrets(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	server := &Server{
		config: &config.Config{
			Telegram: config.TelegramConfig{
				Enabled:       true,
				BotToken:      "123:secret-token",
				WebhookSecret: "super-secret",
			},
		},
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/config", nil)

	server.handleGetConfig(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if body := recorder.Body.String(); strings.Contains(body, "123:secret-token") || strings.Contains(body, "super-secret") {
		t.Fatalf("expected config response to hide telegram secrets, got %s", body)
	}

	var resp Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected response data map, got %T", resp.Data)
	}

	telegramData, ok := data["telegram"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected telegram config map, got %T", data["telegram"])
	}

	if value, exists := telegramData["bot_token"]; exists && value != "" {
		t.Fatalf("expected bot_token to be empty when present, got %#v", value)
	}
	if value, exists := telegramData["webhook_secret"]; exists && value != "" {
		t.Fatalf("expected webhook_secret to be empty when present, got %#v", value)
	}
	if value, ok := telegramData["bot_token_configured"].(bool); !ok || !value {
		t.Fatalf("expected bot_token_configured to be true, got %#v", telegramData["bot_token_configured"])
	}
	if value, ok := telegramData["webhook_secret_configured"].(bool); !ok || !value {
		t.Fatalf("expected webhook_secret_configured to be true, got %#v", telegramData["webhook_secret_configured"])
	}
}

func TestHandleGetConfigTelegramSecretFlagsFalseWhenSecretsBlank(t *testing.T) {
	t.Parallel()

	server := &Server{
		config: &config.Config{
			Telegram: config.TelegramConfig{
				BotToken:      "   ",
				WebhookSecret: "\t",
			},
		},
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/config", nil)

	server.handleGetConfig(ctx)

	var resp Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected response data map, got %T", resp.Data)
	}

	telegramData, ok := data["telegram"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected telegram config map, got %T", data["telegram"])
	}

	if value, ok := telegramData["bot_token_configured"].(bool); !ok || value {
		t.Fatalf("expected bot_token_configured to be false, got %#v", telegramData["bot_token_configured"])
	}
	if value, ok := telegramData["webhook_secret_configured"].(bool); !ok || value {
		t.Fatalf("expected webhook_secret_configured to be false, got %#v", telegramData["webhook_secret_configured"])
	}
}

func TestMergeConfigFromMapPreservesTelegramBotTokenWhenEmpty(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Telegram: config.TelegramConfig{
			Enabled:            true,
			BotToken:           "persist-me",
			WebhookSecret:      "persist-secret",
			Mode:               "polling",
			PollTimeoutSeconds: 30,
			AllowedChatIDs:     []int64{1001},
			AllowedUserIDs:     []int64{2001},
			AllowedChatTypes:   []string{"private"},
			MaxURLsPerMessage:  1,
			NotifyOnAccept:     true,
			NotifyOnComplete:   true,
			NotifyOnFail:       true,
		},
	}

	mergeConfigFromMap(cfg, map[string]interface{}{
		"telegram": map[string]interface{}{
			"enabled":              false,
			"bot_token":            "",
			"webhook_secret":       "",
			"mode":                 "polling",
			"poll_timeout_seconds": float64(45),
			"allowed_chat_ids":     []interface{}{float64(3001), float64(3002)},
			"allowed_user_ids":     []interface{}{float64(4001)},
			"allowed_chat_types":   []interface{}{"private"},
			"max_urls_per_message": float64(1),
			"notify_on_accept":     false,
			"notify_on_complete":   false,
			"notify_on_fail":       true,
		},
	})

	if cfg.Telegram.BotToken != "persist-me" {
		t.Fatalf("expected existing token to remain when bot_token is blank, got %q", cfg.Telegram.BotToken)
	}
	if cfg.Telegram.WebhookSecret != "persist-secret" {
		t.Fatalf("expected existing webhook secret to remain when webhook_secret is blank, got %q", cfg.Telegram.WebhookSecret)
	}
	if cfg.Telegram.Enabled {
		t.Fatal("expected telegram enabled flag to be updated")
	}
	if cfg.Telegram.PollTimeoutSeconds != 45 {
		t.Fatalf("expected poll timeout to update, got %d", cfg.Telegram.PollTimeoutSeconds)
	}
	if len(cfg.Telegram.AllowedChatIDs) != 2 || cfg.Telegram.AllowedChatIDs[0] != 3001 || cfg.Telegram.AllowedChatIDs[1] != 3002 {
		t.Fatalf("expected allowed_chat_ids to update, got %#v", cfg.Telegram.AllowedChatIDs)
	}
	if len(cfg.Telegram.AllowedUserIDs) != 1 || cfg.Telegram.AllowedUserIDs[0] != 4001 {
		t.Fatalf("expected allowed_user_ids to update, got %#v", cfg.Telegram.AllowedUserIDs)
	}
	if cfg.Telegram.NotifyOnAccept {
		t.Fatal("expected notify_on_accept to update")
	}
	if cfg.Telegram.NotifyOnComplete {
		t.Fatal("expected notify_on_complete to update")
	}
	if !cfg.Telegram.NotifyOnFail {
		t.Fatal("expected notify_on_fail to stay enabled")
	}
}

func TestMergeConfigFromMapUpdatesTelegramBotTokenWhenProvided(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Telegram: config.TelegramConfig{
			BotToken:      "old-token",
			WebhookSecret: "old-secret",
		},
	}

	mergeConfigFromMap(cfg, map[string]interface{}{
		"telegram": map[string]interface{}{
			"bot_token":      "new-token",
			"webhook_secret": "new-secret",
		},
	})

	if cfg.Telegram.BotToken != "new-token" {
		t.Fatalf("expected token to update when provided, got %q", cfg.Telegram.BotToken)
	}
	if cfg.Telegram.WebhookSecret != "new-secret" {
		t.Fatalf("expected webhook secret to update when provided, got %q", cfg.Telegram.WebhookSecret)
	}
}

func TestHandleUpdateConfigAppliesTelegramConfigWithoutRestartWhenHotReloadSucceeds(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	cfg, configPath := loadConfigForUpdateHandlerTest(t)
	var appliedConfig *config.Config

	server := &Server{
		config:     cfg,
		configPath: configPath,
		biliClient: bilibili.NewClient(cfg),
		telegramService: &fakeConfigTelegramService{
			applyConfigFn: func(next *config.Config) (bool, error) {
				appliedConfig = next
				return false, nil
			},
		},
	}

	body := []byte(`{
		"telegram": {
			"enabled": true,
			"bot_token": "123:updated-token",
			"mode": "polling",
			"poll_timeout_seconds": 45,
			"allowed_chat_ids": [1001],
			"allowed_user_ids": [],
			"allowed_chat_types": ["private"],
			"max_urls_per_message": 1,
			"notify_on_accept": false,
			"notify_on_complete": true,
			"notify_on_fail": true
		}
	}`)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	server.handleUpdateConfig(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if appliedConfig == nil {
		t.Fatal("expected telegram runtime config to be applied")
	}
	if !server.config.Telegram.Enabled {
		t.Fatal("expected in-memory telegram config to be updated")
	}
	if server.config.Telegram.BotToken != "123:updated-token" {
		t.Fatalf("expected new telegram token to be stored, got %q", server.config.Telegram.BotToken)
	}

	var resp Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected response data map, got %T", resp.Data)
	}

	if restartNeeded, ok := data["restart_needed"].(bool); !ok || restartNeeded {
		t.Fatalf("expected restart_needed=false, got %#v", data["restart_needed"])
	}

	requiresRestart, ok := data["requires_restart"].([]interface{})
	if !ok {
		t.Fatalf("expected requires_restart array, got %T", data["requires_restart"])
	}
	if len(requiresRestart) != 0 {
		t.Fatalf("expected no restart requirements, got %#v", requiresRestart)
	}
}

func TestHandleUpdateConfigRequiresTelegramRestartWhenRuntimeApplyFails(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	cfg, configPath := loadConfigForUpdateHandlerTest(t)

	server := &Server{
		config:     cfg,
		configPath: configPath,
		biliClient: bilibili.NewClient(cfg),
		telegramService: &fakeConfigTelegramService{
			applyConfigFn: func(next *config.Config) (bool, error) {
				return false, assertAnError("apply failed")
			},
		},
	}

	body := []byte(`{
		"telegram": {
			"enabled": true,
			"bot_token": "123:updated-token",
			"mode": "polling",
			"poll_timeout_seconds": 45,
			"allowed_chat_ids": [1001],
			"allowed_user_ids": [],
			"allowed_chat_types": ["private"],
			"max_urls_per_message": 1,
			"notify_on_accept": true,
			"notify_on_complete": true,
			"notify_on_fail": true
		}
	}`)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	server.handleUpdateConfig(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var resp Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected response data map, got %T", resp.Data)
	}

	if restartNeeded, ok := data["restart_needed"].(bool); !ok || !restartNeeded {
		t.Fatalf("expected restart_needed=true, got %#v", data["restart_needed"])
	}

	requiresRestart, ok := data["requires_restart"].([]interface{})
	if !ok {
		t.Fatalf("expected requires_restart array, got %T", data["requires_restart"])
	}
	if len(requiresRestart) != 1 || requiresRestart[0] != "telegram" {
		t.Fatalf("expected telegram restart requirement, got %#v", requiresRestart)
	}
}

type assertAnError string

func (e assertAnError) Error() string {
	return string(e)
}
