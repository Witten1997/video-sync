package danmaku

import (
	"math"
)

// Lane 弹幕轨道
type Lane struct {
	index       int     // 轨道索引
	lastEndTime float64 // 最后一个弹幕的结束时间
	lastEndX    float64 // 最后一个弹幕的结束X坐标
	occupied    bool    // 是否被占用
}

// LaneManager 轨道管理器
type LaneManager struct {
	lanes       []*Lane
	laneCount   int
	laneHeight  int
	videoWidth  int
	videoHeight int
}

// NewLaneManager 创建轨道管理器
func NewLaneManager(laneCount, laneHeight, videoWidth, videoHeight int) *LaneManager {
	lanes := make([]*Lane, laneCount)
	for i := 0; i < laneCount; i++ {
		lanes[i] = &Lane{
			index:       i,
			lastEndTime: 0,
			lastEndX:    0,
			occupied:    false,
		}
	}

	return &LaneManager{
		lanes:       lanes,
		laneCount:   laneCount,
		laneHeight:  laneHeight,
		videoWidth:  videoWidth,
		videoHeight: videoHeight,
	}
}

// AllocateLane 分配轨道（滚动弹幕）
func (m *LaneManager) AllocateLane(startTime, duration, textWidth float64, horizontalGap int) int {
	endTime := startTime + duration
	startX := float64(m.videoWidth)
	endX := -textWidth

	// 计算弹幕的速度（像素/秒）
	distance := startX - endX
	speed := distance / duration

	// 查找可用轨道
	for _, lane := range m.lanes {
		// 检查轨道是否可用
		if m.isLaneAvailable(lane, startTime, startX, speed, horizontalGap) {
			// 更新轨道状态
			lane.lastEndTime = endTime
			lane.lastEndX = endX
			lane.occupied = true
			return lane.index
		}
	}

	// 没有可用轨道，返回-1
	return -1
}

// isLaneAvailable 检查轨道是否可用
func (m *LaneManager) isLaneAvailable(lane *Lane, startTime, startX, speed float64, horizontalGap int) bool {
	// 如果轨道未被占用，直接可用
	if !lane.occupied {
		return true
	}

	// 检查时间是否已经过了上一个弹幕的结束时间
	if startTime >= lane.lastEndTime {
		return true
	}

	// 计算上一个弹幕在当前时间的位置
	timePassed := startTime - (lane.lastEndTime - 0) // 简化计算
	prevX := lane.lastEndX + speed*timePassed

	// 检查是否有足够的水平间距
	if startX-prevX >= float64(horizontalGap) {
		return true
	}

	return false
}

// Reset 重置所有轨道
func (m *LaneManager) Reset() {
	for _, lane := range m.lanes {
		lane.lastEndTime = 0
		lane.lastEndX = 0
		lane.occupied = false
	}
}

// GetLaneY 获取轨道的Y坐标
func (m *LaneManager) GetLaneY(laneIndex int) int {
	if laneIndex < 0 || laneIndex >= m.laneCount {
		return 0
	}
	return laneIndex * m.laneHeight
}

// FixedLaneManager 固定弹幕轨道管理器（顶部和底部）
type FixedLaneManager struct {
	lanes       []*FixedLane
	laneCount   int
	laneHeight  int
	videoHeight int
	isTop       bool // true为顶部，false为底部
}

// FixedLane 固定弹幕轨道
type FixedLane struct {
	index       int     // 轨道索引
	lastEndTime float64 // 最后一个弹幕的结束时间
	occupied    bool    // 是否被占用
}

// NewFixedLaneManager 创建固定弹幕轨道管理器
func NewFixedLaneManager(laneCount, laneHeight, videoHeight int, isTop bool) *FixedLaneManager {
	lanes := make([]*FixedLane, laneCount)
	for i := 0; i < laneCount; i++ {
		lanes[i] = &FixedLane{
			index:       i,
			lastEndTime: 0,
			occupied:    false,
		}
	}

	return &FixedLaneManager{
		lanes:       lanes,
		laneCount:   laneCount,
		laneHeight:  laneHeight,
		videoHeight: videoHeight,
		isTop:       isTop,
	}
}

// AllocateLane 分配固定弹幕轨道
func (m *FixedLaneManager) AllocateLane(startTime, duration float64) int {
	endTime := startTime + duration

	// 查找可用轨道
	for _, lane := range m.lanes {
		// 如果轨道未被占用或已经过了结束时间
		if !lane.occupied || startTime >= lane.lastEndTime {
			lane.lastEndTime = endTime
			lane.occupied = true
			return lane.index
		}
	}

	// 没有可用轨道，返回-1
	return -1
}

// Reset 重置所有轨道
func (m *FixedLaneManager) Reset() {
	for _, lane := range m.lanes {
		lane.lastEndTime = 0
		lane.occupied = false
	}
}

