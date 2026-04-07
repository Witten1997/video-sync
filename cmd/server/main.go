package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"bili-download/internal/api"
	"bili-download/internal/app"
	"bili-download/internal/auth"
	"bili-download/internal/bilibili"
	"bili-download/internal/config"
	"bili-download/internal/database"
	"bili-download/internal/downloader"
	"bili-download/internal/service"
	"bili-download/internal/telegram"
	"bili-download/internal/utils"
	"bili-download/internal/version"
	frontend "bili-download/web"
)

var (
	configPath string
)

func init() {
	flag.StringVar(&configPath, "config", "", "config file path")
	flag.Parse()
}

func main() {
	fmt.Printf("bili-download v%s (built at %s)\n", version.Version, version.BuildTime)

	if cwd, err := os.Getwd(); err == nil {
		fmt.Printf("working directory: %s\n", cwd)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}
	fmt.Printf("config file: %s\n", config.GetConfigPath())

	if err := cfg.Validate(); err != nil {
		log.Fatalf("validate config failed: %v", err)
	}

	if err := utils.InitLogger(cfg); err != nil {
		log.Fatalf("init logger failed: %v", err)
	}
	utils.Info("logger initialized")
	utils.Info("config file: %s", config.GetConfigPath())

	db, err := database.Connect(cfg)
	if err != nil {
		utils.Error("connect database failed: %v", err)
		log.Fatalf("connect database failed: %v", err)
	}
	defer database.Close()
	utils.Info("database connected")

	if err := database.Migrate(db); err != nil {
		utils.Error("database migration failed: %v", err)
		log.Fatalf("database migration failed: %v", err)
	}
	utils.Info("database migration finished")

	auth.InitJWTSecret(cfg.Server.JWTSecret)

	if err := api.SeedDefaultUser(db); err != nil {
		utils.Warn("seed default user failed: %v", err)
	}

	biliClient := bilibili.NewClient(cfg)
	utils.Info("bilibili client initialized")

	downloadMgr, err := downloader.NewDownloadManager(cfg, db, biliClient)
	if err != nil {
		utils.Error("create download manager failed: %v", err)
		log.Fatalf("create download manager failed: %v", err)
	}

	if err := downloadMgr.Start(); err != nil {
		utils.Error("start download manager failed: %v", err)
		log.Fatalf("start download manager failed: %v", err)
	}
	defer downloadMgr.Stop()
	utils.Info("download manager started")

	urlDownloadService := service.NewURLDownloadService(cfg, db, biliClient, downloadMgr)
	telegramService := telegram.NewBotService(cfg, db, urlDownloadService)

	server, err := api.NewServer(cfg, configPath, db, biliClient, downloadMgr, urlDownloadService, frontend.GetFS())
	if err != nil {
		utils.Error("create http server failed: %v", err)
		log.Fatalf("create http server failed: %v", err)
	}
	server.AttachTelegramService(telegramService)

	if err := server.StartScheduler(); err != nil {
		utils.Warn("start scheduler failed: %v", err)
	} else {
		utils.Info("scheduler started")
	}

	go func() {
		if err := server.Start(); err != nil {
			utils.Error("start http server failed: %v", err)
		}
	}()

	telegramCtx, telegramCancel := context.WithCancel(context.Background())
	defer telegramCancel()

	go func() {
		if err := telegramService.Start(telegramCtx); err != nil {
			utils.Error("start telegram service failed: %v", err)
		}
	}()

	utils.Info("server listening at %s", cfg.Server.BindAddress)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		utils.Info("shutting down services")
		telegramCancel()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			utils.Error("shutdown http server failed: %v", err)
		}

		utils.Info("services stopped")

	case newBinaryPath := <-server.UpgradeSignal:
		utils.Info("received upgrade signal")
		telegramCancel()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			utils.Error("shutdown http server failed: %v", err)
		}

		currentBinary, err := os.Executable()
		if err != nil {
			utils.Error("resolve current binary failed: %v", err)
			return
		}
		currentBinary, _ = filepath.EvalSymlinks(currentBinary)

		backupPath := currentBinary + ".old"
		os.Remove(backupPath)
		if err := os.Rename(currentBinary, backupPath); err != nil {
			utils.Error("backup current binary failed: %v", err)
			return
		}

		if err := moveFile(newBinaryPath, currentBinary); err != nil {
			utils.Error("replace binary failed: %v", err)
			_ = os.Rename(backupPath, currentBinary)
			return
		}

		if runtime.GOOS != "windows" {
			_ = os.Chmod(currentBinary, 0755)
		}

		_ = os.RemoveAll(filepath.Join("storage", "temp", "upgrade"))
		utils.Info("upgrade finished, restarting process")

		if err := app.RestartProcess(currentBinary, os.Args, os.Environ()); err != nil {
			utils.Error("restart failed: %v", err)
		}
	}
}

func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := out.ReadFrom(in); err != nil {
		return err
	}

	if err := in.Close(); err != nil {
		return err
	}

	return os.Remove(src)
}
