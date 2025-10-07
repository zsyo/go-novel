package core

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"go-novel/internal/model"
	"go-novel/internal/util"

	"github.com/PuerkitoBio/goquery"
	"github.com/bmaupin/go-epub"
)

// saveBook 保存书籍
func (c *Crawler) saveBook(ctx context.Context, book *model.Book, chapters []model.Chapter) error {
	// 获取配置
	cfg := c.config

	// 创建下载目录
	downloadDir, err := util.CreateDownloadDir(cfg.Download.DownloadPath, book.BookName, book.Author, cfg.Download.ExtName)
	if err != nil {
		return fmt.Errorf("创建下载目录失败: %w", err)
	}

	// 保存章节内容
	err = c.saveChapters(downloadDir, chapters, cfg.Download.ExtName)
	if err != nil {
		return fmt.Errorf("保存章节失败: %w", err)
	}

	// 根据配置合并文件
	extName := strings.ToLower(cfg.Download.ExtName)
	switch extName {
	case "txt":
		err = c.mergeToTxt(downloadDir, book, cfg.Download.DownloadPath)
		if err != nil {
			return fmt.Errorf("TXT格式合并失败: %w", err)
		}
	case "epub":
		// EPUB合并实现
		err = c.mergeToEpub(ctx, downloadDir, book, cfg.Download.DownloadPath)
		if err != nil {
			return fmt.Errorf("EPUB格式合并失败: %w", err)
		}
	default:
		return fmt.Errorf("不支持的文件格式: %s", extName)
	}

	// 如果不保留章节缓存，删除章节目录
	if cfg.Download.PreserveChapterCache == 0 {
		os.RemoveAll(downloadDir)
	}

	return nil
}

// saveChapters 保存章节内容
func (c *Crawler) saveChapters(downloadDir string, chapters []model.Chapter, extName string) error {
	// 计算数字位数，用于补零
	digitCount := len(strconv.Itoa(len(chapters)))

	// 保存每个章节
	for _, chapter := range chapters {
		// 生成文件名
		orderStr := strconv.Itoa(chapter.Order)
		if len(orderStr) < digitCount {
			orderStr = strings.Repeat("0", digitCount-len(orderStr)) + orderStr
		}

		var filename string
		switch extName {
		case "txt":
			sanitizedTitle := util.SanitizeFileName(chapter.Title)
			filename = fmt.Sprintf("%s_%s.txt", orderStr, sanitizedTitle)
		case "epub":
			sanitizedTitle := util.SanitizeFileName(chapter.Title)
			filename = fmt.Sprintf("%s_%s.html", orderStr, sanitizedTitle)
		default:
			filename = fmt.Sprintf("%s_.html", orderStr)
		}

		// 构建文件路径
		filePath := path.Join(downloadDir, filename)

		// 写入文件
		err := os.WriteFile(filePath, []byte(chapter.Content), 0644)
		if err != nil {
			fmt.Printf("保存章节失败 %s: %v\n", chapter.Title, err)
			continue
		}
	}

	return nil
}

