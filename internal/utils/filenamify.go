package utils

import (
	"regexp"
	"strings"
)

var (
	// 不允许在文件名中使用的字符
	invalidFileNameChars = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)

	// 不允许作为文件名的保留字（Windows）
	reservedNames = map[string]bool{
		"CON": true, "PRN": true, "AUX": true, "NUL": true,
		"COM1": true, "COM2": true, "COM3": true, "COM4": true,
		"COM5": true, "COM6": true, "COM7": true, "COM8": true,
		"COM9": true, "LPT1": true, "LPT2": true, "LPT3": true,
		"LPT4": true, "LPT5": true, "LPT6": true, "LPT7": true,
		"LPT8": true, "LPT9": true,
	}
)

// Filenamify 将字符串转换为安全的文件名
func Filenamify(name string) string {
	// 替换不允许的字符为下划线
	name = invalidFileNameChars.ReplaceAllString(name, "_")

	// 去除首尾空格和点
	name = strings.Trim(name, " .")

	// 检查是否为保留字
	upperName := strings.ToUpper(name)
	if reservedNames[upperName] {
		name = "_" + name
	}

	// 限制长度（Windows 最大路径长度限制）
	if len(name) > 200 {
		name = name[:200]
	}

	// 如果结果为空，使用默认名称
	if name == "" {
		name = "unnamed"
	}

	return name
}

// TruncateString 截断字符串到指定长度
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	// 尽量在合适的位置截断（避免截断多字节字符）
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}

	return string(runes[:maxLen]) + "..."
}

// SanitizePath 清理路径，移除危险字符
func SanitizePath(path string) string {
	// 替换反斜杠为正斜杠（统一路径分隔符）
	path = strings.ReplaceAll(path, "\\", "/")

	// 移除路径遍历尝试
	path = strings.ReplaceAll(path, "../", "")
	path = strings.ReplaceAll(path, "./", "")

	// 移除多余的斜杠
	path = regexp.MustCompile(`/+`).ReplaceAllString(path, "/")

	// 去除首尾斜杠
	path = strings.Trim(path, "/")

	return path
}
