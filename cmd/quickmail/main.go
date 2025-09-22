package main

import (
	"log"
	"os"

	"QuickMail/internal/config"
	"QuickMail/internal/email"
	"QuickMail/internal/server"
)

const (
	defaultPort      = "8080"
	configFilePath   = "data/providers.json"
	secretEnvKey     = "MAIL_CONFIG_SECRET"
	apiKeyEnvKey     = "QUICKMAIL_API_KEY"
	listenAddressEnv = "PORT"
)

func main() {
	logger := log.New(os.Stdout, "quickmail ", log.LstdFlags)

	secret := os.Getenv(secretEnvKey)
	if len(secret) == 0 {
		logger.Fatalf("%s environment variable is required", secretEnvKey)
	}

	store, err := config.NewStore(configFilePath, []byte(secret))
	if err != nil {
		logger.Fatalf("failed to initialize provider store: %v", err)
	}

	sender := &email.Sender{Store: store, Logger: logger}
	apiKey := os.Getenv(apiKeyEnvKey)

	srv := server.New(store, sender, apiKey)

	port := os.Getenv(listenAddressEnv)
	if port == "" {
		port = defaultPort
	}

	addr := ":" + port
	logger.Printf("starting server on %s", addr)
	if err := srv.Run(addr); err != nil {
		logger.Fatalf("server error: %v", err)
	}
}
