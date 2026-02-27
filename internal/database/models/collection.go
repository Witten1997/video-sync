package models

import (
	"time"
)

// Collection 合集模型
type Collection struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CID       int64     `gorm:"uniqueIndex;not null" json:"c_id"` // B站合集 ID
	CType     string    `gorm:"size:20" json:"c_type"`            // 合集类型（series/season）
	Name      string    `gorm:"size:255;not null" json:"name"`
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
	Videos []Video `gorm:"foreignKey:CollectionID" json:"videos,omitempty"`
}

// TableName 指定表名
func (Collection) TableName() string {
	return "collection"
}
