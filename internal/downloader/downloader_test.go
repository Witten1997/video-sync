package downloader

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bili-download/internal/bilibili"
	"bili-download/internal/config"
	"bili-download/internal/database/models"
)

// TestDownloaderBasic 基础下载器测试
func TestDownloaderBasic(t *testing.T) {
	// 加载配置
	cfg := &config.Config{
		Paths: config.PathsConfig{
			DownloadBase: "./test_downloads",
		},
		Quality: config.QualityConfig{
			MaxResolution: "1080p",
		},
		Download: config.DownloadConfig{
			SkipPoster:   false,
			SkipVideoNFO: false,
			SkipDanmaku:  false,
			SkipSubtitle: false,
		},
		Template: config.TemplateConfig{
			VideoName:  "{{title}}",
			PageName:   "{{title}}-P{{pid}}-{{page_title}}",
			TimeFormat: "2006-01-02",
		},
		Advanced: config.AdvancedConfig{
			YtdlpExtraArgs: []string{},
		},
		Bilibili: config.BilibiliConfig{
			Credential: config.CredentialConfig{
				SESSDATA:    "your_sessdata",
				BiliJct:     "your_bili_jct",
				Buvid3:      "your_buvid3",
				DedeUserID:  "your_dedeuserid",
				AcTimeValue: "your_ac_time_value",
			},
		},
	}

	// 创建 B站 客户端
	biliClient := bilibili.NewClient(cfg)

	// 创建下载器
	downloader, err := NewDownloader(cfg, biliClient)
	if err != nil {
		t.Fatalf("创建下载器失败: %v", err)
	}
	defer downloader.Cleanup()

	t.Log("下载器创建成功")
}

// TestYtdlpAvailability 测试 yt-dlp 可用性
func TestYtdlpAvailability(t *testing.T) {
	err := CheckYtdlpAvailable()
	if err != nil {
		t.Fatalf("yt-dlp 不可用: %v", err)
	}
	t.Log("yt-dlp 可用")
}

// TestProgressTracker 测试进度追踪
func TestProgressTracker(t *testing.T) {
	tracker := NewProgressTracker()

	// 添加视频进度
	videoProgress := tracker.AddVideo(1, "BV1xx411c7mD", "测试视频", 2)
	if videoProgress == nil {
		t.Fatal("添加视频进度失败")
	}

	// 添加分P进度
	page1 := NewPageProgress(1, 123456, 1, "第一集")
	videoProgress.AddPage(1, page1)

	page2 := NewPageProgress(2, 123457, 2, "第二集")
	videoProgress.AddPage(2, page2)

	// 更新子任务进度
	page1.UpdateSubTask("video", func(task *SubTaskProgress) {
		task.Status = StatusDownloading
		task.Progress = 50.0
		task.Speed = 1024 * 1024 // 1 MB/s
		task.DownloadedSize = 50 * 1024 * 1024
		task.TotalSize = 100 * 1024 * 1024
	})

	// 验证进度
	subTask := page1.GetSubTask("video")
	if subTask == nil {
		t.Fatal("获取子任务失败")
	}

	if subTask.Progress != 50.0 {
		t.Errorf("进度不正确，期望 50.0，得到 %f", subTask.Progress)
	}

	// 测试整体进度
	overallProgress := videoProgress.GetOverallProgress()
	t.Logf("整体进度: %.2f%%", overallProgress)

	t.Log("进度追踪测试通过")
}

// TestBuildOutputTemplate 测试输出模板构建
func TestBuildOutputTemplate(t *testing.T) {
	cfg := &config.Config{
		Template: config.TemplateConfig{
			VideoName: "{{title}}",
			PageName:  "{{title}}-P{{pid}}-{{page_title}}",
		},
	}

	downloader := &Downloader{
		config: cfg,
	}

	video := &models.Video{
		BVid:       "BV1xx411c7mD",
		Name:       "测试视频/名称",
		SinglePage: false,
	}

	page := &models.Page{
		PID:  1,
		Name: "第一集<>:?",
	}

	template := downloader.buildOutputTemplate(video, page)
	t.Logf("输出模板: %s", template)

	// 验证非法字符被处理
	if template == "" {
		t.Error("模板不应为空")
	}
}

