package downloader

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ProbeResult ffprobe 探测结果
type ProbeResult struct {
	Width     int
	Height    int
	FrameRate float32
}

type ffprobeOutput struct {
	Streams []struct {
		Width        int    `json:"width"`
		Height       int    `json:"height"`
		RFrameRate   string `json:"r_frame_rate"`
		AvgFrameRate string `json:"avg_frame_rate"`
	} `json:"streams"`
}

// ProbeVideo 使用 ffprobe 探测视频实际分辨率与帧率
func ProbeVideo(ctx context.Context, filePath string) (*ProbeResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height,r_frame_rate,avg_frame_rate",
		"-of", "json",
		filePath,
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe 执行失败: %w", err)
	}

	var parsed ffprobeOutput
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil, fmt.Errorf("解析 ffprobe 输出失败: %w", err)
	}
	if len(parsed.Streams) == 0 {
		return nil, fmt.Errorf("未找到视频流")
	}

	s := parsed.Streams[0]
	fps := parseFrameRate(s.RFrameRate)
	if fps <= 0 {
		fps = parseFrameRate(s.AvgFrameRate)
	}

	return &ProbeResult{
		Width:     s.Width,
		Height:    s.Height,
		FrameRate: fps,
	}, nil
}

// parseFrameRate 解析形如 "60000/1001" 的帧率字符串
func parseFrameRate(s string) float32 {
	if s == "" || s == "0/0" {
		return 0
	}
	parts := strings.SplitN(s, "/", 2)
	if len(parts) == 1 {
		v, _ := strconv.ParseFloat(parts[0], 32)
		return float32(v)
	}
	num, err1 := strconv.ParseFloat(parts[0], 32)
	den, err2 := strconv.ParseFloat(parts[1], 32)
	if err1 != nil || err2 != nil || den == 0 {
		return 0
	}
	return float32(num / den)
}
