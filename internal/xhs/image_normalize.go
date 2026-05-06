package xhs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// normalizeImageHEIC 若文件为 HEIC/HEIF，转为 JPEG（q=92）并删除原文件
// 返回新路径与新大小；非 HEIC 时原样返回
func normalizeImageHEIC(ctx context.Context, srcPath string) (string, int64, error) {
	ext := strings.ToLower(filepath.Ext(srcPath))
	if ext != ".heic" && ext != ".heif" {
		return srcPath, fileSize(srcPath), nil
	}

	dstPath := strings.TrimSuffix(srcPath, filepath.Ext(srcPath)) + ".jpg"

	jpegBytes, goErr := convertToJPEG(srcPath, 92)
	if goErr == nil {
		if err := os.WriteFile(dstPath, jpegBytes, 0644); err != nil {
			return srcPath, fileSize(srcPath), fmt.Errorf("写入 JPEG 失败: %w", err)
		}
		_ = os.Remove(srcPath)
		return dstPath, fileSize(dstPath), nil
	}

	if err := convertImageToJPEGWithFFmpeg(ctx, srcPath, dstPath); err != nil {
		return srcPath, fileSize(srcPath), fmt.Errorf("Go 转 JPEG 失败: %v; ffmpeg 失败: %w", goErr, err)
	}
	_ = os.Remove(srcPath)
	return dstPath, fileSize(dstPath), nil
}

func fileSize(path string) int64 {
	fi, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return fi.Size()
}
