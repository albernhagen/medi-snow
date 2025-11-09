package types

type SnowDepth struct {
	AmountInFeet   float64
	AmountInMeters float64
}

func NewSnowDepthFromFeet(amountInFeet float64) SnowDepth {
	return SnowDepth{
		AmountInFeet:   amountInFeet,
		AmountInMeters: amountInFeet * FeetToMeters,
	}
}
