package usgs

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
)

// API Docs: https://epqs.nationalmap.gov/v1/docs
// Sample request: https://epqs.nationalmap.gov/v1/json?x=-107.65840&y=39.0639&units=Feet
const (
	baseElevationURL = "https://epqs.nationalmap.gov/v1/json"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	logger     *slog.Logger
}

func NewClient(logger *slog.Logger) *Client {
	return &Client{
		httpClient: &http.Client{},
		baseURL:    baseElevationURL,
		logger:     logger.With("component", "usgs-client"),
	}
}

func (c *Client) GetElevationPoint(latitude, longitude float64) (*ElevationPointAPIResponse, error) {
	// Build URL with query parameters
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	q := u.Query()
	q.Set("y", fmt.Sprintf("%f", latitude))
	q.Set("x", fmt.Sprintf("%f", longitude))
	q.Set("units", "Feet")
	u.RawQuery = q.Encode()

	c.logger.Debug("fetching USGS elevation data",
		"latitude", latitude,
		"longitude", longitude,
		"url", u.String(),
	)

	// Make the HTTP request
	resp, err := c.httpClient.Get(u.String())
	if err != nil {
		c.logger.Error("failed to fetch USGS elevation data",
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
		c.logger.Error("USGS API returned error",
			"status_code", resp.StatusCode,
			"latitude", latitude,
			"longitude", longitude,
			"response_body", string(body),
		)
		return nil, fmt.Errorf("fetch returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the JSON response
	var apiResp ElevationPointAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		c.logger.Error("failed to decode USGS response",
			"latitude", latitude,
			"longitude", longitude,
			"error", err,
		)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Debug("successfully fetched USGS elevation data",
		"latitude", latitude,
		"longitude", longitude,
		"elevation_feet", apiResp.Value,
	)

	return &apiResp, nil
}
