package main

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// registerRoutes sets up all API endpoints
func (app *App) registerRoutes() {
	// Health check endpoint
	app.router.GET("/ping", app.handlePing)

	// Location endpoints
	app.router.GET("/location/forecast-point", app.handleGetForecastPoint)

	// Swagger documentation
	app.router.GET("/swagger/*any", func(c *gin.Context) {
		path := c.Param("any")
		if path == "/" {
			c.Redirect(301, "/swagger/index.html")
			return
		}
		ginSwagger.WrapHandler(swaggerFiles.Handler)(c)
	})
}
