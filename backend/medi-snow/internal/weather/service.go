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
		Temperature: map[string]types.Temperature{
			ModelGfsSeamless:        types.NewTemperatureFromFahrenheit(apiResponse.Hourly.Temperature2MGfsSeamless[nowIndex]),
			ModelGemSeamless:        types.NewTemperatureFromFahrenheit(apiResponse.Hourly.Temperature2MGemSeamless[nowIndex]),
			ModelEcmwIfs:            types.NewTemperatureFromFahrenheit(apiResponse.Hourly.Temperature2MEcmwfIfs[nowIndex]),
			ModelNcepNbmConus:       types.NewTemperatureFromFahrenheit(apiResponse.Hourly.Temperature2MNcepNbmConus[nowIndex]),
			ModelGfsGraphcast025:    types.NewTemperatureFromFahrenheit(apiResponse.Hourly.Temperature2MGfsGraphcast025[nowIndex]),
			ModelEcmwfAifs025Single: types.NewTemperatureFromFahrenheit(apiResponse.Hourly.Temperature2MEcmwfAifs025Single[nowIndex]),
			ModelNcepNamConus:       types.NewTemperatureFromFahrenheit(apiResponse.Hourly.Temperature2MNcepNamConus[nowIndex]),
		},
		Weather: map[string]types.Weather{
			ModelGfsSeamless:        types.NewWeather(apiResponse.Hourly.WeatherCodeGfsSeamless[nowIndex]),
			ModelGemSeamless:        types.NewWeather(apiResponse.Hourly.WeatherCodeGemSeamless[nowIndex]),
			ModelEcmwIfs:            types.NewWeather(apiResponse.Hourly.WeatherCodeEcmwfIfs[nowIndex]),
			ModelNcepNbmConus:       types.NewWeather(apiResponse.Hourly.WeatherCodeNcepNbmConus[nowIndex]),
			ModelGfsGraphcast025:    types.NewWeather(apiResponse.Hourly.WeatherCodeGfsGraphcast025[nowIndex]),
			ModelEcmwfAifs025Single: types.NewWeather(apiResponse.Hourly.WeatherCodeEcmwfAifs025Single[nowIndex]),
			ModelNcepNamConus:       types.NewWeather(apiResponse.Hourly.WeatherCodeNcepNamConus[nowIndex]),
		},
		Wind: map[string]types.Wind{
			ModelGfsSeamless:        types.NewWind(apiResponse.Hourly.WindSpeed10MGfsSeamless[nowIndex], apiResponse.Hourly.WindGusts10MGfsSeamless[nowIndex], apiResponse.Hourly.WindDirection10MGfsSeamless[nowIndex]),
			ModelGemSeamless:        types.NewWind(apiResponse.Hourly.WindSpeed10MGemSeamless[nowIndex], apiResponse.Hourly.WindGusts10MGemSeamless[nowIndex], apiResponse.Hourly.WindDirection10MGemSeamless[nowIndex]),
			ModelEcmwIfs:            types.NewWind(apiResponse.Hourly.WindSpeed10MEcmwfIfs[nowIndex], apiResponse.Hourly.WindGusts10MEcmwfIfs[nowIndex], apiResponse.Hourly.WindDirection10MEcmwfIfs[nowIndex]),
			ModelNcepNbmConus:       types.NewWind(apiResponse.Hourly.WindSpeed10MNcepNbmConus[nowIndex], apiResponse.Hourly.WindGusts10MNcepNbmConus[nowIndex], apiResponse.Hourly.WindDirection10MNcepNbmConus[nowIndex]),
			ModelGfsGraphcast025:    types.NewWind(apiResponse.Hourly.WindSpeed10MGfsGraphcast025[nowIndex], -1, apiResponse.Hourly.WindDirection10MGfsGraphcast025[nowIndex]),
			ModelEcmwfAifs025Single: types.NewWind(apiResponse.Hourly.WindSpeed10MEcmwfAifs025Single[nowIndex], -1, apiResponse.Hourly.WindDirection10MEcmwfAifs025Single[nowIndex]),
			ModelNcepNamConus:       types.NewWind(apiResponse.Hourly.WindSpeed10MNcepNamConus[nowIndex], apiResponse.Hourly.WindGusts10MNcepNamConus[nowIndex], apiResponse.Hourly.WindDirection10MNcepNamConus[nowIndex]),
		},
		Visibility: map[string]float64{
			ModelGfsSeamless:  apiResponse.Hourly.VisibilityGfsSeamless[nowIndex],
			ModelEcmwIfs:      apiResponse.Hourly.VisibilityEcmwfIfs[nowIndex],
			ModelNcepNbmConus: apiResponse.Hourly.VisibilityNcepNbmConus[nowIndex],
			ModelNcepNamConus: apiResponse.Hourly.VisibilityNcepNamConus[nowIndex],
			// No data
			// ModelGemSeamless: 		apiResponse.Hourly.VisibilityGemSeamless[nowIndex],
			// ModelGfsGraphcast025:    apiResponse.Hourly.VisibilityGfsGraphcast025[nowIndex],
			// ModelEcmwfAifs025Single: apiResponse.Hourly.VisibilityEcmwfAifs025Single[nowIndex],
		},
		CloudCover: map[string]float64{
			ModelGfsSeamless:        toPercentage(apiResponse.Hourly.CloudCoverGfsSeamless[nowIndex]),
			ModelGemSeamless:        toPercentage(apiResponse.Hourly.CloudCoverGemSeamless[nowIndex]),
			ModelEcmwIfs:            toPercentage(apiResponse.Hourly.CloudCoverEcmwfIfs[nowIndex]),
			ModelNcepNbmConus:       toPercentage(apiResponse.Hourly.CloudCoverNcepNbmConus[nowIndex]),
			ModelGfsGraphcast025:    toPercentage(apiResponse.Hourly.CloudCoverGfsGraphcast025[nowIndex]),
			ModelEcmwfAifs025Single: toPercentage(apiResponse.Hourly.CloudCoverEcmwfAifs025Single[nowIndex]),
			ModelNcepNamConus:       toPercentage(apiResponse.Hourly.CloudCoverNcepNamConus[nowIndex]),
		},
		RelativeHumidity: map[string]float64{
			ModelGfsSeamless:        toPercentage(apiResponse.Hourly.RelativeHumidity2MGfsSeamless[nowIndex]),
			ModelGemSeamless:        toPercentage(apiResponse.Hourly.RelativeHumidity2MGemSeamless[nowIndex]),
			ModelEcmwIfs:            toPercentage(apiResponse.Hourly.RelativeHumidity2MEcmwfIfs[nowIndex]),
			ModelNcepNbmConus:       toPercentage(apiResponse.Hourly.RelativeHumidity2MNcepNbmConus[nowIndex]),
			ModelEcmwfAifs025Single: toPercentage(apiResponse.Hourly.RelativeHumidity2MEcmwfAifs025Single[nowIndex]),
			ModelNcepNamConus:       toPercentage(apiResponse.Hourly.RelativeHumidity2MNcepNamConus[nowIndex]),
			// No data
			// ModelGfsGraphcast025:    toPercentage(apiResponse.Hourly.RelativeHumidity2MGfsGraphcast025[nowIndex]),
		},
		CloudCoverLow: map[string]float64{
			ModelGfsSeamless:        toPercentage(apiResponse.Hourly.CloudCoverLowGfsSeamless[nowIndex]),
			ModelGemSeamless:        toPercentage(apiResponse.Hourly.CloudCoverLowGemSeamless[nowIndex]),
			ModelEcmwIfs:            toPercentage(apiResponse.Hourly.CloudCoverLowEcmwfIfs[nowIndex]),
			ModelGfsGraphcast025:    toPercentage(apiResponse.Hourly.CloudCoverLowGfsGraphcast025[nowIndex]),
			ModelEcmwfAifs025Single: toPercentage(apiResponse.Hourly.CloudCoverLowEcmwfAifs025Single[nowIndex]),
			ModelNcepNamConus:       toPercentage(apiResponse.Hourly.CloudCoverLowNcepNamConus[nowIndex]),
			// No data
			// ModelNcepNbmConus:       toPercentage(apiResponse.Hourly.CloudCoverLowNcepNbmConus[nowIndex]),
		},
		CloudCoverMid: map[string]float64{
			ModelGfsSeamless:        toPercentage(apiResponse.Hourly.CloudCoverMidGfsSeamless[nowIndex]),
			ModelGemSeamless:        toPercentage(apiResponse.Hourly.CloudCoverMidGemSeamless[nowIndex]),
			ModelEcmwIfs:            toPercentage(apiResponse.Hourly.CloudCoverMidEcmwfIfs[nowIndex]),
			ModelGfsGraphcast025:    toPercentage(apiResponse.Hourly.CloudCoverMidGfsGraphcast025[nowIndex]),
			ModelEcmwfAifs025Single: toPercentage(apiResponse.Hourly.CloudCoverMidEcmwfAifs025Single[nowIndex]),
			ModelNcepNamConus:       toPercentage(apiResponse.Hourly.CloudCoverMidNcepNamConus[nowIndex]),
			// No data
			// ModelNcepNbmConus:       toPercentage(apiResponse.Hourly.CloudCoverMidNcepNbmConus[nowIndex]),
		},
		CloudCoverHigh: map[string]float64{
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

				// TODO construct hourly forecast
				hourlyForecast := HourlyForecast{}
				hourlyForecasts = append(hourlyForecasts, hourlyForecast)
			} else {
				break
			}
		}

		var hourlySliceStart = dailyForecastHourlyIndexes[0]
		var hourlySliceEnd = dailyForecastHourlyIndexes[len(dailyForecastHourlyIndexes)]

		// TODO construct daily forecast
		dailyForecast := DailyForecast{
			HourlyForecasts: hourlyForecasts,
			Timestamp:       dayTime,
			Weather: map[string]types.Weather{
				ModelGfsSeamless:        types.NewWeather(apiResponse.Daily.WeatherCodeGfsSeamless[i]),
				ModelGemSeamless:        types.NewWeather(apiResponse.Daily.WeatherCodeGemSeamless[i]),
				ModelEcmwIfs:            types.NewWeather(apiResponse.Daily.WeatherCodeEcmwfIfs[i]),
				ModelNcepNbmConus:       types.NewWeather(apiResponse.Daily.WeatherCodeNcepNbmConus[i]),
				ModelGfsGraphcast025:    types.NewWeather(apiResponse.Daily.WeatherCodeGfsGraphcast025[i]),
				ModelEcmwfAifs025Single: types.NewWeather(apiResponse.Daily.WeatherCodeEcmwfAifs025Single[i]),
				ModelNcepNamConus:       types.NewWeather(apiResponse.Daily.WeatherCodeNcepNamConus[i]),
			},
			SnowfallWaterEquivalentSum: map[string]float64{
				ModelGfsSeamless:        apiResponse.Daily.SnowfallWaterEquivalentSumGfsSeamless[i],
				ModelGemSeamless:        apiResponse.Daily.SnowfallWaterEquivalentSumGemSeamless[i],
				ModelEcmwIfs:            apiResponse.Daily.SnowfallWaterEquivalentSumEcmwfIfs[i],
				ModelNcepNbmConus:       apiResponse.Daily.SnowfallWaterEquivalentSumNcepNbmConus[i],
				ModelEcmwfAifs025Single: apiResponse.Daily.SnowfallWaterEquivalentSumEcmwfAifs025Single[i],
				ModelNcepNamConus:       apiResponse.Daily.SnowfallWaterEquivalentSumNcepNamConus[i],
				// No data
				// ModelGfsGraphcast025:    apiResponse.Daily.SnowfallWaterEquivalentSumGfsGraphcast025[i],
			},
			Sunrise: map[string]time.Time{
				ModelGfsSeamless:        toTime(apiResponse.Daily.SunriseGfsSeamless[i]),
				ModelGemSeamless:        toTime(apiResponse.Daily.SunriseGemSeamless[i]),
				ModelEcmwIfs:            toTime(apiResponse.Daily.SunriseEcmwfIfs[i]),
				ModelNcepNbmConus:       toTime(apiResponse.Daily.SunriseNcepNbmConus[i]),
				ModelGfsGraphcast025:    toTime(apiResponse.Daily.SunriseGfsGraphcast025[i]),
				ModelEcmwfAifs025Single: toTime(apiResponse.Daily.SunriseEcmwfAifs025Single[i]),
				ModelNcepNamConus:       toTime(apiResponse.Daily.SunriseNcepNamConus[i]),
			},
			Sunset: map[string]time.Time{
				ModelGfsSeamless:        toTime(apiResponse.Daily.SunsetGfsSeamless[i]),
				ModelGemSeamless:        toTime(apiResponse.Daily.SunsetGemSeamless[i]),
				ModelEcmwIfs:            toTime(apiResponse.Daily.SunsetEcmwfIfs[i]),
				ModelNcepNbmConus:       toTime(apiResponse.Daily.SunsetNcepNbmConus[i]),
				ModelGfsGraphcast025:    toTime(apiResponse.Daily.SunsetGfsGraphcast025[i]),
				ModelEcmwfAifs025Single: toTime(apiResponse.Daily.SunsetEcmwfAifs025Single[i]),
				ModelNcepNamConus:       toTime(apiResponse.Daily.SunsetNcepNamConus[i]),
			},
			WindDominantDirection: map[string]types.WindDirection{
				ModelGfsSeamless:        types.NewWindDirection(apiResponse.Daily.WindDirection10MDominantGfsSeamless[i]),
				ModelGemSeamless:        types.NewWindDirection(apiResponse.Daily.WindDirection10MDominantGemSeamless[i]),
				ModelEcmwIfs:            types.NewWindDirection(apiResponse.Daily.WindDirection10MDominantEcmwfIfs[i]),
				ModelNcepNbmConus:       types.NewWindDirection(apiResponse.Daily.WindDirection10MDominantNcepNbmConus[i]),
				ModelEcmwfAifs025Single: types.NewWindDirection(apiResponse.Daily.WindDirection10MDominantEcmwfAifs025Single[i]),
				ModelNcepNamConus:       types.NewWindDirection(apiResponse.Daily.WindDirection10MDominantNcepNamConus[i]),
				// No data
				// ModelGfsGraphcast025:    types.NewWindDirection(apiResponse.Daily.WindDirection10MDominantGfsGraphcast025[i]),
			},
			HighestFreezingLevelHeightFt: map[string]float64{
				ModelGfsSeamless: maxFloat(apiResponse.Hourly.FreezingLevelHeightGfsSeamless[hourlySliceStart:hourlySliceEnd]),
			},
			LowestFreezingLevelHeightFt: map[string]float64{
				ModelGfsSeamless: minFloat(apiResponse.Hourly.FreezingLevelHeightGfsSeamless[hourlySliceStart:hourlySliceEnd]),
			},
			HighTemperature: map[string]types.Temperature{
				ModelGfsSeamless:        types.NewTemperatureFromFahrenheit(maxFloat(apiResponse.Hourly.Temperature2MGfsSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelGemSeamless:        types.NewTemperatureFromFahrenheit(maxFloat(apiResponse.Hourly.Temperature2MGemSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwIfs:            types.NewTemperatureFromFahrenheit(maxFloat(apiResponse.Hourly.Temperature2MEcmwfIfs[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNbmConus:       types.NewTemperatureFromFahrenheit(maxFloat(apiResponse.Hourly.Temperature2MNcepNbmConus[hourlySliceStart:hourlySliceEnd])),
				ModelGfsGraphcast025:    types.NewTemperatureFromFahrenheit(maxFloat(apiResponse.Hourly.Temperature2MGfsGraphcast025[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNamConus:       types.NewTemperatureFromFahrenheit(maxFloat(apiResponse.Hourly.Temperature2MNcepNamConus[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwfAifs025Single: types.NewTemperatureFromFahrenheit(maxFloat(apiResponse.Hourly.Temperature2MEcmwfAifs025Single[hourlySliceStart:hourlySliceEnd])),
			},
			LowTemperature: map[string]types.Temperature{
				ModelGfsSeamless:        types.NewTemperatureFromFahrenheit(minFloat(apiResponse.Hourly.Temperature2MGfsSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelGemSeamless:        types.NewTemperatureFromFahrenheit(minFloat(apiResponse.Hourly.Temperature2MGemSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwIfs:            types.NewTemperatureFromFahrenheit(minFloat(apiResponse.Hourly.Temperature2MEcmwfIfs[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNbmConus:       types.NewTemperatureFromFahrenheit(minFloat(apiResponse.Hourly.Temperature2MNcepNbmConus[hourlySliceStart:hourlySliceEnd])),
				ModelGfsGraphcast025:    types.NewTemperatureFromFahrenheit(minFloat(apiResponse.Hourly.Temperature2MGfsGraphcast025[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNamConus:       types.NewTemperatureFromFahrenheit(minFloat(apiResponse.Hourly.Temperature2MNcepNamConus[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwfAifs025Single: types.NewTemperatureFromFahrenheit(minFloat(apiResponse.Hourly.Temperature2MEcmwfAifs025Single[hourlySliceStart:hourlySliceEnd])),
			},
			MaxWindSpeed: map[string]types.WindSpeed{
				ModelGfsSeamless:        types.NewWindSpeedFromMph(maxFloat(apiResponse.Hourly.WindSpeed10MGfsSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelGemSeamless:        types.NewWindSpeedFromMph(maxFloat(apiResponse.Hourly.WindSpeed10MGemSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwIfs:            types.NewWindSpeedFromMph(maxFloat(apiResponse.Hourly.WindSpeed10MEcmwfIfs[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNbmConus:       types.NewWindSpeedFromMph(maxFloat(apiResponse.Hourly.WindSpeed10MNcepNbmConus[hourlySliceStart:hourlySliceEnd])),
				ModelGfsGraphcast025:    types.NewWindSpeedFromMph(maxFloat(apiResponse.Hourly.WindSpeed10MGfsGraphcast025[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwfAifs025Single: types.NewWindSpeedFromMph(maxFloat(apiResponse.Hourly.WindSpeed10MEcmwfAifs025Single[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNamConus:       types.NewWindSpeedFromMph(maxFloat(apiResponse.Hourly.WindSpeed10MNcepNamConus[hourlySliceStart:hourlySliceEnd])),
			},
			MinWindSpeed: map[string]types.WindSpeed{
				ModelGfsSeamless:        types.NewWindSpeedFromMph(minFloat(apiResponse.Hourly.WindSpeed10MGfsSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelGemSeamless:        types.NewWindSpeedFromMph(minFloat(apiResponse.Hourly.WindSpeed10MGemSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwIfs:            types.NewWindSpeedFromMph(minFloat(apiResponse.Hourly.WindSpeed10MEcmwfIfs[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNbmConus:       types.NewWindSpeedFromMph(minFloat(apiResponse.Hourly.WindSpeed10MNcepNbmConus[hourlySliceStart:hourlySliceEnd])),
				ModelGfsGraphcast025:    types.NewWindSpeedFromMph(minFloat(apiResponse.Hourly.WindSpeed10MGfsGraphcast025[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwfAifs025Single: types.NewWindSpeedFromMph(minFloat(apiResponse.Hourly.WindSpeed10MEcmwfAifs025Single[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNamConus:       types.NewWindSpeedFromMph(minFloat(apiResponse.Hourly.WindSpeed10MNcepNamConus[hourlySliceStart:hourlySliceEnd])),
			},
			MaxWindGusts: map[string]types.WindSpeed{
				ModelGfsSeamless:  types.NewWindSpeedFromMph(maxFloat(apiResponse.Hourly.WindGusts10MGfsSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelGemSeamless:  types.NewWindSpeedFromMph(maxFloat(apiResponse.Hourly.WindGusts10MGemSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwIfs:      types.NewWindSpeedFromMph(maxFloat(apiResponse.Hourly.WindGusts10MEcmwfIfs[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNbmConus: types.NewWindSpeedFromMph(maxFloat(apiResponse.Hourly.WindGusts10MNcepNbmConus[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNamConus: types.NewWindSpeedFromMph(maxFloat(apiResponse.Hourly.WindGusts10MNcepNamConus[hourlySliceStart:hourlySliceEnd])),
				// No data
				// ModelGfsGraphcast025:    types.NewWindSpeedFromMph(maxFloat(apiResponse.Hourly.WindGusts10MGfsGraphcast025[hourlySliceStart:hourlySliceEnd])),
				// ModelEcmwfAifs025Single: types.NewWindSpeedFromMph(maxFloat(apiResponse.Hourly.WindGusts10MEcmwfAifs025Single[hourlySliceStart:hourlySliceEnd])),
			},
			MinWindGusts: map[string]types.WindSpeed{
				ModelGfsSeamless:  types.NewWindSpeedFromMph(minFloat(apiResponse.Hourly.WindGusts10MGfsSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelGemSeamless:  types.NewWindSpeedFromMph(minFloat(apiResponse.Hourly.WindGusts10MGemSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwIfs:      types.NewWindSpeedFromMph(minFloat(apiResponse.Hourly.WindGusts10MEcmwfIfs[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNbmConus: types.NewWindSpeedFromMph(minFloat(apiResponse.Hourly.WindGusts10MNcepNbmConus[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNamConus: types.NewWindSpeedFromMph(minFloat(apiResponse.Hourly.WindGusts10MNcepNamConus[hourlySliceStart:hourlySliceEnd])),
				// No data
				// ModelGfsGraphcast025:    types.NewWindSpeedFromMph(minFloat(apiResponse.Hourly.WindGusts10MGfsGraphcast025[hourlySliceStart:hourlySliceEnd])),
				// ModelEcmwfAifs025Single: types.NewWindSpeedFromMph(minFloat(apiResponse.Hourly.WindGusts10MEcmwfAifs025Single[hourlySliceStart:hourlySliceEnd])),
			},
			TotalRainfall: map[string]types.Precipitation{
				ModelGfsSeamless:        types.NewPrecipitationFromInches(sum(apiResponse.Hourly.RainGfsSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelGemSeamless:        types.NewPrecipitationFromInches(sum(apiResponse.Hourly.RainGemSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwIfs:            types.NewPrecipitationFromInches(sum(apiResponse.Hourly.RainEcmwfIfs[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNbmConus:       types.NewPrecipitationFromInches(sum(apiResponse.Hourly.RainNcepNbmConus[hourlySliceStart:hourlySliceEnd])),
				ModelGfsGraphcast025:    types.NewPrecipitationFromInches(sum(apiResponse.Hourly.RainGfsGraphcast025[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwfAifs025Single: types.NewPrecipitationFromInches(sum(apiResponse.Hourly.RainEcmwfAifs025Single[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNamConus:       types.NewPrecipitationFromInches(sum(apiResponse.Hourly.RainNcepNamConus[hourlySliceStart:hourlySliceEnd])),
			},
			TotalPrecipitation: map[string]types.Precipitation{
				ModelGfsSeamless:        types.NewPrecipitationFromInches(sum(apiResponse.Hourly.PrecipitationGfsSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelGemSeamless:        types.NewPrecipitationFromInches(sum(apiResponse.Hourly.PrecipitationGemSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwIfs:            types.NewPrecipitationFromInches(sum(apiResponse.Hourly.PrecipitationEcmwfIfs[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNbmConus:       types.NewPrecipitationFromInches(sum(apiResponse.Hourly.PrecipitationNcepNbmConus[hourlySliceStart:hourlySliceEnd])),
				ModelGfsGraphcast025:    types.NewPrecipitationFromInches(sum(apiResponse.Hourly.PrecipitationGfsGraphcast025[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwfAifs025Single: types.NewPrecipitationFromInches(sum(apiResponse.Hourly.PrecipitationEcmwfAifs025Single[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNamConus:       types.NewPrecipitationFromInches(sum(apiResponse.Hourly.PrecipitationNcepNamConus[hourlySliceStart:hourlySliceEnd])),
			},
			TotalShowers: map[string]types.Precipitation{
				ModelGfsSeamless:        types.NewPrecipitationFromInches(sum(apiResponse.Hourly.ShowersGfsSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelGemSeamless:        types.NewPrecipitationFromInches(sum(apiResponse.Hourly.ShowersGemSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwIfs:            types.NewPrecipitationFromInches(sum(apiResponse.Hourly.ShowersEcmwfIfs[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNbmConus:       types.NewPrecipitationFromInches(sum(apiResponse.Hourly.ShowersNcepNbmConus[hourlySliceStart:hourlySliceEnd])),
				ModelGfsGraphcast025:    types.NewPrecipitationFromInches(sum(apiResponse.Hourly.ShowersGfsGraphcast025[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwfAifs025Single: types.NewPrecipitationFromInches(sum(apiResponse.Hourly.ShowersEcmwfAifs025Single[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNamConus:       types.NewPrecipitationFromInches(sum(apiResponse.Hourly.ShowersNcepNamConus[hourlySliceStart:hourlySliceEnd])),
			},
			TotalSnowfall: map[string]types.Precipitation{
				ModelGfsSeamless:        types.NewPrecipitationFromInches(sum(apiResponse.Hourly.SnowfallGfsSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelGemSeamless:        types.NewPrecipitationFromInches(sum(apiResponse.Hourly.SnowfallGemSeamless[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwIfs:            types.NewPrecipitationFromInches(sum(apiResponse.Hourly.SnowfallEcmwfIfs[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNbmConus:       types.NewPrecipitationFromInches(sum(apiResponse.Hourly.SnowfallNcepNbmConus[hourlySliceStart:hourlySliceEnd])),
				ModelGfsGraphcast025:    types.NewPrecipitationFromInches(sum(apiResponse.Hourly.SnowfallGfsGraphcast025[hourlySliceStart:hourlySliceEnd])),
				ModelEcmwfAifs025Single: types.NewPrecipitationFromInches(sum(apiResponse.Hourly.SnowfallEcmwfAifs025Single[hourlySliceStart:hourlySliceEnd])),
				ModelNcepNamConus:       types.NewPrecipitationFromInches(sum(apiResponse.Hourly.SnowfallNcepNamConus[hourlySliceStart:hourlySliceEnd])),
			},
		}

		totalLiquidPrecipitation := make(map[string]types.Precipitation)
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
	if t, err := time.Parse("2006-01-02T15:04", value); err != nil {
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
