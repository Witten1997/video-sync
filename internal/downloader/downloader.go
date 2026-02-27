package downloader

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bili-download/internal/bilibili"
	"bili-download/internal/config"
	"bili-download/internal/database/models"
	"bili-download/internal/nfo"
	"bili-download/internal/utils"
)

// Downloader B站视频下载器
type Downloader struct {
	config      *config.Config
	biliClient  *bilibili.Client
	ytdlp       *YtdlpDownloader
	tracker     *ProgressTracker
	cookiesFile string
	maxRetries  int
}

// NewDownloader 创建新的下载器
func NewDownloader(cfg *config.Config, biliClient *bilibili.Client) (*Downloader, error) {
	tracker := NewProgressTracker()
	ytdlp := NewYtdlpDownloader(cfg, tracker)

	// 检查 yt-dlp 是否可用
	if err := CheckYtdlpAvailable(); err != nil {
		return nil, err
	}

	d := &Downloader{
		config:     cfg,
		biliClient: biliClient,
		ytdlp:      ytdlp,
		tracker:    tracker,
		maxRetries: 3,
	}

	// 创建 cookies 文件
	if err := d.createCookiesFile(); err != nil {
		utils.Warn("创建 cookies 文件失败: %v", err)
	}

	return d, nil
}

// createCookiesFile 创建 Netscape 格式的 cookies 文件
func (d *Downloader) createCookiesFile() error {
	cred := d.biliClient.GetCredential()
	if cred == nil {
		return fmt.Errorf("未设置凭据")
	}

	// 创建临时 cookies 文件
	tmpFile, err := os.CreateTemp("", "bili-cookies-*.txt")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer tmpFile.Close()

	d.cookiesFile = tmpFile.Name()

	// 写入 Netscape 格式的 cookies
	// 格式: domain  flag  path  secure  expiration  name  value
	cookies := fmt.Sprintf(`# Netscape HTTP Cookie File
.bilibili.com	TRUE	/	FALSE	%d	SESSDATA	%s
.bilibili.com	TRUE	/	FALSE	%d	bili_jct	%s
.bilibili.com	TRUE	/	FALSE	%d	buvid3	%s
.bilibili.com	TRUE	/	FALSE	%d	DedeUserID	%s
.bilibili.com	TRUE	/	FALSE	%d	ac_time_value	%s
`,
		time.Now().Add(365*24*time.Hour).Unix(), cred.SESSDATA,
		time.Now().Add(365*24*time.Hour).Unix(), cred.BiliJct,
		time.Now().Add(365*24*time.Hour).Unix(), cred.Buvid3,
		time.Now().Add(365*24*time.Hour).Unix(), cred.DedeUserID,
		time.Now().Add(365*24*time.Hour).Unix(), cred.AcTimeValue,
	)

	if _, err := tmpFile.WriteString(cookies); err != nil {
		return fmt.Errorf("写入 cookies 失败: %w", err)
	}

	utils.Info("已创建 cookies 文件: %s", d.cookiesFile)
	return nil
}

// Cleanup 清理资源
func (d *Downloader) Cleanup() {
	if d.cookiesFile != "" {
		os.Remove(d.cookiesFile)
		utils.Info("已删除 cookies 文件: %s", d.cookiesFile)
	}
}

