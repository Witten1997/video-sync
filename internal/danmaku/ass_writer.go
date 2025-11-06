package danmaku

import (
	"fmt"
	"os"
	"strings"
	"text/template"
)

// ASSConfig ASS 配置
type ASSConfig struct {
	// 视频尺寸
	VideoWidth  int
	VideoHeight int

	// 弹幕配置
	DanmakuDuration  float64 // 弹幕持续时间（秒）
	FontName         string  // 字体名称
	FontSize         int     // 字体大小
	WidthRatio       float64 // 宽度比例
	HorizontalGap    int     // 水平间距
	LaneSize         int     // 轨道大小（0为自动）
	FloatPercentage  float64 // 滚动弹幕区域百分比
	TopPercentage    float64 // 顶部弹幕区域百分比
	BottomPercentage float64 // 底部弹幕区域百分比
	Opacity          int     // 不透明度（0-255）
	OutlineWidth     float64 // 描边宽度
	TimeOffset       float64 // 时间偏移（秒）
	Bold             bool    // 粗体
}

// DefaultASSConfig 默认 ASS 配置
func DefaultASSConfig() ASSConfig {
	return ASSConfig{
		VideoWidth:       1920,
		VideoHeight:      1080,
		DanmakuDuration:  12.0,
		FontName:         "Microsoft YaHei",
		FontSize:         38,
		WidthRatio:       1.5,
		HorizontalGap:    30,
		LaneSize:         0,
		FloatPercentage:  0.5,
		TopPercentage:    0.25,
		BottomPercentage: 0.25,
		Opacity:          180,
		OutlineWidth:     1.5,
		TimeOffset:       0.0,
		Bold:             false,
	}
}

// ASSWriter ASS 文件写入器
type ASSWriter struct {
	config ASSConfig
	canvas *Canvas
	events []ASSEvent
}

// ASSEvent ASS 事件
type ASSEvent struct {
	Start string // 开始时间（HH:MM:SS.SS）
	End   string // 结束时间（HH:MM:SS.SS）
	Style string // 样式名称
	Text  string // 文本内容（可能包含特效标签）
}

// NewASSWriter 创建 ASS 写入器
func NewASSWriter(config ASSConfig) *ASSWriter {
	// 创建画布
	canvas := NewCanvas(
		config.VideoWidth,
		config.VideoHeight,
		config.FontSize,
		config.HorizontalGap,
		config.FloatPercentage,
		config.TopPercentage,
		config.BottomPercentage,
	)

	return &ASSWriter{
		config: config,
		canvas: canvas,
		events: make([]ASSEvent, 0),
	}
}

// AddDanmaku 添加弹幕
func (w *ASSWriter) AddDanmaku(danmaku Danmaku) {
	// 计算时间
	startTime := float64(danmaku.Progress)/1000.0 + w.config.TimeOffset
	duration := w.config.DanmakuDuration

	// 根据弹幕类型分配轨道
	var event ASSEvent

	switch danmaku.Mode {
	case 1, 2, 3: // 滚动弹幕
		event = w.createScrollEvent(danmaku, startTime, duration)
	case 4: // 底部弹幕
		event = w.createFixedEvent(danmaku, startTime, duration, false)
	case 5: // 顶部弹幕
		event = w.createFixedEvent(danmaku, startTime, duration, true)
	default:
		return
	}

	if event.Text != "" {
		w.events = append(w.events, event)
	}
}

// createScrollEvent 创建滚动弹幕事件
func (w *ASSWriter) createScrollEvent(danmaku Danmaku, startTime, duration float64) ASSEvent {
	// 估算文本宽度
	textWidth := EstimateTextWidth(danmaku.Content, w.config.FontSize, w.config.WidthRatio)

	// 分配轨道
	laneIndex, y := w.canvas.AllocateScrollLane(startTime, duration, textWidth)
	if laneIndex < 0 {
		// 没有可用轨道，跳过
		return ASSEvent{}
	}

	// 计算起始和结束位置
	startX := float64(w.config.VideoWidth)
	endX := -textWidth

	// 创建移动效果
	moveEffect := fmt.Sprintf("\\move(%.0f,%d,%.0f,%d)", startX, y, endX, y)

	// 创建文本
	text := w.formatText(danmaku.Content, danmaku.Color, moveEffect)

	return ASSEvent{
		Start: formatTime(startTime),
		End:   formatTime(startTime + duration),
		Style: "Default",
		Text:  text,
	}
}

