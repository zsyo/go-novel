package rules

import (
	"testing"
)

func TestRuleManager(t *testing.T) {
	// 获取规则管理器实例
	manager := GetRuleManager()

	// 测试加载规则
	rules, err := manager.LoadRules("main-rules.json")
	if err != nil {
		t.Fatalf("加载规则失败: %v", err)
	}

	// 检查是否加载了规则
	if len(rules) == 0 {
		t.Error("没有加载到任何规则")
	}

	// 测试获取特定ID的规则
	rule, err := manager.GetRuleById("main-rules.json", 1)
	if err != nil {
		t.Fatalf("获取规则失败: %v", err)
	}

	if rule == nil {
		t.Error("没有找到ID为1的规则")
	}

	// 测试获取所有规则
	allRules, err := manager.GetAllRules("main-rules.json")
	if err != nil {
		t.Fatalf("获取所有规则失败: %v", err)
	}

	if len(allRules) != len(rules) {
		t.Error("获取的所有规则数量不匹配")
	}

	// 测试获取可搜索规则
	searchableRules, err := manager.GetSearchableRules("main-rules.json")
	if err != nil {
		t.Fatalf("获取可搜索规则失败: %v", err)
	}

	if len(searchableRules) == 0 {
		t.Error("没有获取到可搜索规则")
	}
}