// DownloadPage 下载单个分P
func (d *Downloader) DownloadPage(ctx context.Context, video *models.Video, page *models.Page, outputDir string) error {
	// 创建分P进度
	pageProgress := NewPageProgress(page.ID, page.CID, page.PID, page.Name)

	videoProgress := d.tracker.GetVideo(video.ID)
	if videoProgress == nil {
		videoProgress = d.tracker.AddVideo(video.ID, video.BVid, video.Name, len(video.Pages))
	}
	videoProgress.AddPage(page.PID, pageProgress)

	// 下载视频
	if err := d.downloadPageVideo(ctx, video, page, outputDir, pageProgress); err != nil {
		return err
	}

	// 下载封面
	if !d.config.Download.SkipPoster {
		if err := d.downloadPoster(ctx, video, page, outputDir, pageProgress); err != nil {
			utils.Error("下载封面失败: %v", err)
		}
	}

	// 下载字幕
	if !d.config.Download.SkipSubtitle {
		if err := d.downloadSubtitles(ctx, video, page, outputDir, pageProgress); err != nil {
			utils.Error("下载字幕失败: %v", err)
		}
	}

	// 下载弹幕
	if !d.config.Download.SkipDanmaku {
		if err := d.downloadDanmaku(ctx, video, page, outputDir, pageProgress); err != nil {
			utils.Error("下载弹幕失败: %v", err)
		}
	}

	// 生成NFO元数据
	if !d.config.Download.SkipVideoNFO {
		if err := d.generateNFO(ctx, video, page, outputDir, pageProgress); err != nil {
			utils.Error("生成NFO失败: %v", err)
		}
	}

	// 更新状态
	pageProgress.UpdateStatus(StatusSucceeded)
	return nil
}

