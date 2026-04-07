package telegram

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"bili-download/internal/config"
	"bili-download/internal/database/models"
	downloadservice "bili-download/internal/service"
)

type fakeBotAPI struct {
	me        *User
	sendCalls []sendCall
	editCalls []editCall
	sendErr   error
	editErr   error
	sendHook  func()
}

type sendCall struct {
	chatID           int64
	text             string
	replyToMessageID int64
}

type editCall struct {
	chatID    int64
	messageID int64
	text      string
}

type reconnectRuntimeStateStore struct{}

type noopURLDownloadSubmitter struct{}

type reconnectBlockingBotAPI struct {
	mu              sync.Mutex
	getUpdatesCalls int
	callStarted     chan int
	setWebhookCalls []webhookCall
	deleteCalls     []bool
	webhookStarted  chan webhookCall
}

type recordingRuntimeStateStore struct {
	mu         sync.Mutex
	state      *models.TelegramRuntimeState
	savedIDs   []int64
	errorTexts []string
}

type botScopedRecordingRuntimeStateStore struct {
	mu         sync.Mutex
	states     map[string]*models.TelegramRuntimeState
	errorTexts []string
}

type webhookCall struct {
	url    string
	secret string
}

func (f *fakeBotAPI) GetMe(context.Context) (*User, error) {
	return f.me, nil
}

func (f *fakeBotAPI) GetUpdates(context.Context, int64, int) ([]Update, error) {
	return nil, nil
}

func (f *fakeBotAPI) SendMessage(_ context.Context, chatID int64, text string, replyToMessageID int64) (*Message, error) {
	if f.sendHook != nil {
		f.sendHook()
	}
	f.sendCalls = append(f.sendCalls, sendCall{
		chatID:           chatID,
		text:             text,
		replyToMessageID: replyToMessageID,
	})
	if f.sendErr != nil {
		return nil, f.sendErr
	}
	return &Message{MessageID: 9001}, nil
}

func (f *fakeBotAPI) EditMessageText(_ context.Context, chatID int64, messageID int64, text string) (*Message, error) {
	f.editCalls = append(f.editCalls, editCall{
		chatID:    chatID,
		messageID: messageID,
		text:      text,
	})
	if f.editErr != nil {
		return nil, f.editErr
	}
	return &Message{MessageID: messageID}, nil
}

func (f *fakeBotAPI) SetWebhook(context.Context, string, string) error {
	return nil
}

func (f *fakeBotAPI) DeleteWebhook(context.Context, bool) error {
	return nil
}

func (f *reconnectRuntimeStateStore) LoadOrCreate(context.Context, string) (*models.TelegramRuntimeState, error) {
	return &models.TelegramRuntimeState{BotName: "demo-bot"}, nil
}

func (f *reconnectRuntimeStateStore) SaveProgress(context.Context, string, int64, int64, time.Time) error {
	return nil
}

func (f *reconnectRuntimeStateStore) SaveError(context.Context, string, string, time.Time) error {
	return nil
}

func (noopURLDownloadSubmitter) Submit(context.Context, downloadservice.URLDownloadRequest) (*downloadservice.URLDownloadResult, error) {
	return nil, nil
}

func (s *recordingRuntimeStateStore) LoadOrCreate(context.Context, string) (*models.TelegramRuntimeState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == nil {
		s.state = &models.TelegramRuntimeState{BotName: "demo-bot"}
	}
	return s.state, nil
}

func (s *recordingRuntimeStateStore) SaveProgress(_ context.Context, _ string, lastUpdateID int64, processedUpdateID int64, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == nil {
		s.state = &models.TelegramRuntimeState{BotName: "demo-bot"}
	}
	if lastUpdateID < s.state.LastUpdateID {
		lastUpdateID = s.state.LastUpdateID
	}
	s.state.LastUpdateID = lastUpdateID
	s.state.WebhookRecentUpdateIDs = appendWebhookRecentUpdateID(s.state.WebhookRecentUpdateIDs, processedUpdateID)
	s.savedIDs = append(s.savedIDs, lastUpdateID)
	return nil
}

func (s *recordingRuntimeStateStore) SaveError(_ context.Context, _ string, errText string, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errorTexts = append(s.errorTexts, errText)
	return nil
}

func (s *botScopedRecordingRuntimeStateStore) LoadOrCreate(_ context.Context, botName string) (*models.TelegramRuntimeState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.states == nil {
		s.states = make(map[string]*models.TelegramRuntimeState)
	}
	if s.states[botName] == nil {
		s.states[botName] = &models.TelegramRuntimeState{BotName: botName}
	}
	return s.states[botName], nil
}

