package weather

import (
	"fmt"
	"log/slog"
	"medi-snow/internal/config"
	"medi-snow/internal/providers/nws"
	"medi-snow/internal/providers/openmeteo"
	"medi-snow/internal/timezone"
	"medi-snow/internal/types"
	"time"
)

type ForecastProvider interface {
	// GetForecast fetches the weather forecast for the given latitude, longitude, elevation, and timezone
	GetForecast(latitude, longitude, elevationMeters float64, forecastDays int, timezone string) (*openmeteo.ForecastAPIResponse, error)
}

type ForecastDiscussionProvider interface {
	GetPoint(latitude, longitude float64) (*nws.PointAPIResponse, error)
	GetAreaForecastDiscussion(locationId string) (*nws.AFDAPIResponse, error)
}

type Service interface {
	GetForecast(point types.ForecastPoint) (*Forecast, error)
}

type weatherService struct {
	forecastProvider           ForecastProvider
	forecastDiscussionProvider ForecastDiscussionProvider
	timezoneService            timezone.Service
	cfg                        *config.Config
	logger                     *slog.Logger
}

func NewWeatherService(config *config.Config, logger *slog.Logger) (Service, error) {
	tzSvc, err := timezone.NewService()
	if err != nil {
		return nil, fmt.Errorf("failed to create timezone service: %w", err)
	}
	return NewWeatherServiceWithProvider(openmeteo.NewClient(logger), nws.NewClient(logger), tzSvc, config, logger), nil
}

func NewWeatherServiceWithProvider(
	forecastProvider ForecastProvider,
	forecastDiscussionProvider ForecastDiscussionProvider,
	timezoneService timezone.Service,
	cfg *config.Config,
	logger *slog.Logger,
) Service {
	return &weatherService{
		forecastProvider:           forecastProvider,
		forecastDiscussionProvider: forecastDiscussionProvider,
		timezoneService:            timezoneService,
		cfg:                        cfg,
		logger:                     logger.With("component", "weather-service"),
	}
}

func (s *weatherService) GetForecast(forecastPoint types.ForecastPoint) (*Forecast, error) {
	// TODO validate forecastPoint data
	forecastDays := s.cfg.App.ForecastDays

	// TODO improve model selection logic and coordination
	primaryModel := ModelGfsSeamless

	// Look up timezone for the location
	tz, err := s.timezoneService.GetTimezone(forecastPoint.Coordinates.Latitude, forecastPoint.Coordinates.Longitude)
	if err != nil {
		s.logger.Error("failed to determine timezone",
			"latitude", forecastPoint.Coordinates.Latitude,
			"longitude", forecastPoint.Coordinates.Longitude,
			"error", err,
		)
		return nil, fmt.Errorf("failed to determine timezone: %w", err)
	}

	s.logger.Debug("determined timezone for location",
		"latitude", forecastPoint.Coordinates.Latitude,
		"longitude", forecastPoint.Coordinates.Longitude,
		"timezone", tz,
	)

	// Get forecast with timezone
	apiResponse, err := s.forecastProvider.GetForecast(
		forecastPoint.Coordinates.Latitude,
		forecastPoint.Coordinates.Longitude,
		forecastPoint.Elevation.Meters,
		forecastDays,
		tz,
	)
	if err != nil {
		s.logger.Error("failed to get forecast from provider", "error", err)
		return nil, fmt.Errorf("failed to get forecast: %w", err)
	}

	return mapForecastAPIResponseToForecast(forecastPoint, primaryModel, apiResponse)
}

func (s *weatherService) GetForecastDiscussion(forecastPoint types.ForecastPoint) (string, error) {
	// Get point data from NWS
	pointResp, err := s.forecastDiscussionProvider.GetPoint(
		forecastPoint.Coordinates.Latitude,
		forecastPoint.Coordinates.Longitude,
	)
	if err != nil {
		s.logger.Error("failed to get NWS point data",
			"latitude", forecastPoint.Coordinates.Latitude,
			"longitude", forecastPoint.Coordinates.Longitude,
			"error", err,
		)
		return "", fmt.Errorf("failed to get NWS point data: %w", err)
	}

	// Get area forecast discussion using location ID
	locationId := pointResp.Properties.Cwa
	afdResp, err := s.forecastDiscussionProvider.GetAreaForecastDiscussion(locationId)
	if err != nil {
		s.logger.Error("failed to get NWS area forecast discussion",
			"location_id", locationId,
			"error", err,
		)
		return "", fmt.Errorf("failed to get NWS area forecast discussion: %w", err)
	}

	return afdResp.ProductText, nil
}

