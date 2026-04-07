package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bili-download/internal/config"
	"bili-download/internal/telegram"

	"github.com/gin-gonic/gin"
)

type fakeTelegramAPI struct {
	sendMessageFn func(ctx context.Context, chatID int64, text string, replyToMessageID int64) (*telegram.Message, error)
}

type fakeTelegramService struct {
	isRunning             bool
	reconnectFn           func() error
	applyConfigFn         func(*config.Config) (bool, error)
	handleWebhookUpdateFn func(context.Context, telegram.Update) error
}

func (f *fakeTelegramAPI) GetMe(context.Context) (*telegram.User, error) {
	return nil, nil
}

func (f *fakeTelegramAPI) GetUpdates(context.Context, int64, int) ([]telegram.Update, error) {
	return nil, nil
}

func (f *fakeTelegramAPI) SendMessage(ctx context.Context, chatID int64, text string, replyToMessageID int64) (*telegram.Message, error) {
	if f.sendMessageFn != nil {
		return f.sendMessageFn(ctx, chatID, text, replyToMessageID)
	}
	return &telegram.Message{MessageID: 1}, nil
}

func (f *fakeTelegramAPI) EditMessageText(context.Context, int64, int64, string) (*telegram.Message, error) {
	return nil, nil
}

func (f *fakeTelegramAPI) SetWebhook(context.Context, string, string) error {
	return nil
}

func (f *fakeTelegramAPI) DeleteWebhook(context.Context, bool) error {
	return nil
}

func (f *fakeTelegramService) IsRunning() bool {
	return f.isRunning
}

func (f *fakeTelegramService) Reconnect() error {
	if f.reconnectFn != nil {
		return f.reconnectFn()
	}
	return nil
}

func (f *fakeTelegramService) ApplyConfig(cfg *config.Config) (bool, error) {
	if f.applyConfigFn != nil {
		return f.applyConfigFn(cfg)
	}
	return false, nil
}

func (f *fakeTelegramService) HandleWebhookUpdate(ctx context.Context, update telegram.Update) error {
	if f.handleWebhookUpdateFn != nil {
		return f.handleWebhookUpdateFn(ctx, update)
	}
	return nil
}

func TestHandleTelegramTestSend(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	fakeClient := &fakeTelegramAPI{}
	var gotChatID int64
	var gotText string
	var gotReplyTo int64
	fakeClient.sendMessageFn = func(_ context.Context, chatID int64, text string, replyToMessageID int64) (*telegram.Message, error) {
		gotChatID = chatID
		gotText = text
		gotReplyTo = replyToMessageID
		return &telegram.Message{MessageID: 99}, nil
	}

	server := &Server{
		config: &config.Config{
			Telegram: config.TelegramConfig{
				Enabled:            true,
				BotToken:           "123:token",
				PollTimeoutSeconds: 30,
			},
		},
		telegramClientFactory: func(cfg config.TelegramConfig, proxyCfg config.ProxyConfig) telegram.BotAPI {
			if cfg.BotToken != "123:token" {
				t.Fatalf("expected bot token to flow into client factory, got %q", cfg.BotToken)
			}
			if proxyCfg.Enabled {
				t.Fatal("expected proxy config to remain disabled in this test")
			}
			return fakeClient
		},
	}

	body, err := json.Marshal(map[string]interface{}{
		"chat_id": 1001,
		"message": "phase5 ping",
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/telegram/test-send", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	server.handleTelegramTestSend(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if gotChatID != 1001 {
		t.Fatalf("expected chat_id 1001, got %d", gotChatID)
	}
	if gotText != "phase5 ping" {
		t.Fatalf("expected message text to pass through, got %q", gotText)
	}
	if gotReplyTo != 0 {
		t.Fatalf("expected test send not to reply to a message, got %d", gotReplyTo)
	}

	var resp Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected api success code 0, got %d", resp.Code)
	}
}

func TestHandleTelegramTestSendAllowsNegativeGroupChatID(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	fakeClient := &fakeTelegramAPI{}
	var gotChatID int64
	fakeClient.sendMessageFn = func(_ context.Context, chatID int64, text string, replyToMessageID int64) (*telegram.Message, error) {
		gotChatID = chatID
		return &telegram.Message{MessageID: 99}, nil
	}

	server := &Server{
		config: &config.Config{
			Telegram: config.TelegramConfig{
				Enabled:            true,
				BotToken:           "123:token",
				PollTimeoutSeconds: 30,
			},
		},
		telegramClientFactory: func(config.TelegramConfig, config.ProxyConfig) telegram.BotAPI {
			return fakeClient
		},
	}

	body, err := json.Marshal(map[string]interface{}{
		"chat_id": -1001234567890,
		"message": "phase5 group ping",
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/telegram/test-send", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	server.handleTelegramTestSend(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if gotChatID != -1001234567890 {
		t.Fatalf("expected negative group chat_id to pass through, got %d", gotChatID)
	}
}

func TestHandleTelegramTestSendRequiresChatID(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	server := &Server{
		config: &config.Config{
			Telegram: config.TelegramConfig{
				BotToken: "123:token",
			},
		},
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/telegram/test-send", bytes.NewBufferString(`{"message":"ping"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	server.handleTelegramTestSend(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}

func TestHandleTelegramTestSendRequiresSavedBotToken(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	server := &Server{
		config: &config.Config{},
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/telegram/test-send", bytes.NewBufferString(`{"chat_id":1001}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	server.handleTelegramTestSend(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}

func TestHandleTelegramReconnect(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	called := false
	server := &Server{
		telegramService: &fakeTelegramService{
			isRunning: true,
			reconnectFn: func() error {
				called = true
				return nil
			},
		},
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/telegram/reconnect", nil)

	server.handleTelegramReconnect(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if !called {
		t.Fatal("expected reconnect to be requested")
	}
}

func TestHandleTelegramReconnectRequiresAttachedService(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	server := &Server{}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/telegram/reconnect", nil)

	server.handleTelegramReconnect(ctx)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, recorder.Code)
	}
}

func TestHandleTelegramWebhook(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	called := false
	var gotUpdate telegram.Update
	server := &Server{
		config: &config.Config{
			Telegram: config.TelegramConfig{
				Enabled:       true,
				Mode:          "webhook",
				WebhookSecret: "secret-123",
			},
		},
		telegramService: &fakeTelegramService{
			handleWebhookUpdateFn: func(_ context.Context, update telegram.Update) error {
				called = true
				gotUpdate = update
				return nil
			},
		},
	}

	body := `{"update_id": 42, "message": {"message_id": 7, "text": "https://example.com", "chat": {"id": 1001, "type": "private"}, "from": {"id": 2002, "username": "demo"}}}`
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/telegram/webhook", bytes.NewBufferString(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Request.Header.Set("X-Telegram-Bot-Api-Secret-Token", "secret-123")

	server.handleTelegramWebhook(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if !called {
		t.Fatal("expected webhook update to be forwarded")
	}
	if gotUpdate.UpdateID != 42 {
		t.Fatalf("expected update id 42, got %d", gotUpdate.UpdateID)
	}
}

func TestHandleTelegramWebhookRejectsInvalidSecret(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	server := &Server{
		config: &config.Config{
			Telegram: config.TelegramConfig{
				Enabled:       true,
				Mode:          "webhook",
				WebhookSecret: "secret-123",
			},
		},
		telegramService: &fakeTelegramService{},
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/telegram/webhook", bytes.NewBufferString(`{"update_id":1}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Request.Header.Set("X-Telegram-Bot-Api-Secret-Token", "wrong")

	server.handleTelegramWebhook(ctx)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}
