package main

import (
	"context"
	"medi-snow/internal/types"
)

// GetForecastPointInput defines the query parameters for the forecast point endpoint
type GetForecastPointInput struct {
	Latitude  float64 `query:"latitude" required:"true" minimum:"-90" maximum:"90" doc:"Latitude in decimal degrees" example:"39.11539"`
	Longitude float64 `query:"longitude" required:"true" minimum:"-180" maximum:"180" doc:"Longitude in decimal degrees" example:"-107.65840"`
}

// GetForecastPointOutput represents the response for the forecast point endpoint
type GetForecastPointOutput struct {
	Body types.ForecastPoint
}

// handleGetForecastPoint retrieves comprehensive location data for a given coordinate
func (app *App) handleGetForecastPoint(ctx context.Context, input *GetForecastPointInput) (*GetForecastPointOutput, error) {
	app.logger.Info("getting forecast point",
		"latitude", input.Latitude,
		"longitude", input.Longitude,
	)

	forecastPoint, err := app.locationService.GetForecastPoint(input.Latitude, input.Longitude)
	if err != nil {
		app.logger.Error("failed to get forecast point",
			"latitude", input.Latitude,
			"longitude", input.Longitude,
			"error", err,
		)
		return nil, err
	}

	app.logger.Debug("successfully retrieved forecast point",
		"latitude", input.Latitude,
		"longitude", input.Longitude,
		"location_name", forecastPoint.Location.Name,
	)

	return &GetForecastPointOutput{Body: *forecastPoint}, nil
}
