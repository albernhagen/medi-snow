package timezone

import (
	"fmt"
	"sync"

	"github.com/ringsaturn/tzf"
)

// Service provides timezone lookup functionality
type Service interface {
	GetTimezone(latitude, longitude float64) (string, error)
}

// service implements timezone lookup using tzf
type service struct {
	finder tzf.F
	mu     sync.RWMutex
}

var (
	instance *service
	once     sync.Once
)

// NewService creates or returns the singleton timezone service
// Uses singleton pattern because tzf.Finder loads timezone data into memory (~50MB)
func NewService() (Service, error) {
	var err error
	once.Do(func() {
		finder, findErr := tzf.NewDefaultFinder()
		if findErr != nil {
			err = fmt.Errorf("failed to initialize timezone finder: %w", findErr)
			return
		}
		instance = &service{
			finder: finder,
		}
	})
	if err != nil {
		return nil, err
	}
	return instance, nil
}

// GetTimezone returns the IANA timezone name for the given coordinates
// Returns timezone names like "America/Denver", "Europe/London", etc.
func (s *service) GetTimezone(latitude, longitude float64) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	timezone := s.finder.GetTimezoneName(longitude, latitude)
	if timezone == "" {
		return "", fmt.Errorf("could not determine timezone for coordinates lat=%f, lon=%f", latitude, longitude)
	}

	return timezone, nil
}
