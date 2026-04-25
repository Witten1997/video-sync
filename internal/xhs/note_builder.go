package xhs

import (
	"encoding/json"
	"strconv"
	"strings"
)

// buildNoteFromJSON 从笔记 JSON 节点构造 Note 对象
func buildNoteFromJSON(node map[string]json.RawMessage) (*Note, error) {
	note := &Note{}

	// 笔记ID
	note.NoteID = strRaw(node, "noteId")

	// 标题与描述
	note.Title = strRaw(node, "title")
	note.Description = strRaw(node, "desc")
	if note.Description == "" {
		note.Description = strRaw(node, "description")
	}

	// 作者信息
	if rawUser, ok := node["user"]; ok {
		var u map[string]json.RawMessage
		if json.Unmarshal(rawUser, &u) == nil {
			note.Author.UserID = firstStr(u, "userId", "userid", "user_id", "id")
			note.Author.Nickname = firstStr(u, "nickname", "nickName", "name", "userName")
			note.Author.Avatar = firstStr(u, "avatar", "image")
			note.Author.RedID = firstStr(u, "redId", "red_id", "redID")
		}
	}

	// 发布时间
	note.PublishTime = parseTime(node)

	// 标签
	if rawTags, ok := node["tagList"]; ok {
		var tags []map[string]json.RawMessage
		if json.Unmarshal(rawTags, &tags) == nil {
			for _, t := range tags {
				if name := strRaw(t, "name"); name != "" {
					note.Tags = append(note.Tags, name)
				}
			}
		}
	}

	// 笔记类型与媒体
	hasVideo := extractVideoMedia(node, note)
	hasImages := extractImageMedia(node, note)

	if hasVideo && !hasImages {
		note.Type = NoteTypeVideo
	} else {
		note.Type = NoteTypeNormal
	}

	if !hasVideo && !hasImages {
		// 如果没有匹配到任何媒体，仍然返回，但调用方应判断
	}

	return note, nil
}

// extractVideoMedia 提取视频媒体（视频笔记），追加到 note.MediaItems。返回是否有视频。
func extractVideoMedia(node map[string]json.RawMessage, note *Note) bool {
	rawVideo, ok := node["video"]
	if !ok {
		return false
	}
	var video map[string]json.RawMessage
	if err := json.Unmarshal(rawVideo, &video); err != nil {
		return false
	}

	// 优先：consumer.originVideoKey → 拼接 sns-video-bd.xhscdn.com
	if rawConsumer, ok := video["consumer"]; ok {
		var consumer map[string]json.RawMessage
		if json.Unmarshal(rawConsumer, &consumer) == nil {
			if key := strRaw(consumer, "originVideoKey"); key != "" {
				note.MediaItems = append(note.MediaItems, MediaItem{
					Type:     MediaTypeVideo,
					VideoURL: "https://sns-video-bd.xhscdn.com/" + key,
				})
				return true
			}
		}
	}

	// 备选：media.stream.h265[0].masterUrl 或 h264[0].masterUrl
	if rawMedia, ok := video["media"]; ok {
		var media map[string]json.RawMessage
		if json.Unmarshal(rawMedia, &media) == nil {
			if rawStream, ok := media["stream"]; ok {
				var stream map[string]json.RawMessage
				if json.Unmarshal(rawStream, &stream) == nil {
					for _, codec := range []string{"h265", "h264"} {
						if rawArr, ok := stream[codec]; ok {
							var arr []map[string]json.RawMessage
							if json.Unmarshal(rawArr, &arr) == nil && len(arr) > 0 {
								if u := firstStr(arr[0], "masterUrl", "url"); u != "" {
									note.MediaItems = append(note.MediaItems, MediaItem{
										Type:     MediaTypeVideo,
										VideoURL: u,
										Width:    intRaw(arr[0], "width"),
										Height:   intRaw(arr[0], "height"),
									})
									return true
								}
							}
						}
					}
				}
			}
		}
	}
	return false
}

