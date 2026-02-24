package location

import (
	"encoding/json"
	"errors"
	"log/slog"
	"medi-meteorology/internal/providers/openstreetmap"
	"medi-meteorology/internal/providers/usgs"
	"medi-meteorology/internal/types"
	"os"
	"strings"
	"testing"
)

// Mock providers for testing

type mockElevationProvider struct {
	response *usgs.ElevationPointAPIResponse
	err      error
}

func (m *mockElevationProvider) GetElevationPoint(latitude, longitude float64) (*usgs.ElevationPointAPIResponse, error) {
	return m.response, m.err
}

type mockLocationProvider struct {
	response *openstreetmap.LookupAPIResponse
	err      error
}

func (m *mockLocationProvider) Lookup(latitude, longitude float64) (*openstreetmap.LookupAPIResponse, error) {
	return m.response, m.err
}

func TestLocationService_GetForecastPoint_AspenSnapshot(t *testing.T) {
	// Load USGS elevation snapshot
	elevData, err := os.ReadFile("testdata/usgs_elevation_response.json")
	if err != nil {
		t.Fatalf("Failed to read elevation testdata: %v", err)
	}
	var elevResp usgs.ElevationPointAPIResponse
	if err := json.Unmarshal(elevData, &elevResp); err != nil {
		t.Fatalf("Failed to unmarshal elevation: %v", err)
	}

	// Load OpenStreetMap lookup snapshot
	locData, err := os.ReadFile("testdata/openstreetmap_lookup_response.json")
	if err != nil {
		t.Fatalf("Failed to read location testdata: %v", err)
	}
	var locResp openstreetmap.LookupAPIResponse
	if err := json.Unmarshal(locData, &locResp); err != nil {
		t.Fatalf("Failed to unmarshal location: %v", err)
	}

	elevProvider := &mockElevationProvider{response: &elevResp}
	locProvider := &mockLocationProvider{response: &locResp}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	service := &locationService{
		elevationProvider: elevProvider,
		locationProvider:  locProvider,
		logger:            logger,
	}

	fp, err := service.GetForecastPoint(39.11539, -107.65840)
	if err != nil {
		t.Fatalf("GetForecastPoint returned error: %v", err)
	}

	// Validate coordinates
	if fp.Coordinates.Latitude != 39.11539 {
		t.Errorf("Latitude = %v, want 39.11539", fp.Coordinates.Latitude)
	}
	if fp.Coordinates.Longitude != -107.65840 {
		t.Errorf("Longitude = %v, want -107.65840", fp.Coordinates.Longitude)
	}

	// Validate elevation is reasonable for Aspen area (~10,000+ ft)
	if fp.Elevation.Feet < 9000 || fp.Elevation.Feet > 12000 {
		t.Errorf("Elevation.Feet = %v, expected between 9000-12000", fp.Elevation.Feet)
	}
	t.Logf("Elevation: %.1f ft (%.1f m)", fp.Elevation.Feet, fp.Elevation.Meters)

	// Validate location info
	if fp.Location.State != "Colorado" {
		t.Errorf("Location.State = %q, want Colorado", fp.Location.State)
	}
	if fp.Location.Country != "United States" {
		t.Errorf("Location.Country = %q, want United States", fp.Location.Country)
	}
	if fp.Location.CountryCode != "us" {
		t.Errorf("Location.CountryCode = %q, want us", fp.Location.CountryCode)
	}
	if fp.Location.Name == "" {
		t.Error("Location.Name is empty")
	}
	if fp.Location.County == "" {
		t.Error("Location.County is empty")
	}

	t.Logf("ForecastPoint: name=%s county=%s state=%s elevation=%.0fft",
		fp.Location.Name, fp.Location.County, fp.Location.State, fp.Elevation.Feet)
}

