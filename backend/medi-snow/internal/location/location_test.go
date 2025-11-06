package location

import (
	"errors"
	"medi-snow/internal/providers/openstreetmap"
	"medi-snow/internal/providers/usgs"
	"medi-snow/internal/types"
	"strings"
	"testing"
)

// Mock providers for testing

type mockElevationProvider struct {
	response *usgs.ElevationPointAPIResponse
	err      error
}

func (m *mockElevationProvider) GetElevation(latitude, longitude float64) (*usgs.ElevationPointAPIResponse, error) {
	return m.response, m.err
}

type mockLocationProvider struct {
	response *openstreetmap.LookupAPIResponse
	err      error
}

func (m *mockLocationProvider) Lookup(latitude, longitude float64) (*openstreetmap.LookupAPIResponse, error) {
	return m.response, m.err
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
				Elevation: []float64{2743.5},
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
				if fp.Elevation.Meters != 2743.5 {
					t.Errorf("Elevation.Meters = %v, want %v", fp.Elevation.Meters, 2743.5)
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
				Elevation: []float64{2743.5},
			},
			locationResponse: nil,
			locationErr:      errors.New("location API error"),
			wantErr:          true,
			errContains:      "failed to get location",
		},
		{
			name: "elevation adapter error - empty array",
			lat:  39.11539,
			lon:  -107.65840,
			elevationResponse: &usgs.ElevationPointAPIResponse{
				Elevation: []float64{},
			},
			locationResponse: &openstreetmap.LookupAPIResponse{
				DisplayName: "Test Location",
			},
			wantErr:     true,
			errContains: "elevation response contains no data",
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

			// Create service with mocks
			service := NewLocationServiceWithProviders(elevProvider, locProvider)

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