func (s *botScopedRecordingRuntimeStateStore) SaveProgress(ctx context.Context, botName string, lastUpdateID int64, processedUpdateID int64, _ time.Time) error {
	state, err := s.LoadOrCreate(ctx, botName)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if lastUpdateID < state.LastUpdateID {
		lastUpdateID = state.LastUpdateID
	}
	state.LastUpdateID = lastUpdateID
	state.WebhookRecentUpdateIDs = appendWebhookRecentUpdateID(state.WebhookRecentUpdateIDs, processedUpdateID)
	return nil
}

func (s *botScopedRecordingRuntimeStateStore) SaveError(_ context.Context, _ string, errText string, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errorTexts = append(s.errorTexts, errText)
	return nil
}

func (f *reconnectBlockingBotAPI) GetMe(context.Context) (*User, error) {
	return &User{ID: 1, Username: "demo-bot"}, nil
}

func (f *reconnectBlockingBotAPI) GetUpdates(ctx context.Context, _ int64, _ int) ([]Update, error) {
	f.mu.Lock()
	f.getUpdatesCalls++
	callNumber := f.getUpdatesCalls
	f.mu.Unlock()

	if f.callStarted != nil {
		f.callStarted <- callNumber
	}

	<-ctx.Done()
	return nil, ctx.Err()
}

func (f *reconnectBlockingBotAPI) SendMessage(context.Context, int64, string, int64) (*Message, error) {
	return &Message{MessageID: 1}, nil
}

func (f *reconnectBlockingBotAPI) EditMessageText(context.Context, int64, int64, string) (*Message, error) {
	return &Message{MessageID: 1}, nil
}

func (f *reconnectBlockingBotAPI) SetWebhook(_ context.Context, webhookURL string, secretToken string) error {
	call := webhookCall{url: webhookURL, secret: secretToken}
	f.mu.Lock()
	f.setWebhookCalls = append(f.setWebhookCalls, call)
	f.mu.Unlock()

	if f.webhookStarted != nil {
		f.webhookStarted <- call
	}

	return nil
}

func (f *reconnectBlockingBotAPI) DeleteWebhook(_ context.Context, dropPendingUpdates bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleteCalls = append(f.deleteCalls, dropPendingUpdates)
	return nil
}

func TestBotServiceShouldNotifyStage(t *testing.T) {
	t.Parallel()

	service := &BotService{
		cfg: &config.Config{
			Telegram: config.TelegramConfig{
				NotifyOnAccept:   false,
				NotifyOnComplete: true,
				NotifyOnFail:     false,
			},
		},
	}

	if service.shouldNotifyStage(TelegramRequestStatusQueued) {
		t.Fatal("expected accept notifications to be disabled")
	}
	if !service.shouldNotifyStage(TelegramRequestStatusCompleted) {
		t.Fatal("expected completion notifications to be enabled")
	}
	if service.shouldNotifyStage(TelegramRequestStatusFailed) {
		t.Fatal("expected failure notifications to be disabled")
	}
}

func TestDeliverTerminalReplyFallsBackToNewMessageWhenNoTrackedReplyExists(t *testing.T) {
	t.Parallel()

	client := &fakeBotAPI{me: &User{Username: "mybot"}}
	service := &BotService{client: client}

	delivered := service.deliverTerminalReply(context.Background(), RequestSummary{
		Log: models.TelegramRequestLog{
			ChatID:    1001,
			MessageID: 2002,
			TaskID:    "task-1",
		},
		Title: "demo video",
	}, TelegramRequestStatusCompleted)

	if !delivered {
		t.Fatal("expected completion reply to be delivered")
	}
	if len(client.editCalls) != 0 {
		t.Fatalf("expected no edit attempt, got %d", len(client.editCalls))
	}
	if len(client.sendCalls) != 1 {
		t.Fatalf("expected one send attempt, got %d", len(client.sendCalls))
	}
	if client.sendCalls[0].replyToMessageID != 2002 {
		t.Fatalf("expected reply to original message id 2002, got %d", client.sendCalls[0].replyToMessageID)
	}
}

func TestDeliverTerminalReplyEditsExistingReplyFirst(t *testing.T) {
	t.Parallel()

	replyMessageID := int64(3003)
	client := &fakeBotAPI{}
	service := &BotService{client: client}

	delivered := service.deliverTerminalReply(context.Background(), RequestSummary{
		Log: models.TelegramRequestLog{
			ChatID:         1001,
			MessageID:      2002,
			ReplyMessageID: &replyMessageID,
			TaskID:         "task-1",
		},
		Title: "demo video",
	}, TelegramRequestStatusCompleted)

	if !delivered {
		t.Fatal("expected completion reply to be delivered")
	}
	if len(client.editCalls) != 1 {
		t.Fatalf("expected one edit attempt, got %d", len(client.editCalls))
	}
	if len(client.sendCalls) != 0 {
		t.Fatalf("expected no fallback send, got %d", len(client.sendCalls))
	}
	if client.editCalls[0].messageID != 3003 {
		t.Fatalf("expected edit target 3003, got %d", client.editCalls[0].messageID)
	}
}

