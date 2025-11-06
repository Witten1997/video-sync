package api

import (
	"encoding/json"
	"fmt"

	"bili-download/internal/config"

	"github.com/gin-gonic/gin"
)

// handleGetConfig 获取配置
func (s *Server) handleGetConfig(c *gin.Context) {
	// 返回当前配置
	respondSuccess(c, s.config)
}

// handleUpdateConfig 更新配置
func (s *Server) handleUpdateConfig(c *gin.Context) {
	// 先解析为 map 来检查发送了哪些字段
	var partialConfigMap map[string]interface{}
	if err := c.ShouldBindJSON(&partialConfigMap); err != nil {
		respondValidationError(c, err.Error())
		return
	}

	// 如果是空对象，直接返回
	if len(partialConfigMap) == 0 {
		respondValidationError(c, "没有提供配置更新数据")
		return
	}

	// 将 map 转回 JSON 再解析为 Config 结构
	partialConfigBytes, err := json.Marshal(partialConfigMap)
	if err != nil {
		respondInternalError(c, fmt.Errorf("配置数据序列化失败: %w", err))
		return
	}

	var partialConfig config.Config
	if err := json.Unmarshal(partialConfigBytes, &partialConfig); err != nil {
		respondValidationError(c, err.Error())
		return
	}

	// 复制当前配置作为基础
	newConfig := *s.config

	// 使用 map 智能合并配置，只更新提交的字段
	mergeConfigFromMap(&newConfig, partialConfigMap)

	// 保留当前的 server 和 database 配置（这些不应该从 Web 界面修改）
	newConfig.Server = s.config.Server
	newConfig.Database = s.config.Database
	newConfig.Logging = s.config.Logging

	// 验证配置
	if err := newConfig.Validate(); err != nil {
		respondValidationError(c, fmt.Sprintf("配置验证失败: %v", err))
		return
	}

	// 保存配置到文件
	if err := config.Save(&newConfig, s.configPath); err != nil {
		respondInternalError(c, fmt.Errorf("保存配置失败: %w", err))
		return
	}

	// 检查 B站认证信息是否发生变化
	credentialChanged := s.config.Bilibili.Credential != newConfig.Bilibili.Credential

	// 更新服务器内存中的配置
	s.config = &newConfig

	// 同步更新 downloadMgr 中的配置引用
	if s.downloadMgr != nil {
		s.downloadMgr.UpdateConfig(&newConfig)
	}

	// 如果 B站认证信息发生变化，更新 biliClient 的 credential
	if credentialChanged {
		s.biliClient.UpdateCredential(&newConfig.Bilibili.Credential)
	}

	// 检查哪些配置需要重启服务才能生效
	requiresRestart := []string{}

	// 数据库配置更改需要重启
	if s.config.Database != newConfig.Database {
		requiresRestart = append(requiresRestart, "database")
	}

	// 服务器绑定地址更改需要重启
	if s.config.Server.BindAddress != newConfig.Server.BindAddress {
		requiresRestart = append(requiresRestart, "server.bind_address")
	}

	respondSuccess(c, gin.H{
		"message":          "配置更新成功",
		"requires_restart": requiresRestart,
		"restart_needed":   len(requiresRestart) > 0,
	})
}

