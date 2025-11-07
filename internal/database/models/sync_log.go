package models

import (
	"time"

	"gorm.io/datatypes"
)

// SyncLog 同步任务日志
type SyncLog struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	TaskID      string     `gorm:"uniqueIndex;not null" json:"task_id"`
	TriggerType string     `gorm:"not null;index" json:"trigger_type"` // auto/manual
	Status      string     `gorm:"not null;index" json:"status"`       // running/completed/failed/cancelled
	StartAt     time.Time  `gorm:"not null;index" json:"start_at"`
	EndAt       *time.Time `json:"end_at"`
	DurationMs  int        `json:"duration_ms"`

	SourcesTotal   int `gorm:"default:0" json:"sources_total"`
	SourcesScanned int `gorm:"default:0" json:"sources_scanned"`
	SourcesFailed  int `gorm:"default:0" json:"sources_failed"`

	VideosFound    int `gorm:"default:0" json:"videos_found"`
	VideosNew      int `gorm:"default:0" json:"videos_new"`
	VideosFiltered int `gorm:"default:0" json:"videos_filtered"`
	VideosQueued   int `gorm:"default:0" json:"videos_queued"`

	TasksCreated   int `gorm:"default:0" json:"tasks_created"`
	TasksCompleted int `gorm:"default:0" json:"tasks_completed"`
	TasksFailed    int `gorm:"default:0" json:"tasks_failed"`

	ErrorMessage string         `json:"error_message"`
	Metadata     datatypes.JSON `gorm:"type:jsonb" json:"metadata"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 关联
	SourceScans []VideoSourceScan `gorm:"foreignKey:SyncLogID;constraint:OnDelete:CASCADE" json:"source_scans,omitempty"`
}

// TableName 指定表名
func (SyncLog) TableName() string {
	return "sync_logs"
}

// VideoSourceScan 视频源扫描记录
type VideoSourceScan struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	SyncLogID  uint   `gorm:"not null;index" json:"sync_log_id"`
	SourceID   string `gorm:"not null;index" json:"source_id"`
	SourceType string `gorm:"not null" json:"source_type"` // favorite/submission/collection/watch_later
	SourceName string `json:"source_name"`

	ScannedAt    time.Time `gorm:"not null" json:"scanned_at"`
	DurationMs   int       `json:"duration_ms"`
	Success      bool      `gorm:"default:true" json:"success"`
	ErrorMessage string    `json:"error_message"`

	VideosFound    int `gorm:"default:0" json:"videos_found"`
	VideosNew      int `gorm:"default:0" json:"videos_new"`
	VideosFiltered int `gorm:"default:0" json:"videos_filtered"`
	VideosQueued   int `gorm:"default:0" json:"videos_queued"`

	Metadata  datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
	CreatedAt time.Time      `json:"created_at"`
}

// TableName 指定表名
func (VideoSourceScan) TableName() string {
	return "video_source_scans"
}