func TestDeliverTerminalReplyFallsBackWhenEditFails(t *testing.T) {
	t.Parallel()

	replyMessageID := int64(3003)
	client := &fakeBotAPI{editErr: errors.New("edit failed")}
	service := &BotService{client: client}

	delivered := service.deliverTerminalReply(context.Background(), RequestSummary{
		Log: models.TelegramRequestLog{
			ChatID:         1001,
			MessageID:      2002,
			ReplyMessageID: &replyMessageID,
			TaskID:         "task-1",
		},
		Title:        "demo video",
		ErrorMessage: "network",
	}, TelegramRequestStatusFailed)

	if !delivered {
		t.Fatal("expected failure reply to fall back to a new message")
	}
	if len(client.editCalls) != 1 {
		t.Fatalf("expected one edit attempt, got %d", len(client.editCalls))
	}
	if len(client.sendCalls) != 1 {
		t.Fatalf("expected one fallback send, got %d", len(client.sendCalls))
	}
	if client.sendCalls[0].replyToMessageID != 2002 {
		t.Fatalf("expected fallback reply to target original message id 2002, got %d", client.sendCalls[0].replyToMessageID)
	}
}

func TestBotServiceReconnectRequiresRunningPoller(t *testing.T) {
	t.Parallel()

	service := &BotService{
		cfg: &config.Config{
			Telegram: config.TelegramConfig{
				Enabled: true,
			},
		},
	}

	if err := service.Reconnect(); err == nil {
		t.Fatal("expected reconnect to fail when the service is not running")
	}
}

