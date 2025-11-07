package scheduler

import (
	"encoding/json"
	"strings"
	"time"

	"bili-download/internal/adapter"
	"bili-download/internal/utils"
)

// FilterRule 过滤规则
type FilterRule struct {
	// 关键词过滤
	Keywords        []string `json:"keywords"`         // 标题包含（OR关系）
	ExcludeKeywords []string `json:"exclude_keywords"` // 标题排除（AND关系）
	KeywordMode     string   `json:"keyword_mode"`     // and / or（keywords之间的关系）

	// 时长过滤
	MinDuration int `json:"min_duration"` // 最小时长（秒）
	MaxDuration int `json:"max_duration"` // 最大时长（秒）

	// 时间过滤
	PubDateAfter  string `json:"pub_date_after"`  // 发布时间晚于（RFC3339格式）
	PubDateBefore string `json:"pub_date_before"` // 发布时间早于
	FavDateAfter  string `json:"fav_date_after"`  // 收藏时间晚于
	FavDateBefore string `json:"fav_date_before"` // 收藏时间早于

	// UP主过滤
	AllowedUppers []int64 `json:"allowed_uppers"` // UP主白名单
	BlockedUppers []int64 `json:"blocked_uppers"` // UP主黑名单

	// 其他
	OnlyOriginal bool `json:"only_original"` // 仅原创
	MinViews     int  `json:"min_views"`     // 最小播放量
}

// FilterEngine 过滤引擎
type FilterEngine struct {
	globalRule *FilterRule // 全局规则
}

// NewFilterEngine 创建过滤引擎
func NewFilterEngine(globalRule *FilterRule) *FilterEngine {
	return &FilterEngine{
		globalRule: globalRule,
	}
}

// ShouldDownload 判断视频是否应该下载
func (fe *FilterEngine) ShouldDownload(video adapter.VideoInfo, sourceRule *FilterRule) (bool, string) {
	// 合并规则（视频源规则优先）
	rule := fe.mergeRules(sourceRule)

	// 1. 检查关键词
	if !fe.matchKeywords(video.Title, rule) {
		return false, "标题关键词不匹配"
	}

	// 2. 检查排除关键词
	if fe.hasExcludedKeywords(video.Title, rule) {
		return false, "标题包含排除关键词"
	}

	// 3. 检查时长
	if !fe.checkDuration(video.Duration, rule) {
		return false, "时长不符合要求"
	}

	// 4. 检查发布时间
	if !fe.checkDateRange(video.PubDate, video.AddTime, rule) {
		return false, "时间不符合要求"
	}

	// 5. 检查UP主
	if !fe.checkUpper(video.Owner.Mid, rule) {
		return false, "UP主不符合要求"
	}

	// 6. 检查播放量
	if rule.MinViews > 0 && video.Stats.View < rule.MinViews {
		return false, "播放量不符合要求"
	}

	// 所有检查通过
	return true, ""
}

// ParseRuleFromJSON 从JSON解析规则
func ParseRuleFromJSON(jsonStr string) (*FilterRule, error) {
	if jsonStr == "" {
		return &FilterRule{}, nil
	}

	var rule FilterRule
	if err := json.Unmarshal([]byte(jsonStr), &rule); err != nil {
		return nil, err
	}

	return &rule, nil
}

// mergeRules 合并规则（视频源规则优先）
func (fe *FilterEngine) mergeRules(sourceRule *FilterRule) *FilterRule {
	if sourceRule == nil {
		return fe.globalRule
	}

	if fe.globalRule == nil {
		return sourceRule
	}

	// 简单合并：视频源规则非零值覆盖全局规则
	merged := &FilterRule{}
	*merged = *fe.globalRule // 复制全局规则

	// 覆盖非零值
	if len(sourceRule.Keywords) > 0 {
		merged.Keywords = sourceRule.Keywords
	}
	if len(sourceRule.ExcludeKeywords) > 0 {
		merged.ExcludeKeywords = sourceRule.ExcludeKeywords
	}
	if sourceRule.KeywordMode != "" {
		merged.KeywordMode = sourceRule.KeywordMode
	}
	if sourceRule.MinDuration > 0 {
		merged.MinDuration = sourceRule.MinDuration
	}
	if sourceRule.MaxDuration > 0 {
		merged.MaxDuration = sourceRule.MaxDuration
	}
	if sourceRule.PubDateAfter != "" {
		merged.PubDateAfter = sourceRule.PubDateAfter
	}
	if sourceRule.PubDateBefore != "" {
		merged.PubDateBefore = sourceRule.PubDateBefore
	}
	if sourceRule.FavDateAfter != "" {
		merged.FavDateAfter = sourceRule.FavDateAfter
	}
	if sourceRule.FavDateBefore != "" {
		merged.FavDateBefore = sourceRule.FavDateBefore
	}
	if len(sourceRule.AllowedUppers) > 0 {
		merged.AllowedUppers = sourceRule.AllowedUppers
	}
	if len(sourceRule.BlockedUppers) > 0 {
		merged.BlockedUppers = sourceRule.BlockedUppers
	}
	if sourceRule.OnlyOriginal {
		merged.OnlyOriginal = true
	}
	if sourceRule.MinViews > 0 {
		merged.MinViews = sourceRule.MinViews
	}

	return merged
}

