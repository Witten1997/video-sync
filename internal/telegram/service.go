package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"bili-download/internal/config"
	"bili-download/internal/database/models"
	downloadservice "bili-download/internal/service"
	"bili-download/internal/utils"

	"gorm.io/gorm"
)

const defaultBotName = "default"
const recentWebhookUpdateLimit = 1024

var errReconnectRequested = errors.New("telegram reconnect requested")

type BotService struct {
	cfg                *config.Config
	client             BotAPI
	clientFactory      func(config.TelegramConfig, config.ProxyConfig) BotAPI
	runtimeStore       RuntimeStateStore
	requestStore       *RequestStore
	urlDownloadService downloadservice.URLDownloadSubmitter

	mu                 sync.RWMutex
	webhookMu          sync.Mutex
	currentBotName     string
	webhookUpdateIDs   map[int64]struct{}
	webhookUpdateOrder []int64
	running            bool
	pollerCancel       context.CancelFunc
	reconnectRequested bool
}

func NewBotService(cfg *config.Config, db *gorm.DB, urlDownloadService downloadservice.URLDownloadSubmitter) *BotService {
	service := &BotService{
		cfg: cfg,
		clientFactory: func(cfg config.TelegramConfig, proxyCfg config.ProxyConfig) BotAPI {
			return NewClient(cfg.BotToken, cfg.PollTimeoutSeconds, proxyCfg)
		},
		runtimeStore:       newRuntimeStateStore(db),
		requestStore:       NewRequestStore(db),
		urlDownloadService: urlDownloadService,
		webhookUpdateIDs:   make(map[int64]struct{}),
	}

	if cfg != nil {
		service.client = service.newClient(cfg.Telegram, cfg.Proxy)
	}

	return service
}

func (s *BotService) Start(ctx context.Context) error {
	if s.urlDownloadService == nil {
		return fmt.Errorf("telegram url download service is required")
	}

	if s.requestStore != nil && s.requestStore.db != nil {
		go s.runNotifier(ctx)
	}

	for {
		if ctx.Err() != nil {
			s.stopActivePoller(false)
			return nil
		}

		telegramCfg, client := s.snapshotTelegramRuntime()
		if !telegramCfg.Enabled {
			s.stopActivePoller(false)
			if !waitOrDone(ctx, time.Second) {
				return nil
			}
			continue
		}
		if client == nil {
			utils.Warn("telegram client is not configured")
			if !waitOrDone(ctx, 3*time.Second) {
				return nil
			}
			continue
		}

		botName, err := s.resolveBotName(ctx, client)
		if err != nil {
			utils.Error("telegram getMe failed: %v", err)
			if !waitOrDone(ctx, 3*time.Second) {
				return nil
			}
			continue
		}
		s.setCurrentBotName(botName)

		var runtimeErr error
		switch telegramCfg.Mode {
		case "webhook":
			runtimeErr = s.runWebhookCycle(ctx, botName, client, telegramCfg)
		default:
			runtimeErr = s.runPollerCycle(ctx, botName, client, telegramCfg.PollTimeoutSeconds)
		}
		if runtimeErr != nil {
			if errors.Is(runtimeErr, errReconnectRequested) {
				utils.Info("telegram reconnect requested, restarting runtime")
				continue
			}
			if ctx.Err() != nil {
				return nil
			}

			utils.Error("telegram runtime stopped: %v", runtimeErr)
			if !waitOrDone(ctx, 3*time.Second) {
				return nil
			}
			continue
		}

		continue
	}
}

