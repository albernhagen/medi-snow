//go:build integration

package snapshot

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"medi-meteorology/internal/providers/nac"
	"medi-meteorology/internal/providers/nws"
	"medi-meteorology/internal/providers/openmeteo"
	"medi-meteorology/internal/providers/openstreetmap"
	"medi-meteorology/internal/providers/usgs"
)

const (
	aspenLat = 39.11539
	aspenLon = -107.65840
)

// repoRoot returns the path to the Go module root (where go.mod lives).
func repoRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..")
}

// saveJSON marshals v as indented JSON and writes it to path, creating
// directories as needed.
func saveJSON(t *testing.T, path string, v any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	t.Logf("saved %s (%d bytes)", path, len(data))
}

// saveRawJSON fetches a URL and saves the raw JSON response to path.
// This preserves the exact API response without any marshal/unmarshal
// round-trip issues (e.g., for types with custom UnmarshalJSON).
func saveRawJSON(t *testing.T, path string, url string) []byte {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET %s returned %d: %s", url, resp.StatusCode, string(body))
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	// Re-indent for readability
	var raw json.RawMessage
	if err := json.Unmarshal(data, &raw); err == nil {
		if indented, err := json.MarshalIndent(raw, "", "  "); err == nil {
			data = indented
		}
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	t.Logf("saved %s (%d bytes)", path, len(data))
	return data
}

func TestCaptureSnapshots(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	root := repoRoot()

	// --- OpenMeteo forecast ---
	t.Run("openmeteo_forecast", func(t *testing.T) {
		client := openmeteo.NewClient(logger)
		resp, err := client.GetForecast(aspenLat, aspenLon, 2743.5*0.3048, 16, "America/Denver")
		if err != nil {
			t.Fatalf("openmeteo GetForecast: %v", err)
		}
		saveJSON(t, filepath.Join(root, "internal/weather/testdata/openmeteo_forecast_response.json"), resp)
	})

	// --- NAC map layer (save raw to preserve geometry coordinates) ---
	var nacZone *nac.MapLayerFeature
	t.Run("nac_map_layer", func(t *testing.T) {
		path := filepath.Join(root, "internal/avalanche/testdata/nac_map_layer_response.json")
		rawData := saveRawJSON(t, path, "https://api.avalanche.org/v2/public/products/map-layer")

		// Parse it to find the zone for subsequent steps
		var mapLayer nac.MapLayerResponse
		if err := json.Unmarshal(rawData, &mapLayer); err != nil {
			t.Fatalf("unmarshal map layer: %v", err)
		}
		nacZone = nac.FindZone(aspenLat, aspenLon, &mapLayer)
		if nacZone == nil {
			t.Fatal("no NAC zone found for Aspen coordinates")
		}
		t.Logf("found zone: id=%d name=%s center=%s", nacZone.Id, nacZone.Properties.Name, nacZone.Properties.CenterId)
	})

	// --- NAC forecast (save raw to preserve json.RawMessage fields) ---
	t.Run("nac_forecast", func(t *testing.T) {
		if nacZone == nil {
			t.Skip("no zone found in previous step")
		}
		url := fmt.Sprintf("https://api.avalanche.org/v2/public/product?type=forecast&center_id=%s&zone_id=%d",
			nacZone.Properties.CenterId, nacZone.Id)
		saveRawJSON(t, filepath.Join(root, "internal/avalanche/testdata/nac_forecast_response.json"), url)
	})

	// --- USGS elevation ---
	t.Run("usgs_elevation", func(t *testing.T) {
		client := usgs.NewClient(logger)
		resp, err := client.GetElevationPoint(aspenLat, aspenLon)
		if err != nil {
			t.Fatalf("usgs GetElevationPoint: %v", err)
		}
		saveJSON(t, filepath.Join(root, "internal/location/testdata/usgs_elevation_response.json"), resp)
	})

	// --- OpenStreetMap reverse geocode ---
	t.Run("openstreetmap_lookup", func(t *testing.T) {
		client := openstreetmap.NewClient(logger)
		resp, err := client.Lookup(aspenLat, aspenLon)
		if err != nil {
			t.Fatalf("openstreetmap Lookup: %v", err)
		}
		saveJSON(t, filepath.Join(root, "internal/location/testdata/openstreetmap_lookup_response.json"), resp)
	})

	// --- NWS point ---
	var cwa string
	t.Run("nws_point", func(t *testing.T) {
		client := nws.NewClient(logger)
		resp, err := client.GetPoint(aspenLat, aspenLon)
		if err != nil {
			t.Fatalf("nws GetPoint: %v", err)
		}
		saveJSON(t, filepath.Join(root, "internal/weather/testdata/nws_point_response.json"), resp)
		cwa = resp.Properties.Cwa
		t.Logf("CWA: %s", cwa)
	})

	// --- NWS area forecast discussion ---
	t.Run("nws_afd", func(t *testing.T) {
		if cwa == "" {
			t.Skip("no CWA found in previous step")
		}
		client := nws.NewClient(logger)
		resp, err := client.GetAreaForecastDiscussion(cwa)
		if err != nil {
			t.Fatalf("nws GetAreaForecastDiscussion: %v", err)
		}
		saveJSON(t, filepath.Join(root, "internal/weather/testdata/nws_afd_response.json"), resp)
	})
}
