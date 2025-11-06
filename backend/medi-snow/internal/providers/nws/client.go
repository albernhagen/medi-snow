package nws

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
)

// API Docs: https://www.weather.gov/documentation/services-web-api
// Sample requests:
// - https://api.weather.gov/points/39.1154,-107.65840
// - https://api.weather.gov/products/types/AFD/locations/GJT/latest
const (
	baseURL = "https://api.weather.gov"
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
		logger:     logger.With("component", "nws-client"),
	}
}

func (c *Client) GetPoint(latitude, longitude float64) (*PointAPIResponse, error) {
	// Build URL with query parameters
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	u.Path = fmt.Sprintf("/points/%f,%f", latitude, longitude)

	c.logger.Debug("fetching NWS point data",
		"latitude", latitude,
		"longitude", longitude,
		"url", u.String(),
	)

	// Make the HTTP request
	resp, err := c.httpClient.Get(u.String())
	if err != nil {
		c.logger.Error("failed to fetch NWS point data",
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
		c.logger.Error("NWS API returned error",
			"status_code", resp.StatusCode,
			"latitude", latitude,
			"longitude", longitude,
			"response_body", string(body),
		)
		return nil, fmt.Errorf("fetch returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the JSON response
	var apiResp PointAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		c.logger.Error("failed to decode NWS response",
			"latitude", latitude,
			"longitude", longitude,
			"error", err,
		)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Debug("successfully fetched NWS point data",
		"latitude", latitude,
		"longitude", longitude,
	)

	return &apiResp, nil
}

func (c *Client) GetAFD(locationId string) (*AFDAPIResponse, error) {
	// Build URL with query parameters
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	u.Path = fmt.Sprintf("/products/types/AFD/locations/%s/latest", locationId)

	c.logger.Debug("fetching NWS AFD data",
		"location_id", locationId,
		"url", u.String(),
	)

	// Make the HTTP request
	resp, err := c.httpClient.Get(u.String())
	if err != nil {
		c.logger.Error("failed to fetch NWS AFD data",
			"location_id", locationId,
			"error", err,
		)
		return nil, fmt.Errorf("failed to fetch: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.logger.Error("NWS AFD API returned error",
			"status_code", resp.StatusCode,
			"location_id", locationId,
			"response_body", string(body),
		)
		return nil, fmt.Errorf("fetch returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the JSON response
	var apiResp AFDAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		c.logger.Error("failed to decode NWS AFD response",
			"location_id", locationId,
			"error", err,
		)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Debug("successfully fetched NWS AFD data",
		"location_id", locationId,
	)

	return &apiResp, nil
}
