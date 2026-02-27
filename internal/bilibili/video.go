package bilibili

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
)

// VideoDetail 视频详细信息
type VideoDetail struct {
	BVid      string `json:"bvid"`      // bvid
	Aid       int64  `json:"aid"`       // avid
	Videos    int    `json:"videos"`    // 分P总数
	Tid       int    `json:"tid"`       // 分区tid
	TName     string `json:"tname"`     // 分区名称
	Copyright int    `json:"copyright"` // 版权：1-原创，2-转载
	Pic       string `json:"pic"`       // 封面图
	Title     string `json:"title"`     // 标题
	PubDate   int64  `json:"pubdate"`   // 发布时间（时间戳）
	CTime     int64  `json:"ctime"`     // 投稿时间（时间戳）
	Desc      string `json:"desc"`      // 简介
	Duration  int    `json:"duration"`  // 总时长（秒）
	State     int    `json:"state"`     // 视频状态
	Dynamic   string `json:"dynamic"`   // 动态文字
	CID       int64  `json:"cid"`       // 1P的cid

	Owner     Owner         `json:"owner"`      // UP主信息
	Stat      VideoStat     `json:"stat"`       // 统计信息
	Pages     []VideoPage   `json:"pages"`      // 分P列表
	Subtitle  SubtitleInfo  `json:"subtitle"`   // 字幕信息
	Rights    VideoRights   `json:"rights"`     // 权限信息
	UGCSeason *UGCSeason    `json:"ugc_season"` // 合集信息（可能为null）
	Staff     []StaffMember `json:"staff"`      // 合作成员（可能为null）
}

// VideoStat 视频统计信息
type VideoStat struct {
	Aid      int64 `json:"aid"`      // avid
	View     int   `json:"view"`     // 播放数
	Danmaku  int   `json:"danmaku"`  // 弹幕数
	Reply    int   `json:"reply"`    // 评论数
	Favorite int   `json:"favorite"` // 收藏数
	Coin     int   `json:"coin"`     // 投币数
	Share    int   `json:"share"`    // 分享数
	NowRank  int   `json:"now_rank"` // 当前排名
	HisRank  int   `json:"his_rank"` // 历史最高排名
	Like     int   `json:"like"`     // 点赞数
	Dislike  int   `json:"dislike"`  // 点踩数
}

// VideoPage 视频分P信息
type VideoPage struct {
	CID        int64          `json:"cid"`         // cid
	Page       int            `json:"page"`        // 分P序号
	From       string         `json:"from"`        // 视频来源
	Part       string         `json:"part"`        // 分P标题
	Duration   int            `json:"duration"`    // 时长（秒）
	Vid        string         `json:"vid"`         // 站外视频vid
	Weblink    string         `json:"weblink"`     // 站外视频链接
	Dimension  VideoDimension `json:"dimension"`   // 分辨率
	FirstFrame string         `json:"first_frame"` // 封面图
}

// VideoDimension 视频分辨率
type VideoDimension struct {
	Width  int `json:"width"`  // 宽度
	Height int `json:"height"` // 高度
	Rotate int `json:"rotate"` // 是否旋转：0-否，1-是
}

// VideoRights 视频权限信息
type VideoRights struct {
	BP            int `json:"bp"`              // 是否允许承包
	Elec          int `json:"elec"`            // 是否支持充电
	Download      int `json:"download"`        // 是否允许下载
	Movie         int `json:"movie"`           // 是否电影
	Pay           int `json:"pay"`             // 是否PGC付费
	HD5           int `json:"hd5"`             // 是否有高码率
	NoReprint     int `json:"no_reprint"`      // 是否禁止转载
	Autoplay      int `json:"autoplay"`        // 是否自动播放
	UGCPay        int `json:"ugc_pay"`         // 是否UGC付费
	IsCooperation int `json:"is_cooperation"`  // 是否联合投稿
	UGCPayPreview int `json:"ugc_pay_preview"` // UGC付费预览
	NoBackground  int `json:"no_background"`   // 禁止后台播放
	IsSteinGate   int `json:"is_stein_gate"`   // 是否互动视频
	Is360         int `json:"is_360"`          // 是否全景视频
}

// SubtitleInfo 字幕信息
type SubtitleInfo struct {
	AllowSubmit bool           `json:"allow_submit"` // 是否允许提交字幕
	List        []SubtitleItem `json:"list"`         // 字幕列表
}

