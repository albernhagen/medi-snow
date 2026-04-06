package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// PingResponse represents the response for the ping endpoint
type PingResponse struct {
	Message string `json:"message" example:"pong"` // Response message
}

// handlePing godoc
// @Summary Ping health check
// @Description Check if the API is running
// @Tags health
// @Produce json
// @Success 200 {object} PingResponse
// @Router /ping [get]
func (app *App) handlePing(c *gin.Context) {
	c.JSON(http.StatusOK, PingResponse{
		Message: "pong",
	})
}
