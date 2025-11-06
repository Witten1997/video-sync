package models

import (
	"time"
)

// Favorite 收藏夹模型
type Favorite struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	FID         int64      `gorm:"uniqueIndex;not null" json:"f_id"` // B站收藏夹 ID
	Name        string     `gorm:"size:255;not null" json:"name"`
	Path        string     `gorm:"size:500" json:"path"` // 保存路径模板
	Enabled     bool       `gorm:"default:true;index" json:"enabled"`
	LatestRowAt *time.Time `json:"latest_row_at,omitempty"`          // 最后扫描到的视频时间
	Rule        string     `gorm:"type:jsonb" json:"rule,omitempty"` // 过滤规则 JSON
	CreatedAt   time.Time  `json:"created_at"`

	// 关联
	Videos []Video `gorm:"foreignKey:FavoriteID;constraint:OnDelete:CASCADE" json:"videos,omitempty"`
}

// TableName 指定表名
func (Favorite) TableName() string {
	return "favorite"
}
