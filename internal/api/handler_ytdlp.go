package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"bili-download/internal/utils"

	"github.com/gin-gonic/gin"
)

// YtdlpVersionInfo yt-dlp 版本信息
type YtdlpVersionInfo struct {
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	HasUpdate      bool   `json:"has_update"`
	UpdateTime     string `json:"update_time,omitempty"`
	Platform       string `json:"platform"`
	UpdateMethod   string `json:"update_method"`
}

// YtdlpUpdateRequest yt-dlp 更新请求
type YtdlpUpdateRequest struct {
	Force bool `json:"force"`
}

// handleGetYtdlpVersion 获取 yt-dlp 版本信息
func (s *Server) handleGetYtdlpVersion(c *gin.Context) {
	// 获取当前版本
	currentVersion, err := getYtdlpCurrentVersion()
	if err != nil {
		utils.Error("获取 yt-dlp 当前版本失败: %v", err)
		respondError(c, http.StatusInternalServerError, fmt.Sprintf("获取当前版本失败: %v", err))
		return
	}

	// 获取最新版本
	latestVersion, err := getYtdlpLatestVersion()
	if err != nil {
		utils.Error("获取 yt-dlp 最新版本失败: %v", err)
		respondError(c, http.StatusInternalServerError, fmt.Sprintf("获取最新版本失败: %v", err))
		return
	}

	// 比较版本
	hasUpdate := currentVersion != latestVersion

	// 获取更新方法说明
	updateMethod := getUpdateMethod()

	respondSuccess(c, YtdlpVersionInfo{
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
		HasUpdate:      hasUpdate,
		Platform:       runtime.GOOS,
		UpdateMethod:   updateMethod,
	})
}

// handleUpdateYtdlp 更新 yt-dlp
func (s *Server) handleUpdateYtdlp(c *gin.Context) {
	var req YtdlpUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Force = false
	}

	// 获取当前版本
	currentVersion, err := getYtdlpCurrentVersion()
	if err != nil {
		utils.Error("获取 yt-dlp 当前版本失败: %v", err)
		respondError(c, http.StatusInternalServerError, fmt.Sprintf("获取当前版本失败: %v", err))
		return
	}

	utils.Info("开始更新 yt-dlp，当前版本: %s, 平台: %s", currentVersion, runtime.GOOS)

	// 执行更新
	output, err := updateYtdlp()
	if err != nil {
		utils.Error("更新 yt-dlp 失败: %v, 输出: %s", err, output)
		respondError(c, http.StatusInternalServerError, fmt.Sprintf("更新失败: %v", err))
		return
	}

	// 获取更新后的版本
	newVersion, err := getYtdlpCurrentVersion()
	if err != nil {
		utils.Error("获取更新后版本失败: %v", err)
		respondError(c, http.StatusInternalServerError, fmt.Sprintf("更新完成但获取新版本失败: %v", err))
		return
	}

	utils.Info("yt-dlp 更新完成，新版本: %s", newVersion)

	respondSuccess(c, gin.H{
		"success":         true,
		"current_version": newVersion,
		"old_version":     currentVersion,
		"message":         "更新成功",
		"output":          output,
	})
}

// getYtdlpCurrentVersion 获取当前安装的 yt-dlp 版本
func getYtdlpCurrentVersion() (string, error) {
	cmd := exec.Command("yt-dlp", "--version")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("执行命令失败: %w, 输出: %s", err, out.String())
	}

	version := strings.TrimSpace(out.String())
	return version, nil
}

// getYtdlpLatestVersion 获取 yt-dlp 最新版本
func getYtdlpLatestVersion() (string, error) {
	// 从 GitHub API 获取最新版本
	resp, err := http.Get("https://api.github.com/repos/yt-dlp/yt-dlp/releases/latest")
	if err != nil {
		return "", fmt.Errorf("请求 GitHub API 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API 返回错误状态码: %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("解析 GitHub API 响应失败: %w", err)
	}

	// 移除版本号前的 'v' 前缀
	version := strings.TrimPrefix(release.TagName, "v")
	return version, nil
}

// updateYtdlp 更新 yt-dlp (跨平台支持)
func updateYtdlp() (string, error) {
	var cmd *exec.Cmd
	var out bytes.Buffer

	switch runtime.GOOS {
	case "windows":
		// Windows 环境：优先使用 yt-dlp 自更新命令
		// 如果是通过 pip 安装的，也支持 pip 更新
		utils.Info("检测到 Windows 环境，使用 yt-dlp -U 命令更新")
		cmd = exec.Command("yt-dlp", "-U")

	case "linux", "darwin":
		// Linux/Mac 环境：优先使用 pip3 更新
		// 检查是否有 pip3
		if _, err := exec.LookPath("pip3"); err == nil {
			utils.Info("检测到 pip3，使用 pip3 更新")
			cmd = exec.Command("pip3", "install", "--upgrade", "yt-dlp")
		} else if _, err := exec.LookPath("pip"); err == nil {
			utils.Info("使用 pip 更新")
			cmd = exec.Command("pip", "install", "--upgrade", "yt-dlp")
		} else {
			// 没有 pip，使用 yt-dlp 自更新
			utils.Info("未找到 pip，使用 yt-dlp -U 命令更新")
			cmd = exec.Command("yt-dlp", "-U")
		}

	default:
		return "", fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}

	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return out.String(), fmt.Errorf("执行更新命令失败: %w", err)
	}

	return out.String(), nil
}

// getUpdateMethod 获取当前平台的更新方法说明
func getUpdateMethod() string {
	switch runtime.GOOS {
	case "windows":
		return "使用 yt-dlp -U 自更新"
	case "linux", "darwin":
		if _, err := exec.LookPath("pip3"); err == nil {
			return "使用 pip3 install --upgrade yt-dlp"
		}
		if _, err := exec.LookPath("pip"); err == nil {
			return "使用 pip install --upgrade yt-dlp"
		}
		return "使用 yt-dlp -U 自更新"
	default:
		return "未知"
	}
}

// parseVersion 解析版本号（用于版本比较）
func parseVersion(version string) ([]int, error) {
	// 移除非数字字符，保留点号
	re := regexp.MustCompile(`[^\d.]`)
	version = re.ReplaceAllString(version, "")

	parts := strings.Split(version, ".")
	result := make([]int, len(parts))

	for i, part := range parts {
		var num int
		_, err := fmt.Sscanf(part, "%d", &num)
		if err != nil {
			return nil, err
		}
		result[i] = num
	}

	return result, nil
}
