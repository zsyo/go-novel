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

// Client 表示一个SSE客户端
type Client struct {
	ID   string
	Chan chan string
}

// ClientManager SSE客户端管理器
type ClientManager struct {
	clients map[string]*Client
	mutex   sync.RWMutex
}

var manager = &ClientManager{
	clients: make(map[string]*Client),
}

// ProgressSSE SSE处理函数
func ProgressSSE(c *gin.Context) {
	// 获取客户端ID参数，必须提供
	clientID := c.Query("clientId")
	if clientID == "" {
		// 如果没有提供clientId，返回错误
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少clientId参数"})
		return
	}

	// 设置SSE响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	// 设置超时时间，避免长时间阻塞
	c.Writer.Flush()

	// 创建客户端通道，增加缓冲区大小以减少阻塞风险
	clientChan := make(chan string, 50)

	// 创建客户端对象
	client := &Client{
		ID:   clientID,
		Chan: clientChan,
	}

	// 添加客户端到管理器
	manager.addClient(client)
	defer manager.removeClient(clientID)

	// 发送初始连接消息表示连接已建立，包含客户端ID
	initialMsg := fmt.Sprintf(`{"type":"connected","message":"connected","clientId":"%s"}`, clientID)
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
func (m *ClientManager) addClient(client *Client) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.clients[client.ID] = client
}

// 移除客户端
func (m *ClientManager) removeClient(clientID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if client, exists := m.clients[clientID]; exists {
		delete(m.clients, clientID)
		close(client.Chan)
	}
}

// 向特定客户端推送消息
func (m *ClientManager) sendMessageToClient(clientID, message string) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if client, exists := m.clients[clientID]; exists {
		select {
		case client.Chan <- message:
			return true
		default:
			// 如果通道已满，跳过该客户端
			return false
		}
	}
	return false
}

// 广播消息到所有客户端
func (m *ClientManager) broadcastMessage(message string) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, client := range m.clients {
		select {
		case client.Chan <- message:
		default:
			// 如果通道已满，跳过该客户端
		}
	}
}

// PushMessageToClient 推送消息到特定客户端
func PushMessageToClient(clientID, message string) bool {
	return manager.sendMessageToClient(clientID, message)
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

// SendProgress 发送下载进度到特定客户端
func SendProgressToClient(clientID string, current, total int) {
	message := fmt.Sprintf(`{"type":"book-download","index":%d,"total":%d}`, current, total)
	PushMessageToClient(clientID, message)
}

// SendErrorToClient 发送错误信息到特定客户端
func SendErrorToClient(clientID, message string) {
	// 转义JSON中的特殊字符
	escapedMessage := strings.ReplaceAll(message, "\"", "\\\"")
	escapedMessage = strings.ReplaceAll(escapedMessage, "\n", " ")

	errorMessage := fmt.Sprintf(`{"type":"book-download-error","message":"%s"}`, escapedMessage)
	PushMessageToClient(clientID, errorMessage)
}

// SendCompleteToClient 发送完成消息到特定客户端
func SendCompleteToClient(clientID string, total int) {
	message := fmt.Sprintf(`{"type":"book-download-complete","total":%d}`, total)
	PushMessageToClient(clientID, message)
}
