package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"bili-download/internal/auth"
	"bili-download/internal/bilibili"
	"bili-download/internal/config"
	"bili-download/internal/downloader"
	"bili-download/internal/scheduler"
	"bili-download/internal/utils"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Server API 服务器
type Server struct {
	config       *config.Config
	configPath   string
	db           *gorm.DB
	biliClient   *bilibili.Client
	downloadMgr  *downloader.DownloadManager
	scheduler    *scheduler.Scheduler
	router       *gin.Engine
	httpServer   *http.Server
	websocketHub *WebSocketHub
}

// NewServer 创建新的 API 服务器
func NewServer(cfg *config.Config, configPath string, db *gorm.DB, biliClient *bilibili.Client, downloadMgr *downloader.DownloadManager) (*Server, error) {
	// 设置 Gin 模式
	if cfg.Logging.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	s := &Server{
		config:       cfg,
		configPath:   configPath,
		db:           db,
		biliClient:   biliClient,
		downloadMgr:  downloadMgr,
		websocketHub: NewWebSocketHub(),
	}

	// 创建调度器
	s.scheduler = scheduler.NewScheduler(cfg, db, downloadMgr)

	// 创建路由
	s.setupRouter()

	return s, nil
}

// setupRouter 设置路由
func (s *Server) setupRouter() {
	router := gin.New()

	// 中间件
	router.Use(gin.Recovery())
	router.Use(s.loggerMiddleware())
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 鉴权中间件
	router.Use(s.authMiddleware())

	// API 路由组
	api := router.Group("/api")
	{
		// 健康检查
		api.GET("/health", s.handleHealth)

		// 登录（公开接口）
		api.POST("/auth/login", s.handleLogin)

		// 仪表盘
		api.GET("/dashboard", s.handleDashboard)

		// 系统信息
		api.GET("/system/info", s.handleSystemInfo)
		api.GET("/system/stats", s.handleSystemStats)

		// 认证（二维码登录）
		auth := api.Group("/auth")
		{
			auth.GET("/qrcode/generate", s.handleQRCodeGenerate) // 生成二维码
			auth.GET("/qrcode/poll", s.handleQRCodePoll)         // 轮询二维码状态
		}

		// 用户管理
		users := api.Group("/users")
		{
			users.GET("", s.handleListUsers)
			users.POST("", s.handleCreateUser)
			users.GET("/me", s.handleGetCurrentUser)
			users.PUT("/me/password", s.handleChangePassword)
			users.PUT("/:id", s.handleUpdateUser)
			users.DELETE("/:id", s.handleDeleteUser)
		}

		// yt-dlp 版本管理
		ytdlp := api.Group("/ytdlp")
		{
			ytdlp.GET("/version", s.handleGetYtdlpVersion)
			ytdlp.POST("/update", s.handleUpdateYtdlp)
		}

		// 配置管理
		config := api.Group("/config")
		{
			config.GET("", s.handleGetConfig)
			config.POST("", s.handleUpdateConfig)
			config.POST("/validate", s.handleValidateConfig)
			config.POST("/validate-credential", s.handleValidateBilibiliCredential)
		}

		// 视频源管理
		sources := api.Group("/sources")
		{
			sources.GET("", s.handleListSources)
			sources.POST("", s.handleAddSource)
			sources.GET("/:id", s.handleGetSource)
			sources.PUT("/:id", s.handleUpdateSource)
			sources.DELETE("/:id", s.handleDeleteSource)
			sources.POST("/:id/scan", s.handleScanSource)
			sources.PUT("/:id/enable", s.handleEnableSource)
		}

		// 视频源管理（兼容性路由，映射到 /sources）
		videoSources := api.Group("/video_sources")
		{
			videoSources.GET("", s.handleListSources)
			videoSources.POST("", s.handleAddSource)
			videoSources.GET("/:id", s.handleGetSource)
			videoSources.PUT("/:id", s.handleUpdateSource)
			videoSources.DELETE("/:id", s.handleDeleteSource)
			videoSources.POST("/:id/scan", s.handleScanSource)
			videoSources.PUT("/:id/enable", s.handleEnableSource)
		}

		// 视频管理
		videos := api.Group("/videos")
		{
			videos.GET("", s.handleListVideos)
			videos.POST("/download-by-url", s.handleDownloadByURL) // 通过URL下载
			videos.GET("/:id", s.handleGetVideo)
			videos.PUT("/:id", s.handleUpdateVideo)
			videos.DELETE("/:id", s.handleDeleteVideo)
			videos.POST("/:id/download", s.handleDownloadVideo)
			videos.GET("/:id/pages", s.handleGetVideoPages)
		}

		// 下载记录
		downloadRecords := api.Group("/download-records")
		{
			downloadRecords.GET("", s.handleListDownloadRecords)
			downloadRecords.GET("/:id", s.handleGetDownloadRecord)
			downloadRecords.POST("/:id/retry", s.handleRetryDownloadRecord)
			downloadRecords.DELETE("/:id", s.handleDeleteDownloadRecord)
			downloadRecords.POST("/batch-delete", s.handleBatchDeleteDownloadRecords)
		}

		// 图片代理（用于解决B站防盗链问题）
		api.GET("/image-proxy", s.handleImageProxy)

		// 快捷订阅
		subscription := api.Group("/subscription")
		{
			// 获取列表
			subscription.GET("/favorites", s.handleGetMyFavorites)   // 我的收藏夹列表
			subscription.GET("/followings", s.handleGetMyFollowings) // 我关注的UP主列表

			// 订阅操作
			subscription.POST("/favorites", s.handleSubscribeFavorite) // 订阅收藏夹
			subscription.POST("/uppers", s.handleSubscribeUpper)       // 订阅UP主

			// 取消订阅
			subscription.DELETE("/favorites/:fid", s.handleUnsubscribeFavorite) // 取消订阅收藏夹
			subscription.DELETE("/uppers/:mid", s.handleUnsubscribeUpper)       // 取消订阅UP主
		}

		// WebSocket
		api.GET("/ws", s.handleWebSocket)

		// 调度器路由
		s.registerSchedulerRoutes(api)
	}

	// 静态文件服务（下载文件）
	downloadPath := s.config.Paths.DownloadBase
	if downloadPath != "" {
		router.Static("/downloads", downloadPath)
		utils.Info("下载目录静态文件服务: /downloads -> %s", downloadPath)
	}

	// 静态文件服务（前端）
	router.Static("/assets", "./web/dist/assets")
	router.StaticFile("/", "./web/dist/index.html")
	router.NoRoute(func(c *gin.Context) {
		c.File("./web/dist/index.html")
	})

	s.router = router
}

