package bilibili

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
)

// SubmissionVideo UP主投稿视频信息
type SubmissionVideo struct {
	Aid              int64      `json:"aid"`                // avid
	BVid             string     `json:"bvid"`               // bvid
	Title            string     `json:"title"`              // 标题
	Description      string     `json:"description"`        // 简介
	Pic              string     `json:"pic"`                // 封面
	Author           string     `json:"author"`             // UP主（可能是合作者）
	Mid              int64      `json:"mid"`                // UP主ID（可能是合作者）
	Created          int64      `json:"created"`            // 投稿时间（时间戳）
	Length           string     `json:"length"`             // 视频长度 MM:SS
	Comment          int        `json:"comment"`            // 评论数
	Play             int        `json:"play"`               // 播放数
	VideoReview      int        `json:"video_review"`       // 弹幕数
	Copyright        string     `json:"copyright"`          // 版权类型
	TypeID           int        `json:"typeid"`             // 分区tid
	IsUnionVideo     int        `json:"is_union_video"`     // 是否合作视频
	IsLivePlayback   int        `json:"is_live_playback"`   // 是否直播回放
	IsSteinsGate     int        `json:"is_steins_gate"`     // 是否互动视频
	IsChargingArc    bool       `json:"is_charging_arc"`    // 是否充电视频
	IsLessonVideo    int        `json:"is_lesson_video"`    // 是否课堂视频
	IsLessonFinished int        `json:"is_lesson_finished"` // 课堂是否完结
	SeasonID         int64      `json:"season_id"`          // 合集ID（0表示不属于）
	PlaybackPosition int        `json:"playback_position"`  // 播放进度百分比
	Meta             *VideoMeta `json:"meta"`               // 所属合集信息（可能为null）
}

// VideoMeta 视频所属合集/课堂信息
type VideoMeta struct {
	ID       int64     `json:"id"`        // 合集ID
	Title    string    `json:"title"`     // 合集标题
	Cover    string    `json:"cover"`     // 合集封面
	Mid      int64     `json:"mid"`       // UP主ID（课堂为0）
	Intro    string    `json:"intro"`     // 合集介绍
	EpCount  int       `json:"ep_count"`  // 视频数量
	FirstAid int64     `json:"first_aid"` // 首个视频aid
	PTime    int64     `json:"ptime"`     // 最后更新时间
	Stat     *MetaStat `json:"stat"`      // 统计信息
}

// MetaStat 合集统计信息
type MetaStat struct {
	SeasonID int64 `json:"season_id"` // 合集ID
	View     int   `json:"view"`      // 播放量
	Danmaku  int   `json:"danmaku"`   // 弹幕数
	Reply    int   `json:"reply"`     // 评论数
	Favorite int   `json:"favorite"`  // 收藏数
	Coin     int   `json:"coin"`      // 投币数
	Share    int   `json:"share"`     // 分享数
	Like     int   `json:"like"`      // 点赞数
}

// SubmissionListResponse UP主投稿列表响应
type SubmissionListResponse struct {
	List SubmissionList `json:"list"` // 列表信息
	Page SubmissionPage `json:"page"` // 分页信息
}

// SubmissionList 投稿列表信息
type SubmissionList struct {
	Tlist map[string]TidInfo `json:"tlist"` // 分区索引
	Vlist []SubmissionVideo  `json:"vlist"` // 视频列表
}

// TidInfo 分区信息
type TidInfo struct {
	Tid   int    `json:"tid"`   // 分区tid
	Count int    `json:"count"` // 该分区视频数
	Name  string `json:"name"`  // 分区名称
}

// SubmissionPage 投稿分页信息
type SubmissionPage struct {
	Count int `json:"count"` // 总视频数
	Pn    int `json:"pn"`    // 当前页码
	Ps    int `json:"ps"`    // 每页数量
}

// SubmissionListParams 获取UP主投稿列表参数
type SubmissionListParams struct {
	Mid     string // UP主ID（必需）
	Pn      int    // 页码（默认1）
	Ps      int    // 每页数量（默认30）
	Order   string // 排序方式：pubdate-最新发布，click-最多播放，stow-最多收藏
	Tid     int    // 筛选分区（0-不筛选）
	Keyword string // 关键词筛选
}

