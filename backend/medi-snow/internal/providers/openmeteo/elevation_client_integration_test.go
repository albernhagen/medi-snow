//go:build integration

package openmeteo

import (
	"encoding/json"
	"testing"
)

func TestElevationClient_GetElevation_Integration(t *testing.T) {
	// Test coordinates: Aspen, CO area
	lat := 39.11539
	lon := -107.65840

	client := NewElevationClient()

	t.Logf("Making API call to OpenMeteo Elevation API...")
	t.Logf("Coordinates: lat=%f, lon=%f", lat, lon)

	resp, err := client.GetElevation(lat, lon)
	if err != nil {
		t.Fatalf("Failed to get elevation: %v", err)
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

	if len(resp.Elevation) == 0 {
		t.Fatal("Elevation array is empty")
	}

	t.Logf("Elevation: %v meters", resp.Elevation[0])

	// Sanity check - Aspen area should be between 2000-4000 meters
	if resp.Elevation[0] < 1000 || resp.Elevation[0] > 5000 {
		t.Errorf("Elevation seems unreasonable: %v meters", resp.Elevation[0])
	}

	t.Log("✓ API call successful, response structure valid")
}
