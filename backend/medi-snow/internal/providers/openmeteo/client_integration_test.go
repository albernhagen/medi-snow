//go:build integration

package openmeteo

import (
	"encoding/json"
	"testing"
)

func TestForecastClient_GetForecast_Integration(t *testing.T) {
	// Test coordinates: Aspen, CO area
	lat := 39.11539
	lon := -107.65840
	elevation := 4352.0 // meters
	forecastDays := 1

	client := NewClient()

	t.Logf("Making API call to OpenMeteo Forecast API...")
	t.Logf("Coordinates: lat=%f, lon=%f, elevation=%f meters", lat, lon, elevation)

	resp, err := client.GetForecast(lat, lon, elevation, forecastDays)
	if err != nil {
		t.Fatalf("Failed to get forecast: %v", err)
	}

	// Pretty print the raw response (truncated for readability)
	rawJSON, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	t.Logf("Raw API Response:\n%s", string(rawJSON))

	// Verify response structure
	if resp == nil {
		t.Fatal("Response is nil")
	}

	// Check basic metadata
	t.Logf("Response metadata:")
	t.Logf("  Latitude: %f", resp.Latitude)
	t.Logf("  Longitude: %f", resp.Longitude)
	t.Logf("  Elevation: %f meters", resp.Elevation)
	t.Logf("  Timezone: %s", resp.Timezone)
	t.Logf("  Generation time: %.2f ms", resp.GenerationtimeMs)

	if resp.Latitude < lat-1 || resp.Latitude > lat+1 {
		t.Errorf("Latitude mismatch: expected ~%f, got %f", lat, resp.Latitude)
	}

	if resp.Longitude < lon-1 || resp.Longitude > lon+1 {
		t.Errorf("Longitude mismatch: expected ~%f, got %f", lon, resp.Longitude)
	}

	// Check hourly data
	if len(resp.Hourly.Time) == 0 {
		t.Fatal("No hourly time data")
	}

	t.Logf("Hourly forecast contains %d time points", len(resp.Hourly.Time))
	t.Logf("First timestamp: %s", resp.Hourly.Time[0])
	if len(resp.Hourly.Time) > 1 {
		t.Logf("Last timestamp: %s", resp.Hourly.Time[len(resp.Hourly.Time)-1])
	}

	// Check GFS Seamless data (should always be present)
	if len(resp.Hourly.Temperature2MGfsSeamless) > 0 {
		t.Logf("Sample GFS Seamless data (first point):")
		t.Logf("  Temperature: %.1f°F", resp.Hourly.Temperature2MGfsSeamless[0])
		t.Logf("  Wind Speed: %.1f mph", resp.Hourly.WindSpeed10MGfsSeamless[0])
		t.Logf("  Snowfall: %.2f in", resp.Hourly.SnowfallGfsSeamless[0])
	} else {
		t.Error("No GFS Seamless temperature data")
	}

	// Check daily data
	if len(resp.Daily.Time) == 0 {
		t.Fatal("No daily time data")
	}

	t.Logf("Daily forecast contains %d days", len(resp.Daily.Time))
	if len(resp.Daily.SunriseGfsSeamless) > 0 && len(resp.Daily.SunsetGfsSeamless) > 0 {
		t.Logf("Day 1 - Sunrise: %s, Sunset: %s",
			resp.Daily.SunriseGfsSeamless[0],
			resp.Daily.SunsetGfsSeamless[0])
	}

	t.Log("✓ API call successful, response structure valid")
}

// truncate limits a string to maxLen characters
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n... (truncated)"
}
