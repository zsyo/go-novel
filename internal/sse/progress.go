package sse

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ClientManager SSE客户端管理器
type ClientManager struct {
	clients map[chan string]bool
	mutex   sync.RWMutex
}

var manager = &ClientManager{
	clients: make(map[chan string]bool),
}

// ProgressSSE SSE处理函数
func ProgressSSE(c *gin.Context) {
	// 设置SSE响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	// 设置超时时间，避免长时间阻塞
	c.Writer.Flush()

	// 创建客户端通道，增加缓冲区大小以减少阻塞风险
	clientChan := make(chan string, 50)

	// 添加客户端到管理器
	manager.addClient(clientChan)
	defer manager.removeClient(clientChan)

	// 发送初始连接消息表示连接已建立
	initialMsg := `{"type":"connected","message":"connected"}`
	clientChan <- initialMsg

	// 使用stream模式发送SSE消息
	c.Stream(func(w io.Writer) bool {
		select {
		case msg, ok := <-clientChan:
			if !ok {
				// 通道已关闭，结束流
				return false
			}
			// 渲染消息
			c.Render(-1, sseRender{Data: msg})
			// 检查客户端连接是否仍然有效
			if c.Writer.Written() && c.Writer.Status() != http.StatusOK {
				// 状态码不是200，可能是客户端断开连接
				return false
			}
			return true
		case <-c.Request.Context().Done():
			// 请求上下文已取消，客户端可能断开连接
			return false
		}
	})
}

// sseRender SSE渲染器
type sseRender struct {
	Data string
}

func (r sseRender) Render(w http.ResponseWriter) error {
	_, err := w.Write([]byte(fmt.Sprintf("data: %s\n\n", r.Data)))
	return err
}

func (r sseRender) WriteContentType(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
}

// 添加客户端
func (m *ClientManager) addClient(client chan string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.clients[client] = true
}

// 移除客户端
func (m *ClientManager) removeClient(client chan string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.clients, client)
	close(client)
}

// 广播消息到所有客户端
func (m *ClientManager) broadcastMessage(message string) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// fmt.Printf("Debug: Current clients length: %d\n", len(m.clients))
	for client := range m.clients {
		select {
		case client <- message:
			// fmt.Printf("Debug: Broadcasting message to client: %s\n", message)
		default:
			// 如果通道已满，跳过该客户端
		}
	}
}

// PushMessageToAll 推送消息到所有客户端
func PushMessageToAll(message string) {
	manager.broadcastMessage(message)
}

// SendHeartbeat 发送心跳包
func SendHeartbeat() {
	// 发送JSON格式的心跳包
	heartbeatMessage := fmt.Sprintf(`{"type":"heartbeat","timestamp":%d}`, time.Now().Unix())
	PushMessageToAll(heartbeatMessage)
}

// SendProgress 发送下载进度
// 简化数据结构，只发送必要的进度信息
func SendProgress(current, total int) {
	message := fmt.Sprintf(`{"type":"book-download","index":%d,"total":%d}`, current, total)
	PushMessageToAll(message)
}

// SendError 发送错误信息
func SendError(message string) {
	// 转义JSON中的特殊字符
	message = strings.ReplaceAll(message, "\"", "\\\"")
	message = strings.ReplaceAll(message, "\n", " ")

	errorMessage := fmt.Sprintf(`{"type":"book-download-error","message":"%s"}`, message)
	PushMessageToAll(errorMessage)
}

// SendProgressWithChapterName 发送带章节名的下载进度
// 简化版本，忽略章节名，直接调用SendProgress
func SendProgressWithChapterName(current, total int, chapterName string) {
	// 不再发送章节名，直接调用简化版本
	SendProgress(current, total)
}

// SendComplete 发送完成消息
func SendComplete(total int) {
	message := fmt.Sprintf(`{"type":"book-download-complete","total":%d}`, total)
	PushMessageToAll(message)
}
