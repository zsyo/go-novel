package core

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"go-novel/internal/model"

	"github.com/PuerkitoBio/goquery"
)

// parseBookInfo 解析书籍信息
func (c *Crawler) parseBookInfo(bookUrl string, rule *model.Rule) (*model.Book, error) {
	// 发起HTTP请求
	fmt.Printf("Debug: 开始解析书籍信息，URL: %s\n", bookUrl)

	// 为每次请求创建独立的HTTP客户端，避免共用超时设置
	client := c.NewHTTPClinet()

	resp, err := client.Get(bookUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP请求失败，状态码: %d", resp.StatusCode)
	}

	// 解析HTML文档
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	// 提取书籍信息
	book := &model.Book{
		URL:      bookUrl,
		SourceId: rule.ID,
	}

	// 显示页面标题
	title := doc.Find("title").Text()
	fmt.Printf("Debug: 页面标题: %s\n", title)

	// 根据规则提取信息
	if rule.Book.BookName != "" {
		book.BookName = c.extractText(doc.Selection, rule.Book.BookName)
		fmt.Printf("Debug: 根据规则提取到书名: '%s'\n", book.BookName)
	}
	if rule.Book.Author != "" {
		book.Author = c.extractText(doc.Selection, rule.Book.Author)
		fmt.Printf("Debug: 根据规则提取到作者: '%s'\n", book.Author)
	}
	if rule.Book.Intro != "" {
		book.Intro = c.extractText(doc.Selection, rule.Book.Intro)
		if len(book.Intro) > 50 {
			fmt.Printf("Debug: 根据规则提取到简介: %s...\n", book.Intro[:50])
		} else {
			fmt.Printf("Debug: 根据规则提取到简介: %s\n", book.Intro)
		}
	}
	if rule.Book.Category != "" {
		book.Category = c.extractText(doc.Selection, rule.Book.Category)
		fmt.Printf("Debug: 根据规则提取到分类: '%s'\n", book.Category)
	}
	if rule.Book.CoverUrl != "" {
		book.CoverUrl = c.extractAttr(doc.Selection, rule.Book.CoverUrl, "src")
		if book.CoverUrl == "" {
			// 尝试从content属性中获取
			book.CoverUrl = c.extractAttr(doc.Selection, rule.Book.CoverUrl, "content")
		}
		fmt.Printf("Debug: 根据规则提取到封面URL: '%s'\n", book.CoverUrl)
	}
	if rule.Book.LatestChapter != "" {
		book.LatestChapter = c.extractText(doc.Selection, rule.Book.LatestChapter)
		fmt.Printf("Debug: 根据规则提取到最新章节: '%s'\n", book.LatestChapter)
	}
	if rule.Book.LastUpdateTime != "" {
		book.LastUpdateTime = c.extractText(doc.Selection, rule.Book.LastUpdateTime)
		fmt.Printf("Debug: 根据规则提取到更新时间: '%s'\n", book.LastUpdateTime)
	}
	if rule.Book.Status != "" {
		book.Status = c.extractText(doc.Selection, rule.Book.Status)
		fmt.Printf("Debug: 根据规则提取到状态: '%s'\n", book.Status)
	}
	if rule.Book.WordCount != "" {
		book.WordCount = c.extractText(doc.Selection, rule.Book.WordCount)
		fmt.Printf("Debug: 根据规则提取到字数: '%s'\n", book.WordCount)
	}

	// 尝试从URL中提取书名和作者（作为备选方案）
	if book.BookName == "" {
		// 尝试从URL参数中获取
		parsedUrl, err := url.Parse(bookUrl)
		if err == nil {
			query := parsedUrl.Query()
			if bookName := query.Get("bookName"); bookName != "" {
				book.BookName = bookName
				fmt.Printf("Debug: 从URL参数中提取到书名: %s\n", book.BookName)
			}
		}
	}

	if book.Author == "" {
		// 尝试从URL参数中获取
		parsedUrl, err := url.Parse(bookUrl)
		if err == nil {
			query := parsedUrl.Query()
			if author := query.Get("author"); author != "" {
				book.Author = author
				fmt.Printf("Debug: 从URL参数中提取到作者: %s\n", book.Author)
			}
		}
	}

	// 如果仍然没有获取到书名和作者，尝试从页面中查找更准确的信息
	if book.BookName == "" || book.Author == "" {
		fmt.Printf("Debug: 尝试从页面中查找更准确的书名和作者信息\n")

		// 查找页面中的书名和作者信息
		// 尝试查找meta标签中的信息
		if book.BookName == "" {
			book.BookName = doc.Find("meta[property='og:novel:book_name']").AttrOr("content", "")
			fmt.Printf("Debug: 从meta标签提取到书名: '%s'\n", book.BookName)
		}

		if book.Author == "" {
			book.Author = doc.Find("meta[property='og:novel:author']").AttrOr("content", "")
			fmt.Printf("Debug: 从meta标签提取到作者: '%s'\n", book.Author)
		}

		// 如果还是没有找到，尝试查找页面中的其他元素
		if book.BookName == "" {
			// 尝试查找页面中的h1标签或其他标题元素
			h1Text := doc.Find("h1").First().Text()
			if h1Text != "" {
				book.BookName = strings.TrimSpace(h1Text)
				fmt.Printf("Debug: 从h1标签提取到书名: '%s'\n", book.BookName)
			}
		}

		// 尝试查找作者信息的其他位置
		if book.Author == "" {
			authorText := doc.Find("meta[property='og:novel:author']").AttrOr("content", "")
			if authorText == "" {
				// 尝试查找页面中的作者信息
				authorElements := []string{
					"meta[name='author']",
					".author",
					"#author",
					"[class*='author']",
				}

				for _, selector := range authorElements {
					authorText = doc.Find(selector).First().Text()
					if authorText != "" {
						break
					}
					// 也尝试从content属性获取
					authorText = doc.Find(selector).First().AttrOr("content", "")
					if authorText != "" {
						break
					}
				}
			}

			if authorText != "" {
				book.Author = strings.TrimSpace(authorText)
				fmt.Printf("Debug: 从页面元素提取到作者: '%s'\n", book.Author)
			}
		}
	}

	// 如果仍然没有获取到书名和作者，尝试从页面标题中提取（使用更智能的方法）
	if book.BookName == "" || book.Author == "" {
		fmt.Printf("Debug: 尝试从页面标题中智能提取书名和作者\n")
		// 获取页面标题
		title := doc.Find("title").Text()
		fmt.Printf("Debug: 页面标题: %s\n", title)

		// 尝试使用正则表达式提取书名和作者
		// 常见格式: "书名(作者),其他信息" 或 "书名 - 作者 - 其他信息"
		// 首先尝试匹配 "书名(作者)" 格式
		re := regexp.MustCompile(`^(.+?)\((.+?)\)`)
		matches := re.FindStringSubmatch(title)
		if len(matches) >= 3 && book.BookName == "" && book.Author == "" {
			book.BookName = strings.TrimSpace(matches[1])
			book.Author = strings.TrimSpace(matches[2])
			fmt.Printf("Debug: 从标题中提取到书名: '%s', 作者: '%s'\n", book.BookName, book.Author)
		} else if book.BookName == "" || book.Author == "" {
			// 尝试用" - "分割
			parts := strings.Split(title, " - ")
			if len(parts) >= 2 {
				if book.BookName == "" {
					book.BookName = strings.TrimSpace(parts[0])
					fmt.Printf("Debug: 从标题中提取到书名: '%s'\n", book.BookName)
				}

				if book.Author == "" && len(parts) >= 2 {
					book.Author = strings.TrimSpace(parts[1])
					// 移除可能的额外信息
					authorParts := strings.Split(book.Author, ",")
					if len(authorParts) > 0 {
						book.Author = strings.TrimSpace(authorParts[0])
					}
					fmt.Printf("Debug: 从标题中提取到作者: '%s'\n", book.Author)
				}
			}
		}
	}

	// 设置默认值
	if book.BookName == "" {
		book.BookName = "未知书名"
		fmt.Printf("Debug: 使用默认书名: %s\n", book.BookName)
	}

	if book.Author == "" {
		book.Author = "未知作者"
		fmt.Printf("Debug: 使用默认作者: %s\n", book.Author)
	}

	return book, nil
}

