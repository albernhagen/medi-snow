package main

import (
	"github.com/danielgtaylor/huma/v2"
)

// registerRoutes sets up all API endpoints
func (app *App) registerRoutes() {
	app.logger.Info("registering routes")

	// Health check endpoint
	huma.Register(app.api, huma.Operation{
		OperationID: "ping",
		Method:      "GET",
		Path:        "/ping",
		Summary:     "Ping health check",
		Description: "Check if the API is running",
		Tags:        []string{"health"},
	}, app.handlePing)
	app.logger.Debug("registered route", "method", "GET", "path", "/ping")

	// Location endpoints
	huma.Register(app.api, huma.Operation{
		OperationID: "get-forecast-point",
		Method:      "GET",
		Path:        "/location/forecast-point",
		Summary:     "Get forecast point data",
		Description: "Retrieve comprehensive location data including coordinates, elevation, and location metadata for a given latitude and longitude",
		Tags:        []string{"location"},
	}, app.handleGetForecastPoint)
	app.logger.Debug("registered route", "method", "GET", "path", "/location/forecast-point")

	app.logger.Info("all routes registered")
}
