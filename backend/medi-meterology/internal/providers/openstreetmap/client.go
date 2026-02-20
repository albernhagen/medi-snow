package openstreetmap

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
)

// API Docs: https://nominatim.org/release-docs/develop/api/Lookup/
// Sample request: https://nominatim.openstreetmap.org/reverse?lat=39.11&lon=-107.65&format=json
const (
	baseURL = "https://nominatim.openstreetmap.org/reverse"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	logger     *slog.Logger
}

func NewClient(logger *slog.Logger) *Client {
	return &Client{
		httpClient: &http.Client{},
		baseURL:    baseURL,
		logger:     logger.With("component", "openstreetmap-client"),
	}
}

func (c *Client) Lookup(latitude, longitude float64) (*LookupAPIResponse, error) {
	// Build URL with query parameters
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	q := u.Query()
	q.Set("lat", fmt.Sprintf("%f", latitude))
	q.Set("lon", fmt.Sprintf("%f", longitude))
	q.Set("format", "json")
	u.RawQuery = q.Encode()

	c.logger.Debug("fetching OpenStreetMap location data",
		"latitude", latitude,
		"longitude", longitude,
		"url", u.String(),
	)

	// Make the HTTP request
	resp, err := c.httpClient.Get(u.String())
	if err != nil {
		c.logger.Error("failed to fetch OpenStreetMap data",
			"latitude", latitude,
			"longitude", longitude,
			"error", err,
		)
		return nil, fmt.Errorf("failed to fetch: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.logger.Error("OpenStreetMap API returned error",
			"status_code", resp.StatusCode,
			"latitude", latitude,
			"longitude", longitude,
			"response_body", string(body),
		)
		return nil, fmt.Errorf("fetch returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the JSON response
	var apiResp LookupAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		c.logger.Error("failed to decode OpenStreetMap response",
			"latitude", latitude,
			"longitude", longitude,
			"error", err,
		)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Debug("successfully fetched OpenStreetMap location data",
		"latitude", latitude,
		"longitude", longitude,
		"display_name", apiResp.DisplayName,
	)

	return &apiResp, nil
}
