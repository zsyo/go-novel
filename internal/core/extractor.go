package core

import (
	"fmt"
	"regexp"
	"strings"

	"so-novel/internal/util"

	"github.com/PuerkitoBio/goquery"
	"github.com/antchfx/htmlquery"
)

// extractText 从选择器中提取文本，支持JavaScript调用和XPath
func (c *Crawler) extractText(s *goquery.Selection, selector string) string {
	if selector == "" {
		return ""
	}

	// 处理JavaScript调用
	if strings.Contains(selector, "@js:") {
		parts := strings.Split(selector, "@js:")
		selector = parts[0]

		// 首先提取原始文本
		text := c.extractTextSimple(s, selector)
		// fmt.Printf("Debug: 使用选择器 '%s' 提取到原始文本: '%s'\n", selector, text)

		// 如果有JavaScript代码
		if len(parts) > 1 {
			jsCode := parts[1]
			// fmt.Printf("Debug: 执行JS代码: '%s' 对文本进行处理\n", jsCode)

			// 使用完整的JavaScript引擎处理
			result, err := util.CallJs(jsCode, text)
			if err != nil {
				fmt.Printf("Debug: JavaScript执行出错: %v\n", err)
				return text
			}
			text = result
			// fmt.Printf("Debug: JavaScript执行结果: '%s'\n", text)
		}

		return text
	}

	// 检查是否是XPath
	if strings.HasPrefix(selector, "/") || strings.HasPrefix(selector, "//") || strings.HasPrefix(selector, "(") {
		// 使用XPath提取
		return c.extractTextWithXPath(s, selector)
	}

	// 处理meta标签的特殊情况
	if strings.HasPrefix(selector, "meta[") {
		fmt.Printf("Debug: 检测到meta标签选择器: '%s'\n", selector)
		// 如果是meta标签，尝试提取content属性
		content := c.extractAttr(s, selector, "content")
		fmt.Printf("Debug: 仞meta标签提取到content属性值: '%s'\n", content)
		return content
	}

	// 默认使用CSS选择器
	result := strings.TrimSpace(s.Find(selector).Text())
	// fmt.Printf("Debug: 使用CSS选择器'%s'提取到文本: '%s'\n", selector, result)
	return result
}

// extractTextSimple 使用CSS选择器简单提取文本
func (c *Crawler) extractTextSimple(s *goquery.Selection, selector string) string {
	if selector == "" {
		return ""
	}

	// 默认使用CSS选择器
	return strings.TrimSpace(s.Find(selector).Text())
}

// extractTextWithXPath 使用XPath提取文本
func (c *Crawler) extractTextWithXPath(s *goquery.Selection, xpathExpr string) string {
	// 将goquery.Selection转换为HTML字符串
	htmlStr, err := s.Html()
	if err != nil {
		return ""
	}

	// 解析HTML
	doc, err := htmlquery.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return ""
	}

	// 使用XPath表达式查找元素
	node, err := htmlquery.Query(doc, xpathExpr)
	if err != nil || node == nil {
		return ""
	}

	// 返回文本内容
	return strings.TrimSpace(htmlquery.InnerText(node))
}

// extractAttr 从选择器中提取属性，支持JavaScript调用和XPath
func (c *Crawler) extractAttr(s *goquery.Selection, selector, attr string) string {
	if selector == "" {
		return ""
	}

	// 处理JavaScript调用
	if strings.Contains(selector, "@js:") {
		parts := strings.Split(selector, "@js:")
		selector = parts[0]

		// 首先提取属性值
		var value string
		if strings.HasPrefix(selector, "/") || strings.HasPrefix(selector, "//") || strings.HasPrefix(selector, "(") {
			// 使用XPath提取
			value = c.extractAttrWithXPath(s, selector, attr)
		} else {
			// 默认使用CSS选择器
			value, _ = s.Find(selector).First().Attr(attr)
		}

		// fmt.Printf("Debug: 使用选择器 '%s' 提取到属性 '%s' 的值: '%s'\n", selector, attr, value)

		// 如果有JavaScript代码
		if len(parts) > 1 && value != "" {
			jsCode := parts[1]
			// fmt.Printf("Debug: 执行JS代码: '%s' 对属性值进行处理\n", jsCode)

			// 使用完整的JavaScript引擎处理
			result, err := util.CallJs(jsCode, value)
			if err != nil {
				fmt.Printf("Debug: JavaScript执行出错: %v\n", err)
				return value
			}
			value = result
			// fmt.Printf("Debug: JavaScript执行结果: '%s'\n", value)
		}

		return value
	}

	// 检查是否是XPath
	if strings.HasPrefix(selector, "/") || strings.HasPrefix(selector, "//") || strings.HasPrefix(selector, "(") {
		// 使用XPath提取
		return c.extractAttrWithXPath(s, selector, attr)
	}

	// 处理meta标签的特殊情况
	if strings.HasPrefix(selector, "meta[") {
		// fmt.Printf("Debug: 检测到meta标签选择器: '%s', 要提取的属性: '%s'\n", selector, attr)
		// 使用根元素来确保可以选中全局meta标签
		root := s
		// 如果s不是文档节点，发现其根节点
		if s.Parent().Length() > 0 {
			root = s.Parent()
			for root.Parent().Length() > 0 {
				root = root.Parent()
			}
		}

		// 使用根节点选中meta标签
		meta := root.Find(selector)
		if meta.Length() > 0 {
			value, exists := meta.First().Attr(attr)
			if exists {
				// fmt.Printf("Debug: 从 meta标签提取到%s属性值: '%s'\n", attr, value)
				return value
			} else if attr == "href" {
				// 如果要获取href属性，但meta标签没有，尝试获取content属性
				value, exists = meta.First().Attr("content")
				if exists {
					// fmt.Printf("Debug: meta标签没有href属性，使用content属性作为替代: '%s'\n", value)
					return value
				}
			}
		}
		return ""
	}

	// 默认使用CSS选择器
	value, _ := s.Find(selector).First().Attr(attr)
	return value
}

