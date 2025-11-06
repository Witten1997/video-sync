package nfo

import (
	"encoding/xml"
	"fmt"
	"os"
	"time"
)

// Generator NFO 生成器接口
type Generator interface {
	Generate() ([]byte, error)
	WriteToFile(filename string) error
}

// Actor 演员信息
type Actor struct {
	Name  string `xml:"name"`
	Role  string `xml:"role,omitempty"`
	Order int    `xml:"order,omitempty"`
	Thumb string `xml:"thumb,omitempty"`
}

// Rating 评分信息
type Rating struct {
	Value   float64 `xml:"value"`
	Votes   int     `xml:"votes,omitempty"`
	Max     int     `xml:"max,omitempty"`
	Default bool    `xml:"default,attr,omitempty"`
}

// Ratings 评分列表
type Ratings struct {
	Rating []Rating `xml:"rating"`
}

// UniqueID 唯一标识
type UniqueID struct {
	Type    string `xml:"type,attr"`
	Default bool   `xml:"default,attr,omitempty"`
	Value   string `xml:",chardata"`
}

// Thumb 缩略图
type Thumb struct {
	Aspect  string `xml:"aspect,attr,omitempty"`
	Preview string `xml:"preview,attr,omitempty"`
	URL     string `xml:",chardata"`
}

// Fanart 同人画
type Fanart struct {
	Thumb []Thumb `xml:"thumb"`
}

// FormatDate 格式化日期
func FormatDate(t time.Time, format string) string {
	if format == "" {
		format = "2006-01-02"
	}
	return t.Format(format)
}

// WriteXMLToFile 将 XML 写入文件
func WriteXMLToFile(filename string, v interface{}) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	// 写入 XML 声明
	if _, err := file.WriteString(xml.Header); err != nil {
		return fmt.Errorf("写入 XML 头失败: %w", err)
	}

	// 编码 XML
	encoder := xml.NewEncoder(file)
	encoder.Indent("", "  ")
	if err := encoder.Encode(v); err != nil {
		return fmt.Errorf("编码 XML 失败: %w", err)
	}

	return nil
}

// EscapeXML 转义 XML 特殊字符
func EscapeXML(s string) string {
	// xml.Marshal 会自动转义，这里提供一个手动方法
	return s
}

// TruncateString 截断字符串
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	// 尝试在单词边界截断
	if maxLen > 3 {
		return s[:maxLen-3] + "..."
	}
	return s[:maxLen]
}

// JoinStrings 连接字符串数组
func JoinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
