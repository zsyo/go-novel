package web

import (
	"embed"
	"fmt"
	"io"
	"net/http"
	"path"
	"so-novel/internal/config"
	"so-novel/internal/handler"
	"so-novel/internal/sse"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

//go:embed static/*
var staticFiles embed.FS

// 启动SSE心跳服务
func startSSEHeartbeat() {
	go func() {
		ticker := time.NewTicker(15 * time.Second) // 缩短心跳包间隔为15秒
		defer ticker.Stop()

		for range ticker.C {
			sse.PushMessageToAll(":heartbeat")
		}
	}()
}

// serveEmbedFile 从嵌入的文件系统中提供文件
func serveEmbedFile(c *gin.Context, fs embed.FS, filePath string) {
	file, err := fs.Open(filePath)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	// 根据文件扩展名设置 Content-Type
	var contentType string
	switch path.Ext(filePath) {
	case ".css":
		contentType = "text/css; charset=utf-8"
	case ".html":
		contentType = "text/html; charset=utf-8"
	case ".js":
		contentType = "application/javascript"
	case ".ico":
		contentType = "image/x-icon"
	default:
		contentType = http.DetectContentType(content)
	}

	c.Data(http.StatusOK, contentType, content)
}

func StartServer(cfg *config.Config) {
	if cfg.Web.Enabled != 1 {
		fmt.Println("Web server is disabled")
		return
	}

	// 设置为发布模式
	// gin.SetMode(gin.ReleaseMode)

	// 设置为debug模式
	gin.SetMode(gin.DebugMode)

	// 创建gin引擎
	r := gin.Default()

	// 设置静态文件服务，使用嵌入的文件系统
	r.GET("/css/*filepath", func(c *gin.Context) {
		filepath := c.Param("filepath")
		// 移除前导斜杠
		filepath = strings.TrimPrefix(filepath, "/")
		// 构造完整的路径
		fullPath := path.Join("css", filepath)
		serveEmbedFile(c, staticFiles, path.Join("static", fullPath))
	})

	// 提供 favicon.ico 文件
	r.GET("/favicon.ico", func(c *gin.Context) {
		serveEmbedFile(c, staticFiles, "static/favicon.ico")
	})

	// 提供 index.html 文件
	r.GET("/", func(c *gin.Context) {
		serveEmbedFile(c, staticFiles, "static/index.html")
	})

	// API路由
	api := r.Group("/api")
	{
		api.GET("/search/aggregated", handler.AggregatedSearch)
		api.GET("/book/fetch", handler.BookFetch)
		api.GET("/book/download", handler.BookDownload)
		api.GET("/local/books", handler.LocalBooks)
		api.DELETE("/book", handler.DeleteBook)
	}

	// SSE路由
	r.GET("/sse/book/progress", sse.ProgressSSE)

	// 启动SSE心跳服务
	startSSEHeartbeat()

	// 启动服务器
	port := cfg.Web.Port
	if port == 0 {
		port = 7765
	}

	fmt.Printf("服务器启动在端口 %d，访问地址：http://localhost:%d\n", port, port)
	r.Run(":" + fmt.Sprintf("%d", port))
}
