package bilibili

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// WBI 签名混淆密钥编码表
var mixinKeyEncTab = []int{
	46, 47, 18, 2, 53, 8, 23, 32, 15, 50, 10, 31, 58, 3, 45, 35, 27, 43, 5, 49,
	33, 9, 42, 19, 29, 28, 14, 39, 12, 38, 41, 13, 37, 48, 7, 16, 24, 55, 40, 61,
	26, 17, 0, 1, 60, 51, 30, 4, 22, 25, 54, 21, 56, 59, 6, 63, 57, 62, 11, 36,
	20, 34, 44, 52,
}

// WbiImg WBI 图片信息
type WbiImg struct {
	ImgURL string `json:"img_url"`
	SubURL string `json:"sub_url"`
}

// GetMixinKey 从 WbiImg 生成混淆密钥
func (w *WbiImg) GetMixinKey() string {
	imgKey := getFilename(w.ImgURL)
	subKey := getFilename(w.SubURL)

	if imgKey == "" || subKey == "" {
		return ""
	}

	combined := imgKey + subKey

	// 根据编码表生成混淆密钥
	var mixinKey strings.Builder
	for _, idx := range mixinKeyEncTab[:32] {
		if idx < len(combined) {
			mixinKey.WriteByte(combined[idx])
		}
	}

	return mixinKey.String()
}

// getFilename 从 URL 中提取文件名（不含扩展名）
func getFilename(urlStr string) string {
	// 查找最后一个 /
	lastSlash := strings.LastIndex(urlStr, "/")
	if lastSlash == -1 {
		return ""
	}

	filename := urlStr[lastSlash+1:]

	// 去除扩展名
	dotIndex := strings.LastIndex(filename, ".")
	if dotIndex == -1 {
		return filename
	}

	return filename[:dotIndex]
}

// GetWbiImg 获取 WBI 图片信息
func (c *Client) GetWbiImg() (*WbiImg, error) {
	type NavResponse struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			WbiImg WbiImg `json:"wbi_img"`
		} `json:"data"`
	}

	var resp NavResponse
	err := c.GetJSON("https://api.bilibili.com/x/web-interface/nav", nil, &resp)
	if err != nil {
		return nil, fmt.Errorf("获取 WBI 信息失败: %w", err)
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("获取 WBI 信息失败: %s", resp.Message)
	}

	return &resp.Data.WbiImg, nil
}

// SignWBI 对请求参数进行 WBI 签名
func SignWBI(params url.Values, mixinKey string) url.Values {
	// 添加时间戳
	timestamp := time.Now().Unix()
	params.Set("wts", strconv.FormatInt(timestamp, 10))

	// 按照 key 排序
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 构建查询字符串
	var queryParts []string
	for _, k := range keys {
		v := params.Get(k)
		// URL 编码，空格替换为 %20
		encoded := url.QueryEscape(v)
		encoded = strings.ReplaceAll(encoded, "+", "%20")
		queryParts = append(queryParts, k+"="+encoded)
	}
	queryStr := strings.Join(queryParts, "&")

	// 计算 MD5
	hash := md5.Sum([]byte(queryStr + mixinKey))
	wRid := hex.EncodeToString(hash[:])

	// 添加签名
	params.Set("w_rid", wRid)

	return params
}

// GetWbiSignedParams 获取带 WBI 签名的参数
func (c *Client) GetWbiSignedParams(params url.Values) (url.Values, error) {
	// 获取 WBI 图片信息
	wbiImg, err := c.GetWbiImg()
	if err != nil {
		return nil, err
	}

	// 生成混淆密钥
	mixinKey := wbiImg.GetMixinKey()
	if mixinKey == "" {
		return nil, fmt.Errorf("生成混淆密钥失败")
	}

	// 签名参数
	return SignWBI(params, mixinKey), nil
}

// GetJSONWithWBI 使用 WBI 签名发送 GET 请求并解析 JSON 响应
func (c *Client) GetJSONWithWBI(baseURL string, params url.Values, result interface{}) error {
	// 对参数进行 WBI 签名
	signedParams, err := c.GetWbiSignedParams(params)
	if err != nil {
		return err
	}

	// 构建完整 URL
	fullURL := baseURL
	if len(signedParams) > 0 {
		fullURL += "?" + signedParams.Encode()
	}

	return c.GetJSON(fullURL, nil, result)
}
