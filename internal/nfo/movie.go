package nfo

import (
	"encoding/xml"
	"time"
)

// MovieNFO 电影 NFO（单页视频）
type MovieNFO struct {
	XMLName       xml.Name   `xml:"movie"`
	Title         string     `xml:"title"`
	OriginalTitle string     `xml:"originaltitle,omitempty"`
	SortTitle     string     `xml:"sorttitle,omitempty"`
	Plot          string     `xml:"plot,omitempty"`
	Outline       string     `xml:"outline,omitempty"`
	Tagline       string     `xml:"tagline,omitempty"`
	Runtime       int        `xml:"runtime,omitempty"` // 分钟
	Thumb         []Thumb    `xml:"thumb,omitempty"`
	Fanart        *Fanart    `xml:"fanart,omitempty"`
	MPAARating    string     `xml:"mpaa,omitempty"`
	Premiered     string     `xml:"premiered,omitempty"` // YYYY-MM-DD
	Year          int        `xml:"year,omitempty"`
	Studio        string     `xml:"studio,omitempty"`
	Director      string     `xml:"director,omitempty"`
	Actor         []Actor    `xml:"actor,omitempty"`
	Genre         []string   `xml:"genre,omitempty"`
	Tag           []string   `xml:"tag,omitempty"`
	Country       string     `xml:"country,omitempty"`
	Credits       string     `xml:"credits,omitempty"`
	UniqueID      []UniqueID `xml:"uniqueid,omitempty"`
	Ratings       *Ratings   `xml:"ratings,omitempty"`
	UserRating    float64    `xml:"userrating,omitempty"`
	PlayCount     int        `xml:"playcount,omitempty"`
	Watched       bool       `xml:"watched,omitempty"`
	DateAdded     string     `xml:"dateadded,omitempty"` // YYYY-MM-DD HH:MM:SS
	FileInfo      *FileInfo  `xml:"fileinfo,omitempty"`
}

// FileInfo 文件信息
type FileInfo struct {
	StreamDetails *StreamDetails `xml:"streamdetails,omitempty"`
}

// StreamDetails 流详情
type StreamDetails struct {
	Video    []VideoStream    `xml:"video,omitempty"`
	Audio    []AudioStream    `xml:"audio,omitempty"`
	Subtitle []SubtitleStream `xml:"subtitle,omitempty"`
}

// VideoStream 视频流
type VideoStream struct {
	Codec             string  `xml:"codec,omitempty"`
	Aspect            float64 `xml:"aspect,omitempty"`
	Width             int     `xml:"width,omitempty"`
	Height            int     `xml:"height,omitempty"`
	DurationInSeconds int     `xml:"durationinseconds,omitempty"`
	StereoMode        string  `xml:"stereomode,omitempty"`
}

// AudioStream 音频流
type AudioStream struct {
	Codec    string `xml:"codec,omitempty"`
	Language string `xml:"language,omitempty"`
	Channels int    `xml:"channels,omitempty"`
}

// SubtitleStream 字幕流
type SubtitleStream struct {
	Language string `xml:"language,omitempty"`
}

// MovieGenerator 电影 NFO 生成器
type MovieGenerator struct {
	nfo *MovieNFO
}

// NewMovieGenerator 创建电影 NFO 生成器
func NewMovieGenerator() *MovieGenerator {
	return &MovieGenerator{
		nfo: &MovieNFO{
			Thumb:    make([]Thumb, 0),
			Actor:    make([]Actor, 0),
			Genre:    make([]string, 0),
			Tag:      make([]string, 0),
			UniqueID: make([]UniqueID, 0),
		},
	}
}

// SetTitle 设置标题
func (g *MovieGenerator) SetTitle(title string) *MovieGenerator {
	g.nfo.Title = title
	return g
}

// SetOriginalTitle 设置原始标题
func (g *MovieGenerator) SetOriginalTitle(title string) *MovieGenerator {
	g.nfo.OriginalTitle = title
	return g
}

// SetPlot 设置剧情
func (g *MovieGenerator) SetPlot(plot string) *MovieGenerator {
	g.nfo.Plot = plot
	return g
}

// SetRuntime 设置时长（秒）
func (g *MovieGenerator) SetRuntime(seconds int) *MovieGenerator {
	g.nfo.Runtime = seconds / 60 // 转换为分钟
	return g
}

// SetPremiered 设置首映日期
func (g *MovieGenerator) SetPremiered(t time.Time) *MovieGenerator {
	g.nfo.Premiered = FormatDate(t, "2006-01-02")
	g.nfo.Year = t.Year()
	return g
}

