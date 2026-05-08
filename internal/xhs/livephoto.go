package xhs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
)

// CreateLivePhoto 将图片+视频合成为 Android Live Photo（"动态照片"）。
//
// 实现：把任意格式的图片标准化成 JPEG，把视频标准化成 MP4，然后字节级拼接（cover.jpg 后追加 motion.mp4）。
// 实测华为相册的识别只看"JPG 末尾紧跟一段合法 MP4"这个结构特征，不依赖任何 EXIF / XMP / mdta 元数据。
func CreateLivePhoto(ctx context.Context, imagePath, videoPath, outputPath string) error {
	normalizedImagePath, cleanupImage, err := normalizeLivePhotoImage(ctx, imagePath)
	if err != nil {
		return fmt.Errorf("图片标准化失败: %w", err)
	}
	defer cleanupImage()

	normalizedVideoPath, cleanupVideo, err := normalizeMotionVideo(ctx, videoPath)
	if err != nil {
		return fmt.Errorf("视频标准化失败: %w", err)
	}
	defer cleanupVideo()

	jpegBytes, err := os.ReadFile(normalizedImagePath)
	if err != nil {
		return fmt.Errorf("读取标准化 JPEG 失败: %w", err)
	}
	if len(jpegBytes) < 2 || jpegBytes[0] != 0xFF || jpegBytes[1] != 0xD8 {
		return fmt.Errorf("无效的 JPEG 头部")
	}
	// 确保有 APP0 JFIF 段。ffmpeg 把 webp/png 等非 JPEG 源转成 jpg 时
	// 不会自动写 JFIF 段，导致华为相册等严格解析器拒绝识别为正常 JPEG。
	jpegBytes = ensureJFIFAPP0(jpegBytes)

	videoInfo, err := os.Stat(normalizedVideoPath)
	if err != nil {
		return fmt.Errorf("读取标准化视频失败: %w", err)
	}
	mp4Size := videoInfo.Size()
	if mp4Size <= 0 {
		return fmt.Errorf("视频文件为空: %s", normalizedVideoPath)
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer out.Close()

	if _, err := out.Write(jpegBytes); err != nil {
		return fmt.Errorf("写入 JPEG 失败: %w", err)
	}

	// 16 字节 0 padding：与已验证的 test_G 样本对齐，避免边角解析差异。
	if _, err := out.Write(make([]byte, 16)); err != nil {
		return fmt.Errorf("写入 padding 失败: %w", err)
	}

	video, err := os.Open(normalizedVideoPath)
	if err != nil {
		return fmt.Errorf("打开标准化视频失败: %w", err)
	}
	defer video.Close()

	if _, err := io.Copy(out, video); err != nil {
		return fmt.Errorf("追加视频失败: %w", err)
	}

	// 40 字节 LIVE footer：华为相册识别动态照片的私有魔法尾。
	// 格式：<W:H 空格补到 20 字节><LIVE_<mp4字节数> 空格补到 20 字节>
	// W:H 数值不参与校验（真机文件也写不匹配的值），mp4 字节数必须等于追加的 mp4 长度。
	if _, err := out.Write(buildHuaweiLiveFooter(mp4Size)); err != nil {
		return fmt.Errorf("写入 LIVE footer 失败: %w", err)
	}
	return nil
}

func buildHuaweiLiveFooter(mp4Size int64) []byte {
	footer := bytes.Repeat([]byte{' '}, 40)
	copy(footer[:20], []byte("1024:542"))
	copy(footer[20:], []byte(fmt.Sprintf("LIVE_%d", mp4Size)))
	return footer
}

// ensureJFIFAPP0 检查 SOI 后是否有 APP0 'JFIF' 段，没有就在 SOI 之后插入一个标准
// JFIF 段（version 1.02, no units, density 96x96, 无缩略图）。
func ensureJFIFAPP0(data []byte) []byte {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return data
	}
	// 查紧跟 SOI 的下一个 marker
	if data[2] == 0xFF && data[3] == 0xE0 && len(data) >= 11 && string(data[6:10]) == "JFIF" {
		return data
	}
	jfif := []byte{
		0xFF, 0xE0, 0x00, 0x10,
		'J', 'F', 'I', 'F', 0x00,
		0x01, 0x02, 0x00,
		0x00, 0x60, 0x00, 0x60,
		0x00, 0x00,
	}
	out := make([]byte, 0, len(data)+len(jfif))
	out = append(out, data[:2]...)
	out = append(out, jfif...)
	out = append(out, data[2:]...)
	return out
}
