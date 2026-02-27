package models

import (
	"time"

	"github.com/lib/pq"
)

// Video 视频模型
type Video struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	BVid           string         `gorm:"column:bvid;index;size:20;not null" json:"bvid"`
	Name           string         `gorm:"size:255;not null" json:"name"`
	Intro          string         `gorm:"type:text" json:"intro"`
	Cover          string         `gorm:"size:500" json:"cover"`
	Tags           pq.StringArray `gorm:"type:text[]" json:"tags"`
	UpperID        int64          `gorm:"not null;index" json:"upper_id"`
	UpperName      string         `gorm:"size:100" json:"upper_name"`
	UpperFace      string         `gorm:"size:500" json:"upper_face"`
	ViewCount      int            `gorm:"default:0" json:"view_count"`
	Category       int            `json:"category"`
	PubTime        time.Time      `gorm:"column:pubtime;not null;index" json:"pubtime"`
	FavTime        time.Time      `gorm:"column:favtime;not null;index" json:"favtime"`
	CTime          time.Time      `gorm:"column:ctime;not null" json:"ctime"`
	SinglePage     bool           `json:"single_page"`
	Valid          bool           `gorm:"default:true" json:"valid"`
	ShouldDownload bool           `gorm:"default:true" json:"should_download"`
	DownloadStatus int            `gorm:"default:0" json:"download_status"` // 位标志
	Path           string         `gorm:"size:500" json:"path"`

	// 外键关系
	FavoriteID   *uint `gorm:"index" json:"favorite_id,omitempty"`
	WatchLaterID *uint `gorm:"index" json:"watch_later_id,omitempty"`
	CollectionID *uint `gorm:"index" json:"collection_id,omitempty"`
	SubmissionID *uint `gorm:"index" json:"submission_id,omitempty"`

	// 关联
	Pages []Page `gorm:"foreignKey:VideoID" json:"pages,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名
func (Video) TableName() string {
	return "video"
}
