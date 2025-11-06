package bilibili

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
)

// CollectionType 合集类型
type CollectionType int

const (
	// CollectionTypeSeries 视频列表/系列
	CollectionTypeSeries CollectionType = 1
	// CollectionTypeSeason 合集
	CollectionTypeSeason CollectionType = 2
)

func (c CollectionType) String() string {
	switch c {
	case CollectionTypeSeries:
		return "列表"
	case CollectionTypeSeason:
		return "合集"
	default:
		return "未知"
	}
}

// CollectionInfo 合集/系列信息
type CollectionInfo struct {
	Name           string         `json:"name"`        // 名称
	Mid            int64          `json:"mid"`         // UP主ID
	SID            int64          `json:"-"`           // ID (season_id 或 series_id)
	SeasonID       int64          `json:"season_id"`   // 合集ID
	SeriesID       int64          `json:"series_id"`   // 系列ID
	Cover          string         `json:"cover"`       // 封面
	Description    string         `json:"description"` // 描述
	Total          int            `json:"total"`       // 视频总数
	PTime          int64          `json:"ptime"`       // 发布时间（合集）
	CTime          int64          `json:"ctime"`       // 创建时间（系列）
	CollectionType CollectionType `json:"-"`           // 类型
}

// CollectionArchive 合集/系列中的视频
type CollectionArchive struct {
	Aid              int64  `json:"aid"`               // avid
	BVid             string `json:"bvid"`              // bvid
	CTime            int64  `json:"ctime"`             // 创建时间
	PubDate          int64  `json:"pubdate"`           // 发布时间
	Duration         int    `json:"duration"`          // 时长（秒）
	Title            string `json:"title"`             // 标题
	Pic              string `json:"pic"`               // 封面
	InteractiveVideo bool   `json:"interactive_video"` // 是否互动视频
	PlaybackPosition int    `json:"playback_position"` // 播放进度百分比
	State            int    `json:"state"`             // 状态
	UGCPay           int    `json:"ugc_pay"`           // UGC付费

	Stat CollectionStat `json:"stat"` // 统计信息
}

// CollectionStat 合集视频统计信息
type CollectionStat struct {
	View int `json:"view"` // 播放量
	VT   int `json:"vt"`   // VT值
}

// CollectionListResponse 合集视频列表响应
type CollectionListResponse struct {
	Aids     []int64             `json:"aids"`     // avid列表
	Archives []CollectionArchive `json:"archives"` // 视频列表
	Meta     CollectionInfo      `json:"meta"`     // 合集信息
	Page     PaginationInfo      `json:"page"`     // 分页信息
}

// PaginationInfo 分页信息
type PaginationInfo struct {
	PageNum  int `json:"page_num"`  // 页码（合集）
	PageSize int `json:"page_size"` // 页大小（合集）
	Num      int `json:"num"`       // 页码（系列）
	Size     int `json:"size"`      // 页大小（系列）
	Total    int `json:"total"`     // 总数
}

// CollectionListParams 获取合集视频列表参数
type CollectionListParams struct {
	Mid            string         // UP主ID（必需）
	SeasonID       string         // 合集ID（Season时必需）
	SeriesID       string         // 系列ID（Series时必需）
	PageNum        int            // 页码（默认1）
	PageSize       int            // 每页数量（默认30，最大30）
	SortReverse    bool           // 是否倒序（仅合集）
	Sort           string         // 排序方式（仅系列）：desc-倒序，asc-正序
	OnlyNormal     bool           // 是否只显示正常视频（仅系列）
	CollectionType CollectionType // 合集类型
}

// GetCollectionInfo 获取合集/系列信息
func (c *Client) GetCollectionInfo(params CollectionListParams) (*CollectionInfo, error) {
	// 对于 Season，我们需要先获取第一页来获取 meta 信息
	// 对于 Series，有专门的接口
	if params.CollectionType == CollectionTypeSeries {
		return c.getSeriesInfo(params.SeriesID)
	}

	// Season 类型，获取第一页来提取 meta
	params.PageNum = 1
	params.PageSize = 1
	resp, err := c.GetCollectionList(params)
	if err != nil {
		return nil, err
	}

	return &resp.Meta, nil
}

