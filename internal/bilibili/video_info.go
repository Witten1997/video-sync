package bilibili

import (
	"time"
)

// VideoInfo 视频基本信息
type VideoInfo struct {
	BVid     string    `json:"bvid"`
	Title    string    `json:"title"`
	Intro    string    `json:"intro"`    // 简介
	Cover    string    `json:"cover"`    // 封面
	PubTime  time.Time `json:"pubtime"`  // 发布时间
	CTime    time.Time `json:"ctime"`    // 创建时间
	FavTime  time.Time `json:"fav_time"` // 收藏时间（仅收藏夹）
	Duration int       `json:"duration"` // 时长（秒）

	// UP 主信息
	Upper Upper `json:"upper"`

	// 分P信息
	Pages []PageInfo `json:"pages"`

	// 视频状态
	State int `json:"state"` // 视频状态：0-正常，其他-异常
}

// Upper UP主信息
type Upper struct {
	Mid  int64  `json:"mid"`
	Name string `json:"name"`
	Face string `json:"face"`
}

// PageInfo 分P信息
type PageInfo struct {
	CID        int64  `json:"cid"`
	Page       int    `json:"page"`     // 分P编号
	Part       string `json:"part"`     // 分P标题
	Duration   int    `json:"duration"` // 时长（秒）
	Width      int    `json:"dimension.width"`
	Height     int    `json:"dimension.height"`
	FirstFrame string `json:"first_frame"` // 封面图
}

// Dimension 视频尺寸
type Dimension struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}
