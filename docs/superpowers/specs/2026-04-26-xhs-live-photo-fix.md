# XHS Live Photo 修复 Spec

## 目标

修复小红书 live 图下载后经常回退为单独 `jpg + mp4` 的问题，使正常可用的 live 图源尽量稳定产出单个 `*_live.jpg`。

## 根因

当前合成链路有两个薄弱点：

1. 图片部分直接走 Go `image.Decode`，只稳定支持 `jpeg/png/gif/webp`，遇到小红书返回的其他格式时会在转 JPEG 阶段失败。
2. 合成后的 XMP 只写了简化版 `MicroVideo` 元数据，兼容性弱，容易出现“文件合成成功但相册不识别”的情况。

## 修改范围

### 1. 图片标准化

把 live 图封面统一标准化成 JPEG 再参与拼接：

- 优先走 `ffmpeg` 转成 JPEG
- `ffmpeg` 失败时，再回退到当前 Go 解码转 JPEG
- 标准化结果用临时文件承接，合成后自动清理

这样可以覆盖更多图片格式，同时让带方向信息的图片在输出时尽量落成最终朝向。

### 2. 视频标准化

把 live 图视频部分统一标准化成 MP4：

- 原文件已经是 `.mp4` 时直接复用
- 不是 `.mp4` 时先尝试 `ffmpeg` 无损 remux 到 MP4
- remux 失败时再回退到 `ffmpeg` 转码为 H.264/AAC MP4

这样保证嵌入 JPEG 尾部的视频段和 XMP 中声明的 `video/mp4` 一致。

### 3. XMP 增强

把当前简化 XMP 升级为兼容性更高的版本，补齐：

- `GCamera:MotionPhoto`
- `GCamera:MicroVideo`
- `OpCamera:*`
- `MiCamera:*`
- `Container:Directory`

目标是尽量对齐 Android 项目现有写法，提高安卓相册识别率。

### 4. 调用链调整

- `CreateLivePhoto` 增加 `context.Context`
- 下载器在 live 图合成时传入任务上下文
- 合成失败日志保留具体阶段错误，方便区分是图片标准化、视频标准化还是 XMP/写文件失败

### 5. 下载阶段扩展名修正

- 下载响应返回 `Content-Type` 时，优先按响应头修正实际落盘扩展名
- 解决 URL 猜测扩展名与真实媒体格式不一致的问题
- 这样能减少 live 图源文件被错误落成 `.jpg` 或 `.mp4`

## 非目标

- 不改 live 图解析与配对逻辑
- 不新增配置项
- 不改普通图片/普通视频下载链路
- 这次不补测试代码

## 验证

- `go build ./...`
- 抽查一条含 live 图的笔记，确认优先产出单个 `*_live.jpg`
- 只有真正合成失败时才回退为单独源文件