// extractImageMedia 提取图片/动态照片媒体，追加到 note.MediaItems。返回是否有图片。
func extractImageMedia(node map[string]json.RawMessage, note *Note) bool {
	var rawList json.RawMessage
	if v, ok := node["imageList"]; ok {
		rawList = v
	} else if v, ok := node["images"]; ok {
		rawList = v
	} else {
		return false
	}

	var images []map[string]json.RawMessage
	if err := json.Unmarshal(rawList, &images); err != nil || len(images) == 0 {
		return false
	}

	for _, img := range images {
		imageURL := pickImageURL(img)
		if imageURL == "" {
			continue
		}
		// 转换为原图地址
		imageURL = transformXhsCdnURL(imageURL)

		liveVideoURL := pickLivePhotoVideoURL(img)
		item := MediaItem{
			ImageURL: imageURL,
			Width:    intRaw(img, "width"),
			Height:   intRaw(img, "height"),
		}
		if liveVideoURL != "" {
			item.Type = MediaTypeLivePhoto
			item.VideoURL = liveVideoURL
		} else {
			item.Type = MediaTypeImage
		}
		note.MediaItems = append(note.MediaItems, item)
	}
	return len(images) > 0
}

// pickImageURL 从单个图片对象中挑选最佳图片 URL
func pickImageURL(img map[string]json.RawMessage) string {
	if u := strRaw(img, "urlDefault"); u != "" {
		return u
	}
	if u := strRaw(img, "url"); u != "" {
		return u
	}
	if t := strRaw(img, "traceId"); t != "" {
		return "https://sns-img-qc.xhscdn.com/" + t
	}
	if rawInfo, ok := img["infoList"]; ok {
		var infoList []map[string]json.RawMessage
		if json.Unmarshal(rawInfo, &infoList) == nil {
			for _, info := range infoList {
				if u := strRaw(info, "url"); u != "" {
					return u
				}
			}
		}
	}
	return ""
}

// pickLivePhotoVideoURL 从图片对象的 stream.h264[0] 中提取动态照片视频 URL
func pickLivePhotoVideoURL(img map[string]json.RawMessage) string {
	rawStream, ok := img["stream"]
	if !ok {
		return ""
	}
	var stream map[string]json.RawMessage
	if json.Unmarshal(rawStream, &stream) != nil {
		return ""
	}
	for _, codec := range []string{"h264", "h265"} {
		if rawArr, ok := stream[codec]; ok {
			var arr []map[string]json.RawMessage
			if json.Unmarshal(rawArr, &arr) == nil && len(arr) > 0 {
				if u := firstStr(arr[0], "masterUrl", "url"); u != "" {
					return u
				}
			}
		}
	}
	return ""
}

// transformXhsCdnURL 将 xhscdn.com 缩略图 URL 转换为 ci.xiaohongshu.com 的原图 URL
func transformXhsCdnURL(u string) string {
	if u == "" || !strings.Contains(u, "xhscdn.com") {
		return u
	}
	// 视频不转换
	if strings.Contains(u, "sns-video") || strings.Contains(u, "/spectrum/") {
		return u
	}
	parts := strings.Split(u, "/")
	if len(parts) <= 5 {
		return u
	}
	token := strings.Join(parts[5:], "/")
	if i := strings.IndexAny(token, "!?"); i >= 0 {
		token = token[:i]
	}
	if token == "" {
		return u
	}
	return "https://ci.xiaohongshu.com/" + token
}

// strRaw 从 map 中读取字符串字段
func strRaw(m map[string]json.RawMessage, key string) string {
	raw, ok := m[key]
	if !ok {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	return ""
}

// intRaw 从 map 中读取整数字段
func intRaw(m map[string]json.RawMessage, key string) int {
	raw, ok := m[key]
	if !ok {
		return 0
	}
	var n int
	if json.Unmarshal(raw, &n) == nil {
		return n
	}
	var f float64
	if json.Unmarshal(raw, &f) == nil {
		return int(f)
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		if v, err := strconv.Atoi(s); err == nil {
			return v
		}
	}
	return 0
}

// firstStr 按多个候选键挑选第一个非空字符串
func firstStr(m map[string]json.RawMessage, keys ...string) string {
	for _, k := range keys {
		if v := strRaw(m, k); v != "" {
			return v
		}
	}
	return ""
}

// parseTime 解析发布时间（毫秒）
func parseTime(node map[string]json.RawMessage) int64 {
	for _, key := range []string{"time", "publishTime", "publish_time", "createTime", "timestamp"} {
		raw, ok := node[key]
		if !ok {
			continue
		}
		var n int64
		if json.Unmarshal(raw, &n) == nil && n > 0 {
			if n < 1_000_000_000_000 {
				n *= 1000
			}
			return n
		}
		var s string
		if json.Unmarshal(raw, &s) == nil && s != "" {
			if v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64); err == nil && v > 0 {
				if v < 1_000_000_000_000 {
					v *= 1000
				}
				return v
			}
		}
	}
	return 0
}
