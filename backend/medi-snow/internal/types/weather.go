package types

// WeatherCode represents a WMO weather code
type WeatherCode int

// Weather represents weather conditions with a code and description
type Weather struct {
	Code        int    `json:"code"`
	Description string `json:"description"`
}

// Weather code constants
const (
	ClearSky                     WeatherCode = 0
	MainlyClear                  WeatherCode = 1
	PartlyCloudy                 WeatherCode = 2
	Overcast                     WeatherCode = 3
	Fog                          WeatherCode = 45
	DepositingRimeFog            WeatherCode = 48
	DrizzleLight                 WeatherCode = 51
	DrizzleModerate              WeatherCode = 53
	DrizzleDense                 WeatherCode = 55
	FreezingDrizzleLight         WeatherCode = 56
	FreezingDrizzleDense         WeatherCode = 57
	RainSlight                   WeatherCode = 61
	RainModerate                 WeatherCode = 63
	RainHeavy                    WeatherCode = 65
	FreezingRainLight            WeatherCode = 66
	FreezingRainHeavy            WeatherCode = 67
	SnowFallSlight               WeatherCode = 71
	SnowFallModerate             WeatherCode = 73
	SnowFallHeavy                WeatherCode = 75
	SnowGrains                   WeatherCode = 77
	RainShowersSlight            WeatherCode = 80
	RainShowersModerate          WeatherCode = 81
	RainShowersViolent           WeatherCode = 82
	SnowShowersSlight            WeatherCode = 85
	SnowShowersHeavy             WeatherCode = 86
	ThunderstormSlightOrModerate WeatherCode = 95
	ThunderstormWithSlightHail   WeatherCode = 96
	ThunderstormWithHeavyHail    WeatherCode = 99
)

// weatherDescriptions maps weather codes to their descriptions
var weatherDescriptions = map[int]string{
	0:  "Clear sky",
	1:  "Mainly clear",
	2:  "Partly cloudy",
	3:  "Overcast",
	45: "Fog",
	48: "Depositing rime fog",
	51: "Drizzle: Light intensity",
	53: "Drizzle: Moderate intensity",
	55: "Drizzle: Dense intensity",
	56: "Freezing Drizzle: Light intensity",
	57: "Freezing Drizzle: Dense intensity",
	61: "Rainfall: Slight intensity",
	63: "Rainfall: Moderate intensity",
	65: "Rainfall: Heavy intensity",
	66: "Freezing Rainfall: Light intensity",
	67: "Freezing Rainfall: Heavy intensity",
	71: "Snow fall: Slight intensity",
	73: "Snow fall: Moderate intensity",
	75: "Snow fall: Heavy intensity",
	77: "Snow grains",
	80: "Rainfall showers: Slight",
	81: "Rainfall showers: Moderate",
	82: "Rainfall showers: Violent",
	85: "Snow showers: Slight",
	86: "Snow showers: Heavy",
	95: "Thunderstorm: Slight or moderate",
	96: "Thunderstorm with slight hail",
	99: "Thunderstorm with heavy hail",
}

// GetWeatherDescription returns the description for a given weather code
func GetWeatherDescription(code int) string {
	if desc, ok := weatherDescriptions[code]; ok {
		return desc
	}
	return "Unknown"
}

// NewWeather creates a Weather instance from a weather code
func NewWeather(code int) Weather {
	return Weather{
		Code:        code,
		Description: GetWeatherDescription(code),
	}
}
