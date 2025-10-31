package types

// ForecastPoint represents a geographic location with metadata
// used for weather forecasting
type ForecastPoint struct {
	Coordinates Coords
	Elevation   Elevation
	Location    LocationInfo
}
