package main

import (
	"github.com/gin-gonic/gin"
)

// App encapsulates application dependencies
type App struct {
	router *gin.Engine
}

// NewApp creates a new application with injected dependencies
func NewApp() *App {
	app := &App{
		router: gin.Default(),
	}

	// Register routes
	app.registerRoutes()

	return app
}

// Run starts the HTTP server
func (app *App) Run(addr string) error {
	return app.router.Run(addr)
}
