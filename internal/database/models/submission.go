package models

import (
	"time"
)

// Submission UP主投稿模型
type Submission struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	UpperID       int64      `gorm:"uniqueIndex;not null" json:"upper_id"` // UP主 ID
	Name          string     `gorm:"size:255;not null" json:"name"`
	Path          string     `gorm:"size:500" json:"path"`
	Enabled       bool       `gorm:"default:true;index" json:"enabled"`
	LatestRowAt   *time.Time `json:"latest_row_at,omitempty"`
	UseDynamicAPI bool       `gorm:"default:false" json:"use_dynamic_api"` // 是否使用动态 API
	Rule          string     `gorm:"type:jsonb" json:"rule,omitempty"`     // 过滤规则 JSON
	CreatedAt     time.Time  `json:"created_at"`

	// 关联
	Videos []Video `gorm:"foreignKey:SubmissionID;constraint:OnDelete:CASCADE" json:"videos,omitempty"`
}

// TableName 指定表名
func (Submission) TableName() string {
	return "submission"
}
