package rules

import (
	"encoding/json"
	"os"
	"path/filepath"
	"so-novel/internal/model"
	"sync"
)

type RuleManager struct {
	rules map[string][]model.Rule
	mutex sync.RWMutex
}

var (
	manager *RuleManager
	once    sync.Once
)

func GetRuleManager() *RuleManager {
	once.Do(func() {
		manager = &RuleManager{
			rules: make(map[string][]model.Rule),
		}
	})
	return manager
}

func (rm *RuleManager) LoadRules(filename string) ([]model.Rule, error) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	// 如果已经加载过，直接返回
	if rules, exists := rm.rules[filename]; exists {
		return rules, nil
	}

	// 构建文件路径
	pathsToTry := []string{
		filename,
		filepath.Join("configs", "rules", filename),
		filepath.Join("..", "configs", "rules", filename),
	}

	var data []byte
	var err error
	var path string

	// 尝试不同的路径
	for _, p := range pathsToTry {
		if _, statErr := os.Stat(p); statErr == nil {
			path = p
			data, err = os.ReadFile(path)
			if err == nil {
				break
			}
		}
	}

	// 如果所有路径都失败了，返回最后一个错误
	if err != nil {
		return nil, err
	}

	// 解析JSON
	var rules []model.Rule
	if err := json.Unmarshal(data, &rules); err != nil {
		return nil, err
	}

	// 缓存规则
	rm.rules[filename] = rules

	return rules, nil
}

func (rm *RuleManager) GetRuleById(filename string, id int) (*model.Rule, error) {
	rules, err := rm.LoadRules(filename)
	if err != nil {
		return nil, err
	}

	for _, rule := range rules {
		if rule.ID == id {
			return &rule, nil
		}
	}

	return nil, nil
}

func (rm *RuleManager) GetAllRules(filename string) ([]model.Rule, error) {
	return rm.LoadRules(filename)
}

func (rm *RuleManager) GetSearchableRules(filename string) ([]model.Rule, error) {
	allRules, err := rm.LoadRules(filename)
	if err != nil {
		return nil, err
	}

	var searchableRules []model.Rule
	for _, rule := range allRules {
		if !rule.Search.Disabled {
			searchableRules = append(searchableRules, rule)
		}
	}

	return searchableRules, nil
}