// GetLaneY 获取轨道的Y坐标
func (m *FixedLaneManager) GetLaneY(laneIndex int) int {
	if laneIndex < 0 || laneIndex >= m.laneCount {
		return 0
	}

	if m.isTop {
		// 顶部弹幕从上到下
		return laneIndex * m.laneHeight
	} else {
		// 底部弹幕从下到上
		return m.videoHeight - (laneIndex+1)*m.laneHeight
	}
}

// Canvas 弹幕画布
type Canvas struct {
	width            int
	height           int
	scrollLaneMgr    *LaneManager
	topLaneMgr       *FixedLaneManager
	bottomLaneMgr    *FixedLaneManager
	scrollPercentage float64
	topPercentage    float64
	bottomPercentage float64
	fontSize         int
	horizontalGap    int
}

// NewCanvas 创建弹幕画布
func NewCanvas(width, height, fontSize, horizontalGap int, scrollPct, topPct, bottomPct float64) *Canvas {
	// 计算轨道数量
	laneHeight := fontSize + 4

	scrollLaneCount := CalculateLaneCount(height, fontSize, scrollPct)
	topLaneCount := CalculateLaneCount(height, fontSize, topPct)
	bottomLaneCount := CalculateLaneCount(height, fontSize, bottomPct)

	return &Canvas{
		width:            width,
		height:           height,
		scrollLaneMgr:    NewLaneManager(scrollLaneCount, laneHeight, width, height),
		topLaneMgr:       NewFixedLaneManager(topLaneCount, laneHeight, height, true),
		bottomLaneMgr:    NewFixedLaneManager(bottomLaneCount, laneHeight, height, false),
		scrollPercentage: scrollPct,
		topPercentage:    topPct,
		bottomPercentage: bottomPct,
		fontSize:         fontSize,
		horizontalGap:    horizontalGap,
	}
}

// AllocateScrollLane 分配滚动弹幕轨道
func (c *Canvas) AllocateScrollLane(startTime, duration, textWidth float64) (laneIndex int, y int) {
	laneIndex = c.scrollLaneMgr.AllocateLane(startTime, duration, textWidth, c.horizontalGap)
	if laneIndex < 0 {
		return -1, 0
	}
	y = c.scrollLaneMgr.GetLaneY(laneIndex)
	return laneIndex, y
}

// AllocateTopLane 分配顶部弹幕轨道
func (c *Canvas) AllocateTopLane(startTime, duration float64) (laneIndex int, y int) {
	laneIndex = c.topLaneMgr.AllocateLane(startTime, duration)
	if laneIndex < 0 {
		return -1, 0
	}
	y = c.topLaneMgr.GetLaneY(laneIndex)
	return laneIndex, y
}

// AllocateBottomLane 分配底部弹幕轨道
func (c *Canvas) AllocateBottomLane(startTime, duration float64) (laneIndex int, y int) {
	laneIndex = c.bottomLaneMgr.AllocateLane(startTime, duration)
	if laneIndex < 0 {
		return -1, 0
	}
	y = c.bottomLaneMgr.GetLaneY(laneIndex)
	return laneIndex, y
}

// Reset 重置画布
func (c *Canvas) Reset() {
	c.scrollLaneMgr.Reset()
	c.topLaneMgr.Reset()
	c.bottomLaneMgr.Reset()
}

// GetDimensions 获取画布尺寸
func (c *Canvas) GetDimensions() (width, height int) {
	return c.width, c.height
}

// CalculateTextWidthAccurate 精确计算文本宽度（使用字体测量）
// 注意：这需要字体文件，这里提供简化版本
func CalculateTextWidthAccurate(text string, fontSize int, widthRatio float64) float64 {
	// 简化计算，实际应该使用字体库进行测量
	return EstimateTextWidth(text, fontSize, widthRatio)
}

// CalculateOptimalLaneSize 计算最优轨道大小
func CalculateOptimalLaneSize(fontSize int, lineSpacing int) int {
	// 轨道大小 = 字体大小 + 行间距
	return fontSize + lineSpacing
}

// GetOverlapFactor 计算重叠因子
func GetOverlapFactor(danmakuCount int, duration float64, laneCount int) float64 {
	if laneCount == 0 || duration == 0 {
		return 0
	}

	// 理想情况下，每个轨道在整个时长内可以容纳的弹幕数
	idealCapacity := float64(laneCount) * duration / 12.0 // 假设每条弹幕12秒

	// 实际弹幕数 / 理想容量 = 重叠因子
	return float64(danmakuCount) / idealCapacity
}

// AdjustLaneCountByDensity 根据弹幕密度调整轨道数量
func AdjustLaneCountByDensity(baseLaneCount int, density float64, maxLaneCount int) int {
	// 根据密度调整轨道数量
	// 密度越高，需要更多轨道
	adjustedCount := int(math.Ceil(float64(baseLaneCount) * (1 + density/100.0)))

	if adjustedCount > maxLaneCount {
		adjustedCount = maxLaneCount
	}

	if adjustedCount < baseLaneCount {
		adjustedCount = baseLaneCount
	}

	return adjustedCount
}
