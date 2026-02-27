package adapter

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"bili-download/internal/bilibili"
)

// SubmissionAdapter UP主投稿适配器
type SubmissionAdapter struct {
	client *bilibili.Client
	config *SubmissionConfig
}

// NewSubmissionAdapter 创建UP主投稿适配器
func NewSubmissionAdapter(client *bilibili.Client, config *SubmissionConfig) *SubmissionAdapter {
	return &SubmissionAdapter{
		client: client,
		config: config,
	}
}

// GetType 获取视频源类型
func (a *SubmissionAdapter) GetType() VideoSourceType {
	return SourceTypeSubmission
}

// GetID 获取视频源唯一标识
func (a *SubmissionAdapter) GetID() string {
	return a.config.Mid
}

// GetName 获取视频源名称
func (a *SubmissionAdapter) GetName() string {
	if a.config.Name != "" {
		return a.config.Name
	}

	// 从API获取UP主名称
	card, err := a.client.GetUpperCard(a.config.Mid)
	if err != nil {
		return fmt.Sprintf("UP主_%s", a.config.Mid)
	}

	return card.Name
}

// Scan 扫描视频源
func (a *SubmissionAdapter) Scan(ctx context.Context, opts *ScanOptions) ([]VideoInfo, error) {
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
		params := bilibili.SubmissionListParams{
			Mid:   a.config.Mid,
			Pn:    page,
			Ps:    pageSize,
			Order: a.config.Order,
			Tid:   a.config.Tid,
		}

		if params.Order == "" {
			params.Order = "pubdate" // 默认按发布时间排序
		}

		resp, err := a.client.GetSubmissionList(params)
		if err != nil {
			return nil, fmt.Errorf("获取UP主投稿列表失败: %w", err)
		}

		// 转换为统一的VideoInfo格式
		for _, video := range resp.List.Vlist {
			// 应用过滤条件
			if !a.matchFilter(video, opts) {
				continue
			}

			videoInfo := a.convertToVideoInfo(video)
			allVideos = append(allVideos, videoInfo)

			// 如果设置了限制且已达到，返回
			if opts.Limit > 0 && len(allVideos) >= opts.Limit {
				return allVideos, nil
			}
		}

		// 检查是否还有更多视频
		if len(resp.List.Vlist) == 0 {
			break
		}

		// 检查分页信息
		if resp.Page.Count <= page*pageSize {
			break
		}

		page++

		// 翻页间隔，防止触发B站风控
		time.Sleep(300 * time.Millisecond)
	}

	return allVideos, nil
}

// GetVideoCount 获取视频总数
func (a *SubmissionAdapter) GetVideoCount(ctx context.Context) (int, error) {
	params := bilibili.SubmissionListParams{
		Mid:   a.config.Mid,
		Pn:    1,
		Ps:    1,
		Order: a.config.Order,
		Tid:   a.config.Tid,
	}

	resp, err := a.client.GetSubmissionList(params)
	if err != nil {
		return 0, fmt.Errorf("获取UP主投稿信息失败: %w", err)
	}

	return resp.Page.Count, nil
}

// Validate 验证配置
func (a *SubmissionAdapter) Validate(ctx context.Context) error {
	if a.config.Mid == "" {
		return fmt.Errorf("UP主ID不能为空")
	}

	// 尝试获取UP主信息
	_, err := a.client.GetUpperCard(a.config.Mid)
	if err != nil {
		return fmt.Errorf("UP主验证失败: %w", err)
	}

	return nil
}

// matchFilter 检查视频是否匹配过滤条件
func (a *SubmissionAdapter) matchFilter(video bilibili.SubmissionVideo, opts *ScanOptions) bool {
	filter := opts.Filter
	if filter == nil && a.config.Filter == nil {
		return true
	}

	// 优先使用扫描选项中的过滤器
	if filter == nil {
		filter = a.config.Filter
	}

	// 解析视频时长（格式为 MM:SS 或 HH:MM:SS）
	duration := parseDuration(video.Length)

	// 检查时长
	if filter.MinDuration > 0 && duration < filter.MinDuration {
		return false
	}
	if filter.MaxDuration > 0 && duration > filter.MaxDuration {
		return false
	}

	// 检查发布时间
	pubTime := time.Unix(video.Created, 0)
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
		pubTime := time.Unix(video.Created, 0)
		if !pubTime.After(opts.LastScanTime) {
			return false
		}
	}

	return true
}

// convertToVideoInfo 转换为统一的VideoInfo格式
func (a *SubmissionAdapter) convertToVideoInfo(video bilibili.SubmissionVideo) VideoInfo {
	duration := parseDuration(video.Length)

	videoInfo := VideoInfo{
		BVid:        video.BVid,
		Aid:         video.Aid,
		Title:       video.Title,
		Description: video.Description,
		Duration:    duration,
		PubDate:     time.Unix(video.Created, 0),
		Owner: OwnerInfo{
			Mid:  video.Mid,
			Name: video.Author,
			Face: "",
		},
		Cover: video.Pic,
		Stats: StatsInfo{
			View:    video.Play,
			Danmaku: video.VideoReview,
			Reply:   video.Comment,
		},
		SourceType: SourceTypeSubmission,
		SourceID:   a.config.Mid,
		AddTime:    time.Unix(video.Created, 0),
	}

	// 获取视频详情以获取Pages信息
	if detail, err := a.client.GetVideoDetail(video.BVid); err == nil {
		// 转换Pages信息
		pages := make([]PageInfo, 0, len(detail.Pages))
		for _, p := range detail.Pages {
			pages = append(pages, PageInfo{
				CID:      p.CID,
				Page:     p.Page,
				Part:     p.Part,
				Duration: p.Duration,
				Width:    p.Dimension.Width,
				Height:   p.Dimension.Height,
			})
		}
		videoInfo.Pages = pages
	}

	return videoInfo
}

// parseDuration 解析时长字符串（MM:SS 或 HH:MM:SS）为秒数
func parseDuration(durationStr string) int {
	parts := strings.Split(durationStr, ":")
	var duration int

	switch len(parts) {
	case 2: // MM:SS
		minutes, _ := strconv.Atoi(parts[0])
		seconds, _ := strconv.Atoi(parts[1])
		duration = minutes*60 + seconds
	case 3: // HH:MM:SS
		hours, _ := strconv.Atoi(parts[0])
		minutes, _ := strconv.Atoi(parts[1])
		seconds, _ := strconv.Atoi(parts[2])
		duration = hours*3600 + minutes*60 + seconds
	}

	return duration
}
