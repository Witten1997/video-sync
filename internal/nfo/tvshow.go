package nfo

import (
	"encoding/xml"
	"time"
)

// TVShowNFO 电视剧 NFO
type TVShowNFO struct {
	XMLName       xml.Name   `xml:"tvshow"`
	Title         string     `xml:"title"`
	OriginalTitle string     `xml:"originaltitle,omitempty"`
	SortTitle     string     `xml:"sorttitle,omitempty"`
	Plot          string     `xml:"plot,omitempty"`
	Outline       string     `xml:"outline,omitempty"`
	Tagline       string     `xml:"tagline,omitempty"`
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
	Season        int        `xml:"season,omitempty"`  // 季数（通常为1）
	Episode       int        `xml:"episode,omitempty"` // 总集数
	UniqueID      []UniqueID `xml:"uniqueid,omitempty"`
	Ratings       *Ratings   `xml:"ratings,omitempty"`
	UserRating    float64    `xml:"userrating,omitempty"`
	Status        string     `xml:"status,omitempty"` // Continuing/Ended
	DateAdded     string     `xml:"dateadded,omitempty"`
}

// TVShowGenerator 电视剧 NFO 生成器
type TVShowGenerator struct {
	nfo *TVShowNFO
}

// NewTVShowGenerator 创建电视剧 NFO 生成器
func NewTVShowGenerator() *TVShowGenerator {
	return &TVShowGenerator{
		nfo: &TVShowNFO{
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
func (g *TVShowGenerator) SetTitle(title string) *TVShowGenerator {
	g.nfo.Title = title
	return g
}

// SetOriginalTitle 设置原始标题
func (g *TVShowGenerator) SetOriginalTitle(title string) *TVShowGenerator {
	g.nfo.OriginalTitle = title
	return g
}

// SetPlot 设置剧情
func (g *TVShowGenerator) SetPlot(plot string) *TVShowGenerator {
	g.nfo.Plot = plot
	return g
}

// SetPremiered 设置首映日期
func (g *TVShowGenerator) SetPremiered(t time.Time) *TVShowGenerator {
	g.nfo.Premiered = FormatDate(t, "2006-01-02")
	g.nfo.Year = t.Year()
	return g
}

// SetDateAdded 设置添加日期
func (g *TVShowGenerator) SetDateAdded(t time.Time) *TVShowGenerator {
	g.nfo.DateAdded = FormatDate(t, "2006-01-02 15:04:05")
	return g
}

// SetStudio 设置工作室
func (g *TVShowGenerator) SetStudio(studio string) *TVShowGenerator {
	g.nfo.Studio = studio
	return g
}

// SetEpisodeCount 设置集数
func (g *TVShowGenerator) SetEpisodeCount(count int) *TVShowGenerator {
	g.nfo.Episode = count
	return g
}

// SetStatus 设置状态
func (g *TVShowGenerator) SetStatus(status string) *TVShowGenerator {
	g.nfo.Status = status
	return g
}

// AddActor 添加演员
func (g *TVShowGenerator) AddActor(name, role, thumb string) *TVShowGenerator {
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
func (g *TVShowGenerator) AddGenre(genre string) *TVShowGenerator {
	g.nfo.Genre = append(g.nfo.Genre, genre)
	return g
}

// AddTag 添加标签
func (g *TVShowGenerator) AddTag(tag string) *TVShowGenerator {
	g.nfo.Tag = append(g.nfo.Tag, tag)
	return g
}

// AddTags 添加多个标签
func (g *TVShowGenerator) AddTags(tags []string) *TVShowGenerator {
	g.nfo.Tag = append(g.nfo.Tag, tags...)
	return g
}

// AddThumb 添加缩略图
func (g *TVShowGenerator) AddThumb(url, aspect string) *TVShowGenerator {
	thumb := Thumb{
		URL:    url,
		Aspect: aspect,
	}
	g.nfo.Thumb = append(g.nfo.Thumb, thumb)
	return g
}

// AddUniqueID 添加唯一标识
func (g *TVShowGenerator) AddUniqueID(idType, value string, isDefault bool) *TVShowGenerator {
	uid := UniqueID{
		Type:    idType,
		Value:   value,
		Default: isDefault,
	}
	g.nfo.UniqueID = append(g.nfo.UniqueID, uid)
	return g
}

// Generate 生成 NFO
func (g *TVShowGenerator) Generate() ([]byte, error) {
	return xml.MarshalIndent(g.nfo, "", "  ")
}

// WriteToFile 写入文件
func (g *TVShowGenerator) WriteToFile(filename string) error {
	return WriteXMLToFile(filename, g.nfo)
}

// GetNFO 获取 NFO 对象
func (g *TVShowGenerator) GetNFO() *TVShowNFO {
	return g.nfo
}
