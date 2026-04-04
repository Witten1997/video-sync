package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

// NormalizeRelativePath 将配置中的相对路径标准化。
// 允许空字符串和前导分隔符，但不允许跳出基础目录或使用盘符绝对路径。
func NormalizeRelativePath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", nil
	}

	if filepath.VolumeName(trimmed) != "" {
		return "", fmt.Errorf("路径不能包含盘符或绝对路径: %s", path)
	}

	trimmed = strings.TrimLeft(trimmed, `/\`)
	if trimmed == "" {
		return "", nil
	}

	cleaned := filepath.Clean(trimmed)
	if cleaned == "." {
		return "", nil
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("路径不能跳出下载基础目录: %s", path)
	}

	return cleaned, nil
}

// NormalizedURLDownloadPath 返回标准化后的 URL 下载相对路径。
func (c PathsConfig) NormalizedURLDownloadPath() (string, error) {
	return NormalizeRelativePath(c.URLDownloadPath)
}

// URLDownloadBase 返回 URL 下载的实际基础目录。
func (c PathsConfig) URLDownloadBase() string {
	base := filepath.Clean(c.DownloadBase)
	relative, err := c.NormalizedURLDownloadPath()
	if err != nil || relative == "" {
		return base
	}
	return filepath.Join(base, relative)
}
