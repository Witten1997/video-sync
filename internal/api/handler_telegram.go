package api

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"bili-download/internal/config"
	"bili-download/internal/database/models"
	"bili-download/internal/telegram"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type telegramTestSendRequest struct {
	ChatID  int64  `json:"chat_id"`
	Message string `json:"message"`
}

type telegramApproveAccessCandidateRequest struct {
	ApproveChatID bool `json:"approve_chat_id"`
	ApproveUserID bool `json:"approve_user_id"`
}

func (s *Server) handleTelegramStatus(c *gin.Context) {
	status := gin.H{
		"enabled":        s.config.Telegram.Enabled,
		"running":        s.telegramService != nil && s.telegramService.IsRunning(),
		"mode":           s.config.Telegram.Mode,
		"bot_name":       "",
		"last_update_id": int64(0),
		"last_poll_at":   nil,
		"last_error":     "",
		"last_error_at":  nil,
	}

	var runtimeState models.TelegramRuntimeState
	err := s.db.Order("updated_at DESC").First(&runtimeState).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		respondInternalError(c, err)
		return
	}
	if err == nil {
		status["bot_name"] = runtimeState.BotName
		status["last_update_id"] = runtimeState.LastUpdateID
		status["last_poll_at"] = runtimeState.LastPollAt
		status["last_error"] = runtimeState.LastError
		status["last_error_at"] = runtimeState.LastErrorAt
	}

	respondSuccess(c, status)
}

func (s *Server) handleTelegramRequestLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	query := s.db.Model(&models.TelegramRequestLog{})

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if chatID := c.Query("chat_id"); chatID != "" {
		query = query.Where("chat_id = ?", chatID)
	}
	if userID := c.Query("user_id"); userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if taskID := c.Query("task_id"); taskID != "" {
		query = query.Where("task_id = ?", taskID)
	}
	if recordID := c.Query("record_id"); recordID != "" {
		query = query.Where("record_id = ?", recordID)
	}
	if keyword := c.Query("keyword"); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("raw_url ILIKE ? OR raw_text ILIKE ? OR error_message ILIKE ?", like, like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		respondInternalError(c, err)
		return
	}

	var items []models.TelegramRequestLog
	if err := query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&items).Error; err != nil {
		respondInternalError(c, err)
		return
	}

	respondSuccess(c, gin.H{
		"items":       items,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

func (s *Server) handleTelegramAccessCandidates(c *gin.Context) {
	if s.telegramAccessCandidateStore == nil {
		respondError(c, http.StatusServiceUnavailable, "telegram access candidate store is not available")
		return
	}

	items, err := s.telegramAccessCandidateStore.ListPending(c.Request.Context(), 50)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	respondSuccess(c, items)
}

func (s *Server) handleTelegramApproveAccessCandidate(c *gin.Context) {
	if s.config == nil {
		respondInternalError(c, errors.New("telegram config is not loaded"))
		return
	}
	if s.telegramAccessCandidateStore == nil {
		respondError(c, http.StatusServiceUnavailable, "telegram access candidate store is not available")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		respondValidationError(c, "invalid candidate id")
		return
	}

	var req telegramApproveAccessCandidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, err.Error())
		return
	}
	if !req.ApproveChatID && !req.ApproveUserID {
		respondValidationError(c, "at least one approval target must be selected")
		return
	}

	candidate, err := s.telegramAccessCandidateStore.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusNotFound, "telegram access candidate not found")
			return
		}
		respondInternalError(c, err)
		return
	}

	newConfig := *s.config
	if req.ApproveChatID {
		newConfig.Telegram.AllowedChatIDs = appendUniqueInt64(newConfig.Telegram.AllowedChatIDs, candidate.ChatID)
	}
	if req.ApproveUserID {
		newConfig.Telegram.AllowedUserIDs = appendUniqueInt64(newConfig.Telegram.AllowedUserIDs, candidate.UserID)
	}
	if candidate.ChatType != "" {
		newConfig.Telegram.AllowedChatTypes = appendUniqueString(newConfig.Telegram.AllowedChatTypes, candidate.ChatType)
	}

	if err := newConfig.Validate(); err != nil {
		respondValidationError(c, fmt.Sprintf("config validation failed: %v", err))
		return
	}
	if err := config.Save(&newConfig, s.configPath); err != nil {
		respondInternalError(c, err)
		return
	}

	if err := s.applyTelegramRuntimeConfig(&newConfig); err != nil {
		respondInternalError(c, err)
		return
	}
	s.config = &newConfig

	if err := s.telegramAccessCandidateStore.MarkApproved(c.Request.Context(), uint(id)); err != nil {
		respondInternalError(c, err)
		return
	}

	respondSuccess(c, gin.H{
		"message":            "telegram access candidate approved",
		"allowed_chat_ids":   newConfig.Telegram.AllowedChatIDs,
		"allowed_user_ids":   newConfig.Telegram.AllowedUserIDs,
		"allowed_chat_types": newConfig.Telegram.AllowedChatTypes,
	})
}

