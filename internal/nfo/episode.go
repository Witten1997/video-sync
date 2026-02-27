package nfo

import (
	"encoding/xml"
	"time"
)

// EpisodeNFO 剧集 NFO
type EpisodeNFO struct {
	XMLName        xml.Name   `xml:"episodedetails"`
	Title          string     `xml:"title"`
	SortTitle      string     `xml:"sorttitle,omitempty"`
	ShowTitle      string     `xml:"showtitle,omitempty"`
	Plot           string     `xml:"plot,omitempty"`
	Outline        string     `xml:"outline,omitempty"`
	Runtime        int        `xml:"runtime,omitempty"` // 分钟
	Thumb          []Thumb    `xml:"thumb,omitempty"`
	MPAARating     string     `xml:"mpaa,omitempty"`
	PlayCount      int        `xml:"playcount,omitempty"`
	Watched        bool       `xml:"watched,omitempty"`
	Season         int        `xml:"season"`
	Episode        int        `xml:"episode"`
	DisplaySeason  int        `xml:"displayseason,omitempty"`
	DisplayEpisode int        `xml:"displayepisode,omitempty"`
	Aired          string     `xml:"aired,omitempty"` // YYYY-MM-DD
	Year           int        `xml:"year,omitempty"`
	Studio         string     `xml:"studio,omitempty"`
	Director       string     `xml:"director,omitempty"`
	Credits        string     `xml:"credits,omitempty"`
	Actor          []Actor    `xml:"actor,omitempty"`
	Genre          []string   `xml:"genre,omitempty"`
	Tag            []string   `xml:"tag,omitempty"`
	UniqueID       []UniqueID `xml:"uniqueid,omitempty"`
	Ratings        *Ratings   `xml:"ratings,omitempty"`
	DateAdded      string     `xml:"dateadded,omitempty"`
	FileInfo       *FileInfo  `xml:"fileinfo,omitempty"`
}

// EpisodeGenerator 剧集 NFO 生成器
type EpisodeGenerator struct {
	nfo *EpisodeNFO
}

// NewEpisodeGenerator 创建剧集 NFO 生成器
func NewEpisodeGenerator() *EpisodeGenerator {
	return &EpisodeGenerator{
		nfo: &EpisodeNFO{
			Thumb:    make([]Thumb, 0),
			Actor:    make([]Actor, 0),
			Genre:    make([]string, 0),
			Tag:      make([]string, 0),
			UniqueID: make([]UniqueID, 0),
			Season:   1, // 默认第一季
		},
	}
}

// SetTitle 设置标题
func (g *EpisodeGenerator) SetTitle(title string) *EpisodeGenerator {
	g.nfo.Title = title
	return g
}

// SetShowTitle 设置剧集标题
func (g *EpisodeGenerator) SetShowTitle(title string) *EpisodeGenerator {
	g.nfo.ShowTitle = title
	return g
}

// SetPlot 设置剧情
func (g *EpisodeGenerator) SetPlot(plot string) *EpisodeGenerator {
	g.nfo.Plot = plot
	return g
}

// SetRuntime 设置时长（秒）
func (g *EpisodeGenerator) SetRuntime(seconds int) *EpisodeGenerator {
	g.nfo.Runtime = seconds / 60 // 转换为分钟
	return g
}

// SetSeasonEpisode 设置季和集
func (g *EpisodeGenerator) SetSeasonEpisode(season, episode int) *EpisodeGenerator {
	g.nfo.Season = season
	g.nfo.Episode = episode
	g.nfo.DisplaySeason = season
	g.nfo.DisplayEpisode = episode
	return g
}

// SetAired 设置播出日期
func (g *EpisodeGenerator) SetAired(t time.Time) *EpisodeGenerator {
	g.nfo.Aired = FormatDate(t, "2006-01-02")
	g.nfo.Year = t.Year()
	return g
}

// SetDateAdded 设置添加日期
func (g *EpisodeGenerator) SetDateAdded(t time.Time) *EpisodeGenerator {
	g.nfo.DateAdded = FormatDate(t, "2006-01-02 15:04:05")
	return g
}

// SetStudio 设置工作室
func (g *EpisodeGenerator) SetStudio(studio string) *EpisodeGenerator {
	g.nfo.Studio = studio
	return g
}

// SetDirector 设置导演
func (g *EpisodeGenerator) SetDirector(director string) *EpisodeGenerator {
	g.nfo.Director = director
	return g
}

// AddActor 添加演员
func (g *EpisodeGenerator) AddActor(name, role, thumb string) *EpisodeGenerator {
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
func (g *EpisodeGenerator) AddGenre(genre string) *EpisodeGenerator {
	g.nfo.Genre = append(g.nfo.Genre, genre)
	return g
}

// AddTag 添加标签
func (g *EpisodeGenerator) AddTag(tag string) *EpisodeGenerator {
	g.nfo.Tag = append(g.nfo.Tag, tag)
	return g
}

// AddTags 添加多个标签
func (g *EpisodeGenerator) AddTags(tags []string) *EpisodeGenerator {
	g.nfo.Tag = append(g.nfo.Tag, tags...)
	return g
}

// AddThumb 添加缩略图
func (g *EpisodeGenerator) AddThumb(url, aspect string) *EpisodeGenerator {
	thumb := Thumb{
		URL:    url,
		Aspect: aspect,
	}
	g.nfo.Thumb = append(g.nfo.Thumb, thumb)
	return g
}

// AddUniqueID 添加唯一标识
func (g *EpisodeGenerator) AddUniqueID(idType, value string, isDefault bool) *EpisodeGenerator {
	uid := UniqueID{
		Type:    idType,
		Value:   value,
		Default: isDefault,
	}
	g.nfo.UniqueID = append(g.nfo.UniqueID, uid)
	return g
}

// SetPlayCount 设置播放量
func (g *EpisodeGenerator) SetPlayCount(count int) *EpisodeGenerator {
	g.nfo.PlayCount = count
	return g
}

// SetVideoInfo 设置视频信息
func (g *EpisodeGenerator) SetVideoInfo(codec string, width, height, duration int) *EpisodeGenerator {
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
func (g *EpisodeGenerator) SetAudioInfo(codec, language string, channels int) *EpisodeGenerator {
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
func (g *EpisodeGenerator) Generate() ([]byte, error) {
	return xml.MarshalIndent(g.nfo, "", "  ")
}

// WriteToFile 写入文件
func (g *EpisodeGenerator) WriteToFile(filename string) error {
	return WriteXMLToFile(filename, g.nfo)
}

// GetNFO 获取 NFO 对象
func (g *EpisodeGenerator) GetNFO() *EpisodeNFO {
	return g.nfo
}
