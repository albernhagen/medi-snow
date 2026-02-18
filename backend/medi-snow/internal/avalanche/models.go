package avalanche

import (
	"fmt"
	"strings"
	"time"
)

// AvalancheForecast is the top-level provider-agnostic domain model.
type AvalancheForecast struct {
	Zone             ForecastZone
	Center           AvalancheCenter
	PublishedTime    time.Time
	ExpiresTime      time.Time
	Author           string
	BottomLine       string // HTML summary
	HazardDiscussion string // HTML discussion
	DangerRatings    []DangerRating
	Problems         []AvalancheProblem
	ForecastURL      string // link to center's web page
}

// ForecastZone identifies the geographic forecast zone.
type ForecastZone struct {
	Id    int
	Name  string
	State string
	URL   string
}

// AvalancheCenter identifies the issuing avalanche center.
type AvalancheCenter struct {
	Id    string // e.g. "CAIC", "BTAC"
	Name  string
	URL   string
	City  string
	State string
}

// DangerLevel is a normalized integer enum (0-5) matching the North American
// Avalanche Danger Scale.
type DangerLevel int

const (
	DangerNone         DangerLevel = 0
	DangerLow          DangerLevel = 1
	DangerModerate     DangerLevel = 2
	DangerConsiderable DangerLevel = 3
	DangerHigh         DangerLevel = 4
	DangerExtreme      DangerLevel = 5
)

var dangerLevelNames = map[DangerLevel]string{
	DangerNone:         "No Rating",
	DangerLow:          "Low",
	DangerModerate:     "Moderate",
	DangerConsiderable: "Considerable",
	DangerHigh:         "High",
	DangerExtreme:      "Extreme",
}

func (d DangerLevel) String() string {
	if name, ok := dangerLevelNames[d]; ok {
		return name
	}
	return fmt.Sprintf("Unknown (%d)", int(d))
}

// DangerRating represents danger by elevation band for a given day.
type DangerRating struct {
	ValidDay string // "current" or "tomorrow"
	Lower    DangerLevel
	Middle   DangerLevel
	Upper    DangerLevel
}

// AvalancheProblem describes a specific avalanche problem in the forecast.
type AvalancheProblem struct {
	Name       string
	Rank       int // 1 = primary problem
	Likelihood Likelihood
	Discussion string   // HTML
	Location   []string // aspect/elevation combos, e.g. "north upper"
	Size       AvalancheSize
	MediaURL   string // image URL if available (original size)
}

// Likelihood is a normalized enum for avalanche problem likelihood.
type Likelihood int

const (
	LikelihoodUnlikely      Likelihood = 1
	LikelihoodPossible      Likelihood = 2
	LikelihoodLikely        Likelihood = 3
	LikelihoodVeryLikely    Likelihood = 4
	LikelihoodAlmostCertain Likelihood = 5
)

var likelihoodNames = map[Likelihood]string{
	LikelihoodUnlikely:      "Unlikely",
	LikelihoodPossible:      "Possible",
	LikelihoodLikely:        "Likely",
	LikelihoodVeryLikely:    "Very Likely",
	LikelihoodAlmostCertain: "Almost Certain",
}

func (l Likelihood) String() string {
	if name, ok := likelihoodNames[l]; ok {
		return name
	}
	return fmt.Sprintf("Unknown (%d)", int(l))
}

// ParseLikelihood normalizes likelihood strings from various NAC centers.
// It handles both camelCase ("veryLikely") and space-separated ("very likely") formats.
func ParseLikelihood(s string) Likelihood {
	normalized := strings.ToLower(strings.TrimSpace(s))
	// Remove spaces and underscores for uniform comparison
	normalized = strings.ReplaceAll(normalized, " ", "")
	normalized = strings.ReplaceAll(normalized, "_", "")

	switch normalized {
	case "unlikely":
		return LikelihoodUnlikely
	case "possible":
		return LikelihoodPossible
	case "likely":
		return LikelihoodLikely
	case "verylikely":
		return LikelihoodVeryLikely
	case "almostcertain":
		return LikelihoodAlmostCertain
	default:
		return 0
	}
}

// AvalancheSize represents the min and max destructive size of an avalanche problem.
type AvalancheSize struct {
	Min float64
	Max float64
}