// matchKeywords 匹配关键词
func (fe *FilterEngine) matchKeywords(title string, rule *FilterRule) bool {
	if len(rule.Keywords) == 0 {
		return true // 没有关键词要求，通过
	}

	title = strings.ToLower(title)

	if rule.KeywordMode == "and" {
		// AND模式：所有关键词都要包含
		for _, keyword := range rule.Keywords {
			if !strings.Contains(title, strings.ToLower(keyword)) {
				return false
			}
		}
		return true
	} else {
		// OR模式（默认）：至少包含一个关键词
		for _, keyword := range rule.Keywords {
			if strings.Contains(title, strings.ToLower(keyword)) {
				return true
			}
		}
		return false
	}
}

// hasExcludedKeywords 检查是否包含排除关键词
func (fe *FilterEngine) hasExcludedKeywords(title string, rule *FilterRule) bool {
	if len(rule.ExcludeKeywords) == 0 {
		return false // 没有排除关键词
	}

	title = strings.ToLower(title)

	for _, keyword := range rule.ExcludeKeywords {
		if strings.Contains(title, strings.ToLower(keyword)) {
			return true // 包含排除关键词
		}
	}

	return false
}

// checkDuration 检查时长
func (fe *FilterEngine) checkDuration(duration int, rule *FilterRule) bool {
	if rule.MinDuration > 0 && duration < rule.MinDuration {
		return false
	}

	if rule.MaxDuration > 0 && duration > rule.MaxDuration {
		return false
	}

	return true
}

// checkDateRange 检查时间范围
func (fe *FilterEngine) checkDateRange(pubtime, favtime time.Time, rule *FilterRule) bool {
	// 检查发布时间
	if rule.PubDateAfter != "" {
		after, err := time.Parse(time.RFC3339, rule.PubDateAfter)
		if err != nil {
			utils.Warn("解析发布时间失败: %v", err)
		} else if pubtime.Before(after) {
			return false
		}
	}

	if rule.PubDateBefore != "" {
		before, err := time.Parse(time.RFC3339, rule.PubDateBefore)
		if err != nil {
			utils.Warn("解析发布时间失败: %v", err)
		} else if pubtime.After(before) {
			return false
		}
	}

	// 检查收藏时间
	if rule.FavDateAfter != "" {
		after, err := time.Parse(time.RFC3339, rule.FavDateAfter)
		if err != nil {
			utils.Warn("解析收藏时间失败: %v", err)
		} else if favtime.Before(after) {
			return false
		}
	}

	if rule.FavDateBefore != "" {
		before, err := time.Parse(time.RFC3339, rule.FavDateBefore)
		if err != nil {
			utils.Warn("解析收藏时间失败: %v", err)
		} else if favtime.After(before) {
			return false
		}
	}

	return true
}

// checkUpper 检查UP主
func (fe *FilterEngine) checkUpper(upperID int64, rule *FilterRule) bool {
	// 检查黑名单
	if len(rule.BlockedUppers) > 0 {
		for _, blocked := range rule.BlockedUppers {
			if upperID == blocked {
				return false
			}
		}
	}

	// 检查白名单
	if len(rule.AllowedUppers) > 0 {
		for _, allowed := range rule.AllowedUppers {
			if upperID == allowed {
				return true
			}
		}
		return false // 有白名单但不在白名单中
	}

	return true
}
