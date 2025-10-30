package openmeteo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// API Docs: https://open-meteo.com/en/docs
// Sample request: https://api.open-meteo.com/v1/forecast?latitude=39.11&longitude=-107.65&daily=snowfall_water_equivalent_sum,weather_code,sunrise,sunset,wind_direction_10m_dominant&hourly=freezing_level_height,is_day,temperature_2m,weather_code,apparent_temperature,precipitation_probability,precipitation,cloud_cover,cloud_cover_low,cloud_cover_mid,cloud_cover_high,visibility,wind_speed_10m,wind_direction_10m,wind_gusts_10m,relative_humidity_2m,rain,showers,snowfall,snow_depth&models=gem_seamless,ecmwf_ifs,gfs_seamless,ncep_nbm_conus,gfs_graphcast025,ecmwf_aifs025_single,ncep_nam_conus&timezone=GMT&forecast_days=16&timeformat=iso8601&wind_speed_unit=mph&temperature_unit=fahrenheit&precipitation_unit=inch
const (
	baseForecastURL = "https://api.open-meteo.com/v1/forecast"
)

type ForecastClient struct {
	httpClient *http.Client
	baseURL    string
}

func NewForecastClient() *ForecastClient {
	return &ForecastClient{
		httpClient: &http.Client{},
		baseURL:    baseForecastURL,
	}
}

// GetForecast fetches the weather forecast for the given latitude, longitude, and elevation in meters
func (c *ForecastClient) GetForecast(latitude, longitude, elevationMeters float64, forecastDays int) (*ForecastAPIResponse, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	hourlyVars := []string{
		"freezing_level_height",
		"is_day",
		"temperature_2m",
		"weather_code",
		"apparent_temperature",
		"precipitation_probability",
		"precipitation",
		"cloud_cover",
		"cloud_cover_low",
		"cloud_cover_mid",
		"cloud_cover_high",
		"visibility",
		"wind_speed_10m",
		"wind_direction_10m",
		"wind_gusts_10m",
		"relative_humidity_2m",
		"rain,showers",
		"snowfall",
		"snow_depth",
	}

	dailyVars := []string{
		"snowfall_water_equivalent_sum",
		"weather_code",
		"sunrise",
		"sunset",
		"wind_direction_10m_dominant",
	}

	modelVars := []string{
		"gem_seamless",
		"ecmwf_ifs",
		"gfs_seamless",
		"ncep_nbm_conus",
		"gfs_graphcast025",
		"ecmwf_aifs025_single",
		"ncep_nam_conus",
	}

	q := u.Query()

	q.Set("latitude", fmt.Sprintf("%f", latitude))
	q.Set("longitude", fmt.Sprintf("%f", longitude))
	q.Set("elevation", fmt.Sprintf("%f", elevationMeters))
	q.Set("hourly", strings.Join(hourlyVars, ","))
	q.Set("daily", strings.Join(dailyVars, ","))
	q.Set("models", strings.Join(modelVars, ","))

	q.Set("timezone", "GMT")
	q.Set("forecast_days", strconv.Itoa(forecastDays))
	q.Set("timeformat", "iso8601")
	q.Set("wind_speed_unit", "mph")
	q.Set("temperature_unit", "fahrenheit")
	q.Set("precipitation_unit", "inch")
	u.RawQuery = q.Encode()

	resp, err := c.httpClient.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fetch returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp ForecastAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &apiResp, nil
}