// downloadPageVideo 下载分P视频
func (d *Downloader) downloadPageVideo(ctx context.Context, video *models.Video, page *models.Page, outputDir string, pageProgress *PageProgress) error {
	utils.Info("开始下载视频: %s [BV%s] P%d - %s", video.Name, video.BVid, page.PID, page.Name)
	utils.Debug("输出目录: %s", outputDir)

	// 更新子任务状态
	pageProgress.UpdateSubTask("video", func(task *SubTaskProgress) {
		task.Status = StatusDownloading
		task.StartTime = time.Now()
	})

	// 构建视频 URL
	videoURL := fmt.Sprintf("https://www.bilibili.com/video/%s?p=%d", video.BVid, page.PID)

	// 构建输出文件名
	outputTemplate := d.buildOutputTemplate(video, page)
	utils.Debug("输出文件名模板: %s", outputTemplate)

	// 构建格式选择器
	format := d.buildFormatSelector()

	// 下载选项
	opts := &DownloadOptions{
		URL:            videoURL,
		OutputPath:     outputDir,
		OutputTemplate: outputTemplate,
		Cookies:        d.cookiesFile,
		Format:         format,
		WriteSubtitles: !d.config.Download.SkipSubtitle,
		SubtitleLangs:  []string{"zh-CN", "zh-Hans"},
		WriteThumbnail: false, // 缩略图单独下载
		ExtraArgs:      d.config.Advanced.YtdlpExtraArgs,
	}

	// 进度回调 - 累加多个流（视频流+音频流）的大小
	var completedStreamSize int64
	var lastStreamTotal int64
	progressCallback := func(progress *ProgressInfo) {
		// 当新流开始时（totalBytes 变小），累加上一个流的大小
		currentTotal := progress.TotalBytes
		if currentTotal == 0 {
			currentTotal = progress.TotalBytesEst
		}
		if currentTotal > 0 && lastStreamTotal > 0 && currentTotal < lastStreamTotal/2 {
			completedStreamSize += lastStreamTotal
		}
		if currentTotal > 0 {
			lastStreamTotal = currentTotal
		}

		pageProgress.UpdateSubTask("video", func(task *SubTaskProgress) {
			task.Progress = progress.Percentage
			task.Speed = progress.Speed
			task.DownloadedSize = completedStreamSize + progress.DownloadedBytes
			task.TotalSize = completedStreamSize + currentTotal
			task.ETA = progress.ETA
		})

		// 通知进度更新
		d.tracker.NotifyProgress(video.ID, page.PID, "video", pageProgress.GetSubTask("video"))
	}

	// 执行下载
	err := d.ytdlp.DownloadWithRetry(ctx, opts, d.maxRetries, progressCallback)

	if err != nil {
		pageProgress.UpdateSubTask("video", func(task *SubTaskProgress) {
			task.Status = StatusFailed
			task.Error = err.Error()
			task.EndTime = time.Now()
		})
		return fmt.Errorf("下载视频失败: %w", err)
	}

	// 验证视频文件是否实际生成，并清理中间文件
	videoFileExists := false
	var videoFileSize int64
	var videoFileName string
	videoExts := []string{".mp4", ".mkv", ".webm", ".flv", ".avi", ".m4v"}
	baseName := utils.Filenamify(video.Name)
	entries, _ := os.ReadDir(outputDir)
	// 正则匹配 yt-dlp 中间文件: filename.fXXXXX.ext（音视频流分离时产生）
	var tempFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		for _, ext := range videoExts {
			if !strings.HasSuffix(strings.ToLower(name), ext) || !strings.HasPrefix(name, baseName) {
				continue
			}
			// 检查是否是中间文件（包含 .fXXXXX. 格式ID后缀）
			nameWithoutExt := name[:len(name)-len(ext)]
			if idx := strings.LastIndex(nameWithoutExt, ".f"); idx > 0 {
				suffix := nameWithoutExt[idx+2:]
				isTemp := len(suffix) > 0
				for _, c := range suffix {
					if c < '0' || c > '9' {
						isTemp = false
						break
					}
				}
				if isTemp {
					tempFiles = append(tempFiles, filepath.Join(outputDir, name))
					break
				}
			}
			info, err := entry.Info()
			if err == nil && info.Size() > 0 {
				videoFileExists = true
				videoFileSize = info.Size()
				videoFileName = name
			}
			break
		}
		if videoFileExists {
			break
		}
	}

	// 清理 yt-dlp 合并后残留的中间文件
	if videoFileExists && len(tempFiles) > 0 {
		for _, f := range tempFiles {
			if err := os.Remove(f); err != nil {
				utils.Warn("清理中间文件失败: %s, %v", f, err)
			} else {
				utils.Info("已清理中间文件: %s", filepath.Base(f))
			}
		}
	}

	// 如果没找到合并后的文件，但有中间文件，说明合并失败
	if !videoFileExists && len(tempFiles) > 0 {
		_ = videoFileName
		pageProgress.UpdateSubTask("video", func(task *SubTaskProgress) {
			task.Status = StatusFailed
			task.Error = "视频流下载成功但合并失败，请检查 ffmpeg 是否正确安装"
			task.EndTime = time.Now()
		})
		return fmt.Errorf("视频合并失败，存在未合并的中间文件: %s", outputDir)
	}

	if !videoFileExists {
		pageProgress.UpdateSubTask("video", func(task *SubTaskProgress) {
			task.Status = StatusFailed
			task.Error = "下载完成但未找到视频文件，可能被B站限流或需要大会员"
			task.EndTime = time.Now()
		})
		return fmt.Errorf("下载完成但未找到视频文件: %s", outputDir)
	}

	// 更新子任务状态（使用磁盘实际文件大小，防止进度回调未触发导致size=0）
	pageProgress.UpdateSubTask("video", func(task *SubTaskProgress) {
		task.Status = StatusSucceeded
		task.Progress = 100
		task.EndTime = time.Now()
		if task.DownloadedSize == 0 {
			task.DownloadedSize = videoFileSize
			task.TotalSize = videoFileSize
		}
	})
	d.tracker.NotifyProgress(video.ID, page.PID, "video", pageProgress.GetSubTask("video"))

	utils.Info("视频下载完成: %s [BV%s] P%d", video.Name, video.BVid, page.PID)
	return nil
}

// buildOutputTemplate 构建输出文件名模板
func (d *Downloader) buildOutputTemplate(video *models.Video, page *models.Page) string {
	// 注意：outputDir已经是视频专属文件夹（例如：D:/Downloads/waasd/视频名/）
	// 所以这里只需要返回文件名，不需要再包含视频名称

	// 单页视频: {video_name}.%(ext)s
	// 多页视频: {video_name}-{ptitle}.%(ext)s
	if video.SinglePage {
		return fmt.Sprintf("%s.%%(ext)s", utils.Filenamify(video.Name))
	}

	return fmt.Sprintf("%s-%s.%%(ext)s",
		utils.Filenamify(video.Name),
		utils.Filenamify(page.Name))
}

