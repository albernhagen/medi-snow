package avalanche

import (
	"encoding/json"
	"medi-meteorology/internal/providers/nac"
	"strconv"
)

// mapForecastResponse translates a NAC ForecastResponse into a provider-agnostic
// AvalancheForecast domain model.
func mapForecastResponse(zone *nac.MapLayerFeature, resp *nac.ForecastResponse) *AvalancheForecast {
	forecast := &AvalancheForecast{
		Zone: ForecastZone{
			Id:   zone.Id,
			Name: zone.Properties.Name,
			URL:  zone.Properties.Link,
		},
		Center: AvalancheCenter{
			Id:    resp.AvalancheCenter.Id,
			Name:  resp.AvalancheCenter.Name,
			URL:   resp.AvalancheCenter.Url,
			City:  resp.AvalancheCenter.City,
			State: resp.AvalancheCenter.State,
		},
		PublishedTime:    resp.PublishedTime,
		ExpiresTime:      resp.ExpiresTime,
		Author:           resp.Author,
		BottomLine:       resp.BottomLine,
		HazardDiscussion: resp.HazardDiscussion,
		ForecastURL:      zone.Properties.Link,
	}

	// Map the forecast zone state from the response if available
	if len(resp.ForecastZone) > 0 {
		for _, fz := range resp.ForecastZone {
			if fz.Id == zone.Id {
				forecast.Zone.State = fz.State
				if fz.Url != "" {
					forecast.Zone.URL = fz.Url
				}
				break
			}
		}
	}

	// Map danger ratings
	forecast.DangerRatings = mapDangerRatings(resp)

	// Map avalanche problems
	forecast.Problems = mapAvalancheProblems(resp)

	return forecast
}

// mapDangerRatings converts NAC danger entries to domain DangerRating values.
func mapDangerRatings(resp *nac.ForecastResponse) []DangerRating {
	ratings := make([]DangerRating, 0, len(resp.Danger))
	for _, d := range resp.Danger {
		ratings = append(ratings, DangerRating{
			ValidDay: d.ValidDay,
			Lower:    DangerLevel(d.Lower),
			Middle:   DangerLevel(d.Middle),
			Upper:    DangerLevel(d.Upper),
		})
	}
	return ratings
}

// mapAvalancheProblems converts NAC avalanche problem entries to domain AvalancheProblem values.
func mapAvalancheProblems(resp *nac.ForecastResponse) []AvalancheProblem {
	problems := make([]AvalancheProblem, 0, len(resp.ForecastAvalancheProblems))
	for _, p := range resp.ForecastAvalancheProblems {
		problem := AvalancheProblem{
			Name:       p.Name,
			Rank:       p.Rank,
			Likelihood: ParseLikelihood(p.Likelihood),
			Discussion: p.Discussion,
			Location:   p.Location,
			Size:       parseSize(p.Size),
			MediaURL:   extractMediaURL(p.Media.Url),
		}
		problems = append(problems, problem)
	}
	return problems
}

// parseSize converts a slice of size strings (e.g. ["1", "2.5"]) into an
// AvalancheSize with Min and Max values.
func parseSize(sizes []string) AvalancheSize {
	if len(sizes) == 0 {
		return AvalancheSize{}
	}

	result := AvalancheSize{}
	for i, s := range sizes {
		val, err := strconv.ParseFloat(s, 64)
		if err != nil {
			continue
		}
		if i == 0 {
			result.Min = val
			result.Max = val
		} else {
			if val < result.Min {
				result.Min = val
			}
			if val > result.Max {
				result.Max = val
			}
		}
	}
	return result
}

// extractMediaURL handles the polymorphic media.url field from NAC.
// It may be a JSON object with size keys (Large, Medium, Original, Thumbnail)
// or a plain JSON string. Returns the Original URL if available.
func extractMediaURL(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	// Try as a struct first (most common case)
	var urlStruct struct {
		Large     string `json:"large"`
		Medium    string `json:"medium"`
		Original  string `json:"original"`
		Thumbnail string `json:"thumbnail"`
	}
	if err := json.Unmarshal(raw, &urlStruct); err == nil && urlStruct.Original != "" {
		return urlStruct.Original
	}

	// Try as a plain string (GNFAC returns this)
	var urlString string
	if err := json.Unmarshal(raw, &urlString); err == nil {
		return urlString
	}

	return ""
}
