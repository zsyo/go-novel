package handler

import (
	"fmt"
	"go-novel/internal/config"
	"go-novel/internal/core"
	"go-novel/internal/util"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// BookFetch 获取书籍处理函数
func BookFetch(c *gin.Context) {
	// 获取参数
	bookName := c.Query("bookName")
	author := c.Query("author")
	bookUrl := c.Query("url")
	sourceId, _ := strconv.Atoi(c.Query("sourceId"))
	// 获取format参数，默认为epub
	format := c.Query("format")
	if format == "" {
		format = "epub"
	}
	// 验证format参数
	if format != "epub" && format != "txt" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的格式，仅支持epub和txt"})
		return
	}
	// 获取下载ID参数
	downloadId := c.Query("downloadId")

	if bookName == "" || bookUrl == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "书名或URL不能为空"})
		return
	}

	// 获取配置
	cfg := config.GetConfig()

	// 设置下载配置
	downloadCfg := *cfg // 复制一份配置
	downloadCfg.Source.SourceId = sourceId
	// 设置文件格式
	downloadCfg.Download.ExtName = format
	// 设置下载ID
	downloadCfg.Download.DownloadId = downloadId

	// 确保下载目录存在
	if !util.FileExists(cfg.Download.DownloadPath) {
		os.MkdirAll(cfg.Download.DownloadPath, 0755)
	}

	// 创建爬虫实例
	crawler := core.NewCrawler(&downloadCfg)

	// 开启一个 goroutine 执行下载
	go func() {
		err := crawler.Crawl(bookUrl)
		if err != nil {
			log.Printf("下载书籍失败: %v", err)
		}
	}()

	// 等待一段时间确保爬虫开始执行
	time.Sleep(100 * time.Millisecond)

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"message":  "已开始下载书籍",
		"bookName": bookName,
		"author":   author,
		"sourceId": sourceId,
		"format":   format,
	})
}

// BookDownload 下载书籍文件处理函数
func BookDownload(c *gin.Context) {
	filename := c.Query("filename")
	if filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件名不能为空"})
		return
	}

	// 获取配置
	cfg := config.GetConfig()

	// 获取工作目录
	wd, err := os.Getwd()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("获取工作目录失败: %v", err)})
		return
	}

	// 构建下载目录的绝对路径
	var downloadDir string
	if filepath.IsAbs(cfg.Download.DownloadPath) {
		downloadDir = cfg.Download.DownloadPath
	} else {
		downloadDir = filepath.Join(wd, cfg.Download.DownloadPath)
	}

	// 构建文件的绝对路径
	filePath := filepath.Join(downloadDir, filename)

	// 记录调试信息
	log.Printf("尝试下载文件: %s, 当前工作目录: %s", filePath, wd)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("文件不存在: %s", filePath)
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	// 获取文件信息
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("获取文件信息失败: %v", err)})
		return
	}

	// 如果是目录，拒绝下载
	if fileInfo.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法下载目录"})
		return
	}

	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("打开文件失败: %v", err)})
		return
	}
	defer file.Close()

	// 设置文件类型
	extension := filepath.Ext(filename)
	contentType := "application/octet-stream" // 默认类型

	// 根据扩展名设置正确的内容类型
	switch strings.ToLower(extension) {
	case ".epub":
		contentType = "application/epub+zip"
	case ".pdf":
		contentType = "application/pdf"
	case ".txt":
		contentType = "text/plain"
	case ".html":
		contentType = "text/html"
	}

	// 设置响应头
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", contentType)
	c.Header("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Cache-Control", "no-cache")

	// 手动发送文件
	c.Status(http.StatusOK)
	io.Copy(c.Writer, file)
}

// LocalBooks 获取本地书籍列表处理函数
func LocalBooks(c *gin.Context) {
	// 获取配置
	cfg := config.GetConfig()

	// 获取工作目录
	wd, err := os.Getwd()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("获取工作目录失败: %v", err)})
		return
	}

	// 构建下载目录的绝对路径
	var downloadPath string
	if filepath.IsAbs(cfg.Download.DownloadPath) {
		downloadPath = cfg.Download.DownloadPath
	} else {
		downloadPath = filepath.Join(wd, cfg.Download.DownloadPath)
	}

	// 确保下载目录存在
	if !util.FileExists(downloadPath) {
		os.MkdirAll(downloadPath, 0755)
	}

	// 读取目录中的文件
	files, err := os.ReadDir(downloadPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("读取下载目录失败: %v", err)})
		return
	}

	// 构建书籍列表
	books := []map[string]interface{}{}
	for _, file := range files {
		// 只处理文件，跳过目录
		if file.IsDir() {
			continue
		}

		// 获取文件信息
		filePath := filepath.Join(downloadPath, file.Name())
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			continue
		}

		// 添加到书籍列表
		books = append(books, map[string]interface{}{
			"name":      file.Name(),
			"size":      fileInfo.Size(),
			"timestamp": fileInfo.ModTime().UnixMilli(),
		})
	}

	// 按时间倒序排序
	sort.Slice(books, func(i, j int) bool {
		return books[i]["timestamp"].(int64) > books[j]["timestamp"].(int64)
	})

	log.Printf("找到 %d 本书籍，下载目录: %s", len(books), downloadPath)

	c.JSON(http.StatusOK, gin.H{
		"data": books,
	})
}

// DeleteBook 删除本地书籍文件处理函数
func DeleteBook(c *gin.Context) {
	filename := c.Query("filename")
	if filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件名不能为空"})
		return
	}

	// 防止路径遍历攻击
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文件名"})
		return
	}

	// 获取配置
	cfg := config.GetConfig()

	// 获取工作目录
	wd, err := os.Getwd()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("获取工作目录失败: %v", err)})
		return
	}

	// 构建下载目录的绝对路径
	var downloadPath string
	if filepath.IsAbs(cfg.Download.DownloadPath) {
		downloadPath = cfg.Download.DownloadPath
	} else {
		downloadPath = filepath.Join(wd, cfg.Download.DownloadPath)
	}

	// 构建文件的绝对路径
	filePath := filepath.Join(downloadPath, filename)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	// 删除文件
	err = os.Remove(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("删除文件失败: %v", err)})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"message": "文件删除成功",
	})
}

// StopDownload 停止下载处理函数
func StopDownload(c *gin.Context) {
	downloadId := c.Query("downloadId")
	if downloadId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "下载ID不能为空"})
		return
	}

	log.Printf("收到停止下载请求，下载ID: %s", downloadId)

	// 获取下载管理器实例
	downloadManager := core.GetDownloadManager()

	// 尝试取消下载任务
	if downloadManager.CancelTask(downloadId) {
		log.Printf("成功取消下载任务，下载ID: %s", downloadId)
		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("下载任务已停止，下载ID: %s", downloadId),
		})
	} else {
		log.Printf("未找到下载任务，下载ID: %s", downloadId)
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("未找到下载任务，下载ID: %s", downloadId),
		})
	}
}
