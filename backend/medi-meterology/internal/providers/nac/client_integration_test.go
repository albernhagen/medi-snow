//go:build integration

package nac

import (
	"encoding/json"
	"log/slog"
	"os"
	"testing"
)

func TestClient_GetMapLayer_Integration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	client := NewClient(logger)

	t.Log("Making API call to NAC map-layer endpoint...")

	resp, err := client.GetMapLayer()
	if err != nil {
		t.Fatalf("Failed to get map layer: %v", err)
	}

	if resp == nil {
		t.Fatal("Response is nil")
	}

	t.Logf("Map Layer Details:")
	t.Logf("  Type: %s", resp.Type)
	t.Logf("  Feature Count: %d", len(resp.Features))

	if resp.Type != "FeatureCollection" {
		t.Errorf("Expected type FeatureCollection, got %s", resp.Type)
	}

	if len(resp.Features) == 0 {
		t.Fatal("No features returned")
	}

	// Log first feature as sample
	first := resp.Features[0]
	t.Logf("  Sample Feature:")
	t.Logf("    ID: %d", first.Id)
	t.Logf("    Name: %s", first.Properties.Name)
	t.Logf("    Center ID: %s", first.Properties.CenterId)
	t.Logf("    Danger Level: %d", first.Properties.DangerLevel)
	t.Logf("    Geometry Type: %s", first.Geometry.Type)
	t.Logf("    Ring Count: %d", len(first.Geometry.Coordinates()))

	t.Log("✓ GetMapLayer API call successful, response structure valid")
}

func TestClient_GetForecast_Integration(t *testing.T) {
	centerId := "CAIC"
	zoneId := 2690

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	client := NewClient(logger)

	t.Logf("Making API call to NAC forecast endpoint...")
	t.Logf("Center ID: %s, Zone ID: %d", centerId, zoneId)

	resp, err := client.GetForecast(centerId, zoneId)
	if err != nil {
		t.Fatalf("Failed to get forecast: %v", err)
	}

	if resp == nil {
		t.Fatal("Response is nil")
	}

	rawJSON, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	// Truncate for readability
	output := string(rawJSON)
	if len(output) > 2000 {
		output = output[:2000] + "\n... (truncated)"
	}
	t.Logf("Raw API Response:\n%s", output)

	t.Logf("Forecast Details:")
	t.Logf("  ID: %d", resp.Id)
	t.Logf("  Product Type: %s", resp.ProductType)
	t.Logf("  Author: %s", resp.Author)
	t.Logf("  Published: %s", resp.PublishedTime)
	t.Logf("  Expires: %s", resp.ExpiresTime)
	t.Logf("  Center: %s (%s)", resp.AvalancheCenter.Name, resp.AvalancheCenter.Id)
	t.Logf("  Danger Ratings: %d", len(resp.Danger))
	t.Logf("  Avalanche Problems: %d", len(resp.ForecastAvalancheProblems))

	if resp.Id == 0 {
		t.Error("Forecast ID is 0")
	}

	if resp.AvalancheCenter.Id == "" {
		t.Error("Avalanche center ID is empty")
	}

	t.Log("✓ GetForecast API call successful, response structure valid")
}

func TestClient_FindZoneByCoordinates_Integration(t *testing.T) {
	// Aspen, CO coordinates
	lat := 39.11539
	lon := -107.65840

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	client := NewClient(logger)

	t.Log("Fetching NAC map layer...")
	mapLayer, err := client.GetMapLayer()
	if err != nil {
		t.Fatalf("Failed to get map layer: %v", err)
	}

	t.Logf("Finding zone for coordinates: lat=%f, lon=%f", lat, lon)
	zone := FindZone(lat, lon, mapLayer)
	if zone == nil {
		t.Fatal("No zone found for Aspen coordinates")
	}

	t.Logf("Found Zone:")
	t.Logf("  ID: %d", zone.Id)
	t.Logf("  Name: %s", zone.Properties.Name)
	t.Logf("  Center ID: %s", zone.Properties.CenterId)
	t.Logf("  Danger Level: %d", zone.Properties.DangerLevel)

	if zone.Properties.CenterId != "CAIC" {
		t.Errorf("Expected center_id CAIC, got %s", zone.Properties.CenterId)
	}

	// Now fetch the forecast for this zone
	t.Logf("Fetching forecast for zone %d (%s)...", zone.Id, zone.Properties.Name)
	forecast, err := client.GetForecast(zone.Properties.CenterId, zone.Id)
	if err != nil {
		t.Fatalf("Failed to get forecast for zone: %v", err)
	}

	t.Logf("Forecast:")
	t.Logf("  ID: %d", forecast.Id)
	t.Logf("  Author: %s", forecast.Author)
	t.Logf("  Published: %s", forecast.PublishedTime)
	t.Logf("  Danger Ratings: %d", len(forecast.Danger))

	if forecast.Id == 0 {
		t.Error("Forecast ID is 0")
	}

	t.Log("✓ Full coordinate-to-forecast flow successful")
}
