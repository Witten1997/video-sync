package xhs

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"bili-download/internal/utils"
)

// Downloader 小红书媒体下载器
type Downloader struct {
	httpClient        *http.Client
	concurrent        int  // 单笔记内并发下载数（0=串行）
	enableLivePhoto   bool // 是否合成 Live Photo（默认 true）
	keepLivePhotoSrc  bool // 合成 Live Photo 后是否保留原始图+视频文件（默认 false）
}

// NewDownloader 创建下载器
func NewDownloader(httpClient *http.Client) *Downloader {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Downloader{
		httpClient:      httpClient,
		concurrent:      4,
		enableLivePhoto: true,
	}
}

// SetConcurrent 设置单笔记内的并发下载数
func (d *Downloader) SetConcurrent(n int) {
	if n < 1 {
		n = 1
	}
	d.concurrent = n
}

// SetEnableLivePhoto 设置是否启用 Live Photo 合成
func (d *Downloader) SetEnableLivePhoto(enable bool) {
	d.enableLivePhoto = enable
}

// SetKeepLivePhotoSource 设置 Live Photo 合成后是否保留原始图+视频
func (d *Downloader) SetKeepLivePhotoSource(keep bool) {
	d.keepLivePhotoSrc = keep
}

// downloadJob 内部下载任务
type downloadJob struct {
	groupIndex   int       // 媒体组序号（同组的图+视频共享）
	url          string
	filename     string
	mtype        MediaType
	livePhotoOf  *livePhotoCtx // 非空表示这是 Live Photo 的图/视频部分
}

// livePhotoCtx Live Photo 合成上下文
type livePhotoCtx struct {
	imageJob   *downloadJob
	videoJob   *downloadJob
	finalName  string // 最终合成文件名
}

// DownloadNote 下载笔记的全部媒体到指定目录
func (d *Downloader) DownloadNote(ctx context.Context, note *Note, outputDir string, onProgress ProgressCallback) (*DownloadResult, error) {
	if note == nil {
		return nil, fmt.Errorf("note 为空")
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("创建输出目录失败: %w", err)
	}

	result := &DownloadResult{Note: note, OutputDir: outputDir}
	baseName := buildBaseName(note)

	jobs, livePhotoCtxs := d.planJobs(note, baseName)

	type taskResult struct {
		file DownloadedFile
		err  error
	}
	results := make([]taskResult, len(jobs))

	concurrent := d.concurrent
	if concurrent < 1 {
		concurrent = 1
	}
	if concurrent > len(jobs) {
		concurrent = len(jobs)
	}
	if concurrent < 1 {
		concurrent = 1
	}

	sem := make(chan struct{}, concurrent)
	done := make(chan struct{})
	go func() {
		var wg sync.WaitGroup
		for i, j := range jobs {
			wg.Add(1)
			sem <- struct{}{}
			go func(i int, j downloadJob) {
				defer wg.Done()
				defer func() { <-sem }()

				dst := filepath.Join(outputDir, j.filename)
				size, err := d.downloadFile(ctx, j.url, dst, j.filename, onProgress)
				if err != nil {
					results[i] = taskResult{err: err}
					return
				}
				results[i] = taskResult{file: DownloadedFile{
					Path:      dst,
					URL:       j.url,
					MediaType: j.mtype,
					Size:      size,
				}}
			}(i, j)
		}
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		return result, ctx.Err()
	}

	// 收集普通文件结果
	jobIndexFile := make(map[int]*DownloadedFile, len(jobs))
	for i, r := range results {
		if r.err != nil {
			result.FailedNum++
			utils.Warn("下载失败 (%s): %v", jobs[i].url, r.err)
			continue
		}
		f := r.file
		jobIndexFile[i] = &f
	}

	// 处理 Live Photo 合成
	livePhotoJobs := make(map[int]bool)
	for _, lpCtx := range livePhotoCtxs {
		var imgIdx, vidIdx = -1, -1
		for i, j := range jobs {
			if &jobs[i] == lpCtx.imageJob || j.filename == lpCtx.imageJob.filename {
				imgIdx = i
			}
			if &jobs[i] == lpCtx.videoJob || j.filename == lpCtx.videoJob.filename {
				vidIdx = i
			}
		}
		if imgIdx < 0 || vidIdx < 0 {
			continue
		}
		livePhotoJobs[imgIdx] = true
		livePhotoJobs[vidIdx] = true

		imgFile := jobIndexFile[imgIdx]
		vidFile := jobIndexFile[vidIdx]
		if imgFile == nil || vidFile == nil {
			// 任一下载失败：保留已成功的文件，不合成
			if imgFile != nil {
				result.Files = append(result.Files, *imgFile)
				result.SuccessNum++
			}
			if vidFile != nil {
				result.Files = append(result.Files, *vidFile)
				result.SuccessNum++
			}
			continue
		}

		outputFile := filepath.Join(outputDir, lpCtx.finalName)
		if err := CreateLivePhoto(imgFile.Path, vidFile.Path, outputFile); err != nil {
			utils.Warn("Live Photo 合成失败 (%s)，保留原始文件: %v", lpCtx.finalName, err)
			result.Files = append(result.Files, *imgFile)
			result.Files = append(result.Files, *vidFile)
			result.SuccessNum += 2
			continue
		}

		fi, _ := os.Stat(outputFile)
		var size int64
		if fi != nil {
			size = fi.Size()
		}
		result.Files = append(result.Files, DownloadedFile{
			Path:      outputFile,
			URL:       imgFile.URL,
			MediaType: MediaTypeLivePhoto,
			Size:      size,
		})
		result.SuccessNum++

		if !d.keepLivePhotoSrc {
			os.Remove(imgFile.Path)
			os.Remove(vidFile.Path)
		} else {
			result.Files = append(result.Files, *imgFile)
			result.Files = append(result.Files, *vidFile)
			result.SuccessNum += 2
		}
	}

	// 收集非 Live Photo 普通文件
	for i, f := range jobIndexFile {
		if livePhotoJobs[i] {
			continue
		}
		result.Files = append(result.Files, *f)
		result.SuccessNum++
	}
	return result, nil
}

