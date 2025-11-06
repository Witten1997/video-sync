package bilibili

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
)

// FavoriteInfo 收藏夹元数据
type FavoriteInfo struct {
	ID         int64  `json:"id"`          // 收藏夹 mlid（完整id）
	FID        int64  `json:"fid"`         // 收藏夹原始id
	Mid        int64  `json:"mid"`         // 创建者mid
	Title      string `json:"title"`       // 收藏夹标题
	Cover      string `json:"cover"`       // 封面图片url
	Intro      string `json:"intro"`       // 简介
	MediaCount int    `json:"media_count"` // 内容数量
	CTime      int64  `json:"ctime"`       // 创建时间（时间戳）
	MTime      int64  `json:"mtime"`       // 修改时间（时间戳）
	State      int    `json:"state"`       // 状态
	FavState   int    `json:"fav_state"`   // 收藏状态
	LikeState  int    `json:"like_state"`  // 点赞状态
	Attr       int    `json:"attr"`        // 属性：0-正常，1-失效

	Upper   Upper   `json:"upper"`    // 创建者信息
	CntInfo CntInfo `json:"cnt_info"` // 状态数
}

// CntInfo 状态计数
type CntInfo struct {
	Collect int `json:"collect"`  // 收藏数
	Play    int `json:"play"`     // 播放数
	ThumbUp int `json:"thumb_up"` // 点赞数
	Share   int `json:"share"`    // 分享数
	Danmaku int `json:"danmaku"`  // 弹幕数
}

// FavoriteMedia 收藏夹内容
type FavoriteMedia struct {
	ID       int64  `json:"id"`       // 内容id
	Type     int    `json:"type"`     // 内容类型：2-视频，12-音频，21-合集
	Title    string `json:"title"`    // 标题
	Cover    string `json:"cover"`    // 封面url
	Intro    string `json:"intro"`    // 简介
	Page     int    `json:"page"`     // 视频分P数
	Duration int    `json:"duration"` // 时长（秒）
	Link     string `json:"link"`     // 跳转uri
	CTime    int64  `json:"ctime"`    // 投稿时间（时间戳）
	PubTime  int64  `json:"pubtime"`  // 发布时间（时间戳）
	FavTime  int64  `json:"fav_time"` // 收藏时间（时间戳）
	BVid     string `json:"bvid"`     // 视频bvid
	Attr     int    `json:"attr"`     // 失效状态：0-正常，9-UP主删除，1-其他原因删除

	Upper   Upper   `json:"upper"`    // UP主信息
	CntInfo CntInfo `json:"cnt_info"` // 状态数
}

// FavoriteListResponse 收藏夹内容列表响应
type FavoriteListResponse struct {
	Info    FavoriteInfo    `json:"info"`     // 收藏夹信息
	Medias  []FavoriteMedia `json:"medias"`   // 内容列表
	HasMore bool            `json:"has_more"` // 是否有下一页
}

// FavoriteListParams 获取收藏夹列表参数
type FavoriteListParams struct {
	MediaID  string // 收藏夹ID（必需）
	Pn       int    // 页码（默认1）
	Ps       int    // 每页数量（1-20，必需）
	Keyword  string // 搜索关键字
	Order    string // 排序方式：mtime-收藏时间，view-播放量，pubtime-投稿时间
	Type     int    // 查询范围：0-当前收藏夹，1-全部收藏夹
	Tid      int    // 分区tid
	Platform string // 平台标识：web
}

// GetFavoriteInfo 获取收藏夹元数据
func (c *Client) GetFavoriteInfo(mediaID string) (*FavoriteInfo, error) {
	type Response struct {
		Code    int          `json:"code"`
		Message string       `json:"message"`
		Data    FavoriteInfo `json:"data"`
	}

	// 构建查询参数
	params := url.Values{}
	params.Set("media_id", mediaID)

	apiURL := "https://api.bilibili.com/x/v3/fav/folder/info?" + params.Encode()

	var resp Response
	err := c.GetJSON(apiURL, nil, &resp)
	if err != nil {
		return nil, fmt.Errorf("获取收藏夹信息失败: %w", err)
	}

	if resp.Code != 0 {
		return nil, &BiliError{
			Code:    resp.Code,
			Message: resp.Message,
		}
	}

	return &resp.Data, nil
}

// GetFavoriteList 获取收藏夹内容列表
func (c *Client) GetFavoriteList(params FavoriteListParams) (*FavoriteListResponse, error) {
	// 构建查询参数
	query := url.Values{}
	query.Set("media_id", params.MediaID)

	// 设置分页参数
	if params.Pn <= 0 {
		params.Pn = 1
	}
	query.Set("pn", strconv.Itoa(params.Pn))

	if params.Ps <= 0 || params.Ps > 20 {
		params.Ps = 20
	}
	query.Set("ps", strconv.Itoa(params.Ps))

	// 设置排序方式（默认按收藏时间）
	if params.Order == "" {
		params.Order = "mtime"
	}
	query.Set("order", params.Order)

	// 设置查询类型
	query.Set("type", strconv.Itoa(params.Type))

	// 设置分区
	query.Set("tid", strconv.Itoa(params.Tid))

	// 可选参数
	if params.Keyword != "" {
		query.Set("keyword", params.Keyword)
	}
	if params.Platform != "" {
		query.Set("platform", params.Platform)
	}

	// 构建URL
	apiURL := "https://api.bilibili.com/x/v3/fav/resource/list?" + query.Encode()

	// 发送请求
	resp, err := c.Get(apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("获取收藏夹列表失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析响应
	var result struct {
		Code    int                  `json:"code"`
		Message string               `json:"message"`
		Data    FavoriteListResponse `json:"data"`
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

	return &result.Data, nil
}

// GetAllFavoriteVideos 获取收藏夹所有视频（自动翻页）
func (c *Client) GetAllFavoriteVideos(mediaID string) ([]FavoriteMedia, error) {
	var allVideos []FavoriteMedia
	page := 1
	pageSize := 20

	for {
		params := FavoriteListParams{
			MediaID: mediaID,
			Pn:      page,
			Ps:      pageSize,
			Order:   "mtime", // 按收藏时间排序
		}

		listResp, err := c.GetFavoriteList(params)
		if err != nil {
			return nil, fmt.Errorf("获取第 %d 页失败: %w", page, err)
		}

		// 如果没有内容，说明到达末尾或收藏夹为空
		if listResp.Medias == nil || len(listResp.Medias) == 0 {
			break
		}

		allVideos = append(allVideos, listResp.Medias...)

		// 检查是否还有更多
		if !listResp.HasMore {
			break
		}

		page++
	}

	return allVideos, nil
}

// GetFavoriteIDs 获取收藏夹所有内容ID
func (c *Client) GetFavoriteIDs(mediaID string) ([]struct {
	ID   int64  `json:"id"`
	Type int    `json:"type"`
	BVid string `json:"bvid"`
}, error) {
	type Response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    []struct {
			ID   int64  `json:"id"`
			Type int    `json:"type"`
			BVid string `json:"bvid"`
		} `json:"data"`
	}

	params := url.Values{}
	params.Set("media_id", mediaID)

	apiURL := "https://api.bilibili.com/x/v3/fav/resource/ids?" + params.Encode()

	var resp Response
	err := c.GetJSON(apiURL, nil, &resp)
	if err != nil {
		return nil, fmt.Errorf("获取收藏夹ID列表失败: %w", err)
	}

	if resp.Code != 0 {
		return nil, &BiliError{
			Code:    resp.Code,
			Message: resp.Message,
		}
	}

	return resp.Data, nil
}
