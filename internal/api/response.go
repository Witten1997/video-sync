package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 标准 API 响应
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// respondSuccess 成功响应
func respondSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// respondError 错误响应
func respondError(c *gin.Context, httpStatus int, message string) {
	c.JSON(httpStatus, Response{
		Code:    httpStatus,
		Message: message,
	})
}

// respondValidationError 验证错误响应
func respondValidationError(c *gin.Context, message string) {
	respondError(c, http.StatusBadRequest, message)
}

// respondNotFound 未找到响应
func respondNotFound(c *gin.Context, message string) {
	respondError(c, http.StatusNotFound, message)
}

// respondInternalError 内部错误响应
func respondInternalError(c *gin.Context, err error) {
	respondError(c, http.StatusInternalServerError, err.Error())
}
