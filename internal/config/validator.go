package config

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var telegramWebhookSecretPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,256}$`)
var telegramAllowedChatTypes = map[string]struct{}{
	"private":    {},
	"group":      {},
	"supergroup": {},
}

func (c *Config) Validate() error {
	if err := c.Server.Validate(); err != nil {
		return fmt.Errorf("server config error: %w", err)
	}
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("database config error: %w", err)
	}
	if err := c.Proxy.Validate(); err != nil {
		return fmt.Errorf("proxy config error: %w", err)
	}
	if err := c.Sync.Validate(); err != nil {
		return fmt.Errorf("sync config error: %w", err)
	}
	if err := c.Paths.Validate(); err != nil {
		return fmt.Errorf("paths config error: %w", err)
	}
	if err := c.Template.Validate(); err != nil {
		return fmt.Errorf("template config error: %w", err)
	}
	if err := c.Quality.Validate(); err != nil {
		return fmt.Errorf("quality config error: %w", err)
	}
	if err := c.Danmaku.Validate(); err != nil {
		return fmt.Errorf("danmaku config error: %w", err)
	}
	if err := c.Advanced.Validate(); err != nil {
		return fmt.Errorf("advanced config error: %w", err)
	}
	if err := c.Logging.Validate(); err != nil {
		return fmt.Errorf("logging config error: %w", err)
	}
	if err := c.Telegram.Validate(); err != nil {
		return fmt.Errorf("telegram config error: %w", err)
	}

	return nil
}

func (c *ServerConfig) Validate() error {
	if c.BindAddress == "" {
		return errors.New("bind_address cannot be empty")
	}
	return nil
}

func (c *DatabaseConfig) Validate() error {
	if c.Host == "" {
		return errors.New("host cannot be empty")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return errors.New("port must be between 1 and 65535")
	}
	if c.User == "" {
		return errors.New("user cannot be empty")
	}
	if c.DBName == "" {
		return errors.New("dbname cannot be empty")
	}
	if c.MaxOpenConns <= 0 {
		return errors.New("max_open_conns must be greater than 0")
	}
	if c.MaxIdleConns <= 0 {
		return errors.New("max_idle_conns must be greater than 0")
	}
	if c.MaxIdleConns > c.MaxOpenConns {
		return errors.New("max_idle_conns cannot be greater than max_open_conns")
	}
	if c.ConnMaxLifetime <= 0 {
		return errors.New("conn_max_lifetime must be greater than 0")
	}
	return nil
}

func (c *SyncConfig) Validate() error {
	if c.Interval <= 0 {
		return errors.New("interval must be greater than 0")
	}
	return nil
}

func (c *PathsConfig) Validate() error {
	if c.DownloadBase == "" {
		return errors.New("download_base cannot be empty")
	}
	if _, err := c.NormalizedURLDownloadPath(); err != nil {
		return fmt.Errorf("url_download_path error: %w", err)
	}
	if c.UpperPath == "" {
		return errors.New("upper_path cannot be empty")
	}
	return nil
}

func (c *TemplateConfig) Validate() error {
	if c.VideoName == "" {
		return errors.New("video_name cannot be empty")
	}
	if c.PageName == "" {
		return errors.New("page_name cannot be empty")
	}
	if c.TimeFormat == "" {
		return errors.New("time_format cannot be empty")
	}
	return nil
}

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
		return fmt.Errorf("max_resolution must be one of: %v", validResolutions)
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
			return fmt.Errorf("codec_priority contains invalid codec: %s", codec)
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
		return fmt.Errorf("audio_quality must be one of: %v", validAudioQualities)
	}

	return nil
}

func (c *DanmakuConfig) Validate() error {
	if c.Duration <= 0 {
		return errors.New("duration must be greater than 0")
	}
	if c.FontName == "" {
		return errors.New("font_name cannot be empty")
	}
	if c.FontSize <= 0 {
		return errors.New("font_size must be greater than 0")
	}
	if c.WidthRatio <= 0 {
		return errors.New("width_ratio must be greater than 0")
	}
	if c.HorizontalGap < 0 {
		return errors.New("horizontal_gap cannot be negative")
	}
	if c.LaneSize < 0 {
		return errors.New("lane_size cannot be negative")
	}
	if c.FloatPercentage < 0 || c.FloatPercentage > 1 {
		return errors.New("float_percentage must be between 0 and 1")
	}
	if c.BottomPercentage < 0 || c.BottomPercentage > 1 {
		return errors.New("bottom_percentage must be between 0 and 1")
	}
	if c.Opacity < 0 || c.Opacity > 255 {
		return errors.New("opacity must be between 0 and 255")
	}
	if c.OutlineWidth < 0 {
		return errors.New("outline_width cannot be negative")
	}
	return nil
}

func (c *AdvancedConfig) Validate() error {
	if c.ConcurrentLimit.Video <= 0 {
		return errors.New("concurrent_limit.video must be greater than 0")
	}
	if c.ConcurrentLimit.Page <= 0 {
		return errors.New("concurrent_limit.page must be greater than 0")
	}
	if c.RateLimit.DurationMS <= 0 {
		return errors.New("rate_limit.duration_ms must be greater than 0")
	}
	if c.RateLimit.Limit <= 0 {
		return errors.New("rate_limit.limit must be greater than 0")
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
		return fmt.Errorf("nfo_time_type must be one of: %v", validNFOTimeTypes)
	}
	return nil
}

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
		return fmt.Errorf("level must be one of: %v", validLevels)
	}
	if c.MaxSizeMB <= 0 {
		return errors.New("max_size_mb must be greater than 0")
	}
	if c.MaxBackups < 0 {
		return errors.New("max_backups cannot be negative")
	}
	if c.MaxAgeDays < 0 {
		return errors.New("max_age_days cannot be negative")
	}
	return nil
}

func (c *TelegramConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.BotToken == "" {
		return errors.New("bot_token cannot be empty when telegram is enabled")
	}
	if c.Mode != "polling" && c.Mode != "webhook" {
		return errors.New("telegram mode must be either polling or webhook")
	}
	if c.Mode == "polling" {
		if c.PollTimeoutSeconds < 10 || c.PollTimeoutSeconds > 60 {
			return errors.New("poll_timeout_seconds must be between 10 and 60")
		}
	}
	if c.Mode == "webhook" {
		if c.WebhookURL == "" {
			return errors.New("webhook_url cannot be empty when telegram webhook mode is enabled")
		}
		parsed, err := url.ParseRequestURI(c.WebhookURL)
		if err != nil || parsed.Scheme != "https" || strings.TrimSpace(parsed.Host) == "" {
			return errors.New("webhook_url must be a valid https URL")
		}
		if c.WebhookSecret == "" {
			return errors.New("webhook_secret cannot be empty when telegram webhook mode is enabled")
		}
		if !telegramWebhookSecretPattern.MatchString(c.WebhookSecret) {
			return errors.New("webhook_secret must be 1-256 characters using only letters, digits, underscores, or hyphens")
		}
	}
	if c.MaxURLsPerMessage != 1 {
		return errors.New("max_urls_per_message must be 1 in the first telegram milestone")
	}
	if len(c.AllowedChatTypes) == 0 {
		return errors.New("allowed_chat_types must contain at least one chat type")
	}
	for _, chatType := range c.AllowedChatTypes {
		if _, ok := telegramAllowedChatTypes[chatType]; !ok {
			return errors.New("allowed_chat_types only supports private, group, and supergroup")
		}
	}
	return nil
}