// createFixedEvent 创建固定弹幕事件（顶部或底部）
func (w *ASSWriter) createFixedEvent(danmaku Danmaku, startTime, duration float64, isTop bool) ASSEvent {
	var laneIndex, y int

	// 分配轨道
	if isTop {
		laneIndex, y = w.canvas.AllocateTopLane(startTime, duration)
	} else {
		laneIndex, y = w.canvas.AllocateBottomLane(startTime, duration)
	}

	if laneIndex < 0 {
		// 没有可用轨道，跳过
		return ASSEvent{}
	}

	// 创建位置效果（居中）
	alignment := "\\an8" // 顶部居中
	if !isTop {
		alignment = "\\an2" // 底部居中
	}
	posEffect := fmt.Sprintf("%s\\pos(%d,%d)", alignment, w.config.VideoWidth/2, y)

	// 创建文本
	text := w.formatText(danmaku.Content, danmaku.Color, posEffect)

	return ASSEvent{
		Start: formatTime(startTime),
		End:   formatTime(startTime + duration),
		Style: "Default",
		Text:  text,
	}
}

// formatText 格式化文本
func (w *ASSWriter) formatText(content string, color uint32, effect string) string {
	// 转义特殊字符
	content = strings.ReplaceAll(content, "\\", "\\\\")
	content = strings.ReplaceAll(content, "{", "\\{")
	content = strings.ReplaceAll(content, "}", "\\}")

	// 颜色
	colorTag := fmt.Sprintf("\\c%s", ColorToBGR(color))

	// 组合所有标签
	return fmt.Sprintf("{%s%s}%s", effect, colorTag, content)
}

// WriteToFile 写入到文件
func (w *ASSWriter) WriteToFile(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	// 写入ASS头部
	if err := w.writeHeader(file); err != nil {
		return fmt.Errorf("写入头部失败: %w", err)
	}

	// 写入事件
	if err := w.writeEvents(file); err != nil {
		return fmt.Errorf("写入事件失败: %w", err)
	}

	return nil
}

// writeHeader 写入ASS头部
func (w *ASSWriter) writeHeader(file *os.File) error {
	// ASS头部模板
	headerTemplate := `[Script Info]
Title: Bilibili Danmaku
ScriptType: v4.00+
WrapStyle: 0
ScaledBorderAndShadow: yes
YCbCr Matrix: TV.709
PlayResX: {{.VideoWidth}}
PlayResY: {{.VideoHeight}}

[V4+ Styles]
Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding
Style: Default,{{.FontName}},{{.FontSize}},&H{{.Opacity}}FFFFFF,&H{{.Opacity}}FFFFFF,&H{{.Opacity}}000000,&H{{.Opacity}}000000,{{.Bold}},0,0,0,100,100,0,0,1,{{.OutlineWidth}},0,2,0,0,0,1

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
`

	tmpl, err := template.New("header").Parse(headerTemplate)
	if err != nil {
		return err
	}

	boldValue := 0
	if w.config.Bold {
		boldValue = -1
	}

	opacityHex := fmt.Sprintf("%02X", 255-w.config.Opacity)

	data := map[string]interface{}{
		"VideoWidth":   w.config.VideoWidth,
		"VideoHeight":  w.config.VideoHeight,
		"FontName":     w.config.FontName,
		"FontSize":     w.config.FontSize,
		"Opacity":      opacityHex,
		"Bold":         boldValue,
		"OutlineWidth": w.config.OutlineWidth,
	}

	return tmpl.Execute(file, data)
}

// writeEvents 写入事件
func (w *ASSWriter) writeEvents(file *os.File) error {
	for _, event := range w.events {
		line := fmt.Sprintf("Dialogue: 0,%s,%s,%s,,0,0,0,,%s\n",
			event.Start,
			event.End,
			event.Style,
			event.Text,
		)
		if _, err := file.WriteString(line); err != nil {
			return err
		}
	}
	return nil
}

// formatTime 格式化时间为ASS格式 (H:MM:SS.CC)
func formatTime(seconds float64) string {
	hours := int(seconds) / 3600
	minutes := (int(seconds) % 3600) / 60
	secs := int(seconds) % 60
	centiseconds := int((seconds - float64(int(seconds))) * 100)

	return fmt.Sprintf("%d:%02d:%02d.%02d", hours, minutes, secs, centiseconds)
}

// GetEventCount 获取事件数量
func (w *ASSWriter) GetEventCount() int {
	return len(w.events)
}

// Reset 重置写入器
func (w *ASSWriter) Reset() {
	w.events = make([]ASSEvent, 0)
	w.canvas.Reset()
}

// ConvertDanmakuToASS 转换弹幕为ASS文件（便捷方法）
func ConvertDanmakuToASS(danmakus []Danmaku, config ASSConfig, outputFile string) error {
	writer := NewASSWriter(config)

	// 添加所有弹幕
	for _, danmaku := range danmakus {
		writer.AddDanmaku(danmaku)
	}

	// 写入文件
	return writer.WriteToFile(outputFile)
}
