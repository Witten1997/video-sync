package config

import (
	"errors"
	"fmt"
)

// Validate 验证配置
func (c *Config) Validate() error {
	if err := c.Server.Validate(); err != nil {
		return fmt.Errorf("server 配置错误: %w", err)
	}

	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("database 配置错误: %w", err)
	}

	if err := c.Sync.Validate(); err != nil {
		return fmt.Errorf("sync 配置错误: %w", err)
	}

	if err := c.Paths.Validate(); err != nil {
		return fmt.Errorf("paths 配置错误: %w", err)
	}

	if err := c.Template.Validate(); err != nil {
		return fmt.Errorf("template 配置错误: %w", err)
	}

	if err := c.Quality.Validate(); err != nil {
		return fmt.Errorf("quality 配置错误: %w", err)
	}

	if err := c.Danmaku.Validate(); err != nil {
		return fmt.Errorf("danmaku 配置错误: %w", err)
	}

	if err := c.Advanced.Validate(); err != nil {
		return fmt.Errorf("advanced 配置错误: %w", err)
	}

	if err := c.Logging.Validate(); err != nil {
		return fmt.Errorf("logging 配置错误: %w", err)
	}

	return nil
}

// Validate 验证服务器配置
func (c *ServerConfig) Validate() error {
	if c.BindAddress == "" {
		return errors.New("bind_address 不能为空")
	}
	return nil
}

// Validate 验证数据库配置
func (c *DatabaseConfig) Validate() error {
	if c.Host == "" {
		return errors.New("host 不能为空")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return errors.New("port 必须在 1-65535 之间")
	}
	if c.User == "" {
		return errors.New("user 不能为空")
	}
	if c.DBName == "" {
		return errors.New("dbname 不能为空")
	}
	if c.MaxOpenConns <= 0 {
		return errors.New("max_open_conns 必须大于 0")
	}
	if c.MaxIdleConns <= 0 {
		return errors.New("max_idle_conns 必须大于 0")
	}
	if c.MaxIdleConns > c.MaxOpenConns {
		return errors.New("max_idle_conns 不能大于 max_open_conns")
	}
	if c.ConnMaxLifetime <= 0 {
		return errors.New("conn_max_lifetime 必须大于 0")
	}
	return nil
}

// Validate 验证同步配置
func (c *SyncConfig) Validate() error {
	if c.Interval <= 0 {
		return errors.New("interval 必须大于 0")
	}
	return nil
}

// Validate 验证路径配置
func (c *PathsConfig) Validate() error {
	if c.DownloadBase == "" {
		return errors.New("download_base 不能为空")
	}
	if c.UpperPath == "" {
		return errors.New("upper_path 不能为空")
	}
	return nil
}

// Validate 验证模板配置
func (c *TemplateConfig) Validate() error {
	if c.VideoName == "" {
		return errors.New("video_name 不能为空")
	}
	if c.PageName == "" {
		return errors.New("page_name 不能为空")
	}
	if c.TimeFormat == "" {
		return errors.New("time_format 不能为空")
	}
	return nil
}

// Validate 验证质量配置
func (c *QualityConfig) Validate() error {
	validResolutions := []string{"8K", "DOLBY", "HDR", "4K", "1080P60", "1080P+", "1080P", "720P", "480P", "360P"}
	valid := false
	for _, r := range validResolutions {
		if c.MaxResolution == r {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("max_resolution 必须是以下值之一: %v", validResolutions)
	}

	validCodecs := []string{"AVC", "HEVC", "AV1"}
	for _, codec := range c.CodecPriority {
		valid := false
		for _, vc := range validCodecs {
			if codec == vc {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("codec_priority 包含无效的编码: %s", codec)
		}
	}

	validAudioQualities := []string{"30251", "30250", "30280", "30232", "30216"}
	valid = false
	for _, q := range validAudioQualities {
		if c.AudioQuality == q {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("audio_quality 必须是以下值之一: %v (30251=Hi-RES, 30250=杜比全景声, 30280=192K, 30232=132K, 30216=64K)", validAudioQualities)
	}

	return nil
}

// Validate 验证弹幕配置
func (c *DanmakuConfig) Validate() error {
	if c.Duration <= 0 {
		return errors.New("duration 必须大于 0")
	}
	if c.FontName == "" {
		return errors.New("font_name 不能为空")
	}
	if c.FontSize <= 0 {
		return errors.New("font_size 必须大于 0")
	}
	if c.WidthRatio <= 0 {
		return errors.New("width_ratio 必须大于 0")
	}
	if c.HorizontalGap < 0 {
		return errors.New("horizontal_gap 不能为负数")
	}
	if c.LaneSize < 0 {
		return errors.New("lane_size 不能为负数")
	}
	if c.FloatPercentage < 0 || c.FloatPercentage > 1 {
		return errors.New("float_percentage 必须在 0-1 之间")
	}
	if c.BottomPercentage < 0 || c.BottomPercentage > 1 {
		return errors.New("bottom_percentage 必须在 0-1 之间")
	}
	if c.Opacity < 0 || c.Opacity > 255 {
		return errors.New("opacity 必须在 0-255 之间")
	}
	if c.OutlineWidth < 0 {
		return errors.New("outline_width 不能为负数")
	}
	return nil
}

// Validate 验证高级配置
func (c *AdvancedConfig) Validate() error {
	if c.ConcurrentLimit.Video <= 0 {
		return errors.New("concurrent_limit.video 必须大于 0")
	}
	if c.ConcurrentLimit.Page <= 0 {
		return errors.New("concurrent_limit.page 必须大于 0")
	}
	if c.RateLimit.DurationMS <= 0 {
		return errors.New("rate_limit.duration_ms 必须大于 0")
	}
	if c.RateLimit.Limit <= 0 {
		return errors.New("rate_limit.limit 必须大于 0")
	}
	validNFOTimeTypes := []string{"favtime", "pubtime"}
	valid := false
	for _, t := range validNFOTimeTypes {
		if c.NFOTimeType == t {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("nfo_time_type 必须是以下值之一: %v", validNFOTimeTypes)
	}
	return nil
}

// Validate 验证日志配置
func (c *LoggingConfig) Validate() error {
	validLevels := []string{"debug", "info", "warn", "error"}
	valid := false
	for _, l := range validLevels {
		if c.Level == l {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("level 必须是以下值之一: %v", validLevels)
	}
	if c.MaxSizeMB <= 0 {
		return errors.New("max_size_mb 必须大于 0")
	}
	if c.MaxBackups < 0 {
		return errors.New("max_backups 不能为负数")
	}
	if c.MaxAgeDays < 0 {
		return errors.New("max_age_days 不能为负数")
	}
	return nil
}
