package types

type Temperature struct {
	Celsius    float64
	Fahrenheit float64
}

func NewTemperatureFromFahrenheit(fahrenheit float64) Temperature {
	var celsius = (fahrenheit - 32) * 5 / 9
	return Temperature{
		Celsius:    celsius,
		Fahrenheit: fahrenheit,
	}
}
