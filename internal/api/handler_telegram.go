package api

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"bili-download/internal/database/models"
	"bili-download/internal/telegram"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type telegramTestSendRequest struct {
	ChatID  int64  `json:"chat_id"`
	Message string `json:"message"`
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