// Start 启动服务器
func (s *Server) Start() error {
	// 启动 WebSocket Hub
	go s.websocketHub.Run()

	// 注意：下载管理器应该在创建 Server 之前已经启动（在 main.go 中）
	// 这里不再重复启动，避免 "管理器已在运行" 错误

	// 添加日志钩子
	logHook := NewWebSocketLogHook(s.websocketHub)
	logger := utils.GetLogger()
	logger.AddHook(logHook)

	// 监听下载管理器事件，推送到 WebSocket
	s.downloadMgr.AddEventHandler(func(event downloader.ManagerEvent) {
		// 新下载记录创建事件
		if event.Type == downloader.EventRecordCreated && event.Record != nil {
			s.websocketHub.Broadcast(WebSocketMessage{
				Type:      "download_record_created",
				Data:      event.Record,
				Timestamp: event.Timestamp,
			})
			return
		}

		// 下载记录进度事件
		if event.Type == "download_record_progress" && event.Task != nil && event.Progress != nil {
			s.websocketHub.Broadcast(WebSocketMessage{
				Type: "download_progress",
				Data: gin.H{
					"record_id": event.Task.RecordID,
					"file_name": event.Message,
					"status":    event.Progress.Status,
					"progress":  event.Progress.Progress,
					"speed":     event.Progress.Speed,
					"size":      event.Progress.DownloadedSize,
				},
				Timestamp: event.Timestamp,
			})
			return
		}

		// 任务完成/失败时推送下载记录状态变更
		if (event.Type == downloader.EventTaskCompleted || event.Type == downloader.EventTaskFailed) && event.Task != nil && event.Task.RecordID > 0 {
			status := "completed"
			if event.Type == downloader.EventTaskFailed {
				status = "failed"
			}
			s.websocketHub.Broadcast(WebSocketMessage{
				Type: "download_status",
				Data: gin.H{
					"record_id": event.Task.RecordID,
					"status":    status,
				},
				Timestamp: event.Timestamp,
			})
		}

		// 原有的通用事件推送
		s.websocketHub.Broadcast(WebSocketMessage{
			Type:      string(event.Type),
			Data:      event,
			Timestamp: time.Now(),
		})
	})

	// 监听调度器事件，推送到 WebSocket
	s.scheduler.OnEvent(func(event scheduler.Event) {
		s.websocketHub.Broadcast(WebSocketMessage{
			Type:      string(event.Type),
			Data:      event.Data,
			Timestamp: event.Timestamp,
		})
	})

	// 创建 HTTP 服务器
	s.httpServer = &http.Server{
		Addr:         s.config.Server.BindAddress,
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	utils.Info("HTTP 服务器启动在: %s", s.config.Server.BindAddress)

	// 启动服务器
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("启动 HTTP 服务器失败: %w", err)
	}

	return nil
}

