package bilibili

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// WatchLaterVideo 稍后再看视频信息
type WatchLaterVideo struct {
	Aid       int64  `json:"aid"`       // 稿件avid
	BVid      string `json:"bvid"`      // 稿件bvid
	Videos    int    `json:"videos"`    // 稿件分P总数
	Tid       int    `json:"tid"`       // 分区tid
	TName     string `json:"tname"`     // 子分区名称
	Copyright int    `json:"copyright"` // 是否转载：1-原创，2-转载
	Pic       string `json:"pic"`       // 封面图片url
	Title     string `json:"title"`     // 标题
	PubDate   int64  `json:"pubdate"`   // 发布时间（时间戳）
	CTime     int64  `json:"ctime"`     // 创建时间（时间戳）
	Desc      string `json:"desc"`      // 简介
	State     int    `json:"state"`     // 视频状态
	Duration  int    `json:"duration"`  // 总时长（秒）
	Dynamic   string `json:"dynamic"`   // 动态文字内容
	CID       int64  `json:"cid"`       // 视频cid
	Progress  int    `json:"progress"`  // 观看进度（秒）
	AddAt     int64  `json:"add_at"`    // 添加时间（时间戳）

	Owner     Owner     `json:"owner"`     // UP主信息
	Stat      Stat      `json:"stat"`      // 状态数
	Dimension Dimension `json:"dimension"` // 分辨率
	Rights    Rights    `json:"rights"`    // 属性标志
}

// Stat 视频状态数
type Stat struct {
	Aid      int64 `json:"aid"`      // 稿件avid
	View     int   `json:"view"`     // 播放数
	Danmaku  int   `json:"danmaku"`  // 弹幕数
	Reply    int   `json:"reply"`    // 评论数
	Favorite int   `json:"favorite"` // 收藏数
	Coin     int   `json:"coin"`     // 投币数
	Share    int   `json:"share"`    // 分享数
	NowRank  int   `json:"now_rank"` // 当前排名
	HisRank  int   `json:"his_rank"` // 历史最高排名
	Like     int   `json:"like"`     // 点赞数
	Dislike  int   `json:"dislike"`  // 点踩数（已废弃）
}

// Owner UP主信息
type Owner struct {
	Mid  int64  `json:"mid"`  // UP主mid
	Name string `json:"name"` // UP主昵称
	Face string `json:"face"` // UP主头像url
}

// Rights 视频属性标志
type Rights struct {
	BP            int `json:"bp"`              // 是否允许承包
	Elec          int `json:"elec"`            // 是否支持充电
	Download      int `json:"download"`        // 是否允许下载
	Movie         int `json:"movie"`           // 是否电影
	Pay           int `json:"pay"`             // 是否PGC付费
	HD5           int `json:"hd5"`             // 是否有高码率
	NoReprint     int `json:"no_reprint"`      // 是否禁止转载
	Autoplay      int `json:"autoplay"`        // 是否自动播放
	UGCPay        int `json:"ugc_pay"`         // 是否UGC付费
	IsCooperation int `json:"is_cooperation"`  // 是否合作视频
	UGCPayPreview int `json:"ugc_pay_preview"` // 是否UGC付费预览
	NoBackground  int `json:"no_background"`   // 是否禁止背景播放
}

// WatchLaterListResponse 稍后再看列表响应
type WatchLaterListResponse struct {
	Count int               `json:"count"` // 稍后再看视频数
	List  []WatchLaterVideo `json:"list"`  // 视频列表
}

// GetWatchLaterList 获取稍后再看视频列表
func (c *Client) GetWatchLaterList() (*WatchLaterListResponse, error) {
	resp, err := c.Get("https://api.bilibili.com/x/v2/history/toview", nil)
	if err != nil {
		return nil, fmt.Errorf("获取稍后再看列表失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var result struct {
		Code    int                    `json:"code"`
		Message string                 `json:"message"`
		Data    WatchLaterListResponse `json:"data"`
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

// AddToWatchLater 添加视频到稍后再看
func (c *Client) AddToWatchLater(bvid string) error {
	if c.credential == nil || c.credential.BiliJct == "" {
		return fmt.Errorf("需要登录凭据（bili_jct）")
	}

	// 构建表单数据
	formData := fmt.Sprintf("bvid=%s&csrf=%s", bvid, c.credential.BiliJct)

	resp, err := c.Post(
		"https://api.bilibili.com/x/v2/history/toview/add",
		map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		},
		bytes.NewBufferString(formData),
	)
	if err != nil {
		return fmt.Errorf("添加到稍后再看失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	if result.Code != 0 {
		return &BiliError{
			Code:    result.Code,
			Message: result.Message,
		}
	}

	return nil
}

// DeleteFromWatchLater 从稍后再看删除视频
func (c *Client) DeleteFromWatchLater(aid int64) error {
	if c.credential == nil || c.credential.BiliJct == "" {
		return fmt.Errorf("需要登录凭据（bili_jct）")
	}

	// 构建表单数据
	formData := fmt.Sprintf("aid=%d&csrf=%s", aid, c.credential.BiliJct)

	resp, err := c.Post(
		"https://api.bilibili.com/x/v2/history/toview/del",
		map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		},
		bytes.NewBufferString(formData),
	)
	if err != nil {
		return fmt.Errorf("从稍后再看删除失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	if result.Code != 0 {
		return &BiliError{
			Code:    result.Code,
			Message: result.Message,
		}
	}

	return nil
}

// ClearWatchLater 清空稍后再看列表
func (c *Client) ClearWatchLater() error {
	if c.credential == nil || c.credential.BiliJct == "" {
		return fmt.Errorf("需要登录凭据（bili_jct）")
	}

	// 构建表单数据
	formData := fmt.Sprintf("csrf=%s", c.credential.BiliJct)

	resp, err := c.Post(
		"https://api.bilibili.com/x/v2/history/toview/clear",
		map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		},
		bytes.NewBufferString(formData),
	)
	if err != nil {
		return fmt.Errorf("清空稍后再看失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	if result.Code != 0 {
		return &BiliError{
			Code:    result.Code,
			Message: result.Message,
		}
	}

	return nil
}
