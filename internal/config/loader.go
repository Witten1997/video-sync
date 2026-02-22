package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

var (
	globalConfig *Config
)

// Load 加载配置文件
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// 设置配置文件路径
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// 默认配置文件路径
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")
	}

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// 配置文件不存在，创建默认配置
			return createDefaultConfig(v)
		}
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 从环境变量覆盖数据库配置
	loadEnvOverrides(&cfg)

	globalConfig = &cfg
	return &cfg, nil
}

// createDefaultConfig 创建默认配置
func createDefaultConfig(v *viper.Viper) (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			BindAddress: "0.0.0.0:8080",
			AuthToken:   "",
		},
		Database: DatabaseConfig{
			Host:            "localhost",
			Port:            5432,
			User:            "bili_sync",
			Password:        "",
			DBName:          "bili_sync",
			SSLMode:         "disable",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 300,
		},
		Sync: SyncConfig{
			Interval: 3600,
			ScanOnly: false,
		},
		Paths: PathsConfig{
			DownloadBase: "./downloads",
			UpperPath:    "./metadata/people",
		},
		Template: TemplateConfig{
			VideoName:  "{{title}}",
			PageName:   "{{title}}",
			TimeFormat: "%Y-%m-%d",
		},
		Quality: QualityConfig{
			MaxResolution: "1080P+",
			CodecPriority: []string{"AVC", "HEVC", "AV1"},
			AudioQuality:  "30280",
			CDNSort:       false,
		},
		Download: DownloadConfig{
			SkipPoster:   false,
			SkipVideoNFO: false,
			SkipUpper:    false,
			SkipDanmaku:  false,
			SkipSubtitle: false,
		},
		Danmaku: DanmakuConfig{
			Duration:         12.0,
			FontName:         "Microsoft YaHei",
			FontSize:         38,
			WidthRatio:       1.5,
			HorizontalGap:    30,
			LaneSize:         0,
			FloatPercentage:  0.5,
			BottomPercentage: 0.25,
			Opacity:          180,
			OutlineWidth:     1.5,
			TimeOffset:       0.0,
			Bold:             false,
			CustomColor:      "#FFFFFF",
			ForceCustomColor: false,
		},
		Advanced: AdvancedConfig{
			ConcurrentLimit: ConcurrentLimitConfig{
				Video: 3,
				Page:  2,
			},
			RateLimit: RateLimitConfig{
				DurationMS: 250,
				Limit:      4,
			},
			NFOTimeType:    "favtime",
			YtdlpExtraArgs: []string{},
		},
		Logging: LoggingConfig{
			Level:      "info",
			File:       "",
			MaxSizeMB:  100,
			MaxBackups: 3,
			MaxAgeDays: 30,
		},
	}

	// 创建配置目录
	configDir := "./configs"
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("创建配置目录失败: %w", err)
	}

	// 保存默认配置
	configFile := filepath.Join(configDir, "config.yaml")
	v.SetConfigFile(configFile)

	// 将配置写入 viper
	v.Set("server", cfg.Server)
	v.Set("database", cfg.Database)
	v.Set("sync", cfg.Sync)
	v.Set("paths", cfg.Paths)
	v.Set("template", cfg.Template)
	v.Set("bilibili", cfg.Bilibili)
	v.Set("quality", cfg.Quality)
	v.Set("download", cfg.Download)
	v.Set("danmaku", cfg.Danmaku)
	v.Set("advanced", cfg.Advanced)
	v.Set("logging", cfg.Logging)

	if err := v.WriteConfig(); err != nil {
		return nil, fmt.Errorf("保存默认配置失败: %w", err)
	}

	globalConfig = cfg
	return cfg, nil
}

// loadEnvOverrides 从环境变量加载覆盖配置
func loadEnvOverrides(cfg *Config) {
	if host := os.Getenv("DB_HOST"); host != "" {
		cfg.Database.Host = host
	}
	if port := os.Getenv("DB_PORT"); port != "" {
		fmt.Sscanf(port, "%d", &cfg.Database.Port)
	}
	if user := os.Getenv("DB_USER"); user != "" {
		cfg.Database.User = user
	} else if user := os.Getenv("POSTGRES_USER"); user != "" {
		cfg.Database.User = user
	}
	if password := os.Getenv("DB_PASSWORD"); password != "" {
		cfg.Database.Password = password
	} else if password := os.Getenv("POSTGRES_PASSWORD"); password != "" {
		cfg.Database.Password = password
	}
	if dbname := os.Getenv("DB_NAME"); dbname != "" {
		cfg.Database.DBName = dbname
	} else if dbname := os.Getenv("POSTGRES_DB"); dbname != "" {
		cfg.Database.DBName = dbname
	}
	if sslmode := os.Getenv("DB_SSLMODE"); sslmode != "" {
		cfg.Database.SSLMode = sslmode
	}
}

// Get 获取全局配置
func Get() *Config {
	return globalConfig
}

// Save 保存配置到文件
func Save(cfg *Config, configPath string) error {
	v := viper.New()

	// 设置配置文件路径
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// 默认配置文件路径
		configDir := "./configs"
		configFile := filepath.Join(configDir, "config.yaml")
		v.SetConfigFile(configFile)
	}

	// 将配置写入 viper
	v.Set("server", cfg.Server)
	v.Set("database", cfg.Database)
	v.Set("sync", cfg.Sync)
	v.Set("paths", cfg.Paths)
	v.Set("template", cfg.Template)
	v.Set("bilibili", cfg.Bilibili)
	v.Set("quality", cfg.Quality)
	v.Set("download", cfg.Download)
	v.Set("danmaku", cfg.Danmaku)
	v.Set("advanced", cfg.Advanced)
	v.Set("logging", cfg.Logging)

	// 写入配置文件
	if err := v.WriteConfig(); err != nil {
		// 如果配置文件不存在，使用 SafeWriteConfig
		if os.IsNotExist(err) {
			if err := v.SafeWriteConfig(); err != nil {
				return fmt.Errorf("保存配置失败: %w", err)
			}
		} else {
			return fmt.Errorf("保存配置失败: %w", err)
		}
	}

	// 更新全局配置
	globalConfig = cfg
	return nil
}
