package bilibili

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"bili-download/internal/config"
)

const (
	// DefaultUserAgent 默认 User-Agent
	DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	// DefaultReferer 默认 Referer
	DefaultReferer = "https://www.bilibili.com"
)

// Client B站 HTTP 客户端
type Client struct {
	httpClient *http.Client
	credential *Credential
}

// NewClient 创建新的 B站 客户端
func NewClient(cfg *config.Config) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		credential: &Credential{
			SESSDATA:    cfg.Bilibili.Credential.SESSDATA,
			BiliJct:     cfg.Bilibili.Credential.BiliJct,
			Buvid3:      cfg.Bilibili.Credential.Buvid3,
			DedeUserID:  cfg.Bilibili.Credential.DedeUserID,
			AcTimeValue: cfg.Bilibili.Credential.AcTimeValue,
		},
	}
}

// Request 发送 HTTP 请求
func (c *Client) Request(method, url string, headers map[string]string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置默认 headers
	req.Header.Set("User-Agent", DefaultUserAgent)
	req.Header.Set("Referer", DefaultReferer)

	// 添加凭据 cookies
	if c.credential != nil {
		c.credential.AddCookies(req)
	}

	// 添加自定义 headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}

	return resp, nil
}

// Get 发送 GET 请求
func (c *Client) Get(url string, headers map[string]string) (*http.Response, error) {
	return c.Request(http.MethodGet, url, headers, nil)
}

// Post 发送 POST 请求
func (c *Client) Post(url string, headers map[string]string, body io.Reader) (*http.Response, error) {
	if headers == nil {
		headers = make(map[string]string)
	}
	headers["Content-Type"] = "application/x-www-form-urlencoded"
	return c.Request(http.MethodPost, url, headers, body)
}

// GetJSON 发送 GET 请求并解析 JSON 响应
func (c *Client) GetJSON(url string, headers map[string]string, result interface{}) error {
	resp, err := c.Get(url, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

// PostJSON 发送 POST 请求并解析 JSON 响应
func (c *Client) PostJSON(url string, headers map[string]string, body io.Reader, result interface{}) error {
	resp, err := c.Post(url, headers, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

// SetCredential 设置凭据
func (c *Client) SetCredential(credential *Credential) {
	c.credential = credential
}

// GetCredential 获取凭据
func (c *Client) GetCredential() *Credential {
	return c.credential
}

// UpdateCredential 更新客户端凭据
func (c *Client) UpdateCredential(credentialCfg *config.CredentialConfig) {
	c.credential = &Credential{
		SESSDATA:    credentialCfg.SESSDATA,
		BiliJct:     credentialCfg.BiliJct,
		Buvid3:      credentialCfg.Buvid3,
		DedeUserID:  credentialCfg.DedeUserID,
		AcTimeValue: credentialCfg.AcTimeValue,
	}
}

// ValidateCredential 验证认证信息是否有效
func (c *Client) ValidateCredential() error {
	// 使用现有的 CheckCredentialValid 方法
	valid, err := c.CheckCredentialValid()
	if err != nil {
		return fmt.Errorf("验证失败: %w", err)
	}

	if !valid {
		return fmt.Errorf("账号未登录或 Cookie 已过期")
	}

	// 尝试获取用户信息确认登录状态
	_, err = c.GetMe()
	if err != nil {
		return fmt.Errorf("获取用户信息失败: %w", err)
	}

	return nil
}
