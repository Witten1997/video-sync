package models

import (
	"time"
)

// Collection 合集模型
type Collection struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	CID         int64      `gorm:"uniqueIndex;not null" json:"c_id"` // B站合集 ID
	CType       string     `gorm:"size:20" json:"c_type"`            // 合集类型（series/season）
	Name        string     `gorm:"size:255;not null" json:"name"`
	Path        string     `gorm:"size:500" json:"path"`
	Enabled     bool       `gorm:"default:true;index" json:"enabled"`
	LatestRowAt *time.Time `json:"latest_row_at,omitempty"`
	Rule        string     `gorm:"type:jsonb" json:"rule,omitempty"` // 过滤规则 JSON
	CreatedAt   time.Time  `json:"created_at"`

	// 关联
	Videos []Video `gorm:"foreignKey:CollectionID;constraint:OnDelete:CASCADE" json:"videos,omitempty"`
}

// TableName 指定表名
func (Collection) TableName() string {
	return "collection"
}
