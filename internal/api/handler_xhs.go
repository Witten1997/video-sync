package api

import (
	"net/http"

	"bili-download/internal/xhs"

	"github.com/gin-gonic/gin"
)

// xhsDownloadRequest 小红书下载请求
type xhsDownloadRequest struct {
	URL string `json:"url" binding:"required"`
}

// handleXHSParse 仅解析笔记元信息，不下载（前端预览用）
func (s *Server) handleXHSParse(c *gin.Context) {
	var req xhsDownloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, "参数无效: "+err.Error())
		return
	}
	client := xhs.NewClient(s.config, "")
	note, err := client.Parser().Parse(c.Request.Context(), req.URL)
	if err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, note)
}

// handleXHSDownload 解析并下载小红书笔记
func (s *Server) handleXHSDownload(c *gin.Context) {
	var req xhsDownloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, "参数无效: "+err.Error())
		return
	}
	client := xhs.NewClient(s.config, "")
	result, err := client.DownloadByURL(c.Request.Context(), req.URL, nil)
	if err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, result)
}
