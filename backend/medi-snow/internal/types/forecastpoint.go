package types

// ForecastPoint represents a geographic location with metadata
// used for weather forecasting
type ForecastPoint struct {
	Coordinates Coords       `json:"coordinates" doc:"Geographic coordinates"`
	Elevation   Elevation    `json:"elevation" doc:"Elevation data"`
	Location    LocationInfo `json:"location" doc:"Human-readable location information"`
}
