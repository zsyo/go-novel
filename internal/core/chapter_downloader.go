package core

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"go-novel/internal/model"

	"github.com/PuerkitoBio/goquery"
)

// downloadChapters 下载章节
func (c *Crawler) downloadChapters(book *model.Book, chapters []model.Chapter, rule *model.Rule) error {
	total := len(chapters)
	fmt.Printf("共计 %d 章\n", total)

	// 获取客户端ID，必须提供
	clientID := ""
	if c.config.Download.DownloadId != "" {
		downloadManager := GetDownloadManager()
		clientID, _ = downloadManager.GetClientID(c.config.Download.DownloadId)
	}

	// 如果没有客户端ID，返回错误
	if clientID == "" {
		return errors.New("缺少客户端ID，无法发送进度更新")
	}

	// 发送开始下载消息到特定客户端
	sendProgressToClient(clientID, 0, total)

	// 设置线程数
	threads := c.config.Crawl.Threads
	if threads == -1 {
		// 自动设置线程数，使用CPU核心数的2倍，但最多不超过8个
		threads = runtime.NumCPU() * 2
		if threads > 8 {
			threads = 8
		}
	}
	fmt.Printf("开始下载《%s》(%s) 共计 %d 章 | 线程数：%d\n",
		book.BookName, book.Author, total, threads)

	// 获取context，如果下载ID存在的话
	var ctx context.Context
	var cancel context.CancelFunc
	if c.config.Download.DownloadId != "" {
		downloadManager := GetDownloadManager()
		// 获取任务的context
		downloadManager.mutex.RLock()
		task, exists := downloadManager.tasks[c.config.Download.DownloadId]
		downloadManager.mutex.RUnlock()

		if exists {
			ctx = task.Context
			cancel = task.Cancel
		} else {
			// 如果任务不存在，创建新的context
			ctx, cancel = context.WithCancel(context.Background())
			defer cancel()
		}
	} else {
		// 如果没有下载ID，创建新的context
		ctx, cancel = context.WithCancel(context.Background())
		defer cancel()
	}

	// 创建信号量控制并发
	sem := make(chan struct{}, threads)
	// 使用WaitGroup等待所有goroutine完成
	var wg sync.WaitGroup
	// 使用互斥锁保护进度更新
	var mutex sync.Mutex
	// 记录已完成数量
	completed := 0

	// 创建错误通道，收集下载过程中的错误
	errChan := make(chan error, total)

	// 下载开始时间
	startTime := time.Now()

	// 并发下载章节
exit:
	for i := range chapters {
		// 检查context是否已取消
		select {
		case <-ctx.Done():
			// context已取消，停止创建新的goroutine
			fmt.Println("检测到下载取消信号，停止创建新的下载任务")
			break exit
		default:
		}

		sem <- struct{}{} // 获取信号量

		wg.Go(func() {
			defer func() {
				<-sem // 释放信号量
			}()

			// 检查context是否已取消
			select {
			case <-ctx.Done():
				// context已取消，直接返回
				return
			default:
			}

			// 下载章节内容
			content, err := c.downloadChapterContent(ctx, chapters[i].URL, rule)
			if err != nil {
				errMsg := fmt.Sprintf("下载章节失败 %s: %v", chapters[i].Title, err)
				errChan <- errors.New(errMsg)
				// 发送错误信息到特定客户端
				sendErrorToClient(clientID, errMsg)
				// 取消context，停止所有下载
				cancel()
				return
			}

			// 更新章节内容
			mutex.Lock()
			chapters[i].Content = content
			completed++
			// 发送进度更新到特定客户端
			sendProgressToClient(clientID, completed, total)
			mutex.Unlock()

			// 控制下载速度
			minInterval := c.config.Crawl.MinInterval
			maxInterval := c.config.Crawl.MaxInterval
			if maxInterval > minInterval {
				interval := time.Duration(minInterval + rand.Intn(maxInterval-minInterval))
				time.Sleep(time.Millisecond * interval)
			}
		})
	}

	// 等待所有下载完成或被取消
	wg.Wait()
	close(errChan)

	// 检查是否因章节错误而取消
	select {
	case <-ctx.Done():
		// context被取消，说明有章节下载失败或用户手动取消
		fmt.Println("下载已被取消，可能是因为章节下载失败或用户手动取消")
		// 发送错误消息到特定客户端
		sendErrorToClient(clientID, "下载已被取消")
		return errors.New("下载已被取消")
	default:
	}

	// 输出错误信息
	errCount := 0
	for err := range errChan {
		fmt.Println(err)
		errCount++
	}

	// 计算下载用时
	elapsed := time.Since(startTime)
	fmt.Printf("下载完成！总耗时: %.2f 秒, 成功: %d章, 失败: %d章\n",
		elapsed.Seconds(), completed, errCount)

	// 保存书籍
	err := c.saveBook(ctx, book, chapters)
	if err != nil {
		// 发送错误消息到特定客户端
		errMsg := fmt.Sprintf("保存书籍失败: %w", err)
		sendErrorToClient(clientID, errMsg)
		return fmt.Errorf("保存书籍失败: %w", err)
	}

	// 发送最终的完成消息到特定客户端
	sendFinalProgressToClient(clientID, total)

	return nil
}

