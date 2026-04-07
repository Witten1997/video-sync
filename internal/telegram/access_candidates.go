package telegram

import (
	"context"
	"strings"
	"time"

	"bili-download/internal/database/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	TelegramAccessCandidateStatusPending  = "pending"
	TelegramAccessCandidateStatusApproved = "approved"
)

type AccessCandidateInput struct {
	ChatID      int64
	UserID      int64
	ChatType    string
	Username    string
	FirstName   string
	LastName    string
	LastMessage string
}

type AccessCandidateStore interface {
	Capture(ctx context.Context, candidate AccessCandidateInput) error
	ListPending(ctx context.Context, limit int) ([]models.TelegramAccessCandidate, error)
	GetByID(ctx context.Context, id uint) (*models.TelegramAccessCandidate, error)
	MarkApproved(ctx context.Context, id uint) error
}

type GormAccessCandidateStore struct {
	db *gorm.DB
}

func NewAccessCandidateStore(db *gorm.DB) AccessCandidateStore {
	if db == nil {
		return nil
	}
	return &GormAccessCandidateStore{db: db}
}

func (s *GormAccessCandidateStore) Capture(ctx context.Context, candidate AccessCandidateInput) error {
	now := time.Now().UTC()
	row := models.TelegramAccessCandidate{
		ChatID:      candidate.ChatID,
		UserID:      candidate.UserID,
		ChatType:    strings.TrimSpace(candidate.ChatType),
		Username:    strings.TrimSpace(candidate.Username),
		FirstName:   strings.TrimSpace(candidate.FirstName),
		LastName:    strings.TrimSpace(candidate.LastName),
		LastMessage: strings.TrimSpace(candidate.LastMessage),
		Status:      TelegramAccessCandidateStatusPending,
		FirstSeenAt: now,
		LastSeenAt:  now,
	}

	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "chat_id"}, {Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"chat_type":    row.ChatType,
			"username":     row.Username,
			"first_name":   row.FirstName,
			"last_name":    row.LastName,
			"last_message": row.LastMessage,
			"last_seen_at": now,
			"status":       TelegramAccessCandidateStatusPending,
			"approved_at":  nil,
		}),
	}).Create(&row).Error
}

func (s *GormAccessCandidateStore) ListPending(ctx context.Context, limit int) ([]models.TelegramAccessCandidate, error) {
	if limit <= 0 {
		limit = 50
	}

	var items []models.TelegramAccessCandidate
	err := s.db.WithContext(ctx).
		Where("status = ?", TelegramAccessCandidateStatusPending).
		Order("last_seen_at DESC").
		Limit(limit).
		Find(&items).Error
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (s *GormAccessCandidateStore) GetByID(ctx context.Context, id uint) (*models.TelegramAccessCandidate, error) {
	var item models.TelegramAccessCandidate
	if err := s.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *GormAccessCandidateStore) MarkApproved(ctx context.Context, id uint) error {
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Model(&models.TelegramAccessCandidate{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":      TelegramAccessCandidateStatusApproved,
		"approved_at": &now,
	}).Error
}
