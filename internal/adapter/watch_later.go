package adapter

import (
	"context"
	"fmt"
	"strings"
	"time"

	"bili-download/internal/bilibili"
)

// WatchLaterAdapter 稍后再看适配器
type WatchLaterAdapter struct {
	client *bilibili.Client
	config *WatchLaterConfig
}

// NewWatchLaterAdapter 创建稍后再看适配器
func NewWatchLaterAdapter(client *bilibili.Client, config *WatchLaterConfig) *WatchLaterAdapter {
	return &WatchLaterAdapter{
		client: client,
		config: config,
	}
}

// GetType 获取视频源类型
func (a *WatchLaterAdapter) GetType() VideoSourceType {
	return SourceTypeWatchLater
}

// GetID 获取视频源唯一标识
func (a *WatchLaterAdapter) GetID() string {
	return "watch_later"
}

// GetName 获取视频源名称
func (a *WatchLaterAdapter) GetName() string {
	if a.config.Name != "" {
		return a.config.Name
	}
	return "稍后再看"
}

// Scan 扫描视频源
func (a *WatchLaterAdapter) Scan(ctx context.Context, opts *ScanOptions) ([]VideoInfo, error) {
	if opts == nil {
		opts = &ScanOptions{}
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// 获取稍后再看列表
	resp, err := a.client.GetWatchLaterList()
	if err != nil {
		return nil, fmt.Errorf("获取稍后再看列表失败: %w", err)
	}

	var result []VideoInfo

	// 转换为统一的VideoInfo格式
	for i, video := range resp.List {
		// 应用偏移量
		if opts.Offset > 0 && i < opts.Offset {
			continue
		}

		// 应用过滤条件
		if !a.matchFilter(video, opts) {
			continue
		}

		videoInfo := a.convertToVideoInfo(video)
		result = append(result, videoInfo)

		// 如果设置了限制且已达到，返回
		if opts.Limit > 0 && len(result) >= opts.Limit {
			break
		}
	}

	return result, nil
}

// GetVideoCount 获取视频总数
func (a *WatchLaterAdapter) GetVideoCount(ctx context.Context) (int, error) {
	resp, err := a.client.GetWatchLaterList()
	if err != nil {
		return 0, fmt.Errorf("获取稍后再看列表失败: %w", err)
	}
	return resp.Count, nil
}

// Validate 验证配置
func (a *WatchLaterAdapter) Validate(ctx context.Context) error {
	// 尝试获取稍后再看列表
	_, err := a.client.GetWatchLaterList()
	if err != nil {
		return fmt.Errorf("稍后再看验证失败: %w", err)
	}
	return nil
}

// matchFilter 检查视频是否匹配过滤条件
func (a *WatchLaterAdapter) matchFilter(video bilibili.WatchLaterVideo, opts *ScanOptions) bool {
	filter := opts.Filter
	if filter == nil && a.config.Filter == nil {
		return true
	}

	// 优先使用扫描选项中的过滤器
	if filter == nil {
		filter = a.config.Filter
	}

	// 检查时长
	if filter.MinDuration > 0 && video.Duration < filter.MinDuration {
		return false
	}
	if filter.MaxDuration > 0 && video.Duration > filter.MaxDuration {
		return false
	}

	// 检查发布时间
	pubTime := time.Unix(video.PubDate, 0)
	if !filter.MinPublishTime.IsZero() && pubTime.Before(filter.MinPublishTime) {
		return false
	}
	if !filter.MaxPublishTime.IsZero() && pubTime.After(filter.MaxPublishTime) {
		return false
	}

	// 检查关键词
	if len(filter.Keywords) > 0 {
		matched := false
		titleLower := strings.ToLower(video.Title)
		for _, keyword := range filter.Keywords {
			if strings.Contains(titleLower, strings.ToLower(keyword)) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// 检查排除关键词
	if len(filter.ExcludeKeywords) > 0 {
		titleLower := strings.ToLower(video.Title)
		for _, keyword := range filter.ExcludeKeywords {
			if strings.Contains(titleLower, strings.ToLower(keyword)) {
				return false
			}
		}
	}

	// 检查是否只要新视频
	if opts.OnlyNew && !opts.LastScanTime.IsZero() {
		addTime := time.Unix(video.AddAt, 0)
		if !addTime.After(opts.LastScanTime) {
			return false
		}
	}

	return true
}

// convertToVideoInfo 转换为统一的VideoInfo格式
func (a *WatchLaterAdapter) convertToVideoInfo(video bilibili.WatchLaterVideo) VideoInfo {
	return VideoInfo{
		BVid:        video.BVid,
		Aid:         video.Aid,
		Title:       video.Title,
		Description: video.Desc,
		Duration:    video.Duration,
		PubDate:     time.Unix(video.PubDate, 0),
		Owner: OwnerInfo{
			Mid:  video.Owner.Mid,
			Name: video.Owner.Name,
			Face: video.Owner.Face,
		},
		Cover: video.Pic,
		Stats: StatsInfo{
			View:     video.Stat.View,
			Danmaku:  video.Stat.Danmaku,
			Reply:    video.Stat.Reply,
			Like:     video.Stat.Like,
			Coin:     video.Stat.Coin,
			Favorite: video.Stat.Favorite,
			Share:    video.Stat.Share,
		},
		SourceType: SourceTypeWatchLater,
		SourceID:   "watch_later",
		AddTime:    time.Unix(video.AddAt, 0),
	}
}
