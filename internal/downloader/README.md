# Downloader 模块

B站视频下载器模块，基于 yt-dlp 封装，提供视频、封面、字幕、弹幕的下载功能和详细的进度追踪。

## 功能特性

- ✅ **yt-dlp 封装**: 完整的命令行构建和输出解析
- ✅ **进度追踪**: 实时追踪下载进度，支持多任务并发
- ✅ **重试机制**: 自动重试失败的下载，支持可配置的重试次数
- ✅ **多内容下载**: 支持视频、封面、字幕、弹幕的完整下载
- ✅ **Cookie 认证**: 自动生成 Netscape 格式的 cookies 文件
- ✅ **格式选择**: 根据配置自动选择最佳视频格式和分辨率
- ✅ **并发安全**: 使用读写锁保证进度追踪的线程安全

## 文件结构

```
internal/downloader/
├── progress.go          # 进度追踪相关数据结构
├── ytdlp.go            # yt-dlp 命令封装和执行
├── downloader.go       # B站视频下载器主逻辑
├��─ downloader_test.go  # 单元测试
└── README.md           # 本文档
```

## 核心组件

### 1. ProgressTracker (进度追踪器)

追踪所有视频和分P的下载进度。

**数据结构**:
- `VideoProgress`: 视频级别的进度
- `PageProgress`: 分P级别的进度
- `SubTaskProgress`: 子任务级别的进度（video, poster, subtitle, danmaku, nfo, upper）

**状态枚举**:
- `pending`: 待处理
- `downloading`: 下载中
- `succeeded`: 成功
- `skipped`: 跳过
- `failed`: 失败（可重试）
- `fixed_failed`: 永久失败
- `ignored`: 忽略的错误

**使用示例**:
```go
tracker := NewProgressTracker()

// 添加视频进度
videoProgress := tracker.AddVideo(1, "BV1xx411c7mD", "测试视频", 2)

// 添加分P进度
page1 := NewPageProgress(1, 123456, 1, "第一集")
videoProgress.AddPage(1, page1)

// 更新子任务进度
page1.UpdateSubTask("video", func(task *SubTaskProgress) {
    task.Status = StatusDownloading
    task.Progress = 50.0
    task.Speed = 1024 * 1024 // 1 MB/s
})

// 获取整体进度
overallProgress := videoProgress.GetOverallProgress()
```

### 2. YtdlpDownloader (yt-dlp 下载器)

封装 yt-dlp 命令行工具。

**功能**:
- 命令构建: 根据选项构建完整的 yt-dlp 命令
- 进度解析: 解析 yt-dlp 的 JSON 进度输出
- 错误处理: 捕获和分类错误
- 重试逻辑: 自动重试可恢复的错误

**使用示例**:
```go
ytdlp := NewYtdlpDownloader(config, tracker)

opts := &DownloadOptions{
    URL:            "https://www.bilibili.com/video/BV1xx411c7mD",
    OutputPath:     "./downloads",
    OutputTemplate: "%(title)s.%(ext)s",
    Cookies:        "./cookies.txt",
    Format:         "bestvideo+bestaudio/best",
    WriteSubtitles: true,
    SubtitleLangs:  []string{"zh-CN"},
}

err := ytdlp.DownloadWithRetry(ctx, opts, 3, progressCallback)
```

### 3. Downloader (B站下载器)

主下载器，集成所有功能。

**功能**:
- 自动生成 cookies 文件
- 下载视频、封面、字幕、弹幕
- 集成进度追踪
- 支持进度回调
- 自动清理资源

**使用示例**:
```go
// 创建下载器
downloader, err := NewDownloader(config, biliClient)
if err != nil {
    log.Fatal(err)
}
defer downloader.Cleanup()

// 设置进度回调
downloader.SetProgressCallback(func(videoID uint, pid int, taskName string, progress *SubTaskProgress) {
    fmt.Printf("视频 %d P%d %s: %.2f%%\n", videoID, pid, taskName, progress.Progress)
})

// 下载分P
err = downloader.DownloadPage(ctx, video, page, outputDir)
```

