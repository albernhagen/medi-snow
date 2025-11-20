//go:build integration

package nws

import (
	"encoding/json"
	"log/slog"
	"os"
	"testing"
)

func TestClient_GetPoint_Integration(t *testing.T) {
	// Test coordinates: Aspen, CO area
	lat := 39.11539
	lon := -107.65840

	// Create logger for test
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	client := NewClient(logger)

	t.Logf("Making API call to NWS Points API...")
	t.Logf("Coordinates: lat=%f, lon=%f", lat, lon)

	resp, err := client.GetPoint(lat, lon)
	if err != nil {
		t.Fatalf("Failed to get point data: %v", err)
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

	t.Logf("Point Details:")
	t.Logf("  Type: %s", resp.Type)
	t.Logf("  CWA: %s", resp.Properties.Cwa)
	t.Logf("  Forecast Office: %s", resp.Properties.ForecastOffice)
	t.Logf("  Grid ID: %s", resp.Properties.GridId)
	t.Logf("  Grid X: %d", resp.Properties.GridX)
	t.Logf("  Grid Y: %d", resp.Properties.GridY)
	t.Logf("  Time Zone: %s", resp.Properties.TimeZone)

	// Verify relative location
	if resp.Properties.RelativeLocation.Properties.City != "" {
		t.Logf("  City: %s", resp.Properties.RelativeLocation.Properties.City)
	}
	if resp.Properties.RelativeLocation.Properties.State != "" {
		t.Logf("  State: %s", resp.Properties.RelativeLocation.Properties.State)
	}

	// Verify geometry
	if resp.Geometry.Type == "" {
		t.Error("Geometry Type is empty")
	}
	if len(resp.Geometry.Coordinates) == 0 {
		t.Error("Geometry Coordinates are empty")
	} else {
		t.Logf("  Coordinates: %v", resp.Geometry.Coordinates)
	}

	// Basic sanity checks
	if resp.Properties.GridId == "" {
		t.Error("GridId is empty")
	}

	if resp.Properties.Cwa == "" {
		t.Error("CWA is empty")
	}

	if resp.Properties.Forecast == "" {
		t.Error("Forecast URL is empty")
	} else {
		t.Logf("  Forecast URL: %s", resp.Properties.Forecast)
	}

	if resp.Properties.ForecastHourly == "" {
		t.Error("ForecastHourly URL is empty")
	} else {
		t.Logf("  Forecast Hourly URL: %s", resp.Properties.ForecastHourly)
	}

	t.Log("✓ GetPoint API call successful, response structure valid")
}

func TestClient_GetAFD_Integration(t *testing.T) {
	// Use Grand Junction, CO office (GJT) - covers Aspen area
	locationId := "GJT"

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	client := NewClient(logger)

	t.Logf("Making API call to NWS AFD API...")
	t.Logf("Location ID: %s", locationId)

	resp, err := client.GetAreaForecastDiscussion(locationId)
	if err != nil {
		t.Fatalf("Failed to get AFD data: %v", err)
	}

	// Pretty print the raw response (excluding productText which can be very long)
	type afdSummary struct {
		Id              string `json:"@id"`
		Id1             string `json:"id"`
		WmoCollectiveId string `json:"wmoCollectiveId"`
		IssuingOffice   string `json:"issuingOffice"`
		IssuanceTime    string `json:"issuanceTime"`
		ProductCode     string `json:"productCode"`
		ProductName     string `json:"productName"`
		ProductTextLen  int    `json:"productTextLength"`
	}

	summary := afdSummary{
		Id:              resp.Id,
		Id1:             resp.Id1,
		WmoCollectiveId: resp.WmoCollectiveId,
		IssuingOffice:   resp.IssuingOffice,
		IssuanceTime:    resp.IssuanceTime.String(),
		ProductCode:     resp.ProductCode,
		ProductName:     resp.ProductName,
		ProductTextLen:  len(resp.ProductText),
	}

	rawJSON, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	t.Logf("Raw API Response Summary:\n%s", string(rawJSON))

	// Verify response structure
	if resp == nil {
		t.Fatal("Response is nil")
	}

	t.Logf("AFD Details:")
	t.Logf("  Product Code: %s", resp.ProductCode)
	t.Logf("  Product Name: %s", resp.ProductName)
	t.Logf("  Issuing Office: %s", resp.IssuingOffice)
	t.Logf("  Issuance Time: %s", resp.IssuanceTime)
	t.Logf("  Product Text Length: %d characters", len(resp.ProductText))

	// Basic sanity checks
	if resp.ProductCode == "" {
		t.Error("ProductCode is empty")
	}

	if resp.ProductName == "" {
		t.Error("ProductName is empty")
	}

	if resp.IssuingOffice == "" {
		t.Error("IssuingOffice is empty")
	}

	if resp.ProductText == "" {
		t.Error("ProductText is empty")
	} else {
		// Show first 200 characters of product text
		previewLen := 200
		if len(resp.ProductText) < previewLen {
			previewLen = len(resp.ProductText)
		}
		t.Logf("  Product Text Preview: %s...", resp.ProductText[:previewLen])
	}

	if resp.IssuanceTime.IsZero() {
		t.Error("IssuanceTime is zero")
	}

	t.Log("✓ GetAreaForecastDiscussion API call successful, response structure valid")
}
