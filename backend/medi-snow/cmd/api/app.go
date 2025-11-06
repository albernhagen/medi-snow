package main

import (
	"log/slog"
	"medi-snow/internal/location"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

// App encapsulates application dependencies
type App struct {
	mux             *http.ServeMux
	api             huma.API
	logger          *slog.Logger
	locationService location.Service
}

// NewApp creates a new application with injected dependencies
func NewApp(logger *slog.Logger) *App {
	// Create standard library HTTP mux
	mux := http.NewServeMux()

	// Create Huma API with standard library adapter
	config := huma.DefaultConfig("Medi-Snow API", "1.0.0")
	config.Info.Description = "Weather and avalanche forecasting API for ski areas"
	config.Info.Contact = &huma.Contact{
		Name:  "API Support",
		Email: "support@example.com",
	}
	config.Servers = []*huma.Server{
		{URL: "http://localhost:8080", Description: "Development server"},
	}

	api := humago.New(mux, config)

	app := &App{
		mux:             mux,
		api:             api,
		logger:          logger,
		locationService: location.NewLocationService(logger),
	}

	logger.Info("application initialized")

	// Register routes
	app.registerRoutes()

	return app
}

// Run starts the HTTP server
func (app *App) Run(addr string) error {
	return http.ListenAndServe(addr, app.mux)
}