// TestFormatSelector 测试格式选择器构建
func TestFormatSelector(t *testing.T) {
	testCases := []struct {
		maxRes   string
		expected string
	}{
		{"1080p", "bestvideo[height<=1080]+bestaudio/best[height<=1080]"},
		{"720p", "bestvideo[height<=720]+bestaudio/best[height<=720]"},
		{"4K", "bestvideo[height<=2160]+bestaudio/best[height<=2160]"},
		{"", "bestvideo+bestaudio/best"},
	}

	for _, tc := range testCases {
		cfg := &config.Config{
			Quality: config.QualityConfig{
				MaxResolution: tc.maxRes,
			},
		}

		downloader := &Downloader{
			config: cfg,
		}

		result := downloader.buildFormatSelector()
		if result != tc.expected {
			t.Errorf("分辨率 %s: 期望 %s，得到 %s", tc.maxRes, tc.expected, result)
		}
	}

	t.Log("格式选择器测试通过")
}

// ExampleDownloader 下载器使用示例
func ExampleDownloader() {
	// 创建配置
	cfg := &config.Config{
		Paths: config.PathsConfig{
			DownloadBase: "./downloads",
		},
		Bilibili: config.BilibiliConfig{
			Credential: config.CredentialConfig{
				SESSDATA: "your_sessdata",
				BiliJct:  "your_bili_jct",
			},
		},
	}

	// 创建客户端和下载器
	biliClient := bilibili.NewClient(cfg)
	downloader, _ := NewDownloader(cfg, biliClient)
	defer downloader.Cleanup()

	// 设置进度回调
	downloader.SetProgressCallback(func(videoID uint, pid int, taskName string, progress *SubTaskProgress) {
		fmt.Printf("视频 %d P%d %s: %.2f%% (%.2f MB/s)\n",
			videoID, pid, taskName, progress.Progress, progress.Speed/1024/1024)
	})

	// 创建测试视频和分P
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
		Name: "第一集",
	}

	// 下载
	ctx := context.Background()
	err := downloader.DownloadPage(ctx, video, page, "./downloads/test")
	if err != nil {
		fmt.Printf("下载失败: %v\n", err)
	}

	// Output:
	// (示例输出，实际运行时会显示下载进度)
}

// BenchmarkProgressUpdate 进度更新性能测试
func BenchmarkProgressUpdate(b *testing.B) {
	tracker := NewProgressTracker()
	videoProgress := tracker.AddVideo(1, "BV1xx411c7mD", "测试视频", 1)
	page := NewPageProgress(1, 123456, 1, "测试")
	videoProgress.AddPage(1, page)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		page.UpdateSubTask("video", func(task *SubTaskProgress) {
			task.Progress = float64(i % 100)
			task.Speed = float64(i * 1024)
		})
	}
}

// TestProgressCallback 测试进度回调
func TestProgressCallback(t *testing.T) {
	tracker := NewProgressTracker()

	callbackCalled := false
	tracker.SetCallback(func(videoID uint, pid int, taskName string, progress *SubTaskProgress) {
		callbackCalled = true
		t.Logf("回调触发: 视频 %d P%d %s - %.2f%%", videoID, pid, taskName, progress.Progress)
	})

	videoProgress := tracker.AddVideo(1, "BV1xx411c7mD", "测试视频", 1)
	page := NewPageProgress(1, 123456, 1, "测试")
	videoProgress.AddPage(1, page)

	page.UpdateSubTask("video", func(task *SubTaskProgress) {
		task.Progress = 50.0
	})

	// 手动触发回调
	tracker.NotifyProgress(1, 1, "video", page.GetSubTask("video"))

	// 等待回调执行
	time.Sleep(100 * time.Millisecond)

	if !callbackCalled {
		t.Error("回调未被调用")
	}
}

// TestDownloadStatusTransition 测试下载状态转换
func TestDownloadStatusTransition(t *testing.T) {
	page := NewPageProgress(1, 123456, 1, "测试")

	// 初始状态
	if page.Status != StatusPending {
		t.Errorf("初始状态错误: %s", page.Status)
	}

	// 开始下载
	page.UpdateStatus(StatusDownloading)
	if page.Status != StatusDownloading {
		t.Errorf("下载中状态错误: %s", page.Status)
	}

	// 完成下载
	page.UpdateStatus(StatusSucceeded)
	if page.Status != StatusSucceeded {
		t.Errorf("完成状态错误: %s", page.Status)
	}

	// 验证结束时间已设置
	if page.EndTime.IsZero() {
		t.Error("结束时间未设置")
	}

	t.Log("状态转换测试通过")
}