// mergeToTxt 合并为TXT文件
func (c *Crawler) mergeToTxt(chapterDir string, book *model.Book, downloadPath string) error {
	// 确保书名和作者不为空
	if book.BookName == "" {
		book.BookName = "未知书名"
	}
	if book.Author == "" {
		book.Author = "未知作者"
	}

	// 生成目标文件路径
	filename := util.SanitizeFileName(fmt.Sprintf("%s(%s).txt", book.BookName, book.Author))
	targetPath := path.Join(downloadPath, filename)

	// 创建目标文件
	targetFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer targetFile.Close()

	// 写入书籍信息
	bookInfo := fmt.Sprintf("书名：%s\n作者：%s\n简介：%s\n\n",
		book.BookName, book.Author, book.Intro)
	targetFile.WriteString(bookInfo)

	// 读取章节文件并合并
	chapterFiles, err := os.ReadDir(chapterDir)
	if err != nil {
		return fmt.Errorf("读取章节目录失败: %w", err)
	}

	// 按文件名排序
	sort.Slice(chapterFiles, func(i, j int) bool {
		return chapterFiles[i].Name() < chapterFiles[j].Name()
	})

	// 合并章节内容
	for _, file := range chapterFiles {
		if file.IsDir() {
			continue
		}

		// 读取章节内容
		content, err := os.ReadFile(path.Join(chapterDir, file.Name()))
		if err != nil {
			fmt.Printf("读取章节文件失败 %s: %v\n", file.Name(), err)
			continue
		}

		// 处理章节内容，改善排版
		txtContent := string(content)

		// 从文件名提取章节标题
		title := file.Name()
		title = strings.TrimSuffix(title, path.Ext(title))
		if underscoreIndex := strings.Index(title, "_"); underscoreIndex != -1 {
			title = title[underscoreIndex+1:]
		}

		// 清理HTML标签
		txtContent = c.cleanHtmlTags(txtContent)

		// 写入章节标题
		chapterTitle := fmt.Sprintf("\n\n第%s章 %s\n\n",
			strings.Split(file.Name(), "_")[0], title)
		targetFile.WriteString(chapterTitle)

		// 段落处理
		paragraphs := strings.Split(txtContent, "\n")
		for _, paragraph := range paragraphs {
			paragraph = strings.TrimSpace(paragraph)
			if paragraph != "" {
				// 添加段落缩进（全角空格）
				indentedParagraph := "　　" + paragraph + "\n\n"
				targetFile.WriteString(indentedParagraph)
			}
		}
	}

	fmt.Printf("书籍已保存为TXT文件: %s\n", targetPath)
	return nil
}

// cleanHtmlTags 清理HTML标签
func (c *Crawler) cleanHtmlTags(html string) string {
	// 移除所有HTML标签
	re := regexp.MustCompile("<[^>]*>")
	text := re.ReplaceAllString(html, "")

	// 替换HTML实体
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&quot;", "\"")

	// 清理多余空白
	text = strings.TrimSpace(text)

	return text
}

