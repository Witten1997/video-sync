package models

import (
	"time"
)

// SchedulerState 调度器状态
type SchedulerState struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	IsRunning     bool       `gorm:"default:false" json:"is_running"`
	LastRunAt     *time.Time `json:"last_run_at"`
	NextRunAt     *time.Time `json:"next_run_at"`
	CurrentSyncID string     `json:"current_sync_id"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// TableName 指定表名
func (SchedulerState) TableName() string {
	return "scheduler_state"
}
