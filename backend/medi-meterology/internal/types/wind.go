package types

type Wind struct {
	Speed     WindSpeed
	Gusts     WindSpeed
	Direction WindDirection
}

type WindDirection struct {
	Degrees  float64
	Cardinal string
}

func NewWindSpeedFromMph(speedInMph float64) WindSpeed {
	return WindSpeed{
		Mph: speedInMph,
		Kph: speedInMph * MphToKph,
	}
}

func NewWindDirection(degrees int) WindDirection {
	if degrees < 0 || degrees >= 360 {
		return WindDirection{
			Degrees:  -1,
			Cardinal: "Unknown",
		}
	}

	degreesFloat := float64(degrees)
	direction := (degreesFloat / 22.5) + .5 // .5 for rounding
	var directionMap = make(map[int]string)
	directionMap[0] = "N"
	directionMap[1] = "NNE"
	directionMap[2] = "NE"
	directionMap[3] = "ENE"
	directionMap[4] = "E"
	directionMap[5] = "ESE"
	directionMap[6] = "SE"
	directionMap[7] = "SSE"
	directionMap[8] = "S"
	directionMap[9] = "SSW"
	directionMap[10] = "SW"
	directionMap[11] = "WSW"
	directionMap[12] = "W"
	directionMap[13] = "WNW"
	directionMap[14] = "NW"
	directionMap[15] = "NNW"

	index := int(direction) % 16
	directionCardinal := directionMap[index]

	windDirection := WindDirection{
		Degrees:  degreesFloat,
		Cardinal: directionCardinal,
	}

	return windDirection
}

type WindSpeed struct {
	Mph float64
	Kph float64
}

func NewWind(speedInMph, gustsInMph float64, directionDegrees int) Wind {

	speed := NewWindSpeedFromMph(speedInMph)
	gusts := NewWindSpeedFromMph(gustsInMph)
	direction := NewWindDirection(directionDegrees)
	return Wind{
		Speed:     speed,
		Gusts:     gusts,
		Direction: direction,
	}
}