## 下载选项

### DownloadOptions

| 字段 | 类型 | 说明 |
|------|------|------|
| URL | string | 视频URL |
| OutputPath | string | 输出路径 |
| OutputTemplate | string | 输出文件名模板 |
| Cookies | string | Cookies 文件路径 |
| Headers | []string | 自定义请求头 |
| Format | string | 视频格式选择器 |
| SubtitleLangs | []string | 字幕语言 |
| WriteSubtitles | bool | 是否下载字幕 |
| WriteThumbnail | bool | 是否下载缩略图 |
| ExtraArgs | []string | 额外的 yt-dlp 参数 |

### 格式选择器

根据配置的最大分辨率自动构建格式选择器：

- `4K/2160p`: `bestvideo[height<=2160]+bestaudio/best[height<=2160]`
- `1080p`: `bestvideo[height<=1080]+bestaudio/best[height<=1080]`
- `720p`: `bestvideo[height<=720]+bestaudio/best[height<=720]`
- `480p`: `bestvideo[height<=480]+bestaudio/best[height<=480]`
- 默认: `bestvideo+bestaudio/best`

## 进度回调

进度回调函数签名：
```go
type ProgressCallback func(videoID uint, pid int, taskName string, progress *SubTaskProgress)
```

**参数**:
- `videoID`: 视频ID
- `pid`: 分P编号
- `taskName`: 任务名称（video, poster, subtitle, danmaku, nfo, upper）
- `progress`: 子任务进度信息

**SubTaskProgress 字段**:
```go
type SubTaskProgress struct {
    Name           string         // 子任务名称
    Status         DownloadStatus // 下载状态
    Progress       float64        // 进度百分比 (0-100)
    Speed          float64        // 下载速度 (bytes/sec)
    DownloadedSize int64          // 已下载大小
    TotalSize      int64          // 总大小
    ETA            float64        // 预计剩余时间 (秒)
    Error          string         // 错误信息
    RetryCount     int            // 重试次数
    StartTime      time.Time      // 开始时间
    EndTime        time.Time      // 结束时间
}
```

## 错误处理

### 错误类型

**可重试错误**:
- 网络超时
- 连接失败
- 临时服务器错误

**不可重试错误**:
- Video unavailable (视频不可用)
- Private video (私有视频)
- Deleted video (已删除)
- This video is not available (视频不可用)
- requested format not available (格式不可用)
- Unsupported URL (不支持的URL)

### 重试策略

- 默认最大重试次数: 3次
- 重试间隔: 5秒 * 重试次数（递增）
- 不可重���错误会立即返回失败

## 配置要求

下载器需要以下配置项：

```yaml
# 路径配置
paths:
  download_base: "./downloads"  # 下载基础路径

# 质量配置
quality:
  max_resolution: "1080p"       # 最大分辨率

# 下载配置
download:
  skip_poster: false            # 是否跳过封面
  skip_video_nfo: false         # 是否跳过NFO
  skip_danmaku: false           # 是否跳过弹幕
  skip_subtitle: false          # 是否跳过字幕

# 模板配置
template:
  video_name: "{{title}}"       # 视频文件名模板
  page_name: "{{title}}-P{{pid}}-{{page_title}}"  # 分P文件名模板

# 高级配置
advanced:
  ytdlp_extra_args: []          # yt-dlp 额外参数
```

## 依赖

### 外部依赖
- **yt-dlp**: 视频下载引擎（需要系统安装）

### 内部依赖
- `internal/config`: 配置管理
- `internal/bilibili`: B站API客户端
- `internal/database/models`: 数据模型
- `internal/utils`: 工具函数（日志、文件名安全化）

## 安装 yt-dlp

### Windows
```bash
# 使用 winget
winget install yt-dlp

# 或使用 pip
pip install yt-dlp
```

### Linux/macOS
```bash
# 使用 pip
pip install yt-dlp

# 或使用包管理器
# Ubuntu/Debian
sudo apt install yt-dlp

# macOS
brew install yt-dlp
```

