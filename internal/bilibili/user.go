package bilibili

import (
	"fmt"
	"net/url"
	"strconv"
)

// UserInfo 用户信息
type UserInfo struct {
	Mid       int64  `json:"mid"`        // 用户 ID
	Uname     string `json:"uname"`      // 用户名
	Face      string `json:"face"`       // 头像
	Sign      string `json:"sign"`       // 签名
	Level     int    `json:"level"`      // 等级
	VipType   int    `json:"vip_type"`   // 会员类型
	VipStatus int    `json:"vip_status"` // 会员状态
}

// UserFavoriteFolder 用户收藏夹
type UserFavoriteFolder struct {
	ID         int64  `json:"id"`          // 收藏夹 ID
	FID        int64  `json:"fid"`         // 收藏夹原始 ID
	Mid        int64  `json:"mid"`         // 创建者 mid
	Title      string `json:"title"`       // 收藏夹标题
	Cover      string `json:"cover"`       // 封面
	MediaCount int    `json:"media_count"` // 内容数量
	Attr       int    `json:"attr"`        // 属性
	CTime      int64  `json:"ctime"`       // 创建时间
	MTime      int64  `json:"mtime"`       // 修改时间
}

// UserCollection 用户关注的合集
type UserCollection struct {
	ID        int64  `json:"id"`         // 合集 ID
	Mid       int64  `json:"mid"`        // UP 主 mid
	Title     string `json:"title"`      // 合集标题
	Cover     string `json:"cover"`      // 封面
	Intro     string `json:"intro"`      // 简介
	Total     int    `json:"total"`      // 总数
	Type      int    `json:"type"`       // 类型
	MTime     int64  `json:"mtime"`      // 修改时间
	UpperName string `json:"upper_name"` // UP主名称
}

// GetMe 获取当前登录用户信息
func (c *Client) GetMe() (*UserInfo, error) {
	type NavResponse struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			IsLogin   bool   `json:"isLogin"`
			Mid       int64  `json:"mid"`
			Uname     string `json:"uname"`
			Face      string `json:"face"`
			Sign      string `json:"sign"`
			LevelInfo struct {
				CurrentLevel int `json:"current_level"`
			} `json:"level_info"`
			VipType   int `json:"vipType"`
			VipStatus int `json:"vipStatus"`
		} `json:"data"`
	}

	var resp NavResponse
	err := c.GetJSON("https://api.bilibili.com/x/web-interface/nav", nil, &resp)
	if err != nil {
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}

	if resp.Code != 0 {
		return nil, &BiliError{
			Code:    resp.Code,
			Message: resp.Message,
		}
	}

	if !resp.Data.IsLogin {
		return nil, &BiliError{
			Code:    CodeUnauthorized,
			Message: "未登录",
		}
	}

	return &UserInfo{
		Mid:       resp.Data.Mid,
		Uname:     resp.Data.Uname,
		Face:      resp.Data.Face,
		Sign:      resp.Data.Sign,
		Level:     resp.Data.LevelInfo.CurrentLevel,
		VipType:   resp.Data.VipType,
		VipStatus: resp.Data.VipStatus,
	}, nil
}

// CheckCredentialValid 检查凭据是否有效
func (c *Client) CheckCredentialValid() (bool, error) {
	type CookieInfoResponse struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Refresh bool `json:"refresh"`
		} `json:"data"`
	}

	var resp CookieInfoResponse
	err := c.GetJSON("https://passport.bilibili.com/x/passport-login/web/cookie/info", nil, &resp)
	if err != nil {
		return false, fmt.Errorf("检查凭据失败: %w", err)
	}

	if resp.Code != 0 {
		// 如果是未授权错误，说明凭据无效
		if resp.Code == CodeUnauthorized {
			return false, nil
		}
		return false, &BiliError{
			Code:    resp.Code,
			Message: resp.Message,
		}
	}

	// refresh 为 true 表示需要刷新，false 表示凭据有效
	return !resp.Data.Refresh, nil
}

