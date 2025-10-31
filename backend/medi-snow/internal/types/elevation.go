package types

const FeetToMeters = 0.3048

type Elevation struct {
	Feet   float64
	Meters float64
}

func NewElevationFromFeet(feet float64) Elevation {
	return Elevation{
		Meters: feet * FeetToMeters,
		Feet:   feet,
	}
}
