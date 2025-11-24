package weather

import (
	"encoding/json"
	"medi-snow/internal/providers/openmeteo"
	"medi-snow/internal/types"
	"os"
	"testing"
	"time"
)

func TestToPercentage(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected float64
	}{
		{
			name:     "100 percent",
			input:    100,
			expected: 1.0,
		},
		{
			name:     "50 percent",
			input:    50,
			expected: 0.5,
		},
		{
			name:     "0 percent",
			input:    0,
			expected: 0.0,
		},
		{
			name:     "75 percent",
			input:    75,
			expected: 0.75,
		},
		{
			name:     "1 percent",
			input:    1,
			expected: 0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toPercentage(tt.input)
			if result != tt.expected {
				t.Errorf("toPercentage(%d) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToTime(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectNil bool
	}{
		{
			name:      "valid time",
			input:     "2025-01-15T10:30",
			expectNil: false,
		},
		{
			name:      "invalid format",
			input:     "not a time",
			expectNil: true,
		},
		{
			name:      "empty string",
			input:     "",
			expectNil: true,
		},
		{
			name:      "different format",
			input:     "2025-01-15 10:30:00",
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toTime(tt.input)

			if tt.expectNil {
				if !result.IsZero() {
					t.Errorf("toTime(%q) expected zero time, got %v", tt.input, result)
				}
			} else {
				if result.IsZero() {
					t.Errorf("toTime(%q) expected non-zero time, got zero", tt.input)
				}

				// Verify the parsed time matches input
				expected, _ := time.Parse("2006-01-02T15:04", tt.input)
				if !result.Equal(expected) {
					t.Errorf("toTime(%q) = %v, want %v", tt.input, result, expected)
				}
			}
		})
	}
}

func TestMinFloat(t *testing.T) {
	tests := []struct {
		name     string
		input    []float64
		expected float64
	}{
		{
			name:     "single value",
			input:    []float64{5.5},
			expected: 5.5,
		},
		{
			name:     "multiple values",
			input:    []float64{5.5, 2.2, 8.8, 1.1},
			expected: 1.1,
		},
		{
			name:     "negative values",
			input:    []float64{-5.5, -2.2, -8.8},
			expected: -8.8,
		},
		{
			name:     "mixed positive and negative",
			input:    []float64{5.5, -2.2, 8.8},
			expected: -2.2,
		},
		{
			name:     "empty slice",
			input:    []float64{},
			expected: -1,
		},
		{
			name:     "all same values",
			input:    []float64{3.0, 3.0, 3.0},
			expected: 3.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := minFloat(tt.input)
			if result != tt.expected {
				t.Errorf("minFloat(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMaxFloat(t *testing.T) {
	tests := []struct {
		name     string
		input    []float64
		expected float64
	}{
		{
			name:     "single value",
			input:    []float64{5.5},
			expected: 5.5,
		},
		{
			name:     "multiple values",
			input:    []float64{5.5, 2.2, 8.8, 1.1},
			expected: 8.8,
		},
		{
			name:     "negative values",
			input:    []float64{-5.5, -2.2, -8.8},
			expected: -2.2,
		},
		{
			name:     "mixed positive and negative",
			input:    []float64{5.5, -2.2, 8.8},
			expected: 8.8,
		},
		{
			name:     "empty slice",
			input:    []float64{},
			expected: -1,
		},
		{
			name:     "all same values",
			input:    []float64{3.0, 3.0, 3.0},
			expected: 3.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maxFloat(tt.input)
			if result != tt.expected {
				t.Errorf("maxFloat(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSum(t *testing.T) {
	tests := []struct {
		name     string
		input    []float64
		expected float64
	}{
		{
			name:     "single value",
			input:    []float64{5.5},
			expected: 5.5,
		},
		{
			name:     "multiple values",
			input:    []float64{1.0, 2.0, 3.0},
			expected: 6.0,
		},
		{
			name:     "negative values",
			input:    []float64{-1.0, -2.0, -3.0},
			expected: -6.0,
		},
		{
			name:     "mixed positive and negative",
			input:    []float64{5.0, -2.0, 3.0},
			expected: 6.0,
		},
		{
			name:     "empty slice",
			input:    []float64{},
			expected: 0.0,
		},
		{
			name:     "zero values",
			input:    []float64{0.0, 0.0, 0.0},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sum(tt.input)
			if result != tt.expected {
				t.Errorf("sum(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMapForecastAPIResponseToForecast(t *testing.T) {
	// Load real API response from testdata file
	data, err := os.ReadFile("testdata/openmeteo_forecast_response.json")
	if err != nil {
		t.Fatalf("Failed to read testdata file: %v", err)
	}

	var apiResponse openmeteo.ForecastAPIResponse
	if err := json.Unmarshal(data, &apiResponse); err != nil {
		t.Fatalf("Failed to unmarshal API response: %v", err)
	}

	// Create forecast point matching the API response (Aspen, Colorado)
	forecastPoint := types.ForecastPoint{
		Coordinates: types.Coords{
			Latitude:  39.11539,
			Longitude: -107.6584,
		},
		Elevation: types.Elevation{
			Meters: 4352,
		},
	}

	forecast, err := mapForecastAPIResponseToForecast(forecastPoint, ModelGfsSeamless, &apiResponse)

	if err != nil {
		t.Fatalf("mapForecastAPIResponseToForecast returned error: %v", err)
	}

	// Verify basic forecast properties
	if forecast.Timezone != "America/Denver" {
		t.Errorf("Timezone = %v, want America/Denver", forecast.Timezone)
	}

	if forecast.PrimaryModel != ModelGfsSeamless {
		t.Errorf("PrimaryModel = %v, want %v", forecast.PrimaryModel, ModelGfsSeamless)
	}

	if forecast.ForecastPoint.Coordinates.Latitude != forecastPoint.Coordinates.Latitude {
		t.Errorf("ForecastPoint.Latitude = %v, want %v", forecast.ForecastPoint.Coordinates.Latitude, forecastPoint.Coordinates.Latitude)
	}

	// Verify current conditions were mapped (should have 7 models)
	if len(forecast.CurrentConditions.Temperature) != 7 {
		t.Errorf("CurrentConditions.Temperature has %d models, want 7", len(forecast.CurrentConditions.Temperature))
	}

	// Verify temperature mapping for GFS model (values from real API will vary)
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

	// Verify wind mapping for GFS model
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

	// Verify daily forecasts were created (should be 16 days based on real API)
	if len(forecast.DailyForecasts) != 16 {
		t.Errorf("DailyForecasts has %d days, want 16", len(forecast.DailyForecasts))
	}

	// Verify first daily forecast has valid data
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

	t.Logf("Successfully mapped forecast with %d daily forecasts and %d models",
		len(forecast.DailyForecasts), len(forecast.CurrentConditions.Temperature))
}

func TestMapForecastAPIResponseToForecast_InvalidTimezone(t *testing.T) {
	apiResponse := &openmeteo.ForecastAPIResponse{
		Timezone: "Invalid/Timezone",
		Hourly: struct {
			Time                                       []string      `json:"time"`
			FreezingLevelHeightGemSeamless             []interface{} `json:"freezing_level_height_gem_seamless"`
			IsDayGemSeamless                           []int         `json:"is_day_gem_seamless"`
			Temperature2MGemSeamless                   []float64     `json:"temperature_2m_gem_seamless"`
			WeatherCodeGemSeamless                     []int         `json:"weather_code_gem_seamless"`
			ApparentTemperatureGemSeamless             []float64     `json:"apparent_temperature_gem_seamless"`
			PrecipitationProbabilityGemSeamless        []int         `json:"precipitation_probability_gem_seamless"`
			PrecipitationGemSeamless                   []float64     `json:"precipitation_gem_seamless"`
			CloudCoverGemSeamless                      []int         `json:"cloud_cover_gem_seamless"`
			CloudCoverLowGemSeamless                   []int         `json:"cloud_cover_low_gem_seamless"`
			CloudCoverMidGemSeamless                   []int         `json:"cloud_cover_mid_gem_seamless"`
			CloudCoverHighGemSeamless                  []int         `json:"cloud_cover_high_gem_seamless"`
			VisibilityGemSeamless                      []interface{} `json:"visibility_gem_seamless"`
			WindSpeed10MGemSeamless                    []float64     `json:"wind_speed_10m_gem_seamless"`
			WindDirection10MGemSeamless                []int         `json:"wind_direction_10m_gem_seamless"`
			WindGusts10MGemSeamless                    []float64     `json:"wind_gusts_10m_gem_seamless"`
			RelativeHumidity2MGemSeamless              []int         `json:"relative_humidity_2m_gem_seamless"`
			RainGemSeamless                            []float64     `json:"rain_gem_seamless"`
			ShowersGemSeamless                         []float64     `json:"showers_gem_seamless"`
			SnowfallGemSeamless                        []float64     `json:"snowfall_gem_seamless"`
			SnowDepthGemSeamless                       []float64     `json:"snow_depth_gem_seamless"`
			FreezingLevelHeightEcmwfIfs                []interface{} `json:"freezing_level_height_ecmwf_ifs"`
			IsDayEcmwfIfs                              []int         `json:"is_day_ecmwf_ifs"`
			Temperature2MEcmwfIfs                      []float64     `json:"temperature_2m_ecmwf_ifs"`
			WeatherCodeEcmwfIfs                        []int         `json:"weather_code_ecmwf_ifs"`
			ApparentTemperatureEcmwfIfs                []float64     `json:"apparent_temperature_ecmwf_ifs"`
			PrecipitationProbabilityEcmwfIfs           []int         `json:"precipitation_probability_ecmwf_ifs"`
			PrecipitationEcmwfIfs                      []float64     `json:"precipitation_ecmwf_ifs"`
			CloudCoverEcmwfIfs                         []int         `json:"cloud_cover_ecmwf_ifs"`
			CloudCoverLowEcmwfIfs                      []int         `json:"cloud_cover_low_ecmwf_ifs"`
			CloudCoverMidEcmwfIfs                      []int         `json:"cloud_cover_mid_ecmwf_ifs"`
			CloudCoverHighEcmwfIfs                     []int         `json:"cloud_cover_high_ecmwf_ifs"`
			VisibilityEcmwfIfs                         []float64     `json:"visibility_ecmwf_ifs"`
			WindSpeed10MEcmwfIfs                       []float64     `json:"wind_speed_10m_ecmwf_ifs"`
			WindDirection10MEcmwfIfs                   []int         `json:"wind_direction_10m_ecmwf_ifs"`
			WindGusts10MEcmwfIfs                       []float64     `json:"wind_gusts_10m_ecmwf_ifs"`
			RelativeHumidity2MEcmwfIfs                 []int         `json:"relative_humidity_2m_ecmwf_ifs"`
			RainEcmwfIfs                               []float64     `json:"rain_ecmwf_ifs"`
			ShowersEcmwfIfs                            []float64     `json:"showers_ecmwf_ifs"`
			SnowfallEcmwfIfs                           []float64     `json:"snowfall_ecmwf_ifs"`
			SnowDepthEcmwfIfs                          []float64     `json:"snow_depth_ecmwf_ifs"`
			FreezingLevelHeightGfsSeamless             []float64     `json:"freezing_level_height_gfs_seamless"`
			IsDayGfsSeamless                           []int         `json:"is_day_gfs_seamless"`
			Temperature2MGfsSeamless                   []float64     `json:"temperature_2m_gfs_seamless"`
			WeatherCodeGfsSeamless                     []int         `json:"weather_code_gfs_seamless"`
			ApparentTemperatureGfsSeamless             []float64     `json:"apparent_temperature_gfs_seamless"`
			PrecipitationProbabilityGfsSeamless        []int         `json:"precipitation_probability_gfs_seamless"`
			PrecipitationGfsSeamless                   []float64     `json:"precipitation_gfs_seamless"`
			CloudCoverGfsSeamless                      []int         `json:"cloud_cover_gfs_seamless"`
			CloudCoverLowGfsSeamless                   []int         `json:"cloud_cover_low_gfs_seamless"`
			CloudCoverMidGfsSeamless                   []int         `json:"cloud_cover_mid_gfs_seamless"`
			CloudCoverHighGfsSeamless                  []int         `json:"cloud_cover_high_gfs_seamless"`
			VisibilityGfsSeamless                      []float64     `json:"visibility_gfs_seamless"`
			WindSpeed10MGfsSeamless                    []float64     `json:"wind_speed_10m_gfs_seamless"`
			WindDirection10MGfsSeamless                []int         `json:"wind_direction_10m_gfs_seamless"`
			WindGusts10MGfsSeamless                    []float64     `json:"wind_gusts_10m_gfs_seamless"`
			RelativeHumidity2MGfsSeamless              []int         `json:"relative_humidity_2m_gfs_seamless"`
			RainGfsSeamless                            []float64     `json:"rain_gfs_seamless"`
			ShowersGfsSeamless                         []float64     `json:"showers_gfs_seamless"`
			SnowfallGfsSeamless                        []float64     `json:"snowfall_gfs_seamless"`
			SnowDepthGfsSeamless                       []float64     `json:"snow_depth_gfs_seamless"`
			FreezingLevelHeightNcepNbmConus            []interface{} `json:"freezing_level_height_ncep_nbm_conus"`
			IsDayNcepNbmConus                          []int         `json:"is_day_ncep_nbm_conus"`
			Temperature2MNcepNbmConus                  []float64     `json:"temperature_2m_ncep_nbm_conus"`
			WeatherCodeNcepNbmConus                    []int         `json:"weather_code_ncep_nbm_conus"`
			ApparentTemperatureNcepNbmConus            []float64     `json:"apparent_temperature_ncep_nbm_conus"`
			PrecipitationProbabilityNcepNbmConus       []int         `json:"precipitation_probability_ncep_nbm_conus"`
			PrecipitationNcepNbmConus                  []float64     `json:"precipitation_ncep_nbm_conus"`
			CloudCoverNcepNbmConus                     []int         `json:"cloud_cover_ncep_nbm_conus"`
			CloudCoverLowNcepNbmConus                  []interface{} `json:"cloud_cover_low_ncep_nbm_conus"`
			CloudCoverMidNcepNbmConus                  []interface{} `json:"cloud_cover_mid_ncep_nbm_conus"`
			CloudCoverHighNcepNbmConus                 []interface{} `json:"cloud_cover_high_ncep_nbm_conus"`
			VisibilityNcepNbmConus                     []float64     `json:"visibility_ncep_nbm_conus"`
			WindSpeed10MNcepNbmConus                   []float64     `json:"wind_speed_10m_ncep_nbm_conus"`
			WindDirection10MNcepNbmConus               []int         `json:"wind_direction_10m_ncep_nbm_conus"`
			WindGusts10MNcepNbmConus                   []float64     `json:"wind_gusts_10m_ncep_nbm_conus"`
			RelativeHumidity2MNcepNbmConus             []int         `json:"relative_humidity_2m_ncep_nbm_conus"`
			RainNcepNbmConus                           []float64     `json:"rain_ncep_nbm_conus"`
			ShowersNcepNbmConus                        []float64     `json:"showers_ncep_nbm_conus"`
			SnowfallNcepNbmConus                       []float64     `json:"snowfall_ncep_nbm_conus"`
			SnowDepthNcepNbmConus                      []interface{} `json:"snow_depth_ncep_nbm_conus"`
			FreezingLevelHeightGfsGraphcast025         []interface{} `json:"freezing_level_height_gfs_graphcast025"`
			IsDayGfsGraphcast025                       []int         `json:"is_day_gfs_graphcast025"`
			Temperature2MGfsGraphcast025               []float64     `json:"temperature_2m_gfs_graphcast025"`
			WeatherCodeGfsGraphcast025                 []int         `json:"weather_code_gfs_graphcast025"`
			ApparentTemperatureGfsGraphcast025         []interface{} `json:"apparent_temperature_gfs_graphcast025"`
			PrecipitationProbabilityGfsGraphcast025    []interface{} `json:"precipitation_probability_gfs_graphcast025"`
			PrecipitationGfsGraphcast025               []float64     `json:"precipitation_gfs_graphcast025"`
			CloudCoverGfsGraphcast025                  []int         `json:"cloud_cover_gfs_graphcast025"`
			CloudCoverLowGfsGraphcast025               []int         `json:"cloud_cover_low_gfs_graphcast025"`
			CloudCoverMidGfsGraphcast025               []int         `json:"cloud_cover_mid_gfs_graphcast025"`
			CloudCoverHighGfsGraphcast025              []int         `json:"cloud_cover_high_gfs_graphcast025"`
			VisibilityGfsGraphcast025                  []interface{} `json:"visibility_gfs_graphcast025"`
			WindSpeed10MGfsGraphcast025                []float64     `json:"wind_speed_10m_gfs_graphcast025"`
			WindDirection10MGfsGraphcast025            []int         `json:"wind_direction_10m_gfs_graphcast025"`
			WindGusts10MGfsGraphcast025                []interface{} `json:"wind_gusts_10m_gfs_graphcast025"`
			RelativeHumidity2MGfsGraphcast025          []interface{} `json:"relative_humidity_2m_gfs_graphcast025"`
			RainGfsGraphcast025                        []float64     `json:"rain_gfs_graphcast025"`
			ShowersGfsGraphcast025                     []float64     `json:"showers_gfs_graphcast025"`
			SnowfallGfsGraphcast025                    []float64     `json:"snowfall_gfs_graphcast025"`
			SnowDepthGfsGraphcast025                   []interface{} `json:"snow_depth_gfs_graphcast025"`
			FreezingLevelHeightEcmwfAifs025Single      []interface{} `json:"freezing_level_height_ecmwf_aifs025_single"`
			IsDayEcmwfAifs025Single                    []int         `json:"is_day_ecmwf_aifs025_single"`
			Temperature2MEcmwfAifs025Single            []float64     `json:"temperature_2m_ecmwf_aifs025_single"`
			WeatherCodeEcmwfAifs025Single              []int         `json:"weather_code_ecmwf_aifs025_single"`
			ApparentTemperatureEcmwfAifs025Single      []float64     `json:"apparent_temperature_ecmwf_aifs025_single"`
			PrecipitationProbabilityEcmwfAifs025Single []interface{} `json:"precipitation_probability_ecmwf_aifs025_single"`
			PrecipitationEcmwfAifs025Single            []float64     `json:"precipitation_ecmwf_aifs025_single"`
			CloudCoverEcmwfAifs025Single               []int         `json:"cloud_cover_ecmwf_aifs025_single"`
			CloudCoverLowEcmwfAifs025Single            []int         `json:"cloud_cover_low_ecmwf_aifs025_single"`
			CloudCoverMidEcmwfAifs025Single            []int         `json:"cloud_cover_mid_ecmwf_aifs025_single"`
			CloudCoverHighEcmwfAifs025Single           []int         `json:"cloud_cover_high_ecmwf_aifs025_single"`
			VisibilityEcmwfAifs025Single               []interface{} `json:"visibility_ecmwf_aifs025_single"`
			WindSpeed10MEcmwfAifs025Single             []float64     `json:"wind_speed_10m_ecmwf_aifs025_single"`
			WindDirection10MEcmwfAifs025Single         []int         `json:"wind_direction_10m_ecmwf_aifs025_single"`
			WindGusts10MEcmwfAifs025Single             []interface{} `json:"wind_gusts_10m_ecmwf_aifs025_single"`
			RelativeHumidity2MEcmwfAifs025Single       []int         `json:"relative_humidity_2m_ecmwf_aifs025_single"`
			RainEcmwfAifs025Single                     []float64     `json:"rain_ecmwf_aifs025_single"`
			ShowersEcmwfAifs025Single                  []float64     `json:"showers_ecmwf_aifs025_single"`
			SnowfallEcmwfAifs025Single                 []float64     `json:"snowfall_ecmwf_aifs025_single"`
			SnowDepthEcmwfAifs025Single                []interface{} `json:"snow_depth_ecmwf_aifs025_single"`
			FreezingLevelHeightNcepNamConus            []interface{} `json:"freezing_level_height_ncep_nam_conus"`
			IsDayNcepNamConus                          []int         `json:"is_day_ncep_nam_conus"`
			Temperature2MNcepNamConus                  []float64     `json:"temperature_2m_ncep_nam_conus"`
			WeatherCodeNcepNamConus                    []int         `json:"weather_code_ncep_nam_conus"`
			ApparentTemperatureNcepNamConus            []float64     `json:"apparent_temperature_ncep_nam_conus"`
			PrecipitationProbabilityNcepNamConus       []interface{} `json:"precipitation_probability_ncep_nam_conus"`
			PrecipitationNcepNamConus                  []float64     `json:"precipitation_ncep_nam_conus"`
			CloudCoverNcepNamConus                     []int         `json:"cloud_cover_ncep_nam_conus"`
			CloudCoverLowNcepNamConus                  []int         `json:"cloud_cover_low_ncep_nam_conus"`
			CloudCoverMidNcepNamConus                  []int         `json:"cloud_cover_mid_ncep_nam_conus"`
			CloudCoverHighNcepNamConus                 []int         `json:"cloud_cover_high_ncep_nam_conus"`
			VisibilityNcepNamConus                     []float64     `json:"visibility_ncep_nam_conus"`
			WindSpeed10MNcepNamConus                   []float64     `json:"wind_speed_10m_ncep_nam_conus"`
			WindDirection10MNcepNamConus               []int         `json:"wind_direction_10m_ncep_nam_conus"`
			WindGusts10MNcepNamConus                   []float64     `json:"wind_gusts_10m_ncep_nam_conus"`
			RelativeHumidity2MNcepNamConus             []int         `json:"relative_humidity_2m_ncep_nam_conus"`
			RainNcepNamConus                           []float64     `json:"rain_ncep_nam_conus"`
			ShowersNcepNamConus                        []float64     `json:"showers_ncep_nam_conus"`
			SnowfallNcepNamConus                       []float64     `json:"snowfall_ncep_nam_conus"`
			SnowDepthNcepNamConus                      []float64     `json:"snow_depth_ncep_nam_conus"`
		}{
			Time: []string{"2025-01-23T00:00"},
		},
		Daily: struct {
			Time                                         []string      `json:"time"`
			SnowfallWaterEquivalentSumGemSeamless        []float64     `json:"snowfall_water_equivalent_sum_gem_seamless"`
			WeatherCodeGemSeamless                       []int         `json:"weather_code_gem_seamless"`
			SunriseGemSeamless                           []string      `json:"sunrise_gem_seamless"`
			SunsetGemSeamless                            []string      `json:"sunset_gem_seamless"`
			WindDirection10MDominantGemSeamless          []int         `json:"wind_direction_10m_dominant_gem_seamless"`
			SnowfallWaterEquivalentSumEcmwfIfs           []float64     `json:"snowfall_water_equivalent_sum_ecmwf_ifs"`
			WeatherCodeEcmwfIfs                          []int         `json:"weather_code_ecmwf_ifs"`
			SunriseEcmwfIfs                              []string      `json:"sunrise_ecmwf_ifs"`
			SunsetEcmwfIfs                               []string      `json:"sunset_ecmwf_ifs"`
			WindDirection10MDominantEcmwfIfs             []int         `json:"wind_direction_10m_dominant_ecmwf_ifs"`
			SnowfallWaterEquivalentSumGfsSeamless        []float64     `json:"snowfall_water_equivalent_sum_gfs_seamless"`
			WeatherCodeGfsSeamless                       []int         `json:"weather_code_gfs_seamless"`
			SunriseGfsSeamless                           []string      `json:"sunrise_gfs_seamless"`
			SunsetGfsSeamless                            []string      `json:"sunset_gfs_seamless"`
			WindDirection10MDominantGfsSeamless          []int         `json:"wind_direction_10m_dominant_gfs_seamless"`
			SnowfallWaterEquivalentSumNcepNbmConus       []float64     `json:"snowfall_water_equivalent_sum_ncep_nbm_conus"`
			WeatherCodeNcepNbmConus                      []int         `json:"weather_code_ncep_nbm_conus"`
			SunriseNcepNbmConus                          []string      `json:"sunrise_ncep_nbm_conus"`
			SunsetNcepNbmConus                           []string      `json:"sunset_ncep_nbm_conus"`
			WindDirection10MDominantNcepNbmConus         []int         `json:"wind_direction_10m_dominant_ncep_nbm_conus"`
			SnowfallWaterEquivalentSumGfsGraphcast025    []interface{} `json:"snowfall_water_equivalent_sum_gfs_graphcast025"`
			WeatherCodeGfsGraphcast025                   []int         `json:"weather_code_gfs_graphcast025"`
			SunriseGfsGraphcast025                       []string      `json:"sunrise_gfs_graphcast025"`
			SunsetGfsGraphcast025                        []string      `json:"sunset_gfs_graphcast025"`
			WindDirection10MDominantGfsGraphcast025      []int         `json:"wind_direction_10m_dominant_gfs_graphcast025"`
			SnowfallWaterEquivalentSumEcmwfAifs025Single []float64     `json:"snowfall_water_equivalent_sum_ecmwf_aifs025_single"`
			WeatherCodeEcmwfAifs025Single                []int         `json:"weather_code_ecmwf_aifs025_single"`
			SunriseEcmwfAifs025Single                    []string      `json:"sunrise_ecmwf_aifs025_single"`
			SunsetEcmwfAifs025Single                     []string      `json:"sunset_ecmwf_aifs025_single"`
			WindDirection10MDominantEcmwfAifs025Single   []int         `json:"wind_direction_10m_dominant_ecmwf_aifs025_single"`
			SnowfallWaterEquivalentSumNcepNamConus       []float64     `json:"snowfall_water_equivalent_sum_ncep_nam_conus"`
			WeatherCodeNcepNamConus                      []int         `json:"weather_code_ncep_nam_conus"`
			SunriseNcepNamConus                          []string      `json:"sunrise_ncep_nam_conus"`
			SunsetNcepNamConus                           []string      `json:"sunset_ncep_nam_conus"`
			WindDirection10MDominantNcepNamConus         []int         `json:"wind_direction_10m_dominant_ncep_nam_conus"`
		}{
			Time: []string{},
		},
	}

	forecastPoint := types.ForecastPoint{}

	_, err := mapForecastAPIResponseToForecast(forecastPoint, ModelGfsSeamless, apiResponse)

	if err == nil {
		t.Fatal("Expected error for invalid timezone, got nil")
	}
}
