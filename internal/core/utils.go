package core

import (
	"fmt"
	"math/rand"
	"net/url"
	"strconv"
	"strings"

	"so-novel/internal/sse"
)

// randomUserAgent 生成随机User-Agent
func randomUserAgent() string {
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 Edg/91.0.864.59",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0 Mobile/15E148 Safari/604.1",
	}

	return userAgents[rand.Intn(len(userAgents))]
}

// joinURL 连接基础URL和相对URL
func joinURL(baseURL, relativeURL string) string {
	// 解析基础URL
	base, err := url.Parse(baseURL)
	if err != nil {
		return baseURL + relativeURL
	}

	// 解析相对URL
	rel, err := url.Parse(relativeURL)
	if err != nil {
		return baseURL + relativeURL
	}

	// 解析并返回完整URL
	return base.ResolveReference(rel).String()
}

// sendProgress 发送进度更新
func sendProgress(current, total int) {
	// 发送进度到所有SSE客户端
	sse.SendProgress(current, total)

	// 记录日志，方便调试
	fmt.Printf("下载进度: %d/%d (%.2f%%)\n", current, total, float64(current)/float64(total)*100)
}

// sendFinalProgress 发送最终完成进度
func sendFinalProgress(total int) {
	// 确保进度为100%
	sse.SendProgress(total, total)

	// 发送下载完成消息，使用标准的SendComplete函数
	sse.SendComplete(total)

	// 记录日志
	fmt.Printf("下载完成，总章节数: %d\n", total)
}

// sendError 发送错误信息到前端
func sendError(message string) {
	errorMessage := fmt.Sprintf(`{"type":"book-download-error","message":"%s"}`, message)
	sse.PushMessageToAll(errorMessage)
	fmt.Printf("下载错误: %s\n", message)
}

// getSourceIdFromUrl 从URL中提取源ID
func getSourceIdFromUrl(bookUrl string) int {
	// 查找URL中的sourceId参数
	if strings.Contains(bookUrl, "sourceId=") {
		// 解析URL
		parsedURL, err := url.Parse(bookUrl)
		if err != nil {
			return -1
		}

		// 获取查询参数
		query := parsedURL.Query()
		sourceIdStr := query.Get("sourceId")
		if sourceIdStr != "" {
			// 转换为整数
			sourceId, err := strconv.Atoi(sourceIdStr)
			if err == nil {
				return sourceId
			}
		}
	}

	// 如果没有找到sourceId参数，返回默认值-1
	return -1
}
