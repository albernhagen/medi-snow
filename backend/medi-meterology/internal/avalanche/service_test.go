package avalanche

import (
	"encoding/json"
	"medi-meteorology/internal/providers/nac"
	"testing"
	"time"
)

func TestParseLikelihood(t *testing.T) {
	tests := []struct {
		input    string
		expected Likelihood
	}{
		{"unlikely", LikelihoodUnlikely},
		{"Unlikely", LikelihoodUnlikely},
		{"possible", LikelihoodPossible},
		{"Possible", LikelihoodPossible},
		{"likely", LikelihoodLikely},
		{"Likely", LikelihoodLikely},
		{"veryLikely", LikelihoodVeryLikely},
		{"very likely", LikelihoodVeryLikely},
		{"Very Likely", LikelihoodVeryLikely},
		{"very_likely", LikelihoodVeryLikely},
		{"almostCertain", LikelihoodAlmostCertain},
		{"almost certain", LikelihoodAlmostCertain},
		{"Almost Certain", LikelihoodAlmostCertain},
		{"", 0},
		{"unknown", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseLikelihood(tt.input)
			if result != tt.expected {
				t.Errorf("ParseLikelihood(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDangerLevel_String(t *testing.T) {
	tests := []struct {
		level    DangerLevel
		expected string
	}{
		{DangerNone, "No Rating"},
		{DangerLow, "Low"},
		{DangerModerate, "Moderate"},
		{DangerConsiderable, "Considerable"},
		{DangerHigh, "High"},
		{DangerExtreme, "Extreme"},
		{DangerLevel(99), "Unknown (99)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.level.String()
			if result != tt.expected {
				t.Errorf("DangerLevel(%d).String() = %q, want %q", tt.level, result, tt.expected)
			}
		})
	}
}

func TestLikelihood_String(t *testing.T) {
	tests := []struct {
		likelihood Likelihood
		expected   string
	}{
		{LikelihoodUnlikely, "Unlikely"},
		{LikelihoodPossible, "Possible"},
		{LikelihoodLikely, "Likely"},
		{LikelihoodVeryLikely, "Very Likely"},
		{LikelihoodAlmostCertain, "Almost Certain"},
		{Likelihood(0), "Unknown (0)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.likelihood.String()
			if result != tt.expected {
				t.Errorf("Likelihood(%d).String() = %q, want %q", tt.likelihood, result, tt.expected)
			}
		})
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected AvalancheSize
	}{
		{"empty", nil, AvalancheSize{}},
		{"single integer", []string{"1"}, AvalancheSize{Min: 1, Max: 1}},
		{"single decimal", []string{"1.5"}, AvalancheSize{Min: 1.5, Max: 1.5}},
		{"range", []string{"1", "2"}, AvalancheSize{Min: 1, Max: 2}},
		{"range with decimals", []string{"1.5", "2.5"}, AvalancheSize{Min: 1.5, Max: 2.5}},
		{"range with mixed formats", []string{"1.0", "2"}, AvalancheSize{Min: 1, Max: 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSize(tt.input)
			if result.Min != tt.expected.Min || result.Max != tt.expected.Max {
				t.Errorf("parseSize(%v) = {Min: %f, Max: %f}, want {Min: %f, Max: %f}",
					tt.input, result.Min, result.Max, tt.expected.Min, tt.expected.Max)
			}
		})
	}
}

func TestExtractMediaURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"struct with original",
			`{"large":"https://example.com/large.jpg","medium":"https://example.com/medium.jpg","original":"https://example.com/original.jpg","thumbnail":"https://example.com/thumb.jpg"}`,
			"https://example.com/original.jpg",
		},
		{
			"plain string URL",
			`"https://example.com/image.jpg"`,
			"https://example.com/image.jpg",
		},
		{
			"empty",
			``,
			"",
		},
		{
			"null",
			`null`,
			"",
		},
		{
			"empty struct",
			`{"large":"","medium":"","original":"","thumbnail":""}`,
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractMediaURL(json.RawMessage(tt.input))
			if result != tt.expected {
				t.Errorf("extractMediaURL(%s) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMapForecastResponse(t *testing.T) {
	published := time.Date(2025, 1, 15, 7, 0, 0, 0, time.UTC)
	expires := time.Date(2025, 1, 16, 7, 0, 0, 0, time.UTC)

	zone := &nac.MapLayerFeature{
		Id: 2690,
		Properties: nac.MapLayerProperties{
			Name:     "Aspen",
			CenterId: "CAIC",
			Link:     "https://avalanche.state.co.us/forecasts/backcountry-avalanche/aspen",
		},
	}

	mediaURL := json.RawMessage(`{"large":"https://example.com/large.jpg","medium":"https://example.com/med.jpg","original":"https://example.com/orig.jpg","thumbnail":"https://example.com/thumb.jpg"}`)

	resp := &nac.ForecastResponse{
		Id:               12345,
		PublishedTime:    published,
		ExpiresTime:      expires,
		Author:           "John Doe",
		BottomLine:       "<p>Moderate danger</p>",
		HazardDiscussion: "<p>Watch for wind slabs</p>",
		AvalancheCenter: struct {
			Id    string `json:"id"`
			Name  string `json:"name"`
			Url   string `json:"url"`
			City  string `json:"city"`
			State string `json:"state"`
		}{
			Id:    "CAIC",
			Name:  "Colorado Avalanche Information Center",
			Url:   "https://avalanche.state.co.us",
			City:  "Boulder",
			State: "CO",
		},
		Danger: []struct {
			Lower    int    `json:"lower"`
			Upper    int    `json:"upper"`
			Middle   int    `json:"middle"`
			ValidDay string `json:"valid_day"`
		}{
			{Lower: 1, Middle: 2, Upper: 3, ValidDay: "current"},
			{Lower: 1, Middle: 1, Upper: 2, ValidDay: "tomorrow"},
		},
		ForecastZone: []struct {
			Id     int         `json:"id"`
			Name   string      `json:"name"`
			Url    string      `json:"url"`
			State  string      `json:"state"`
			ZoneId string      `json:"zone_id"`
			Config interface{} `json:"config"`
		}{
			{Id: 2690, Name: "Aspen", Url: "https://avalanche.state.co.us/forecasts/aspen", State: "CO"},
		},
	}

	// Set forecast avalanche problems
	resp.ForecastAvalancheProblems = []struct {
		Id                 int    `json:"id"`
		ForecastId         int    `json:"forecast_id"`
		AvalancheProblemId int    `json:"avalanche_problem_id"`
		Rank               int    `json:"rank"`
		Likelihood         string `json:"likelihood"`
		Discussion         string `json:"discussion"`
		Media              struct {
			Url     json.RawMessage `json:"url"`
			Type    string          `json:"type"`
			Title   interface{}     `json:"title"`
			Caption string          `json:"caption"`
		} `json:"media"`
		Location           []string `json:"location"`
		Size               []string `json:"size"`
		Name               string   `json:"name"`
		ProblemDescription string   `json:"problem_description"`
		Icon               string   `json:"icon"`
	}{
		{
			Rank:       1,
			Likelihood: "veryLikely",
			Discussion: "<p>Wind slab discussion</p>",
			Name:       "Wind Slab",
			Location:   []string{"north upper", "northeast upper"},
			Size:       []string{"1.5", "2.5"},
			Media: struct {
				Url     json.RawMessage `json:"url"`
				Type    string          `json:"type"`
				Title   interface{}     `json:"title"`
				Caption string          `json:"caption"`
			}{Url: mediaURL},
		},
		{
			Rank:       2,
			Likelihood: "possible",
			Discussion: "<p>Persistent slab discussion</p>",
			Name:       "Persistent Slab",
			Location:   []string{"north upper"},
			Size:       []string{"2", "3"},
		},
	}

	forecast := mapForecastResponse(zone, resp)

	// Verify zone
	if forecast.Zone.Id != 2690 {
		t.Errorf("Zone.Id = %d, want 2690", forecast.Zone.Id)
	}
	if forecast.Zone.Name != "Aspen" {
		t.Errorf("Zone.Name = %q, want %q", forecast.Zone.Name, "Aspen")
	}
	if forecast.Zone.State != "CO" {
		t.Errorf("Zone.State = %q, want %q", forecast.Zone.State, "CO")
	}

	// Verify center
	if forecast.Center.Id != "CAIC" {
		t.Errorf("Center.Id = %q, want %q", forecast.Center.Id, "CAIC")
	}
	if forecast.Center.Name != "Colorado Avalanche Information Center" {
		t.Errorf("Center.Name = %q, want %q", forecast.Center.Name, "Colorado Avalanche Information Center")
	}

	// Verify times
	if !forecast.PublishedTime.Equal(published) {
		t.Errorf("PublishedTime = %v, want %v", forecast.PublishedTime, published)
	}
	if !forecast.ExpiresTime.Equal(expires) {
		t.Errorf("ExpiresTime = %v, want %v", forecast.ExpiresTime, expires)
	}

	// Verify author and HTML fields
	if forecast.Author != "John Doe" {
		t.Errorf("Author = %q, want %q", forecast.Author, "John Doe")
	}
	if forecast.BottomLine != "<p>Moderate danger</p>" {
		t.Errorf("BottomLine = %q, want %q", forecast.BottomLine, "<p>Moderate danger</p>")
	}

	// Verify danger ratings
	if len(forecast.DangerRatings) != 2 {
		t.Fatalf("DangerRatings count = %d, want 2", len(forecast.DangerRatings))
	}
	if forecast.DangerRatings[0].ValidDay != "current" {
		t.Errorf("DangerRatings[0].ValidDay = %q, want %q", forecast.DangerRatings[0].ValidDay, "current")
	}
	if forecast.DangerRatings[0].Lower != DangerLow {
		t.Errorf("DangerRatings[0].Lower = %d, want %d", forecast.DangerRatings[0].Lower, DangerLow)
	}
	if forecast.DangerRatings[0].Middle != DangerModerate {
		t.Errorf("DangerRatings[0].Middle = %d, want %d", forecast.DangerRatings[0].Middle, DangerModerate)
	}
	if forecast.DangerRatings[0].Upper != DangerConsiderable {
		t.Errorf("DangerRatings[0].Upper = %d, want %d", forecast.DangerRatings[0].Upper, DangerConsiderable)
	}

	// Verify problems
	if len(forecast.Problems) != 2 {
		t.Fatalf("Problems count = %d, want 2", len(forecast.Problems))
	}

	// First problem
	p1 := forecast.Problems[0]
	if p1.Name != "Wind Slab" {
		t.Errorf("Problems[0].Name = %q, want %q", p1.Name, "Wind Slab")
	}
	if p1.Rank != 1 {
		t.Errorf("Problems[0].Rank = %d, want 1", p1.Rank)
	}
	if p1.Likelihood != LikelihoodVeryLikely {
		t.Errorf("Problems[0].Likelihood = %d, want %d (VeryLikely)", p1.Likelihood, LikelihoodVeryLikely)
	}
	if p1.Size.Min != 1.5 || p1.Size.Max != 2.5 {
		t.Errorf("Problems[0].Size = {%f, %f}, want {1.5, 2.5}", p1.Size.Min, p1.Size.Max)
	}
	if p1.MediaURL != "https://example.com/orig.jpg" {
		t.Errorf("Problems[0].MediaURL = %q, want %q", p1.MediaURL, "https://example.com/orig.jpg")
	}
	if len(p1.Location) != 2 {
		t.Errorf("Problems[0].Location count = %d, want 2", len(p1.Location))
	}

	// Second problem
	p2 := forecast.Problems[1]
	if p2.Likelihood != LikelihoodPossible {
		t.Errorf("Problems[1].Likelihood = %d, want %d (Possible)", p2.Likelihood, LikelihoodPossible)
	}
	if p2.MediaURL != "" {
		t.Errorf("Problems[1].MediaURL = %q, want empty", p2.MediaURL)
	}
}

func TestMapForecastResponse_EmptyForecast(t *testing.T) {
	zone := &nac.MapLayerFeature{
		Id: 100,
		Properties: nac.MapLayerProperties{
			Name:     "Sparse Zone",
			CenterId: "UAC",
		},
	}

	resp := &nac.ForecastResponse{
		PublishedTime: time.Now(),
		ExpiresTime:   time.Now().Add(24 * time.Hour),
		AvalancheCenter: struct {
			Id    string `json:"id"`
			Name  string `json:"name"`
			Url   string `json:"url"`
			City  string `json:"city"`
			State string `json:"state"`
		}{Id: "UAC", Name: "Utah Avalanche Center"},
	}

	forecast := mapForecastResponse(zone, resp)

	if forecast == nil {
		t.Fatal("Expected non-nil forecast for empty response")
	}
	if len(forecast.DangerRatings) != 0 {
		t.Errorf("DangerRatings = %d, want 0", len(forecast.DangerRatings))
	}
	if len(forecast.Problems) != 0 {
		t.Errorf("Problems = %d, want 0", len(forecast.Problems))
	}
	if forecast.Center.Id != "UAC" {
		t.Errorf("Center.Id = %q, want %q", forecast.Center.Id, "UAC")
	}
}
