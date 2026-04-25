package xhs

// MediaType 媒体类型
type MediaType string

const (
	MediaTypeImage     MediaType = "image"      // 普通图片
	MediaTypeVideo     MediaType = "video"      // 普通视频
	MediaTypeLivePhoto MediaType = "live_photo" // 动态照片（图+视频）
)

// NoteType 笔记类型
type NoteType string

const (
	NoteTypeNormal NoteType = "normal" // 图文笔记
	NoteTypeVideo  NoteType = "video"  // 视频笔记
)

// Note 小红书笔记元信息
type Note struct {
	NoteID      string      `json:"note_id"`     // 笔记ID
	Type        NoteType    `json:"type"`        // 笔记类型
	Title       string      `json:"title"`       // 标题
	Description string      `json:"description"` // 描述
	Author      Author      `json:"author"`      // 作者
	PublishTime int64       `json:"publish_time"`// 发布时间（毫秒）
	Tags        []string    `json:"tags"`        // 标签
	MediaItems  []MediaItem `json:"media_items"` // 媒体列表
	OriginalURL string      `json:"original_url"`// 原始链接
}

// Author 作者信息
type Author struct {
	UserID   string `json:"user_id"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	RedID    string `json:"red_id"` // 小红书号
}

// MediaItem 媒体项
type MediaItem struct {
	Type     MediaType `json:"type"`      // 媒体类型
	ImageURL string    `json:"image_url"` // 图片URL（图片或动态照片的封面）
	VideoURL string    `json:"video_url"` // 视频URL（视频笔记或动态照片的视频部分）
	Width    int       `json:"width"`     // 宽度
	Height   int       `json:"height"`    // 高度
}

// DownloadResult 下载结果
type DownloadResult struct {
	Note       *Note            `json:"note"`        // 笔记元信息
	Files      []DownloadedFile `json:"files"`       // 已下载的文件列表
	OutputDir  string           `json:"output_dir"`  // 输出目录
	SuccessNum int              `json:"success_num"` // 成功数
	FailedNum  int              `json:"failed_num"`  // 失败数
}

// DownloadedFile 已下载的文件
type DownloadedFile struct {
	Path       string    `json:"path"`        // 本地路径
	URL        string    `json:"url"`         // 来源URL
	MediaType  MediaType `json:"media_type"`  // 媒体类型
	Size       int64     `json:"size"`        // 文件大小（字节）
	GroupIndex int       `json:"group_index"` // 所属媒体组序号（1-based，对应 Note.MediaItems 索引+1）
}

// ProgressCallback 下载进度回调
type ProgressCallback func(filename string, downloaded, total int64)
