package core

import (
	"so-novel/internal/util"
	"testing"
)

func TestJavaScriptProcessing(t *testing.T) {
	// 测试JavaScript处理
	// 测试replace方法
	result, err := util.CallJs("r=r.replace('作者：', '')", "作者：忆小邪")
	if err != nil {
		t.Errorf("JavaScript replace执行失败: %v", err)
	}
	if result != "忆小邪" {
		t.Errorf("JavaScript replace结果不正确，期望: 忆小邪, 实际: %s", result)
	}

	// 测试replaceAll方法
	result, err = util.CallJs("r=r.replaceAll('最新章节', '')", "最新章节第123章")
	if err != nil {
		t.Errorf("JavaScript replaceAll执行失败: %v", err)
	}
	if result != "第123章" {
		t.Errorf("JavaScript replaceAll结果不正确，期望: 第123章, 实际: %s", result)
	}

	// 测试字符串连接
	result, err = util.CallJs("r='https://www.example.com/'+r", "/book/123.html")
	if err != nil {
		t.Errorf("JavaScript字符串连接执行失败: %v", err)
	}
	if result != "https://www.example.com//book/123.html" {
		t.Errorf("JavaScript字符串连接结果不正确，期望: https://www.example.com//book/123.html, 实际: %s", result)
	}
}

func TestPaginationLogic(t *testing.T) {
	// 这个测试需要网络请求，暂时跳过
	t.Skip("跳过需要网络请求的分页测试")
}

func TestBuildSearchGetParams(t *testing.T) {
	// 测试GET参数构建函数
	keyword := "仙逆"
	url := "https://www.example.com/search.html?searchkey=%s"

	result := BuildSearchGetParams(url, keyword)
	// 期望的结果应该是URL编码后的关键字
	expected := "https://www.example.com/search.html?searchkey=%E4%BB%99%E9%80%86"

	if result != expected {
		t.Errorf("BuildSearchGetParams结果不正确，期望: %s, 实际: %s", expected, result)
	}

	// 测试路径中的%s会被处理
	url2 := "https://www.example.com/search/%s.html?searchkey=test"
	result2 := BuildSearchGetParams(url2, keyword)
	// 路径中的%s应该被替换
	expected2 := "https://www.example.com/search/%E4%BB%99%E9%80%86.html?searchkey=test"

	if result2 != expected2 {
		t.Errorf("BuildSearchGetParams结果不正确，期望: %s, 实际: %s", expected2, result2)
	}
}

func TestBuildSearchPostData(t *testing.T) {
	// 测试POST数据构建函数
	keyword := "仙逆"

	// 测试JSON格式数据
	dataStr := `{"searchkey": "%s", "searchtype": "all"}`
	result := BuildSearchPostData(dataStr, keyword)
	// 期望的结果应该是URL编码后的表单数据
	expected := "searchkey=%E4%BB%99%E9%80%86&searchtype=all"

	if result != expected {
		t.Errorf("BuildSearchPostData结果不正确，期望: %s, 实际: %s", expected, result)
	}

	// 测试另一种JSON格式
	dataStr2 := `{"searchkey": "%s", "page": 1}`
	result2 := BuildSearchPostData(dataStr2, keyword)
	expected2 := "page=1&searchkey=%E4%BB%99%E9%80%86"

	if result2 != expected2 {
		t.Errorf("BuildSearchPostData结果不正确，期望: %s, 实际: %s", expected2, result2)
	}
}