func TestLocationService_GetForecastPoint(t *testing.T) {
	tests := []struct {
		name              string
		lat               float64
		lon               float64
		elevationResponse *usgs.ElevationPointAPIResponse
		elevationErr      error
		locationResponse  *openstreetmap.LookupAPIResponse
		locationErr       error
		wantErr           bool
		errContains       string
		validate          func(*testing.T, *types.ForecastPoint)
	}{
		{
			name: "successful forecast point retrieval",
			lat:  39.11539,
			lon:  -107.65840,
			elevationResponse: &usgs.ElevationPointAPIResponse{
				Value: 2743.5,
			},
			locationResponse: &openstreetmap.LookupAPIResponse{
				Name:        "Aspen",
				DisplayName: "Aspen, Pitkin County, Colorado, United States",
				Address: struct {
					County       string `json:"county"`
					State        string `json:"state"`
					ISO31662Lvl4 string `json:"ISO3166-2-lvl4"`
					Country      string `json:"country"`
					CountryCode  string `json:"country_code"`
				}{
					County:      "Pitkin County",
					State:       "Colorado",
					Country:     "United States",
					CountryCode: "us",
				},
			},
			wantErr: false,
			validate: func(t *testing.T, fp *types.ForecastPoint) {
				if fp == nil {
					t.Fatal("ForecastPoint is nil")
				}
				if fp.Coordinates.Latitude != 39.11539 {
					t.Errorf("Latitude = %v, want %v", fp.Coordinates.Latitude, 39.11539)
				}
				if fp.Coordinates.Longitude != -107.65840 {
					t.Errorf("Longitude = %v, want %v", fp.Coordinates.Longitude, -107.65840)
				}
				if fp.Elevation.Feet != 2743.5 {
					t.Errorf("Elevation.Feet = %v, want %v", fp.Elevation.Feet, 2743.5)
				}
				if fp.Location.Name != "Aspen" {
					t.Errorf("Location.Name = %v, want %v", fp.Location.Name, "Aspen")
				}
				if fp.Location.State != "Colorado" {
					t.Errorf("Location.State = %v, want %v", fp.Location.State, "Colorado")
				}
			},
		},
		{
			name:              "elevation provider error",
			lat:               39.11539,
			lon:               -107.65840,
			elevationResponse: nil,
			elevationErr:      errors.New("elevation API error"),
			locationResponse: &openstreetmap.LookupAPIResponse{
				DisplayName: "Test Location",
			},
			wantErr:     true,
			errContains: "failed to get elevation",
		},
		{
			name: "location provider error",
			lat:  39.11539,
			lon:  -107.65840,
			elevationResponse: &usgs.ElevationPointAPIResponse{
				Value: 2743.5,
			},
			locationResponse: nil,
			locationErr:      errors.New("location API error"),
			wantErr:          true,
			errContains:      "failed to get location",
		},
		{
			name:              "elevation adapter error - nil response",
			lat:               39.11539,
			lon:               -107.65840,
			elevationResponse: nil,
			locationResponse: &openstreetmap.LookupAPIResponse{
				DisplayName: "Test Location",
			},
			wantErr:     true,
			errContains: "elevation response is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock providers
			elevProvider := &mockElevationProvider{
				response: tt.elevationResponse,
				err:      tt.elevationErr,
			}
			locProvider := &mockLocationProvider{
				response: tt.locationResponse,
				err:      tt.locationErr,
			}

			// Create test logger that discards output
			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
				Level: slog.LevelError, // Only log errors in tests
			}))

			// Create service with mocks
			service := &locationService{
				elevationProvider: elevProvider,
				locationProvider:  locProvider,
				logger:            logger,
			}

			// Call GetForecastPoint
			got, err := service.GetForecastPoint(tt.lat, tt.lon)

			// Check error expectations
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetForecastPoint() expected error but got none")
					return
				}
				if tt.errContains != "" {
					if !strings.Contains(err.Error(), tt.errContains) {
						t.Errorf("GetForecastPoint() error = %v, want error containing %v", err, tt.errContains)
					}
				}
				return
			}

			// Check no unexpected error
			if err != nil {
				t.Errorf("GetForecastPoint() unexpected error = %v", err)
				return
			}

			// Run validation if provided
			if tt.validate != nil {
				tt.validate(t, got)
			}
		})
	}
}
