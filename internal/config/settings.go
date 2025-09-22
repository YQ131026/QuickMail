package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	DefaultProviderStorePath = "data/providers.json"
	DefaultPort              = "8080"
	DefaultConfigPath        = "config/config.json"

	ConfigFileEnvKey = "QUICKMAIL_CONFIG_FILE"
)

// Settings 聚合服务运行所需的核心配置。
type Settings struct {
	ProviderStorePath string `json:"provider_store_path"`
	APIKey            string `json:"api_key"`
	Secret            string `json:"secret"`
	Port              string `json:"port"`
}

// LoadSettings 仅从 JSON 配置文件加载核心配置。
func LoadSettings() (Settings, error) {
	path := strings.TrimSpace(os.Getenv(ConfigFileEnvKey))
	if path == "" {
		path = DefaultConfigPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Settings{}, fmt.Errorf("read config file %s: %w", path, err)
	}

	var cfg Settings
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Settings{}, fmt.Errorf("parse config file %s: %w", path, err)
	}

	cfg.ProviderStorePath = strings.TrimSpace(cfg.ProviderStorePath)
	if cfg.ProviderStorePath == "" {
		cfg.ProviderStorePath = DefaultProviderStorePath
	}

	cfg.Port = strings.TrimSpace(cfg.Port)
	if cfg.Port == "" {
		cfg.Port = DefaultPort
	}

	cfg.Secret = strings.TrimSpace(cfg.Secret)
	if cfg.Secret == "" {
		return Settings{}, errors.New("config secret must not be empty")
	}

	return cfg, nil
}

// ListenAddr 返回可直接用于 Gin 的监听地址。
func (s Settings) ListenAddr() string {
	if s.Port == "" {
		return ":" + DefaultPort
	}

	if strings.HasPrefix(s.Port, ":") {
		return s.Port
	}

	return ":" + s.Port
}
