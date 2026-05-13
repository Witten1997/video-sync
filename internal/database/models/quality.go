package models

// 画质编码常量
// 数值越大画质越高；预留间隔便于扩展
const (
	QualityUnknown int8 = 0  // 未知
	Quality360P    int8 = 10 // 360P
	Quality480P    int8 = 20 // 480P
	Quality720P    int8 = 30 // 720P
	Quality1080P   int8 = 40 // 1080P
	Quality1080P60 int8 = 45 // 1080P60
	Quality4K      int8 = 50 // 4K
	Quality8K      int8 = 60 // 8K
)

// 方向编码
const (
	OrientationUnknown   int8 = 0
	OrientationLandscape int8 = 1 // 横屏
	OrientationPortrait  int8 = 2 // 竖屏
)

// QualityLabel 将画质编码转换为展示文本
func QualityLabel(code int8) string {
	switch code {
	case Quality8K:
		return "8K"
	case Quality4K:
		return "4K"
	case Quality1080P60:
		return "1080P60"
	case Quality1080P:
		return "1080P"
	case Quality720P:
		return "720P"
	case Quality480P:
		return "480P"
	case Quality360P:
		return "360P"
	default:
		return ""
	}
}

// CalcQuality 根据宽高和帧率计算画质编码
// 用短边作为画质标准，避免竖屏视频按高度被高估（如 720x1280 应为 720P 而非 1080P）
func CalcQuality(width, height int, fps float32) int8 {
	short := height
	if width > 0 && width < short {
		short = width
	}
	switch {
	case short >= 4320:
		return Quality8K
	case short >= 2160:
		return Quality4K
	case short >= 1080 && fps >= 50:
		return Quality1080P60
	case short >= 1080:
		return Quality1080P
	case short >= 720:
		return Quality720P
	case short >= 480:
		return Quality480P
	case short > 0:
		return Quality360P
	default:
		return QualityUnknown
	}
}

// CalcOrientation 根据宽高计算方向
func CalcOrientation(width, height int) int8 {
	if width <= 0 || height <= 0 {
		return OrientationUnknown
	}
	if width >= height {
		return OrientationLandscape
	}
	return OrientationPortrait
}

// OrientationLabel 方向编码转文本
func OrientationLabel(code int8) string {
	switch code {
	case OrientationLandscape:
		return "横屏"
	case OrientationPortrait:
		return "竖屏"
	default:
		return ""
	}
}
