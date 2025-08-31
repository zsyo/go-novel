// Package embed provides embedded configuration and rule files
package embed

import (
	"embed"
)

//go:embed configs/config.ini
var ConfigFile []byte

//go:embed configs/rules/main-rules.json
var MainRulesFile []byte

//go:embed configs/rules/flowlimit-rules.json
var FlowLimitRulesFile []byte

//go:embed configs/rules/non-searchable-rules.json
var NonSearchableRulesFile []byte

//go:embed configs/rules/proxy-rules.json
var ProxyRulesFile []byte

//go:embed static/*
var StaticFiles embed.FS

// GetEmbeddedConfigFile 获取嵌入的配置文件内容
func GetEmbeddedConfigFile() []byte {
	return ConfigFile
}

// GetEmbeddedRulesFile 获取嵌入的规则文件内容
func GetEmbeddedRulesFile(filename string) []byte {
	switch filename {
	case "main-rules.json":
		return MainRulesFile
	case "flowlimit-rules.json":
		return FlowLimitRulesFile
	case "non-searchable-rules.json":
		return NonSearchableRulesFile
	case "proxy-rules.json":
		return ProxyRulesFile
	default:
		return nil
	}
}

// GetEmbeddedStaticFiles 获取嵌入的静态文件系统
func GetEmbeddedStaticFiles() embed.FS {
	return StaticFiles
}