// GetUpperInfo 获取 UP 主信息
func (c *Client) GetUpperInfo(mid int64) (*UserInfo, error) {
	type UserInfoResponse struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Mid       int64  `json:"mid"`
			Name      string `json:"name"`
			Face      string `json:"face"`
			Sign      string `json:"sign"`
			Level     int    `json:"level"`
			VipType   int    `json:"vip_type"`
			VipStatus int    `json:"vip_status"`
		} `json:"data"`
	}

	url := fmt.Sprintf("https://api.bilibili.com/x/space/acc/info?mid=%d", mid)
	var resp UserInfoResponse
	err := c.GetJSON(url, nil, &resp)
	if err != nil {
		return nil, fmt.Errorf("获取 UP 主信息失败: %w", err)
	}

	if resp.Code != 0 {
		return nil, &BiliError{
			Code:    resp.Code,
			Message: resp.Message,
		}
	}

	return &UserInfo{
		Mid:       resp.Data.Mid,
		Uname:     resp.Data.Name,
		Face:      resp.Data.Face,
		Sign:      resp.Data.Sign,
		Level:     resp.Data.Level,
		VipType:   resp.Data.VipType,
		VipStatus: resp.Data.VipStatus,
	}, nil
}

// GetUserCreatedFavorites 获取用户创建的收藏夹列表
func (c *Client) GetUserCreatedFavorites(mid int64) ([]UserFavoriteFolder, error) {
	type Response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Count int                  `json:"count"`
			List  []UserFavoriteFolder `json:"list"`
		} `json:"data"`
	}

	// 构建URL
	params := url.Values{}
	params.Set("up_mid", strconv.FormatInt(mid, 10))
	apiURL := "https://api.bilibili.com/x/v3/fav/folder/created/list-all?" + params.Encode()

	var resp Response
	err := c.GetJSON(apiURL, nil, &resp)
	if err != nil {
		return nil, fmt.Errorf("获取用户收藏夹列表失败: %w", err)
	}

	if resp.Code != 0 {
		return nil, &BiliError{
			Code:    resp.Code,
			Message: resp.Message,
		}
	}

	return resp.Data.List, nil
}

// FollowingUser 关注的UP主信息
type FollowingUser struct {
	Mid   int64  `json:"mid"`   // UP主 mid
	Uname string `json:"uname"` // UP主名称
	Face  string `json:"face"`  // 头像
	Sign  string `json:"sign"`  // 签名
	MTime int64  `json:"mtime"` // 关注时间
}

// GetUserFollowings 获取用户关注的UP主列表
func (c *Client) GetUserFollowings(mid int64, pn, ps int) ([]FollowingUser, int, error) {
	type Response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			List  []FollowingUser `json:"list"`
			Total int             `json:"total"`
		} `json:"data"`
	}

	// 设置默认分页参数
	if pn <= 0 {
		pn = 1
	}
	if ps <= 0 || ps > 50 {
		ps = 50
	}

	// 构建URL
	params := url.Values{}
	params.Set("vmid", strconv.FormatInt(mid, 10))
	params.Set("pn", strconv.Itoa(pn))
	params.Set("ps", strconv.Itoa(ps))
	params.Set("order", "desc")
	apiURL := "https://api.bilibili.com/x/relation/followings?" + params.Encode()

	var resp Response
	err := c.GetJSON(apiURL, nil, &resp)
	if err != nil {
		return nil, 0, fmt.Errorf("获取关注列表失败: %w", err)
	}

	if resp.Code != 0 {
		return nil, 0, &BiliError{
			Code:    resp.Code,
			Message: resp.Message,
		}
	}

	return resp.Data.List, resp.Data.Total, nil
}

// SearchFollowings 搜索关注的UP主
func (c *Client) SearchFollowings(mid int64, name string, pn, ps int) ([]FollowingUser, int, error) {
	type Response struct {
		Code int `json:"code"`

		Message string `json:"message"`
		Data    struct {
			List  []FollowingUser `json:"list"`
			Total int             `json:"total"`
		} `json:"data"`
	}

	if pn <= 0 {
		pn = 1
	}
	if ps <= 0 || ps > 50 {
		ps = 50
	}

	params := url.Values{}
	params.Set("vmid", strconv.FormatInt(mid, 10))
	params.Set("name", name)
	params.Set("pn", strconv.Itoa(pn))
	params.Set("ps", strconv.Itoa(ps))
	params.Set("order", "desc")
	apiURL := "https://api.bilibili.com/x/relation/followings/search?" + params.Encode()

	var resp Response
	err := c.GetJSON(apiURL, nil, &resp)
	if err != nil {
		return nil, 0, fmt.Errorf("搜索关注列表失败: %w", err)
	}

	if resp.Code != 0 {
		return nil, 0, &BiliError{
			Code:    resp.Code,
			Message: resp.Message,
		}
	}

	return resp.Data.List, resp.Data.Total, nil
}
