package core

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"so-novel/internal/config"
	"so-novel/internal/rules"
)

const (
	// SearchTimeout 搜索超时时间（秒）
	SearchTimeout = 10
)

// Crawler 爬虫结构体
type Crawler struct {
	config *config.Config
	client *http.Client
}

// GetClient 返回HTTP客户端，用于测试
func (c *Crawler) GetClient() *http.Client {
	return c.client
}

// NewCrawler 创建新的爬虫实例
func NewCrawler(cfg *config.Config) *Crawler {
	// 使用固定的搜索超时时间
	timeout := SearchTimeout

	fmt.Printf("Debug: 使用HTTP超时时间: %d秒\n", timeout)

	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	// 如果启用了代理配置，则设置代理
	if cfg.Proxy.Enabled == 1 && cfg.Proxy.Host != "" && cfg.Proxy.Port > 0 {
		proxyURL := fmt.Sprintf("http://%s:%d", cfg.Proxy.Host, cfg.Proxy.Port)
		proxy, err := url.Parse(proxyURL)
		if err == nil {
			// 创建带代理的传输
			transport := &http.Transport{
				Proxy: http.ProxyURL(proxy),
			}
			client.Transport = transport
			fmt.Printf("Debug: 代理已启用，地址: %s\n", proxyURL)
		} else {
			fmt.Printf("Debug: 代理配置解析失败: %v\n", err)
		}
	} else {
		fmt.Printf("Debug: 代理未启用或配置不完整\n")
	}

	// 设置Cookie jar以支持Cookie持久化
	client.Jar, _ = cookiejar.New(nil)

	return &Crawler{
		config: cfg,
		client: client,
	}
}

// Crawl 开始爬取书籍
func (c *Crawler) Crawl(bookUrl string) error {
	// 使用配置中的源ID
	sourceId := c.config.Source.SourceId

	// 如果源ID无效，尝试从URL中获取
	if sourceId <= 0 {
		sourceId = getSourceIdFromUrl(bookUrl)
	}

	// 加载规则
	ruleManager := rules.GetRuleManager()
	rule, err := ruleManager.GetRuleById(c.config.Source.ActiveRules, sourceId)
	if err != nil || rule == nil {
		return fmt.Errorf("无法加载规则: %v (源ID: %d)", err, sourceId)
	}

	// 解析书籍信息
	book, err := c.parseBookInfo(bookUrl, rule)
	if err != nil {
		return fmt.Errorf("解析书籍信息失败: %v", err)
	}

	// 解析章节目录
	chapters, err := c.parseToc(bookUrl, rule)
	if err != nil {
		return fmt.Errorf("解析章节目录失败: %v", err)
	}

	// 下载章节
	err = c.downloadChapters(book, chapters, rule)
	if err != nil {
		return fmt.Errorf("下载章节失败: %v", err)
	}

	return nil
}
