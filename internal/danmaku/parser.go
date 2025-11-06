package danmaku

import (
	"fmt"
	"math"
	"sort"

	"bili-download/internal/bilibili"
)

// Danmaku 弹幕结构
type Danmaku struct {
	Progress int    // 出现时间（毫秒）
	Mode     int    // 弹幕类型：1-3滚动，4底部，5顶部
	FontSize int    // 字体大小
	Color    uint32 // RGB颜色
	Content  string // 弹幕内容
	SendTime int64  // 发送时间戳
	Pool     int    // 弹幕池
	SenderID string // 发送者ID
	RowID    int64  // 弹幕ID
}

// DanmakuParser 弹幕解析器
type DanmakuParser struct {
	danmakus []Danmaku
}

// NewDanmakuParser 创建弹幕解析器
func NewDanmakuParser() *DanmakuParser {
	return &DanmakuParser{
		danmakus: make([]Danmaku, 0),
	}
}

// ParseFromBilibili 从B站弹幕响应解析
func (p *DanmakuParser) ParseFromBilibili(resp *bilibili.DanmakuResponse) error {
	if resp == nil {
		return fmt.Errorf("弹幕响应为空")
	}

	p.danmakus = make([]Danmaku, 0, len(resp.Danmakus))

	for _, elem := range resp.Danmakus {
		// 过滤不支持的弹幕类型
		if elem.Mode < 1 || elem.Mode > 5 {
			continue
		}

		// 过滤空弹幕
		if elem.Content == "" {
			continue
		}

		danmaku := Danmaku{
			Progress: elem.Progress,
			Mode:     elem.Mode,
			FontSize: elem.FontSize,
			Color:    elem.Color,
			Content:  elem.Content,
			SendTime: elem.SendTime,
			Pool:     elem.Pool,
			SenderID: elem.SenderID,
			RowID:    elem.RowID,
		}

		p.danmakus = append(p.danmakus, danmaku)
	}

	// 按出现时间排序
	sort.Slice(p.danmakus, func(i, j int) bool {
		return p.danmakus[i].Progress < p.danmakus[j].Progress
	})

	return nil
}

// GetDanmakus 获取所有弹幕
func (p *DanmakuParser) GetDanmakus() []Danmaku {
	return p.danmakus
}

// GetDanmakuCount 获取弹幕数量
func (p *DanmakuParser) GetDanmakuCount() int {
	return len(p.danmakus)
}

// FilterByTime 按时间范围过滤弹幕
func (p *DanmakuParser) FilterByTime(startMs, endMs int) []Danmaku {
	result := make([]Danmaku, 0)
	for _, dm := range p.danmakus {
		if dm.Progress >= startMs && dm.Progress <= endMs {
			result = append(result, dm)
		}
	}
	return result
}

// FilterByMode 按弹幕类型过滤
func (p *DanmakuParser) FilterByMode(modes ...int) []Danmaku {
	modeMap := make(map[int]bool)
	for _, mode := range modes {
		modeMap[mode] = true
	}

	result := make([]Danmaku, 0)
	for _, dm := range p.danmakus {
		if modeMap[dm.Mode] {
			result = append(result, dm)
		}
	}
	return result
}

// GetStatistics 获取弹幕统计信息
func (p *DanmakuParser) GetStatistics() DanmakuStatistics {
	stats := DanmakuStatistics{
		Total: len(p.danmakus),
	}

	if stats.Total == 0 {
		return stats
	}

	modeCount := make(map[int]int)
	for _, dm := range p.danmakus {
		modeCount[dm.Mode]++
	}

	stats.ScrollCount = modeCount[1] + modeCount[2] + modeCount[3]
	stats.TopCount = modeCount[5]
	stats.BottomCount = modeCount[4]

	// 计算弹幕密度（每分钟）
	if len(p.danmakus) > 0 {
		firstTime := p.danmakus[0].Progress
		lastTime := p.danmakus[len(p.danmakus)-1].Progress
		durationMinutes := float64(lastTime-firstTime) / 60000.0
		if durationMinutes > 0 {
			stats.Density = float64(stats.Total) / durationMinutes
		}
	}

	return stats
}

// DanmakuStatistics 弹幕统计信息
type DanmakuStatistics struct {
	Total       int     // 总数
	ScrollCount int     // 滚动弹幕数量
	TopCount    int     // 顶部弹幕数量
	BottomCount int     // 底部弹幕数量
	Density     float64 // 弹幕密度（每分钟）
}

// ColorToRGB 将颜色值转换为RGB
func ColorToRGB(color uint32) (r, g, b uint8) {
	r = uint8((color >> 16) & 0xFF)
	g = uint8((color >> 8) & 0xFF)
	b = uint8(color & 0xFF)
	return
}

// RGBToColor 将RGB转换为颜色值
func RGBToColor(r, g, b uint8) uint32 {
	return uint32(r)<<16 | uint32(g)<<8 | uint32(b)
}

// ColorToBGR 将颜色值转换为BGR（ASS格式）
func ColorToBGR(color uint32) string {
	r, g, b := ColorToRGB(color)
	return fmt.Sprintf("&H%02X%02X%02X", b, g, r)
}

// CalculateScrollDuration 计算滚动弹幕持续时间
func CalculateScrollDuration(config ASSConfig) float64 {
	if config.DanmakuDuration > 0 {
		return config.DanmakuDuration
	}
	return 12.0 // 默认12秒
}

// CalculateScrollDistance 计算滚动距离
func CalculateScrollDistance(textWidth float64, config ASSConfig) float64 {
	screenWidth := float64(config.VideoWidth)
	if screenWidth == 0 {
		screenWidth = 1920 // 默认宽度
	}
	return screenWidth + textWidth + float64(config.HorizontalGap)
}

// EstimateTextWidth 估算文本宽度（粗略计算）
func EstimateTextWidth(text string, fontSize int, widthRatio float64) float64 {
	// 简单估算：中文字符宽度约等于字号，英文字符宽度约为字号的一半
	width := 0.0
	for _, ch := range text {
		if ch < 128 {
			// ASCII字符
			width += float64(fontSize) * 0.5
		} else {
			// 非ASCII字符（主要是中文）
			width += float64(fontSize)
		}
	}
	return width * widthRatio
}

// CalculateLaneCount 计算轨道数量
func CalculateLaneCount(videoHeight, fontSize int, percentage float64) int {
	if percentage <= 0 || percentage > 1 {
		percentage = 0.5
	}
	availableHeight := float64(videoHeight) * percentage
	laneSize := fontSize + 4 // 字体大小 + 4像素间距
	count := int(math.Floor(availableHeight / float64(laneSize)))
	if count < 1 {
		count = 1
	}
	return count
}