// extractAttrWithXPath 使用XPath提取属性
func (c *Crawler) extractAttrWithXPath(s *goquery.Selection, xpathExpr, attr string) string {
	// 将goquery.Selection转换为HTML字符串
	htmlStr, err := s.Html()
	if err != nil {
		return ""
	}

	// 解析HTML
	doc, err := htmlquery.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return ""
	}

	// 使用XPath表达式查找元素
	node, err := htmlquery.Query(doc, xpathExpr)
	if err != nil || node == nil {
		return ""
	}

	// 获取属性值
	for _, a := range node.Attr {
		if a.Key == attr {
			return a.Val
		}
	}

	return ""
}

// extractAbsAttr 从选择器中提取属性，并转换为绝对URL
func (c *Crawler) extractAbsAttr(s *goquery.Selection, selector, attr string, baseURL string) string {
	relativeURL := c.extractAttr(s, selector, attr)
	if relativeURL == "" {
		return ""
	}

	// 将相对URL转换为绝对URL
	absoluteURL := joinURL(baseURL, relativeURL)
	return absoluteURL
}

// extractBookIdFromUrl 从书籍URL中提取书籍ID
func extractBookIdFromUrl(bookUrl string) string {
	// 使用多种正则表达式尝试提取书籍ID

	// 模式1: /book/12345.html
	re1 := regexp.MustCompile(`/book/(\d+)\.html`)
	matches := re1.FindStringSubmatch(bookUrl)
	if len(matches) > 1 {
		fmt.Printf("Debug: 匹配模式1, 提取到ID=%s\n", matches[1])
		return matches[1]
	}

	// 模式2: /book/12345/ 或 /book/12345
	re2 := regexp.MustCompile(`/book/(\d+)/?`)
	matches = re2.FindStringSubmatch(bookUrl)
	if len(matches) > 1 {
		fmt.Printf("Debug: 匹配模式2, 提取到ID=%s\n", matches[1])
		return matches[1]
	}

	// 模式3: ?id=12345 或 &id=12345
	re3 := regexp.MustCompile(`[?&]id=(\d+)`)
	matches = re3.FindStringSubmatch(bookUrl)
	if len(matches) > 1 {
		fmt.Printf("Debug: 匹配模式3, 提取到ID=%s\n", matches[1])
		return matches[1]
	}

	// 模式4: /book/abc12345.html (字母+数字)
	re4 := regexp.MustCompile(`/book/[a-zA-Z]*(\d+)\.html`)
	matches = re4.FindStringSubmatch(bookUrl)
	if len(matches) > 1 {
		fmt.Printf("Debug: 匹配模式4, 提取到ID=%s\n", matches[1])
		return matches[1]
	}

	// 模式5: /book/12345 (结尾)
	re5 := regexp.MustCompile(`/book/(\d+)$`)
	matches = re5.FindStringSubmatch(bookUrl)
	if len(matches) > 1 {
		fmt.Printf("Debug: 匹配模式5, 提取到ID=%s\n", matches[1])
		return matches[1]
	}

	fmt.Printf("Debug: 无法从URL %s 中提取书籍ID\n", bookUrl)
	return ""
}
