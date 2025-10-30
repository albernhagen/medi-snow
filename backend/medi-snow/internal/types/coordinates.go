package types

type Coords struct {
	Latitude  float64
	Longitude float64
}

func NewCoords(latitude, longitude float64) Coords {
	return Coords{
		Latitude:  latitude,
		Longitude: longitude,
	}
}
