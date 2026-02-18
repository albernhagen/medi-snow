//go:build integration

package avalanche

import (
	"log/slog"
	"os"
	"testing"
)

func TestAvalancheService_GetForecast_Integration(t *testing.T) {
	// Aspen, CO coordinates
	lat := 39.11539
	lon := -107.65840

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	svc := NewAvalancheService(logger)

	t.Logf("Fetching avalanche forecast for coordinates: lat=%f, lon=%f", lat, lon)

	forecast, err := svc.GetForecast(lat, lon)
	if err != nil {
		t.Fatalf("Failed to get avalanche forecast: %v", err)
	}

	if forecast == nil {
		t.Fatal("Forecast is nil")
	}

	t.Logf("Avalanche Forecast:")
	t.Logf("  Zone: %s (ID: %d)", forecast.Zone.Name, forecast.Zone.Id)
	t.Logf("  Center: %s (%s)", forecast.Center.Name, forecast.Center.Id)
	t.Logf("  Author: %s", forecast.Author)
	t.Logf("  Published: %s", forecast.PublishedTime)
	t.Logf("  Expires: %s", forecast.ExpiresTime)
	t.Logf("  Forecast URL: %s", forecast.ForecastURL)

	// Verify core fields are populated
	if forecast.Zone.Name == "" {
		t.Error("Zone name is empty")
	}
	if forecast.Center.Id == "" {
		t.Error("Center ID is empty")
	}
	if forecast.PublishedTime.IsZero() {
		t.Error("PublishedTime is zero")
	}

	// Log danger ratings
	t.Logf("  Danger Ratings: %d", len(forecast.DangerRatings))
	for _, dr := range forecast.DangerRatings {
		t.Logf("    %s - Lower: %s, Middle: %s, Upper: %s",
			dr.ValidDay, dr.Lower, dr.Middle, dr.Upper)
	}

	// Log avalanche problems
	t.Logf("  Avalanche Problems: %d", len(forecast.Problems))
	for _, p := range forecast.Problems {
		t.Logf("    #%d %s - Likelihood: %s, Size: %.1f-%.1f",
			p.Rank, p.Name, p.Likelihood, p.Size.Min, p.Size.Max)
		if p.MediaURL != "" {
			t.Logf("       Media: %s", p.MediaURL)
		}
	}

	t.Log("Successfully fetched and mapped avalanche forecast")
}
