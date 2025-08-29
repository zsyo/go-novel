package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SanitizeFileName 清理文件名，移除非法字符
func SanitizeFileName(filename string) string {
	// 替换非法字符
	illegalChars := []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*"}
	for _, char := range illegalChars {
		filename = strings.ReplaceAll(filename, char, "_")
	}

	// 限制文件名长度
	if len(filename) > 255 {
		filename = filename[:255]
	}

	return filename
}

// CreateDownloadDir 创建下载目录
func CreateDownloadDir(basePath, bookName, author, ext string) (string, error) {
	// 构造目录名
	dirName := SanitizeFileName(fmt.Sprintf("%s (%s) %s", bookName, author, strings.ToUpper(ext)))
	dirPath := filepath.Join(basePath, dirName)

	// 创建目录
	err := os.MkdirAll(dirPath, 0755)
	if err != nil {
		return "", fmt.Errorf("创建目录失败: %v", err)
	}

	return dirPath, nil
}

// FileExists 检查文件是否存在
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// GetFileSize 获取文件大小
func GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}
