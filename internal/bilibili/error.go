package bilibili

import (
	"fmt"
)

// APIResponse B站 API 通用响应结构
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	TTL     int         `json:"ttl,omitempty"`
	Data    interface{} `json:"data"`
}

// BiliError B站 API 错误
type BiliError struct {
	Code    int
	Message string
}

func (e *BiliError) Error() string {
	return fmt.Sprintf("B站 API 错误 [%d]: %s", e.Code, e.Message)
}

// 常见错误码
const (
	CodeSuccess           = 0     // 成功
	CodeUnauthorized      = -101  // 账号未登录
	CodeInvalidParam      = -400  // 请求错误
	CodeNotFound          = -404  // 无视频
	CodeRiskControl       = -412  // 请求被拦截（风控）
	CodeTooManyRequests   = -509  // 请求过于频繁
	CodeVideoNotAvailable = 62002 // 视频不可见/审核中
	CodeVideoBeenDeleted  = 62004 // 视频已删除
)

// 错误类型判断
func IsRiskControlError(code int) bool {
	return code == CodeRiskControl
}

func IsNotFoundError(code int) bool {
	return code == CodeNotFound || code == CodeVideoNotAvailable || code == CodeVideoBeenDeleted
}

func IsUnauthorizedError(code int) bool {
	return code == CodeUnauthorized
}

func IsTooManyRequestsError(code int) bool {
	return code == CodeTooManyRequests
}

// ValidateResponse 验证 API 响应
func ValidateResponse(resp *APIResponse) error {
	if resp.Code == CodeSuccess {
		return nil
	}
	return &BiliError{
		Code:    resp.Code,
		Message: resp.Message,
	}
}