// buildFormatSelector 构建格式选择器
func (d *Downloader) buildFormatSelector() string {
	// 根据配置构建yt-dlp格式选择器
	// 将配置的分辨率转换为实际像素高度
	var maxHeight string
	switch d.config.Quality.MaxResolution {
	case "8K":
		maxHeight = "4320"
	case "DOLBY", "HDR", "4K":
		maxHeight = "2160"
	case "1080P60", "1080P+", "1080P":
		maxHeight = "1080"
	case "720P":
		maxHeight = "720"
	case "480P":
		maxHeight = "480"
	case "360P":
		maxHeight = "360"
	default:
		maxHeight = "1080" // 默认1080P
	}

	// 构建视频编码格式过滤条件
	var videoFilters []string
	if len(d.config.Quality.CodecPriority) > 0 {
		// 为每个编码格式生成独立的选择器，按优先级排序
		for _, codec := range d.config.Quality.CodecPriority {
			var codecName string
			switch codec {
			case "AVC":
				codecName = "avc"
			case "HEVC":
				codecName = "hev"
			case "AV1":
				codecName = "av01"
			default:
				continue
			}
			// 每个编码格式生成一个独立的bestvideo选择器
			videoFilters = append(videoFilters, fmt.Sprintf("bestvideo[height<=%s][vcodec^=%s]", maxHeight, codecName))
		}
		// 添加不限编码的备选方案
		videoFilters = append(videoFilters, fmt.Sprintf("bestvideo[height<=%s]", maxHeight))
	} else {
		// 没有指定编码格式，直接使用最高分辨率
		videoFilters = append(videoFilters, fmt.Sprintf("bestvideo[height<=%s]", maxHeight))
	}

	// 构建音频过滤器 - 简化为纯 bestaudio，让 yt-dlp 自动选择最佳音频
	audioFilter := "bestaudio"

	// 组合视频和音频选择器
	// 格式: (视频1+音频)/(视频2+音频)/(视频3+音频)/best
	// 为每个视频选择器配对音频
	var finalSelectors []string
	for _, videoFilter := range videoFilters {
		finalSelectors = append(finalSelectors, fmt.Sprintf("%s+%s", videoFilter, audioFilter))
	}
	// 添加 best 作为最终备选（包含音视频）
	finalSelectors = append(finalSelectors, "best")

	return joinWithSlash(finalSelectors)
}

// joinWithSlash 用斜杠连接字符串数组
func joinWithSlash(parts []string) string {
	result := ""
	for i, part := range parts {
		if i > 0 {
			result += "/"
		}
		result += part
	}
	return result
}

