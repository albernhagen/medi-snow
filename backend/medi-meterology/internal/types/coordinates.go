package types

type Coords struct {
	Latitude  float64 `json:"latitude" example:"40.7128" doc:"Latitude in decimal degrees"`
	Longitude float64 `json:"longitude" example:"-74.0060" doc:"Longitude in decimal degrees"`
}

func NewCoords(latitude, longitude float64) Coords {
	return Coords{
		Latitude:  latitude,
		Longitude: longitude,
	}
}