func (s *BotService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

func (s *BotService) ApplyConfig(cfg *config.Config) (bool, error) {
	if cfg == nil {
		return true, errors.New("telegram config is not loaded")
	}

	nextCfg := cloneConfig(cfg)
	nextClient := s.newClient(nextCfg.Telegram, nextCfg.Proxy)

	resetWebhookState := false
	s.mu.Lock()
	currentToken := ""
	if s.cfg != nil {
		currentToken = strings.TrimSpace(s.cfg.Telegram.BotToken)
	}
	s.cfg = nextCfg
	s.client = nextClient

	cancel := s.pollerCancel
	if nextCfg.Telegram.Enabled && cancel != nil {
		s.reconnectRequested = true
	} else if !nextCfg.Telegram.Enabled {
		s.reconnectRequested = false
	}
	nextToken := strings.TrimSpace(nextCfg.Telegram.BotToken)
	resetWebhookState = currentToken != nextToken || !nextCfg.Telegram.Enabled || nextCfg.Telegram.Mode != "webhook"
	s.mu.Unlock()

	if resetWebhookState {
		s.resetWebhookRuntimeState()
	}
	if cancel != nil {
		cancel()
	}

	return false, nil
}

func (s *BotService) HandleWebhookUpdate(ctx context.Context, update Update) error {
	s.webhookMu.Lock()
	defer s.webhookMu.Unlock()

	telegramCfg := s.telegramConfig()
	if !telegramCfg.Enabled {
		return errors.New("telegram service is disabled")
	}
	if telegramCfg.Mode != "webhook" {
		return errors.New("telegram service is not in webhook mode")
	}

	client := s.currentClient()
	if client == nil {
		return errors.New("telegram client is not configured")
	}

	botName := s.botName()
	if botName == "" {
		resolvedBotName, err := s.resolveBotName(ctx, client)
		if err != nil {
			return err
		}
		botName = resolvedBotName
		s.setCurrentBotName(botName)
	}

	if update.UpdateID > 0 {
		state, err := s.runtimeStore.LoadOrCreate(ctx, botName)
		if err != nil {
			return err
		}
		if shouldSkipWebhookUpdate(state, update.UpdateID) || s.hasRecentWebhookUpdate(update.UpdateID) {
			return nil
		}
	}

	if err := s.handleUpdate(ctx, update); err != nil {
		_ = s.runtimeStore.SaveError(ctx, botName, err.Error(), time.Now())
		return err
	}

	if update.UpdateID > 0 {
		s.rememberWebhookUpdate(update.UpdateID)
		if err := s.runtimeStore.SaveProgress(ctx, botName, update.UpdateID, update.UpdateID, time.Now()); err != nil {
			_ = s.runtimeStore.SaveError(ctx, botName, err.Error(), time.Now())
			return err
		}
	}

	return nil
}

func (s *BotService) Reconnect() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cfg == nil || !s.cfg.Telegram.Enabled {
		return errors.New("telegram service is disabled")
	}
	if !s.running {
		return errors.New("telegram service is not running")
	}
	if s.pollerCancel == nil {
		return errors.New("telegram poller is not active")
	}

	s.reconnectRequested = true
	s.pollerCancel()
	return nil
}

func (s *BotService) handleUpdate(ctx context.Context, update Update) error {
	message := update.Message
	if message == nil || message.Chat == nil || message.From == nil || message.Text == "" {
		return nil
	}

	result := ParseMessageForChat(message.Text, s.telegramConfig().MaxURLsPerMessage, message.Chat.Type, s.botName())
	if result.Kind == ParseResultKindIgnore {
		return nil
	}

	telegramCfg := s.telegramConfig()
	accessErr := CheckAccess(AccessConfig{
		AllowedChatTypes: telegramCfg.AllowedChatTypes,
		AllowedChatIDs:   telegramCfg.AllowedChatIDs,
		AllowedUserIDs:   telegramCfg.AllowedUserIDs,
	}, message.Chat.ID, message.From.ID, message.Chat.Type)
	if accessErr != nil {
		s.sendReply(ctx, message.Chat.ID, "Access denied.", message.MessageID)
		return nil
	}

	switch result.Kind {
	case ParseResultKindIgnore:
		return nil
	case ParseResultKindReject:
		_, _ = s.sendReply(ctx, message.Chat.ID, result.ReplyText, message.MessageID)
		return nil
	case ParseResultKindStatus:
		return s.handleStatusQuery(ctx, message.Chat.ID, message.From.ID, message.MessageID, result.TaskID)
	case ParseResultKindSubmit:
		return s.handleSubmission(ctx, update, message, result.URL)
	default:
		return nil
	}
}

func (s *BotService) resolveBotName(ctx context.Context, client BotAPI) (string, error) {
	if client == nil {
		return "", errors.New("telegram client is not configured")
	}

	me, err := client.GetMe(ctx)
	if err != nil {
		return "", err
	}

	if me == nil {
		return defaultBotName, nil
	}
	if me.Username != "" {
		return me.Username, nil
	}
	if me.ID > 0 {
		return strconv.FormatInt(me.ID, 10), nil
	}
	return defaultBotName, nil
}

