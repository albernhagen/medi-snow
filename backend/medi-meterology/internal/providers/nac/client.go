package nac

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
)

const baseURL = "https://api.avalanche.org"

type Client struct {
	httpClient *http.Client
	baseURL    string
	logger     *slog.Logger
}

func NewClient(logger *slog.Logger) *Client {
	return &Client{
		httpClient: &http.Client{},
		baseURL:    baseURL,
		logger:     logger.With("component", "nac-client"),
	}
}

// GetMapLayer fetches the GeoJSON map layer with all forecast zone polygons.
func (c *Client) GetMapLayer() (*MapLayerResponse, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	u.Path = "/v2/public/products/map-layer"

	c.logger.Debug("fetching NAC map layer", "url", u.String())

	resp, err := c.httpClient.Get(u.String())
	if err != nil {
		c.logger.Error("failed to fetch NAC map layer", "error", err)
		return nil, fmt.Errorf("failed to fetch: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.logger.Error("NAC map layer API returned error",
			"status_code", resp.StatusCode,
			"response_body", string(body),
		)
		return nil, fmt.Errorf("fetch returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp MapLayerResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		c.logger.Error("failed to decode NAC map layer response", "error", err)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Debug("successfully fetched NAC map layer", "feature_count", len(apiResp.Features))

	return &apiResp, nil
}

// GetForecast fetches an avalanche forecast for a specific center and zone.
func (c *Client) GetForecast(centerId string, zoneId int) (*ForecastResponse, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	u.Path = "/v2/public/product"
	q := u.Query()
	q.Set("type", "forecast")
	q.Set("center_id", centerId)
	q.Set("zone_id", fmt.Sprintf("%d", zoneId))
	u.RawQuery = q.Encode()

	c.logger.Debug("fetching NAC forecast",
		"center_id", centerId,
		"zone_id", zoneId,
		"url", u.String(),
	)

	resp, err := c.httpClient.Get(u.String())
	if err != nil {
		c.logger.Error("failed to fetch NAC forecast",
			"center_id", centerId,
			"zone_id", zoneId,
			"error", err,
		)
		return nil, fmt.Errorf("failed to fetch: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.logger.Error("NAC forecast API returned error",
			"status_code", resp.StatusCode,
			"center_id", centerId,
			"zone_id", zoneId,
			"response_body", string(body),
		)
		return nil, fmt.Errorf("fetch returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp ForecastResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		c.logger.Error("failed to decode NAC forecast response",
			"center_id", centerId,
			"zone_id", zoneId,
			"error", err,
		)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Debug("successfully fetched NAC forecast",
		"center_id", centerId,
		"zone_id", zoneId,
	)

	return &apiResp, nil
}
