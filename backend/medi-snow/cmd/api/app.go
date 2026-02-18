package main

import (
	"log/slog"
	"medi-meteorology/internal/config"
	"medi-meteorology/internal/location"
	"medi-meteorology/internal/weather"

	"github.com/gin-gonic/gin"

	_ "medi-meteorology/docs" // Ensure docs are imported
)

// App encapsulates application dependencies
type App struct {
	router          *gin.Engine
	logger          *slog.Logger
	locationService location.Service
	weatherService  weather.Service
	cfg             *config.Config
}

// NewApp creates a new application with injected dependencies
func NewApp(cfg *config.Config, logger *slog.Logger) (*App, error) {
	// Set Gin mode from configuration
	gin.SetMode(cfg.Server.GinMode)

	// Create Gin router
	router := gin.New()

	// Add middleware
	router.Use(gin.Recovery())

	// Initialize weather service
	weatherSvc, err := weather.NewWeatherService(cfg, logger)
	if err != nil {
		return nil, err
	}

	app := &App{
		router:          router,
		logger:          logger,
		locationService: location.NewLocationService(logger),
		cfg:             cfg,
		weatherService:  weatherSvc,
	}

	// Register routes
	app.registerRoutes()

	return app, nil
}

// Run starts the HTTP server
func (app *App) Run(addr string) error {
	return app.router.Run(addr)
}
