package types

const MetersToFeet = 3.28084

type Elevation struct {
	Feet   float64
	Meters float64
}

func NewElevationFromMeters(meters float64) Elevation {
	return Elevation{
		Meters: meters,
		Feet:   meters * MetersToFeet,
	}
}
