package main

import (
	"log"
	"medi-snow/internal/config"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create app
	app := NewApp()

	// Start server
	log.Printf("Starting server on %s", cfg.GetServerAddr())
	if err := app.Run(cfg.GetServerAddr()); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
