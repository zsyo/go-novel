package main

import (
	"go-novel/internal/config"
	"go-novel/internal/embed"
	"go-novel/internal/web"
)

func main() {
	// 初始化配置
	cfg := config.InitConfig()

	err := embed.WriteEmbeddedFiles()
	if err != nil {
		panic(err)
	}

	// 启动Web服务器
	web.StartServer(cfg)
}
