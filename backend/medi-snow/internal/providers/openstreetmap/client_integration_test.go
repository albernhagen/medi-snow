//go:build integration

package openstreetmap

import (
	"encoding/json"
	"testing"
)

func TestClient_GetElevation_Integration(t *testing.T) {
	// Test coordinates: Aspen, CO area
	lat := 39.11539
	lon := -107.65840

	client := NewClient()

	t.Logf("Making API call to OpenStreetMap Nominatim API...")
	t.Logf("Coordinates: lat=%f, lon=%f", lat, lon)

	resp, err := client.GetElevation(lat, lon)
	if err != nil {
		t.Fatalf("Failed to get location data: %v", err)
	}

	// Pretty print the raw response
	rawJSON, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	t.Logf("Raw API Response:\n%s", string(rawJSON))

	// Verify response structure
	if resp == nil {
		t.Fatal("Response is nil")
	}

	t.Logf("Location Details:")
	t.Logf("  Place ID: %d", resp.PlaceId)
	t.Logf("  Display Name: %s", resp.DisplayName)
	t.Logf("  Type: %s", resp.Type)
	t.Logf("  Class: %s", resp.Class)

	if resp.Address.County != "" {
		t.Logf("  County: %s", resp.Address.County)
	}
	if resp.Address.State != "" {
		t.Logf("  State: %s", resp.Address.State)
	}
	if resp.Address.Country != "" {
		t.Logf("  Country: %s", resp.Address.Country)
	}

	// Verify coordinates match
	if resp.Lat == "" || resp.Lon == "" {
		t.Error("Lat/Lon fields are empty")
	}

	t.Logf("  Returned Coordinates: lat=%s, lon=%s", resp.Lat, resp.Lon)

	// Basic sanity checks
	if resp.PlaceId == 0 {
		t.Error("PlaceId is 0")
	}

	if resp.DisplayName == "" {
		t.Error("DisplayName is empty")
	}

	if len(resp.Boundingbox) != 4 {
		t.Errorf("Expected boundingbox to have 4 values, got %d", len(resp.Boundingbox))
	} else {
		t.Logf("  Bounding Box: %v", resp.Boundingbox)
	}

	t.Log("✓ API call successful, response structure valid")
}
