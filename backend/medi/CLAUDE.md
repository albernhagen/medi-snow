# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run Commands

```bash
make build              # Build API binary to bin/api
make test               # Run all unit tests (excludes integration tests)
make test-coverage      # Run tests with HTML coverage report
make tools              # Install swag for Swagger generation

# Run a single test
go test -run TestFunctionName ./internal/package/

# Run integration tests (makes real HTTP calls to external APIs)
go test -tags integration ./internal/...
go test -tags integration -run TestCaptureSnapshots ./cmd/snapshot/

# Run the API server (reads config.yaml, default port 8080)
./bin/api
```

## Architecture

This is a Go backend API that aggregates weather, avalanche, and location data from multiple external providers into unified domain models.

### Three-Layer Pattern

**Providers** (`internal/providers/`) — HTTP clients that call external APIs and return raw API response structs. Each provider has `client.go` (HTTP methods) and `models.go` (response types). Providers: `openmeteo`, `nac` (National Avalanche Center), `nws` (National Weather Service), `usgs` (elevation), `openstreetmap` (reverse geocode).

**Domain Services** (`internal/weather/`, `internal/avalanche/`, `internal/location/`) — Business logic layer. Each defines provider interfaces at the service level (not in provider packages), maps provider responses to domain models, and exposes a `Service` interface. Every service has two constructors:
- `NewXxxService(logger)` — production, creates real provider clients
- `NewXxxServiceWithProviders(logger, ...providers)` — testable, accepts interface dependencies

**API Handlers** (`cmd/api/`) — Gin HTTP handlers that compose domain services. `App` struct holds all service dependencies.

### Key Design Decisions

- **Provider interfaces are defined by consumers**: e.g. `weather.ForecastProvider` lives in `internal/weather/`, not in `internal/providers/openmeteo/`. This is standard Go interface segregation.
- **`nac.Client` satisfies two interfaces**: Both `avalanche.MapLayerProvider` and `avalanche.ForecastProvider`, so `NewAvalancheService` passes the same client twice.
- **`nac.MapLayerGeometry` has custom UnmarshalJSON**: It handles both GeoJSON Polygon and MultiPolygon types. The coordinates are stored in unexported fields, so serializing a `MapLayerResponse` back to JSON loses geometry data. Use raw HTTP response capture when saving fixtures.
- **Weather service uses 7 parallel weather models**: `ModelValues[T]` is a `map[string]T` keyed by model name constants. Some models lack certain fields (commented out in `service.go`).
- **Timezone lookup uses tzf library**: `timezone.NewService()` is a singleton that loads ~50MB of timezone polygon data via `sync.Once`.

### Testing Patterns

- **No external test libraries** — all tests use only the standard `testing` package.
- **Unit tests** call unexported mapping functions directly (e.g. `mapForecastResponse`, `mapForecastAPIResponseToForecast`) rather than going through the service interface.
- **Mock providers** are defined inline in test files as simple structs implementing the provider interfaces.
- **Snapshot-based tests** load JSON from `testdata/` directories via `os.ReadFile`, unmarshal to provider response types, and feed them through domain mapping functions.
- **Integration tests** use `//go:build integration` build tag and make real HTTP calls. They are excluded from `make test` / `go test ./...`.
- **Testdata request files** (`*_request.json`) document the HTTP request details (URL, method, query params) alongside response snapshots for reference.

## Configuration

Config loaded via Viper from `config.yaml` or env vars prefixed with `MEDI_`. Defaults: port 8080, 16 forecast days, release gin mode.
