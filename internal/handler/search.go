package handler

import (
	"log"
	"net/http"
	"so-novel/internal/config"
	"so-novel/internal/core"
	"so-novel/internal/model"
	"so-novel/internal/rules"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// AggregatedSearch 聚合搜索处理函数
func AggregatedSearch(c *gin.Context) {
	keyword := c.Query("kw")
	if keyword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "搜索关键字不能为空"})
		return
	}

	// 获取配置
	cfg := config.GetConfig()

	// 执行聚合搜索
	results := performAggregatedSearch(keyword, cfg)

	// 返回结果
	c.JSON(http.StatusOK, gin.H{
		"data": results,
	})
}

// performAggregatedSearch 执行聚合搜索
func performAggregatedSearch(keyword string, cfg *config.Config) []model.SearchResult {
	// 获取规则管理器
	ruleManager := rules.GetRuleManager()

	// 获取可搜索的规则
	searchableRules, err := ruleManager.GetSearchableRules(cfg.Source.ActiveRules)
	if err != nil {
		log.Printf("加载规则失败: %v", err)
		return []model.SearchResult{}
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

			// 创建爬虫实例
			searchCfg := *cfg // 复制配置
			searchCfg.Source.SourceId = r.ID
			crawler := core.NewCrawler(&searchCfg)

			// 执行搜索并重试
			var searchResults []model.SearchResult
			var searchErr error
			maxRetries := 2 // 最多重试两次

			for retry := 0; retry <= maxRetries; retry++ {
				searchResults, searchErr = crawler.Search(keyword)
				if searchErr == nil {
					// 搜索成功，退出重试循环
					break
				}

				// 检查是否是超时错误，如果是则不重试
				if isTimeoutError(searchErr) {
					log.Printf("搜索源 %s (%d) 超时错误: %v, 跳过重试", r.Name, r.ID, searchErr)
					break
				}

				// 记录错误和重试信息
				log.Printf("搜索源 %s (%d) 异常: %v, 重试: %d/%d", r.Name, r.ID, searchErr, retry+1, maxRetries)

				// 如果还有重试机会，等待一段时间后重试
				if retry < maxRetries {
					waitTime := time.Duration(2*(retry+1)) * time.Second
					time.Sleep(waitTime)
				}
			}

			// 如果搜索仍然失败，记录并返回
			if searchErr != nil {
				log.Printf("搜索源 %s (%d) 失败: %v", r.Name, r.ID, searchErr)
				return
			}

			if len(searchResults) > 0 {
				log.Printf("书源 %d (%s) 搜索到 %d 条记录", r.ID, r.Name, len(searchResults))

				// 安全地添加到结果列表
				resultsMutex.Lock()
				results = append(results, searchResults...)
				resultsMutex.Unlock()
			} else {
				// 即使没有结果也记录一下，方便调试
				log.Printf("书源 %d (%s) 搜索到 0 条记录", r.ID, r.Name)
			}
		}(rule)
	}

	// 等待所有搜索完成
	wg.Wait()

	// 排序结果
	sortedResults := sortSearchResults(results, keyword)

	// 记录最终结果数量
	log.Printf("聚合搜索完成，共找到 %d 条结果", len(sortedResults))

	return sortedResults
}

// isTimeoutError 检查是否是超时错误
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	// 检查是否包含超时相关的错误信息
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "timed out") ||
		strings.Contains(errStr, "context deadline exceeded") ||
		strings.Contains(errStr, "Client.Timeout exceeded")
}

// sortSearchResults 根据关键字对搜索结果进行排序
func sortSearchResults(results []model.SearchResult, keyword string) []model.SearchResult {
	if len(results) == 0 {
		return results
	}

	// 计算书名和作者的相似度权重，判断搜索类型
	bookNameScore := 0.0
	authorScore := 0.0

	for _, result := range results {
		bookNameScore += calculateSimilarity(keyword, result.BookName)
		authorScore += calculateSimilarity(keyword, result.Author)
	}

	// 判断是书名搜索还是作者搜索
	isAuthorSearch := bookNameScore < authorScore

	// 为每个结果计算相似度分数
	type scoredResult struct {
		result model.SearchResult
		score  float64
	}

	scoredResults := make([]scoredResult, len(results))
	for i, result := range results {
		var score float64
		if isAuthorSearch {
			score = calculateSimilarity(keyword, result.Author)
		} else {
			score = calculateSimilarity(keyword, result.BookName)
		}
		scoredResults[i] = scoredResult{result: result, score: score}
	}

	// 按相似度排序
	sort.Slice(scoredResults, func(i, j int) bool {
		// 首先按相似度降序排列
		if scoredResults[i].score != scoredResults[j].score {
			return scoredResults[i].score > scoredResults[j].score
		}

		// 相似度相同时，按作者或书名排序
		if isAuthorSearch {
			return scoredResults[i].result.BookName < scoredResults[j].result.BookName
		} else {
			return scoredResults[i].result.Author < scoredResults[j].result.Author
		}
	})

	// 过滤低相似度结果，但保留更多结果
	var filteredResults []model.SearchResult
	for _, scored := range scoredResults {
		// 只保留相似度大于0.1的结果（原来是0.3，现在降低阈值）
		if scored.score > 0.1 {
			filteredResults = append(filteredResults, scored.result)
		}
	}

	// 如果过滤后没有结果，返回原始结果中相似度大于0的结果
	if len(filteredResults) == 0 {
		for _, scored := range scoredResults {
			if scored.score > 0 {
				filteredResults = append(filteredResults, scored.result)
			}
		}
	}

	// 限制返回结果数量，避免过多结果
	if len(filteredResults) > 100 {
		filteredResults = filteredResults[:100]
	}

	return filteredResults
}

// calculateSimilarity 计算两个字符串的相似度
func calculateSimilarity(s1, s2 string) float64 {
	if s1 == "" || s2 == "" {
		return 0.0
	}

	// 转换为小写进行比较
	s1 = strings.ToLower(s1)
	s2 = strings.ToLower(s2)

	// 完全匹配
	if s1 == s2 {
		return 1.0
	}

	// 计算相似度（简化版本）
	// 使用简单的包含关系和长度比例计算
	len1 := len(s1)
	len2 := len(s2)

	// 如果一个包含另一个
	if strings.Contains(s1, s2) || strings.Contains(s2, s1) {
		minLen := len1
		if len2 < minLen {
			minLen = len2
		}
		return float64(minLen) / float64(max(len1, len2))
	}

	// 计算公共子串长度
	commonLen := longestCommonSubstring(s1, s2)
	return float64(commonLen) / float64(max(len1, len2))
}

// longestCommonSubstring 计算最长公共子串长度
func longestCommonSubstring(s1, s2 string) int {
	len1 := len(s1)
	len2 := len(s2)

	if len1 == 0 || len2 == 0 {
		return 0
	}

	// 创建二维数组存储长度
	dp := make([][]int, len1+1)
	for i := range dp {
		dp[i] = make([]int, len2+1)
	}

	maxLen := 0

	// 动态规划计算最长公共子串
	for i := 1; i <= len1; i++ {
		for j := 1; j <= len2; j++ {
			if s1[i-1] == s2[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
				if dp[i][j] > maxLen {
					maxLen = dp[i][j]
				}
			} else {
				dp[i][j] = 0
			}
		}
	}

	return maxLen
}

// max 返回两个整数中的较大值
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
