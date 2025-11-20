package weather

import (
	"medi-snow/internal/types"
	"time"
)

const (
	TimezoneDenver = "America/Denver"
)

// Weather model names
const (
	ModelGemSeamless        = "GemSeamless"
	ModelEcmwIfs            = "EcmwIfs"
	ModelGfsSeamless        = "GfsSeamless"
	ModelNcepNbmConus       = "NcepNbmConus"
	ModelGfsGraphcast025    = "GfsGraphcast025"
	ModelEcmwfAifs025Single = "EcmwfAifs025Single"
	ModelNcepNamConus       = "NcepNamConus"
)

// ModelValues maps weather model names to their values
type ModelValues[T any] map[string]T

// GetForModel retrieves the value for a specific weather model
func (w ModelValues[T]) GetForModel(model string) (T, bool) {
	val, ok := w[model]
	return val, ok
}

// Models returns a slice of all model names in the map
func (w ModelValues[T]) Models() []string {
	models := make([]string, 0, len(w))
	for model := range w {
		models = append(models, model)
	}
	return models
}

// HasModel checks if a model exists in the map
func (w ModelValues[T]) HasModel(model string) bool {
	_, ok := w[model]
	return ok
}

type Forecast struct {
	Timestamp         time.Time
	ForecastPoint     types.ForecastPoint
	Timezone          string
	PrimaryModel      string
	CurrentConditions CurrentConditions
	DailyForecasts    []DailyForecast
}

type CurrentConditions struct {
	Temperature      ModelValues[types.Temperature]
	Weather          ModelValues[types.Weather]
	Wind             ModelValues[types.Wind]
	Visibility       ModelValues[float64]
	CloudCover       ModelValues[float64]
	RelativeHumidity ModelValues[float64]
	CloudCoverLow    ModelValues[float64]
	CloudCoverMid    ModelValues[float64]
	CloudCoverHigh   ModelValues[float64]
}

type DailyForecast struct {
	Timestamp       time.Time
	HourlyForecasts []HourlyForecast

	SnowfallWaterEquivalentSum ModelValues[float64]
	Weather                    ModelValues[types.Weather]
	Sunrise                    ModelValues[time.Time]
	Sunset                     ModelValues[time.Time]
	WindDominantDirection      ModelValues[int]

	HighestFreezingLevelHeight ModelValues[float64]
	LowestFreezingLevelHeight  ModelValues[float64]
	HighTemperature            ModelValues[types.Temperature]
	LowTemperature             ModelValues[types.Temperature]
	TotalPrecipitation         ModelValues[types.Precipitation]
	TotalRain                  ModelValues[types.Precipitation]
	TotalShowers               ModelValues[types.Precipitation]
	TotalSnowfall              ModelValues[types.Precipitation]
	TotalLiquidPrecipitation   ModelValues[types.Precipitation]
	MaxWindSpeed               ModelValues[float64]
	MinWindSpeed               ModelValues[float64]
	MaxWindGusts               ModelValues[float64]
	MinWindGusts               ModelValues[float64]
}

// TODO openmeteo precip note: Some variables like precipitation are calculated from the preceding hour as an average or sum.
type HourlyForecast struct {
	Start                    time.Time
	End                      time.Time
	FreezingLevelHeight      ModelValues[float64]
	IsDay                    ModelValues[bool]
	Weather                  ModelValues[bool]
	Temperature              ModelValues[types.Temperature]
	ApparentTemperature      ModelValues[types.Temperature]
	PrecipitationProbability ModelValues[float64]
	Precipitation            ModelValues[types.Precipitation]
	CloudCover               ModelValues[float64]
	CloudCoverLow            ModelValues[float64]
	CloudCoverMid            ModelValues[float64]
	CloudCoverHigh           ModelValues[float64]
	Visibility               ModelValues[float64]
	Wind                     ModelValues[types.Wind]
	RelativeHumidity         ModelValues[float64]
	Rain                     ModelValues[types.Precipitation]
	Showers                  ModelValues[types.Precipitation]
	Snowfall                 ModelValues[types.Precipitation]
	SnowDepth                ModelValues[types.SnowDepth]

	// Sum of Rain and Showers
	LiquidPrecipitation ModelValues[types.Precipitation]
}
