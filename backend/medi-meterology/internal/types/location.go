package types

// LocationInfo contains human-readable location metadata
type LocationInfo struct {
	Name        string `json:"name" example:"Denver" doc:"Location name"`
	County      string `json:"county" example:"Denver County" doc:"County name"`
	State       string `json:"state" example:"Colorado" doc:"State or province name"`
	Country     string `json:"country" example:"United States" doc:"Country name"`
	CountryCode string `json:"country_code" example:"US" doc:"ISO country code"`
}
