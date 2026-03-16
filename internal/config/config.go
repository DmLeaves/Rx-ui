package config

import (
	"os"
	"path/filepath"
)

// Config 应用配置
type Config struct {
	// 基础配置
	Debug    bool   `json:"debug"`
	LogLevel string `json:"logLevel"` // debug, info, warn, error

	// 数据库
	DBPath string `json:"dbPath"`

	// Web服务
	Web WebConfig `json:"web"`

	// 前端资源（方向1扩展）
	Frontend FrontendConfig `json:"frontend"`

	// Xray配置
	Xray XrayConfig `json:"xray"`
}

// WebConfig Web服务配置
type WebConfig struct {
	Listen   string `json:"listen"`
	Port     int    `json:"port"`
	BasePath string `json:"basePath"`
	CertFile string `json:"certFile"`
	KeyFile  string `json:"keyFile"`
}

// FrontendConfig 前端资源配置（方向1：CDN动态处理）
type FrontendConfig struct {
	// 模式: embedded（嵌入二进制）| cdn（外部CDN）| local（本地文件）
	Mode string `json:"mode"`

	// CDN提供商列表（按优先级排序）
	CDNProviders []CDNProvider `json:"cdnProviders"`

	// 当CDN不可用时是否回退到嵌入资源
	FallbackToEmbedded bool `json:"fallbackToEmbedded"`
}

// CDNProvider CDN提供商配置
type CDNProvider struct {
	Name     string `json:"name"`     // 名称（用于日志）
	BaseURL  string `json:"baseUrl"`  // 基础URL
	Priority int    `json:"priority"` // 优先级（越小越高）
	Enabled  bool   `json:"enabled"`  // 是否启用
}

// XrayConfig Xray配置
type XrayConfig struct {
	BinPath      string `json:"binPath"`      // Xray二进制路径
	ConfigPath   string `json:"configPath"`   // 配置文件路径
	AssetPath    string `json:"assetPath"`    // 资源文件路径（geoip, geosite）
	LogPath      string `json:"logPath"`      // 日志路径
	RestartDelay int    `json:"restartDelay"` // 重启延迟（秒）
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".rx-ui")

	return &Config{
		Debug:    false,
		LogLevel: "info",
		DBPath:   filepath.Join(dataDir, "rx-ui.db"),
		Web: WebConfig{
			Listen:   "",
			Port:     54321,
			BasePath: "/",
		},
		Frontend: FrontendConfig{
			Mode:               "embedded",
			CDNProviders:       []CDNProvider{},
			FallbackToEmbedded: true,
		},
		Xray: XrayConfig{
			BinPath:      filepath.Join(dataDir, "bin", "xray"),
			ConfigPath:   filepath.Join(dataDir, "xray.json"),
			AssetPath:    filepath.Join(dataDir, "bin"),
			LogPath:      filepath.Join(dataDir, "logs"),
			RestartDelay: 3,
		},
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	// TODO: 添加配置验证逻辑
	return nil
}
