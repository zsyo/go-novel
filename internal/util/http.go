package util

import (
	"fmt"
	"net/http"
	"time"
)

// CreateHTTPClient 创建HTTP客户端
func CreateHTTPClient(timeoutSeconds int) *http.Client {
	return &http.Client{
		Timeout: time.Duration(timeoutSeconds) * time.Second,
	}
}

// GetWithRetry 带重试机制的GET请求
func GetWithRetry(client *http.Client, url string, maxRetries int) (*http.Response, error) {
	var resp *http.Response
	var err error

	for i := 0; i <= maxRetries; i++ {
		resp, err = client.Get(url)
		if err == nil && resp.StatusCode == 200 {
			return resp, nil
		}

		// 如果不是最后一次尝试，等待后重试
		if i < maxRetries {
			// 指数退避
			waitTime := time.Duration(i+1) * time.Second
			time.Sleep(waitTime)
		}
	}

	return resp, fmt.Errorf("请求失败，已重试%d次: %v", maxRetries, err)
}
