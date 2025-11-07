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

	// 调度相关字段
	Priority            int        `gorm:"default:0" json:"priority"`              // 优先级 (0-10)
	HealthStatus        string     `gorm:"default:'healthy'" json:"health_status"` // healthy/degraded/unhealthy
	ConsecutiveFailures int        `gorm:"default:0" json:"consecutive_failures"`  // 连续失败次数
	LastScanAt          *time.Time `json:"last_scan_at,omitempty"`                 // 最后扫描时间
	LastScanError       string     `json:"last_scan_error,omitempty"`              // 最后扫描错误
	LastSuccessAt       *time.Time `json:"last_success_at,omitempty"`              // 最后成功时间

	// 关联
	Videos []Video `gorm:"foreignKey:SubmissionID;constraint:OnDelete:CASCADE" json:"videos,omitempty"`
}

// TableName 指定表名
func (Submission) TableName() string {
	return "submission"
}
