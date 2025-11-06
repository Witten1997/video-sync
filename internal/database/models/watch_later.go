package models

import (
	"time"
)

// WatchLater 稍后再看模型
type WatchLater struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Name        string     `gorm:"size:255;default:'稍后再看'" json:"name"`
	Path        string     `gorm:"size:500" json:"path"`
	Enabled     bool       `gorm:"default:true;index" json:"enabled"`
	LatestRowAt *time.Time `json:"latest_row_at,omitempty"`
	Rule        string     `gorm:"type:jsonb" json:"rule,omitempty"` // 过滤规则 JSON
	CreatedAt   time.Time  `json:"created_at"`

	// 关联
	Videos []Video `gorm:"foreignKey:WatchLaterID;constraint:OnDelete:CASCADE" json:"videos,omitempty"`
}

// TableName 指定表名
func (WatchLater) TableName() string {
	return "watch_later"
}