// mergeToEpub 合并为EPUB文件
func (c *Crawler) mergeToEpub(ctx context.Context, chapterDir string, book *model.Book, downloadPath string) error {
	// 检查context是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 确保书名和作者不为空
	fmt.Printf("Debug: 开始生成EPUB文件，原始书名: '%s', 原始作者: '%s'\n", book.BookName, book.Author)

	if book.BookName == "" {
		book.BookName = "未知书名"
	}
	if book.Author == "" {
		book.Author = "未知作者"
	}

	fmt.Printf("Debug: 处理后的书名: '%s', 处理后的作者: '%s'\n", book.BookName, book.Author)

	// 生成目标文件路径
	filename := util.SanitizeFileName(fmt.Sprintf("%s(%s).epub", book.BookName, book.Author))
	targetPath := path.Join(downloadPath, filename)

	fmt.Printf("Debug: EPUB文件路径: %s\n", targetPath)

	// 用于存储临时文件路径，以便在EPUB生成后清理
	var tempFiles []string

	// 创建连接清理临时文件的defer函数，确保在函数结束时执行
	defer func() {
		// 清理所有临时文件
		for _, tempFile := range tempFiles {
			os.Remove(tempFile)
			fmt.Printf("Debug: 清理临时文件: %s\n", tempFile)
		}
	}()

	// 创建EPUB
	epub := epub.NewEpub(book.BookName)

	// 设置EPUB元数据
	epub.SetAuthor(book.Author)
	epub.SetDescription(book.Intro)

	// 添加封面图片
	if book.CoverUrl != "" {
		fmt.Printf("Debug: 尝试添加封面图片: %s\n", book.CoverUrl)

		// 下载并添加封面图片，但不添加为内容页
		tempCoverFile := c.downloadAndAddCoverImage(ctx, book.CoverUrl, epub)
		if tempCoverFile != "" {
			// 记录临时文件路径，以便后续清理
			tempFiles = append(tempFiles, tempCoverFile)
		}
	} else {
		fmt.Printf("Debug: 没有封面图片URL\n")
	}

	// 移除标题页创建代码

	// 读取章节文件
	chapterFiles, err := os.ReadDir(chapterDir)
	if err != nil {
		return fmt.Errorf("读取章节目录失败: %w", err)
	}

	// 按文件名排序
	sort.Slice(chapterFiles, func(i, j int) bool {
		return chapterFiles[i].Name() < chapterFiles[j].Name()
	})

	// CSS样式文件
	// 创建临时CSS文件
	tempCssFile := path.Join(os.TempDir(), "style.css")
	cssContent := `body {
    font-family: "PingFang SC", "Microsoft YaHei", SimSun, serif;
    text-align: justify;
    line-height: 1.8;
    margin: 1.2em 1.6em;
    font-size: 1em;
    font-weight: normal;
    color: #333333;
}
h1 {
    text-align: center;
    font-weight: bold;
    font-size: 1.4em;
    margin: 2em 0 1.5em 0;
    padding-bottom: 0.5em;
    border-bottom: 1px solid #dddddd;
    color: #333333;
}
p {
    margin: 0.8em 0;
    text-indent: 2em;
    line-height: 1.8;
    font-weight: normal;
    letter-spacing: 0.05em;
}
.chapter {
    page-break-before: always;
}
.chapter:first-child {
    page-break-before: avoid;
}
`
	err = os.WriteFile(tempCssFile, []byte(cssContent), 0644)
	if err != nil {
		fmt.Printf("创建CSS文件失败: %v\n", err)
		return err
	}

	// 添加CSS文件
	cssPath, err := epub.AddCSS(tempCssFile, "style.css")
	if err != nil {
		fmt.Printf("添加CSS样式失败: %v\n", err)
		return err
	}

	// 记录临时CSS文件路径，以便后续清理
	tempFiles = append(tempFiles, tempCssFile)

	// 添加章节到EPUB
	for _, file := range chapterFiles {
		if file.IsDir() {
			continue
		}

		// 检查context是否已取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 读取章节内容
		content, err := os.ReadFile(path.Join(chapterDir, file.Name()))
		if err != nil {
			fmt.Printf("读取章节文件失败 %s: %v\n", file.Name(), err)
			continue
		}

		// 从文件名提取章节标题
		title := file.Name()
		title = strings.TrimSuffix(title, path.Ext(title))
		if underscoreIndex := strings.Index(title, "_"); underscoreIndex != -1 {
			title = title[underscoreIndex+1:]
		}

		// 处理HTML内容，正确处理段落和格式
		chapterHTML := string(content)

		// 1. 解析原始的HTML内容
		reader := strings.NewReader(chapterHTML)
		doc, err := goquery.NewDocumentFromReader(reader)
		if err != nil {
			fmt.Printf("解析HTML内容失败: %v\n", err)
			continue
		}

		// 2. 提取章节文本内容并处理段落
		// 首先尝试从文档中直接提取段落
		var paragraphs []string

		// 尝试查找文档中的段落标签
		pTags := doc.Find("p")
		if pTags.Length() > 0 {
			// 如果文档中有段落标签，直接提取它们
			pTags.Each(func(i int, s *goquery.Selection) {
				text := strings.TrimSpace(s.Text())
				if text != "" {
					paragraphs = append(paragraphs, text)
				}
			})
		} else {
			// 如果没有段落标签，尝试使用processParagraphs函数处理
			textContent := doc.Text()
			paragraphs = processParagraphs(textContent)
		}

		// 3. 生成符合XHTML标准的内容
		xhtmlContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">
<html xmlns="http://www.w3.org/1999/xhtml" xml:lang="zh-CN">
<head>
  <title>%s</title>
  <link rel="stylesheet" href="%s" type="text/css" />
  <meta http-equiv="Content-Type" content="application/xhtml+xml; charset=utf-8" />
</head>
<body>
  <div class="chapter">
    <h1>%s</h1>
    <div class="content">
`, title, cssPath, title)

		// 4. 处理段落文本，确保每个段落都正确包装在<p>标签中
		for _, para := range paragraphs {
			if para != "" {
				// 对HTML实体进行编码以避免标记问题
				para = strings.ReplaceAll(para, "&", "&amp;")
				para = strings.ReplaceAll(para, "<", "&lt;")
				para = strings.ReplaceAll(para, ">", "&gt;")
				para = strings.ReplaceAll(para, "\"", "&quot;")
				xhtmlContent += fmt.Sprintf("    <p>%s</p>\n", para)
			}
		}

		xhtmlContent += "    </div>\n  </div>\n</body>\n</html>"

		// 添加章节到EPUB
		_, err = epub.AddSection(xhtmlContent, title, "", "")
		if err != nil {
			fmt.Printf("添加章节到EPUB失败 %s: %v\n", title, err)
			continue
		}
	}

	// 保存EPUB文件
	err = epub.Write(targetPath)
	if err != nil {
		return fmt.Errorf("保存EPUB文件失败: %w", err)
	}

	fmt.Printf("书籍已保存为EPUB文件: %s\n", targetPath)
	return nil
}

// processParagraphs 处理文本内容，智能分段
func processParagraphs(text string) []string {
	var paragraphs []string

	// 按行分割文本
	lines := strings.Split(text, "\n")

	// 使用空行作为段落分隔符
	var currentParagraph string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 如果是空行，表示段落结束
		if line == "" {
			if currentParagraph != "" {
				paragraphs = append(paragraphs, currentParagraph)
				currentParagraph = ""
			}
			continue
		}

		// 开始新段落或继续当前段落
		if currentParagraph == "" {
			currentParagraph = line
		} else {
			// 如果前一段落以标点符号结束，开始新段落
			lastChar := currentParagraph[len(currentParagraph)-1:]
			if strings.Contains("。！？、”，；：……", lastChar) {
				paragraphs = append(paragraphs, currentParagraph)
				currentParagraph = line
			} else {
				// 否则合并到当前段落
				currentParagraph += " " + line
			}
		}
	}

	// 添加最后一个段落
	if currentParagraph != "" {
		paragraphs = append(paragraphs, currentParagraph)
	}

	// 如果没有提取到段落，则尝试以行为单位
	if len(paragraphs) == 0 {
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				paragraphs = append(paragraphs, line)
			}
		}
	}

	return paragraphs
}

// downloadAndAddCoverImage 下载并添加封面图片，但不添加为内容页
func (c *Crawler) downloadAndAddCoverImage(ctx context.Context, coverUrl string, epub *epub.Epub) string {
	// 下载封面图片（带重试机制）
	coverResp, err := c.getWithRetry(ctx, coverUrl)
	if err != nil || coverResp.StatusCode != 200 {
		fmt.Printf("Debug: 下载封面图片失败: %v\n", err)
		return ""
	}
	defer coverResp.Body.Close()

	// 读取图片数据
	coverData, err := io.ReadAll(coverResp.Body)
	if err != nil {
		fmt.Printf("Debug: 读取封面图片数据失败: %v\n", err)
		return ""
	}

	// 保存图片到临时文件
	tempCoverFile := path.Join(os.TempDir(), "cover.jpg")
	err = os.WriteFile(tempCoverFile, coverData, 0644)
	if err != nil {
		fmt.Printf("Debug: 保存图片到临时文件失败: %v\n", err)
		return ""
	}

	// 添加封面图片
	coverImgPath, err := epub.AddImage(tempCoverFile, "cover.jpg")
	if err != nil {
		fmt.Printf("Debug: 添加封面图片失败: %v\n", err)
		os.Remove(tempCoverFile) // 清理临时文件
		return ""
	}

	// 设置封面，但不添加为内容页
	epub.SetCover(coverImgPath, "")
	fmt.Printf("Debug: 成功添加封面图片\n")

	return tempCoverFile
}
