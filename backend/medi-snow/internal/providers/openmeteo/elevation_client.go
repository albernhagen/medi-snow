package openmeteo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// API Docs: https://open-meteo.com/en/docs/elevation-api
// Sample request: https://api.open-meteo.com/v1/elevation?latitude=39.1178&longitude=-106.4452
const (
	baseElevationURL = "https://api.open-meteo.com/v1/elevation"
)

type ElevationClient struct {
	httpClient *http.Client
	baseURL    string
}

func NewElevationClient() *ElevationClient {
	return &ElevationClient{
		httpClient: &http.Client{},
		baseURL:    baseElevationURL,
	}
}

func (c *ElevationClient) GetElevation(latitude, longitude float64) (*ElevationAPIResponse, error) {
	// Build URL with query parameters
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	q := u.Query()
	q.Set("latitude", fmt.Sprintf("%f", latitude))
	q.Set("longitude", fmt.Sprintf("%f", longitude))
	u.RawQuery = q.Encode()

	// Make the HTTP request
	resp, err := c.httpClient.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fetch returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the JSON response
	var apiResp ElevationAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &apiResp, nil
}
