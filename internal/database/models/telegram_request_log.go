package models

import "time"

type TelegramRequestLog struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UpdateID       int64     `gorm:"not null;uniqueIndex:idx_tg_update_url" json:"update_id"`
	ChatID         int64     `gorm:"not null;index" json:"chat_id"`
	MessageID      int64     `gorm:"not null;index" json:"message_id"`
	UserID         int64     `gorm:"not null;index" json:"user_id"`
	RawText        string    `gorm:"type:text" json:"raw_text"`
	RawURL         string    `gorm:"size:1000" json:"raw_url"`
	URLHash        string    `gorm:"size:128;not null;uniqueIndex:idx_tg_update_url" json:"url_hash"`
	Status         string    `gorm:"size:32;not null;index" json:"status"`
	VideoID        *uint     `gorm:"index" json:"video_id"`
	RecordID       *uint     `gorm:"index" json:"record_id"`
	TaskID         string    `gorm:"size:128;index" json:"task_id"`
	ReplyMessageID *int64    `json:"reply_message_id"`
	ErrorMessage   string    `gorm:"type:text" json:"error_message"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (TelegramRequestLog) TableName() string {
	return "telegram_request_logs"
}
