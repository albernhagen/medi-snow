package main

//go:generate go run github.com/swaggo/swag/cmd/swag@latest init -g docs.go -o ../../docs --parseDependency

import (
	"log"
	"log/slog"
	"medi-meteorology/internal/config"

	_ "medi-meteorology/docs" // Import generated docs
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logger := cfg.NewLogger()
	slog.SetDefault(logger) // Set as default logger for the application

	// Create app
	app, err := NewApp(cfg, logger)
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	// Start server
	logger.Info("starting server", "addr", cfg.GetServerAddr())
	if err := app.Run(cfg.GetServerAddr()); err != nil {
		logger.Error("server failed", "error", err)
		log.Fatal(err)
	}
}
