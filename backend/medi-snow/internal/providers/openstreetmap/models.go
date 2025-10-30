package openstreetmap

type LookupAPIResponse struct {
	PlaceId     int     `json:"place_id"`
	Licence     string  `json:"licence"`
	OsmType     string  `json:"osm_type"`
	OsmId       int     `json:"osm_id"`
	Lat         string  `json:"lat"`
	Lon         string  `json:"lon"`
	Class       string  `json:"class"`
	Type        string  `json:"type"`
	PlaceRank   int     `json:"place_rank"`
	Importance  float64 `json:"importance"`
	Addresstype string  `json:"addresstype"`
	Name        string  `json:"name"`
	DisplayName string  `json:"display_name"`
	Address     struct {
		County       string `json:"county"`
		State        string `json:"state"`
		ISO31662Lvl4 string `json:"ISO3166-2-lvl4"`
		Country      string `json:"country"`
		CountryCode  string `json:"country_code"`
	} `json:"address"`
	Boundingbox []string `json:"boundingbox"`
}