// downloadPoster 下载封面
func (d *Downloader) downloadPoster(ctx context.Context, video *models.Video, page *models.Page, outputDir string, pageProgress *PageProgress) error {
	pageProgress.UpdateSubTask("poster", func(task *SubTaskProgress) {
		task.Status = StatusDownloading
		task.StartTime = time.Now()
	})

	defer func() {
		pageProgress.UpdateSubTask("poster", func(task *SubTaskProgress) {
			task.EndTime = time.Now()
		})
	}()

	// 使用页面封面，如果没有则使用视频封面
	coverURL := page.Image
	if coverURL == "" {
		coverURL = video.Cover
	}

	if coverURL == "" {
		pageProgress.UpdateSubTask("poster", func(task *SubTaskProgress) {
			task.Status = StatusSkipped
		})
		utils.Warn("视频 [BV%s] P%d 没有封面信息", video.BVid, page.PID)
		return nil
	}

	// 确保使用 HTTPS 协议（B站API可能返回HTTP链接）
	coverURL = normalizeImageURL(coverURL)
	utils.Info("下载封面: %s", coverURL)

	// 构建输出文件名
	// 单页视频: {video_name}-poster.jpg
	// 多页视频: {video_name}-{ptitle}-poster.jpg
	ext := getImageExtension(coverURL)
	var posterFile string
	if video.SinglePage {
		posterFile = fmt.Sprintf("%s-poster%s",
			utils.Filenamify(video.Name),
			ext)
	} else {
		posterFile = fmt.Sprintf("%s-%s-poster%s",
			utils.Filenamify(video.Name),
			utils.Filenamify(page.Name),
			ext)
	}
	posterPath := filepath.Join(outputDir, posterFile)

	// 下载封面
	resp, err := d.biliClient.Get(coverURL, nil)
	if err != nil {
		pageProgress.UpdateSubTask("poster", func(task *SubTaskProgress) {
			task.Status = StatusFailed
			task.Error = err.Error()
		})
		return fmt.Errorf("下载封面失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查HTTP状态码
	if resp.StatusCode != 200 {
		errMsg := fmt.Sprintf("下载封面失败: HTTP %d", resp.StatusCode)
		pageProgress.UpdateSubTask("poster", func(task *SubTaskProgress) {
			task.Status = StatusFailed
			task.Error = errMsg
		})
		return fmt.Errorf(errMsg)
	}

	// 创建文件
	file, err := os.Create(posterPath)
	if err != nil {
		pageProgress.UpdateSubTask("poster", func(task *SubTaskProgress) {
			task.Status = StatusFailed
			task.Error = err.Error()
		})
		return fmt.Errorf("创建封面文件失败: %w", err)
	}
	defer file.Close()

	// 写入文件并统计大小
	written, err := io.Copy(file, resp.Body)
	if err != nil {
		pageProgress.UpdateSubTask("poster", func(task *SubTaskProgress) {
			task.Status = StatusFailed
			task.Error = err.Error()
		})
		return fmt.Errorf("写入封面文件失败: %w", err)
	}

	pageProgress.UpdateSubTask("poster", func(task *SubTaskProgress) {
		task.Status = StatusSucceeded
		task.Progress = 100
		task.DownloadedSize = written
		task.TotalSize = written
	})
	d.tracker.NotifyProgress(video.ID, page.PID, "poster", pageProgress.GetSubTask("poster"))

	utils.Info("封面下载完成: %s (%.2f KB)", posterPath, float64(written)/1024)
	return nil
}

// normalizeImageURL 标准化图片URL，确保使用HTTPS
func normalizeImageURL(url string) string {
	// B站的图片URL可能使用HTTP，需要转换为HTTPS
	if strings.HasPrefix(url, "http://") {
		return "https://" + url[7:]
	}
	// 如果URL没有协议前缀（某些API返回 //i0.hdslb.com/...）
	if strings.HasPrefix(url, "//") {
		return "https:" + url
	}
	return url
}

// getImageExtension 从URL获取图片扩展名
func getImageExtension(url string) string {
	// 常见的图片扩展名
	extensions := []string{".jpg", ".jpeg", ".png", ".webp", ".gif"}

	lowerURL := strings.ToLower(url)
	for _, ext := range extensions {
		if strings.Contains(lowerURL, ext) {
			return ext
		}
	}

	// 默认使用 .jpg
	return ".jpg"
}

// downloadSubtitles 下载字幕（yt-dlp 已经处理）
func (d *Downloader) downloadSubtitles(ctx context.Context, video *models.Video, page *models.Page, outputDir string, pageProgress *PageProgress) error {
	// 字幕已经由 yt-dlp 下载，这里只是标记状态
	pageProgress.UpdateSubTask("subtitle", func(task *SubTaskProgress) {
		task.Status = StatusSucceeded
		task.Progress = 100
	})
	d.tracker.NotifyProgress(video.ID, page.PID, "subtitle", pageProgress.GetSubTask("subtitle"))
	return nil
}

// downloadDanmaku 下载弹幕
func (d *Downloader) downloadDanmaku(ctx context.Context, video *models.Video, page *models.Page, outputDir string, pageProgress *PageProgress) error {
	pageProgress.UpdateSubTask("danmaku", func(task *SubTaskProgress) {
		task.Status = StatusDownloading
		task.StartTime = time.Now()
	})

	defer func() {
		pageProgress.UpdateSubTask("danmaku", func(task *SubTaskProgress) {
			task.EndTime = time.Now()
		})
	}()

	// 获取弹幕
	danmakuResp, err := d.biliClient.GetDanmakuXML(page.CID)
	if err != nil {
		pageProgress.UpdateSubTask("danmaku", func(task *SubTaskProgress) {
			task.Status = StatusFailed
			task.Error = err.Error()
		})
		return fmt.Errorf("获取弹幕失败: %w", err)
	}

	if len(danmakuResp.Danmakus) == 0 {
		pageProgress.UpdateSubTask("danmaku", func(task *SubTaskProgress) {
			task.Status = StatusSkipped
		})
		return nil
	}

	// 构建输出文件名 (ASS格式)
	// 单页视频: {video_name}.zh-CN.default.ass
	// 多页视频: {video_name}-{ptitle}.zh-CN.default.ass
	var danmakuFile string
	if video.SinglePage {
		danmakuFile = fmt.Sprintf("%s.zh-CN.default.ass",
			utils.Filenamify(video.Name))
	} else {
		danmakuFile = fmt.Sprintf("%s-%s.zh-CN.default.ass",
			utils.Filenamify(video.Name),
			utils.Filenamify(page.Name))
	}
	danmakuPath := filepath.Join(outputDir, danmakuFile)

	// 转换为 ASS 格式
	converter := bilibili.NewASSConverter(&d.config.Danmaku, page.Width, page.Height, page.Duration)
	assContent := converter.ConvertToASS(danmakuResp.Danmakus)

	// 写入ASS文件
	file, err := os.Create(danmakuPath)
	if err != nil {
		pageProgress.UpdateSubTask("danmaku", func(task *SubTaskProgress) {
			task.Status = StatusFailed
			task.Error = err.Error()
		})
		return fmt.Errorf("创建弹幕文件失败: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(assContent); err != nil {
		pageProgress.UpdateSubTask("danmaku", func(task *SubTaskProgress) {
			task.Status = StatusFailed
			task.Error = err.Error()
		})
		return fmt.Errorf("写入弹幕文件失败: %w", err)
	}

	pageProgress.UpdateSubTask("danmaku", func(task *SubTaskProgress) {
		task.Status = StatusSucceeded
		task.Progress = 100
	})
	d.tracker.NotifyProgress(video.ID, page.PID, "danmaku", pageProgress.GetSubTask("danmaku"))

	utils.Info("弹幕下载完成: %s (共 %d 条)", danmakuPath, len(danmakuResp.Danmakus))
	return nil
}

// GetTracker 获取进度追踪器
func (d *Downloader) GetTracker() *ProgressTracker {
	return d.tracker
}

// SetProgressCallback 设置进度回调
func (d *Downloader) SetProgressCallback(callback ProgressCallback) {
	d.tracker.SetCallback(callback)
}

// UpdateConfig 更新配置
func (d *Downloader) UpdateConfig(cfg *config.Config) {
	d.config = cfg
	utils.Info("下载器配置已更新")
}

// generateNFO 生成NFO元数据文件
func (d *Downloader) generateNFO(ctx context.Context, video *models.Video, page *models.Page, outputDir string, pageProgress *PageProgress) error {
	pageProgress.UpdateSubTask("nfo", func(task *SubTaskProgress) {
		task.Status = StatusDownloading
		task.StartTime = time.Now()
	})

	defer func() {
		pageProgress.UpdateSubTask("nfo", func(task *SubTaskProgress) {
			task.EndTime = time.Now()
		})
	}()

	// 构建NFO文件名
	// 单页视频: {video_name}.nfo
	// 多页视频: {video_name}-{ptitle}.nfo
	var nfoFile string
	if video.SinglePage {
		nfoFile = fmt.Sprintf("%s.nfo", utils.Filenamify(video.Name))
	} else {
		nfoFile = fmt.Sprintf("%s-%s.nfo",
			utils.Filenamify(video.Name),
			utils.Filenamify(page.Name))
	}
	nfoPath := filepath.Join(outputDir, nfoFile)

	// 根据NFO时间类型选择日期
	var dateAdded time.Time
	if d.config.Advanced.NFOTimeType == "pubtime" {
		dateAdded = video.PubTime
	} else {
		dateAdded = video.FavTime
	}

	// 根据视频类型选择生成器
	if video.SinglePage {
		// 单P视频使用Movie NFO
		generator := nfo.NewMovieGenerator()
		generator.
			SetTitle(video.Name).
			SetOriginalTitle(video.Name).
			SetPlot(video.Intro).
			SetRuntime(page.Duration).
			SetPremiered(video.PubTime).
			SetDateAdded(dateAdded).
			SetStudio("bilibili").
			SetDirector(video.UpperName).
			SetPlayCount(video.ViewCount).
			AddActor(video.UpperName, "UP主", video.UpperFace).
			AddUniqueID("bvid", video.BVid, true).
			AddTags(video.Tags)

		// 添加视频流信息
		if page.Width > 0 && page.Height > 0 {
			generator.SetVideoInfo("h264", page.Width, page.Height, page.Duration)
		}

		// 添加音频流信息
		generator.SetAudioInfo("aac", "zh", 2)

		// 添加封面
		if video.Cover != "" {
			generator.AddThumb(video.Cover, "poster")
		}

		// 写入文件
		if err := generator.WriteToFile(nfoPath); err != nil {
			pageProgress.UpdateSubTask("nfo", func(task *SubTaskProgress) {
				task.Status = StatusFailed
				task.Error = err.Error()
			})
			return fmt.Errorf("写入NFO文件失败: %w", err)
		}

	} else {
		// 多P视频使用Episode NFO
		generator := nfo.NewEpisodeGenerator()
		generator.
			SetTitle(page.Name).
			SetShowTitle(video.Name).
			SetPlot(video.Intro).
			SetRuntime(page.Duration).
			SetSeasonEpisode(1, page.PID). // 默认第一季，集数为分P号
			SetAired(video.PubTime).
			SetDateAdded(dateAdded).
			SetStudio("bilibili").
			SetDirector(video.UpperName).
			SetPlayCount(video.ViewCount).
			AddActor(video.UpperName, "UP主", video.UpperFace).
			AddUniqueID("bvid", video.BVid, true).
			AddTags(video.Tags)

		// 添加视频流信息
		if page.Width > 0 && page.Height > 0 {
			generator.SetVideoInfo("h264", page.Width, page.Height, page.Duration)
		}

		// 添加音频流信息
		generator.SetAudioInfo("aac", "zh", 2)

		// 添加封面
		if page.Image != "" {
			generator.AddThumb(page.Image, "poster")
		} else if video.Cover != "" {
			generator.AddThumb(video.Cover, "poster")
		}

		// 写入文件
		if err := generator.WriteToFile(nfoPath); err != nil {
			pageProgress.UpdateSubTask("nfo", func(task *SubTaskProgress) {
				task.Status = StatusFailed
				task.Error = err.Error()
			})
			return fmt.Errorf("写入NFO文件失败: %w", err)
		}
	}

	pageProgress.UpdateSubTask("nfo", func(task *SubTaskProgress) {
		task.Status = StatusSucceeded
		task.Progress = 100
	})
	d.tracker.NotifyProgress(video.ID, page.PID, "nfo", pageProgress.GetSubTask("nfo"))

	utils.Info("NFO生成完成: %s", nfoPath)
	return nil
}
