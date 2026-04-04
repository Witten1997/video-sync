package bilibili

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	"bili-download/internal/config"
	"bili-download/internal/utils"
)

const (
	// Web 端二维码生成 API
	QRCodeGenerateURL = "https://passport.bilibili.com/x/passport-login/web/qrcode/generate"
	// Web 端二维码状态轮询 API
	QRCodePollURL = "https://passport.bilibili.com/x/passport-login/web/qrcode/poll"
)

// 二维码状态码
const (
	QRCodeStatusSuccess             = 0
	QRCodeStatusNotScanned          = 86101
	QRCodeStatusScannedNotConfirmed = 86090
	QRCodeStatusExpired             = 86038
)

// QRCodeGenerateResponse 二维码生成响应
type QRCodeGenerateResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	TTL     int    `json:"ttl"`
	Data    struct {
		URL       string `json:"url"`
		QRCodeKey string `json:"qrcode_key"`
	} `json:"data"`
}

// QRCodePollResponse 二维码轮询响应
type QRCodePollResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	TTL     int    `json:"ttl"`
	Data    struct {
		URL          string `json:"url"`
		RefreshToken string `json:"refresh_token"`
		Timestamp    int64  `json:"timestamp"`
		Code         int    `json:"code"`
		Message      string `json:"message"`
	} `json:"data"`
}

// QRLoginResult 二维码登录结果
type QRLoginResult struct {
	Status     int
	Message    string
	Credential *Credential
}

// QRLogin 二维码登录管理器
type QRLogin struct {
	httpClient *http.Client
}

// NewQRLogin 创建二维码登录管理器
func NewQRLogin(cfg *config.Config) *QRLogin {
	jar, _ := cookiejar.New(nil)
	httpClient := utils.NewHTTPClient(cfg.Proxy, 30*time.Second, 20, 10)
	httpClient.Jar = jar

	return &QRLogin{
		httpClient: httpClient,
	}
}

// GenerateQRCode 申请二维码
func (q *QRLogin) GenerateQRCode() (*QRCodeGenerateResponse, error) {
	req, err := http.NewRequest(http.MethodGet, QRCodeGenerateURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

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

// PollQRCode 轮询二维码状态
func (q *QRLogin) PollQRCode(qrcodeKey string) (*QRLoginResult, error) {
	requestURL := fmt.Sprintf("%s?qrcode_key=%s", QRCodePollURL, qrcodeKey)

	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

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

	if pollResp.Data.Code == QRCodeStatusSuccess {
		credential, err := q.extractCredentialFromCookies(resp.Header)
		if err != nil {
			return nil, fmt.Errorf("提取登录凭据失败: %w", err)
		}
		result.Credential = credential
	}

	return result, nil
}

// extractCredentialFromCookies 从响应头提取凭据
func (q *QRLogin) extractCredentialFromCookies(header http.Header) (*Credential, error) {
	credential := &Credential{}

	cookies := header.Values("Set-Cookie")
	if len(cookies) == 0 {
		return nil, fmt.Errorf("未找到 Set-Cookie 头")
	}

	for _, cookie := range cookies {
		parts := strings.Split(cookie, ";")
		if len(parts) == 0 {
			continue
		}

		nameValue := strings.TrimSpace(parts[0])
		idx := strings.Index(nameValue, "=")
		if idx == -1 {
			continue
		}

		name := nameValue[:idx]
		value := nameValue[idx+1:]

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

	if credential.SESSDATA == "" || credential.BiliJct == "" {
		return nil, fmt.Errorf("未能获取完整的登录凭据")
	}

	return credential, nil
}

// GetStatusMessage 获取状态码对应的提示
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
