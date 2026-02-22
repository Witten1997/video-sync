package config

import (
	"time"
)

// Config 应用配置结构
type Config struct {
	Server   ServerConfig   `yaml:"server" mapstructure:"server" json:"server"`
	Database DatabaseConfig `yaml:"database" mapstructure:"database" json:"database"`
	Sync     SyncConfig     `yaml:"sync" mapstructure:"sync" json:"sync"`
	Paths    PathsConfig    `yaml:"paths" mapstructure:"paths" json:"paths"`
	Template TemplateConfig `yaml:"template" mapstructure:"template" json:"template"`
	Bilibili BilibiliConfig `yaml:"bilibili" mapstructure:"bilibili" json:"bilibili"`
	Quality  QualityConfig  `yaml:"quality" mapstructure:"quality" json:"quality"`
	Download DownloadConfig `yaml:"download" mapstructure:"download" json:"download"`
	Danmaku  DanmakuConfig  `yaml:"danmaku" mapstructure:"danmaku" json:"danmaku"`
	Advanced AdvancedConfig `yaml:"advanced" mapstructure:"advanced" json:"advanced"`
	Logging  LoggingConfig  `yaml:"logging" mapstructure:"logging" json:"logging"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	BindAddress string `yaml:"bind_address" mapstructure:"bind_address" json:"bind_address"`
	AuthToken   string `yaml:"auth_token" mapstructure:"auth_token" json:"auth_token"`
	JWTSecret   string `yaml:"jwt_secret" mapstructure:"jwt_secret" json:"-"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host            string `yaml:"host" mapstructure:"host" json:"host"`
	Port            int    `yaml:"port" mapstructure:"port" json:"port"`
	User            string `yaml:"user" mapstructure:"user" json:"user"`
	Password        string `yaml:"password" mapstructure:"password" json:"password"`
	DBName          string `yaml:"dbname" mapstructure:"dbname" json:"dbname"`
	SSLMode         string `yaml:"sslmode" mapstructure:"sslmode" json:"sslmode"`
	MaxOpenConns    int    `yaml:"max_open_conns" mapstructure:"max_open_conns" json:"max_open_conns"`
	MaxIdleConns    int    `yaml:"max_idle_conns" mapstructure:"max_idle_conns" json:"max_idle_conns"`
	ConnMaxLifetime int    `yaml:"conn_max_lifetime" mapstructure:"conn_max_lifetime" json:"conn_max_lifetime"` // 秒
}

// SyncConfig 同步配置
type SyncConfig struct {
	Interval int  `yaml:"interval" mapstructure:"interval" json:"interval"` // 秒
	ScanOnly bool `yaml:"scan_only" mapstructure:"scan_only" json:"scan_only"`
}

// PathsConfig 路径配置
type PathsConfig struct {
	DownloadBase string `yaml:"download_base" mapstructure:"download_base" json:"download_base"`
	UpperPath    string `yaml:"upper_path" mapstructure:"upper_path" json:"upper_path"`
}

// TemplateConfig 模板配置
type TemplateConfig struct {
	VideoName  string `yaml:"video_name" mapstructure:"video_name" json:"video_name"`
	PageName   string `yaml:"page_name" mapstructure:"page_name" json:"page_name"`
	TimeFormat string `yaml:"time_format" mapstructure:"time_format" json:"time_format"`
}

// BilibiliConfig B站配置
type BilibiliConfig struct {
	Credential CredentialConfig `yaml:"credential" mapstructure:"credential" json:"credential"`
}

// CredentialConfig 凭据配置
type CredentialConfig struct {
	SESSDATA    string `yaml:"sessdata" mapstructure:"sessdata" json:"sessdata"`
	BiliJct     string `yaml:"bili_jct" mapstructure:"bili_jct" json:"bili_jct"`
	Buvid3      string `yaml:"buvid3" mapstructure:"buvid3" json:"buvid3"`
	DedeUserID  string `yaml:"dedeuserid" mapstructure:"dedeuserid" json:"dedeuserid"`
	AcTimeValue string `yaml:"ac_time_value" mapstructure:"ac_time_value" json:"ac_time_value"`
}

// QualityConfig 视频质量配置
type QualityConfig struct {
	MaxResolution string   `yaml:"max_resolution" mapstructure:"max_resolution" json:"max_resolution"`
	CodecPriority []string `yaml:"codec_priority" mapstructure:"codec_priority" json:"codec_priority"`
	AudioQuality  string   `yaml:"audio_quality" mapstructure:"audio_quality" json:"audio_quality"`
	CDNSort       bool     `yaml:"cdn_sort" mapstructure:"cdn_sort" json:"cdn_sort"`
}

