package main

import (
	"so-novel/internal/config"
	"so-novel/internal/web"
)

func main() {
	// 初始化配置
	cfg := config.InitConfig()

	// 启动Web服务器
	web.StartServer(cfg)
}
