package types

const InchesToMm = 25.4

type Precipitation struct {
	AmountInInches float64
	AmountInMm     float64
}

func NewPrecipitationFromInches(amountInInches float64) Precipitation {
	return Precipitation{
		AmountInInches: amountInInches,
		AmountInMm:     amountInInches * InchesToMm,
	}
}
