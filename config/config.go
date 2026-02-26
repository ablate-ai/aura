package config

import "time"

// Config Prometheus 客户端配置
type Config struct {
	BaseURL    string        // Prometheus 地址
	Timeout    time.Duration // 请求超时时间
	HTTPClient interface{}   // 可选的自定义 HTTP 客户端
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		BaseURL: "http://prom.ooxo.cc/",
		Timeout: 30 * time.Second,
	}
}