func mapForecastAPIResponseToForecast(forecastPoint types.ForecastPoint, primaryModel string, apiResponse *openmeteo.ForecastAPIResponse) (*Forecast, error) {

	// TODO validate response data
	forecast := &Forecast{
		Timestamp:     time.Now().UTC(),
		ForecastPoint: forecastPoint,
		Timezone:      apiResponse.Timezone,
		PrimaryModel:  primaryModel,
	}

	// Daily starts at today
	// Hourly starts at 00:00 today
	// Get current time for the supplied timezone like "America/Denver"
	location, err := time.LoadLocation(apiResponse.Timezone)
	if err != nil {
		return nil, fmt.Errorf("failed to load timezone location %s: %w", apiResponse.Timezone, err)
	}
	currentTime := time.Now().In(location)

	// We want the index in the array of the most recent timestamp that is earlier than now
	nowIndex := 0
	for i, t := range apiResponse.Hourly.Time {
		parsedTime, err := time.ParseInLocation("2006-01-02T15:04", t, location)
		if err != nil {
			continue
		}

		if parsedTime.After(currentTime) {
			break
		}

		nowIndex = i
	}

	currentConditions := CurrentConditions{
		Temperature: ModelValues[types.Temperature]{
			ModelGfsSeamless:        types.NewTemperatureFromFahrenheit(apiResponse.Hourly.Temperature2MGfsSeamless[nowIndex]),
			ModelGemSeamless:        types.NewTemperatureFromFahrenheit(apiResponse.Hourly.Temperature2MGemSeamless[nowIndex]),
			ModelEcmwIfs:            types.NewTemperatureFromFahrenheit(apiResponse.Hourly.Temperature2MEcmwfIfs[nowIndex]),
			ModelNcepNbmConus:       types.NewTemperatureFromFahrenheit(apiResponse.Hourly.Temperature2MNcepNbmConus[nowIndex]),
			ModelGfsGraphcast025:    types.NewTemperatureFromFahrenheit(apiResponse.Hourly.Temperature2MGfsGraphcast025[nowIndex]),
			ModelEcmwfAifs025Single: types.NewTemperatureFromFahrenheit(apiResponse.Hourly.Temperature2MEcmwfAifs025Single[nowIndex]),
			ModelNcepNamConus:       types.NewTemperatureFromFahrenheit(apiResponse.Hourly.Temperature2MNcepNamConus[nowIndex]),
		},
		Weather: ModelValues[types.Weather]{
			ModelGfsSeamless:        types.NewWeather(apiResponse.Hourly.WeatherCodeGfsSeamless[nowIndex]),
			ModelGemSeamless:        types.NewWeather(apiResponse.Hourly.WeatherCodeGemSeamless[nowIndex]),
			ModelEcmwIfs:            types.NewWeather(apiResponse.Hourly.WeatherCodeEcmwfIfs[nowIndex]),
			ModelNcepNbmConus:       types.NewWeather(apiResponse.Hourly.WeatherCodeNcepNbmConus[nowIndex]),
			ModelGfsGraphcast025:    types.NewWeather(apiResponse.Hourly.WeatherCodeGfsGraphcast025[nowIndex]),
			ModelEcmwfAifs025Single: types.NewWeather(apiResponse.Hourly.WeatherCodeEcmwfAifs025Single[nowIndex]),
			ModelNcepNamConus:       types.NewWeather(apiResponse.Hourly.WeatherCodeNcepNamConus[nowIndex]),
		},
		Wind: ModelValues[types.Wind]{
			ModelGfsSeamless:        types.NewWind(apiResponse.Hourly.WindSpeed10MGfsSeamless[nowIndex], apiResponse.Hourly.WindGusts10MGfsSeamless[nowIndex], apiResponse.Hourly.WindDirection10MGfsSeamless[nowIndex]),
			ModelGemSeamless:        types.NewWind(apiResponse.Hourly.WindSpeed10MGemSeamless[nowIndex], apiResponse.Hourly.WindGusts10MGemSeamless[nowIndex], apiResponse.Hourly.WindDirection10MGemSeamless[nowIndex]),
			ModelEcmwIfs:            types.NewWind(apiResponse.Hourly.WindSpeed10MEcmwfIfs[nowIndex], apiResponse.Hourly.WindGusts10MEcmwfIfs[nowIndex], apiResponse.Hourly.WindDirection10MEcmwfIfs[nowIndex]),
			ModelNcepNbmConus:       types.NewWind(apiResponse.Hourly.WindSpeed10MNcepNbmConus[nowIndex], apiResponse.Hourly.WindGusts10MNcepNbmConus[nowIndex], apiResponse.Hourly.WindDirection10MNcepNbmConus[nowIndex]),
			ModelGfsGraphcast025:    types.NewWind(apiResponse.Hourly.WindSpeed10MGfsGraphcast025[nowIndex], -1, apiResponse.Hourly.WindDirection10MGfsGraphcast025[nowIndex]),
			ModelEcmwfAifs025Single: types.NewWind(apiResponse.Hourly.WindSpeed10MEcmwfAifs025Single[nowIndex], -1, apiResponse.Hourly.WindDirection10MEcmwfAifs025Single[nowIndex]),
			ModelNcepNamConus:       types.NewWind(apiResponse.Hourly.WindSpeed10MNcepNamConus[nowIndex], apiResponse.Hourly.WindGusts10MNcepNamConus[nowIndex], apiResponse.Hourly.WindDirection10MNcepNamConus[nowIndex]),
		},
		Visibility: ModelValues[float64]{
			ModelGfsSeamless:  apiResponse.Hourly.VisibilityGfsSeamless[nowIndex],
			ModelEcmwIfs:      apiResponse.Hourly.VisibilityEcmwfIfs[nowIndex],
			ModelNcepNbmConus: apiResponse.Hourly.VisibilityNcepNbmConus[nowIndex],
			ModelNcepNamConus: apiResponse.Hourly.VisibilityNcepNamConus[nowIndex],
			// No data
			// ModelGemSeamless: 		apiResponse.Hourly.VisibilityGemSeamless[nowIndex],
			// ModelGfsGraphcast025:    apiResponse.Hourly.VisibilityGfsGraphcast025[nowIndex],
			// ModelEcmwfAifs025Single: apiResponse.Hourly.VisibilityEcmwfAifs025Single[nowIndex],
		},
		CloudCover: ModelValues[float64]{
			ModelGfsSeamless:        toPercentage(apiResponse.Hourly.CloudCoverGfsSeamless[nowIndex]),
			ModelGemSeamless:        toPercentage(apiResponse.Hourly.CloudCoverGemSeamless[nowIndex]),
			ModelEcmwIfs:            toPercentage(apiResponse.Hourly.CloudCoverEcmwfIfs[nowIndex]),
			ModelNcepNbmConus:       toPercentage(apiResponse.Hourly.CloudCoverNcepNbmConus[nowIndex]),
			ModelGfsGraphcast025:    toPercentage(apiResponse.Hourly.CloudCoverGfsGraphcast025[nowIndex]),
			ModelEcmwfAifs025Single: toPercentage(apiResponse.Hourly.CloudCoverEcmwfAifs025Single[nowIndex]),
			ModelNcepNamConus:       toPercentage(apiResponse.Hourly.CloudCoverNcepNamConus[nowIndex]),
		},
		RelativeHumidity: ModelValues[float64]{
			ModelGfsSeamless:        toPercentage(apiResponse.Hourly.RelativeHumidity2MGfsSeamless[nowIndex]),
			ModelGemSeamless:        toPercentage(apiResponse.Hourly.RelativeHumidity2MGemSeamless[nowIndex]),
			ModelEcmwIfs:            toPercentage(apiResponse.Hourly.RelativeHumidity2MEcmwfIfs[nowIndex]),
			ModelNcepNbmConus:       toPercentage(apiResponse.Hourly.RelativeHumidity2MNcepNbmConus[nowIndex]),
			ModelEcmwfAifs025Single: toPercentage(apiResponse.Hourly.RelativeHumidity2MEcmwfAifs025Single[nowIndex]),
			ModelNcepNamConus:       toPercentage(apiResponse.Hourly.RelativeHumidity2MNcepNamConus[nowIndex]),
			// No data
			// ModelGfsGraphcast025:    toPercentage(apiResponse.Hourly.RelativeHumidity2MGfsGraphcast025[nowIndex]),
		},
		CloudCoverLow: ModelValues[float64]{
			ModelGfsSeamless:        toPercentage(apiResponse.Hourly.CloudCoverLowGfsSeamless[nowIndex]),
			ModelGemSeamless:        toPercentage(apiResponse.Hourly.CloudCoverLowGemSeamless[nowIndex]),
			ModelEcmwIfs:            toPercentage(apiResponse.Hourly.CloudCoverLowEcmwfIfs[nowIndex]),
			ModelGfsGraphcast025:    toPercentage(apiResponse.Hourly.CloudCoverLowGfsGraphcast025[nowIndex]),
			ModelEcmwfAifs025Single: toPercentage(apiResponse.Hourly.CloudCoverLowEcmwfAifs025Single[nowIndex]),
			ModelNcepNamConus:       toPercentage(apiResponse.Hourly.CloudCoverLowNcepNamConus[nowIndex]),
			// No data
			// ModelNcepNbmConus:       toPercentage(apiResponse.Hourly.CloudCoverLowNcepNbmConus[nowIndex]),
		},
		CloudCoverMid: ModelValues[float64]{
			ModelGfsSeamless:        toPercentage(apiResponse.Hourly.CloudCoverMidGfsSeamless[nowIndex]),
			ModelGemSeamless:        toPercentage(apiResponse.Hourly.CloudCoverMidGemSeamless[nowIndex]),
			ModelEcmwIfs:            toPercentage(apiResponse.Hourly.CloudCoverMidEcmwfIfs[nowIndex]),
			ModelGfsGraphcast025:    toPercentage(apiResponse.Hourly.CloudCoverMidGfsGraphcast025[nowIndex]),
			ModelEcmwfAifs025Single: toPercentage(apiResponse.Hourly.CloudCoverMidEcmwfAifs025Single[nowIndex]),
			ModelNcepNamConus:       toPercentage(apiResponse.Hourly.CloudCoverMidNcepNamConus[nowIndex]),
			// No data
			// ModelNcepNbmConus:       toPercentage(apiResponse.Hourly.CloudCoverMidNcepNbmConus[nowIndex]),
		},
		CloudCoverHigh: ModelValues[float64]{
			ModelGfsSeamless:        toPercentage(apiResponse.Hourly.CloudCoverHighGfsSeamless[nowIndex]),
			ModelGemSeamless:        toPercentage(apiResponse.Hourly.CloudCoverHighGemSeamless[nowIndex]),
			ModelEcmwIfs:            toPercentage(apiResponse.Hourly.CloudCoverHighEcmwfIfs[nowIndex]),
			ModelGfsGraphcast025:    toPercentage(apiResponse.Hourly.CloudCoverHighGfsGraphcast025[nowIndex]),
			ModelEcmwfAifs025Single: toPercentage(apiResponse.Hourly.CloudCoverHighEcmwfAifs025Single[nowIndex]),
			ModelNcepNamConus:       toPercentage(apiResponse.Hourly.CloudCoverHighNcepNamConus[nowIndex]),
			// No data
			// ModelNcepNbmConus:       toPercentage(apiResponse.Hourly.CloudCoverHighNcepNbmConus[nowIndex]),
		},
	}

	forecast.CurrentConditions = currentConditions

	dailyForecasts := make([]DailyForecast, 0, len(apiResponse.Daily.Time))
	hourlyIndex := 0

	// Get each daily forecast
	for i, day := range apiResponse.Daily.Time {

		dailyForecastHourlyIndexes := make([]int, 24)

		dayTime, err := time.ParseInLocation("2006-01-02", day, location)
		if err != nil {
			continue
		}

		hourlyForecasts := make([]HourlyForecast, 0)
		for j := hourlyIndex; j < len(apiResponse.Hourly.Time); j++ {
			hourlyTime, err := time.ParseInLocation("2006-01-02T15:04", apiResponse.Hourly.Time[j], location)
			if err != nil {
				continue
			}

			// Check if hourly time is within the current day
			if hourlyTime.Year() == dayTime.Year() && hourlyTime.Month() == dayTime.Month() && hourlyTime.Day() == dayTime.Day() {
				hourlyIndex = j

				start, startErr := time.ParseInLocation("2006-01-02T15:04", apiResponse.Hourly.Time[j], location)
				if startErr != nil {
					return nil, fmt.Errorf("failed to parse hourly time: %w", startErr)
				}

				end := start.Add(1 * time.Hour)

				// TODO construct hourly forecast
				hourlyForecast := HourlyForecast{
					Start: start,
					End:   end,
					FreezingLevelHeight: map[string]float64{
						ModelGfsSeamless: apiResponse.Hourly.FreezingLevelHeightGfsSeamless[j],
					},
					IsDay: ModelValues[bool]{
						ModelGfsSeamless:        apiResponse.Hourly.IsDayGfsSeamless[j] == 1,
						ModelGemSeamless:        apiResponse.Hourly.IsDayGemSeamless[j] == 1,
						ModelEcmwIfs:            apiResponse.Hourly.IsDayEcmwfIfs[j] == 1,
						ModelNcepNbmConus:       apiResponse.Hourly.IsDayNcepNbmConus[j] == 1,
						ModelGfsGraphcast025:    apiResponse.Hourly.IsDayGfsGraphcast025[j] == 1,
						ModelEcmwfAifs025Single: apiResponse.Hourly.IsDayEcmwfAifs025Single[j] == 1,
						ModelNcepNamConus:       apiResponse.Hourly.IsDayNcepNamConus[j] == 1,
					},
					Weather: ModelValues[types.Weather]{
						ModelGfsSeamless:        types.NewWeather(apiResponse.Hourly.WeatherCodeGfsSeamless[j]),
						ModelGemSeamless:        types.NewWeather(apiResponse.Hourly.WeatherCodeGemSeamless[j]),
						ModelEcmwIfs:            types.NewWeather(apiResponse.Hourly.WeatherCodeEcmwfIfs[j]),
						ModelNcepNbmConus:       types.NewWeather(apiResponse.Hourly.WeatherCodeNcepNbmConus[j]),
						ModelGfsGraphcast025:    types.NewWeather(apiResponse.Hourly.WeatherCodeGfsGraphcast025[j]),
						ModelEcmwfAifs025Single: types.NewWeather(apiResponse.Hourly.WeatherCodeEcmwfAifs025Single[j]),
						ModelNcepNamConus:       types.NewWeather(apiResponse.Hourly.WeatherCodeNcepNamConus[j]),
					},
					Temperature: ModelValues[types.Temperature]{
						ModelGfsSeamless:        types.NewTemperatureFromFahrenheit(apiResponse.Hourly.Temperature2MGfsSeamless[j]),
						ModelGemSeamless:        types.NewTemperatureFromFahrenheit(apiResponse.Hourly.Temperature2MGemSeamless[j]),
						ModelEcmwIfs:            types.NewTemperatureFromFahrenheit(apiResponse.Hourly.Temperature2MEcmwfIfs[j]),
						ModelNcepNbmConus:       types.NewTemperatureFromFahrenheit(apiResponse.Hourly.Temperature2MNcepNbmConus[j]),
						ModelGfsGraphcast025:    types.NewTemperatureFromFahrenheit(apiResponse.Hourly.Temperature2MGfsGraphcast025[j]),
						ModelEcmwfAifs025Single: types.NewTemperatureFromFahrenheit(apiResponse.Hourly.Temperature2MEcmwfAifs025Single[j]),
						ModelNcepNamConus:       types.NewTemperatureFromFahrenheit(apiResponse.Hourly.Temperature2MNcepNamConus[j]),
					},
					ApparentTemperature: ModelValues[types.Temperature]{
						ModelGfsSeamless:        types.NewTemperatureFromFahrenheit(apiResponse.Hourly.ApparentTemperatureGfsSeamless[j]),
						ModelGemSeamless:        types.NewTemperatureFromFahrenheit(apiResponse.Hourly.ApparentTemperatureGemSeamless[j]),
						ModelEcmwIfs:            types.NewTemperatureFromFahrenheit(apiResponse.Hourly.ApparentTemperatureEcmwfIfs[j]),
						ModelNcepNbmConus:       types.NewTemperatureFromFahrenheit(apiResponse.Hourly.ApparentTemperatureNcepNbmConus[j]),
						ModelEcmwfAifs025Single: types.NewTemperatureFromFahrenheit(apiResponse.Hourly.ApparentTemperatureEcmwfAifs025Single[j]),
						ModelNcepNamConus:       types.NewTemperatureFromFahrenheit(apiResponse.Hourly.ApparentTemperatureNcepNamConus[j]),
						// No data
						// ModelGfsGraphcast025:    types.NewTemperatureFromFahrenheit(apiResponse.Hourly.ApparentTemperatureGfsGraphcast025[j]),
					},
					PrecipitationProbability: ModelValues[float64]{
						ModelGfsSeamless:  toPercentage(apiResponse.Hourly.PrecipitationProbabilityGfsSeamless[j]),
						ModelGemSeamless:  toPercentage(apiResponse.Hourly.PrecipitationProbabilityGemSeamless[j]),
						ModelEcmwIfs:      toPercentage(apiResponse.Hourly.PrecipitationProbabilityEcmwfIfs[j]),
						ModelNcepNbmConus: toPercentage(apiResponse.Hourly.PrecipitationProbabilityNcepNbmConus[j]),
						// No data
						// ModelGfsGraphcast025:    toPercentage(apiResponse.Hourly.PrecipitationProbabilityGfsGraphcast025[j]),
						// ModelEcmwfAifs025Single: toPercentage(apiResponse.Hourly.PrecipitationProbabilityEcmwfAifs025Single[j]),
						// ModelNcepNamConus:       toPercentage(apiResponse.Hourly.PrecipitationProbabilityNcepNamConus[j])
					},
					Precipitation: ModelValues[types.Precipitation]{
						ModelGfsSeamless:        types.NewPrecipitationFromInches(apiResponse.Hourly.PrecipitationGfsSeamless[j]),
						ModelGemSeamless:        types.NewPrecipitationFromInches(apiResponse.Hourly.PrecipitationGemSeamless[j]),
						ModelEcmwIfs:            types.NewPrecipitationFromInches(apiResponse.Hourly.PrecipitationEcmwfIfs[j]),
						ModelNcepNbmConus:       types.NewPrecipitationFromInches(apiResponse.Hourly.PrecipitationNcepNbmConus[j]),
						ModelGfsGraphcast025:    types.NewPrecipitationFromInches(apiResponse.Hourly.PrecipitationGfsGraphcast025[j]),
						ModelEcmwfAifs025Single: types.NewPrecipitationFromInches(apiResponse.Hourly.PrecipitationEcmwfAifs025Single[j]),
						ModelNcepNamConus:       types.NewPrecipitationFromInches(apiResponse.Hourly.PrecipitationNcepNamConus[j]),
					},
					CloudCover: ModelValues[float64]{
						ModelGfsSeamless:        toPercentage(apiResponse.Hourly.CloudCoverGfsSeamless[j]),
						ModelGemSeamless:        toPercentage(apiResponse.Hourly.CloudCoverGemSeamless[j]),
						ModelEcmwIfs:            toPercentage(apiResponse.Hourly.CloudCoverEcmwfIfs[j]),
						ModelNcepNbmConus:       toPercentage(apiResponse.Hourly.CloudCoverNcepNbmConus[j]),
						ModelGfsGraphcast025:    toPercentage(apiResponse.Hourly.CloudCoverGfsGraphcast025[j]),
						ModelEcmwfAifs025Single: toPercentage(apiResponse.Hourly.CloudCoverEcmwfAifs025Single[j]),
						ModelNcepNamConus:       toPercentage(apiResponse.Hourly.CloudCoverNcepNamConus[j]),
					},
					CloudCoverLow: ModelValues[float64]{
						ModelGfsSeamless:        toPercentage(apiResponse.Hourly.CloudCoverLowGfsSeamless[j]),
						ModelGemSeamless:        toPercentage(apiResponse.Hourly.CloudCoverLowGemSeamless[j]),
						ModelEcmwIfs:            toPercentage(apiResponse.Hourly.CloudCoverLowEcmwfIfs[j]),
						ModelGfsGraphcast025:    toPercentage(apiResponse.Hourly.CloudCoverLowGfsGraphcast025[j]),
						ModelEcmwfAifs025Single: toPercentage(apiResponse.Hourly.CloudCoverLowEcmwfAifs025Single[j]),
						ModelNcepNamConus:       toPercentage(apiResponse.Hourly.CloudCoverLowNcepNamConus[j]),
						// No data
						// ModelNcepNbmConus:       toPercentage(apiResponse.Hourly.CloudCoverLowNcepNbmConus[j]),
					},
					CloudCoverMid: ModelValues[float64]{
						ModelGfsSeamless:        toPercentage(apiResponse.Hourly.CloudCoverMidGfsSeamless[j]),
						ModelGemSeamless:        toPercentage(apiResponse.Hourly.CloudCoverMidGemSeamless[j]),
						ModelEcmwIfs:            toPercentage(apiResponse.Hourly.CloudCoverMidEcmwfIfs[j]),
						ModelGfsGraphcast025:    toPercentage(apiResponse.Hourly.CloudCoverMidGfsGraphcast025[j]),
						ModelEcmwfAifs025Single: toPercentage(apiResponse.Hourly.CloudCoverMidEcmwfAifs025Single[j]),
						ModelNcepNamConus:       toPercentage(apiResponse.Hourly.CloudCoverMidNcepNamConus[j]),
						// No data
						// ModelNcepNbmConus:       toPercentage(apiResponse.Hourly.CloudCoverMidNcepNbmConus[j]),
					},
					CloudCoverHigh: ModelValues[float64]{
						ModelGfsSeamless:        toPercentage(apiResponse.Hourly.CloudCoverHighGfsSeamless[j]),
						ModelGemSeamless:        toPercentage(apiResponse.Hourly.CloudCoverHighGemSeamless[j]),
						ModelEcmwIfs:            toPercentage(apiResponse.Hourly.CloudCoverHighEcmwfIfs[j]),
						ModelGfsGraphcast025:    toPercentage(apiResponse.Hourly.CloudCoverHighGfsGraphcast025[j]),
						ModelEcmwfAifs025Single: toPercentage(apiResponse.Hourly.CloudCoverHighEcmwfAifs025Single[j]),
						ModelNcepNamConus:       toPercentage(apiResponse.Hourly.CloudCoverHighNcepNamConus[j]),
						// No data
						// ModelNcepNbmConus:       toPercentage(apiResponse.Hourly.CloudCoverHighNcepNbmConus[j]),
					},
					Visibility: ModelValues[float64]{
						ModelGfsSeamless:  apiResponse.Hourly.VisibilityGfsSeamless[j],
						ModelEcmwIfs:      apiResponse.Hourly.VisibilityEcmwfIfs[j],
						ModelNcepNbmConus: apiResponse.Hourly.VisibilityNcepNbmConus[j],
						ModelNcepNamConus: apiResponse.Hourly.VisibilityNcepNamConus[j],
						// No data
						// ModelGemSeamless: 		apiResponse.Hourly.VisibilityGemSeamless[j],
						// ModelGfsGraphcast025:    apiResponse.Hourly.VisibilityGfsGraphcast025[j],
						// ModelEcmwfAifs025Single: apiResponse.Hourly.VisibilityEcmwfAifs025Single[j],
					},
					Wind: ModelValues[types.Wind]{
						ModelGfsSeamless:        types.NewWind(apiResponse.Hourly.WindSpeed10MGfsSeamless[j], apiResponse.Hourly.WindGusts10MGfsSeamless[j], apiResponse.Hourly.WindDirection10MGfsSeamless[j]),
						ModelGemSeamless:        types.NewWind(apiResponse.Hourly.WindSpeed10MGemSeamless[j], apiResponse.Hourly.WindGusts10MGemSeamless[j], apiResponse.Hourly.WindDirection10MGemSeamless[j]),
						ModelEcmwIfs:            types.NewWind(apiResponse.Hourly.WindSpeed10MEcmwfIfs[j], apiResponse.Hourly.WindGusts10MEcmwfIfs[j], apiResponse.Hourly.WindDirection10MEcmwfIfs[j]),
						ModelNcepNbmConus:       types.NewWind(apiResponse.Hourly.WindSpeed10MNcepNbmConus[j], apiResponse.Hourly.WindGusts10MNcepNbmConus[j], apiResponse.Hourly.WindDirection10MNcepNbmConus[j]),
						ModelGfsGraphcast025:    types.NewWind(apiResponse.Hourly.WindSpeed10MGfsGraphcast025[j], -1, apiResponse.Hourly.WindDirection10MGfsGraphcast025[j]),
						ModelEcmwfAifs025Single: types.NewWind(apiResponse.Hourly.WindSpeed10MEcmwfAifs025Single[j], -1, apiResponse.Hourly.WindDirection10MEcmwfAifs025Single[j]),
						ModelNcepNamConus:       types.NewWind(apiResponse.Hourly.WindSpeed10MNcepNamConus[j], apiResponse.Hourly.WindGusts10MNcepNamConus[j], apiResponse.Hourly.WindDirection10MNcepNamConus[j]),
					},
					RelativeHumidity: ModelValues[float64]{
						ModelGfsSeamless:        toPercentage(apiResponse.Hourly.RelativeHumidity2MGfsSeamless[j]),
						ModelGemSeamless:        toPercentage(apiResponse.Hourly.RelativeHumidity2MGemSeamless[j]),
						ModelEcmwIfs:            toPercentage(apiResponse.Hourly.RelativeHumidity2MEcmwfIfs[j]),
						ModelNcepNbmConus:       toPercentage(apiResponse.Hourly.RelativeHumidity2MNcepNbmConus[j]),
						ModelEcmwfAifs025Single: toPercentage(apiResponse.Hourly.RelativeHumidity2MEcmwfAifs025Single[j]),
						ModelNcepNamConus:       toPercentage(apiResponse.Hourly.RelativeHumidity2MNcepNamConus[j]),
						// No data
						// ModelGfsGraphcast025:    toPercentage(apiResponse.Hourly.RelativeHumidity2MGfsGraphcast025[j]),
					},
					Rainfall: ModelValues[types.Precipitation]{
						ModelGfsSeamless:        types.NewPrecipitationFromInches(apiResponse.Hourly.RainGfsSeamless[j]),
						ModelGemSeamless:        types.NewPrecipitationFromInches(apiResponse.Hourly.RainGemSeamless[j]),
						ModelEcmwIfs:            types.NewPrecipitationFromInches(apiResponse.Hourly.RainEcmwfIfs[j]),
						ModelNcepNbmConus:       types.NewPrecipitationFromInches(apiResponse.Hourly.RainNcepNbmConus[j]),
						ModelGfsGraphcast025:    types.NewPrecipitationFromInches(apiResponse.Hourly.RainGfsGraphcast025[j]),
						ModelEcmwfAifs025Single: types.NewPrecipitationFromInches(apiResponse.Hourly.RainEcmwfAifs025Single[j]),
						ModelNcepNamConus:       types.NewPrecipitationFromInches(apiResponse.Hourly.RainNcepNamConus[j]),
					},
					Snowfall: ModelValues[types.Precipitation]{
						ModelGfsSeamless:        types.NewPrecipitationFromInches(apiResponse.Hourly.SnowfallGfsSeamless[j]),
						ModelGemSeamless:        types.NewPrecipitationFromInches(apiResponse.Hourly.SnowfallGemSeamless[j]),
						ModelEcmwIfs:            types.NewPrecipitationFromInches(apiResponse.Hourly.SnowfallEcmwfIfs[j]),
						ModelNcepNbmConus:       types.NewPrecipitationFromInches(apiResponse.Hourly.SnowfallNcepNbmConus[j]),
						ModelGfsGraphcast025:    types.NewPrecipitationFromInches(apiResponse.Hourly.SnowfallGfsGraphcast025[j]),
						ModelEcmwfAifs025Single: types.NewPrecipitationFromInches(apiResponse.Hourly.SnowfallEcmwfAifs025Single[j]),
						ModelNcepNamConus:       types.NewPrecipitationFromInches(apiResponse.Hourly.SnowfallNcepNamConus[j]),
					},
					Showers: ModelValues[types.Precipitation]{
						ModelGfsSeamless:        types.NewPrecipitationFromInches(apiResponse.Hourly.ShowersGfsSeamless[j]),
						ModelGemSeamless:        types.NewPrecipitationFromInches(apiResponse.Hourly.ShowersGemSeamless[j]),
						ModelEcmwIfs:            types.NewPrecipitationFromInches(apiResponse.Hourly.ShowersEcmwfIfs[j]),
						ModelNcepNbmConus:       types.NewPrecipitationFromInches(apiResponse.Hourly.ShowersNcepNbmConus[j]),
						ModelGfsGraphcast025:    types.NewPrecipitationFromInches(apiResponse.Hourly.ShowersGfsGraphcast025[j]),
						ModelEcmwfAifs025Single: types.NewPrecipitationFromInches(apiResponse.Hourly.ShowersEcmwfAifs025Single[j]),
						ModelNcepNamConus:       types.NewPrecipitationFromInches(apiResponse.Hourly.ShowersNcepNamConus[j]),
					},
					SnowDepth: ModelValues[types.SnowDepth]{
						ModelGfsSeamless:  types.NewSnowDepthFromFeet(apiResponse.Hourly.SnowDepthGfsSeamless[j]),
						ModelGemSeamless:  types.NewSnowDepthFromFeet(apiResponse.Hourly.SnowDepthGemSeamless[j]),
						ModelEcmwIfs:      types.NewSnowDepthFromFeet(apiResponse.Hourly.SnowDepthEcmwfIfs[j]),
						ModelNcepNamConus: types.NewSnowDepthFromFeet(apiResponse.Hourly.SnowDepthNcepNamConus[j]),
						// No data
						// ModelNcepNbmConus:       apiResponse.Hourly.SnowDepthNcepNbmConus[j],
						// ModelGfsGraphcast025:    apiResponse.Hourly.SnowDepthGfsGraphcast025[j],
						// ModelEcmwfAifs025Single: apiResponse.Hourly.SnowDepthEcmwfAifs025Single[j],
					},
				}

				// Set liquid precipitation
				liquidPrecipitation := make(ModelValues[types.Precipitation], len(hourlyForecast.Precipitation))
				for model, rainfall := range hourlyForecast.Rainfall {
					showers := hourlyForecast.Showers[model]
					liquidPrecipitation[model] = types.NewPrecipitationFromInches(rainfall.Inches + showers.Inches)
				}
				hourlyForecast.LiquidPrecipitation = liquidPrecipitation

				hourlyForecasts = append(hourlyForecasts, hourlyForecast)
			} else {
				break
			}
		}

		var hourlySliceStart = dailyForecastHourlyIndexes[0]
		var hourlySliceEnd = dailyForecastHourlyIndexes[len(dailyForecastHourlyIndexes)-1]

		// TODO construct daily forecast
		dailyForecast := DailyForecast{
			HourlyForecasts: hourlyForecasts,
			Timestamp:       dayTime,
			Weather: ModelValues[types.Weather]{
				ModelGfsSeamless:        types.NewWeather(apiResponse.Daily.WeatherCodeGfsSeamless[i]),
				ModelGemSeamless:        types.NewWeather(apiResponse.Daily.WeatherCodeGemSeamless[i]),
				ModelEcmwIfs:            types.NewWeather(apiResponse.Daily.WeatherCodeEcmwfIfs[i]),
				ModelNcepNbmConus:       types.NewWeather(apiResponse.Daily.WeatherCodeNcepNbmConus[i]),
				ModelGfsGraphcast025:    types.NewWeather(apiResponse.Daily.WeatherCodeGfsGraphcast025[i]),
				ModelEcmwfAifs025Single: types.NewWeather(apiResponse.Daily.WeatherCodeEcmwfAifs025Single[i]),
				ModelNcepNamConus:       types.NewWeather(apiResponse.Daily.WeatherCodeNcepNamConus[i]),
			},
			SnowfallWaterEquivalentSum: ModelValues[float64]{
				ModelGfsSeamless:        apiResponse.Daily.SnowfallWaterEquivalentSumGfsSeamless[i],
				ModelGemSeamless:        apiResponse.Daily.SnowfallWaterEquivalentSumGemSeamless[i],
				ModelEcmwIfs:            apiResponse.Daily.SnowfallWaterEquivalentSumEcmwfIfs[i],
				ModelNcepNbmConus:       apiResponse.Daily.SnowfallWaterEquivalentSumNcepNbmConus[i],
				ModelEcmwfAifs025Single: apiResponse.Daily.SnowfallWaterEquivalentSumEcmwfAifs025Single[i],
				ModelNcepNamConus:       apiResponse.Daily.SnowfallWaterEquivalentSumNcepNamConus[i],
				// No data
				// ModelGfsGraphcast025:    apiResponse.Daily.SnowfallWaterEquivalentSumGfsGraphcast025[i],
			},
			Sunrise: ModelValues[time.Time]{
				ModelGfsSeamless:        toTime(apiResponse.Daily.SunriseGfsSeamless[i]),
				ModelGemSeamless:        toTime(apiResponse.Daily.SunriseGemSeamless[i]),
				ModelEcmwIfs:            toTime(apiResponse.Daily.SunriseEcmwfIfs[i]),
				ModelNcepNbmConus:       toTime(apiResponse.Daily.SunriseNcepNbmConus[i]),
				ModelGfsGraphcast025:    toTime(apiResponse.Daily.SunriseGfsGraphcast025[i]),
				ModelEcmwfAifs025Single: toTime(apiResponse.Daily.SunriseEcmwfAifs025Single[i]),
				ModelNcepNamConus:       toTime(apiResponse.Daily.SunriseNcepNamConus[i]),
			},
			Sunset: ModelValues[time.Time]{
				ModelGfsSeamless:        toTime(apiResponse.Daily.SunsetGfsSeamless[i]),
				ModelGemSeamless:        toTime(apiResponse.Daily.SunsetGemSeamless[i]),
				ModelEcmwIfs:            toTime(apiResponse.Daily.SunsetEcmwfIfs[i]),
				ModelNcepNbmConus:       toTime(apiResponse.Daily.SunsetNcepNbmConus[i]),
				ModelGfsGraphcast025:    toTime(apiResponse.Daily.SunsetGfsGraphcast025[i]),
				ModelEcmwfAifs025Single: toTime(apiResponse.Daily.SunsetEcmwfAifs025Single[i]),
				ModelNcepNamConus:       toTime(apiResponse.Daily.SunsetNcepNamConus[i]),
			},
			WindDominantDirection: ModelValues[types.WindDirection]{
				ModelGfsSeamless:        types.NewWindDirection(apiResponse.Daily.WindDirection10MDominantGfsSeamless[i]),
				ModelGemSeamless:        types.NewWindDirection(apiResponse.Daily.WindDirection10MDominantGemSeamless[i]),
				ModelEcmwIfs:            types.NewWindDirection(apiResponse.Daily.WindDirection10MDominantEcmwfIfs[i]),
				ModelNcepNbmConus:       types.NewWindDirection(apiResponse.Daily.WindDirection10MDominantNcepNbmConus[i]),
				ModelEcmwfAifs025Single: types.NewWindDirection(apiResponse.Daily.WindDirection10MDominantEcmwfAifs025Single[i]),
				ModelNcepNamConus:       types.NewWindDirection(apiResponse.Daily.WindDirection10MDominantNcepNamConus[i]),
				// No data
				// ModelGfsGraphcast025:    types.NewWindDirection(apiResponse.Daily.WindDirection10MDominantGfsGraphcast025[i]),
			},
			HighestFreezingLevelHeightFt: ModelValues[float64]{
				ModelGfsSeamless: maxFloat(apiResponse.Hourly.FreezingLevelHeightGfsSeamless[hourlySliceStart:hourlySliceEnd]),
			},
			LowestFreezingLevelHeightFt: ModelValues[float64]{
				ModelGfsSeamless: minFloat(apiResponse.Hourly.FreezingLevelHeightGfsSeamless[hourlySliceStart:hourlySliceEnd]),
			},
			HighTemperature: ModelValues[types.Temperature]{
				ModelGfsSeamless:        types.NewTemperatureFromFahrenheit(maxFloat(apiResponse.Hourly.Temperature2MGfsSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelGemSeamless:        types.NewTemperatureFromFahrenheit(maxFloat(apiResponse.Hourly.Temperature2MGemSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwIfs:            types.NewTemperatureFromFahrenheit(maxFloat(apiResponse.Hourly.Temperature2MEcmwfIfs[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNbmConus:       types.NewTemperatureFromFahrenheit(maxFloat(apiResponse.Hourly.Temperature2MNcepNbmConus[hourlySliceStart:hourlySliceEnd])),
				ModelGfsGraphcast025:    types.NewTemperatureFromFahrenheit(maxFloat(apiResponse.Hourly.Temperature2MGfsGraphcast025[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNamConus:       types.NewTemperatureFromFahrenheit(maxFloat(apiResponse.Hourly.Temperature2MNcepNamConus[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwfAifs025Single: types.NewTemperatureFromFahrenheit(maxFloat(apiResponse.Hourly.Temperature2MEcmwfAifs025Single[hourlySliceStart:hourlySliceEnd])),
			},
			LowTemperature: ModelValues[types.Temperature]{
				ModelGfsSeamless:        types.NewTemperatureFromFahrenheit(minFloat(apiResponse.Hourly.Temperature2MGfsSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelGemSeamless:        types.NewTemperatureFromFahrenheit(minFloat(apiResponse.Hourly.Temperature2MGemSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwIfs:            types.NewTemperatureFromFahrenheit(minFloat(apiResponse.Hourly.Temperature2MEcmwfIfs[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNbmConus:       types.NewTemperatureFromFahrenheit(minFloat(apiResponse.Hourly.Temperature2MNcepNbmConus[hourlySliceStart:hourlySliceEnd])),
				ModelGfsGraphcast025:    types.NewTemperatureFromFahrenheit(minFloat(apiResponse.Hourly.Temperature2MGfsGraphcast025[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNamConus:       types.NewTemperatureFromFahrenheit(minFloat(apiResponse.Hourly.Temperature2MNcepNamConus[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwfAifs025Single: types.NewTemperatureFromFahrenheit(minFloat(apiResponse.Hourly.Temperature2MEcmwfAifs025Single[hourlySliceStart:hourlySliceEnd])),
			},
			MaxWindSpeed: ModelValues[types.WindSpeed]{
				ModelGfsSeamless:        types.NewWindSpeedFromMph(maxFloat(apiResponse.Hourly.WindSpeed10MGfsSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelGemSeamless:        types.NewWindSpeedFromMph(maxFloat(apiResponse.Hourly.WindSpeed10MGemSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwIfs:            types.NewWindSpeedFromMph(maxFloat(apiResponse.Hourly.WindSpeed10MEcmwfIfs[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNbmConus:       types.NewWindSpeedFromMph(maxFloat(apiResponse.Hourly.WindSpeed10MNcepNbmConus[hourlySliceStart:hourlySliceEnd])),
				ModelGfsGraphcast025:    types.NewWindSpeedFromMph(maxFloat(apiResponse.Hourly.WindSpeed10MGfsGraphcast025[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwfAifs025Single: types.NewWindSpeedFromMph(maxFloat(apiResponse.Hourly.WindSpeed10MEcmwfAifs025Single[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNamConus:       types.NewWindSpeedFromMph(maxFloat(apiResponse.Hourly.WindSpeed10MNcepNamConus[hourlySliceStart:hourlySliceEnd])),
			},
			MinWindSpeed: ModelValues[types.WindSpeed]{
				ModelGfsSeamless:        types.NewWindSpeedFromMph(minFloat(apiResponse.Hourly.WindSpeed10MGfsSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelGemSeamless:        types.NewWindSpeedFromMph(minFloat(apiResponse.Hourly.WindSpeed10MGemSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwIfs:            types.NewWindSpeedFromMph(minFloat(apiResponse.Hourly.WindSpeed10MEcmwfIfs[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNbmConus:       types.NewWindSpeedFromMph(minFloat(apiResponse.Hourly.WindSpeed10MNcepNbmConus[hourlySliceStart:hourlySliceEnd])),
				ModelGfsGraphcast025:    types.NewWindSpeedFromMph(minFloat(apiResponse.Hourly.WindSpeed10MGfsGraphcast025[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwfAifs025Single: types.NewWindSpeedFromMph(minFloat(apiResponse.Hourly.WindSpeed10MEcmwfAifs025Single[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNamConus:       types.NewWindSpeedFromMph(minFloat(apiResponse.Hourly.WindSpeed10MNcepNamConus[hourlySliceStart:hourlySliceEnd])),
			},
			MaxWindGusts: ModelValues[types.WindSpeed]{
				ModelGfsSeamless:  types.NewWindSpeedFromMph(maxFloat(apiResponse.Hourly.WindGusts10MGfsSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelGemSeamless:  types.NewWindSpeedFromMph(maxFloat(apiResponse.Hourly.WindGusts10MGemSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwIfs:      types.NewWindSpeedFromMph(maxFloat(apiResponse.Hourly.WindGusts10MEcmwfIfs[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNbmConus: types.NewWindSpeedFromMph(maxFloat(apiResponse.Hourly.WindGusts10MNcepNbmConus[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNamConus: types.NewWindSpeedFromMph(maxFloat(apiResponse.Hourly.WindGusts10MNcepNamConus[hourlySliceStart:hourlySliceEnd])),
				// No data
				// ModelGfsGraphcast025:    types.NewWindSpeedFromMph(maxFloat(apiResponse.Hourly.WindGusts10MGfsGraphcast025[hourlySliceStart:hourlySliceEnd])),
				// ModelEcmwfAifs025Single: types.NewWindSpeedFromMph(maxFloat(apiResponse.Hourly.WindGusts10MEcmwfAifs025Single[hourlySliceStart:hourlySliceEnd])),
			},
			MinWindGusts: ModelValues[types.WindSpeed]{
				ModelGfsSeamless:  types.NewWindSpeedFromMph(minFloat(apiResponse.Hourly.WindGusts10MGfsSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelGemSeamless:  types.NewWindSpeedFromMph(minFloat(apiResponse.Hourly.WindGusts10MGemSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwIfs:      types.NewWindSpeedFromMph(minFloat(apiResponse.Hourly.WindGusts10MEcmwfIfs[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNbmConus: types.NewWindSpeedFromMph(minFloat(apiResponse.Hourly.WindGusts10MNcepNbmConus[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNamConus: types.NewWindSpeedFromMph(minFloat(apiResponse.Hourly.WindGusts10MNcepNamConus[hourlySliceStart:hourlySliceEnd])),
				// No data
				// ModelGfsGraphcast025:    types.NewWindSpeedFromMph(minFloat(apiResponse.Hourly.WindGusts10MGfsGraphcast025[hourlySliceStart:hourlySliceEnd])),
				// ModelEcmwfAifs025Single: types.NewWindSpeedFromMph(minFloat(apiResponse.Hourly.WindGusts10MEcmwfAifs025Single[hourlySliceStart:hourlySliceEnd])),
			},
			TotalRainfall: ModelValues[types.Precipitation]{
				ModelGfsSeamless:        types.NewPrecipitationFromInches(sum(apiResponse.Hourly.RainGfsSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelGemSeamless:        types.NewPrecipitationFromInches(sum(apiResponse.Hourly.RainGemSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwIfs:            types.NewPrecipitationFromInches(sum(apiResponse.Hourly.RainEcmwfIfs[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNbmConus:       types.NewPrecipitationFromInches(sum(apiResponse.Hourly.RainNcepNbmConus[hourlySliceStart:hourlySliceEnd])),
				ModelGfsGraphcast025:    types.NewPrecipitationFromInches(sum(apiResponse.Hourly.RainGfsGraphcast025[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwfAifs025Single: types.NewPrecipitationFromInches(sum(apiResponse.Hourly.RainEcmwfAifs025Single[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNamConus:       types.NewPrecipitationFromInches(sum(apiResponse.Hourly.RainNcepNamConus[hourlySliceStart:hourlySliceEnd])),
			},
			TotalPrecipitation: ModelValues[types.Precipitation]{
				ModelGfsSeamless:        types.NewPrecipitationFromInches(sum(apiResponse.Hourly.PrecipitationGfsSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelGemSeamless:        types.NewPrecipitationFromInches(sum(apiResponse.Hourly.PrecipitationGemSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwIfs:            types.NewPrecipitationFromInches(sum(apiResponse.Hourly.PrecipitationEcmwfIfs[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNbmConus:       types.NewPrecipitationFromInches(sum(apiResponse.Hourly.PrecipitationNcepNbmConus[hourlySliceStart:hourlySliceEnd])),
				ModelGfsGraphcast025:    types.NewPrecipitationFromInches(sum(apiResponse.Hourly.PrecipitationGfsGraphcast025[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwfAifs025Single: types.NewPrecipitationFromInches(sum(apiResponse.Hourly.PrecipitationEcmwfAifs025Single[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNamConus:       types.NewPrecipitationFromInches(sum(apiResponse.Hourly.PrecipitationNcepNamConus[hourlySliceStart:hourlySliceEnd])),
			},
			TotalShowers: ModelValues[types.Precipitation]{
				ModelGfsSeamless:        types.NewPrecipitationFromInches(sum(apiResponse.Hourly.ShowersGfsSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelGemSeamless:        types.NewPrecipitationFromInches(sum(apiResponse.Hourly.ShowersGemSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwIfs:            types.NewPrecipitationFromInches(sum(apiResponse.Hourly.ShowersEcmwfIfs[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNbmConus:       types.NewPrecipitationFromInches(sum(apiResponse.Hourly.ShowersNcepNbmConus[hourlySliceStart:hourlySliceEnd])),
				ModelGfsGraphcast025:    types.NewPrecipitationFromInches(sum(apiResponse.Hourly.ShowersGfsGraphcast025[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwfAifs025Single: types.NewPrecipitationFromInches(sum(apiResponse.Hourly.ShowersEcmwfAifs025Single[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNamConus:       types.NewPrecipitationFromInches(sum(apiResponse.Hourly.ShowersNcepNamConus[hourlySliceStart:hourlySliceEnd])),
			},
			TotalSnowfall: ModelValues[types.Precipitation]{
				ModelGfsSeamless:        types.NewPrecipitationFromInches(sum(apiResponse.Hourly.SnowfallGfsSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelGemSeamless:        types.NewPrecipitationFromInches(sum(apiResponse.Hourly.SnowfallGemSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwIfs:            types.NewPrecipitationFromInches(sum(apiResponse.Hourly.SnowfallEcmwfIfs[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNbmConus:       types.NewPrecipitationFromInches(sum(apiResponse.Hourly.SnowfallNcepNbmConus[hourlySliceStart:hourlySliceEnd])),
				ModelGfsGraphcast025:    types.NewPrecipitationFromInches(sum(apiResponse.Hourly.SnowfallGfsGraphcast025[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwfAifs025Single: types.NewPrecipitationFromInches(sum(apiResponse.Hourly.SnowfallEcmwfAifs025Single[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNamConus:       types.NewPrecipitationFromInches(sum(apiResponse.Hourly.SnowfallNcepNamConus[hourlySliceStart:hourlySliceEnd])),
			},
		}

		totalLiquidPrecipitation := make(ModelValues[types.Precipitation], len(dailyForecast.TotalRainfall))
		for model, rain := range dailyForecast.TotalRainfall {
			showers := dailyForecast.TotalShowers[model]
			totalLiquidPrecipitation[model] = types.NewPrecipitationFromInches(rain.Inches + showers.Inches)
		}
		dailyForecast.TotalLiquidPrecipitation = totalLiquidPrecipitation

		dailyForecasts = append(dailyForecasts, dailyForecast)
	}

	forecast.DailyForecasts = dailyForecasts

	return forecast, nil

}

func toPercentage(value int) float64 {
	return float64(value) / 100.0
}

func toTime(value string) time.Time {
	if t, err := time.Parse("2006-01-02T15:04", value); err == nil {
		return t
	}

	return time.Time{}
}

func minFloat(value []float64) float64 {
	if len(value) == 0 {
		return -1
	}

	minValue := value[0]
	for _, v := range value {
		if v < minValue {
			minValue = v
		}
	}
	return minValue
}

func maxFloat(value []float64) float64 {
	if len(value) == 0 {
		return -1
	}

	maxValue := value[0]
	for _, v := range value {
		if v > maxValue {
			maxValue = v
		}
	}
	return maxValue
}

func sum(value []float64) float64 {
	total := 0.0
	for _, v := range value {
		total += v
	}
	return total
}
