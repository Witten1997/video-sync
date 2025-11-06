package nfo

import (
	"encoding/xml"
	"time"
)

// PersonNFO 人物 NFO（UP主）
type PersonNFO struct {
	XMLName    xml.Name   `xml:"person"`
	Name       string     `xml:"name"`
	SortName   string     `xml:"sortname,omitempty"`
	Role       string     `xml:"role,omitempty"`
	Type       string     `xml:"type,omitempty"` // Actor/Director/Writer等
	Thumb      []Thumb    `xml:"thumb,omitempty"`
	Profile    string     `xml:"profile,omitempty"`
	Biography  string     `xml:"biography,omitempty"`
	Born       string     `xml:"born,omitempty"` // YYYY-MM-DD
	Birthplace string     `xml:"birthplace,omitempty"`
	Deathdate  string     `xml:"deathdate,omitempty"` // YYYY-MM-DD
	Deathplace string     `xml:"deathplace,omitempty"`
	UniqueID   []UniqueID `xml:"uniqueid,omitempty"`
	DateAdded  string     `xml:"dateadded,omitempty"`
}

// PersonGenerator 人物 NFO 生成器
type PersonGenerator struct {
	nfo *PersonNFO
}

// NewPersonGenerator 创建人物 NFO 生成器
func NewPersonGenerator() *PersonGenerator {
	return &PersonGenerator{
		nfo: &PersonNFO{
			Thumb:    make([]Thumb, 0),
			UniqueID: make([]UniqueID, 0),
			Type:     "Actor", // 默认为演员（UP主作为演员）
		},
	}
}

// SetName 设置名称
func (g *PersonGenerator) SetName(name string) *PersonGenerator {
	g.nfo.Name = name
	return g
}

// SetRole 设置角色
func (g *PersonGenerator) SetRole(role string) *PersonGenerator {
	g.nfo.Role = role
	return g
}

// SetType 设置类型
func (g *PersonGenerator) SetType(personType string) *PersonGenerator {
	g.nfo.Type = personType
	return g
}

// SetProfile 设置简介
func (g *PersonGenerator) SetProfile(profile string) *PersonGenerator {
	g.nfo.Profile = profile
	return g
}

// SetBiography 设置传记
func (g *PersonGenerator) SetBiography(bio string) *PersonGenerator {
	g.nfo.Biography = bio
	return g
}

// SetBorn 设置生日
func (g *PersonGenerator) SetBorn(t time.Time) *PersonGenerator {
	g.nfo.Born = FormatDate(t, "2006-01-02")
	return g
}

// SetBirthplace 设置出生地
func (g *PersonGenerator) SetBirthplace(place string) *PersonGenerator {
	g.nfo.Birthplace = place
	return g
}

// SetDateAdded 设置添加日期
func (g *PersonGenerator) SetDateAdded(t time.Time) *PersonGenerator {
	g.nfo.DateAdded = FormatDate(t, "2006-01-02 15:04:05")
	return g
}

// AddThumb 添加头像
func (g *PersonGenerator) AddThumb(url, aspect string) *PersonGenerator {
	thumb := Thumb{
		URL:    url,
		Aspect: aspect,
	}
	g.nfo.Thumb = append(g.nfo.Thumb, thumb)
	return g
}

// AddUniqueID 添加唯一标识
func (g *PersonGenerator) AddUniqueID(idType, value string, isDefault bool) *PersonGenerator {
	uid := UniqueID{
		Type:    idType,
		Value:   value,
		Default: isDefault,
	}
	g.nfo.UniqueID = append(g.nfo.UniqueID, uid)
	return g
}

// Generate 生成 NFO
func (g *PersonGenerator) Generate() ([]byte, error) {
	return xml.MarshalIndent(g.nfo, "", "  ")
}

// WriteToFile 写入文件
func (g *PersonGenerator) WriteToFile(filename string) error {
	return WriteXMLToFile(filename, g.nfo)
}

// GetNFO 获取 NFO 对象
func (g *PersonGenerator) GetNFO() *PersonNFO {
	return g.nfo
}
