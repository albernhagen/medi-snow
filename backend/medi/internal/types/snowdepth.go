package types

type SnowDepth struct {
	Feet   float64
	Meters float64
}

func NewSnowDepthFromFeet(amountInFeet float64) SnowDepth {
	return SnowDepth{
		Feet:   amountInFeet,
		Meters: amountInFeet * FeetToMeters,
	}
}
