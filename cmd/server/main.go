package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bili-download/internal/api"
	"bili-download/internal/auth"
	"bili-download/internal/bilibili"
	"bili-download/internal/config"
	"bili-download/internal/database"
	"bili-download/internal/downloader"
	"bili-download/internal/utils"
)

var (
	configPath string
	version    = "dev"
	buildTime  = "unknown"
)

func init() {
	flag.StringVar(&configPath, "config", "", "配置文件路径")
	flag.Parse()
}

func main() {
	// 打印版本信息
	fmt.Printf("bili-download v%s (built at %s)\n", version, buildTime)

	// 加载配置
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		log.Fatalf("配置验证失败: %v", err)
	}

	log.Println("配置加载成功")

	// 初始化日志系统
	if err := utils.InitLogger(cfg); err != nil {
		log.Fatalf("初始化日志系统失败: %v", err)
	}
	utils.Info("日志系统初始化成功")

	// 初始化数据库连接
	db, err := database.Connect(cfg)
	if err != nil {
		utils.Error("连接数据库失败: %v", err)
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer database.Close()
	utils.Info("数据库连接成功")

	// 执行数据库迁移
	if err := database.Migrate(db); err != nil {
		utils.Error("数据库迁移失败: %v", err)
		log.Fatalf("数据库迁移失败: %v", err)
	}
	utils.Info("数据库迁移完成")

	// 初始化 JWT
	auth.InitJWTSecret(cfg.Server.JWTSecret)

	// 创建默认用户
	if err := api.SeedDefaultUser(db); err != nil {
		utils.Warn("创建默认用户失败: %v", err)
	}

	// 初始化 B站 客户端
	biliClient := bilibili.NewClient(cfg)
	utils.Info("B站客户端初始化成功")

	// 初始化下载管理器
	downloadMgr, err := downloader.NewDownloadManager(cfg, db, biliClient)
	if err != nil {
		utils.Error("创建下载管理器失败: %v", err)
		log.Fatalf("创建下载管理器失败: %v", err)
	}

	// 启动下载管理器
	if err := downloadMgr.Start(); err != nil {
		utils.Error("启动下载管理器失败: %v", err)
		log.Fatalf("启动下载管理器失败: %v", err)
	}
	defer downloadMgr.Stop()
	utils.Info("下载管理器已启动")

	// 启动 HTTP 服务器
	server, err := api.NewServer(cfg, configPath, db, biliClient, downloadMgr)
	if err != nil {
		utils.Error("创建 HTTP 服务器失败: %v", err)
		log.Fatalf("创建 HTTP 服务器失败: %v", err)
	}

	// 自动启动调度器
	if err := server.StartScheduler(); err != nil {
		utils.Warn("自动启动调度器失败: %v", err)
	} else {
		utils.Info("调度器已自动启动")
	}

	go func() {
		if err := server.Start(); err != nil {
			utils.Error("启动 HTTP 服务器失败: %v", err)
		}
	}()

	utils.Info("服务启动成功，监听地址: %s", cfg.Server.BindAddress)

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	utils.Info("正在关闭服务...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 关闭 HTTP 服务器
	if err := server.Shutdown(ctx); err != nil {
		utils.Error("关闭 HTTP 服务器失败: %v", err)
	}

	utils.Info("服务已关闭")
}
