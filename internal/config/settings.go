package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	DefaultPort       = "8080"
	DefaultConfigPath = "config/config.json"

	ConfigFileEnvKey = "QUICKMAIL_CONFIG_FILE"
)

// Settings 聚合服务运行所需的核心配置。
type Settings struct {
	ConfigPath string
	APIKey     string
	Secret     string
	Port       string
	providers  []providerRecord
}

type fileConfig struct {
	APIKey    string           `json:"api_key"`
	Secret    string           `json:"secret"`
	Port      string           `json:"port"`
	Providers []providerRecord `json:"providers"`
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

	var cfg fileConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Settings{}, fmt.Errorf("parse config file %s: %w", path, err)
	}

	secret := strings.TrimSpace(cfg.Secret)
	if err := validateSecret(secret); err != nil {
		return Settings{}, err
	}

	port := strings.TrimSpace(cfg.Port)
	if port == "" {
		port = DefaultPort
	}

	return Settings{
		ConfigPath: path,
		APIKey:     strings.TrimSpace(cfg.APIKey),
		Secret:     secret,
		Port:       port,
		providers:  cfg.Providers,
	}, nil
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

func validateSecret(secret string) error {
	if secret == "" {
		return errors.New("config secret must not be empty")
	}

	switch len([]byte(secret)) {
	case 16, 24, 32:
		return nil
	default:
		return errors.New("config secret must be 16, 24, or 32 bytes")
	}
}