func (s *BotService) sendReply(ctx context.Context, chatID int64, text string, replyToMessageID int64) (*Message, error) {
	client := s.currentClient()
	if client == nil {
		return nil, errors.New("telegram client is not configured")
	}

	msg, err := client.SendMessage(ctx, chatID, text, replyToMessageID)
	if err != nil {
		utils.Warn("telegram sendMessage failed: %v", err)
		return nil, err
	}
	return msg, nil
}

func (s *BotService) runPollerCycle(ctx context.Context, botName string, client BotAPI, pollTimeoutSeconds int) error {
	pollerCtx, pollerCancel := context.WithCancel(ctx)
	s.beginPollerCycle(pollerCancel)
	defer pollerCancel()

	if err := client.DeleteWebhook(pollerCtx, false); err != nil {
		_ = s.runtimeStore.SaveError(pollerCtx, botName, err.Error(), time.Now())
		reconnectRequested := s.endPollerCycle()
		if ctx.Err() != nil {
			return nil
		}
		if reconnectRequested {
			return errReconnectRequested
		}
		return err
	}

	poller := NewPoller(botName, client, s.runtimeStore, pollTimeoutSeconds, s.handleUpdate)
	err := poller.Run(pollerCtx)
	reconnectRequested := s.endPollerCycle()

	if ctx.Err() != nil {
		return nil
	}
	if reconnectRequested {
		return errReconnectRequested
	}
	return err
}

func (s *BotService) runWebhookCycle(ctx context.Context, botName string, client BotAPI, telegramCfg config.TelegramConfig) error {
	runtimeCtx, runtimeCancel := context.WithCancel(ctx)
	s.beginPollerCycle(runtimeCancel)
	defer runtimeCancel()

	if err := client.SetWebhook(runtimeCtx, telegramCfg.WebhookURL, telegramCfg.WebhookSecret); err != nil {
		_ = s.runtimeStore.SaveError(runtimeCtx, botName, err.Error(), time.Now())
		reconnectRequested := s.endPollerCycle()
		if ctx.Err() != nil {
			return nil
		}
		if reconnectRequested {
			return errReconnectRequested
		}
		return err
	}

	<-runtimeCtx.Done()

	if s.shouldCleanupWebhookRegistration(ctx, telegramCfg) {
		s.cleanupWebhookRegistration(client, botName)
	}

	reconnectRequested := s.endPollerCycle()
	if ctx.Err() != nil {
		return nil
	}
	if reconnectRequested {
		return errReconnectRequested
	}
	return nil
}

func (s *BotService) beginPollerCycle(cancel context.CancelFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.running = true
	s.pollerCancel = cancel
}

func (s *BotService) endPollerCycle() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.running = false
	s.pollerCancel = nil
	reconnectRequested := s.reconnectRequested
	s.reconnectRequested = false
	return reconnectRequested
}

func (s *BotService) stopActivePoller(requestReconnect bool) {
	s.mu.Lock()
	cancel := s.pollerCancel
	if requestReconnect && cancel != nil {
		s.reconnectRequested = true
	} else if !requestReconnect {
		s.reconnectRequested = false
	}
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
}

