package main

import (
	"log"
	"log/slog"
	"medi-snow/internal/config"
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
	app := NewApp(logger)

	// Start server
	logger.Info("starting server", "addr", cfg.GetServerAddr())
	if err := app.Run(cfg.GetServerAddr()); err != nil {
		logger.Error("server failed", "error", err)
		log.Fatal(err)
	}
}
