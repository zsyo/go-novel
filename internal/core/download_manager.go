package core

import (
	"context"
	"fmt"
	"sync"
)

// DownloadTask 表示一个下载任务
type DownloadTask struct {
	ID       string
	ClientID string
	Context  context.Context
	Cancel   context.CancelFunc
}

// DownloadManager 下载任务管理器
type DownloadManager struct {
	tasks map[string]*DownloadTask
	mutex sync.RWMutex
}

// 全局下载管理器实例
var downloadManager = &DownloadManager{
	tasks: make(map[string]*DownloadTask),
}

// AddTask 添加下载任务
func (dm *DownloadManager) AddTask(id, clientID string, ctx context.Context, cancel context.CancelFunc) {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	dm.tasks[id] = &DownloadTask{
		ID:       id,
		ClientID: clientID,
		Context:  ctx,
		Cancel:   cancel,
	}
}

// RemoveTask 移除下载任务
func (dm *DownloadManager) RemoveTask(id string) {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	delete(dm.tasks, id)
}

// CancelTask 取消下载任务
func (dm *DownloadManager) CancelTask(id string) bool {
	dm.mutex.RLock()
	task, exists := dm.tasks[id]
	dm.mutex.RUnlock()

	if !exists {
		fmt.Printf("任务未找到，下载ID: %s\n", id)
		return false
	}

	// 调用取消函数
	task.Cancel()
	fmt.Printf("已调用取消函数，下载ID: %s\n", id)
	return true
}

// GetClientID 获取下载任务对应的客户端ID
func (dm *DownloadManager) GetClientID(id string) (string, bool) {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	task, exists := dm.tasks[id]
	if !exists {
		return "", false
	}
	return task.ClientID, true
}

// GetDownloadManager 获取下载管理器实例
func GetDownloadManager() *DownloadManager {
	return downloadManager
}
