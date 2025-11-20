package weather

import (
	"fmt"
	"log/slog"
	"medi-snow/internal/config"
	"medi-snow/internal/providers/openmeteo"
	"medi-snow/internal/timezone"
	"medi-snow/internal/types"
)

type ForecastProvider interface {
	// GetForecast fetches the weather forecast for the given latitude, longitude, elevation, and timezone
	GetForecast(latitude, longitude, elevationMeters float64, forecastDays int, timezone string) (*openmeteo.ForecastAPIResponse, error)
}

type Service interface {
	GetForecast(point types.ForecastPoint) (*Forecast, error)
}

type weatherService struct {
	forecastProvider ForecastProvider
	timezoneService  timezone.Service
	cfg              *config.Config
	logger           *slog.Logger
}

func NewWeatherService(config *config.Config, logger *slog.Logger) (Service, error) {
	tzSvc, err := timezone.NewService()
	if err != nil {
		return nil, fmt.Errorf("failed to create timezone service: %w", err)
	}
	return NewWeatherServiceWithProvider(openmeteo.NewClient(logger), tzSvc, config, logger), nil
}

func NewWeatherServiceWithProvider(
	forecastProvider ForecastProvider,
	timezoneService timezone.Service,
	cfg *config.Config,
	logger *slog.Logger,
) Service {
	return &weatherService{
		forecastProvider: forecastProvider,
		timezoneService:  timezoneService,
		cfg:              cfg,
		logger:           logger.With("component", "weather-service"),
	}
}

func (s *weatherService) GetForecast(point types.ForecastPoint) (*Forecast, error) {
	// TODO validate point data
	forecastDays := s.cfg.App.ForecastDays

	// Look up timezone for the location
	tz, err := s.timezoneService.GetTimezone(point.Coordinates.Latitude, point.Coordinates.Longitude)
	if err != nil {
		s.logger.Error("failed to determine timezone",
			"latitude", point.Coordinates.Latitude,
			"longitude", point.Coordinates.Longitude,
			"error", err,
		)
		return nil, fmt.Errorf("failed to determine timezone: %w", err)
	}

	s.logger.Debug("determined timezone for location",
		"latitude", point.Coordinates.Latitude,
		"longitude", point.Coordinates.Longitude,
		"timezone", tz,
	)

	// Get forecast with timezone
	apiResponse, err := s.forecastProvider.GetForecast(
		point.Coordinates.Latitude,
		point.Coordinates.Longitude,
		point.Elevation.Meters,
		forecastDays,
		tz,
	)
	if err != nil {
		s.logger.Error("failed to get forecast from provider", "error", err)
		return nil, fmt.Errorf("failed to get forecast: %w", err)
	}

	return mapForecastAPIResponseToForecast(apiResponse), nil
}

func mapForecastAPIResponseToForecast(apiResponse *openmeteo.ForecastAPIResponse) *Forecast {
	return nil
}
