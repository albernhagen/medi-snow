package weather

import (
	"log/slog"
	"medi-snow/internal/config"
	"medi-snow/internal/providers/openmeteo"
	"medi-snow/internal/types"
)

type ForecastProvider interface {
	// GetForecast fetches the weather forecast for the given latitude, longitude, and elevation in meters
	GetForecast(latitude, longitude, elevationMeters float64, forecastDays int) (*openmeteo.ForecastAPIResponse, error)
}

type Service interface {
	GetForecast(point types.ForecastPoint) (*Forecast, error)
}

type weatherService struct {
	forecastProvider ForecastProvider
	cfg              *config.Config
	logger           *slog.Logger
}

func NewWeatherService(forecastProvider ForecastProvider, config *config.Config) Service {
	return &weatherService{
		forecastProvider: forecastProvider,
		cfg:              config,
		logger:           slog.Default().With("component", "weather-service"),
	}
}

func (s *weatherService) GetForecast(point types.ForecastPoint) (*Forecast, error) {
	// TODO validate point data
	forecastDays := s.cfg.App.ForecastDays

	apiResponse, err := s.forecastProvider.GetForecast(point.Coordinates.Latitude, point.Coordinates.Longitude, point.Elevation.Meters, forecastDays)
	if err != nil {
		s.logger.Error("failed to get forecast from provider: %v", err)
		return nil, err
	}

	return mapForecastAPIResponseToForecast(apiResponse), nil
}

func mapForecastAPIResponseToForecast(apiResponse *openmeteo.ForecastAPIResponse) *Forecast {
	return nil
}
