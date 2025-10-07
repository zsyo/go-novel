// Package embed provides embedded configuration and rule files
package embed

import (
	"embed"
	"fmt"
	"path/filepath"

	"go-novel/internal/util"
)

//go:embed configs/config.ini
var configFile []byte

//go:embed configs/rules/*
var ruleFiles embed.FS

//go:embed static/*
var staticFiles embed.FS

// GetEmbeddedConfigFile 获取嵌入的配置文件内容
func GetEmbeddedConfigFile() []byte {
	return configFile
}

// GetEmbeddedRulesFile 获取嵌入的规则文件内容
func GetEmbeddedRulesFile(filename string) []byte {
	data, err := ruleFiles.ReadFile(filepath.Join("configs/rules", filename))
	if err != nil {
		fmt.Printf("failed to read rule file %s: %v", filename, err)
		return nil
	}
	return data
}

// GetEmbeddedStaticFiles 获取嵌入的静态文件系统
func GetEmbeddedStaticFiles() embed.FS {
	return staticFiles
}

// WriteEmbeddedFiles 将嵌入文件内容写入磁盘
func WriteEmbeddedFiles() error {
	if !util.FileExists("configs/config.ini") {
		if err := util.WriteFile("configs/config.ini", configFile, 0644); err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}
	}

	fs, err := ruleFiles.ReadDir("configs/rules")
	if err != nil {
		return fmt.Errorf("failed to read rule files: %w", err)
	}

	for _, file := range fs {
		if file.IsDir() {
			continue
		}
		data := GetEmbeddedRulesFile(file.Name())
		if data != nil {
			fp := filepath.Join("configs/rules", file.Name())
			if !util.FileExists(fp) {
				if err := util.WriteFile(fp, data, 0644); err != nil {
					return fmt.Errorf("failed to write rule file %s: %w", fp, err)
				}
			}
		}
	}

	return nil
}
