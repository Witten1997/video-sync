package xhs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"bili-download/internal/utils"
)

const (
	// userAgent 模拟小红书安卓客户端 UA，绕过部分页面的鉴权
	userAgent = "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Mobile Safari/537.36 xiaohongshu"
)

var (
	// 匹配各种小红书链接
	reXhsShare    = regexp.MustCompile(`(?:https?://)?www\.xiaohongshu\.com/discovery/item/\S+`)
	reXhsExplore  = regexp.MustCompile(`(?:https?://)?www\.xiaohongshu\.com/explore/\S+`)
	reXhsUser     = regexp.MustCompile(`(?:https?://)?www\.xiaohongshu\.com/user/profile/[a-z0-9]+/\S+`)
	reXhsShort    = regexp.MustCompile(`(?:https?://)?xhslink\.com/[^\s"<>\\^` + "`" + `{|}，。；！？、【】《》]+`)
	reExtractID   = regexp.MustCompile(`(?:explore|item)/([a-zA-Z0-9_\-]+)/?(?:\?|$)`)
	reExtractUID  = regexp.MustCompile(`user/profile/[a-z0-9]+/([a-zA-Z0-9_\-]+)/?(?:\?|$)`)
	reInitialJSON = regexp.MustCompile(`(?s)window\.__INITIAL_STATE__\s*=\s*(\{.*?\})\s*</script>`)
)

// Parser 小红书笔记解析器
type Parser struct {
	httpClient *http.Client
}

// NewParser 创建解析器
func NewParser(httpClient *http.Client) *Parser {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Parser{httpClient: httpClient}
}

// ExtractURL 从输入文本中提取第一个有效的小红书链接
func ExtractURL(input string) string {
	if input == "" {
		return ""
	}
	// 优先匹配短链
	if m := reXhsShort.FindString(input); m != "" {
		return m
	}
	if m := reXhsShare.FindString(input); m != "" {
		return m
	}
	if m := reXhsExplore.FindString(input); m != "" {
		return m
	}
	if m := reXhsUser.FindString(input); m != "" {
		return m
	}
	return ""
}