func (s *Server) handleTelegramTestSend(c *gin.Context) {
	if s.config == nil {
		respondInternalError(c, errors.New("telegram config is not loaded"))
		return
	}
	if strings.TrimSpace(s.config.Telegram.BotToken) == "" {
		respondValidationError(c, "telegram bot token is not configured")
		return
	}

	var req telegramTestSendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, err.Error())
		return
	}
	if req.ChatID == 0 {
		respondValidationError(c, "chat_id must not be 0")
		return
	}

	message := strings.TrimSpace(req.Message)
	if message == "" {
		message = fmt.Sprintf("video-sync Telegram test message at %s", time.Now().Format(time.RFC3339))
	}
	if utf8.RuneCountInString(message) > 4096 {
		respondValidationError(c, "message must be 4096 characters or fewer")
		return
	}

	client := s.newTelegramClient()
	if client == nil {
		respondInternalError(c, errors.New("telegram client is not configured"))
		return
	}

	sentMessage, err := client.SendMessage(c.Request.Context(), req.ChatID, message, 0)
	if err != nil {
		respondError(c, http.StatusBadGateway, fmt.Sprintf("telegram test send failed: %v", err))
		return
	}

	respondSuccess(c, gin.H{
		"chat_id":    req.ChatID,
		"message_id": sentMessage.MessageID,
		"text":       message,
	})
}

func (s *Server) handleTelegramReconnect(c *gin.Context) {
	if s.telegramService == nil {
		respondError(c, http.StatusServiceUnavailable, "telegram service is not attached")
		return
	}

	if err := s.telegramService.Reconnect(); err != nil {
		respondValidationError(c, err.Error())
		return
	}

	respondSuccess(c, gin.H{
		"message": "telegram reconnect requested",
	})
}

func (s *Server) handleTelegramWebhook(c *gin.Context) {
	if s.config == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "telegram config is not loaded"})
		return
	}
	if !s.config.Telegram.Enabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "telegram service is disabled"})
		return
	}
	if s.config.Telegram.Mode != "webhook" {
		c.JSON(http.StatusConflict, gin.H{"message": "telegram webhook mode is not enabled"})
		return
	}
	if s.telegramService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "telegram service is not attached"})
		return
	}

	secret := strings.TrimSpace(s.config.Telegram.WebhookSecret)
	headerSecret := strings.TrimSpace(c.GetHeader("X-Telegram-Bot-Api-Secret-Token"))
	if secret == "" || subtle.ConstantTimeCompare([]byte(secret), []byte(headerSecret)) != 1 {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid telegram webhook secret"})
		return
	}

	var update telegram.Update
	if err := c.ShouldBindJSON(&update); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	if err := s.telegramService.HandleWebhookUpdate(c.Request.Context(), update); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) applyTelegramRuntimeConfig(newConfig *config.Config) error {
	telegramChanged := s.config == nil || !reflect.DeepEqual(s.config.Telegram, newConfig.Telegram)
	if !telegramChanged || s.telegramService == nil {
		return nil
	}

	if _, err := s.telegramService.ApplyConfig(newConfig); err != nil {
		return err
	}
	return nil
}

func appendUniqueInt64(values []int64, value int64) []int64 {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func appendUniqueString(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
