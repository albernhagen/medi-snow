package timezone

import (
	"testing"
)

func TestService_GetTimezone(t *testing.T) {
	svc, err := NewService()
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	tests := []struct {
		name      string
		latitude  float64
		longitude float64
		want      string
	}{
		{
			name:      "Aspen, Colorado",
			latitude:  39.11539,
			longitude: -107.65840,
			want:      "America/Denver",
		},
		{
			name:      "New York City",
			latitude:  40.7128,
			longitude: -74.0060,
			want:      "America/New_York",
		},
		{
			name:      "London, UK",
			latitude:  51.5074,
			longitude: -0.1278,
			want:      "Europe/London",
		},
		{
			name:      "Tokyo, Japan",
			latitude:  35.6762,
			longitude: 139.6503,
			want:      "Asia/Tokyo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.GetTimezone(tt.latitude, tt.longitude)
			if err != nil {
				t.Errorf("GetTimezone() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("GetTimezone() = %v, want %v", got, tt.want)
			}
		})
	}
}
