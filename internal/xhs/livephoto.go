package xhs

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"os"
	"strings"

	_ "image/gif"
	_ "image/png"

	_ "golang.org/x/image/webp"
)

// CreateLivePhoto 将图片+视频合成为 Android Live Photo（Google MotionPhoto / 小米 MicroVideo 格式）
//
// 实现原理：
//  1. 把任意格式的图片解码并重新编码为 JPEG（保证容器一致）
//  2. 在 JPEG 的 SOI(0xFFD8) 之后插入一个 APP1 段，内含 XMP 元数据，标注视频段长度
//  3. 把视频文件字节直接拼接到 JPEG 末尾
//
// 输出文件本身仍是合法 JPEG（任何看图软件都能打开），支持 Live Photo 的相册会识别尾部视频
func CreateLivePhoto(imagePath, videoPath, outputPath string) error {
	jpegBytes, err := convertToJPEG(imagePath)
	if err != nil {
		return fmt.Errorf("图片转 JPEG 失败: %w", err)
	}

	videoInfo, err := os.Stat(videoPath)
	if err != nil {
		return fmt.Errorf("读取视频失败: %w", err)
	}
	videoSize := videoInfo.Size()
	if videoSize <= 0 {
		return fmt.Errorf("视频文件为空: %s", videoPath)
	}

	xmpSegment, err := buildXMPSegment(videoSize)
	if err != nil {
		return fmt.Errorf("构造 XMP 段失败: %w", err)
	}

	// 在 JPEG 头部 (SOI 后) 插入 XMP APP1 段
	if len(jpegBytes) < 2 || jpegBytes[0] != 0xFF || jpegBytes[1] != 0xD8 {
		return fmt.Errorf("无效的 JPEG 头部")
	}
	merged := make([]byte, 0, len(jpegBytes)+len(xmpSegment))
	merged = append(merged, jpegBytes[:2]...)
	merged = append(merged, xmpSegment...)
	merged = append(merged, jpegBytes[2:]...)

	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer out.Close()

	if _, err := out.Write(merged); err != nil {
		return fmt.Errorf("写入 JPEG 失败: %w", err)
	}

	video, err := os.Open(videoPath)
	if err != nil {
		return fmt.Errorf("打开视频失败: %w", err)
	}
	defer video.Close()

	if _, err := io.Copy(out, video); err != nil {
		return fmt.Errorf("追加视频失败: %w", err)
	}
	return nil
}

// convertToJPEG 将任意支持的图片格式（JPEG/PNG/GIF/WebP）解码并重新编码为 JPEG 字节
func convertToJPEG(imagePath string) ([]byte, error) {
	f, err := os.Open(imagePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("解码图片失败: %w", err)
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		return nil, fmt.Errorf("编码 JPEG 失败: %w", err)
	}
	return buf.Bytes(), nil
}

// buildXMPSegment 构造一个包含 GCamera/MicroVideo 元数据的 JPEG APP1 段
//
// JPEG APP1 段格式：
//   FF E1 [length:2 bytes BE] [namespace] 00 [payload...]
// 其中 length 包含自身两字节，但不包含 FFE1 标记。
//
// XMP 命名空间标识："http://ns.adobe.com/xap/1.0/\0"
func buildXMPSegment(videoSize int64) ([]byte, error) {
	xmpPayload := buildXMPPayload(videoSize)

	const xmpNamespace = "http://ns.adobe.com/xap/1.0/\x00"
	body := xmpNamespace + xmpPayload

	// 段长度 = 2(长度本身) + len(body)
	totalLen := 2 + len(body)
	if totalLen > 0xFFFF {
		return nil, fmt.Errorf("XMP 段过大: %d 字节，超过 JPEG APP1 上限", totalLen)
	}

	seg := make([]byte, 0, 4+len(body))
	seg = append(seg, 0xFF, 0xE1)
	lenBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(lenBytes, uint16(totalLen))
	seg = append(seg, lenBytes...)
	seg = append(seg, []byte(body)...)
	return seg, nil
}

// buildXMPPayload 生成 XMP RDF 文本，覆盖 GCamera/Container/小米三种命名空间
// 兼容性：Google Pixel/相册、小米相册、OPPO 部分机型
func buildXMPPayload(videoSize int64) string {
	const tpl = `<x:xmpmeta xmlns:x="adobe:ns:meta/" x:xmptk="XHS-LivePhoto"><rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"><rdf:Description rdf:about="" xmlns:GCamera="http://ns.google.com/photos/1.0/camera/" xmlns:Container="http://ns.google.com/photos/1.0/container/" xmlns:Item="http://ns.google.com/photos/1.0/container/item/" xmlns:xmpNote="http://ns.adobe.com/xmp/note/" GCamera:MicroVideo="1" GCamera:MicroVideoVersion="1" GCamera:MicroVideoOffset="%d" GCamera:MicroVideoPresentationTimestampUs="0"><Container:Directory><rdf:Seq><rdf:li rdf:parseType="Resource"><Container:Item Item:Mime="image/jpeg" Item:Semantic="Primary"/></rdf:li><rdf:li rdf:parseType="Resource"><Container:Item Item:Mime="video/mp4" Item:Semantic="MotionPhoto" Item:Length="%d"/></rdf:li></rdf:Seq></Container:Directory></rdf:Description></rdf:RDF></x:xmpmeta>`
	return strings.Replace(fmt.Sprintf(tpl, videoSize, videoSize), "\n", "", -1)
}
