package adapter

import (
	"context"
	"fmt"
	"strings"
	"time"

	"bili-download/internal/bilibili"
)

// FavoriteAdapter 收藏夹适配器
type FavoriteAdapter struct {
	client *bilibili.Client
	config *FavoriteConfig
}

// NewFavoriteAdapter 创建收藏夹适配器
func NewFavoriteAdapter(client *bilibili.Client, config *FavoriteConfig) *FavoriteAdapter {
	return &FavoriteAdapter{
		client: client,
		config: config,
	}
}

// GetType 获取视频源类型
func (a *FavoriteAdapter) GetType() VideoSourceType {
	return SourceTypeFavorite
}

// GetID 获取视频源唯一标识
func (a *FavoriteAdapter) GetID() string {
	return a.config.MediaID
}

// GetName 获取视频源名称
func (a *FavoriteAdapter) GetName() string {
	if a.config.Name != "" {
		return a.config.Name
	}
	// 如果没有设置名称，从API获取
	info, err := a.client.GetFavoriteInfo(a.config.MediaID)
	if err != nil {
		return fmt.Sprintf("收藏夹_%s", a.config.MediaID)
	}
	return info.Title
}

// Scan 扫描视频源
func (a *FavoriteAdapter) Scan(ctx context.Context, opts *ScanOptions) ([]VideoInfo, error) {
	if opts == nil {
		opts = &ScanOptions{}
	}

	var allVideos []VideoInfo
	page := 1
	pageSize := 20

	// 如果设置了偏移量和限制，计算起始页
	if opts.Offset > 0 {
		page = opts.Offset/pageSize + 1
	}

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// 获取收藏夹视频列表
		params := bilibili.FavoriteListParams{
			MediaID: a.config.MediaID,
			Pn:      page,
			Ps:      pageSize,
			Order:   a.config.Order,
		}

		if params.Order == "" {
			params.Order = "mtime" // 默认按收藏时间排序
		}

		resp, err := a.client.GetFavoriteList(params)
		if err != nil {
			return nil, fmt.Errorf("获取收藏夹列表失败: %w", err)
		}

		// 转换为统一的VideoInfo格式
		for _, media := range resp.Medias {
			// 应用过滤条件
			if !a.matchFilter(media, opts) {
				continue
			}

			videoInfo := a.convertToVideoInfo(media)
			allVideos = append(allVideos, videoInfo)

			// 如果设置了限制且已达到，返回
			if opts.Limit > 0 && len(allVideos) >= opts.Limit {
				return allVideos, nil
			}
		}

		// 检查是否还有更多视频
		if !resp.HasMore || len(resp.Medias) == 0 {
			break
		}

		page++

		// 翻页间隔，防止触发B站风控
		time.Sleep(300 * time.Millisecond)
	}

	return allVideos, nil
}

// GetVideoCount 获取视频总数
func (a *FavoriteAdapter) GetVideoCount(ctx context.Context) (int, error) {
	info, err := a.client.GetFavoriteInfo(a.config.MediaID)
	if err != nil {
		return 0, fmt.Errorf("获取收藏夹信息失败: %w", err)
	}
	return info.MediaCount, nil
}

// Validate 验证配置
func (a *FavoriteAdapter) Validate(ctx context.Context) error {
	if a.config.MediaID == "" {
		return fmt.Errorf("收藏夹ID不能为空")
	}

	// 尝试获取收藏夹信息
	_, err := a.client.GetFavoriteInfo(a.config.MediaID)
	if err != nil {
		return fmt.Errorf("收藏夹验证失败: %w", err)
	}

	return nil
}

// matchFilter 检查视频是否匹配过滤条件
func (a *FavoriteAdapter) matchFilter(media bilibili.FavoriteMedia, opts *ScanOptions) bool {
	filter := opts.Filter
	if filter == nil && a.config.Filter == nil {
		return true
	}

	// 优先使用扫描选项中的过滤器
	if filter == nil {
		filter = a.config.Filter
	}

	// 检查时长
	if filter.MinDuration > 0 && media.Duration < filter.MinDuration {
		return false
	}
	if filter.MaxDuration > 0 && media.Duration > filter.MaxDuration {
		return false
	}

	// 检查发布时间
	pubTime := time.Unix(media.PubTime, 0)
	if !filter.MinPublishTime.IsZero() && pubTime.Before(filter.MinPublishTime) {
		return false
	}
	if !filter.MaxPublishTime.IsZero() && pubTime.After(filter.MaxPublishTime) {
		return false
	}

	// 检查关键词
	if len(filter.Keywords) > 0 {
		matched := false
		titleLower := strings.ToLower(media.Title)
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
		titleLower := strings.ToLower(media.Title)
		for _, keyword := range filter.ExcludeKeywords {
			if strings.Contains(titleLower, strings.ToLower(keyword)) {
				return false
			}
		}
	}

	// 检查是否只要新视频
	if opts.OnlyNew && !opts.LastScanTime.IsZero() {
		favTime := time.Unix(media.FavTime, 0)
		if !favTime.After(opts.LastScanTime) {
			return false
		}
	}

	return true
}

// convertToVideoInfo 转换为统一的VideoInfo格式
func (a *FavoriteAdapter) convertToVideoInfo(media bilibili.FavoriteMedia) VideoInfo {
	videoInfo := VideoInfo{
		BVid:        media.BVid,
		Aid:         media.ID,
		Title:       media.Title,
		Description: media.Intro,
		Duration:    media.Duration,
		PubDate:     time.Unix(media.PubTime, 0),
		Owner: OwnerInfo{
			Mid:  media.Upper.Mid,
			Name: media.Upper.Name,
			Face: media.Upper.Face,
		},
		Cover: media.Cover,
		Stats: StatsInfo{
			View:     media.CntInfo.Play,
			Danmaku:  media.CntInfo.Danmaku,
			Like:     media.CntInfo.ThumbUp,
			Favorite: media.CntInfo.Collect,
			Share:    media.CntInfo.Share,
		},
		SourceType: SourceTypeFavorite,
		SourceID:   a.config.MediaID,
		AddTime:    time.Unix(media.FavTime, 0),
	}

	return videoInfo
}
