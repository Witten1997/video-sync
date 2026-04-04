package api

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"bili-download/internal/utils"
	"bili-download/internal/version"

	"github.com/gin-gonic/gin"
)

// handleGetVersion 获取版本信息
func (s *Server) handleGetVersion(c *gin.Context) {
	info := s.getCheckVersion()
	respondSuccess(c, gin.H{
		"current_version": version.Version,
		"git_tag":         version.GitTag,
		"build_time":      version.BuildTime,
		"has_update":      info.HasUpdate,
		"new_version":     info.NewVersion,
		"download_url":    info.DownloadURL,
		"changelog":       info.Changelog,
		"published_at":    info.PublishedAt,
		"checked_at":      info.CheckedAt,
	})
}

// handleCheckVersion 手动触发版本检查
func (s *Server) handleCheckVersion(c *gin.Context) {
	s.doVersionCheck()
	info := s.getCheckVersion()
	respondSuccess(c, gin.H{
		"current_version": version.Version,
		"has_update":      info.HasUpdate,
		"new_version":     info.NewVersion,
		"download_url":    info.DownloadURL,
		"changelog":       info.Changelog,
		"published_at":    info.PublishedAt,
		"checked_at":      info.CheckedAt,
	})
}

// handleUpgrade 执行升级
func (s *Server) handleUpgrade(c *gin.Context) {
	var req struct {
		Version string `json:"version"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "参数错误")
		return
	}

	targetVersion := req.Version
	if targetVersion == "" || targetVersion == "latest" {
		info := s.getCheckVersion()
		if !info.HasUpdate {
			respondError(c, http.StatusBadRequest, "当前已是最新版本")
			return
		}
		targetVersion = info.NewVersion
	}

	// 构建下载URL
	osName := runtime.GOOS
	arch := runtime.GOARCH
	binaryName := "video-sync"
	if osName == "windows" {
		binaryName = "video-sync.exe"
	}
	downloadURL := fmt.Sprintf("https://github.com/Witten1997/video-sync/releases/download/%s/video-sync-%s-%s-%s.tar.gz", targetVersion, targetVersion, osName, arch)

	utils.Info("开始下载升级包: %s", downloadURL)

	// 创建临时目录
	tempDir := filepath.Join("storage", "temp", "upgrade")
	os.MkdirAll(tempDir, 0755)

	tarPath := filepath.Join(tempDir, fmt.Sprintf("video-sync-%s-%s-%s.tar.gz", targetVersion, osName, arch))

	// 下载文件
	if err := s.downloadFile(downloadURL, tarPath); err != nil {
		utils.Error("下载升级包失败: %v", err)
		respondError(c, http.StatusInternalServerError, fmt.Sprintf("下载失败: %v", err))
		return
	}

	// 解压二进制
	extractedPath, err := extractBinary(tarPath, tempDir, binaryName)
	if err != nil {
		utils.Error("解压升级包失败: %v", err)
		os.Remove(tarPath)
		respondError(c, http.StatusInternalServerError, fmt.Sprintf("解压失败: %v", err))
		return
	}

	// 删除tar包
	os.Remove(tarPath)

	utils.Info("升级包准备完成: %s, 即将重启...", extractedPath)

	respondSuccess(c, gin.H{
		"message": "升级包下载完成，正在重启...",
		"version": targetVersion,
	})

	// 异步触发升级重启
	go func() {
		time.Sleep(1 * time.Second)
		s.UpgradeSignal <- extractedPath
	}()
}

// downloadFile 下载文件到指定路径
func (s *Server) downloadFile(url, dest string) error {
	client := utils.NewHTTPClient(s.config.Proxy, 10*time.Minute, 20, 10)
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// extractBinary 从tar.gz中解压二进制文件
func extractBinary(tarPath, destDir, binaryName string) (string, error) {
	f, err := os.Open(tarPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		// 查找目标二进制文件
		name := filepath.Base(header.Name)
		if name == binaryName && header.Typeflag == tar.TypeReg {
			destPath := filepath.Join(destDir, binaryName)
			out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return "", err
			}
			out.Close()
			return destPath, nil
		}
	}

	// 如果tar.gz里没有子目录，可能就是二进制本身
	// 尝试直接当作gzip处理
	return "", fmt.Errorf("在压缩包中未找到 %s", binaryName)
}
