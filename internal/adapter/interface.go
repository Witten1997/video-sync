package adapter

import (
	"context"
	"time"
)

// VideoSourceType 视频源类型
type VideoSourceType string

const (
	// SourceTypeFavorite 收藏夹
	SourceTypeFavorite VideoSourceType = "favorite"
	// SourceTypeWatchLater 稍后再看
	SourceTypeWatchLater VideoSourceType = "watch_later"
	// SourceTypeCollection 合集/系列
	SourceTypeCollection VideoSourceType = "collection"
	// SourceTypeSubmission UP主投稿
	SourceTypeSubmission VideoSourceType = "submission"
)

// VideoSource 视频源适配器接口
type VideoSource interface {
	// GetType 获取视频源类型
	GetType() VideoSourceType

	// GetID 获取视频源唯一标识（如收藏夹ID、UP主ID等）
	GetID() string

	// GetName 获取视频源名称
	GetName() string

	// Scan 扫描视频源，返回视频列表
	Scan(ctx context.Context, opts *ScanOptions) ([]VideoInfo, error)

	// GetVideoCount 获取视频总数
	GetVideoCount(ctx context.Context) (int, error)

	// Validate 验证视频源配置是否有效
	Validate(ctx context.Context) error
}

// ScanOptions 扫描选项
type ScanOptions struct {
	// OnlyNew 仅扫描新增视频
	OnlyNew bool
	// LastScanTime 上次扫描时间
	LastScanTime time.Time
	// Limit 限制返回数量（0表示不限制）
	Limit int
	// Offset 偏移量（用于分页）
	Offset int
	// Filter 过滤条件
	Filter *VideoFilter
}

// VideoFilter 视频过滤条件
type VideoFilter struct {
	// MinDuration 最小时长（秒）
	MinDuration int
	// MaxDuration 最大时长（秒）
	MaxDuration int
	// Keywords 关键词筛选
	Keywords []string
	// ExcludeKeywords 排除关键词
	ExcludeKeywords []string
	// MinPublishTime 最早发布时间
	MinPublishTime time.Time
	// MaxPublishTime 最晚发布时间
	MaxPublishTime time.Time
}

// VideoInfo 视频信息（适配器统一格式）
type VideoInfo struct {
	// BVid 视频BV号
	BVid string
	// Aid 视频AV号
	Aid int64
	// Title 标题
	Title string
	// Description 简介
	Description string
	// Duration 时长（秒）
	Duration int
	// PubDate 发布时间
	PubDate time.Time
	// Owner UP主信息
	Owner OwnerInfo
	// Cover 封面URL
	Cover string
	// Pages 分P信息
	Pages []PageInfo
	// Tags 标签
	Tags []string
	// Stats 统计信息
	Stats StatsInfo
	// SourceType 来源类型
	SourceType VideoSourceType
	// SourceID 来源ID
	SourceID string
	// AddTime 添加到源的时间（收藏时间、投稿时间等）
	AddTime time.Time
}

// OwnerInfo UP主信息
type OwnerInfo struct {
	Mid  int64  // UP主ID
	Name string // 昵称
	Face string // 头像URL
}

// PageInfo 分P信息
type PageInfo struct {
	CID      int64  // CID
	Page     int    // 分P序号
	Part     string // 分P标题
	Duration int    // 时长（秒）
	Width    int    // 宽度
	Height   int    // 高度
}

// StatsInfo 统计信息
type StatsInfo struct {
	View     int // 播放数
	Danmaku  int // 弹幕数
	Reply    int // 评论数
	Like     int // 点赞数
	Coin     int // 投币数
	Favorite int // 收藏数
	Share    int // 分享数
}

// SourceConfig 视频源配置（基础配置）
type SourceConfig struct {
	// Type 类型
	Type VideoSourceType
	// ID 唯一标识
	ID string
	// Name 名称
	Name string
	// Enabled 是否启用
	Enabled bool
	// Filter 过滤条件
	Filter *VideoFilter
	// ScanInterval 扫描间隔（秒）
	ScanInterval int
	// DownloadPath 下载路径
	DownloadPath string
}

// FavoriteConfig 收藏夹配置
type FavoriteConfig struct {
	SourceConfig
	MediaID string // 收藏夹ID
	Order   string // 排序方式：mtime, view, pubtime
}

// WatchLaterConfig 稍后再看配置
type WatchLaterConfig struct {
	SourceConfig
	// 稍后再看无需额外配置
}

// CollectionConfig 合集/系列配置
type CollectionConfig struct {
	SourceConfig
	Mid            string // UP主ID
	SeasonID       string // 合集ID（合集类型使用）
	SeriesID       string // 系列ID（系列类型使用）
	CollectionType string // 类型：season-合集，series-系列
	SortReverse    bool   // 是否倒序
}

// SubmissionConfig UP主投稿配置
type SubmissionConfig struct {
	SourceConfig
	Mid   string // UP主ID
	Order string // 排序方式：pubdate, click, stow
	Tid   int    // 分区筛选（0表示不筛选）
}
