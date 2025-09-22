package config

import (
	"fmt"
	"os"
	"strings"
)

const (
	DefaultProviderStorePath = "data/providers.json"
	DefaultPort              = "8080"

	SecretEnvKey        = "MAIL_CONFIG_SECRET"
	APIKeyEnvKey        = "QUICKMAIL_API_KEY"
	PortEnvKey          = "PORT"
	ProviderStoreEnvKey = "QUICKMAIL_PROVIDER_STORE"
)

// Settings 聚合服务运行所需的核心配置。
type Settings struct {
	ProviderStorePath string
	APIKey            string
	Secret            string
	Port              string
}

// LoadSettings 从环境变量加载配置，提供统一入口。
func LoadSettings() (Settings, error) {
	secret := os.Getenv(SecretEnvKey)
	if secret == "" {
		return Settings{}, fmt.Errorf("%s environment variable is required", SecretEnvKey)
	}

	storePath := os.Getenv(ProviderStoreEnvKey)
	if storePath == "" {
		storePath = DefaultProviderStorePath
	}

	port := os.Getenv(PortEnvKey)
	if port == "" {
		port = DefaultPort
	}

	return Settings{
		ProviderStorePath: storePath,
		APIKey:            os.Getenv(APIKeyEnvKey),
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
