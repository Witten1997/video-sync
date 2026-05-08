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

	// 优先用 ffmpeg：产物自带 APP0 JFIF + APP2 ICC 段，结构与华为相机/萌制作一致；
	// Go 的 jpeg.Encode 出的是裸 JPEG（无 APP0 JFIF 段），部分严格的解析器会拒绝识别。
	ffmpegErr := convertImageToJPEGWithFFmpeg(ctx, imagePath, tmpPath)
	if ffmpegErr == nil {
		return tmpPath, cleanup, nil
	}

	jpegBytes, goErr := convertToJPEG(imagePath, 95)
	if goErr != nil {
		cleanup()
		return "", nil, fmt.Errorf("ffmpeg 转 JPEG 失败: %v; Go 转 JPEG 失败: %w", ffmpegErr, goErr)
	}
	if err := os.WriteFile(tmpPath, jpegBytes, 0644); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("写入临时 JPEG 失败: %w", err)
	}
	return tmpPath, cleanup, nil
}

func normalizeMotionVideo(ctx context.Context, videoPath string) (string, func(), error) {
	// 不再对 .mp4 直接透传：xhs 的 live 图视频通常没有音轨，但华为相册要求
	// live photo 的 mp4 必须含音频轨道，所以一定要走 ffmpeg 重新封装+补一条静音 AAC 音轨。
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

// remuxMotionVideoToMP4 用流拷贝重封装为 mp4。华为相册要求 live photo 的 mp4 必须包含音频
// 轨道，否则不识别——所以源没有音轨时注入一段静音 AAC。
func remuxMotionVideoToMP4(ctx context.Context, src, dst string) error {
	// 路径 A：源带音轨，整体 -c copy
	if err := runFFmpeg(ctx,
		"-i", src,
		"-map", "0:v:0",
		"-map", "0:a:0",
		"-c", "copy",
		"-movflags", "+faststart",
		dst,
	); err == nil {
		return nil
	}
	// 路径 B：源无音轨，注入静音
	return runFFmpeg(ctx,
		"-i", src,
		"-f", "lavfi",
		"-i", "anullsrc=channel_layout=stereo:sample_rate=44100",
		"-map", "0:v:0",
		"-map", "1:a:0",
		"-shortest",
		"-c:v", "copy",
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart",
		dst,
	)
}

// transcodeMotionVideoToMP4 转码为 H.264+AAC mp4，作为 remux 失败的兜底；同样保证有音轨。
func transcodeMotionVideoToMP4(ctx context.Context, src, dst string) error {
	if err := runFFmpeg(ctx,
		"-i", src,
		"-map", "0:v:0",
		"-map", "0:a:0",
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart",
		dst,
	); err == nil {
		return nil
	}
	return runFFmpeg(ctx,
		"-i", src,
		"-f", "lavfi",
		"-i", "anullsrc=channel_layout=stereo:sample_rate=44100",
		"-map", "0:v:0",
		"-map", "1:a:0",
		"-shortest",
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart",
		dst,
	)
}

// convertToJPEG 将任意支持的图片格式（JPEG/PNG/GIF/WebP/HEIC）解码并重新编码为 JPEG 字节
func convertToJPEG(imagePath string, quality int) ([]byte, error) {
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
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
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
