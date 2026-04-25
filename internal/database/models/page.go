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
	FrameRate      float32   `json:"frame_rate"`                       // 实际帧率，由 ffprobe 探测
	Quality        int8      `gorm:"index;default:0" json:"quality"`   // 画质编码
	Orientation    int8      `gorm:"default:0" json:"orientation"`     // 方向：0未知 1横屏 2竖屏
	Image          string    `gorm:"size:500" json:"image"`            // 封面URL
	DownloadStatus int       `gorm:"default:0" json:"download_status"` // 位标志
	Path           string    `gorm:"size:500" json:"path"`
	Kind           string    `gorm:"size:20;default:'video'" json:"kind"`     // video | image | live_photo
	FilePath       string    `gorm:"size:500" json:"file_path"`               // 单文件落地路径（图集场景使用）
	CreatedAt      time.Time `json:"created_at"`

	// 关联
	Video Video `gorm:"foreignKey:VideoID;constraint:OnDelete:CASCADE" json:"video,omitempty"`
}

// TableName 指定表名
func (Page) TableName() string {
	return "page"
}
