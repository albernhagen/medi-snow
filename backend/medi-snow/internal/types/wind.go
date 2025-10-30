package types

const MphToKph = 1.60934

type Wind struct {
	SpeedInMph        float64
	SpeedInKph        float64
	GustsInMph        float64
	GustsInKph        float64
	DirectionDegrees  float64
	DirectionCardinal string
}

func NewWindFromMph(speedInMph, gustsInMph, directionDegrees float64) Wind {
	direction := (directionDegrees / 22.5) + .5 // .5 for rounding
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

	return Wind{
		SpeedInMph:        speedInMph,
		SpeedInKph:        speedInMph * MphToKph,
		GustsInMph:        gustsInMph,
		GustsInKph:        gustsInMph * MphToKph,
		DirectionDegrees:  directionDegrees,
		DirectionCardinal: directionCardinal,
	}
}
