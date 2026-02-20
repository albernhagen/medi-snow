package types

type Precipitation struct {
	Inches float64
	Mm     float64
}

func NewPrecipitationFromInches(amountInInches float64) Precipitation {
	return Precipitation{
		Inches: amountInInches,
		Mm:     amountInInches * InchesToMm,
	}
}
