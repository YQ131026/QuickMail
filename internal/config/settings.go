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

	SecretEnvKey        = "MAIL_CONFIG_SECRET"
	APIKeyEnvKey        = "QUICKMAIL_API_KEY"
	PortEnvKey          = "PORT"
	ProviderStoreEnvKey = "QUICKMAIL_PROVIDER_STORE"
	ConfigFileEnvKey    = "QUICKMAIL_CONFIG_FILE"
)

// Settings 聚合服务运行所需的核心配置。
type Settings struct {
	ProviderStorePath string
	APIKey            string
	Secret            string
	Port              string
}

type fileSettings struct {
	ProviderStorePath string `json:"provider_store_path"`
	APIKey            string `json:"api_key"`
	Port              string `json:"port"`
}

// LoadSettings 从环境变量加载配置，提供统一入口。
func LoadSettings() (Settings, error) {
	secret := os.Getenv(SecretEnvKey)
	if secret == "" {
		return Settings{}, fmt.Errorf("%s environment variable is required", SecretEnvKey)
	}

	fs, err := readFileSettings()
	if err != nil {
		return Settings{}, err
	}

	providerStore := firstNonEmpty(
		os.Getenv(ProviderStoreEnvKey),
		fs.ProviderStorePath,
		DefaultProviderStorePath,
	)

	apiKey := os.Getenv(APIKeyEnvKey)
	if apiKey == "" {
		apiKey = fs.APIKey
	}

	port := firstNonEmpty(os.Getenv(PortEnvKey), fs.Port, DefaultPort)

	return Settings{
		ProviderStorePath: providerStore,
		APIKey:            apiKey,
		Secret:            secret,
		Port:              port,
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

func readFileSettings() (fileSettings, error) {
	path := strings.TrimSpace(os.Getenv(ConfigFileEnvKey))
	if path == "" {
		path = DefaultConfigPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fileSettings{}, nil
		}
		return fileSettings{}, fmt.Errorf("read config file %s: %w", path, err)
	}

	if len(data) == 0 {
		return fileSettings{}, nil
	}

	var fs fileSettings
	if err := json.Unmarshal(data, &fs); err != nil {
		return fileSettings{}, fmt.Errorf("parse config file %s: %w", path, err)
	}

	return fs, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
