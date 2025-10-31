package location

import "skadi-backend/internal/types"

// LocationService provides location and elevation data for weather forecasting
type LocationService interface {
	// GetForecastPoint retrieves comprehensive location data for a given coordinate
	GetForecastPoint(latitude, longitude float64) (*types.ForecastPoint, error)
}
