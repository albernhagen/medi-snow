package types

const FeetToMeters = 0.3048

type Elevation struct {
	Feet   float64 `json:"feet" example:"5280" doc:"Elevation in feet"`
	Meters float64 `json:"meters" example:"1609.34" doc:"Elevation in meters"`
}

func NewElevationFromFeet(feet float64) Elevation {
	return Elevation{
		Meters: feet * FeetToMeters,
		Feet:   feet,
	}
}
