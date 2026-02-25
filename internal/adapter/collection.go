package adapter

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"bili-download/internal/bilibili"
)

// CollectionAdapter 合集/系列适配器
type CollectionAdapter struct {
	client *bilibili.Client
	config *CollectionConfig
}

// NewCollectionAdapter 创建合集适配器
func NewCollectionAdapter(client *bilibili.Client, config *CollectionConfig) *CollectionAdapter {
	return &CollectionAdapter{
		client: client,
		config: config,
	}
}

// GetType 获取视频源类型
func (a *CollectionAdapter) GetType() VideoSourceType {
	return SourceTypeCollection
}

// GetID 获取视频源唯一标识
func (a *CollectionAdapter) GetID() string {
	if a.config.CollectionType == "season" {
		return fmt.Sprintf("season_%s", a.config.SeasonID)
	}
	return fmt.Sprintf("series_%s", a.config.SeriesID)
}

// GetName 获取视频源名称
func (a *CollectionAdapter) GetName() string {
	if a.config.Name != "" {
		return a.config.Name
	}

	// 从API获取名称
	params := a.buildListParams()
	info, err := a.client.GetCollectionInfo(params)
	if err != nil {
		if a.config.CollectionType == "season" {
			return fmt.Sprintf("合集_%s", a.config.SeasonID)
		}
		return fmt.Sprintf("系列_%s", a.config.SeriesID)
	}

	return info.Name
}

// Scan 扫描视频源
func (a *CollectionAdapter) Scan(ctx context.Context, opts *ScanOptions) ([]VideoInfo, error) {
	if opts == nil {
		opts = &ScanOptions{}
	}

	var allVideos []VideoInfo
	page := 1
	pageSize := 30

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

		// 构建请求参数
		params := a.buildListParams()
		params.PageNum = page
		params.PageSize = pageSize

		resp, err := a.client.GetCollectionList(params)
		if err != nil {
			return nil, fmt.Errorf("获取合集列表失败: %w", err)
		}

		// 转换为统一的VideoInfo格式
		for _, archive := range resp.Archives {
			// 应用过滤条件
			if !a.matchFilter(archive, opts) {
				continue
			}

			videoInfo := a.convertToVideoInfo(archive)
			allVideos = append(allVideos, videoInfo)

			// 如果设置了限制且已达到，返回
			if opts.Limit > 0 && len(allVideos) >= opts.Limit {
				return allVideos, nil
			}
		}

		// 检查是否还有更多视频
		if len(resp.Archives) == 0 {
			break
		}

		// 检查分页信息
		var currentPage, pageSize, total int
		if params.CollectionType == bilibili.CollectionTypeSeries {
			currentPage = resp.Page.Num
			pageSize = resp.Page.Size
			total = resp.Page.Total
		} else {
			currentPage = resp.Page.PageNum
			pageSize = resp.Page.PageSize
			total = resp.Page.Total
		}

		if currentPage*pageSize >= total {
			break
		}

		page++

		// 翻页间隔，防止触发B站风控
		time.Sleep(300 * time.Millisecond)
	}

	return allVideos, nil
}

// GetVideoCount 获取视频总数
func (a *CollectionAdapter) GetVideoCount(ctx context.Context) (int, error) {
	params := a.buildListParams()
	info, err := a.client.GetCollectionInfo(params)
	if err != nil {
		return 0, fmt.Errorf("获取合集信息失败: %w", err)
	}
	return info.Total, nil
}

// Validate 验证配置
func (a *CollectionAdapter) Validate(ctx context.Context) error {
	if a.config.Mid == "" {
		return fmt.Errorf("UP主ID不能为空")
	}

	if a.config.CollectionType == "season" {
		if a.config.SeasonID == "" {
			return fmt.Errorf("合集ID不能为空")
		}
	} else if a.config.CollectionType == "series" {
		if a.config.SeriesID == "" {
			return fmt.Errorf("系列ID不能为空")
		}
	} else {
		return fmt.Errorf("合集类型不正确，必须是 season 或 series")
	}

	// 尝试获取合集信息
	params := a.buildListParams()
	_, err := a.client.GetCollectionInfo(params)
	if err != nil {
		return fmt.Errorf("合集验证失败: %w", err)
	}

	return nil
}

// buildListParams 构建API请求参数
func (a *CollectionAdapter) buildListParams() bilibili.CollectionListParams {
	params := bilibili.CollectionListParams{
		Mid:         a.config.Mid,
		SortReverse: a.config.SortReverse,
	}

	if a.config.CollectionType == "season" {
		params.SeasonID = a.config.SeasonID
		params.CollectionType = bilibili.CollectionTypeSeason
	} else {
		params.SeriesID = a.config.SeriesID
		params.CollectionType = bilibili.CollectionTypeSeries
		// 系列默认按发布时间倒序
		params.Sort = "desc"
	}

	return params
}

// matchFilter 检查视频是否匹配过滤条件
func (a *CollectionAdapter) matchFilter(archive bilibili.CollectionArchive, opts *ScanOptions) bool {
	filter := opts.Filter
	if filter == nil && a.config.Filter == nil {
		return true
	}

	// 优先使用扫描选项中的过滤器
	if filter == nil {
		filter = a.config.Filter
	}

	// 检查时长
	if filter.MinDuration > 0 && archive.Duration < filter.MinDuration {
		return false
	}
	if filter.MaxDuration > 0 && archive.Duration > filter.MaxDuration {
		return false
	}

	// 检查发布时间
	pubTime := time.Unix(archive.PubDate, 0)
	if !filter.MinPublishTime.IsZero() && pubTime.Before(filter.MinPublishTime) {
		return false
	}
	if !filter.MaxPublishTime.IsZero() && pubTime.After(filter.MaxPublishTime) {
		return false
	}

	// 检查关键词
	if len(filter.Keywords) > 0 {
		matched := false
		titleLower := strings.ToLower(archive.Title)
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
		titleLower := strings.ToLower(archive.Title)
		for _, keyword := range filter.ExcludeKeywords {
			if strings.Contains(titleLower, strings.ToLower(keyword)) {
				return false
			}
		}
	}

	// 检查是否只要新视频
	if opts.OnlyNew && !opts.LastScanTime.IsZero() {
		pubTime := time.Unix(archive.PubDate, 0)
		if !pubTime.After(opts.LastScanTime) {
			return false
		}
	}

	return true
}

// convertToVideoInfo 转换为统一的VideoInfo格式
func (a *CollectionAdapter) convertToVideoInfo(archive bilibili.CollectionArchive) VideoInfo {
	// 解析UP主ID
	midInt, _ := strconv.ParseInt(a.config.Mid, 10, 64)

	return VideoInfo{
		BVid:        archive.BVid,
		Aid:         archive.Aid,
		Title:       archive.Title,
		Description: "",
		Duration:    archive.Duration,
		PubDate:     time.Unix(archive.PubDate, 0),
		Owner: OwnerInfo{
			Mid:  midInt,
			Name: "", // 需要从UP主信息获取
			Face: "",
		},
		Cover: archive.Pic,
		Stats: StatsInfo{
			View: archive.Stat.View,
		},
		SourceType: SourceTypeCollection,
		SourceID:   a.GetID(),
		AddTime:    time.Unix(archive.PubDate, 0),
	}
}