// ResolveShortURL 解析短链接重定向后的真实URL
func (p *Parser) ResolveShortURL(ctx context.Context, shortURL string) (string, error) {
	if !strings.HasPrefix(shortURL, "http") {
		shortURL = "https://" + shortURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, shortURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建短链请求失败: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("解析短链失败: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return resp.Request.URL.String(), nil
}

// ExtractNoteID 从URL中提取笔记ID
func ExtractNoteID(url string) string {
	if m := reExtractID.FindStringSubmatch(url); len(m) > 1 {
		return m[1]
	}
	if m := reExtractUID.FindStringSubmatch(url); len(m) > 1 {
		return m[1]
	}
	return ""
}

// FetchHTML 抓取笔记页面 HTML
func (p *Parser) FetchHTML(ctx context.Context, noteURL string) (string, error) {
	if !strings.HasPrefix(noteURL, "http") {
		noteURL = "https://" + noteURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, noteURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=1.0,*/*;q=1.0")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求页面失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("响应状态码异常: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}
	return string(body), nil
}

// Parse 完整解析流程：URL → 笔记元信息+媒体列表
func (p *Parser) Parse(ctx context.Context, inputURL string) (*Note, error) {
	rawURL := ExtractURL(inputURL)
	if rawURL == "" {
		return nil, fmt.Errorf("未识别到有效的小红书链接")
	}
	if !strings.HasPrefix(rawURL, "http") {
		rawURL = "https://" + rawURL
	}

	finalURL := rawURL
	// 短链需先 follow redirect
	if strings.Contains(rawURL, "xhslink.com") {
		resolved, err := p.ResolveShortURL(ctx, rawURL)
		if err != nil {
			utils.Warn("解析短链失败: %v", err)
		} else if resolved != "" {
			finalURL = resolved
		}
	}

	noteID := ExtractNoteID(finalURL)
	if noteID == "" {
		return nil, fmt.Errorf("无法从链接提取笔记ID: %s", finalURL)
	}

	html, err := p.FetchHTML(ctx, finalURL)
	if err != nil {
		return nil, err
	}

	note, err := parseInitialState(html)
	if err != nil {
		return nil, err
	}
	if note.NoteID == "" {
		note.NoteID = noteID
	}
	note.OriginalURL = finalURL
	return note, nil
}

// parseInitialState 从 HTML 中提取 __INITIAL_STATE__ JSON 并解析为 Note
func parseInitialState(html string) (*Note, error) {
	idx := strings.Index(html, "window.__INITIAL_STATE__")
	if idx < 0 {
		return nil, fmt.Errorf("页面未找到 __INITIAL_STATE__")
	}
	end := strings.Index(html[idx:], "</script>")
	if end < 0 {
		return nil, fmt.Errorf("__INITIAL_STATE__ 脚本未闭合")
	}
	snippet := html[idx : idx+end]
	eq := strings.Index(snippet, "=")
	if eq < 0 {
		return nil, fmt.Errorf("__INITIAL_STATE__ 格式异常")
	}
	jsObject := extractFirstJSObject(snippet[eq+1:])
	if jsObject == "" {
		return nil, fmt.Errorf("无法提取 JS 对象字面量")
	}
	jsObject = strings.TrimRight(strings.TrimSpace(jsObject), ";")
	jsObject = replaceJSUndefined(jsObject)

	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(jsObject), &raw); err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %w", err)
	}
	noteJSON := findNoteJSON(raw)
	if noteJSON == nil {
		return nil, fmt.Errorf("未找到笔记数据")
	}
	return buildNoteFromJSON(noteJSON)
}

// extractFirstJSObject 从 JS 片段中提取第一个完整的 {...} 对象字面量
func extractFirstJSObject(s string) string {
	inString := false
	var quote byte
	escape := false
	depth := 0
	start := -1

	for i := 0; i < len(s); i++ {
		c := s[i]
		if inString {
			if escape {
				escape = false
				continue
			}
			if c == '\\' {
				escape = true
				continue
			}
			if c == quote {
				inString = false
			}
			continue
		}
		if c == '"' || c == '\'' {
			inString = true
			quote = c
			continue
		}
		if c == '{' {
			if depth == 0 {
				start = i
			}
			depth++
		} else if c == '}' {
			if depth > 0 {
				depth--
				if depth == 0 && start != -1 {
					return s[start : i+1]
				}
			}
		}
	}
	return ""
}

// replaceJSUndefined 将裸 undefined 替换为 null（仅在字符串外）
func replaceJSUndefined(s string) string {
	if !strings.Contains(s, "undefined") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	inString := false
	var quote byte
	escape := false
	for i := 0; i < len(s); {
		c := s[i]
		if inString {
			b.WriteByte(c)
			if escape {
				escape = false
			} else if c == '\\' {
				escape = true
			} else if c == quote {
				inString = false
			}
			i++
			continue
		}
		if c == '"' || c == '\'' {
			inString = true
			quote = c
			b.WriteByte(c)
			i++
			continue
		}
		if i+9 <= len(s) && s[i:i+9] == "undefined" {
			prevOK := i == 0 || !isJSIdentChar(s[i-1])
			nextOK := i+9 == len(s) || !isJSIdentChar(s[i+9])
			if prevOK && nextOK {
				b.WriteString("null")
				i += 9
				continue
			}
		}
		b.WriteByte(c)
		i++
	}
	return b.String()
}

func isJSIdentChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '$'
}

// findNoteJSON 在 __INITIAL_STATE__ 根对象中查找笔记 JSON 节点（覆盖多种页面结构）
func findNoteJSON(root map[string]json.RawMessage) map[string]json.RawMessage {
	// 1) root.note.noteDetailMap[*].note
	if rawNote, ok := root["note"]; ok {
		var noteRoot map[string]json.RawMessage
		if json.Unmarshal(rawNote, &noteRoot) == nil {
			if rawMap, ok := noteRoot["noteDetailMap"]; ok {
				var detailMap map[string]json.RawMessage
				if json.Unmarshal(rawMap, &detailMap) == nil {
					for _, v := range detailMap {
						var entry map[string]json.RawMessage
						if json.Unmarshal(v, &entry) == nil {
							if inner, ok := entry["note"]; ok {
								var note map[string]json.RawMessage
								if json.Unmarshal(inner, &note) == nil && isNoteLike(note) {
									return note
								}
							}
						}
					}
				}
			}
			if isNoteLike(noteRoot) {
				return noteRoot
			}
		}
	}

	// 2) root.noteData.data.noteData
	if rawND, ok := root["noteData"]; ok {
		var ndRoot map[string]json.RawMessage
		if json.Unmarshal(rawND, &ndRoot) == nil {
			if rawData, ok := ndRoot["data"]; ok {
				var dataRoot map[string]json.RawMessage
				if json.Unmarshal(rawData, &dataRoot) == nil {
					if rawNote, ok := dataRoot["noteData"]; ok {
						var note map[string]json.RawMessage
						if json.Unmarshal(rawNote, &note) == nil {
							return note
						}
					}
					if rawNote, ok := dataRoot["note"]; ok {
						var note map[string]json.RawMessage
						if json.Unmarshal(rawNote, &note) == nil {
							return note
						}
					}
				}
			}
		}
	}

	// 3) 兜底深度遍历
	return deepFindNote(root, 0)
}

func deepFindNote(node map[string]json.RawMessage, depth int) map[string]json.RawMessage {
	if depth > 8 {
		return nil
	}
	if isNoteLike(node) {
		return node
	}
	for _, v := range node {
		var sub map[string]json.RawMessage
		if json.Unmarshal(v, &sub) == nil {
			if found := deepFindNote(sub, depth+1); found != nil {
				return found
			}
		}
		var arr []json.RawMessage
		if json.Unmarshal(v, &arr) == nil {
			for _, item := range arr {
				var subMap map[string]json.RawMessage
				if json.Unmarshal(item, &subMap) == nil {
					if found := deepFindNote(subMap, depth+1); found != nil {
						return found
					}
				}
			}
		}
	}
	return nil
}

func isNoteLike(node map[string]json.RawMessage) bool {
	if _, ok := node["imageList"]; ok {
		return true
	}
	if _, ok := node["images"]; ok {
		return true
	}
	if _, ok := node["video"]; ok {
		return true
	}
	return false
}
