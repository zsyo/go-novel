package config

import (
	"bytes"
	"fmt"
	"sync"

	"so-novel/internal/embed"

	"github.com/spf13/viper"
)

type Config struct {
	Download DownloadConfig `mapstructure:"download"`
	Source   SourceConfig   `mapstructure:"source"`
	Crawl    CrawlConfig    `mapstructure:"crawl"`
	Web      WebConfig      `mapstructure:"web"`
	Proxy    ProxyConfig    `mapstructure:"proxy"`
}

type DownloadConfig struct {
	DownloadPath         string `mapstructure:"download-path"`
	ExtName              string `mapstructure:"extname"`
	PreserveChapterCache int    `mapstructure:"preserve-chapter-cache"`
	DownloadId           string `mapstructure:"download-id"` // 添加下载ID字段
}

type SourceConfig struct {
	Language    string `mapstructure:"language"`
	ActiveRules string `mapstructure:"active-rules"`
	SourceId    int    `mapstructure:"source-id"`
	SearchLimit int    `mapstructure:"search-limit"`
}

type CrawlConfig struct {
	Threads          int `mapstructure:"threads"`
	MinInterval      int `mapstructure:"min-interval"`
	MaxInterval      int `mapstructure:"max-interval"`
	EnableRetry      int `mapstructure:"enable-retry"`
	MaxRetries       int `mapstructure:"max-retries"`
	RetryMinInterval int `mapstructure:"retry-min-interval"`
	RetryMaxInterval int `mapstructure:"retry-max-interval"`
}

type WebConfig struct {
	Enabled int `mapstructure:"enabled"`
	Port    int `mapstructure:"port"`
}

type ProxyConfig struct {
	Enabled int    `mapstructure:"enabled"`
	Host    string `mapstructure:"host"`
	Port    int    `mapstructure:"port"`
}

var (
	config *Config
	once   sync.Once
)

func InitConfig() *Config {
	once.Do(func() {
		// 设置配置文件
		viper.SetConfigName("config") // 配置文件名为 config
		viper.SetConfigType("ini")    // 类型为 ini
		viper.AddConfigPath("./configs")
		viper.AddConfigPath(".")

		// 设置默认值
		viper.SetDefault("download.download-path", "downloads")
		viper.SetDefault("download.extname", "epub")
		viper.SetDefault("download.preserve-chapter-cache", 0)
		viper.SetDefault("source.language", "")
		viper.SetDefault("source.active-rules", "main-rules.json")
		viper.SetDefault("source.source-id", -1)
		viper.SetDefault("source.search-limit", 10)
		viper.SetDefault("crawl.threads", -1)
		viper.SetDefault("crawl.min-interval", 200)
		viper.SetDefault("crawl.max-interval", 400)
		viper.SetDefault("crawl.enable-retry", 1)
		viper.SetDefault("crawl.max-retries", 5)
		viper.SetDefault("crawl.retry-min-interval", 2000)
		viper.SetDefault("crawl.retry-max-interval", 4000)
		viper.SetDefault("web.enabled", 0)
		viper.SetDefault("web.port", 7765)
		viper.SetDefault("proxy.enabled", 0)
		viper.SetDefault("proxy.host", "127.0.0.1")
		viper.SetDefault("proxy.port", 7890)

		// 读取配置文件
		if err := viper.ReadInConfig(); err != nil {
			// 如果是配置文件不存在错误，尝试使用嵌入的配置文件
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				fmt.Printf("配置文件不存在，尝试使用嵌入的配置文件\n")
				// 使用嵌入的配置文件
				if embeddedConfig := embed.GetEmbeddedConfigFile(); embeddedConfig != nil {
					if err := viper.ReadConfig(bytes.NewBuffer(embeddedConfig)); err != nil {
						fmt.Printf("读取嵌入的配置文件遇到错误: %v\n", err)
					} else {
						fmt.Printf("成功读取嵌入的配置文件\n")
					}
				} else {
					fmt.Printf("未找到嵌入的配置文件，使用默认配置\n")
				}
			} else {
				// 其他类型的错误，显示详细信息
				fmt.Printf("读取配置文件遇到错误: %v\n", err)
			}
		} else {
			fmt.Printf("成功读取配置文件: %s\n", viper.ConfigFileUsed())
		}

		config = &Config{}
		if err := viper.Unmarshal(config); err != nil {
			panic(err)
		}
	})

	return config
}

// GetConfig 获取配置实例
func GetConfig() *Config {
	// 如果还没有初始化，先初始化
	if config == nil {
		return InitConfig()
	}
	return config
}
