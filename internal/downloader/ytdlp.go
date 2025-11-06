package downloader

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"bili-download/internal/config"
	"bili-download/internal/utils"
)

// YtdlpDownloader yt-dlp 下载器
type YtdlpDownloader struct {
	config  *config.Config
	tracker *ProgressTracker
}

// NewYtdlpDownloader 创建 yt-dlp 下载器
func NewYtdlpDownloader(cfg *config.Config, tracker *ProgressTracker) *YtdlpDownloader {
	return &YtdlpDownloader{
		config:  cfg,
		tracker: tracker,
	}
}

// DownloadOptions 下载选项
type DownloadOptions struct {
	URL            string   // 视频URL
	OutputPath     string   // 输出路径
	OutputTemplate string   // 输出文件名模板
	Cookies        string   // Cookies 文件路径
	Headers        []string // 自定义请求头
	Format         string   // 视频格式
	SubtitleLangs  []string // 字幕语言
	WriteSubtitles bool     // 是否下载字幕
	WriteThumbnail bool     // 是否下载缩略图
	ExtraArgs      []string // 额外参数
}

// buildCommand 构建 yt-dlp 命令
func (d *YtdlpDownloader) buildCommand(ctx context.Context, opts *DownloadOptions) *exec.Cmd {
	args := []string{
		"--newline",               // 每行输出新进度
		"--no-colors",             // 禁用颜色输出
		"--progress",              // 显示进度
		"--no-warnings",           // 不显示警告
		"--no-check-certificates", // 不检查证书
		"--no-playlist",           // 不下载播放列表
		"--encoding", "UTF-8",     // 输出编码
	}

	// 输出路径和模板
	if opts.OutputPath != "" {
		args = append(args, "-o", filepath.Join(opts.OutputPath, opts.OutputTemplate))
	}

	// Cookies
	if opts.Cookies != "" {
		args = append(args, "--cookies", opts.Cookies)
	}

	// 自定义请求头
	for _, header := range opts.Headers {
		args = append(args, "--add-header", header)
	}

	// 视频格式
	if opts.Format != "" {
		args = append(args, "-f", opts.Format)
	}

	// 字幕
	if opts.WriteSubtitles {
		args = append(args, "--write-subs", "--write-auto-subs", "--sub-format", "srt")
		if len(opts.SubtitleLangs) > 0 {
			args = append(args, "--sub-langs", strings.Join(opts.SubtitleLangs, ","))
		}
	}

	// 缩略图
	if opts.WriteThumbnail {
		args = append(args, "--write-thumbnail")
	}

	// 进度输出为 JSON
	args = append(args, "--progress-template", "%(progress)j")

	// 额外参数
	args = append(args, opts.ExtraArgs...)

	// URL
	args = append(args, opts.URL)

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	return cmd
}

// parseProgressLine 解析进度行
func (d *YtdlpDownloader) parseProgressLine(line string) (*ProgressInfo, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, nil
	}

	// 尝试解析 JSON 格式的进度信息
	var progress ProgressInfo
	if err := json.Unmarshal([]byte(line), &progress); err != nil {
		// 如果不是 JSON，尝试解析文本格式
		return d.parseTextProgress(line)
	}

	return &progress, nil
}

// parseTextProgress 解析文本格式的进度（备用方案）
func (d *YtdlpDownloader) parseTextProgress(line string) (*ProgressInfo, error) {
	// 示例: [download]  45.0% of 123.45MiB at 1.23MiB/s ETA 00:30
	if !strings.Contains(line, "[download]") {
		return nil, nil
	}

	progress := &ProgressInfo{
		Status: "downloading",
	}

	// 这里可以添加更复杂的文本解析逻辑
	// 暂时返回基本信息
	return progress, nil
}

// DownloadVideo 下载视频
func (d *YtdlpDownloader) DownloadVideo(ctx context.Context, opts *DownloadOptions, progressCallback func(*ProgressInfo)) error {
	// 确保输出目录存在
	if opts.OutputPath != "" {
		if err := os.MkdirAll(opts.OutputPath, 0755); err != nil {
			return fmt.Errorf("创建输出目录失败: %w", err)
		}
	}

	// 构建命令
	cmd := d.buildCommand(ctx, opts)

	// 获取标准输出和错误输出管道
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("获取标准输出失败: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("获取标准错误失败: %w", err)
	}

	// 启动命令
	utils.Info("执行 yt-dlp 命令: %s", cmd.String())
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 yt-dlp 失败: %w", err)
	}

	// 读取标准输出（进度信息）
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if progress, err := d.parseProgressLine(line); err == nil && progress != nil {
				if progressCallback != nil {
					progressCallback(progress)
				}
			}
		}
	}()

	// 读取标准错误（错误信息）
	var errorOutput strings.Builder
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			errorOutput.WriteString(line)
			errorOutput.WriteString("\n")
			utils.Warn("yt-dlp stderr: %s", line)
		}
	}()

	// 等待命令完成
	if err := cmd.Wait(); err != nil {
		if errorOutput.Len() > 0 {
			return fmt.Errorf("yt-dlp 执行失败: %w, 错误输出: %s", err, errorOutput.String())
		}
		return fmt.Errorf("yt-dlp 执行失败: %w", err)
	}

	return nil
}

// DownloadWithRetry 带重试的下载
func (d *YtdlpDownloader) DownloadWithRetry(ctx context.Context, opts *DownloadOptions, maxRetries int, progressCallback func(*ProgressInfo)) error {
	var lastErr error

	for retry := 0; retry <= maxRetries; retry++ {
		if retry > 0 {
			utils.Info("重试下载 (第 %d/%d 次)", retry, maxRetries)
			// 重试前等待一段时间
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(retry) * 5 * time.Second):
			}
		}

		err := d.DownloadVideo(ctx, opts, progressCallback)
		if err == nil {
			return nil
		}

		lastErr = err
		utils.Error("下载失败: %v", err)

		// 检查是否是不可重试的错误
		if isNonRetryableError(err) {
			return fmt.Errorf("不可重试的错误: %w", err)
		}
	}

	return fmt.Errorf("下载失败，已重试 %d 次: %w", maxRetries, lastErr)
}

// isNonRetryableError 检查是否是不可重试的错误
func isNonRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()
	nonRetryablePatterns := []string{
		"Video unavailable",
		"Private video",
		"Deleted video",
		"This video is not available",
		"requested format not available",
		"Unsupported URL",
	}

	for _, pattern := range nonRetryablePatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

// GetVideoInfo 获取视频信息（不下载）
func (d *YtdlpDownloader) GetVideoInfo(ctx context.Context, url string, cookies string) (map[string]interface{}, error) {
	args := []string{
		"--dump-json",
		"--no-warnings",
		"--no-playlist",
	}

	if cookies != "" {
		args = append(args, "--cookies", cookies)
	}

	args = append(args, url)

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("获取视频信息失败: %w, 输出: %s", err, string(output))
	}

	var info map[string]interface{}
	if err := json.Unmarshal(output, &info); err != nil {
		return nil, fmt.Errorf("解析视频信息失败: %w", err)
	}

	return info, nil
}

// CheckYtdlpAvailable 检查 yt-dlp 是否可用
func CheckYtdlpAvailable() error {
	cmd := exec.Command("yt-dlp", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("yt-dlp 不可用: %w", err)
	}

	version := strings.TrimSpace(string(output))
	utils.Info("检测到 yt-dlp 版本: %s", version)
	return nil
}