// parseToc 解析章节目录
func (c *Crawler) parseToc(bookUrl string, rule *model.Rule) ([]model.Chapter, error) {
	// 确定目录页URL
	tocUrl := bookUrl
	if rule.Toc.URL != "" {
		tocUrl = rule.Toc.URL
		// 处理URL模板中的占位符
		if strings.Contains(tocUrl, "%s") {
			// 从bookUrl中提取书籍ID
			bookId := extractBookIdFromUrl(bookUrl)
			fmt.Printf("Debug: bookUrl=%s, extracted bookId=%s\n", bookUrl, bookId)
			if bookId != "" {
				tocUrl = fmt.Sprintf(tocUrl, bookId)
			} else {
				// 如果无法提取书籍ID，返回错误
				return nil, fmt.Errorf("无法从URL %s 中提取书籍ID", bookUrl)
			}
		}
	}

	fmt.Printf("Debug: tocUrl=%s\n", tocUrl)

	// 检查URL是否有效
	if strings.Contains(tocUrl, "%s") {
		return nil, fmt.Errorf("URL %s 中仍然包含未替换的占位符", tocUrl)
	}

	// 发起HTTP请求
	// 为每次请求创建独立的HTTP客户端，避免共用超时设置
	client := c.NewHTTPClinet()

	resp, err := client.Get(tocUrl)
	if err != nil {
		return nil, fmt.Errorf("请求目录页失败: %v", err)
	}
	defer resp.Body.Close()

	// 解析HTML文档
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("解析HTML文档失败: %v", err)
	}

	// 提取章节链接
	var chapters []model.Chapter
	doc.Find(rule.Toc.Item).Each(func(i int, s *goquery.Selection) {
		title := s.Text()
		link, exists := s.Attr("href")
		if exists {
			// 如果链接是相对路径，则构建完整URL
			if !strings.HasPrefix(link, "http") {
				link = joinURL(rule.URL, link)
			}

			chapter := model.Chapter{
				Title: title,
				URL:   link,
				Order: i + 1,
			}
			chapters = append(chapters, chapter)
		}
	})

	return chapters, nil
}
