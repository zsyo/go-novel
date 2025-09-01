package core

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// BuildSearchPostData 构建搜索POST数据
func BuildSearchPostData(dataStr string, keyword string) string {
	// 先打印原始数据进行调试
	fmt.Printf("Debug: 原始 POST 数据字符串: %s\n", dataStr)

	// 如果数据是空的，返回空字符串
	if dataStr == "" {
		return ""
	}

	// 尝试解析为JSON格式
	dataMap := make(map[string]interface{})
	if err := json.Unmarshal([]byte(dataStr), &dataMap); err == nil {
		// 成功解析为JSON格式
		formData := url.Values{}
		for key, value := range dataMap {
			// 处理值为字符串的情况
			if strValue, ok := value.(string); ok {
				if strValue == "%s" {
					formData.Add(key, keyword)
				} else if strings.Contains(strValue, "%s") {
					// 如果值中包含%s，进行替换
					replacedValue := strings.ReplaceAll(strValue, "%s", keyword)
					formData.Add(key, replacedValue)
				} else {
					formData.Add(key, strValue)
				}
			} else {
				// 其他类型直接转换为字符串
				formData.Add(key, fmt.Sprintf("%v", value))
			}
		}

		// 生成的表单数据
		encodedData := formData.Encode()
		fmt.Printf("Debug: 生成的 POST 表单数据: %s\n", encodedData)
		return encodedData
	}

	// 如果不是JSON格式，使用原来的处理方式
	// 处理带花括号的情况，去除外层花括号以更好地解析JSON
	processedDataStr := dataStr
	if strings.HasPrefix(dataStr, "{") && strings.HasSuffix(dataStr, "}") {
		processedDataStr = dataStr[1 : len(dataStr)-1]
	}

	// 处理的查询参数对
	keyValuePairs := strings.Split(processedDataStr, ",")
	formData := url.Values{}

	for _, pair := range keyValuePairs {
		// 去除空格
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		// 分割键值对
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) != 2 {
			continue
		}

		// 取出键和值，去除可能的空格和引号
		key := strings.Trim(strings.TrimSpace(parts[0]), `"'`)
		value := strings.Trim(strings.TrimSpace(parts[1]), `"'`)

		// 替换关键字
		if value == "%s" {
			formData.Add(key, keyword)
		} else if strings.Contains(value, "%s") {
			// 如果值中包含%s，进行替换
			replacedValue := strings.ReplaceAll(value, "%s", keyword)
			formData.Add(key, replacedValue)
		} else {
			formData.Add(key, value)
		}
	}

	// 生成的表单数据
	encodedData := formData.Encode()
	fmt.Printf("Debug: 生成的 POST 表单数据: %s\n", encodedData)

	return encodedData
}