// SubtitleItem 字幕条目
type SubtitleItem struct {
	ID          int64          `json:"id"`           // 字幕ID
	Lan         string         `json:"lan"`          // 语言代码
	LanDoc      string         `json:"lan_doc"`      // 语言名称
	IsLock      bool           `json:"is_lock"`      // 是否锁定
	SubtitleURL string         `json:"subtitle_url"` // 字幕文件URL
	Author      SubtitleAuthor `json:"author"`       // 上传者信息
}

// SubtitleAuthor 字幕上传者信息
type SubtitleAuthor struct {
	Mid  int64  `json:"mid"`  // UP主ID
	Name string `json:"name"` // 昵称
	Sex  string `json:"sex"`  // 性别
	Face string `json:"face"` // 头像
	Sign string `json:"sign"` // 签名
}

// UGCSeason 视频合集信息
type UGCSeason struct {
	ID         int64           `json:"id"`          // 合集ID
	Title      string          `json:"title"`       // 合集标题
	Cover      string          `json:"cover"`       // 合集封面
	Mid        int64           `json:"mid"`         // UP主ID
	Intro      string          `json:"intro"`       // 合集简介
	SignState  int             `json:"sign_state"`  // 签名状态
	Attribute  int             `json:"attribute"`   // 属性
	Sections   []SeasonSection `json:"sections"`    // 分节列表
	Stat       SeasonStat      `json:"stat"`        // 统计信息
	EpCount    int             `json:"ep_count"`    // 视频数量
	SeasonType int             `json:"season_type"` // 合集类型
}

// SeasonSection 合集分节
type SeasonSection struct {
	SeasonID int64         `json:"season_id"` // 合集ID
	ID       int64         `json:"id"`        // 分节ID
	Title    string        `json:"title"`     // 分节标题
	Type     int           `json:"type"`      // 类型
	Episodes []EpisodeInfo `json:"episodes"`  // 视频列表
}

// EpisodeInfo 合集中的视频
type EpisodeInfo struct {
	SeasonID  int64           `json:"season_id"`  // 合集ID
	SectionID int64           `json:"section_id"` // 分节ID
	ID        int64           `json:"id"`         // 视频ID
	Aid       int64           `json:"aid"`        // avid
	CID       int64           `json:"cid"`        // cid
	Title     string          `json:"title"`      // 标题
	Page      json.RawMessage `json:"page"`       // 页码（可能是int或object）
	BVid      string          `json:"bvid"`       // bvid
}

// SeasonStat 合集统计信息
type SeasonStat struct {
	SeasonID int64 `json:"season_id"` // 合集ID
	View     int   `json:"view"`      // 播放量
	Danmaku  int   `json:"danmaku"`   // 弹幕数
	Reply    int   `json:"reply"`     // 评论数
	Favorite int   `json:"favorite"`  // 收藏数
	Coin     int   `json:"coin"`      // 投币数
	Share    int   `json:"share"`     // 分享数
	Like     int   `json:"like"`      // 点赞数
}

// StaffMember 合作成员
type StaffMember struct {
	Mid      int64   `json:"mid"`      // 成员ID
	Title    string  `json:"title"`    // 职位
	Name     string  `json:"name"`     // 昵称
	Face     string  `json:"face"`     // 头像
	Official VipInfo `json:"official"` // 认证信息
}

