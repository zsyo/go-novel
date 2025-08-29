package core

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"so-novel/internal/model"
	"so-novel/internal/rules"

	"github.com/PuerkitoBio/goquery"
)

// Search 搜索小说
func (c *Crawler) Search(keyword string) ([]model.SearchResult, error) {
	// 获取源ID
	sourceId := c.config.Source.SourceId

	// 如果sourceId为-1，表示使用所有可搜索的书源进行聚合搜索
	if sourceId == -1 {
		return c.aggregatedSearch(keyword)
	}

	// 加载规则
	ruleManager := rules.GetRuleManager()
	rule, err := ruleManager.GetRuleById(c.config.Source.ActiveRules, sourceId)
	if err != nil {
		return nil, fmt.Errorf("无法加载规则: %v", err)
	}

	// 检查规则是否存在
	if rule == nil {
		return nil, fmt.Errorf("未找到ID为 %d 的规则", sourceId)
	}

	// 检查是否支持搜索
	if rule.Search.Disabled {
		return nil, fmt.Errorf("书源 %s 不支持搜索", rule.Name)
	}

	// 发起搜索请求
	searchResults, err := c.doSearch(keyword, rule)
	if err != nil {
		return nil, err
	}

	// 限制结果数量
	if c.config.Source.SearchLimit > 0 && len(searchResults) > c.config.Source.SearchLimit {
		searchResults = searchResults[:c.config.Source.SearchLimit]
	}

	return searchResults, nil
}

// aggregatedSearch 聚合搜索实现
func (c *Crawler) aggregatedSearch(keyword string) ([]model.SearchResult, error) {
	// 获取规则管理器
	ruleManager := rules.GetRuleManager()

	// 获取可搜索的规则
	searchableRules, err := ruleManager.GetSearchableRules(c.config.Source.ActiveRules)
	if err != nil {
		return nil, fmt.Errorf("加载可搜索规则失败: %v", err)
	}

	// 存储搜索结果
	var results []model.SearchResult
	resultsMutex := &sync.Mutex{}

	// 创建等待组和goroutine池
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // 限制最大并发数为10

	// 对每个源并发搜索
	for _, rule := range searchableRules {
		wg.Add(1)
		semaphore <- struct{}{} // 获取信号量

		go func(r model.Rule) {
			defer func() {
				<-semaphore // 释放信号量
				wg.Done()
			}()

			// 创建新的爬虫实例用于这个源的搜索
			searchCfg := *c.config // 复制配置
			searchCfg.Source.SourceId = r.ID
			searchCrawler := NewCrawler(&searchCfg)

			// 执行搜索
			searchResults, err := searchCrawler.doSearch(keyword, &r)
			if err != nil {
				fmt.Printf("搜索源 %s (%d) 异常: %v\n", r.Name, r.ID, err)
				return
			}

			if len(searchResults) > 0 {
				fmt.Printf("书源 %d (%s) 搜索到 %d 条记录\n", r.ID, r.Name, len(searchResults))

				// 安全地添加到结果列表
				resultsMutex.Lock()
				results = append(results, searchResults...)
				resultsMutex.Unlock()
			} else {
				// 即使没有结果也记录一下，方便调试
				fmt.Printf("书源 %d (%s) 搜索到 0 条记录\n", r.ID, r.Name)
			}
		}(rule)
	}

	// 等待所有搜索完成
	wg.Wait()

	// 限制结果数量
	if c.config.Source.SearchLimit > 0 && len(results) > c.config.Source.SearchLimit {
		results = results[:c.config.Source.SearchLimit]
	}

	return results, nil
}

