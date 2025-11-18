package main

import (
	"log/slog"
	"medi-snow/internal/config"
	"medi-snow/internal/location"

	"github.com/gin-gonic/gin"

	_ "medi-snow/docs" // Ensure docs are imported
)

// App encapsulates application dependencies
type App struct {
	router          *gin.Engine
	logger          *slog.Logger
	locationService location.Service
}

// NewApp creates a new application with injected dependencies
func NewApp(cfg *config.Config, logger *slog.Logger) *App {
	// Set Gin mode from configuration
	gin.SetMode(cfg.Server.GinMode)

	// Create Gin router
	router := gin.New()

	// Add middleware
	router.Use(gin.Recovery())

	app := &App{
		router:          router,
		logger:          logger,
		locationService: location.NewLocationService(logger),
	}

	// Register routes
	app.registerRoutes()

	return app
}

// Run starts the HTTP server
func (app *App) Run(addr string) error {
	return app.router.Run(addr)
}