// SetDateAdded 设置添加日期
func (g *MovieGenerator) SetDateAdded(t time.Time) *MovieGenerator {
	g.nfo.DateAdded = FormatDate(t, "2006-01-02 15:04:05")
	return g
}

// SetStudio 设置工作室
func (g *MovieGenerator) SetStudio(studio string) *MovieGenerator {
	g.nfo.Studio = studio
	return g
}

// SetDirector 设置导演
func (g *MovieGenerator) SetDirector(director string) *MovieGenerator {
	g.nfo.Director = director
	return g
}

// AddActor 添加演员
func (g *MovieGenerator) AddActor(name, role, thumb string) *MovieGenerator {
	actor := Actor{
		Name:  name,
		Role:  role,
		Thumb: thumb,
		Order: len(g.nfo.Actor),
	}
	g.nfo.Actor = append(g.nfo.Actor, actor)
	return g
}

// AddGenre 添加类型
func (g *MovieGenerator) AddGenre(genre string) *MovieGenerator {
	g.nfo.Genre = append(g.nfo.Genre, genre)
	return g
}

// AddTag 添加标签
func (g *MovieGenerator) AddTag(tag string) *MovieGenerator {
	g.nfo.Tag = append(g.nfo.Tag, tag)
	return g
}

// AddTags 添加多个标签
func (g *MovieGenerator) AddTags(tags []string) *MovieGenerator {
	g.nfo.Tag = append(g.nfo.Tag, tags...)
	return g
}

// AddThumb 添加缩略图
func (g *MovieGenerator) AddThumb(url, aspect string) *MovieGenerator {
	thumb := Thumb{
		URL:    url,
		Aspect: aspect,
	}
	g.nfo.Thumb = append(g.nfo.Thumb, thumb)
	return g
}

// AddUniqueID 添加唯一标识
func (g *MovieGenerator) AddUniqueID(idType, value string, isDefault bool) *MovieGenerator {
	uid := UniqueID{
		Type:    idType,
		Value:   value,
		Default: isDefault,
	}
	g.nfo.UniqueID = append(g.nfo.UniqueID, uid)
	return g
}

// SetPlayCount 设置播放量
func (g *MovieGenerator) SetPlayCount(count int) *MovieGenerator {
	g.nfo.PlayCount = count
	return g
}

// SetRating 设置评分
func (g *MovieGenerator) SetRating(value float64, votes int) *MovieGenerator {
	if g.nfo.Ratings == nil {
		g.nfo.Ratings = &Ratings{
			Rating: make([]Rating, 0),
		}
	}
	rating := Rating{
		Value:   value,
		Votes:   votes,
		Max:     10,
		Default: true,
	}
	g.nfo.Ratings.Rating = append(g.nfo.Ratings.Rating, rating)
	return g
}

// SetVideoInfo 设置视频信息
func (g *MovieGenerator) SetVideoInfo(codec string, width, height, duration int) *MovieGenerator {
	if g.nfo.FileInfo == nil {
		g.nfo.FileInfo = &FileInfo{
			StreamDetails: &StreamDetails{
				Video: make([]VideoStream, 0),
				Audio: make([]AudioStream, 0),
			},
		}
	}

	aspect := float64(width) / float64(height)
	video := VideoStream{
		Codec:             codec,
		Width:             width,
		Height:            height,
		Aspect:            aspect,
		DurationInSeconds: duration,
	}
	g.nfo.FileInfo.StreamDetails.Video = append(g.nfo.FileInfo.StreamDetails.Video, video)
	return g
}

// SetAudioInfo 设置音频信息
func (g *MovieGenerator) SetAudioInfo(codec, language string, channels int) *MovieGenerator {
	if g.nfo.FileInfo == nil {
		g.nfo.FileInfo = &FileInfo{
			StreamDetails: &StreamDetails{
				Video: make([]VideoStream, 0),
				Audio: make([]AudioStream, 0),
			},
		}
	}

	audio := AudioStream{
		Codec:    codec,
		Language: language,
		Channels: channels,
	}
	g.nfo.FileInfo.StreamDetails.Audio = append(g.nfo.FileInfo.StreamDetails.Audio, audio)
	return g
}

// Generate 生成 NFO
func (g *MovieGenerator) Generate() ([]byte, error) {
	return xml.MarshalIndent(g.nfo, "", "  ")
}

// WriteToFile 写入文件
func (g *MovieGenerator) WriteToFile(filename string) error {
	return WriteXMLToFile(filename, g.nfo)
}

// GetNFO 获取 NFO 对象
func (g *MovieGenerator) GetNFO() *MovieNFO {
	return g.nfo
}