func TestBotServiceReconnectRestartsPolling(t *testing.T) {
	t.Parallel()

	client := &reconnectBlockingBotAPI{
		callStarted: make(chan int, 4),
	}
	service := &BotService{
		cfg: &config.Config{
			Telegram: config.TelegramConfig{
				Enabled:            true,
				BotToken:           "123:token",
				PollTimeoutSeconds: 30,
				AllowedChatTypes:   []string{"private"},
				MaxURLsPerMessage:  1,
			},
		},
		client:             client,
		runtimeStore:       &reconnectRuntimeStateStore{},
		urlDownloadService: noopURLDownloadSubmitter{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- service.Start(ctx)
	}()

	select {
	case callNumber := <-client.callStarted:
		if callNumber != 1 {
			t.Fatalf("expected first poll call number 1, got %d", callNumber)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the first poll cycle")
	}

	if err := service.Reconnect(); err != nil {
		t.Fatalf("expected reconnect to succeed, got %v", err)
	}

	select {
	case callNumber := <-client.callStarted:
		if callNumber != 2 {
			t.Fatalf("expected second poll call number 2 after reconnect, got %d", callNumber)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the reconnect poll cycle")
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected start to stop cleanly, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for service shutdown")
	}
}

func TestBotServiceApplyConfigStartsPollingAfterEnable(t *testing.T) {
	t.Parallel()

	enabledClient := &reconnectBlockingBotAPI{
		callStarted: make(chan int, 4),
	}
	service := &BotService{
		cfg: &config.Config{
			Telegram: config.TelegramConfig{
				Enabled: false,
			},
		},
		clientFactory: func(cfg config.TelegramConfig, _ config.ProxyConfig) BotAPI {
			if cfg.Enabled && cfg.BotToken == "123:enabled" {
				return enabledClient
			}
			return nil
		},
		runtimeStore:       &reconnectRuntimeStateStore{},
		urlDownloadService: noopURLDownloadSubmitter{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- service.Start(ctx)
	}()

	restartRequired, err := service.ApplyConfig(&config.Config{
		Telegram: config.TelegramConfig{
			Enabled:            true,
			BotToken:           "123:enabled",
			PollTimeoutSeconds: 30,
			AllowedChatTypes:   []string{"private"},
			MaxURLsPerMessage:  1,
		},
	})
	if err != nil {
		t.Fatalf("expected apply config to succeed, got %v", err)
	}
	if restartRequired {
		t.Fatal("expected apply config to hot-reload without restart")
	}

	select {
	case callNumber := <-enabledClient.callStarted:
		if callNumber != 1 {
			t.Fatalf("expected first poll call number 1 after enable, got %d", callNumber)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for polling to start after enabling telegram")
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected start to stop cleanly, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for service shutdown")
	}
}

func TestBotServiceApplyConfigReconnectsRunningPollerWithNewClient(t *testing.T) {
	t.Parallel()

	client1 := &reconnectBlockingBotAPI{
		callStarted: make(chan int, 4),
	}
	client2 := &reconnectBlockingBotAPI{
		callStarted: make(chan int, 4),
	}
	service := &BotService{
		cfg: &config.Config{
			Telegram: config.TelegramConfig{
				Enabled:            true,
				BotToken:           "123:token-a",
				PollTimeoutSeconds: 30,
				AllowedChatTypes:   []string{"private"},
				MaxURLsPerMessage:  1,
			},
		},
		clientFactory: func(cfg config.TelegramConfig, _ config.ProxyConfig) BotAPI {
			switch cfg.BotToken {
			case "123:token-a":
				return client1
			case "123:token-b":
				return client2
			default:
				return nil
			}
		},
		runtimeStore:       &reconnectRuntimeStateStore{},
		urlDownloadService: noopURLDownloadSubmitter{},
	}
	service.client = service.newClient(service.cfg.Telegram, service.cfg.Proxy)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- service.Start(ctx)
	}()

	select {
	case callNumber := <-client1.callStarted:
		if callNumber != 1 {
			t.Fatalf("expected first client poll call number 1, got %d", callNumber)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the initial poll cycle")
	}

	restartRequired, err := service.ApplyConfig(&config.Config{
		Telegram: config.TelegramConfig{
			Enabled:            true,
			BotToken:           "123:token-b",
			PollTimeoutSeconds: 45,
			AllowedChatTypes:   []string{"private"},
			MaxURLsPerMessage:  1,
		},
	})
	if err != nil {
		t.Fatalf("expected apply config to succeed, got %v", err)
	}
	if restartRequired {
		t.Fatal("expected running telegram service to hot-reload without restart")
	}

	select {
	case callNumber := <-client2.callStarted:
		if callNumber != 1 {
			t.Fatalf("expected second client poll call number 1 after hot-reload, got %d", callNumber)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the hot-reload poll cycle")
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected start to stop cleanly, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for service shutdown")
	}
}

func TestBotServiceApplyConfigRegistersWebhookAfterEnable(t *testing.T) {
	t.Parallel()

	webhookClient := &reconnectBlockingBotAPI{
		webhookStarted: make(chan webhookCall, 2),
	}
	service := &BotService{
		cfg: &config.Config{
			Telegram: config.TelegramConfig{
				Enabled: false,
			},
		},
		clientFactory: func(cfg config.TelegramConfig, _ config.ProxyConfig) BotAPI {
			if cfg.Mode == "webhook" && cfg.BotToken == "123:webhook" {
				return webhookClient
			}
			return nil
		},
		runtimeStore:       &reconnectRuntimeStateStore{},
		urlDownloadService: noopURLDownloadSubmitter{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- service.Start(ctx)
	}()

	restartRequired, err := service.ApplyConfig(&config.Config{
		Telegram: config.TelegramConfig{
			Enabled:            true,
			BotToken:           "123:webhook",
			Mode:               "webhook",
			WebhookURL:         "https://example.com/telegram/webhook",
			WebhookSecret:      "secret-123",
			PollTimeoutSeconds: 30,
			AllowedChatTypes:   []string{"private"},
			MaxURLsPerMessage:  1,
			AllowedChatIDs:     []int64{1001},
		},
	})
	if err != nil {
		t.Fatalf("expected apply config to succeed, got %v", err)
	}
	if restartRequired {
		t.Fatal("expected webhook enable to hot-reload without restart")
	}

	select {
	case call := <-webhookClient.webhookStarted:
		if call.url != "https://example.com/telegram/webhook" {
			t.Fatalf("expected webhook URL to be registered, got %q", call.url)
		}
		if call.secret != "secret-123" {
			t.Fatalf("expected webhook secret to be registered, got %q", call.secret)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for webhook registration")
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected start to stop cleanly, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for service shutdown")
	}
	if len(webhookClient.deleteCalls) == 0 {
		t.Fatal("expected webhook registration to be cleaned up on shutdown")
	}
}

func TestBotServiceReconnectKeepsWebhookRegisteredForSameBot(t *testing.T) {
	t.Parallel()

	webhookClient := &reconnectBlockingBotAPI{
		webhookStarted: make(chan webhookCall, 4),
	}
	service := &BotService{
		cfg: &config.Config{
			Telegram: config.TelegramConfig{
				Enabled:            true,
				BotToken:           "123:webhook",
				Mode:               "webhook",
				WebhookURL:         "https://example.com/telegram/webhook",
				WebhookSecret:      "secret-123",
				PollTimeoutSeconds: 30,
				AllowedChatTypes:   []string{"private"},
				MaxURLsPerMessage:  1,
				AllowedChatIDs:     []int64{1001},
			},
		},
		client:             webhookClient,
		runtimeStore:       &reconnectRuntimeStateStore{},
		urlDownloadService: noopURLDownloadSubmitter{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- service.Start(ctx)
	}()

	select {
	case <-webhookClient.webhookStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for initial webhook registration")
	}

	if err := service.Reconnect(); err != nil {
		t.Fatalf("expected reconnect to succeed, got %v", err)
	}

	select {
	case <-webhookClient.webhookStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for webhook re-registration")
	}

	if len(webhookClient.deleteCalls) != 0 {
		t.Fatalf("expected same-bot webhook reconnect to avoid delete gap, got %d deletes", len(webhookClient.deleteCalls))
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected start to stop cleanly, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for service shutdown")
	}
	if len(webhookClient.deleteCalls) != 1 {
		t.Fatalf("expected shutdown to clean up webhook once, got %d deletes", len(webhookClient.deleteCalls))
	}
}

func TestBotServiceHandleWebhookUpdatePersistsProgress(t *testing.T) {
	t.Parallel()

	store := &recordingRuntimeStateStore{}
	service := &BotService{
		cfg: &config.Config{
			Telegram: config.TelegramConfig{
				Enabled:           true,
				Mode:              "webhook",
				BotToken:          "123:webhook",
				WebhookURL:        "https://example.com/telegram/webhook",
				WebhookSecret:     "secret-123",
				AllowedChatTypes:  []string{"private"},
				MaxURLsPerMessage: 1,
				AllowedChatIDs:    []int64{1001},
			},
		},
		client:             &fakeBotAPI{},
		runtimeStore:       store,
		urlDownloadService: noopURLDownloadSubmitter{},
	}
	service.setCurrentBotName("demo-bot")

	err := service.HandleWebhookUpdate(context.Background(), Update{
		UpdateID: 77,
		Message: &Message{
			MessageID: 10,
			Text:      "hello world",
			Chat:      &Chat{ID: 1001, Type: "private"},
			From:      &User{ID: 2002, Username: "demo"},
		},
	})
	if err != nil {
		t.Fatalf("expected webhook update to be handled, got %v", err)
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.savedIDs) != 1 || store.savedIDs[0] != 77 {
		t.Fatalf("expected update progress [77], got %v", store.savedIDs)
	}
}

func TestBotServiceHandleWebhookUpdateDeduplicatesConcurrentDeliveries(t *testing.T) {
	t.Parallel()

	store := &recordingRuntimeStateStore{}
	releaseSend := make(chan struct{})
	client := &fakeBotAPI{
		sendHook: func() {
			<-releaseSend
		},
	}
	service := &BotService{
		cfg: &config.Config{
			Telegram: config.TelegramConfig{
				Enabled:           true,
				Mode:              "webhook",
				BotToken:          "123:webhook",
				WebhookURL:        "https://example.com/telegram/webhook",
				WebhookSecret:     "secret-123",
				AllowedChatTypes:  []string{"private"},
				MaxURLsPerMessage: 1,
				AllowedChatIDs:    []int64{9999},
			},
		},
		client:             client,
		runtimeStore:       store,
		urlDownloadService: noopURLDownloadSubmitter{},
	}
	service.setCurrentBotName("demo-bot")

	update := Update{
		UpdateID: 88,
		Message: &Message{
			MessageID: 10,
			Text:      "https://example.com/video",
			Chat:      &Chat{ID: 1001, Type: "private"},
			From:      &User{ID: 2002, Username: "demo"},
		},
	}

	errCh := make(chan error, 2)
	go func() {
		errCh <- service.HandleWebhookUpdate(context.Background(), update)
	}()
	go func() {
		errCh <- service.HandleWebhookUpdate(context.Background(), update)
	}()

	time.Sleep(100 * time.Millisecond)
	close(releaseSend)

	for i := 0; i < 2; i++ {
		if err := <-errCh; err != nil {
			t.Fatalf("expected webhook handling to succeed, got %v", err)
		}
	}

	if len(client.sendCalls) != 1 {
		t.Fatalf("expected duplicate webhook delivery to be processed once, got %d sends", len(client.sendCalls))
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.savedIDs) != 1 || store.savedIDs[0] != 88 {
		t.Fatalf("expected update progress [88], got %v", store.savedIDs)
	}
}

func TestBotServiceHandleWebhookUpdateSkipsDuplicateUpdateID(t *testing.T) {
	t.Parallel()

	store := &recordingRuntimeStateStore{
		state: &models.TelegramRuntimeState{
			BotName:      "demo-bot",
			LastUpdateID: 77,
		},
	}
	service := &BotService{
		cfg: &config.Config{
			Telegram: config.TelegramConfig{
				Enabled:           true,
				Mode:              "webhook",
				BotToken:          "123:webhook",
				WebhookURL:        "https://example.com/telegram/webhook",
				WebhookSecret:     "secret-123",
				AllowedChatTypes:  []string{"private"},
				MaxURLsPerMessage: 1,
				AllowedChatIDs:    []int64{1001},
			},
		},
		client:             &fakeBotAPI{},
		runtimeStore:       store,
		urlDownloadService: noopURLDownloadSubmitter{},
	}
	service.setCurrentBotName("demo-bot")

	err := service.HandleWebhookUpdate(context.Background(), Update{
		UpdateID: 77,
		Message: &Message{
			MessageID: 10,
			Text:      "hello world",
			Chat:      &Chat{ID: 1001, Type: "private"},
			From:      &User{ID: 2002, Username: "demo"},
		},
	})
	if err != nil {
		t.Fatalf("expected duplicate webhook update to be ignored, got %v", err)
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.savedIDs) != 0 {
		t.Fatalf("expected duplicate update to skip progress save, got %v", store.savedIDs)
	}
	if store.state.LastUpdateID != 77 {
		t.Fatalf("expected last update id to remain 77, got %d", store.state.LastUpdateID)
	}
}

func TestBotServiceHandleWebhookUpdateProcessesOutOfOrderDistinctUpdates(t *testing.T) {
	t.Parallel()

	store := &recordingRuntimeStateStore{}
	client := &fakeBotAPI{}
	service := &BotService{
		cfg: &config.Config{
			Telegram: config.TelegramConfig{
				Enabled:           true,
				Mode:              "webhook",
				BotToken:          "123:webhook",
				WebhookURL:        "https://example.com/telegram/webhook",
				WebhookSecret:     "secret-123",
				AllowedChatTypes:  []string{"private"},
				MaxURLsPerMessage: 1,
				AllowedChatIDs:    []int64{9999},
			},
		},
		client:             client,
		runtimeStore:       store,
		urlDownloadService: noopURLDownloadSubmitter{},
	}
	service.setCurrentBotName("demo-bot")

	first := Update{
		UpdateID: 101,
		Message: &Message{
			MessageID: 10,
			Text:      "hello world",
			Chat:      &Chat{ID: 1001, Type: "private"},
			From:      &User{ID: 2002, Username: "demo"},
		},
	}
	second := Update{
		UpdateID: 100,
		Message: &Message{
			MessageID: 11,
			Text:      "hello again",
			Chat:      &Chat{ID: 1001, Type: "private"},
			From:      &User{ID: 2002, Username: "demo"},
		},
	}

	if err := service.HandleWebhookUpdate(context.Background(), first); err != nil {
		t.Fatalf("expected first webhook update to succeed, got %v", err)
	}
	if err := service.HandleWebhookUpdate(context.Background(), second); err != nil {
		t.Fatalf("expected out-of-order webhook update to succeed, got %v", err)
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	if store.state == nil || store.state.LastUpdateID != 101 {
		t.Fatalf("expected last update id to stay at 101, got %+v", store.state)
	}
	if len(store.savedIDs) != 2 || store.savedIDs[0] != 101 || store.savedIDs[1] != 101 {
		t.Fatalf("expected monotonic progress saves [101 101], got %v", store.savedIDs)
	}
	recentIDs := decodeWebhookRecentUpdateIDs(store.state.WebhookRecentUpdateIDs)
	if len(recentIDs) != 2 || recentIDs[0] != 101 || recentIDs[1] != 100 {
		t.Fatalf("expected persisted recent webhook ids [101 100], got %v", recentIDs)
	}
}

func TestBotServiceHandleWebhookUpdateSkipsPersistedStaleReplayAfterRestart(t *testing.T) {
	t.Parallel()

	store := &recordingRuntimeStateStore{
		state: &models.TelegramRuntimeState{
			BotName:                "demo-bot",
			LastUpdateID:           101,
			WebhookRecentUpdateIDs: `[101,100]`,
		},
	}
	client := &fakeBotAPI{}
	service := &BotService{
		cfg: &config.Config{
			Telegram: config.TelegramConfig{
				Enabled:           true,
				Mode:              "webhook",
				BotToken:          "123:webhook",
				WebhookURL:        "https://example.com/telegram/webhook",
				WebhookSecret:     "secret-123",
				AllowedChatTypes:  []string{"group"},
				MaxURLsPerMessage: 1,
				AllowedChatIDs:    []int64{1001},
			},
		},
		client:             client,
		runtimeStore:       store,
		urlDownloadService: noopURLDownloadSubmitter{},
	}
	service.setCurrentBotName("mybot")

	err := service.HandleWebhookUpdate(context.Background(), Update{
		UpdateID: 100,
		Message: &Message{
			MessageID: 12,
			Text:      "@mybot /status task-123",
			Chat:      &Chat{ID: 9999, Type: "group"},
			From:      &User{ID: 2002, Username: "demo"},
		},
	})
	if err != nil {
		t.Fatalf("expected persisted stale replay to be skipped cleanly, got %v", err)
	}
	if len(client.sendCalls) != 0 {
		t.Fatalf("expected persisted stale replay to skip processing, got %d sends", len(client.sendCalls))
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.savedIDs) != 0 {
		t.Fatalf("expected stale replay to skip progress save, got %v", store.savedIDs)
	}
}

func TestBotServiceHandleWebhookUpdateAllowsDistinctOlderUpdateWithinReplayWindow(t *testing.T) {
	t.Parallel()

	store := &recordingRuntimeStateStore{
		state: &models.TelegramRuntimeState{
			BotName:                "demo-bot",
			LastUpdateID:           recentWebhookUpdateLimit + 200,
			WebhookRecentUpdateIDs: fmt.Sprintf("[%d]", recentWebhookUpdateLimit+200),
		},
	}
	client := &fakeBotAPI{me: &User{Username: "mybot"}}
	service := &BotService{
		cfg: &config.Config{
			Telegram: config.TelegramConfig{
				Enabled:           true,
				Mode:              "webhook",
				BotToken:          "123:webhook",
				WebhookURL:        "https://example.com/telegram/webhook",
				WebhookSecret:     "secret-123",
				AllowedChatTypes:  []string{"group"},
				MaxURLsPerMessage: 1,
				AllowedChatIDs:    []int64{1001},
			},
		},
		client:             client,
		runtimeStore:       store,
		urlDownloadService: noopURLDownloadSubmitter{},
	}
	service.setCurrentBotName("mybot")

	updateID := int64(recentWebhookUpdateLimit + 199)
	err := service.HandleWebhookUpdate(context.Background(), Update{
		UpdateID: updateID,
		Message: &Message{
			MessageID: 13,
			Text:      "@mybot /status task-123",
			Chat:      &Chat{ID: 9999, Type: "group"},
			From:      &User{ID: 2002, Username: "demo"},
		},
	})
	if err != nil {
		t.Fatalf("expected distinct older update within replay window to be processed, got %v", err)
	}
	if len(client.sendCalls) != 1 {
		t.Fatalf("expected replay-window update to reach access handling, got %d sends", len(client.sendCalls))
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.savedIDs) != 1 || store.savedIDs[0] != recentWebhookUpdateLimit+200 {
		t.Fatalf("expected monotonic progress save at current max id, got %v", store.savedIDs)
	}
}

func TestBotServiceHandleWebhookUpdateSkipsStaleReplayOutsideRecentWindow(t *testing.T) {
	t.Parallel()

	store := &recordingRuntimeStateStore{
		state: &models.TelegramRuntimeState{
			BotName:                "demo-bot",
			LastUpdateID:           recentWebhookUpdateLimit + 200,
			WebhookRecentUpdateIDs: fmt.Sprintf("[%d]", recentWebhookUpdateLimit+200),
		},
	}
	client := &fakeBotAPI{me: &User{Username: "mybot"}}
	service := &BotService{
		cfg: &config.Config{
			Telegram: config.TelegramConfig{
				Enabled:           true,
				Mode:              "webhook",
				BotToken:          "123:webhook",
				WebhookURL:        "https://example.com/telegram/webhook",
				WebhookSecret:     "secret-123",
				AllowedChatTypes:  []string{"group"},
				MaxURLsPerMessage: 1,
				AllowedChatIDs:    []int64{1001},
			},
		},
		client:             client,
		runtimeStore:       store,
		urlDownloadService: noopURLDownloadSubmitter{},
	}
	service.setCurrentBotName("mybot")

	err := service.HandleWebhookUpdate(context.Background(), Update{
		UpdateID: 100,
		Message: &Message{
			MessageID: 14,
			Text:      "@mybot /status task-123",
			Chat:      &Chat{ID: 9999, Type: "group"},
			From:      &User{ID: 2002, Username: "demo"},
		},
	})
	if err != nil {
		t.Fatalf("expected stale replay outside recent window to be skipped cleanly, got %v", err)
	}
	if len(client.sendCalls) != 0 {
		t.Fatalf("expected stale replay outside recent window to skip processing, got %d sends", len(client.sendCalls))
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.savedIDs) != 0 {
		t.Fatalf("expected stale replay outside recent window to skip progress save, got %v", store.savedIDs)
	}
}

func TestBotServiceApplyConfigResetsWebhookDedupStateForNewBot(t *testing.T) {
	t.Parallel()

	store := &botScopedRecordingRuntimeStateStore{}
	clientA := &fakeBotAPI{me: &User{Username: "bot-a"}}
	clientB := &fakeBotAPI{me: &User{Username: "bot-b"}}
	service := &BotService{
		cfg: &config.Config{
			Telegram: config.TelegramConfig{
				Enabled:           true,
				Mode:              "webhook",
				BotToken:          "123:bot-a",
				WebhookURL:        "https://example.com/telegram/webhook",
				WebhookSecret:     "secret-123",
				AllowedChatTypes:  []string{"private"},
				MaxURLsPerMessage: 1,
				AllowedChatIDs:    []int64{9999},
			},
		},
		client: clientA,
		clientFactory: func(cfg config.TelegramConfig, _ config.ProxyConfig) BotAPI {
			switch cfg.BotToken {
			case "123:bot-a":
				return clientA
			case "123:bot-b":
				return clientB
			default:
				return nil
			}
		},
		runtimeStore:       store,
		urlDownloadService: noopURLDownloadSubmitter{},
	}

	firstUpdate := Update{
		UpdateID: 42,
		Message: &Message{
			MessageID: 10,
			Text:      "hello from bot a",
			Chat:      &Chat{ID: 1001, Type: "private"},
			From:      &User{ID: 2002, Username: "demo"},
		},
	}
	if err := service.HandleWebhookUpdate(context.Background(), firstUpdate); err != nil {
		t.Fatalf("expected first bot update to succeed, got %v", err)
	}

	restartRequired, err := service.ApplyConfig(&config.Config{
		Telegram: config.TelegramConfig{
			Enabled:           true,
			Mode:              "webhook",
			BotToken:          "123:bot-b",
			WebhookURL:        "https://example.com/telegram/webhook",
			WebhookSecret:     "secret-456",
			AllowedChatTypes:  []string{"private"},
			MaxURLsPerMessage: 1,
			AllowedChatIDs:    []int64{9999},
		},
	})
	if err != nil {
		t.Fatalf("expected apply config to succeed, got %v", err)
	}
	if restartRequired {
		t.Fatal("expected webhook config apply to hot-reload without restart")
	}

	secondUpdate := Update{
		UpdateID: 42,
		Message: &Message{
			MessageID: 11,
			Text:      "hello from bot b",
			Chat:      &Chat{ID: 1001, Type: "private"},
			From:      &User{ID: 2002, Username: "demo"},
		},
	}
	if err := service.HandleWebhookUpdate(context.Background(), secondUpdate); err != nil {
		t.Fatalf("expected new bot update with reused id to succeed, got %v", err)
	}

	if service.botName() != "bot-b" {
		t.Fatalf("expected bot name to refresh to bot-b, got %q", service.botName())
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	if store.states["bot-a"] == nil || store.states["bot-a"].LastUpdateID != 42 {
		t.Fatalf("expected bot-a runtime state to persist update 42, got %+v", store.states["bot-a"])
	}
	if store.states["bot-b"] == nil || store.states["bot-b"].LastUpdateID != 42 {
		t.Fatalf("expected bot-b runtime state to accept reused update 42 after bot switch, got %+v", store.states["bot-b"])
	}
}

func TestBotServiceHandleUpdateIgnoresUnmentionedGroupMessageBeforeAccess(t *testing.T) {
	t.Parallel()

	client := &fakeBotAPI{}
	service := &BotService{
		cfg: &config.Config{
			Telegram: config.TelegramConfig{
				Enabled:           true,
				Mode:              "polling",
				BotToken:          "123:token",
				AllowedChatTypes:  []string{"private"},
				MaxURLsPerMessage: 1,
				AllowedChatIDs:    []int64{1001},
			},
		},
		client: client,
	}
	service.setCurrentBotName("mybot")

	err := service.handleUpdate(context.Background(), Update{
		UpdateID: 55,
		Message: &Message{
			MessageID: 10,
			Text:      "https://example.com/video",
			Chat:      &Chat{ID: 9999, Type: "group"},
			From:      &User{ID: 2002, Username: "demo"},
		},
	})
	if err != nil {
		t.Fatalf("expected unmentioned group message to be ignored, got %v", err)
	}
	if len(client.sendCalls) != 0 {
		t.Fatalf("expected no access-denied reply for unmentioned group message, got %d sends", len(client.sendCalls))
	}
}

func TestBotServiceHandleUpdateRejectsMentionedUnauthorizedGroupMessage(t *testing.T) {
	t.Parallel()

	client := &fakeBotAPI{}
	service := &BotService{
		cfg: &config.Config{
			Telegram: config.TelegramConfig{
				Enabled:           true,
				Mode:              "polling",
				BotToken:          "123:token",
				AllowedChatTypes:  []string{"group"},
				MaxURLsPerMessage: 1,
				AllowedChatIDs:    []int64{1001},
			},
		},
		client: client,
	}
	service.setCurrentBotName("mybot")

	err := service.handleUpdate(context.Background(), Update{
		UpdateID: 56,
		Message: &Message{
			MessageID: 11,
			Text:      "@mybot /status task-123",
			Chat:      &Chat{ID: 9999, Type: "group"},
			From:      &User{ID: 2002, Username: "demo"},
		},
	})
	if err != nil {
		t.Fatalf("expected unauthorized mentioned group message to return nil after reply, got %v", err)
	}
	if len(client.sendCalls) != 1 {
		t.Fatalf("expected one access-denied reply, got %d sends", len(client.sendCalls))
	}
	if client.sendCalls[0].text != "Access denied." {
		t.Fatalf("expected access denied reply, got %q", client.sendCalls[0].text)
	}
}
