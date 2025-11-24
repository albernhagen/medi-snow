//go:build integration

package weather

import (
	"encoding/json"
	"medi-snow/internal/providers/openmeteo"
	"medi-snow/internal/types"
	"net/http"
	"testing"
)

func TestMapForecastAPIResponseToForecast_Integration(t *testing.T) {
	// Real OpenMeteo API request for Aspen, Colorado
	url := "https://api.open-meteo.com/v1/forecast?daily=snowfall_water_equivalent_sum%2Cweather_code%2Csunrise%2Csunset%2Cwind_direction_10m_dominant&elevation=4352.000000&forecast_days=16&hourly=freezing_level_height%2Cis_day%2Ctemperature_2m%2Cweather_code%2Capparent_temperature%2Cprecipitation_probability%2Cprecipitation%2Ccloud_cover%2Ccloud_cover_low%2Ccloud_cover_mid%2Ccloud_cover_high%2Cvisibility%2Cwind_speed_10m%2Cwind_direction_10m%2Cwind_gusts_10m%2Crelative_humidity_2m%2Crain%2Cshowers%2Csnowfall%2Csnow_depth&latitude=39.115390&longitude=-107.658400&models=gem_seamless%2Cecmwf_ifs%2Cgfs_seamless%2Cncep_nbm_conus%2Cgfs_graphcast025%2Cecmwf_aifs025_single%2Cncep_nam_conus&precipitation_unit=inch&temperature_unit=fahrenheit&timeformat=iso8601&timezone=America%2FDenver&wind_speed_unit=mph"

	// Make HTTP request
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Failed to make HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	// Unmarshal response
	var apiResponse openmeteo.ForecastAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Create forecast point
	forecastPoint := types.ForecastPoint{
		Coordinates: types.Coords{
			Latitude:  39.11539,
			Longitude: -107.6584,
		},
		Elevation: types.Elevation{
			Meters: 4352,
		},
	}

	// Map to our forecast structure
	forecast, err := mapForecastAPIResponseToForecast(forecastPoint, ModelGfsSeamless, &apiResponse)
	if err != nil {
		t.Fatalf("Failed to map forecast: %v", err)
	}

	// Validate basic properties
	if forecast.Timezone != "America/Denver" {
		t.Errorf("Expected timezone America/Denver, got %s", forecast.Timezone)
	}

	if forecast.PrimaryModel != ModelGfsSeamless {
		t.Errorf("Expected primary model %s, got %s", ModelGfsSeamless, forecast.PrimaryModel)
	}

	// Validate current conditions were populated
	if len(forecast.CurrentConditions.Temperature) == 0 {
		t.Error("Current conditions temperature map is empty")
	}

	if len(forecast.CurrentConditions.Wind) == 0 {
		t.Error("Current conditions wind map is empty")
	}

	// Validate we have temperature for GFS model
	gfsTemp, ok := forecast.CurrentConditions.Temperature.GetForModel(ModelGfsSeamless)
	if !ok {
		t.Error("GFS model not found in temperature map")
	} else {
		// Temperature should be reasonable (between -50°F and 120°F)
		if gfsTemp.Fahrenheit < -50 || gfsTemp.Fahrenheit > 120 {
			t.Errorf("GFS temperature %v°F seems unreasonable", gfsTemp.Fahrenheit)
		}
		t.Logf("Current GFS temperature: %.1f°F", gfsTemp.Fahrenheit)
	}

	// Validate we have wind for GFS model
	gfsWind, ok := forecast.CurrentConditions.Wind.GetForModel(ModelGfsSeamless)
	if !ok {
		t.Error("GFS model not found in wind map")
	} else {
		// Wind speed should be non-negative and reasonable (< 200 mph)
		if gfsWind.Speed.Mph < 0 || gfsWind.Speed.Mph > 200 {
			t.Errorf("GFS wind speed %v mph seems unreasonable", gfsWind.Speed.Mph)
		}
		t.Logf("Current GFS wind: %.1f mph from %.0f°", gfsWind.Speed.Mph, gfsWind.Direction.Degrees)
	}

	// Validate daily forecasts
	if len(forecast.DailyForecasts) != 16 {
		t.Errorf("Expected 16 daily forecasts, got %d", len(forecast.DailyForecasts))
	}

	// Validate first daily forecast
	if len(forecast.DailyForecasts) > 0 {
		firstDay := forecast.DailyForecasts[0]

		// Should have weather data for GFS model
		gfsDailyWeather, ok := firstDay.Weather.GetForModel(ModelGfsSeamless)
		if !ok {
			t.Error("GFS model not found in daily weather map")
		} else {
			// Weather code should be valid (0-99)
			if gfsDailyWeather.Code < 0 || gfsDailyWeather.Code > 99 {
				t.Errorf("GFS daily weather code %d seems invalid", gfsDailyWeather.Code)
			}
			t.Logf("First day GFS weather code: %d (%s)", gfsDailyWeather.Code, gfsDailyWeather.Description)
		}

		// Should have snowfall data
		gfsSnowfall, ok := firstDay.SnowfallWaterEquivalentSum.GetForModel(ModelGfsSeamless)
		if !ok {
			t.Error("GFS model not found in snowfall map")
		} else {
			// Snowfall should be non-negative
			if gfsSnowfall < 0 {
				t.Errorf("GFS snowfall %v should be non-negative", gfsSnowfall)
			}
			t.Logf("First day GFS snowfall water equivalent: %.2f inches", gfsSnowfall)
		}

		// Validate sunrise/sunset
		gfsSunrise, ok := firstDay.Sunrise.GetForModel(ModelGfsSeamless)
		if !ok {
			t.Error("GFS model not found in sunrise map")
		} else if gfsSunrise.IsZero() {
			t.Error("GFS sunrise time is zero")
		} else {
			t.Logf("First day sunrise: %s", gfsSunrise.Format("15:04"))
		}

		gfsSunset, ok := firstDay.Sunset.GetForModel(ModelGfsSeamless)
		if !ok {
			t.Error("GFS model not found in sunset map")
		} else if gfsSunset.IsZero() {
			t.Error("GFS sunset time is zero")
		} else {
			t.Logf("First day sunset: %s", gfsSunset.Format("15:04"))
		}
	}

	// Validate we have multiple models
	modelCount := len(forecast.CurrentConditions.Temperature)
	if modelCount != 7 {
		t.Errorf("Expected 7 models in temperature map, got %d", modelCount)
	}

	t.Logf("Successfully mapped forecast with %d daily forecasts and %d models",
		len(forecast.DailyForecasts), modelCount)
}
