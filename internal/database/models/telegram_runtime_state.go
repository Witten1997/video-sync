package models

import "time"

type TelegramRuntimeState struct {
	ID                     uint       `gorm:"primaryKey" json:"id"`
	BotName                string     `gorm:"size:128;uniqueIndex" json:"bot_name"`
	LastUpdateID           int64      `gorm:"not null;default:0" json:"last_update_id"`
	WebhookRecentUpdateIDs string     `gorm:"type:text;not null;default:'[]'" json:"-"`
	LastPollAt             *time.Time `json:"last_poll_at"`
	LastError              string     `gorm:"type:text" json:"last_error"`
	LastErrorAt            *time.Time `json:"last_error_at"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

func (TelegramRuntimeState) TableName() string {
	return "telegram_runtime_state"
}
