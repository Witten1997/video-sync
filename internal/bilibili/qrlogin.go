package bilibili

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"
)

const (
	// Web端二维码生成 API
	QRCodeGenerateURL = "https://passport.bilibili.com/x/passport-login/web/qrcode/generate"
	// Web端二维码状态轮询 API
	QRCodePollURL = "https://passport.bilibili.com/x/passport-login/web/qrcode/poll"
)

// 二维码状态码
const (
	QRCodeStatusSuccess             = 0     // 登录成功
	QRCodeStatusNotScanned          = 86101 // 未扫码
	QRCodeStatusScannedNotConfirmed = 86090 // 已扫码未确认
	QRCodeStatusExpired             = 86038 // 二维码已失效
)

// QRCodeGenerateResponse 二维码生成响应
type QRCodeGenerateResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	TTL     int    `json:"ttl"`
	Data    struct {
		URL       string `json:"url"`        // 二维码内容 URL
		QRCodeKey string `json:"qrcode_key"` // 扫码登录秘钥
	} `json:"data"`
}

// QRCodePollResponse 二维码轮询响应
type QRCodePollResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	TTL     int    `json:"ttl"`
	Data    struct {
		URL          string `json:"url"`           // 游戏分站跨域登录 URL
		RefreshToken string `json:"refresh_token"` // 刷新 token
		Timestamp    int64  `json:"timestamp"`     // 登录时间戳（毫秒）
		Code         int    `json:"code"`          // 状态码
		Message      string `json:"message"`       // 状态消息
	} `json:"data"`
}

// QRLoginResult 二维码登录结果
type QRLoginResult struct {
	Status     int         // 状态码（0:成功, 86101:未扫码, 86090:已扫码未确认, 86038:已失效）
	Message    string      // 状态消息
	Credential *Credential // 登录凭据（仅在登录成功时返回）
}

// QRLogin 二维码登录管理器
type QRLogin struct {
	httpClient *http.Client
}

// NewQRLogin 创建二维码登录管理器
func NewQRLogin() *QRLogin {
	// 创建带 cookie jar 的 HTTP 客户端，用于自动处理 cookies
	jar, _ := cookiejar.New(nil)
	return &QRLogin{
		httpClient: &http.Client{
			Jar:     jar,
			Timeout: 30 * time.Second,
		},
	}
}

// GenerateQRCode 申请二维码（Web 端）
func (q *QRLogin) GenerateQRCode() (*QRCodeGenerateResponse, error) {
	req, err := http.NewRequest("GET", QRCodeGenerateURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("User-Agent", DefaultUserAgent)
	req.Header.Set("Referer", DefaultReferer)

	resp, err := q.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var qrResp QRCodeGenerateResponse
	if err := json.Unmarshal(body, &qrResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if qrResp.Code != 0 {
		return nil, fmt.Errorf("申请二维码失败: %s", qrResp.Message)
	}

	return &qrResp, nil
}

// PollQRCode 轮询二维码状态（Web 端）
func (q *QRLogin) PollQRCode(qrcodeKey string) (*QRLoginResult, error) {
	// 构建请求 URL
	url := fmt.Sprintf("%s?qrcode_key=%s", QRCodePollURL, qrcodeKey)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("User-Agent", DefaultUserAgent)
	req.Header.Set("Referer", DefaultReferer)

	resp, err := q.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var pollResp QRCodePollResponse
	if err := json.Unmarshal(body, &pollResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	result := &QRLoginResult{
		Status:  pollResp.Data.Code,
		Message: pollResp.Data.Message,
	}

	// 如果登录成功，从响应头的 Set-Cookie 中提取凭据
	if pollResp.Data.Code == QRCodeStatusSuccess {
		credential, err := q.extractCredentialFromCookies(resp.Header)
		if err != nil {
			return nil, fmt.Errorf("提取登录凭据失败: %w", err)
		}
		result.Credential = credential
	}

	return result, nil
}

// extractCredentialFromCookies 从响应头的 Set-Cookie 中提取凭据
func (q *QRLogin) extractCredentialFromCookies(header http.Header) (*Credential, error) {
	credential := &Credential{}

	// 从 Set-Cookie 头中提取各个 cookie 值
	cookies := header.Values("Set-Cookie")
	if len(cookies) == 0 {
		return nil, fmt.Errorf("未找到 Set-Cookie 头")
	}

	for _, cookie := range cookies {
		// 解析 cookie 字符串，格式为: "name=value; Path=/; Domain=..."
		parts := strings.Split(cookie, ";")
		if len(parts) == 0 {
			continue
		}

		// 获取 name=value 部分
		nameValue := strings.TrimSpace(parts[0])
		idx := strings.Index(nameValue, "=")
		if idx == -1 {
			continue
		}

		name := nameValue[:idx]
		value := nameValue[idx+1:]

		// 根据 cookie 名称设置对应的凭据字段
		switch name {
		case "SESSDATA":
			credential.SESSDATA = value
		case "bili_jct":
			credential.BiliJct = value
		case "DedeUserID":
			credential.DedeUserID = value
		case "buvid3":
			credential.Buvid3 = value
		case "ac_time_value":
			credential.AcTimeValue = value
		}
	}

	// 检查必需的凭据是否存在
	if credential.SESSDATA == "" || credential.BiliJct == "" {
		return nil, fmt.Errorf("未能获取完整的登录凭据（缺少 SESSDATA 或 bili_jct）")
	}

	return credential, nil
}

// GetStatusMessage 获取状态码对应的中文消息
func GetStatusMessage(code int) string {
	switch code {
	case QRCodeStatusSuccess:
		return "登录成功"
	case QRCodeStatusNotScanned:
		return "等待扫码"
	case QRCodeStatusScannedNotConfirmed:
		return "已扫码，等待确认"
	case QRCodeStatusExpired:
		return "二维码已失效"
	default:
		return "未知状态"
	}
}