func waitOrDone(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (s *BotService) handleSubmission(ctx context.Context, update Update, message *Message, rawURL string) error {
	requestLog, created, err := s.requestStore.EnsurePending(ctx, RequestLogInput{
		UpdateID:  update.UpdateID,
		ChatID:    message.Chat.ID,
		MessageID: message.MessageID,
		UserID:    message.From.ID,
		RawText:   message.Text,
		RawURL:    rawURL,
	})
	if err != nil {
		return err
	}

	if !created && requestLog.TaskID != "" {
		if requestLog.ReplyMessageID == nil && s.shouldNotifyStage(TelegramRequestStatusQueued) {
			reply := BuildStatusReply(StatusReplyInput{
				Stage:  TelegramRequestStatusQueued,
				TaskID: requestLog.TaskID,
				Title:  strings.TrimSpace(requestLog.RawURL),
			})
			msg, sendErr := s.sendReply(ctx, message.Chat.ID, reply.Text, message.MessageID)
			if sendErr == nil && msg != nil {
				_ = s.requestStore.MarkReplySent(ctx, requestLog.ID, msg.MessageID)
			}
		}
		return nil
	}

	submitResult, err := s.urlDownloadService.Submit(ctx, downloadservice.URLDownloadRequest{
		URL:           rawURL,
		Channel:       "telegram",
		Requester:     strconv.FormatInt(message.From.ID, 10),
		CorrelationID: fmt.Sprintf("telegram:%d", update.UpdateID),
	})
	if err != nil {
		var downloadErr *downloadservice.URLDownloadError
		if errors.As(err, &downloadErr) && downloadErr.Type == downloadservice.URLDownloadErrorTypeValidation {
			_ = s.requestStore.MarkFailed(ctx, requestLog.ID, downloadErr.Error())
			_, _ = s.sendReply(ctx, message.Chat.ID, "Submit failed: "+downloadErr.Error(), message.MessageID)
			return nil
		}
		return err
	}

	if err := s.requestStore.MarkQueued(ctx, requestLog.ID, submitResult.VideoID, submitResult.RecordID, submitResult.TaskID); err != nil {
		return err
	}

	if s.shouldNotifyStage(TelegramRequestStatusQueued) {
		reply := BuildStatusReply(StatusReplyInput{
			Stage:    TelegramRequestStatusQueued,
			TaskID:   submitResult.TaskID,
			Title:    submitResult.Title,
			RecordID: submitResult.RecordID,
		})
		msg, sendErr := s.sendReply(ctx, message.Chat.ID, reply.Text, message.MessageID)
		if sendErr != nil || msg == nil {
			return sendErr
		}
		if err := s.requestStore.MarkReplySent(ctx, requestLog.ID, msg.MessageID); err != nil {
			return err
		}
	}

	return nil
}

func (s *BotService) handleStatusQuery(ctx context.Context, chatID int64, userID int64, replyToMessageID int64, taskID string) error {
	text, err := s.buildStatusResponse(ctx, chatID, userID, taskID)
	if err != nil {
		return err
	}

	_, _ = s.sendReply(ctx, chatID, text, replyToMessageID)
	return nil
}

func (s *BotService) buildStatusResponse(ctx context.Context, chatID int64, userID int64, taskID string) (string, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID != "" {
		summary, err := s.requestStore.FindByTaskID(ctx, chatID, userID, taskID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return "No matching task found.", nil
			}
			return "", err
		}

		return formatStatusSummary(*summary), nil
	}

	items, err := s.requestStore.ListRecentByUser(ctx, chatID, userID, 5)
	if err != nil {
		return "", err
	}
	if len(items) == 0 {
		return "No recent requests.", nil
	}

	lines := []string{"Recent requests:"}
	for _, item := range items {
		lines = append(lines, formatStatusSummary(item))
	}
	return strings.Join(lines, "\n\n"), nil
}

func formatStatusSummary(item RequestSummary) string {
	status := item.Log.Status
	if item.RecordStatus != "" {
		status = item.RecordStatus
	}

	lines := []string{"Status: " + status}
	if item.Title != "" {
		lines = append(lines, "Title: "+item.Title)
	}
	if item.Log.TaskID != "" {
		lines = append(lines, "Task ID: "+item.Log.TaskID)
	}
	if item.Log.RecordID != nil {
		lines = append(lines, "Record ID: "+fmt.Sprint(*item.Log.RecordID))
	}
	if item.ErrorMessage != "" {
		lines = append(lines, "Reason: "+item.ErrorMessage)
	}

	return strings.Join(lines, "\n")
}

