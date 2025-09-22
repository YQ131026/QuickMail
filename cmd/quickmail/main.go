package main

import (
	"log"
	"os"

	"QuickMail/internal/config"
	"QuickMail/internal/email"
	"QuickMail/internal/server"
)

func main() {
	logger := log.New(os.Stdout, "quickmail ", log.LstdFlags)

	settings, err := config.LoadSettings()
	if err != nil {
		logger.Fatalf("failed to load settings: %v", err)
	}

	store, err := config.NewStore(settings.ProviderStorePath, []byte(settings.Secret))
	if err != nil {
		logger.Fatalf("failed to initialize provider store: %v", err)
	}

	sender := &email.Sender{Store: store, Logger: logger}

	srv := server.New(store, sender, settings.APIKey)

	addr := settings.ListenAddr()
	logger.Printf("starting server on %s", addr)
	if err := srv.Run(addr); err != nil {
		logger.Fatalf("server error: %v", err)
	}
}
