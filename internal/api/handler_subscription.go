package api

import (
	"fmt"
	"strconv"

	"bili-download/internal/bilibili"
	"bili-download/internal/database/models"

	"github.com/gin-gonic/gin"
)

// SubscribeRequest 订阅请求
type SubscribeRequest struct {
	ID   int64  `json:"id" binding:"required"` // 收藏夹ID/UP主mid/合集ID
	Name string `json:"name"`                  // 自定义名称（可选）
	Path string `json:"path"`                  // 自定义保存路径（可选）
}

// handleGetMyFavorites 获取我创建的收藏夹列表
func (s *Server) handleGetMyFavorites(c *gin.Context) {
	// 获取当前用户信息
	userInfo, err := s.biliClient.GetMe()
	if err != nil {
		respondError(c, 401, "获取用户信息失败: "+err.Error())
		return
	}

	// 获取用户收藏夹列表
	favorites, err := s.biliClient.GetUserCreatedFavorites(userInfo.Mid)
	if err != nil {
		respondInternalError(c, fmt.Errorf("获取收藏夹列表失败: %w", err))
		return
	}

	// 查询已订阅的收藏夹ID
	var subscribedFavorites []models.Favorite
	s.db.Find(&subscribedFavorites)
	subscribedMap := make(map[int64]bool)
	for _, fav := range subscribedFavorites {
		subscribedMap[fav.FID] = true
	}

	// 添加订阅状态
	type FavoriteWithStatus struct {
		bilibili.UserFavoriteFolder
		Subscribed bool `json:"subscribed"`
	}

	var result []FavoriteWithStatus
	for _, fav := range favorites {
		result = append(result, FavoriteWithStatus{
			UserFavoriteFolder: fav,
			Subscribed:         subscribedMap[fav.ID],
		})
	}

	respondSuccess(c, result)
}

// handleGetMyFollowings 获取我关注的UP主列表
func (s *Server) handleGetMyFollowings(c *gin.Context) {
	// 获取当前用户信息
	userInfo, err := s.biliClient.GetMe()
	if err != nil {
		respondError(c, 401, "获取用户信息失败: "+err.Error())
		return
	}

	// 获取分页参数
	pn, _ := strconv.Atoi(c.DefaultQuery("pn", "1"))
	ps, _ := strconv.Atoi(c.DefaultQuery("ps", "50"))
	name := c.Query("name")

	var followings []bilibili.FollowingUser
	var total int

	if name != "" {
		// 使用B站搜索API
		followings, total, err = s.biliClient.SearchFollowings(userInfo.Mid, name, pn, ps)
	} else {
		followings, total, err = s.biliClient.GetUserFollowings(userInfo.Mid, pn, ps)
	}
	if err != nil {
		respondInternalError(c, fmt.Errorf("获取关注列表失败: %w", err))
		return
	}

	// 查询已订阅的UP主
	var subscribedSubmissions []models.Submission
	s.db.Find(&subscribedSubmissions)
	subscribedMap := make(map[int64]bool)
	for _, sub := range subscribedSubmissions {
		subscribedMap[sub.UpperID] = true
	}

	// 添加订阅状态
	type FollowingWithStatus struct {
		bilibili.FollowingUser
		Subscribed bool `json:"subscribed"`
	}

	var result []FollowingWithStatus
	for _, following := range followings {
		result = append(result, FollowingWithStatus{
			FollowingUser: following,
			Subscribed:    subscribedMap[following.Mid],
		})
	}

	respondSuccess(c, gin.H{
		"list":  result,
		"total": total,
		"pn":    pn,
		"ps":    ps,
	})
}

// handleSubscribeFavorite 订阅收藏夹
func (s *Server) handleSubscribeFavorite(c *gin.Context) {
	var req SubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, err.Error())
		return
	}

	// 检查是否已订阅
	var existing models.Favorite
	if err := s.db.Where("f_id = ?", req.ID).First(&existing).Error; err == nil {
		respondValidationError(c, "该收藏夹已订阅")
		return
	}

	// 获取收藏夹信息
	favInfo, err := s.biliClient.GetFavoriteInfo(strconv.FormatInt(req.ID, 10))
	if err != nil {
		respondInternalError(c, fmt.Errorf("获取收藏夹信息失败: %w", err))
		return
	}

	// 创建收藏夹记录
	name := req.Name
	if name == "" {
		name = favInfo.Title
	}

	favorite := models.Favorite{
		FID:     req.ID,
		Name:    name,
		Path:    req.Path,
		Enabled: true,
		Rule:    "{}",
	}

	if err := s.db.Create(&favorite).Error; err != nil {
		respondInternalError(c, fmt.Errorf("创建收藏夹失败: %w", err))
		return
	}

	respondSuccess(c, gin.H{
		"message": "订阅成功",
		"source":  favorite,
	})
}

// handleSubscribeUpper 订阅UP主
func (s *Server) handleSubscribeUpper(c *gin.Context) {
	var req SubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, err.Error())
		return
	}

	// 检查是否已订阅
	var existing models.Submission
	if err := s.db.Where("upper_id = ?", req.ID).First(&existing).Error; err == nil {
		respondValidationError(c, "该UP主已订阅")
		return
	}

	// 获取UP主信息
	upperInfo, err := s.biliClient.GetUpperInfo(req.ID)
	if err != nil {
		respondInternalError(c, fmt.Errorf("获取UP主信息失败: %w", err))
		return
	}

	// 创建UP主订阅记录
	name := req.Name
	if name == "" {
		name = upperInfo.Uname
	}

	submission := models.Submission{
		UpperID:   req.ID,
		UpperFace: upperInfo.Face,
		Name:      name,
		Path:      req.Path,
		Enabled:   true,
		Rule:      "{}",
	}

	if err := s.db.Create(&submission).Error; err != nil {
		respondInternalError(c, fmt.Errorf("创建UP主订阅失败: %w", err))
		return
	}

	respondSuccess(c, gin.H{
		"message": "订阅成功",
		"source":  submission,
	})
}

// handleUnsubscribeFavorite 取消订阅收藏夹
func (s *Server) handleUnsubscribeFavorite(c *gin.Context) {
	fidStr := c.Param("fid")
	fid, err := strconv.ParseInt(fidStr, 10, 64)
	if err != nil {
		respondValidationError(c, "无效的收藏夹ID")
		return
	}

	// 查找并删除
	var favorite models.Favorite
	if err := s.db.Where("f_id = ?", fid).First(&favorite).Error; err != nil {
		respondNotFound(c, "未找到该收藏夹订阅")
		return
	}

	if err := s.db.Delete(&favorite).Error; err != nil {
		respondInternalError(c, err)
		return
	}

	respondSuccess(c, gin.H{
		"message": "取消订阅成功",
	})
}

// handleUnsubscribeUpper 取消订阅UP主
func (s *Server) handleUnsubscribeUpper(c *gin.Context) {
	midStr := c.Param("mid")
	mid, err := strconv.ParseInt(midStr, 10, 64)
	if err != nil {
		respondValidationError(c, "无效的UP主ID")
		return
	}

	// 查找并删除
	var submission models.Submission
	if err := s.db.Where("upper_id = ?", mid).First(&submission).Error; err != nil {
		respondNotFound(c, "未找到该UP主订阅")
		return
	}

	if err := s.db.Delete(&submission).Error; err != nil {
		respondInternalError(c, err)
		return
	}

	respondSuccess(c, gin.H{
		"message": "取消订阅成功",
	})
}
