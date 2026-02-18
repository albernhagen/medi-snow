package nac

import (
	"encoding/json"
	"fmt"
	"time"
)

// MapLayerResponse is a GeoJSON FeatureCollection from the NAC map-layer endpoint.
type MapLayerResponse struct {
	Type     string            `json:"type"`
	Features []MapLayerFeature `json:"features"`
}

type MapLayerFeature struct {
	Id         int                `json:"id"`
	Type       string             `json:"type"`
	Properties MapLayerProperties `json:"properties"`
	Geometry   MapLayerGeometry   `json:"geometry"`
}

type MapLayerProperties struct {
	Name         string `json:"name"`
	CenterId     string `json:"center_id"`
	DangerLevel  int    `json:"danger_level"`
	Danger       string `json:"danger"`
	TravelAdvice string `json:"travel_advice"`
	Link         string `json:"link"`
	StartDate    string `json:"start_date"`
	EndDate      string `json:"end_date"`
	OffSeason    bool   `json:"off_season"`
	Warning      struct {
		Product string `json:"product"`
	} `json:"warning"`
}

// MapLayerGeometry handles both Polygon and MultiPolygon GeoJSON types.
type MapLayerGeometry struct {
	Type string `json:"type"`
	// rawCoordinates holds the raw JSON for custom unmarshaling.
	rawCoordinates json.RawMessage
	// polygon stores coordinates as [][][2]float64 regardless of type.
	polygon [][][2]float64
}

func (g *MapLayerGeometry) UnmarshalJSON(data []byte) error {
	var raw struct {
		Type        string          `json:"type"`
		Coordinates json.RawMessage `json:"coordinates"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	g.Type = raw.Type
	g.rawCoordinates = raw.Coordinates

	switch raw.Type {
	case "Polygon":
		var coords [][][2]float64
		if err := json.Unmarshal(raw.Coordinates, &coords); err != nil {
			return fmt.Errorf("failed to unmarshal Polygon coordinates: %w", err)
		}
		g.polygon = coords
	case "MultiPolygon":
		var multi [][][][2]float64
		if err := json.Unmarshal(raw.Coordinates, &multi); err != nil {
			return fmt.Errorf("failed to unmarshal MultiPolygon coordinates: %w", err)
		}
		for _, poly := range multi {
			g.polygon = append(g.polygon, poly...)
		}
	default:
		return fmt.Errorf("unsupported geometry type: %s", raw.Type)
	}
	return nil
}

// Coordinates returns all polygon rings as [][][2]float64.
// For Polygon types this is the rings directly.
// For MultiPolygon types all polygons' rings are flattened into a single slice.
func (g *MapLayerGeometry) Coordinates() [][][2]float64 {
	return g.polygon
}

// ForecastResponse is the response from the NAC forecast endpoint.
type ForecastResponse struct {
	Id                int         `json:"id"`
	PublishedTime     time.Time   `json:"published_time"`
	ExpiresTime       time.Time   `json:"expires_time"`
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`
	Author            string      `json:"author"`
	ProductType       string      `json:"product_type"`
	BottomLine        string      `json:"bottom_line"`
	HazardDiscussion  string      `json:"hazard_discussion"`
	WeatherDiscussion interface{} `json:"weather_discussion"`
	Announcement      interface{} `json:"announcement"`
	Status            string      `json:"status"`
	Media             []struct {
		Id  int `json:"id"`
		Url struct {
			Large     string `json:"large"`
			Medium    string `json:"medium"`
			Original  string `json:"original"`
			Thumbnail string `json:"thumbnail"`
		} `json:"url"`
		Type     string  `json:"type"`
		Title    *string `json:"title"`
		Caption  string  `json:"caption"`
		Favorite bool    `json:"favorite"`
	} `json:"media"`
	WeatherData     interface{} `json:"weather_data"`
	JsonData        interface{} `json:"json_data"`
	AvalancheCenter struct {
		Id    string `json:"id"`
		Name  string `json:"name"`
		Url   string `json:"url"`
		City  string `json:"city"`
		State string `json:"state"`
	} `json:"avalanche_center"`
	ForecastAvalancheProblems []struct {
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
	} `json:"forecast_avalanche_problems"`
	Danger []struct {
		Lower    int    `json:"lower"`
		Upper    int    `json:"upper"`
		Middle   int    `json:"middle"`
		ValidDay string `json:"valid_day"`
	} `json:"danger"`
	ForecastZone []struct {
		Id     int         `json:"id"`
		Name   string      `json:"name"`
		Url    string      `json:"url"`
		State  string      `json:"state"`
		ZoneId string      `json:"zone_id"`
		Config interface{} `json:"config"`
	} `json:"forecast_zone"`
}
