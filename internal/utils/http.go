package utils

import (
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"bili-download/internal/config"
)

// NewHTTPClient 根据代理配置创建 HTTP 客户端
func NewHTTPClient(proxyCfg config.ProxyConfig, timeout time.Duration, maxIdleConns, maxIdleConnsPerHost int) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: NewHTTPTransport(proxyCfg, maxIdleConns, maxIdleConnsPerHost),
	}
}

// NewHTTPTransport 根据代理配置创建 HTTP Transport
func NewHTTPTransport(proxyCfg config.ProxyConfig, maxIdleConns, maxIdleConnsPerHost int) *http.Transport {
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          maxIdleConns,
		MaxIdleConnsPerHost:   maxIdleConnsPerHost,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2: true,
	}

	if proxyURL, err := proxyCfg.ParseURL(); err == nil && proxyURL != nil {
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	return transport
}

// ApplyProxyEnv 为外部命令注入代理环境变量
func ApplyProxyEnv(cmd *exec.Cmd, proxyCfg config.ProxyConfig) {
	if !proxyCfg.IsEnabled() {
		return
	}

	proxyURL := strings.TrimSpace(proxyCfg.URL)
	cmd.Env = append(append(os.Environ(), cmd.Env...),
		"HTTP_PROXY="+proxyURL,
		"http_proxy="+proxyURL,
		"HTTPS_PROXY="+proxyURL,
		"https_proxy="+proxyURL,
		"ALL_PROXY="+proxyURL,
		"all_proxy="+proxyURL,
	)
}
