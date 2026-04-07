package telegram

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"bili-download/internal/database/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RequestStore struct {
	db *gorm.DB
}

type RequestLogInput struct {
	UpdateID  int64
	ChatID    int64
	MessageID int64
	UserID    int64
	RawText   string
	RawURL    string
}

type RequestSummary struct {
	Log          models.TelegramRequestLog
	RecordStatus string
	Title        string
	ErrorMessage string
}

func NewRequestStore(db *gorm.DB) *RequestStore {
	return &RequestStore{db: db}
}

func (s *RequestStore) EnsurePending(ctx context.Context, in RequestLogInput) (*models.TelegramRequestLog, bool, error) {
	log := &models.TelegramRequestLog{
		UpdateID:  in.UpdateID,
		ChatID:    in.ChatID,
		MessageID: in.MessageID,
		UserID:    in.UserID,
		RawText:   in.RawText,
		RawURL:    in.RawURL,
		URLHash:   hashURL(in.RawURL),
		Status:    TelegramRequestStatusPending,
	}

	tx := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "update_id"}, {Name: "url_hash"}},
		DoNothing: true,
	}).Create(log)
	if tx.Error != nil {
		return nil, false, tx.Error
	}
	if tx.RowsAffected > 0 {
		return log, true, nil
	}

	var existing models.TelegramRequestLog
	err := s.db.WithContext(ctx).
		Where("update_id = ? AND url_hash = ?", in.UpdateID, hashURL(in.RawURL)).
		First(&existing).Error
	if err != nil {
		return nil, false, err
	}

	return &existing, false, nil
}

func (s *RequestStore) MarkQueued(ctx context.Context, id uint, videoID uint, recordID uint, taskID string) error {
	return s.db.WithContext(ctx).Model(&models.TelegramRequestLog{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":    TelegramRequestStatusQueued,
		"video_id":  videoID,
		"record_id": recordID,
		"task_id":   taskID,
	}).Error
}

func (s *RequestStore) MarkReplySent(ctx context.Context, id uint, replyMessageID int64) error {
	return s.db.WithContext(ctx).Model(&models.TelegramRequestLog{}).Where("id = ?", id).Update("reply_message_id", replyMessageID).Error
}

func (s *RequestStore) MarkFailed(ctx context.Context, id uint, errMsg string) error {
	return s.db.WithContext(ctx).Model(&models.TelegramRequestLog{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":        TelegramRequestStatusFailed,
		"error_message": errMsg,
	}).Error
}

func (s *RequestStore) MarkCompleted(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Model(&models.TelegramRequestLog{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":        TelegramRequestStatusCompleted,
		"error_message": "",
	}).Error
}

func (s *RequestStore) FindByTaskID(ctx context.Context, chatID int64, userID int64, taskID string) (*RequestSummary, error) {
	var log models.TelegramRequestLog
	err := s.db.WithContext(ctx).Where("chat_id = ? AND user_id = ? AND task_id = ?", chatID, userID, taskID).Order("created_at DESC").First(&log).Error
	if err != nil {
		return nil, err
	}

	return s.hydrateSummary(ctx, log)
}

func (s *RequestStore) ListRecentByUser(ctx context.Context, chatID int64, userID int64, limit int) ([]RequestSummary, error) {
	if limit <= 0 {
		limit = 5
	}

	var logs []models.TelegramRequestLog
	err := s.db.WithContext(ctx).
		Where("chat_id = ? AND user_id = ?", chatID, userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error
	if err != nil {
		return nil, err
	}

	items := make([]RequestSummary, 0, len(logs))
	for _, log := range logs {
		summary, err := s.hydrateSummary(ctx, log)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				items = append(items, RequestSummary{Log: log})
				continue
			}
			return nil, err
		}
		items = append(items, *summary)
	}

	return items, nil
}

func (s *RequestStore) ListNotificationCandidates(ctx context.Context, limit int) ([]RequestSummary, error) {
	if limit <= 0 {
		limit = 20
	}

	var logs []models.TelegramRequestLog
	err := s.db.WithContext(ctx).
		Where("status = ? AND reply_message_id IS NOT NULL AND record_id IS NOT NULL", TelegramRequestStatusQueued).
		Order("created_at ASC").
		Limit(limit).
		Find(&logs).Error
	if err != nil {
		return nil, err
	}

	items := make([]RequestSummary, 0, len(logs))
	for _, log := range logs {
		summary, err := s.hydrateSummary(ctx, log)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return nil, err
		}
		items = append(items, *summary)
	}

	return items, nil
}

func (s *RequestStore) hydrateSummary(ctx context.Context, log models.TelegramRequestLog) (*RequestSummary, error) {
	summary := &RequestSummary{
		Log:          log,
		ErrorMessage: log.ErrorMessage,
	}
	if log.RecordID == nil {
		return summary, nil
	}

	var record models.DownloadRecord
	err := s.db.WithContext(ctx).Preload("Video").First(&record, *log.RecordID).Error
	if err != nil {
		return nil, err
	}

	summary.RecordStatus = record.Status
	summary.Title = record.Video.Name
	if record.ErrorMessage != "" {
		summary.ErrorMessage = record.ErrorMessage
	}
	if summary.Title == "" {
		summary.Title = strings.TrimSpace(log.RawURL)
	}
	return summary, nil
}

func hashURL(rawURL string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(rawURL)))
	return hex.EncodeToString(sum[:])
}
