package xhs

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"bili-download/internal/config"
	"bili-download/internal/utils"
)

// Client 小红书下载客户端，封装解析+下载完整流程
// 后期订阅模块可直接复用此客户端：先 Parse 拿到 Note，再 DownloadNote 下载
type Client struct {
	parser     *Parser
	downloader *Downloader
	baseDir    string // 下载根目录
}

// NewClient 创建客户端
// baseDir 为空时使用 cfg.Paths.DownloadBase 的 xhs 子目录
func NewClient(cfg *config.Config, baseDir string) *Client {
	httpClient := utils.NewHTTPClient(cfg.Proxy, 60*time.Second, 50, 10)
	if baseDir == "" {
		base := cfg.Paths.DownloadBase
		if base == "" {
			base = "./downloads"
		}
		baseDir = filepath.Join(base, "xhs")
	}
	return &Client{
		parser:     NewParser(httpClient),
		downloader: NewDownloader(httpClient),
		baseDir:    baseDir,
	}
}

// Parser 暴露解析器（订阅模块可单独调用）
func (c *Client) Parser() *Parser { return c.parser }

// Downloader 暴露下载器（订阅模块可单独调用）
func (c *Client) Downloader() *Downloader { return c.downloader }

// BaseDir 返回下载根目录
func (c *Client) BaseDir() string { return c.baseDir }

// DownloadByURL 一站式：解析链接 → 下载到 baseDir/作者ID/笔记ID/
func (c *Client) DownloadByURL(ctx context.Context, inputURL string, onProgress ProgressCallback) (*DownloadResult, error) {
	note, err := c.parser.Parse(ctx, inputURL)
	if err != nil {
		return nil, fmt.Errorf("解析失败: %w", err)
	}
	if len(note.MediaItems) == 0 {
		return nil, fmt.Errorf("笔记未发现可下载的媒体")
	}
	outputDir := c.buildOutputDir(note)
	utils.Info("XHS 笔记: %s, 媒体数: %d, 输出: %s", note.NoteID, len(note.MediaItems), outputDir)
	return c.downloader.DownloadNote(ctx, note, outputDir, onProgress)
}

// buildOutputDir 构造输出目录：baseDir/作者昵称_作者ID/笔记ID
func (c *Client) buildOutputDir(note *Note) string {
	authorDir := "unknown"
	if note.Author.Nickname != "" || note.Author.UserID != "" {
		authorDir = utils.Filenamify(note.Author.Nickname + "_" + note.Author.UserID)
	}
	noteDir := note.NoteID
	if noteDir == "" {
		noteDir = "untitled"
	}
	return filepath.Join(c.baseDir, authorDir, noteDir)
}

// NewDefaultHTTPClient 提供默认 HTTP 客户端（不依赖 config，便于独立调用）
func NewDefaultHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout}
}
