package bilibili

import (
	"bytes"
	"compress/flate"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// DanmakuResponse 弹幕响应（XML格式）
type DanmakuResponse struct {
	XMLName  xml.Name      `xml:"i"`
	ChatID   int64         `xml:"chatid"`
	Mission  int           `xml:"mission"`
	MaxLimit int           `xml:"maxlimit"`
	State    int           `xml:"state"`
	RealName int           `xml:"real_name"`
	Source   string        `xml:"source"`
	Danmakus []DanmakuElem `xml:"d"`
}

// DanmakuElem 弹幕元素
type DanmakuElem struct {
	Content  string `xml:",chardata"` // 弹幕内容
	Attr     string `xml:"p,attr"`    // 弹幕属性（逗号分隔）
	Progress int    `xml:"-"`         // 出现时间（毫秒）
	Mode     int    `xml:"-"`         // 弹幕类型
	FontSize int    `xml:"-"`         // 字体大小
	Color    uint32 `xml:"-"`         // 颜色
	SendTime int64  `xml:"-"`         // 发送时间戳
	Pool     int    `xml:"-"`         // 弹幕池
	SenderID string `xml:"-"`         // 发送者ID（哈希）
	RowID    int64  `xml:"-"`         // 弹幕ID
}

// ParseAttr 解析弹幕属性字符串
func (d *DanmakuElem) ParseAttr() error {
	if d.Attr == "" {
		return nil
	}

	parts := strings.Split(d.Attr, ",")
	if len(parts) < 8 {
		return fmt.Errorf("invalid danmaku attr format: %s", d.Attr)
	}

	// 格式：出现时间,模式,字号,颜色,发送时间戳,弹幕池,发送者ID,弹幕ID
	var err error

	// 出现时间（秒，转换为毫秒）
	if timeFloat, err := strconv.ParseFloat(parts[0], 64); err == nil {
		d.Progress = int(timeFloat * 1000)
	}

	// 模式：1-3滚动，4底部，5顶部，6逆向，7高级，8代码，9BAS
	if mode, err := strconv.Atoi(parts[1]); err == nil {
		d.Mode = mode
	}

	// 字号：18小，25标准，36大
	if fontSize, err := strconv.Atoi(parts[2]); err == nil {
		d.FontSize = fontSize
	}

	// 颜色（十进制RGB）
	if color, err := strconv.ParseUint(parts[3], 10, 32); err == nil {
		d.Color = uint32(color)
	}

	// 发送时间戳
	if sendTime, err := strconv.ParseInt(parts[4], 10, 64); err == nil {
		d.SendTime = sendTime
	}

	// 弹幕池：0普通，1字幕，2特殊
	if pool, err := strconv.Atoi(parts[5]); err == nil {
		d.Pool = pool
	}

	// 发送者ID（哈希）
	d.SenderID = parts[6]

	// 弹幕ID
	if rowID, err := strconv.ParseInt(parts[7], 10, 64); err == nil {
		d.RowID = rowID
	}

	return err
}

// GetDanmakuXML 获取弹幕（XML格式）
func (c *Client) GetDanmakuXML(cid int64) (*DanmakuResponse, error) {
	apiURL := fmt.Sprintf("https://api.bilibili.com/x/v1/dm/list.so?oid=%d", cid)

	resp, err := c.Get(apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("获取弹幕失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体（deflate压缩）
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 解压缩（deflate）
	flateReader := flate.NewReader(bytes.NewReader(body))
	defer flateReader.Close()

	decompressed, err := io.ReadAll(flateReader)
	if err != nil {
		return nil, fmt.Errorf("解压缩失败: %w", err)
	}

	// 解析XML
	var danmakuResp DanmakuResponse
	if err := xml.Unmarshal(decompressed, &danmakuResp); err != nil {
		return nil, fmt.Errorf("解析XML失败: %w", err)
	}

	// 解析每个弹幕的属性
	for i := range danmakuResp.Danmakus {
		if err := danmakuResp.Danmakus[i].ParseAttr(); err != nil {
			// 继续处理，但记录错误
			continue
		}
	}

	return &danmakuResp, nil
}

// DanmakuSegmentParams 获取弹幕分段参数（Protobuf格式，推荐）
type DanmakuSegmentParams struct {
	Type         int   // 类型：1-视频，2-漫画
	OID          int64 // 视频CID（必需）
	PID          int64 // 稿件AVID（可选）
	SegmentIndex int   // 分段索引：每6分钟一段（必需）
}

// GetDanmakuSegmentRaw 获取弹幕分段原始数据（Protobuf格式）
// 注意：返回的是protobuf二进制数据，需要使用protobuf库解析
func (c *Client) GetDanmakuSegmentRaw(params DanmakuSegmentParams) ([]byte, error) {
	if params.Type == 0 {
		params.Type = 1
	}

	apiURL := fmt.Sprintf(
		"https://api.bilibili.com/x/v2/dm/web/seg.so?type=%d&oid=%d&segment_index=%d",
		params.Type,
		params.OID,
		params.SegmentIndex,
	)

	if params.PID > 0 {
		apiURL += fmt.Sprintf("&pid=%d", params.PID)
	}

	resp, err := c.Get(apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("获取弹幕分段失败: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	return data, nil
}

// DanmakuMetadata 弹幕元数据
type DanmakuMetadata struct {
	AIDm         DanmakuAI    `json:"ai_dm"`         // AI弹幕信息
	CommandDms   []CommandDm  `json:"command_dms"`   // 互动弹幕
	Mask         MaskInfo     `json:"mask"`          // 蒙版信息
	Subtitle     SubtitleV2   `json:"subtitle"`      // 字幕信息
	PlayerConfig PlayerConfig `json:"player_config"` // 播放器配置
	Report       ReportInfo   `json:"report"`        // 上报信息
}

// DanmakuAI AI弹幕信息
type DanmakuAI struct {
	Count int `json:"count"` // 弹幕总数
}

// CommandDm 互动弹幕
type CommandDm struct {
	ID       int64  `json:"id"`       // 弹幕ID
	OID      int64  `json:"oid"`      // 视频CID
	Mid      int64  `json:"mid"`      // 发送者mid
	Command  string `json:"command"`  // 命令类型
	Content  string `json:"content"`  // 内容
	Progress int    `json:"progress"` // 出现时间（毫秒）
	CTime    string `json:"ctime"`    // 创建时间
	MTime    string `json:"mtime"`    // 修改时间
	Extra    string `json:"extra"`    // 额外JSON数据
}

// MaskInfo 蒙版信息
type MaskInfo struct {
	CID     int64  `json:"cid"`      // 视频CID
	Plat    int    `json:"plat"`     // 平台
	FPS     int    `json:"fps"`      // 帧率
	Time    int64  `json:"time"`     // 时长
	MaskURL string `json:"mask_url"` // 蒙版URL
}

// SubtitleV2 字幕信息V2
type SubtitleV2 struct {
	Subtitles []SubtitleLan `json:"subtitles"` // 字幕列表
}

// SubtitleLan 字幕语言
type SubtitleLan struct {
	ID          int64  `json:"id"`           // 字幕ID
	Lan         string `json:"lan"`          // 语言代码
	LanDoc      string `json:"lan_doc"`      // 语言名称
	SubtitleURL string `json:"subtitle_url"` // 字幕URL
}

// PlayerConfig 播放器配置
type PlayerConfig struct {
	DMSwitch     bool    `json:"dm_switch"`    // 弹幕开关
	AISwitch     bool    `json:"ai_switch"`    // AI云屏蔽开关
	AILevel      int     `json:"ai_level"`     // 云屏蔽等级 0-10
	BlockTop     bool    `json:"blocktop"`     // 屏蔽顶部
	BlockScroll  bool    `json:"blockscroll"`  // 屏蔽滚动
	BlockBottom  bool    `json:"blockbottom"`  // 屏蔽底部
	BlockColor   bool    `json:"blockcolor"`   // 屏蔽彩色
	BlockSpecial bool    `json:"blockspecial"` // 屏蔽特殊
	Opacity      float64 `json:"opacity"`      // 不透明度 0-1
	DMArea       int     `json:"dmarea"`       // 弹幕显示区域 0-100
	SpeedPlus    float64 `json:"speedplus"`    // 弹幕速度 0.4-1.6
	FontSize     float64 `json:"fontsize"`     // 字体大小
}

// ReportInfo 上报信息
type ReportInfo struct {
	StateKey string `json:"state_key"` // 状态Key
}

// GetDanmakuMetadata 获取弹幕元数据
func (c *Client) GetDanmakuMetadata(cid int64, bvid string) (*DanmakuMetadata, error) {
	apiURL := fmt.Sprintf(
		"https://api.bilibili.com/x/v2/dm/web/view?type=1&oid=%d&pid=%s",
		cid,
		bvid,
	)

	var result struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    DanmakuMetadata `json:"data"`
	}

	err := c.GetJSON(apiURL, nil, &result)
	if err != nil {
		return nil, fmt.Errorf("获取弹幕元数据失败: %w", err)
	}

	if result.Code != 0 {
		return nil, &BiliError{
			Code:    result.Code,
			Message: result.Message,
		}
	}

	return &result.Data, nil
}

// GetDanmakuCount 获取弹幕数量
func (c *Client) GetDanmakuCount(cid int64) (int, error) {
	metadata, err := c.GetDanmakuMetadata(cid, "")
	if err != nil {
		return 0, err
	}
	return metadata.AIDm.Count, nil
}
