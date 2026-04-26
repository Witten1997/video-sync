package xhs

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/perkeep/heic"

	_ "image/gif"
	_ "image/png"

	_ "golang.org/x/image/webp"
)

func normalizeLivePhotoImage(ctx context.Context, imagePath string) (string, func(), error) {
	tmpFile, err := os.CreateTemp("", "xhs-live-photo-*.jpg")
	if err != nil {
		return "", nil, fmt.Errorf("创建临时 JPEG 文件失败: %w", err)
	}
	tmpPath := tmpFile.Name()
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return "", nil, fmt.Errorf("关闭临时 JPEG 文件失败: %w", err)
	}

	cleanup := func() {
		_ = os.Remove(tmpPath)
	}

	jpegBytes, goErr := convertToJPEG(imagePath)
	if goErr == nil {
		if err := os.WriteFile(tmpPath, jpegBytes, 0644); err != nil {
			cleanup()
			return "", nil, fmt.Errorf("写入临时 JPEG 失败: %w", err)
		}
		return tmpPath, cleanup, nil
	}

	ffmpegErr := convertImageToJPEGWithFFmpeg(ctx, imagePath, tmpPath)
	if ffmpegErr != nil {
		cleanup()
		return "", nil, fmt.Errorf("Go 转 JPEG 失败: %v; ffmpeg 转 JPEG 失败: %w", goErr, ffmpegErr)
	}
	return tmpPath, cleanup, nil
}

func normalizeMotionVideo(ctx context.Context, videoPath string) (string, func(), error) {
	if strings.EqualFold(filepath.Ext(videoPath), ".mp4") {
		return videoPath, func() {}, nil
	}

	tmpFile, err := os.CreateTemp("", "xhs-live-motion-*.mp4")
	if err != nil {
		return "", nil, fmt.Errorf("创建临时 MP4 文件失败: %w", err)
	}
	tmpPath := tmpFile.Name()
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return "", nil, fmt.Errorf("关闭临时 MP4 文件失败: %w", err)
	}

	cleanup := func() {
		_ = os.Remove(tmpPath)
	}

	remuxErr := remuxMotionVideoToMP4(ctx, videoPath, tmpPath)
	if remuxErr == nil {
		return tmpPath, cleanup, nil
	}

	transcodeErr := transcodeMotionVideoToMP4(ctx, videoPath, tmpPath)
	if transcodeErr != nil {
		cleanup()
		return "", nil, fmt.Errorf("ffmpeg remux 失败: %v; ffmpeg 转码失败: %w", remuxErr, transcodeErr)
	}
	return tmpPath, cleanup, nil
}

func convertImageToJPEGWithFFmpeg(ctx context.Context, src, dst string) error {
	return runFFmpeg(ctx,
		"-i", src,
		"-frames:v", "1",
		"-q:v", "2",
		dst,
	)
}

func remuxMotionVideoToMP4(ctx context.Context, src, dst string) error {
	return runFFmpeg(ctx,
		"-i", src,
		"-map", "0:v:0",
		"-map", "0:a:0?",
		"-c", "copy",
		"-movflags", "+faststart",
		dst,
	)
}

func transcodeMotionVideoToMP4(ctx context.Context, src, dst string) error {
	return runFFmpeg(ctx,
		"-i", src,
		"-map", "0:v:0",
		"-map", "0:a:0?",
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-c:a", "aac",
		"-movflags", "+faststart",
		dst,
	)
}

func runFFmpeg(ctx context.Context, args ...string) error {
	cmdArgs := append([]string{"-hide_banner", "-loglevel", "error", "-y"}, args...)
	cmd := exec.CommandContext(ctx, "ffmpeg", cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}

	msg := strings.TrimSpace(string(out))
	if msg == "" {
		return fmt.Errorf("执行 ffmpeg 失败: %w", err)
	}
	return fmt.Errorf("执行 ffmpeg 失败: %w: %s", err, msg)
}

// convertToJPEG 将任意支持的图片格式（JPEG/PNG/GIF/WebP）解码并重新编码为 JPEG 字节
func convertToJPEG(imagePath string) ([]byte, error) {
	f, err := os.Open(imagePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, err := decodeImageFile(f, imagePath)
	if err != nil {
		return nil, fmt.Errorf("解码图片失败: %w", err)
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		return nil, fmt.Errorf("编码 JPEG 失败: %w", err)
	}
	return buf.Bytes(), nil
}

func decodeImageFile(f *os.File, imagePath string) (image.Image, error) {
	ext := strings.ToLower(filepath.Ext(imagePath))
	switch ext {
	case ".heic", ".heif":
		img, err := heic.Decode(f)
		if err != nil {
			return nil, fmt.Errorf("HEIC 解码失败: %w", err)
		}
		return img, nil
	}

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return img, nil
}