// getSeriesInfo 获取系列信息
func (c *Client) getSeriesInfo(seriesID string) (*CollectionInfo, error) {
	params := url.Values{}
	params.Set("series_id", seriesID)

	apiURL := "https://api.bilibili.com/x/series/series?" + params.Encode()

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Meta CollectionInfo `json:"meta"`
		} `json:"data"`
	}

	err := c.GetJSON(apiURL, nil, &result)
	if err != nil {
		return nil, fmt.Errorf("获取系列信息失败: %w", err)
	}

	if result.Code != 0 {
		return nil, &BiliError{
			Code:    result.Code,
			Message: result.Message,
		}
	}

	info := result.Data.Meta
	info.CollectionType = CollectionTypeSeries
	info.SID = info.SeriesID

	return &info, nil
}

// GetCollectionList 获取合集/系列视频列表
func (c *Client) GetCollectionList(params CollectionListParams) (*CollectionListResponse, error) {
	var apiURL string
	query := url.Values{}

	// 设置分页参数
	if params.PageNum <= 0 {
		params.PageNum = 1
	}
	if params.PageSize <= 0 || params.PageSize > 30 {
		params.PageSize = 30
	}

	// 根据类型构建不同的请求
	if params.CollectionType == CollectionTypeSeries {
		// 系列/列表
		apiURL = "https://api.bilibili.com/x/series/archives"
		query.Set("mid", params.Mid)
		query.Set("series_id", params.SeriesID)
		query.Set("pn", strconv.Itoa(params.PageNum))
		query.Set("ps", strconv.Itoa(params.PageSize))
		query.Set("only_normal", "true")
		if params.Sort == "" {
			params.Sort = "desc"
		}
		query.Set("sort", params.Sort)
	} else {
		// 合集
		apiURL = "https://api.bilibili.com/x/polymer/web-space/seasons_archives_list"
		query.Set("mid", params.Mid)
		query.Set("season_id", params.SeasonID)
		query.Set("page_num", strconv.Itoa(params.PageNum))
		query.Set("page_size", strconv.Itoa(params.PageSize))
		if params.SortReverse {
			query.Set("sort_reverse", "true")
		} else {
			query.Set("sort_reverse", "false")
		}
	}

	fullURL := apiURL + "?" + query.Encode()

	resp, err := c.Get(fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("获取合集列表失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var result struct {
		Code    int                    `json:"code"`
		Message string                 `json:"message"`
		Data    CollectionListResponse `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if result.Code != 0 {
		return nil, &BiliError{
			Code:    result.Code,
			Message: result.Message,
		}
	}

	// 设置 meta 中的类型和 SID
	result.Data.Meta.CollectionType = params.CollectionType
	if params.CollectionType == CollectionTypeSeries {
		result.Data.Meta.SID = result.Data.Meta.SeriesID
	} else {
		result.Data.Meta.SID = result.Data.Meta.SeasonID
	}

	return &result.Data, nil
}

// GetAllCollectionVideos 获取合集/系列所有视频（自动翻页）
func (c *Client) GetAllCollectionVideos(params CollectionListParams) ([]CollectionArchive, error) {
	var allVideos []CollectionArchive
	page := 1

	for {
		params.PageNum = page
		listResp, err := c.GetCollectionList(params)
		if err != nil {
			return nil, fmt.Errorf("获取第 %d 页失败: %w", page, err)
		}

		// 如果没有视频，说明到达末尾
		if listResp.Archives == nil || len(listResp.Archives) == 0 {
			break
		}

		allVideos = append(allVideos, listResp.Archives...)

		// 检查是否还有更多
		pageInfo := listResp.Page
		var currentPage, pageSize, total int

		if params.CollectionType == CollectionTypeSeries {
			currentPage = pageInfo.Num
			pageSize = pageInfo.Size
			total = pageInfo.Total
		} else {
			currentPage = pageInfo.PageNum
			pageSize = pageInfo.PageSize
			total = pageInfo.Total
		}

		// 如果已经获取了所有视频
		if currentPage*pageSize >= total {
			break
		}

		page++
	}

	return allVideos, nil
}