// planJobs 规划下载任务，并标记需要合成 Live Photo 的图+视频对
func (d *Downloader) planJobs(note *Note, baseName string) ([]downloadJob, []*livePhotoCtx) {
	var jobs []downloadJob
	var lpCtxs []*livePhotoCtx
	idx := 0
	for _, item := range note.MediaItems {
		switch item.Type {
		case MediaTypeImage:
			idx++
			jobs = append(jobs, downloadJob{
				groupIndex: idx,
				url:        item.ImageURL,
				filename:   fmt.Sprintf("%s_%02d.%s", baseName, idx, guessExt(item.ImageURL, "jpg")),
				mtype:      MediaTypeImage,
			})
		case MediaTypeVideo:
			idx++
			jobs = append(jobs, downloadJob{
				groupIndex: idx,
				url:        item.VideoURL,
				filename:   fmt.Sprintf("%s_%02d.%s", baseName, idx, guessExt(item.VideoURL, "mp4")),
				mtype:      MediaTypeVideo,
			})
		case MediaTypeLivePhoto:
			idx++
			imgExt := guessExt(item.ImageURL, "jpg")
			vidExt := guessExt(item.VideoURL, "mp4")
			imgJob := downloadJob{
				groupIndex: idx,
				url:        item.ImageURL,
				filename:   fmt.Sprintf("%s_%02d_live_src.%s", baseName, idx, imgExt),
				mtype:      MediaTypeImage,
			}
			vidJob := downloadJob{
				groupIndex: idx,
				url:        item.VideoURL,
				filename:   fmt.Sprintf("%s_%02d_live_src.%s", baseName, idx, vidExt),
				mtype:      MediaTypeVideo,
			}
			jobs = append(jobs, imgJob, vidJob)

			if d.enableLivePhoto {
				ctxItem := &livePhotoCtx{
					imageJob:  &jobs[len(jobs)-2],
					videoJob:  &jobs[len(jobs)-1],
					finalName: fmt.Sprintf("%s_%02d_live.jpg", baseName, idx),
				}
				lpCtxs = append(lpCtxs, ctxItem)
			}
		}
	}
	return jobs, lpCtxs
}

// downloadFile 下载单个文件
func (d *Downloader) downloadFile(ctx context.Context, url, dst, displayName string, onProgress ProgressCallback) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Referer", "https://www.xiaohongshu.com/")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("下载请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("下载状态码异常: %d (%s)", resp.StatusCode, url)
	}

	tmp := dst + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return 0, fmt.Errorf("创建文件失败: %w", err)
	}

	total := resp.ContentLength
	written := int64(0)
	buf := make([]byte, 32*1024)
	for {
		select {
		case <-ctx.Done():
			f.Close()
			os.Remove(tmp)
			return 0, ctx.Err()
		default:
		}
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				f.Close()
				os.Remove(tmp)
				return 0, fmt.Errorf("写入失败: %w", werr)
			}
			written += int64(n)
			if onProgress != nil {
				onProgress(displayName, written, total)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			f.Close()
			os.Remove(tmp)
			return 0, fmt.Errorf("读取响应失败: %w", readErr)
		}
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return 0, fmt.Errorf("关闭文件失败: %w", err)
	}
	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return 0, fmt.Errorf("重命名文件失败: %w", err)
	}
	return written, nil
}

// buildBaseName 构造文件基础名
func buildBaseName(note *Note) string {
	if note.NoteID != "" {
		return note.NoteID
	}
	if note.Title != "" {
		return utils.Filenamify(utils.TruncateString(note.Title, 40))
	}
	return "xhs_note"
}

// guessExt 根据 URL 推断扩展名
func guessExt(u, fallback string) string {
	low := strings.ToLower(u)
	switch {
	case strings.Contains(low, ".mp4"):
		return "mp4"
	case strings.Contains(low, ".mov"):
		return "mov"
	case strings.Contains(low, ".webm"):
		return "webm"
	case strings.Contains(low, ".png"):
		return "png"
	case strings.Contains(low, ".gif"):
		return "gif"
	case strings.Contains(low, ".webp"):
		return "webp"
	case strings.Contains(low, ".jpeg"), strings.Contains(low, ".jpg"):
		return "jpg"
	case strings.Contains(low, "sns-video"), strings.Contains(low, "/spectrum/"):
		return "mp4"
	}
	return fallback
}
