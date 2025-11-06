package main

import (
	"context"
)

// PingOutput represents the response for the ping endpoint
type PingOutput struct {
	Body struct {
		Message string `json:"message" example:"pong" doc:"Response message"`
	}
}

// handlePing is a health check endpoint that returns a simple pong message
func (app *App) handlePing(ctx context.Context, input *struct{}) (*PingOutput, error) {
	resp := &PingOutput{}
	resp.Body.Message = "pong"
	return resp, nil
}