## 测试

运行单元测试：
```bash
cd internal/downloader
go test -v
```

运行特定测试：
```bash
go test -v -run TestProgressTracker
```

运行性能测试：
```bash
go test -bench=. -benchmem
```

## 待完成功能

- [ ] 弹幕转 ASS 格式（当前仅保存为JSON）
- [ ] NFO 元数据生成
- [ ] UP主信息下载
- [ ] 更详细的进度信息（分片下载进度）
- [ ] 下载队列管理
- [ ] 并发下载控制
- [ ] 磁盘空间检查
- [ ] 断点续传优化

## 使用示例

### 完整下载流程

```go
package main

import (
    "context"
    "fmt"
    "time"

    "bili-download/internal/bilibili"
    "bili-download/internal/config"
    "bili-download/internal/database/models"
    "bili-download/internal/downloader"
)

func main() {
    // 加载配置
    cfg, err := config.LoadConfig("./configs/config.yaml")
    if err != nil {
        panic(err)
    }

    // 创建B站客户端
    biliClient := bilibili.NewClient(cfg)

    // 创建下载器
    dl, err := downloader.NewDownloader(cfg, biliClient)
    if err != nil {
        panic(err)
    }
    defer dl.Cleanup()

    // 设置进度回调
    dl.SetProgressCallback(func(videoID uint, pid int, taskName string, progress *downloader.SubTaskProgress) {
        fmt.Printf("[%s] 视频 %d P%d: %.2f%% (%.2f MB/s, ETA: %.0fs)\n",
            taskName, videoID, pid, progress.Progress,
            progress.Speed/1024/1024, progress.ETA)
    })

    // 准备视频和分P信息
    video := &models.Video{
        ID:         1,
        BVid:       "BV1xx411c7mD",
        Name:       "测试视频",
        SinglePage: true,
    }

    page := &models.Page{
        ID:   1,
        PID:  1,
        CID:  123456,
        Name: "正片",
    }

    // 开始下载
    ctx := context.Background()
    outputDir := "./downloads/test"

    fmt.Println("开始下载...")
    startTime := time.Now()

    err = dl.DownloadPage(ctx, video, page, outputDir)
    if err != nil {
        fmt.Printf("下载失败: %v\n", err)
        return
    }

    duration := time.Since(startTime)
    fmt.Printf("下载完成，耗时: %s\n", duration)

    // 获取最终进度
    tracker := dl.GetTracker()
    videoProgress := tracker.GetVideo(video.ID)
    fmt.Printf("整体进度: %.2f%%\n", videoProgress.GetOverallProgress())
}
```

## 常见问题

### Q: yt-dlp 命令未找到
A: 请确保 yt-dlp 已安装并在系统 PATH 中。运行 `yt-dlp --version` 验证。

### Q: 下载失败，提示认证错误
A: 请检查配置文件中的 B站 cookies 是否有效。可以重新登录 B站 并更新 cookies。

### Q: 进度回调没有被调用
A: 确保在调用 `DownloadPage` 之前使用 `SetProgressCallback` 设置了回调函数。

### Q: 下载速度慢
A: 可能是 B站 限速或网络问题。可以尝试：
- 更换 CDN（配置中启用 `cdn_sort`）
- 降低分辨率
- 检查网络连接

### Q: 如何自定义 yt-dlp 参数
A: 在配置文件的 `advanced.ytdlp_extra_args` 中添加额外参数：
```yaml
advanced:
  ytdlp_extra_args:
    - "--concurrent-fragments"
    - "4"
    - "--throttled-rate"
    - "100K"
```

## 性能优化建议

1. **并发下载**: 后续版本将实现下载队列和并发控制
2. **缓存机制**: 可以缓存视频信息避免重复请求
3. **断点续传**: yt-dlp 原生支持，确保不要删除临时文件
4. **磁盘I/O**: 使用 SSD 可以显著提升写入速度

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

与主项目保持一致
