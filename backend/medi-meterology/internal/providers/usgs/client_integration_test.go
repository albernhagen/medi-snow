//go:build integration

package usgs

import (
	"encoding/json"
	"testing"
)

func TestElevationClient_GetElevation_Integration(t *testing.T) {
	// Test coordinates: Aspen, CO area
	lat := 39.11539
	lon := -107.65840

	client := NewClient()

	t.Logf("Making API call to OpenMeteo Elevation API...")
	t.Logf("Coordinates: lat=%f, lon=%f", lat, lon)

	resp, err := client.GetElevationPoint(lat, lon)
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

	t.Logf("Elevation: %v ft", resp.Value)

	// Sanity check
	if resp.Value < 9000 || resp.Value > 11000 {
		t.Errorf("Elevation seems unreasonable: %v ft", resp.Value)
	}

	t.Log("✓ API call successful, response structure valid")
}
