package models

import (
	"time"
)

// WatchLater 稍后再看模型
type WatchLater struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:255;default:'稍后再看'" json:"name"`
	Path      string    `gorm:"size:500" json:"path"`
	Enabled   bool      `gorm:"default:true;index" json:"enabled"`
	Rule      string    `gorm:"type:jsonb" json:"rule,omitempty"` // 过滤规则 JSON
	CreatedAt time.Time `json:"created_at"`

	// 调度相关字段
	Priority            int        `gorm:"default:0" json:"priority"`              // 优先级 (0-10)
	HealthStatus        string     `gorm:"default:'healthy'" json:"health_status"` // healthy/degraded/unhealthy
	ConsecutiveFailures int        `gorm:"default:0" json:"consecutive_failures"`  // 连续失败次数
	LastScanAt          *time.Time `json:"last_scan_at,omitempty"`                 // 最后扫描时间
	LastScanError       string     `json:"last_scan_error,omitempty"`              // 最后扫描错误
	LastSuccessAt       *time.Time `json:"last_success_at,omitempty"`              // 最后成功时间

	// 关联
	Videos []Video `gorm:"foreignKey:WatchLaterID;constraint:OnDelete:CASCADE" json:"videos,omitempty"`
}

// TableName 指定表名
func (WatchLater) TableName() string {
	return "watch_later"
}