// downloadChapterContent 下载章节内容
func (c *Crawler) downloadChapterContent(ctx context.Context, chapterUrl string, rule *model.Rule) (string, error) {
	// 发起HTTP请求（带重试机制）
	resp, err := c.getWithRetry(ctx, chapterUrl)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 解析HTML文档
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}

	// 提取章节内容
	var content string
	if rule.Chapter.Content != "" {
		content = c.extractText(doc.Selection, rule.Chapter.Content)
	}

	// 应用过滤规则
	if rule.Chapter.FilterTxt != "" {
		// 修复正则表达式中的转义问题
		filterTxt := rule.Chapter.FilterTxt
		// 将 \1 替换为 $1，避免正则表达式错误
		filterTxt = strings.ReplaceAll(filterTxt, `\1`, `$1`)
		re := regexp.MustCompile(filterTxt)
		content = re.ReplaceAllString(content, "")
	}

	return content, nil
}

// getWithRetry 带重试机制的HTTP GET请求
func (c *Crawler) getWithRetry(ctx context.Context, url string) (*http.Response, error) {
	var resp *http.Response
	var err error

	// 检查是否启用重试
	maxRetries := 0
	if c.config.Crawl.EnableRetry == 1 {
		maxRetries = c.config.Crawl.MaxRetries
	}

	// 为每次请求创建独立的HTTP客户端，避免共用超时设置
	client := c.NewHTTPClinet()

	// 初始尝试
	resp, err = client.Get(url)
	if err == nil && resp.StatusCode == 200 {
		return resp, nil
	}

	// 如果有错误且启用了重试，则进行重试
	for i := 0; i < maxRetries; i++ {
		if resp != nil {
			resp.Body.Close()
		}

		// 计算重试间隔（指数退避）
		minInterval := c.config.Crawl.RetryMinInterval
		maxInterval := c.config.Crawl.RetryMaxInterval
		interval := minInterval
		if maxInterval > minInterval {
			interval = minInterval + rand.Intn(maxInterval-minInterval)
		}

		// 睡眠时也检查context是否已取消
		sleepTimer := time.NewTimer(time.Duration(interval) * time.Millisecond)
		select {
		case <-sleepTimer.C:
			// 睡眠完成，继续重试
		case <-ctx.Done():
			// context已取消，停止睡眠并返回错误
			sleepTimer.Stop()
			return nil, ctx.Err()
		}

		fmt.Printf("重试下载章节 %s (第 %d/%d 次)\n", url, i+1, maxRetries)

		// 重新发起请求，使用独立的客户端
		resp, err = client.Get(url)
		if err == nil && resp.StatusCode == 200 {
			return resp, nil
		}
	}

	return resp, fmt.Errorf("请求失败，已重试%d次: %w", maxRetries, err)
}
