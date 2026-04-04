package config

import (
	"fmt"
	"net/url"
	"strings"
)

// ProxyConfig 网络代理配置
type ProxyConfig struct {
	Enabled bool   `yaml:"enabled" mapstructure:"enabled" json:"enabled"`
	URL     string `yaml:"url" mapstructure:"url" json:"url"`
}

// IsEnabled 是否启用代理
func (c ProxyConfig) IsEnabled() bool {
	return c.Enabled && strings.TrimSpace(c.URL) != ""
}

// ParseURL 解析代理地址
func (c ProxyConfig) ParseURL() (*url.URL, error) {
	if !c.IsEnabled() {
		return nil, nil
	}

	parsed, err := url.Parse(strings.TrimSpace(c.URL))
	if err != nil {
		return nil, fmt.Errorf("解析代理地址失败: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("代理地址必须使用 http 或 https 协议")
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("代理地址缺少主机信息")
	}

	return parsed, nil
}

// Validate 校验代理配置
func (c ProxyConfig) Validate() error {
	_, err := c.ParseURL()
	return err
}
