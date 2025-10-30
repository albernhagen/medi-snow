package main

// registerRoutes sets up all API endpoints
func (app *App) registerRoutes() {
	// Health check
	app.router.GET("/ping", app.handlePing)
}

// handlePing is a simple health check endpoint
