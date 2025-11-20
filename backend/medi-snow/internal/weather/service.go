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
			ModelGfsSeamless:        types.NewWindFromMph(apiResponse.Hourly.WindSpeed10MGfsSeamless[nowIndex], apiResponse.Hourly.WindGusts10MGfsSeamless[nowIndex], apiResponse.Hourly.WindDirection10MGfsSeamless[nowIndex]),
			ModelGemSeamless:        types.NewWindFromMph(apiResponse.Hourly.WindSpeed10MGemSeamless[nowIndex], apiResponse.Hourly.WindGusts10MGemSeamless[nowIndex], apiResponse.Hourly.WindDirection10MGemSeamless[nowIndex]),
			ModelEcmwIfs:            types.NewWindFromMph(apiResponse.Hourly.WindSpeed10MEcmwfIfs[nowIndex], apiResponse.Hourly.WindGusts10MEcmwfIfs[nowIndex], apiResponse.Hourly.WindDirection10MEcmwfIfs[nowIndex]),
			ModelNcepNbmConus:       types.NewWindFromMph(apiResponse.Hourly.WindSpeed10MNcepNbmConus[nowIndex], apiResponse.Hourly.WindGusts10MNcepNbmConus[nowIndex], apiResponse.Hourly.WindDirection10MNcepNbmConus[nowIndex]),
			ModelGfsGraphcast025:    types.NewWindFromMph(apiResponse.Hourly.WindSpeed10MGfsGraphcast025[nowIndex], safeFloat64(apiResponse.Hourly.WindGusts10MGfsGraphcast025[nowIndex]), apiResponse.Hourly.WindDirection10MGfsGraphcast025[nowIndex]),
			ModelEcmwfAifs025Single: types.NewWindFromMph(apiResponse.Hourly.WindSpeed10MEcmwfAifs025Single[nowIndex], safeFloat64(apiResponse.Hourly.WindGusts10MEcmwfAifs025Single[nowIndex]), apiResponse.Hourly.WindDirection10MEcmwfAifs025Single[nowIndex]),
			ModelNcepNamConus:       types.NewWindFromMph(apiResponse.Hourly.WindSpeed10MNcepNamConus[nowIndex], apiResponse.Hourly.WindGusts10MNcepNamConus[nowIndex], apiResponse.Hourly.WindDirection10MNcepNamConus[nowIndex]),
		},
		Visibility: map[string]float64{
			ModelGfsSeamless:        apiResponse.Hourly.VisibilityGfsSeamless[nowIndex],
			ModelGemSeamless:        safeFloat64(apiResponse.Hourly.VisibilityGemSeamless[nowIndex]),
			ModelEcmwIfs:            apiResponse.Hourly.VisibilityEcmwfIfs[nowIndex],
			ModelNcepNbmConus:       apiResponse.Hourly.VisibilityNcepNbmConus[nowIndex],
			ModelGfsGraphcast025:    safeFloat64(apiResponse.Hourly.VisibilityGfsGraphcast025[nowIndex]),
			ModelEcmwfAifs025Single: safeFloat64(apiResponse.Hourly.VisibilityEcmwfAifs025Single[nowIndex]),
			ModelNcepNamConus:       apiResponse.Hourly.VisibilityNcepNamConus[nowIndex],
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
			ModelGfsGraphcast025:    toPercentage(safeInt(apiResponse.Hourly.RelativeHumidity2MGfsGraphcast025[nowIndex])),
			ModelEcmwfAifs025Single: toPercentage(apiResponse.Hourly.RelativeHumidity2MEcmwfAifs025Single[nowIndex]),
			ModelNcepNamConus:       toPercentage(apiResponse.Hourly.RelativeHumidity2MNcepNamConus[nowIndex]),
		},
		CloudCoverLow: map[string]float64{
			ModelGfsSeamless:        toPercentage(apiResponse.Hourly.CloudCoverLowGfsSeamless[nowIndex]),
			ModelGemSeamless:        toPercentage(apiResponse.Hourly.CloudCoverLowGemSeamless[nowIndex]),
			ModelEcmwIfs:            toPercentage(apiResponse.Hourly.CloudCoverLowEcmwfIfs[nowIndex]),
			ModelNcepNbmConus:       toPercentage(safeInt(apiResponse.Hourly.CloudCoverLowNcepNbmConus[nowIndex])),
			ModelGfsGraphcast025:    toPercentage(apiResponse.Hourly.CloudCoverLowGfsGraphcast025[nowIndex]),
			ModelEcmwfAifs025Single: toPercentage(apiResponse.Hourly.CloudCoverLowEcmwfAifs025Single[nowIndex]),
			ModelNcepNamConus:       toPercentage(apiResponse.Hourly.CloudCoverLowNcepNamConus[nowIndex]),
		},
		CloudCoverMid: map[string]float64{
			ModelGfsSeamless:        toPercentage(apiResponse.Hourly.CloudCoverMidGfsSeamless[nowIndex]),
			ModelGemSeamless:        toPercentage(apiResponse.Hourly.CloudCoverMidGemSeamless[nowIndex]),
			ModelEcmwIfs:            toPercentage(apiResponse.Hourly.CloudCoverMidEcmwfIfs[nowIndex]),
			ModelNcepNbmConus:       toPercentage(safeInt(apiResponse.Hourly.CloudCoverMidNcepNbmConus[nowIndex])),
			ModelGfsGraphcast025:    toPercentage(apiResponse.Hourly.CloudCoverMidGfsGraphcast025[nowIndex]),
			ModelEcmwfAifs025Single: toPercentage(apiResponse.Hourly.CloudCoverMidEcmwfAifs025Single[nowIndex]),
			ModelNcepNamConus:       toPercentage(apiResponse.Hourly.CloudCoverMidNcepNamConus[nowIndex]),
		},
		CloudCoverHigh: map[string]float64{
			ModelGfsSeamless:        toPercentage(apiResponse.Hourly.CloudCoverHighGfsSeamless[nowIndex]),
			ModelGemSeamless:        toPercentage(apiResponse.Hourly.CloudCoverHighGemSeamless[nowIndex]),
			ModelEcmwIfs:            toPercentage(apiResponse.Hourly.CloudCoverHighEcmwfIfs[nowIndex]),
			ModelNcepNbmConus:       toPercentage(safeInt(apiResponse.Hourly.CloudCoverHighNcepNbmConus[nowIndex])),
			ModelGfsGraphcast025:    toPercentage(apiResponse.Hourly.CloudCoverHighGfsGraphcast025[nowIndex]),
			ModelEcmwfAifs025Single: toPercentage(apiResponse.Hourly.CloudCoverHighEcmwfAifs025Single[nowIndex]),
			ModelNcepNamConus:       toPercentage(apiResponse.Hourly.CloudCoverHighNcepNamConus[nowIndex]),
		},
	}

	forecast.CurrentConditions = currentConditions

	dailyForecasts := make([]DailyForecast, 0, len(apiResponse.Daily.Time))
	hourlyIndex := 0

	// Get each daily forecast
	for i, day := range apiResponse.Daily.Time {
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
		}
		dailyForecasts = append(dailyForecasts, dailyForecast)
	}

	forecast.DailyForecasts = dailyForecasts

	return forecast, nil

}

func safeFloat64(v interface{}) float64 {
	if f, ok := v.(float64); ok {
		return f
	}
	return -1.0
}

func safeInt(v interface{}) int {
	if i, ok := v.(int); ok {
		return i
	}
	return -1
}

func toPercentage(value int) float64 {
	return float64(value) / 100.0
}
