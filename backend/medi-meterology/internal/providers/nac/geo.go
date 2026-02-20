package nac

// FindZone returns the first MapLayerFeature whose geometry contains the given
// latitude/longitude, or nil if no zone matches.
func FindZone(lat, lon float64, mapLayer *MapLayerResponse) *MapLayerFeature {
	for i := range mapLayer.Features {
		for _, ring := range mapLayer.Features[i].Geometry.Coordinates() {
			if pointInPolygon(lat, lon, ring) {
				return &mapLayer.Features[i]
			}
		}
	}
	return nil
}

// pointInPolygon uses the ray-casting algorithm to determine if a point is
// inside a polygon. GeoJSON coordinates are [longitude, latitude], so
// polygon[i][0] is longitude and polygon[i][1] is latitude.
func pointInPolygon(lat, lon float64, polygon [][2]float64) bool {
	n := len(polygon)
	inside := false
	for i, j := 0, n-1; i < n; j, i = i, i+1 {
		yi, xi := polygon[i][1], polygon[i][0]
		yj, xj := polygon[j][1], polygon[j][0]

		if ((yi > lat) != (yj > lat)) &&
			(lon < (xj-xi)*(lat-yi)/(yj-yi)+xi) {
			inside = !inside
		}
	}
	return inside
}