// Shutdown 优雅关闭服务器
func (s *Server) Shutdown(ctx context.Context) error {
	utils.Info("正在关闭 HTTP 服务器...")

	// 停止调度器
	if s.scheduler != nil {
		if err := s.scheduler.Stop(); err != nil {
			utils.Warn("停止调度器失败: %v", err)
		} else {
			utils.Info("调度器已停止")
		}
	}

	// 注意：下载管理器的停止由 main.go 中的 defer 处理
	// 这里不再重复停止

	// 关闭 WebSocket Hub
	s.websocketHub.Stop()

	// 关闭 HTTP 服务器
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("关闭 HTTP 服务器失败: %w", err)
		}
	}

	utils.Info("HTTP 服务器已关闭")
	return nil
}

// StartScheduler 启动调度器
func (s *Server) StartScheduler() error {
	if s.scheduler == nil {
		return fmt.Errorf("调度器未初始化")
	}
	return s.scheduler.Start()
}

// StopScheduler 停止调度器
func (s *Server) StopScheduler() error {
	if s.scheduler == nil {
		return fmt.Errorf("调度器未初始化")
	}
	return s.scheduler.Stop()
}

// loggerMiddleware 日志中间件
func (s *Server) loggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		method := c.Request.Method
		clientIP := c.ClientIP()

		if query != "" {
			path = path + "?" + query
		}

		utils.Debug("[API] %s %s %d %v %s",
			method,
			path,
			status,
			latency,
			clientIP,
		)
	}
}

// authMiddleware JWT 鉴权中间件
func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// 跳过公开接口
		if path == "/api/health" || path == "/api/auth/login" {
			c.Next()
			return
		}

		// 非 /api 路径不需要鉴权（静态文件等）
		if len(path) < 4 || path[:4] != "/api" {
			c.Next()
			return
		}

		// 检查 Authorization header
		token := c.GetHeader("Authorization")
		if token == "" {
			token = c.Query("token")
		}

		// 移除 Bearer 前缀
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		if token == "" {
			respondError(c, http.StatusUnauthorized, "未授权的访问")
			c.Abort()
			return
		}

		claims, err := auth.ParseToken(token)
		if err != nil {
			respondError(c, http.StatusUnauthorized, "登录已过期，请重新登录")
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}
