package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/c.chen/aura/api"
	"github.com/c.chen/aura/config"
)

//go:embed web/index.html
var indexHTML []byte

var (
	// Version 版本信息（由 GoReleaser 注入）
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

func main() {
	// 命令行参数
	showVersion := flag.Bool("version", false, "显示版本信息")
	flag.Parse()

	if *showVersion {
		log.Printf("aura %s (commit: %s, built at: %s)", Version, Commit, Date)
		return
	}

	// 配置 - 优先级：环境变量 > 默认值
	cfg := config.DefaultConfig()

	// PROM_BASEURL 环境变量
	if baseURL := os.Getenv("PROM_BASEURL"); baseURL != "" {
		cfg.BaseURL = baseURL
	}

	// API 服务器地址（可通过环境变量覆盖）
	addr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port
	}

	// 打印启动信息
	listenURL := "http://localhost" + addr

	fmt.Printf("\n")
	fmt.Printf("  版本:      %s (commit: %s, built: %s)\n", Version, Commit, Date)
	fmt.Printf("  Prometheus: %s\n", cfg.BaseURL)
	fmt.Printf("  监听地址:   %s\n", listenURL)
	fmt.Printf("\n")
	log.Printf("服务启动成功，访问 %s 查看监控面板", listenURL)

	// 创建并启动服务器
	server := api.NewServer(cfg, addr, indexHTML)

	// 优雅退出处理
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("正在关闭服务器...")
		ctx := context.Background()
		server.Shutdown(ctx)
		log.Println("服务器已关闭")
		os.Exit(0)
	}()

	if err := server.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("服务器启动失败: %v", err)
	}
}

func init() {
	// 添加版本信息到 log 前缀（可选）
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
