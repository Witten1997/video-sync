package models

import (
	"time"

	"gorm.io/datatypes"
)

// DownloadRecord 下载记录
type DownloadRecord struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	VideoID      uint           `gorm:"not null;index" json:"video_id"`
	SyncLogID    *uint          `gorm:"index" json:"sync_log_id"`
	SourceType   string         `gorm:"size:50;index" json:"source_type"`
	SourceID     uint           `gorm:"index" json:"source_id"`
	SourceName   string         `gorm:"size:255" json:"source_name"`
	Status       string         `gorm:"size:20;not null;index;default:pending" json:"status"` // pending/downloading/completed/failed
	FileDetails  datatypes.JSON `gorm:"type:jsonb" json:"file_details"`
	ErrorMessage string         `gorm:"type:text" json:"error_message"`
	StartedAt    *time.Time     `json:"started_at"`
	CompletedAt  *time.Time     `json:"completed_at"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`

	// 关联
	Video Video `gorm:"foreignKey:VideoID;constraint:OnDelete:CASCADE" json:"video,omitempty"`
}

func (DownloadRecord) TableName() string {
	return "download_records"
}

// FileDetail 文件详情（用于序列化/反序列化 FileDetails）
type FileDetail struct {
	Name     string  `json:"name"`     // video/nfo/cover/danmaku/subtitle/upper
	Label    string  `json:"label"`    // 显示名称
	Status   string  `json:"status"`   // pending/downloading/completed/failed/skipped
	Size     int64   `json:"size"`     // 已下载大小
	Progress float64 `json:"progress"` // 0-100
}

// FileDetailsData FileDetails 的 JSON 结构
type FileDetailsData struct {
	Files []FileDetail `json:"files"`
}