// DownloadConfig 下载配置
type DownloadConfig struct {
	SkipPoster   bool `yaml:"skip_poster" mapstructure:"skip_poster" json:"skip_poster"`
	SkipVideoNFO bool `yaml:"skip_video_nfo" mapstructure:"skip_video_nfo" json:"skip_video_nfo"`
	SkipUpper    bool `yaml:"skip_upper" mapstructure:"skip_upper" json:"skip_upper"`
	SkipDanmaku  bool `yaml:"skip_danmaku" mapstructure:"skip_danmaku" json:"skip_danmaku"`
	SkipSubtitle bool `yaml:"skip_subtitle" mapstructure:"skip_subtitle" json:"skip_subtitle"`
}

// DanmakuConfig 弹幕配置
type DanmakuConfig struct {
	Duration         float64 `yaml:"duration" mapstructure:"duration" json:"duration"`
	FontName         string  `yaml:"font_name" mapstructure:"font_name" json:"font_name"`
	FontSize         int     `yaml:"font_size" mapstructure:"font_size" json:"font_size"`
	WidthRatio       float64 `yaml:"width_ratio" mapstructure:"width_ratio" json:"width_ratio"`
	HorizontalGap    int     `yaml:"horizontal_gap" mapstructure:"horizontal_gap" json:"horizontal_gap"`
	LaneSize         int     `yaml:"lane_size" mapstructure:"lane_size" json:"lane_size"`
	FloatPercentage  float64 `yaml:"float_percentage" mapstructure:"float_percentage" json:"float_percentage"`
	BottomPercentage float64 `yaml:"bottom_percentage" mapstructure:"bottom_percentage" json:"bottom_percentage"`
	Opacity          int     `yaml:"opacity" mapstructure:"opacity" json:"opacity"`
	OutlineWidth     float64 `yaml:"outline_width" mapstructure:"outline_width" json:"outline_width"`
	TimeOffset       float64 `yaml:"time_offset" mapstructure:"time_offset" json:"time_offset"`
	Bold             bool    `yaml:"bold" mapstructure:"bold" json:"bold"`
	CustomColor      string  `yaml:"custom_color" mapstructure:"custom_color" json:"custom_color"`                   // 自定义弹幕颜色（十六进制，如 #FFFFFF）
	ForceCustomColor bool    `yaml:"force_custom_color" mapstructure:"force_custom_color" json:"force_custom_color"` // 是否强制使用自定义颜色
}

// AdvancedConfig 高级配置
type AdvancedConfig struct {
	ConcurrentLimit ConcurrentLimitConfig `yaml:"concurrent_limit" mapstructure:"concurrent_limit" json:"concurrent_limit"`
	RateLimit       RateLimitConfig       `yaml:"rate_limit" mapstructure:"rate_limit" json:"rate_limit"`
	NFOTimeType     string                `yaml:"nfo_time_type" mapstructure:"nfo_time_type" json:"nfo_time_type"`
	YtdlpExtraArgs  []string              `yaml:"ytdlp_extra_args" mapstructure:"ytdlp_extra_args" json:"ytdlp_extra_args"`
}

// ConcurrentLimitConfig 并发限制配置
type ConcurrentLimitConfig struct {
	Video int `yaml:"video" mapstructure:"video" json:"video"`
	Page  int `yaml:"page" mapstructure:"page" json:"page"`
}

// RateLimitConfig 速率限制配置
type RateLimitConfig struct {
	DurationMS int `yaml:"duration_ms" mapstructure:"duration_ms" json:"duration_ms"`
	Limit      int `yaml:"limit" mapstructure:"limit" json:"limit"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level      string `yaml:"level" mapstructure:"level" json:"level"`
	File       string `yaml:"file" mapstructure:"file" json:"file"`
	MaxSizeMB  int    `yaml:"max_size_mb" mapstructure:"max_size_mb" json:"max_size_mb"`
	MaxBackups int    `yaml:"max_backups" mapstructure:"max_backups" json:"max_backups"`
	MaxAgeDays int    `yaml:"max_age_days" mapstructure:"max_age_days" json:"max_age_days"`
}

// GetConnMaxLifetime 返回连接最大生命周期
func (c *DatabaseConfig) GetConnMaxLifetime() time.Duration {
	return time.Duration(c.ConnMaxLifetime) * time.Second
}

// GetSyncInterval 返回同步间隔
func (c *SyncConfig) GetSyncInterval() time.Duration {
	return time.Duration(c.Interval) * time.Second
}
