package usgs

type ElevationPointAPIResponse struct {
	Location struct {
		X                float64 `json:"x"`
		Y                float64 `json:"y"`
		SpatialReference struct {
			Wkid       int `json:"wkid"`
			LatestWkid int `json:"latestWkid"`
		} `json:"spatialReference"`
	} `json:"location"`
	LocationId int     `json:"locationId"`
	Value      float64 `json:"value"`
	RasterId   int     `json:"rasterId"`
	Resolution float64 `json:"resolution"`
}
