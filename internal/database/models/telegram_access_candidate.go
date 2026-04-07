package models

import "time"

type TelegramAccessCandidate struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	ChatID      int64      `gorm:"not null;uniqueIndex:idx_tg_access_candidate" json:"chat_id"`
	UserID      int64      `gorm:"not null;uniqueIndex:idx_tg_access_candidate" json:"user_id"`
	ChatType    string     `gorm:"size:32;not null;index" json:"chat_type"`
	Username    string     `gorm:"size:255" json:"username"`
	FirstName   string     `gorm:"size:255" json:"first_name"`
	LastName    string     `gorm:"size:255" json:"last_name"`
	LastMessage string     `gorm:"type:text" json:"last_message"`
	Status      string     `gorm:"size:32;not null;index" json:"status"`
	FirstSeenAt time.Time  `gorm:"not null;index" json:"first_seen_at"`
	LastSeenAt  time.Time  `gorm:"not null;index" json:"last_seen_at"`
	ApprovedAt  *time.Time `json:"approved_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (TelegramAccessCandidate) TableName() string {
	return "telegram_access_candidates"
}