// doSearch 执行搜索请求
func (c *Crawler) doSearch(keyword string, rule *model.Rule) ([]model.SearchResult, error) {
	searchRule := rule.Search

	// 构建请求URL
	var requestURL string
	if strings.ToLower(searchRule.Method) == "get" {
		// GET请求需要对关键字进行URL编码
		encodedKeyword := url.QueryEscape(keyword)
		requestURL = strings.ReplaceAll(searchRule.URL, "%s", encodedKeyword)
	} else {
		// POST请求直接替换
		requestURL = strings.ReplaceAll(searchRule.URL, "%s", keyword)
	}

	// 添加详细日志
	fmt.Printf("搜索源 %s (%d) 开始请求: %s [方法: %s]\n", rule.Name, rule.ID, requestURL, searchRule.Method)
	start := time.Now()

	// 创建请求
	var req *http.Request
	var err error

	if strings.ToLower(searchRule.Method) == "post" {
		// 处理POST请求
		data := BuildSearchPostData(searchRule.Data, keyword)
		req, err = http.NewRequest("POST", requestURL, strings.NewReader(data))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		// 处理GET请求
		req, err = http.NewRequest("GET", requestURL, nil)
		if err != nil {
			return nil, err
		}
	}

	// 设置默认User-Agent（如果未设置）
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", randomUserAgent())
	}

	// 设置Cookies（如果规则中有指定）
	if searchRule.Cookies != "" {
		req.Header.Set("Cookie", searchRule.Cookies)
	}

	// 发起请求
	// 为每次请求创建独立的HTTP客户端，避免共用超时设置
	client := &http.Client{
		Timeout: c.client.Timeout,
	}
	if c.client.Transport != nil {
		client.Transport = c.client.Transport
	}
	if c.client.Jar != nil {
		client.Jar = c.client.Jar
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("请求失败，状态码: %d", resp.StatusCode)
	}

	fmt.Printf("搜索源 %s (%d) 请求成功，耗时: %v\n", rule.Name, rule.ID, time.Since(start))

	// 解析搜索结果
	searchResults, err := c.parseSearchResults(resp, rule, keyword)
	if err != nil {
		return nil, fmt.Errorf("解析搜索结果失败: %v", err)
	}

	fmt.Printf("搜索源 %s (%d) 解析到 %d 条结果\n", rule.Name, rule.ID, len(searchResults))

	return searchResults, nil
}

// parseSearchResults 解析搜索结果
func (c *Crawler) parseSearchResults(resp *http.Response, rule *model.Rule, keyword string) ([]model.SearchResult, error) {
	return c.parseSearchResultsInternal(resp, rule, keyword, true)
}