// GetSubmissionList 获取UP主投稿视频列表（需要WBI签名）
func (c *Client) GetSubmissionList(params SubmissionListParams) (*SubmissionListResponse, error) {
	// 构建查询参数
	query := url.Values{}
	query.Set("mid", params.Mid)

	// 设置分页参数
	if params.Pn <= 0 {
		params.Pn = 1
	}
	query.Set("pn", strconv.Itoa(params.Pn))

	if params.Ps <= 0 {
		params.Ps = 30
	}
	query.Set("ps", strconv.Itoa(params.Ps))

	// 设置排序方式
	if params.Order == "" {
		params.Order = "pubdate"
	}
	query.Set("order", params.Order)

	// 设置分区筛选
	query.Set("tid", strconv.Itoa(params.Tid))

	// 可选参数
	if params.Keyword != "" {
		query.Set("keyword", params.Keyword)
	}

	// 添加其他固定参数
	query.Set("platform", "web")
	query.Set("web_location", "1550101")

	// 进行 WBI 签名
	signedParams, err := c.GetWbiSignedParams(query)
	if err != nil {
		return nil, fmt.Errorf("WBI签名失败: %w", err)
	}

	// 构建完整URL
	apiURL := "https://api.bilibili.com/x/space/wbi/arc/search?" + signedParams.Encode()

	resp, err := c.Get(apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("获取UP主投稿列表失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var result struct {
		Code    int                    `json:"code"`
		Message string                 `json:"message"`
		Data    SubmissionListResponse `json:"data"`
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

// GetAllSubmissionVideos 获取UP主所有投稿视频（自动翻页）
func (c *Client) GetAllSubmissionVideos(mid string, order string) ([]SubmissionVideo, error) {
	var allVideos []SubmissionVideo
	page := 1
	pageSize := 30

	for {
		params := SubmissionListParams{
			Mid:   mid,
			Pn:    page,
			Ps:    pageSize,
			Order: order,
		}

		listResp, err := c.GetSubmissionList(params)
		if err != nil {
			return nil, fmt.Errorf("获取第 %d 页失败: %w", page, err)
		}

		// 如果没有视频，说明到达末尾
		if listResp.List.Vlist == nil || len(listResp.List.Vlist) == 0 {
			break
		}

		allVideos = append(allVideos, listResp.List.Vlist...)

		// 检查是否还有更多
		if listResp.Page.Count <= page*pageSize {
			break
		}

		page++
	}

	return allVideos, nil
}

// GetUpperCard 获取UP主名片信息
func (c *Client) GetUpperCard(mid string) (*UpperCardInfo, error) {
	apiURL := fmt.Sprintf("https://api.bilibili.com/x/web-interface/card?mid=%s", mid)

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Card UpperCardInfo `json:"card"`
		} `json:"data"`
	}

	err := c.GetJSON(apiURL, nil, &result)
	if err != nil {
		return nil, fmt.Errorf("获取UP主信息失败: %w", err)
	}

	if result.Code != 0 {
		return nil, &BiliError{
			Code:    result.Code,
			Message: result.Message,
		}
	}

	return &result.Data.Card, nil
}

// UpperCardInfo UP主名片信息
type UpperCardInfo struct {
	Mid       string  `json:"mid"`       // UP主ID
	Name      string  `json:"name"`      // 昵称
	Sex       string  `json:"sex"`       // 性别
	Face      string  `json:"face"`      // 头像URL
	Sign      string  `json:"sign"`      // 签名
	Rank      int     `json:"rank"`      // 等级
	Level     int     `json:"level"`     // 等级
	Jointime  int64   `json:"jointime"`  // 注册时间
	Moral     int     `json:"moral"`     // 节操
	Silence   int     `json:"silence"`   // 封禁状态
	Coins     int     `json:"coins"`     // 硬币数
	Birthday  string  `json:"birthday"`  // 生日
	Fans      int     `json:"fans"`      // 粉丝数
	Friend    int     `json:"friend"`    // 关注数
	Attention int     `json:"attention"` // 关注数
	Vip       VipInfo `json:"vip"`       // 大会员信息
}

// VipInfo 大会员信息
type VipInfo struct {
	Type       int      `json:"type"`         // 会员类型：0-无，1-月度，2-年度
	Status     int      `json:"status"`       // 会员状态：0-无，1-有
	DueDate    int64    `json:"due_date"`     // 到期时间
	VipPayType int      `json:"vip_pay_type"` // 支付类型
	ThemeType  int      `json:"theme_type"`   // 主题类型
	Label      VipLabel `json:"label"`        // 标签
}

// VipLabel 会员标签
type VipLabel struct {
	Path       string `json:"path"`        // 路径
	Text       string `json:"text"`        // 文本
	LabelTheme string `json:"label_theme"` // 标签主题
}
