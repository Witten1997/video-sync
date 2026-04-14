package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"bili-download/internal/utils"
	"bili-download/internal/version"
)

// CheckVersionInfo 版本检查结果
type CheckVersionInfo struct {
	HasUpdate   bool   `json:"has_update"`
	NewVersion  string `json:"new_version"`
	DownloadURL string `json:"download_url"`
	Changelog   string `json:"changelog"`
	PublishedAt string `json:"published_at"`
	CheckedAt   string `json:"checked_at"`
}

// githubRelease GitHub release API 响应
type githubRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Body        string `json:"body"`
	HTMLURL     string `json:"html_url"`
	PublishedAt string `json:"published_at"`
}

// startVersionChecker 启动后台版本检查
func (s *Server) startVersionChecker() {
	go func() {
		// 启动后延迟10秒再检查，避免启动时请求过多
		time.Sleep(10 * time.Second)
		s.doVersionCheck()

		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			s.doVersionCheck()
		}
	}()
}

func (s *Server) doVersionCheck() {
	client := utils.NewHTTPClient(s.config.Proxy, 15*time.Second, 10, 5)
	resp, err := client.Get("https://api.github.com/repos/Witten1997/video-sync/releases?per_page=1")
	if err != nil {
		utils.Warn("版本检查失败: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		utils.Warn("版本检查失败: HTTP %d", resp.StatusCode)
		return
	}

	var releases []githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		utils.Warn("解析版本信息失败: %v", err)
		return
	}

	if len(releases) == 0 {
		return
	}

	latest := releases[0]
	latestVer := latest.TagName
	if len(latestVer) > 0 && latestVer[0] == 'v' {
		latestVer = latestVer[1:]
	}

	info := CheckVersionInfo{
		HasUpdate:   isNewerVersion(version.Version, latestVer),
		NewVersion:  latestVer,
		DownloadURL: latest.HTMLURL,
		Changelog:   latest.Body,
		PublishedAt: latest.PublishedAt,
		CheckedAt:   time.Now().Format(time.RFC3339),
	}

	s.checkVersionMu.Lock()
	s.checkVersion = info
	s.checkVersionMu.Unlock()

	if info.HasUpdate {
		utils.Info("发现新版本: %s -> %s", version.Version, latestVer)
	}
}

// getCheckVersion 获取版本检查结果（线程安全）
func (s *Server) getCheckVersion() CheckVersionInfo {
	s.checkVersionMu.RLock()
	defer s.checkVersionMu.RUnlock()
	return s.checkVersion
}

// isNewerVersion 判断 remote 是否比 current 更新（语义化版本比较）
func isNewerVersion(current, remote string) bool {
	var c1, c2, c3, r1, r2, r3 int
	if _, err := fmt.Sscanf(current, "%d.%d.%d", &c1, &c2, &c3); err != nil {
		return false
	}
	if _, err := fmt.Sscanf(remote, "%d.%d.%d", &r1, &r2, &r3); err != nil {
		return false
	}
	if r1 != c1 {
		return r1 > c1
	}
	if r2 != c2 {
		return r2 > c2
	}
	return r3 > c3
}