// parseSearchResultsInternal 内部解析搜索结果，allowPagination参数控制是否允许分页
func (c *Crawler) parseSearchResultsInternal(resp *http.Response, rule *model.Rule, keyword string, allowPagination bool) ([]model.SearchResult, error) {
	// 读取响应体
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 将响应体转换为字符串
	bodyStr := string(bodyBytes)

	// 重新创建一个io.Reader以便goquery可以读取
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(bodyStr))
	if err != nil {
		return nil, fmt.Errorf("解析HTML文档失败: %v", err)
	}

	// 获取搜索规则
	searchRule := rule.Search

	// 选择搜索结果元素
	resultElements := doc.Find(searchRule.Result)
	// fmt.Printf("Debug: 选择器'%s'匹配到%d个元素\n", searchRule.Result, resultElements.Length())

	// 特殊处理：部分书源完全匹配时会直接跳转到详情页（搜索结果为空 && 书名不为空）
	if resultElements.Length() == 0 && searchRule.BookName != "" {
		// 检查页面是否包含书籍信息选择器
		bookNameSel := rule.Book.BookName
		if bookNameSel != "" {
			bookNameElements := doc.Find(bookNameSel)
			if bookNameElements.Length() > 0 {
				fmt.Printf("Debug: 检测到直接跳转到详情页的情况\n")

				// 获取当前请求的URL作为书籍URL
				bookUrl := resp.Request.URL.String()

				// 提取书籍信息
				bookName := strings.TrimSpace(bookNameElements.First().Text())
				if bookName != "" {
					result := model.SearchResult{
						SourceId: rule.ID,
						URL:      bookUrl,
						BookName: bookName,
					}

					// 提取其他书籍信息
					if rule.Book.Author != "" {
						authorElements := doc.Find(rule.Book.Author)
						if authorElements.Length() > 0 {
							result.Author = strings.TrimSpace(authorElements.First().Text())
						}
					}

					if rule.Book.LatestChapter != "" {
						latestChapterElements := doc.Find(rule.Book.LatestChapter)
						if latestChapterElements.Length() > 0 {
							result.LatestChapter = strings.TrimSpace(latestChapterElements.First().Text())
						}
					}

					if rule.Book.LastUpdateTime != "" {
						lastUpdateTimeElements := doc.Find(rule.Book.LastUpdateTime)
						if lastUpdateTimeElements.Length() > 0 {
							result.LastUpdateTime = strings.TrimSpace(lastUpdateTimeElements.First().Text())
						}
					}

					fmt.Printf("Debug: 构造了直接跳转的搜索结果: %s\n", bookName)
					return []model.SearchResult{result}, nil
				}
			}
		}
	}

	// 提取搜索结果
	var results []model.SearchResult
	resultElements.Each(func(i int, s *goquery.Selection) {
		result := model.SearchResult{
			SourceId: rule.ID,
		}

		// 提取书籍信息
		if searchRule.BookName != "" {
			// 提取书名
			bookName := c.extractText(s, searchRule.BookName)
			result.BookName = strings.TrimSpace(bookName)
		}

		// 如果书名为空，跳过这个结果
		if result.BookName == "" {
			return
		}

		// 提取作者
		if searchRule.Author != "" {
			author := c.extractText(s, searchRule.Author)
			result.Author = strings.TrimSpace(author)
		}

		// 提取类别
		if searchRule.Category != "" {
			category := c.extractText(s, searchRule.Category)
			result.Category = strings.TrimSpace(category)
		}

		// 提取字数
		if searchRule.WordCount != "" {
			wordCount := c.extractText(s, searchRule.WordCount)
			result.WordCount = strings.TrimSpace(wordCount)
		}

		// 提取状态
		if searchRule.Status != "" {
			status := c.extractText(s, searchRule.Status)
			result.Status = strings.TrimSpace(status)
		}

		// 提取最新章节
		if searchRule.LatestChapter != "" {
			latestChapter := c.extractText(s, searchRule.LatestChapter)
			result.LatestChapter = strings.TrimSpace(latestChapter)
		}

		// 提取最后更新时间
		if searchRule.LastUpdateTime != "" {
			lastUpdateTime := c.extractText(s, searchRule.LastUpdateTime)
			result.LastUpdateTime = strings.TrimSpace(lastUpdateTime)
		}

		// 提取书籍链接
		if searchRule.BookName != "" {
			// 尝试提取href属性并转换为绝对URL
			bookURL := c.extractAbsAttr(s, searchRule.BookName, "href", resp.Request.URL.String())
			if bookURL != "" {
				result.URL = bookURL
			}
		}

		// 如果URL为空，使用请求URL
		if result.URL == "" {
			result.URL = resp.Request.URL.String()
		}

		results = append(results, result)
	})

	// 处理分页（仅在允许分页时处理）
	if allowPagination && searchRule.Pagination && searchRule.NextPage != "" {
		// 提取分页链接
		nextPageElements := doc.Find(searchRule.NextPage)
		if nextPageElements.Length() > 0 {
			fmt.Printf("Debug: 找到 %d 个分页链接\n", nextPageElements.Length())

			// 使用map存储已处理的URL，避免重复请求
			processedURLs := make(map[string]bool)
			processedURLs[resp.Request.URL.String()] = true // 标记当前页已处理

			// 收集所有分页URL
			var pageURLs []string
			nextPageElements.Each(func(i int, s *goquery.Selection) {
				href, exists := s.Attr("href")
				if exists && href != "" {
					// 转换为绝对URL
					absoluteURL := joinURL(resp.Request.URL.String(), href)
					// 避免重复和空URL
					if absoluteURL != "" && !processedURLs[absoluteURL] {
						processedURLs[absoluteURL] = true
						pageURLs = append(pageURLs, absoluteURL)
					}
				}
			})

			// 限制分页数量，避免过多请求
			maxPages := 3
			if len(pageURLs) > maxPages {
				pageURLs = pageURLs[:maxPages]
			}

			// 逐个请求分页并解析结果（不允许再次分页）
			for _, pageURL := range pageURLs {
				// fmt.Printf("Debug: 请求分页: %s\n", pageURL)

				// 创建分页请求
				pageReq, err := http.NewRequest("GET", pageURL, nil)
				if err != nil {
					fmt.Printf("Debug: 创建分页请求失败: %v\n", err)
					continue
				}

				// 设置请求头
				pageReq.Header.Set("User-Agent", randomUserAgent())
				pageReq.Header.Set("Referer", resp.Request.URL.String())

				// 发送请求
				// 为每次请求创建独立的HTTP客户端，避免共用超时设置
				client := &http.Client{
					Timeout: c.client.Timeout,
					Jar:     c.client.Jar,
				}
				if c.client.Transport != nil {
					client.Transport = c.client.Transport
				}

				pageResp, err := client.Do(pageReq)
				if err != nil {
					fmt.Printf("Debug: 分页请求失败: %v\n", err)
					continue
				}

				// 解析分页结果（不允许再次分页）
				pageResults, err := c.parseSearchResultsInternal(pageResp, rule, keyword, false)
				pageResp.Body.Close()
				if err != nil {
					fmt.Printf("Debug: 解析分页结果失败: %v\n", err)
					continue
				}

				// 合并结果
				results = append(results, pageResults...)
				// fmt.Printf("Debug: 从分页 %s 解析到 %d 条结果\n", pageURL, len(pageResults))
			}
		}
	}

	return results, nil
}