func (s *BotService) runNotifier(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		if ctx.Err() != nil {
			return
		}

		if err := s.reconcileNotifications(ctx); err != nil && ctx.Err() == nil {
			utils.Warn("telegram notifier reconcile failed: %v", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *BotService) reconcileNotifications(ctx context.Context) error {
	if s.requestStore == nil {
		return nil
	}
	if s.currentClient() == nil || !s.telegramConfig().Enabled {
		return nil
	}

	items, err := s.requestStore.ListNotificationCandidates(ctx, 20)
	if err != nil {
		return err
	}

	for _, item := range items {
		if item.Log.ReplyMessageID == nil {
			continue
		}

		stage := ""
		switch item.RecordStatus {
		case "completed":
			stage = TelegramRequestStatusCompleted
		case "failed":
			stage = TelegramRequestStatusFailed
		default:
			continue
		}

		if !s.shouldNotifyStage(stage) {
			if err := s.markNotificationDelivered(ctx, item.Log.ID, stage, item.ErrorMessage); err != nil {
				return err
			}
			continue
		}

		if !s.deliverTerminalReply(ctx, item, stage) {
			continue
		}

		if err := s.markNotificationDelivered(ctx, item.Log.ID, stage, item.ErrorMessage); err != nil {
			return err
		}
	}

	return nil
}

func (s *BotService) shouldNotifyStage(stage string) bool {
	telegramCfg := s.telegramConfig()

	switch stage {
	case TelegramRequestStatusQueued:
		return telegramCfg.NotifyOnAccept
	case TelegramRequestStatusCompleted:
		return telegramCfg.NotifyOnComplete
	case TelegramRequestStatusFailed:
		return telegramCfg.NotifyOnFail
	default:
		return false
	}
}

func (s *BotService) deliverTerminalReply(ctx context.Context, item RequestSummary, stage string) bool {
	recordID := uint(0)
	if item.Log.RecordID != nil {
		recordID = *item.Log.RecordID
	}

	replyMessageID := int64(0)
	if item.Log.ReplyMessageID != nil {
		replyMessageID = *item.Log.ReplyMessageID
	}

	reply := BuildStatusReply(StatusReplyInput{
		Stage:        stage,
		Title:        item.Title,
		TaskID:       item.Log.TaskID,
		RecordID:     recordID,
		MessageID:    replyMessageID,
		ErrorMessage: item.ErrorMessage,
	})

	if replyMessageID > 0 {
		client := s.currentClient()
		if client != nil {
			if _, err := client.EditMessageText(ctx, item.Log.ChatID, reply.EditMessageID, reply.Text); err == nil {
				return true
			}
		}
	}

	_, err := s.sendReply(ctx, item.Log.ChatID, reply.Text, item.Log.MessageID)
	return err == nil
}

func (s *BotService) snapshotTelegramRuntime() (config.TelegramConfig, BotAPI) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var telegramCfg config.TelegramConfig
	if s.cfg != nil {
		telegramCfg = s.cfg.Telegram
	}

	return telegramCfg, s.client
}

func (s *BotService) telegramConfig() config.TelegramConfig {
	telegramCfg, _ := s.snapshotTelegramRuntime()
	return telegramCfg
}

func (s *BotService) currentClient() BotAPI {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.client
}

func (s *BotService) botName() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentBotName
}

func (s *BotService) setCurrentBotName(botName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentBotName = botName
}

func (s *BotService) cleanupWebhookRegistration(client BotAPI, botName string) {
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cleanupCancel()

	if err := client.DeleteWebhook(cleanupCtx, false); err != nil {
		utils.Warn("telegram deleteWebhook failed during runtime switch: %v", err)
		_ = s.runtimeStore.SaveError(cleanupCtx, botName, err.Error(), time.Now())
	}
}

func (s *BotService) shouldCleanupWebhookRegistration(parentCtx context.Context, currentCfg config.TelegramConfig) bool {
	if parentCtx.Err() != nil {
		return true
	}

	nextCfg := s.telegramConfig()
	if !nextCfg.Enabled || nextCfg.Mode != "webhook" {
		return true
	}

	return strings.TrimSpace(nextCfg.BotToken) != strings.TrimSpace(currentCfg.BotToken)
}

func (s *BotService) hasRecentWebhookUpdate(updateID int64) bool {
	if s.webhookUpdateIDs == nil {
		return false
	}

	_, ok := s.webhookUpdateIDs[updateID]
	return ok
}

func (s *BotService) rememberWebhookUpdate(updateID int64) {
	if updateID <= 0 {
		return
	}

	if s.webhookUpdateIDs == nil {
		s.webhookUpdateIDs = make(map[int64]struct{})
	}
	if _, ok := s.webhookUpdateIDs[updateID]; ok {
		return
	}

	s.webhookUpdateIDs[updateID] = struct{}{}
	s.webhookUpdateOrder = append(s.webhookUpdateOrder, updateID)

	if len(s.webhookUpdateOrder) <= recentWebhookUpdateLimit {
		return
	}

	evictedID := s.webhookUpdateOrder[0]
	s.webhookUpdateOrder = s.webhookUpdateOrder[1:]
	delete(s.webhookUpdateIDs, evictedID)
}

func (s *BotService) resetWebhookRuntimeState() {
	s.webhookMu.Lock()
	defer s.webhookMu.Unlock()

	s.webhookUpdateIDs = make(map[int64]struct{})
	s.webhookUpdateOrder = nil

	s.mu.Lock()
	s.currentBotName = ""
	s.mu.Unlock()
}

