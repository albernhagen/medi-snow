package location

import (
	"fmt"
	"log/slog"
	"medi-snow/internal/providers/openstreetmap"
	"medi-snow/internal/providers/usgs"
	"medi-snow/internal/types"
	"sync"
)

// Service provides location and elevation data for weather forecasting
type Service interface {
	// GetForecastPoint retrieves comprehensive location data for a given coordinate
	GetForecastPoint(latitude, longitude float64) (*types.ForecastPoint, error)
}

// ElevationProvider defines the interface for elevation data providers
type ElevationProvider interface {
	GetElevationPoint(latitude, longitude float64) (*usgs.ElevationPointAPIResponse, error)
}

// ReverseGeocodeProvider defines the interface for location data providers
type ReverseGeocodeProvider interface {
	Lookup(latitude, longitude float64) (*openstreetmap.LookupAPIResponse, error)
}

// locationService implements the Service interface
type locationService struct {
	elevationProvider ElevationProvider
	locationProvider  ReverseGeocodeProvider
	logger            *slog.Logger
}

// NewLocationService creates a new location service with real provider clients
func NewLocationService(logger *slog.Logger) Service {
	return &locationService{
		elevationProvider: usgs.NewClient(logger),
		locationProvider:  openstreetmap.NewClient(logger),
		logger:            logger.With("component", "location-service"),
	}
}

// NewLocationServiceWithProviders creates a new location service with custom providers
// This is useful for testing with mock providers
func NewLocationServiceWithProviders(
	elevationProvider ElevationProvider,
	locationProvider ReverseGeocodeProvider,
) Service {
	return &locationService{
		elevationProvider: elevationProvider,
		locationProvider:  locationProvider,
	}
}

// GetForecastPoint retrieves comprehensive location data by calling providers in parallel
func (s *locationService) GetForecastPoint(latitude, longitude float64) (*types.ForecastPoint, error) {
	s.logger.Debug("getting forecast point",
		"latitude", latitude,
		"longitude", longitude,
	)

	var (
		wg            sync.WaitGroup
		elevationResp *usgs.ElevationPointAPIResponse
		locationResp  *openstreetmap.LookupAPIResponse
		elevationErr  error
		locationErr   error
	)

	// Launch both API calls in parallel
	wg.Add(2)

	// Get elevation data
	go func() {
		defer wg.Done()
		elevationResp, elevationErr = s.elevationProvider.GetElevationPoint(latitude, longitude)
		if elevationErr != nil {
			elevationErr = fmt.Errorf("failed to get elevation: %w", elevationErr)
		}
	}()

	// Get location data
	go func() {
		defer wg.Done()
		locationResp, locationErr = s.locationProvider.Lookup(latitude, longitude)
		if locationErr != nil {
			locationErr = fmt.Errorf("failed to get location: %w", locationErr)
		}
	}()

	// Wait for both calls to complete
	wg.Wait()

	// Check for errors
	if elevationErr != nil && locationErr != nil {
		s.logger.Error("multiple provider errors",
			"latitude", latitude,
			"longitude", longitude,
			"elevation_error", elevationErr,
			"location_error", locationErr,
		)
		return nil, fmt.Errorf("multiple errors: elevation: %v; location: %v", elevationErr, locationErr)
	}

	if elevationErr != nil {
		s.logger.Error("elevation provider error",
			"latitude", latitude,
			"longitude", longitude,
			"error", elevationErr,
		)
		return nil, elevationErr
	}
	if locationErr != nil {
		s.logger.Error("location provider error",
			"latitude", latitude,
			"longitude", longitude,
			"error", locationErr,
		)
		return nil, locationErr
	}

	// Translate provider responses to domain types
	elevation, err := s.translateElevation(elevationResp)
	if err != nil {
		return nil, err
	}

	locationInfo, err := s.translateLocationInfo(locationResp)
	if err != nil {
		return nil, err
	}

	forecastPoint := &types.ForecastPoint{
		Coordinates: types.NewCoords(latitude, longitude),
		Elevation:   elevation,
		Location:    locationInfo,
	}

	s.logger.Debug("successfully retrieved forecast point",
		"latitude", latitude,
		"longitude", longitude,
		"location_name", locationInfo.Name,
	)

	return forecastPoint, nil
}

// translateElevation converts an OpenMeteo elevation response to domain Elevation type
func (s *locationService) translateElevation(resp *usgs.ElevationPointAPIResponse) (types.Elevation, error) {
	if resp == nil {
		return types.Elevation{}, fmt.Errorf("elevation response is nil")
	}

	// OpenMeteo returns elevation in meters
	return types.NewElevationFromFeet(resp.Value), nil
}

// translateLocationInfo converts an OpenStreetMap reverse lookup response to domain LocationInfo type
func (s *locationService) translateLocationInfo(resp *openstreetmap.LookupAPIResponse) (types.LocationInfo, error) {
	if resp == nil {
		return types.LocationInfo{}, fmt.Errorf("lookup response is nil")
	}

	// Extract the display name or name as the location name
	name := resp.DisplayName
	if resp.Name != "" {
		name = resp.Name
	}

	return types.LocationInfo{
		Name:        name,
		County:      resp.Address.County,
		State:       resp.Address.State,
		Country:     resp.Address.Country,
		CountryCode: resp.Address.CountryCode,
	}, nil
}
