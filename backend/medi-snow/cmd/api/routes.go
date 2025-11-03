package main

import (
	"github.com/danielgtaylor/huma/v2"
)

// registerRoutes sets up all API endpoints
func (app *App) registerRoutes() {
	// Health check endpoint
	huma.Register(app.api, huma.Operation{
		OperationID: "ping",
		Method:      "GET",
		Path:        "/ping",
		Summary:     "Ping health check",
		Description: "Check if the API is running",
		Tags:        []string{"health"},
	}, app.handlePing)
}
