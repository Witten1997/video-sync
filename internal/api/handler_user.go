package api

import (
	"net/http"
	"strconv"

	"bili-download/internal/auth"
	"bili-download/internal/database/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (s *Server) handleLogin(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, "用户名和密码不能为空")
		return
	}

	var user models.User
	if err := s.db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		respondError(c, http.StatusUnauthorized, "用户名或密码错误")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		respondError(c, http.StatusUnauthorized, "用户名或密码错误")
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Username)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	respondSuccess(c, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

func (s *Server) handleGetCurrentUser(c *gin.Context) {
	userID := c.GetUint("user_id")
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		respondNotFound(c, "用户不存在")
		return
	}
	respondSuccess(c, user)
}

func (s *Server) handleListUsers(c *gin.Context) {
	var users []models.User
	if err := s.db.Order("id asc").Find(&users).Error; err != nil {
		respondInternalError(c, err)
		return
	}
	respondSuccess(c, users)
}

func (s *Server) handleCreateUser(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, "用户名和密码不能为空")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	user := models.User{
		Username: req.Username,
		Password: string(hash),
	}
	if err := s.db.Create(&user).Error; err != nil {
		respondError(c, http.StatusConflict, "用户名已存在")
		return
	}
	respondSuccess(c, user)
}

type updateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) handleUpdateUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		respondValidationError(c, "无效的用户ID")
		return
	}

	var user models.User
	if err := s.db.First(&user, id).Error; err != nil {
		respondNotFound(c, "用户不存在")
		return
	}

	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, "请求参数错误")
		return
	}

	updates := map[string]interface{}{}
	if req.Username != "" {
		// 检查用户名是否已被占用
		var existing models.User
		if err := s.db.Where("username = ? AND id != ?", req.Username, id).First(&existing).Error; err == nil {
			respondError(c, http.StatusConflict, "用户名已存在")
			return
		}
		updates["username"] = req.Username
	}
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			respondInternalError(c, err)
			return
		}
		updates["password"] = string(hash)
	}

	if len(updates) > 0 {
		if err := s.db.Model(&user).Updates(updates).Error; err != nil {
			respondInternalError(c, err)
			return
		}
	}

	s.db.First(&user, id)
	respondSuccess(c, user)
}

func (s *Server) handleDeleteUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		respondValidationError(c, "无效的用户ID")
		return
	}

	currentUserID := c.GetUint("user_id")
	if uint(id) == currentUserID {
		respondError(c, http.StatusBadRequest, "不能删除自己")
		return
	}

	if err := s.db.Delete(&models.User{}, id).Error; err != nil {
		respondInternalError(c, err)
		return
	}
	respondSuccess(c, nil)
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

func (s *Server) handleChangePassword(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, "旧密码和新密码不能为空")
		return
	}

	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		respondNotFound(c, "用户不存在")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		respondError(c, http.StatusBadRequest, "旧密码错误")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	s.db.Model(&user).Update("password", string(hash))
	respondSuccess(c, nil)
}

// SeedDefaultUser 创建默认管理员用户
func SeedDefaultUser(db *gorm.DB) error {
	var count int64
	db.Model(&models.User{}).Count(&count)
	if count > 0 {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return db.Create(&models.User{
		Username: "admin",
		Password: string(hash),
	}).Error
}
