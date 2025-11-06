package bilibili

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// SourceType 视频源类型
type SourceType string

const (
	SourceTypeFavorite   SourceType = "favorite"
	SourceTypeWatchLater SourceType = "watch_later"
	SourceTypeCollection SourceType = "collection"
	SourceTypeSubmission SourceType = "submission"
)

// ParsedURL 解析后的 URL 信息
type ParsedURL struct {
	Type    SourceType
	ID      int64  // FID, CID, UpperID 等
	Name    string // 可选的名称
	SubType string // 合集子类型 (series/season)
}

// URLParser URL 解析器
type URLParser struct{}

// NewURLParser 创建 URL 解析器
func NewURLParser() *URLParser {
	return &URLParser{}
}

// Parse 解析 B 站 URL
func (p *URLParser) Parse(rawURL string) (*ParsedURL, error) {
	// 清理 URL
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, fmt.Errorf("URL 不能为空")
	}

	// 解析 URL
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("无效的 URL: %w", err)
	}

	// 检查是否是 B 站域名
	if !strings.Contains(u.Host, "bilibili.com") {
		return nil, fmt.Errorf("不支持的域名: %s", u.Host)
	}

	// 根据路径和参数判断类型
	path := u.Path
	query := u.Query()

	// 收藏夹: https://space.bilibili.com/xxx/favlist?fid=xxx
	if strings.Contains(path, "/favlist") {
		fidStr := query.Get("fid")
		if fidStr == "" {
			return nil, fmt.Errorf("收藏夹 URL 缺少 fid 参数")
		}
		fid, err := strconv.ParseInt(fidStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("无效的收藏夹 ID: %s", fidStr)
		}
		return &ParsedURL{
			Type: SourceTypeFavorite,
			ID:   fid,
		}, nil
	}

	// 稍后再看: https://www.bilibili.com/watchlater/
	if strings.Contains(path, "/watchlater") {
		return &ParsedURL{
			Type: SourceTypeWatchLater,
			ID:   0, // 稍后再看没有 ID
		}, nil
	}

	// 合集 (新版): https://space.bilibili.com/xxx/channel/collectiondetail?sid=xxx
	if strings.Contains(path, "/channel/collectiondetail") {
		sidStr := query.Get("sid")
		if sidStr == "" {
			return nil, fmt.Errorf("合集 URL 缺少 sid 参数")
		}
		sid, err := strconv.ParseInt(sidStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("无效的合集 ID: %s", sidStr)
		}
		return &ParsedURL{
			Type:    SourceTypeCollection,
			ID:      sid,
			SubType: "series",
		}, nil
	}

	// 合集 (旧版): https://space.bilibili.com/xxx/channel/seriesdetail?sid=xxx
	if strings.Contains(path, "/channel/seriesdetail") {
		sidStr := query.Get("sid")
		if sidStr == "" {
			return nil, fmt.Errorf("合集 URL 缺少 sid 参数")
		}
		sid, err := strconv.ParseInt(sidStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("无效的合集 ID: %s", sidStr)
		}
		return &ParsedURL{
			Type:    SourceTypeCollection,
			ID:      sid,
			SubType: "series",
		}, nil
	}

	// 番剧/影视合集: https://www.bilibili.com/bangumi/play/ss12345
	if strings.Contains(path, "/bangumi/play/ss") {
		re := regexp.MustCompile(`/bangumi/play/ss(\d+)`)
		matches := re.FindStringSubmatch(path)
		if len(matches) < 2 {
			return nil, fmt.Errorf("无效的番剧 URL")
		}
		ssid, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("无效的番剧 ID: %s", matches[1])
		}
		return &ParsedURL{
			Type:    SourceTypeCollection,
			ID:      ssid,
			SubType: "season",
		}, nil
	}

	// UP 主投稿: https://space.bilibili.com/xxx 或 https://space.bilibili.com/xxx/video
	if strings.HasPrefix(path, "/") {
		// 提取 UP 主 ID
		re := regexp.MustCompile(`^/(\d+)(?:/video)?/?$`)
		matches := re.FindStringSubmatch(path)
		if len(matches) >= 2 {
			upperID, err := strconv.ParseInt(matches[1], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("无效的 UP 主 ID: %s", matches[1])
			}
			return &ParsedURL{
				Type: SourceTypeSubmission,
				ID:   upperID,
			}, nil
		}
	}

	return nil, fmt.Errorf("无法识别的 B 站 URL 类型")
}

// ParseMultiple 解析多个 URL
func (p *URLParser) ParseMultiple(urls []string) ([]*ParsedURL, []error) {
	var results []*ParsedURL
	var errors []error

	for _, rawURL := range urls {
		parsed, err := p.Parse(rawURL)
		if err != nil {
			errors = append(errors, fmt.Errorf("解析 %s 失败: %w", rawURL, err))
			continue
		}
		results = append(results, parsed)
	}

	return results, errors
}