func shouldSkipWebhookUpdate(state *models.TelegramRuntimeState, updateID int64) bool {
	if state == nil || updateID <= 0 {
		return false
	}
	if state.LastUpdateID == updateID {
		return true
	}
	if hasRecordedWebhookUpdate(state.WebhookRecentUpdateIDs, updateID) {
		return true
	}

	// Keep supporting nearby out-of-order deliveries, but reject much older
	// stale replays that have fallen outside the persisted replay window.
	oldestAllowedUpdateID := state.LastUpdateID - recentWebhookUpdateLimit
	return oldestAllowedUpdateID > 0 && updateID <= oldestAllowedUpdateID
}

func hasRecordedWebhookUpdate(encoded string, updateID int64) bool {
	for _, candidate := range decodeWebhookRecentUpdateIDs(encoded) {
		if candidate == updateID {
			return true
		}
	}
	return false
}

func appendWebhookRecentUpdateID(encoded string, updateID int64) string {
	if updateID <= 0 {
		return normalizeWebhookRecentUpdateIDs(encoded)
	}

	ids := decodeWebhookRecentUpdateIDs(encoded)
	filtered := ids[:0]
	for _, existingID := range ids {
		if existingID != updateID {
			filtered = append(filtered, existingID)
		}
	}
	filtered = append(filtered, updateID)
	if len(filtered) > recentWebhookUpdateLimit {
		filtered = filtered[len(filtered)-recentWebhookUpdateLimit:]
	}

	data, err := json.Marshal(filtered)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func normalizeWebhookRecentUpdateIDs(encoded string) string {
	ids := decodeWebhookRecentUpdateIDs(encoded)
	if ids == nil {
		ids = []int64{}
	}
	data, err := json.Marshal(ids)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func decodeWebhookRecentUpdateIDs(encoded string) []int64 {
	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return nil
	}

	var ids []int64
	if err := json.Unmarshal([]byte(encoded), &ids); err != nil {
		return nil
	}
	if ids == nil {
		return []int64{}
	}
	return ids
}

func (s *BotService) newClient(cfg config.TelegramConfig, proxyCfg config.ProxyConfig) BotAPI {
	if !cfg.Enabled || strings.TrimSpace(cfg.BotToken) == "" {
		return nil
	}
	if s.clientFactory != nil {
		return s.clientFactory(cfg, proxyCfg)
	}
	return NewClient(cfg.BotToken, cfg.PollTimeoutSeconds, proxyCfg)
}

func cloneConfig(cfg *config.Config) *config.Config {
	if cfg == nil {
		return nil
	}

	cloned := *cfg
	return &cloned
}

func (s *BotService) markNotificationDelivered(ctx context.Context, logID uint, stage string, errMsg string) error {
	switch stage {
	case TelegramRequestStatusCompleted:
		return s.requestStore.MarkCompleted(ctx, logID)
	case TelegramRequestStatusFailed:
		return s.requestStore.MarkFailed(ctx, logID, errMsg)
	default:
		return nil
	}
}

type runtimeStateStore struct {
	db *gorm.DB
}

func newRuntimeStateStore(db *gorm.DB) RuntimeStateStore {
	return &runtimeStateStore{db: db}
}

func (s *runtimeStateStore) LoadOrCreate(ctx context.Context, botName string) (*models.TelegramRuntimeState, error) {
	state := &models.TelegramRuntimeState{}
	err := s.db.WithContext(ctx).Where("bot_name = ?", botName).First(state).Error
	if err == nil {
		return state, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	state.BotName = botName
	if err := s.db.WithContext(ctx).Create(state).Error; err != nil {
		return nil, err
	}

	return state, nil
}

func (s *runtimeStateStore) SaveProgress(ctx context.Context, botName string, lastUpdateID int64, processedUpdateID int64, polledAt time.Time) error {
	state, err := s.LoadOrCreate(ctx, botName)
	if err != nil {
		return err
	}

	if lastUpdateID < state.LastUpdateID {
		lastUpdateID = state.LastUpdateID
	}
	state.LastUpdateID = lastUpdateID
	state.WebhookRecentUpdateIDs = appendWebhookRecentUpdateID(state.WebhookRecentUpdateIDs, processedUpdateID)
	state.LastPollAt = &polledAt
	state.LastError = ""
	state.LastErrorAt = nil

	return s.db.WithContext(ctx).Save(state).Error
}

func (s *runtimeStateStore) SaveError(ctx context.Context, botName string, errText string, when time.Time) error {
	state, err := s.LoadOrCreate(ctx, botName)
	if err != nil {
		return err
	}

	state.LastError = errText
	state.LastErrorAt = &when
	return s.db.WithContext(ctx).Save(state).Error
}
