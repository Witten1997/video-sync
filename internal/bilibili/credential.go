package bilibili

import (
	"net/http"
)

// Credential B站登录凭据
type Credential struct {
	SESSDATA    string
	BiliJct     string
	Buvid3      string
	DedeUserID  string
	AcTimeValue string
}

// AddCookies 将凭据添加到 HTTP 请求的 Cookie 中
func (c *Credential) AddCookies(req *http.Request) {
	if c.SESSDATA != "" {
		req.AddCookie(&http.Cookie{
			Name:  "SESSDATA",
			Value: c.SESSDATA,
		})
	}
	if c.BiliJct != "" {
		req.AddCookie(&http.Cookie{
			Name:  "bili_jct",
			Value: c.BiliJct,
		})
	}
	if c.Buvid3 != "" {
		req.AddCookie(&http.Cookie{
			Name:  "buvid3",
			Value: c.Buvid3,
		})
	}
	if c.DedeUserID != "" {
		req.AddCookie(&http.Cookie{
			Name:  "DedeUserID",
			Value: c.DedeUserID,
		})
	}
	if c.AcTimeValue != "" {
		req.AddCookie(&http.Cookie{
			Name:  "ac_time_value",
			Value: c.AcTimeValue,
		})
	}
}

// IsValid 检查凭据是否有效（至少包含 SESSDATA）
func (c *Credential) IsValid() bool {
	return c.SESSDATA != ""
}

// ToCookieString 将凭据转换为 Cookie 字符串
func (c *Credential) ToCookieString() string {
	cookies := ""
	if c.SESSDATA != "" {
		cookies += "SESSDATA=" + c.SESSDATA + "; "
	}
	if c.BiliJct != "" {
		cookies += "bili_jct=" + c.BiliJct + "; "
	}
	if c.Buvid3 != "" {
		cookies += "buvid3=" + c.Buvid3 + "; "
	}
	if c.DedeUserID != "" {
		cookies += "DedeUserID=" + c.DedeUserID + "; "
	}
	if c.AcTimeValue != "" {
		cookies += "ac_time_value=" + c.AcTimeValue
	}
	return cookies
}
