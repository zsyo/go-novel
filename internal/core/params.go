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

// BuildSearchGetParams 构建搜索GET参数
func BuildSearchGetParams(urlStr string, keyword string) string {
	// 对关键字进行URL编码
	encodedKeyword := url.QueryEscape(keyword)

	// 检查URL中是否直接包含%s（在查询参数之外）
	if strings.Contains(urlStr, "?") {
		parts := strings.SplitN(urlStr, "?", 2)
		baseURL := parts[0]
		queryStr := parts[1]

		// 处理查询字符串中的%s
		if strings.Contains(queryStr, "%s") {
			// 按&分割查询参数
			params := strings.Split(queryStr, "&")
			for i, param := range params {
				if strings.Contains(param, "%s") {
					// 按=分割键值对
					kv := strings.SplitN(param, "=", 2)
					if len(kv) == 2 && kv[1] == "%s" {
						// 如果值正好是%s，替换为编码后的关键字
						params[i] = kv[0] + "=" + encodedKeyword
					} else if len(kv) == 2 && strings.Contains(kv[1], "%s") {
						// 如果值包含%s，进行替换
						replacedValue := strings.ReplaceAll(kv[1], "%s", encodedKeyword)
						params[i] = kv[0] + "=" + replacedValue
					} else {
						// 其他情况直接替换
						params[i] = strings.ReplaceAll(param, "%s", encodedKeyword)
					}
				}
			}
			queryStr = strings.Join(params, "&")
			// 处理URL路径中的%s
			baseURL = strings.ReplaceAll(baseURL, "%s", encodedKeyword)
			return baseURL + "?" + queryStr
		}

		// 如果查询字符串中没有%s，但URL路径中有，单独处理路径部分
		if strings.Contains(baseURL, "%s") {
			baseURL = strings.ReplaceAll(baseURL, "%s", encodedKeyword)
			return baseURL + "?" + queryStr
		}

		return urlStr
	} else if strings.Contains(urlStr, "%s") {
		// 简单URL直接替换
		return strings.ReplaceAll(urlStr, "%s", encodedKeyword)
	}

	// 原始逻辑作为备选
	// 解析URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		// 如果解析失败，直接返回原始字符串
		return urlStr
	}

	// 获取查询参数
	query := parsedURL.Query()

	// 创建新的查询参数映射
	newQuery := make(url.Values)

	// 遍历查询参数，将%s替换为关键字
	for key, values := range query {
		newValues := make([]string, len(values))
		for i, value := range values {
			if value == "%s" {
				newValues[i] = encodedKeyword
			} else if strings.Contains(value, "%s") {
				newValues[i] = strings.ReplaceAll(value, "%s", encodedKeyword)
			} else {
				newValues[i] = value
			}
		}
		newQuery[key] = newValues
	}

	// 更新URL的查询参数
	parsedURL.RawQuery = newQuery.Encode()

	// 返回完整URL
	return parsedURL.String()
}
