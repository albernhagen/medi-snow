package avalanche

import (
	"fmt"
	"log/slog"
	"medi-snow/internal/providers/nac"
)

// MapLayerProvider fetches the NAC map layer with all forecast zone polygons.
type MapLayerProvider interface {
	GetMapLayer() (*nac.MapLayerResponse, error)
}

// ForecastProvider fetches an avalanche forecast for a specific center and zone.
type ForecastProvider interface {
	GetForecast(centerId string, zoneId int) (*nac.ForecastResponse, error)
}

// Service provides avalanche forecast data.
type Service interface {
	GetForecast(latitude, longitude float64) (*AvalancheForecast, error)
}

type avalancheService struct {
	mapLayerProvider MapLayerProvider
	forecastProvider ForecastProvider
	logger           *slog.Logger
}

// NewAvalancheService creates a new avalanche service with a real NAC client.
func NewAvalancheService(logger *slog.Logger) Service {
	client := nac.NewClient(logger)
	return NewAvalancheServiceWithProviders(logger, client, client)
}

// NewAvalancheServiceWithProviders creates a new avalanche service with custom providers.
// This is useful for testing with mock providers.
func NewAvalancheServiceWithProviders(
	logger *slog.Logger,
	mapLayerProvider MapLayerProvider,
	forecastProvider ForecastProvider,
) Service {
	return &avalancheService{
		mapLayerProvider: mapLayerProvider,
		forecastProvider: forecastProvider,
		logger:           logger.With("component", "avalanche-service"),
	}
}

// GetForecast retrieves an avalanche forecast for the given coordinates.
// It finds the matching forecast zone, fetches the forecast from NAC, and maps
// it to domain types.
func (s *avalancheService) GetForecast(latitude, longitude float64) (*AvalancheForecast, error) {
	s.logger.Debug("getting avalanche forecast",
		"latitude", latitude,
		"longitude", longitude,
	)

	// Fetch the map layer to find the matching zone
	mapLayer, err := s.mapLayerProvider.GetMapLayer()
	if err != nil {
		s.logger.Error("failed to get map layer", "error", err)
		return nil, fmt.Errorf("failed to get map layer: %w", err)
	}

	// Find the zone containing the given coordinates
	zone := nac.FindZone(latitude, longitude, mapLayer)
	if zone == nil {
		s.logger.Warn("no avalanche forecast zone found for coordinates",
			"latitude", latitude,
			"longitude", longitude,
		)
		return nil, fmt.Errorf("no avalanche forecast zone found for coordinates (%.6f, %.6f)", latitude, longitude)
	}

	s.logger.Debug("found forecast zone",
		"zone_id", zone.Id,
		"zone_name", zone.Properties.Name,
		"center_id", zone.Properties.CenterId,
	)

	// Fetch the forecast for this zone
	forecastResp, err := s.forecastProvider.GetForecast(zone.Properties.CenterId, zone.Id)
	if err != nil {
		s.logger.Error("failed to get forecast",
			"center_id", zone.Properties.CenterId,
			"zone_id", zone.Id,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get forecast: %w", err)
	}

	// Map NAC response to domain model
	forecast := mapForecastResponse(zone, forecastResp)

	s.logger.Debug("successfully mapped avalanche forecast",
		"zone_name", forecast.Zone.Name,
		"center_id", forecast.Center.Id,
		"danger_ratings", len(forecast.DangerRatings),
		"problems", len(forecast.Problems),
	)

	return forecast, nil
}