// mergeConfigFromMap 从 map 合并配置到 Config 结构
// 只更新 map 中实际存在的字段，允许零值更新
func mergeConfigFromMap(cfg *config.Config, configMap map[string]interface{}) {
	// 处理 sync 配置
	if syncMap, ok := configMap["sync"].(map[string]interface{}); ok {
		if interval, exists := syncMap["interval"]; exists {
			if v, ok := interval.(float64); ok {
				cfg.Sync.Interval = int(v)
			}
		}
		if scanOnly, exists := syncMap["scan_only"]; exists {
			if v, ok := scanOnly.(bool); ok {
				cfg.Sync.ScanOnly = v
			}
		}
	}

	// 处理 paths 配置
	if pathsMap, ok := configMap["paths"].(map[string]interface{}); ok {
		if downloadBase, exists := pathsMap["download_base"]; exists {
			if v, ok := downloadBase.(string); ok {
				cfg.Paths.DownloadBase = v
			}
		}
		if upperPath, exists := pathsMap["upper_path"]; exists {
			if v, ok := upperPath.(string); ok {
				cfg.Paths.UpperPath = v
			}
		}
	}

	// 处理 template 配置
	if templateMap, ok := configMap["template"].(map[string]interface{}); ok {
		if videoName, exists := templateMap["video_name"]; exists {
			if v, ok := videoName.(string); ok {
				cfg.Template.VideoName = v
			}
		}
		if pageName, exists := templateMap["page_name"]; exists {
			if v, ok := pageName.(string); ok {
				cfg.Template.PageName = v
			}
		}
		if timeFormat, exists := templateMap["time_format"]; exists {
			if v, ok := timeFormat.(string); ok {
				cfg.Template.TimeFormat = v
			}
		}
	}

	// 处理 bilibili 配置
	if bilibiliMap, ok := configMap["bilibili"].(map[string]interface{}); ok {
		if credMap, ok := bilibiliMap["credential"].(map[string]interface{}); ok {
			if sessdata, exists := credMap["sessdata"]; exists {
				if v, ok := sessdata.(string); ok {
					cfg.Bilibili.Credential.SESSDATA = v
				}
			}
			if biliJct, exists := credMap["bili_jct"]; exists {
				if v, ok := biliJct.(string); ok {
					cfg.Bilibili.Credential.BiliJct = v
				}
			}
			if buvid3, exists := credMap["buvid3"]; exists {
				if v, ok := buvid3.(string); ok {
					cfg.Bilibili.Credential.Buvid3 = v
				}
			}
			if dedeuserid, exists := credMap["dedeuserid"]; exists {
				if v, ok := dedeuserid.(string); ok {
					cfg.Bilibili.Credential.DedeUserID = v
				}
			}
			if acTimeValue, exists := credMap["ac_time_value"]; exists {
				if v, ok := acTimeValue.(string); ok {
					cfg.Bilibili.Credential.AcTimeValue = v
				}
			}
		}
	}

	// 处理 quality 配置
	if qualityMap, ok := configMap["quality"].(map[string]interface{}); ok {
		if maxResolution, exists := qualityMap["max_resolution"]; exists {
			if v, ok := maxResolution.(string); ok {
				cfg.Quality.MaxResolution = v
			}
		}
		if codecPriority, exists := qualityMap["codec_priority"]; exists {
			if v, ok := codecPriority.([]interface{}); ok {
				codecs := make([]string, 0, len(v))
				for _, codec := range v {
					if s, ok := codec.(string); ok {
						codecs = append(codecs, s)
					}
				}
				cfg.Quality.CodecPriority = codecs
			}
		}
		if audioQuality, exists := qualityMap["audio_quality"]; exists {
			if v, ok := audioQuality.(string); ok {
				cfg.Quality.AudioQuality = v
			}
		}
		if cdnSort, exists := qualityMap["cdn_sort"]; exists {
			if v, ok := cdnSort.(bool); ok {
				cfg.Quality.CDNSort = v
			}
		}
	}

	// 处理 download 配置
	if downloadMap, ok := configMap["download"].(map[string]interface{}); ok {
		if skipPoster, exists := downloadMap["skip_poster"]; exists {
			if v, ok := skipPoster.(bool); ok {
				cfg.Download.SkipPoster = v
			}
		}
		if skipVideoNFO, exists := downloadMap["skip_video_nfo"]; exists {
			if v, ok := skipVideoNFO.(bool); ok {
				cfg.Download.SkipVideoNFO = v
			}
		}
		if skipUpper, exists := downloadMap["skip_upper"]; exists {
			if v, ok := skipUpper.(bool); ok {
				cfg.Download.SkipUpper = v
			}
		}
		if skipDanmaku, exists := downloadMap["skip_danmaku"]; exists {
			if v, ok := skipDanmaku.(bool); ok {
				cfg.Download.SkipDanmaku = v
			}
		}
		if skipSubtitle, exists := downloadMap["skip_subtitle"]; exists {
			if v, ok := skipSubtitle.(bool); ok {
				cfg.Download.SkipSubtitle = v
			}
		}
	}

	// 处理 danmaku 配置
	if danmakuMap, ok := configMap["danmaku"].(map[string]interface{}); ok {
		if duration, exists := danmakuMap["duration"]; exists {
			if v, ok := duration.(float64); ok {
				cfg.Danmaku.Duration = v
			}
		}
		if fontName, exists := danmakuMap["font_name"]; exists {
			if v, ok := fontName.(string); ok {
				cfg.Danmaku.FontName = v
			}
		}
		if fontSize, exists := danmakuMap["font_size"]; exists {
			if v, ok := fontSize.(float64); ok {
				cfg.Danmaku.FontSize = int(v)
			}
		}
		if widthRatio, exists := danmakuMap["width_ratio"]; exists {
			if v, ok := widthRatio.(float64); ok {
				cfg.Danmaku.WidthRatio = v
			}
		}
		if horizontalGap, exists := danmakuMap["horizontal_gap"]; exists {
			if v, ok := horizontalGap.(float64); ok {
				cfg.Danmaku.HorizontalGap = int(v)
			}
		}
		if laneSize, exists := danmakuMap["lane_size"]; exists {
			if v, ok := laneSize.(float64); ok {
				cfg.Danmaku.LaneSize = int(v)
			}
		}
		if floatPercentage, exists := danmakuMap["float_percentage"]; exists {
			if v, ok := floatPercentage.(float64); ok {
				cfg.Danmaku.FloatPercentage = v
			}
		}
		if bottomPercentage, exists := danmakuMap["bottom_percentage"]; exists {
			if v, ok := bottomPercentage.(float64); ok {
				cfg.Danmaku.BottomPercentage = v
			}
		}
		if opacity, exists := danmakuMap["opacity"]; exists {
			if v, ok := opacity.(float64); ok {
				cfg.Danmaku.Opacity = int(v)
			}
		}
		if outlineWidth, exists := danmakuMap["outline_width"]; exists {
			if v, ok := outlineWidth.(float64); ok {
				cfg.Danmaku.OutlineWidth = v
			}
		}
		if timeOffset, exists := danmakuMap["time_offset"]; exists {
			if v, ok := timeOffset.(float64); ok {
				cfg.Danmaku.TimeOffset = v
			}
		}
		if bold, exists := danmakuMap["bold"]; exists {
			if v, ok := bold.(bool); ok {
				cfg.Danmaku.Bold = v
			}
		}
		if customColor, exists := danmakuMap["custom_color"]; exists {
			if v, ok := customColor.(string); ok {
				cfg.Danmaku.CustomColor = v
			}
		}
		if forceCustomColor, exists := danmakuMap["force_custom_color"]; exists {
			if v, ok := forceCustomColor.(bool); ok {
				cfg.Danmaku.ForceCustomColor = v
			}
		}
	}

	// 处理 advanced 配置
	if advancedMap, ok := configMap["advanced"].(map[string]interface{}); ok {
		if concurrentMap, ok := advancedMap["concurrent_limit"].(map[string]interface{}); ok {
			if video, exists := concurrentMap["video"]; exists {
				if v, ok := video.(float64); ok {
					cfg.Advanced.ConcurrentLimit.Video = int(v)
				}
			}
			if page, exists := concurrentMap["page"]; exists {
				if v, ok := page.(float64); ok {
					cfg.Advanced.ConcurrentLimit.Page = int(v)
				}
			}
		}
		if rateLimitMap, ok := advancedMap["rate_limit"].(map[string]interface{}); ok {
			if durationMS, exists := rateLimitMap["duration_ms"]; exists {
				if v, ok := durationMS.(float64); ok {
					cfg.Advanced.RateLimit.DurationMS = int(v)
				}
			}
			if limit, exists := rateLimitMap["limit"]; exists {
				if v, ok := limit.(float64); ok {
					cfg.Advanced.RateLimit.Limit = int(v)
				}
			}
		}
		if nfoTimeType, exists := advancedMap["nfo_time_type"]; exists {
			if v, ok := nfoTimeType.(string); ok {
				cfg.Advanced.NFOTimeType = v
			}
		}
		if ytdlpExtraArgs, exists := advancedMap["ytdlp_extra_args"]; exists {
			if v, ok := ytdlpExtraArgs.([]interface{}); ok {
				args := make([]string, 0, len(v))
				for _, arg := range v {
					if s, ok := arg.(string); ok {
						args = append(args, s)
					}
				}
				cfg.Advanced.YtdlpExtraArgs = args
			}
		}
	}
}