// GetVideoDetail 获取视频详细信息（需要WBI签名）
func (c *Client) GetVideoDetail(bvid string) (*VideoDetail, error) {
	// 构建查询参数
	query := url.Values{}
	query.Set("bvid", bvid)

	// 进行 WBI 签名
	signedParams, err := c.GetWbiSignedParams(query)
	if err != nil {
		return nil, fmt.Errorf("WBI签名失败: %w", err)
	}

	// 构建完整URL
	apiURL := "https://api.bilibili.com/x/web-interface/wbi/view?" + signedParams.Encode()

	resp, err := c.Get(apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("获取视频详情失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var result struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    VideoDetail `json:"data"`
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

// GetVideoPages 获取视频分P列表
func (c *Client) GetVideoPages(bvid string) ([]VideoPage, error) {
	apiURL := fmt.Sprintf("https://api.bilibili.com/x/player/pagelist?bvid=%s", bvid)

	var result struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    []VideoPage `json:"data"`
	}

	err := c.GetJSON(apiURL, nil, &result)
	if err != nil {
		return nil, fmt.Errorf("获取视频分P列表失败: %w", err)
	}

	if result.Code != 0 {
		return nil, &BiliError{
			Code:    result.Code,
			Message: result.Message,
		}
	}

	return result.Data, nil
}

// VideoTag 视频标签
type VideoTag struct {
	TagID   int64  `json:"tag_id"`   // 标签ID
	TagName string `json:"tag_name"` // 标签名称
	TagType string `json:"tag_type"` // 标签类型
	JumpURL string `json:"jump_url"` // 跳转URL
	MusicID string `json:"music_id"` // 音乐ID（BGM标签）
}

// GetVideoTags 获取视频标签
func (c *Client) GetVideoTags(bvid string) ([]VideoTag, error) {
	apiURL := fmt.Sprintf("https://api.bilibili.com/x/web-interface/view/detail/tag?bvid=%s", bvid)

	var result struct {
		Code    int        `json:"code"`
		Message string     `json:"message"`
		Data    []VideoTag `json:"data"`
	}

	err := c.GetJSON(apiURL, nil, &result)
	if err != nil {
		return nil, fmt.Errorf("获取视频标签失败: %w", err)
	}

	if result.Code != 0 {
		return nil, &BiliError{
			Code:    result.Code,
			Message: result.Message,
		}
	}

	return result.Data, nil
}

// GetVideoDescription 获取视频简介
func (c *Client) GetVideoDescription(bvid string) (string, error) {
	apiURL := fmt.Sprintf("https://api.bilibili.com/x/web-interface/archive/desc?bvid=%s", bvid)

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    string `json:"data"`
	}

	err := c.GetJSON(apiURL, nil, &result)
	if err != nil {
		return "", fmt.Errorf("获取视频简介失败: %w", err)
	}

	if result.Code != 0 {
		return "", &BiliError{
			Code:    result.Code,
			Message: result.Message,
		}
	}

	return result.Data, nil
}

// SubtitleContent 字幕内容
type SubtitleContent struct {
	Body []SubtitleLine `json:"body"` // 字幕行列表
}

// SubtitleLine 字幕行
type SubtitleLine struct {
	From     float64 `json:"from"`     // 开始时间（秒）
	To       float64 `json:"to"`       // 结束时间（秒）
	Location int     `json:"location"` // 位置
	Content  string  `json:"content"`  // 内容
}

// GetSubtitleContent 下载字幕内容
func (c *Client) GetSubtitleContent(subtitleURL string) (*SubtitleContent, error) {
	// 字幕URL需要添加https前缀
	fullURL := "https:" + subtitleURL

	resp, err := c.Get(fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("下载字幕失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取字幕失败: %w", err)
	}

	var content SubtitleContent
	if err := json.Unmarshal(body, &content); err != nil {
		return nil, fmt.Errorf("解析字幕失败: %w", err)
	}

	return &content, nil
}

// ParseVideoURL 从B站视频URL中解析出BVID
func (c *Client) ParseVideoURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("无效的URL: %w", err)
	}

	// 支持的URL格式:
	// https://www.bilibili.com/video/BVxxxxxxxxxx
	// https://www.bilibili.com/video/BVxxxxxxxxxx?p=1
	// https://b23.tv/BVxxxxxxxxxx (短链接)
	// BVxxxxxxxxxx (直接BVID)

	// 如果是纯BVID
	if len(rawURL) == 12 && rawURL[:2] == "BV" {
		return rawURL, nil
	}

	// 从路径中提取BVID
	path := u.Path
	if len(path) > 0 {
		// 移除前导斜杠
		if path[0] == '/' {
			path = path[1:]
		}

		// 提取BVID
		// 格式: video/BVxxxxxxxxxx
		if len(path) >= 18 && path[:6] == "video/" {
			bvid := path[6:18]
			if bvid[:2] == "BV" {
				return bvid, nil
			}
		}

		// 短链接格式: BVxxxxxxxxxx
		if len(path) == 12 && path[:2] == "BV" {
			return path, nil
		}
	}

	return "", fmt.Errorf("无法从URL中提取BVID: %s", rawURL)
}
