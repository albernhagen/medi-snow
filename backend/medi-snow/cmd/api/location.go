package main

import (
	"errors"
	"net/http"

	"medi-meteorology/internal/location"
	_ "medi-meteorology/internal/types" // imported for swagger type definitions

	"github.com/gin-gonic/gin"
)

// GetForecastPointInput defines the query parameters for the forecast point endpoint
type GetForecastPointInput struct {
	Latitude  float64 `form:"latitude" binding:"required"`  // Latitude in decimal degrees
	Longitude float64 `form:"longitude" binding:"required"` // Longitude in decimal degrees
}

// handleGetForecastPoint godoc
// @Summary Get forecast point data
// @Description Retrieve comprehensive location data including coordinates, elevation, and location metadata for a given latitude and longitude
// @Tags location
// @Accept json
// @Produce json
// @Param latitude query number true "Latitude in decimal degrees" minimum(-90) maximum(90) example(39.11539)
// @Param longitude query number true "Longitude in decimal degrees" minimum(-180) maximum(180) example(-107.65840)
// @Success 200 {object} types.ForecastPoint
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /location/forecast-point [get]
func (app *App) handleGetForecastPoint(c *gin.Context) {
	var input GetForecastPointInput

	// Bind and validate query parameters
	if err := c.ShouldBindQuery(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Delegate to business layer
	forecastPoint, err := app.locationService.GetForecastPoint(input.Latitude, input.Longitude)
	if err != nil {
		// Check if it's a validation error from business layer
		if errors.Is(err, location.ErrInvalidLatitude) || errors.Is(err, location.ErrInvalidLongitude) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Other errors are internal server errors
		app.logger.Error("failed to get forecast point",
			"latitude", input.Latitude,
			"longitude", input.Longitude,
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get forecast point"})
		return
	}

	c.JSON(http.StatusOK, forecastPoint)
}
