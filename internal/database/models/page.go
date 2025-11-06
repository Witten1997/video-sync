package models

import (
	"time"
)

// Page 视频分P模型
type Page struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	VideoID        uint      `gorm:"not null;index" json:"video_id"`
	CID            int64     `gorm:"column:cid;not null" json:"cid"`
	PID            int       `gorm:"column:pid;not null" json:"pid"` // 分P编号
	Name           string    `gorm:"size:255" json:"name"`
	Duration       int       `json:"duration"` // 时长（秒）
	Width          int       `json:"width"`
	Height         int       `json:"height"`
	Image          string    `gorm:"size:500" json:"image"`            // 封面URL
	DownloadStatus int       `gorm:"default:0" json:"download_status"` // 位标志
	Path           string    `gorm:"size:500" json:"path"`
	CreatedAt      time.Time `json:"created_at"`

	// 关联
	Video Video `gorm:"foreignKey:VideoID;constraint:OnDelete:CASCADE" json:"video,omitempty"`
}

// TableName 指定表名
func (Page) TableName() string {
	return "page"
}