// ConfigValidationRequest 配置验证请求
type ConfigValidationRequest struct {
	Config map[string]interface{} `json:"config"`
}

// ConfigValidationResponse 配置验证响应
type ConfigValidationResponse struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}

// handleValidateConfig 验证配置
func (s *Server) handleValidateConfig(c *gin.Context) {
	var req ConfigValidationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, err.Error())
		return
	}

	// TODO: 实现配置验证逻辑
	// 需要:
	// 1. 将 map[string]interface{} 转换为 Config 结构
	// 2. 调用 config.Validate() 方法验证
	// 3. 返回验证结果和错误信息

	// 临时实现：调用配置验证
	tempConfig := &config.Config{}
	// 这里需要将 req.Config map 转换为 Config 结构
	// 可以使用 mapstructure 或其他方式

	errors := []string{}
	if err := tempConfig.Validate(); err != nil {
		errors = append(errors, err.Error())
	}

	response := ConfigValidationResponse{
		Valid:  len(errors) == 0,
		Errors: errors,
	}

	respondSuccess(c, response)
}

// handleValidateBilibiliCredential 验证B站认证信息
func (s *Server) handleValidateBilibiliCredential(c *gin.Context) {
	// 验证当前配置的B站凭证
	if err := s.biliClient.ValidateCredential(); err != nil {
		respondError(c, 400, fmt.Sprintf("认证验证失败: %v", err))
		return
	}

	// 获取用户信息
	userInfo, err := s.biliClient.GetMe()
	if err != nil {
		respondError(c, 500, fmt.Sprintf("获取用户信息失败: %v", err))
		return
	}

	respondSuccess(c, gin.H{
		"valid":     true,
		"message":   "认证信息有效",
		"user_info": userInfo,
	})
}
