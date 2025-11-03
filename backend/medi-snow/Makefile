.PHONY: help
.DEFAULT_GOAL := help

## Help command
help:
	@echo "Medi-Snow Backend - Available Commands"
	@echo ""
	@echo "API Commands:"
	@echo "  make api              - Run the API server (alias for api-run)"
	@echo "  make api-run          - Run the API server"
	@echo "  make api-build        - Build the API binary to bin/api"
	@echo "  make api-test         - Run API-specific tests"
	@echo ""
	@echo "Demo Commands:"
	@echo "  make demo             - Run the avalanche demo CLI"
	@echo "  make demo-build       - Build the demo binary to bin/avalanche-demo"
	@echo ""
	@echo "Testing:"
	@echo "  make test             - Run all unit tests"
	@echo "  make test-integration - Run integration tests (makes real API calls)"
	@echo "  make test-coverage    - Run tests with coverage report"
	@echo ""
	@echo "Project-wide:"
	@echo "  make build            - Build all binaries"
	@echo "  make clean            - Remove build artifacts"
	@echo "  make deps             - Download and tidy dependencies"
	@echo "  make fmt              - Format code"
	@echo "  make lint             - Run linter"

# ============================================================================
# API Commands
# ============================================================================
.PHONY: api api-run api-build api-test

## Alias for api-run (quick shortcut)
api: api-run

## Run the API server
api-run:
	@echo "Starting API server on http://localhost:8080"
	@echo "API docs available at http://localhost:8080/docs"
	@go run ./cmd/api

## Build the API binary
api-build:
	@echo "Building API server..."
	@mkdir -p bin
	@go build -o bin/api ./cmd/api
	@echo "✓ API binary created at bin/api"

## Run API-specific tests
api-test:
	@echo "Running API tests..."
	@go test -v ./cmd/api/...

# ============================================================================
# Demo Commands
# ============================================================================
.PHONY: demo demo-build

## Run the avalanche demo CLI
demo:
	@go run ./cmd/avalanche-demo

## Build the demo binary
demo-build:
	@echo "Building avalanche demo..."
	@mkdir -p bin
	@go build -o bin/avalanche-demo ./cmd/avalanche-demo
	@echo "✓ Demo binary created at bin/avalanche-demo"

# ============================================================================
# Testing
# ============================================================================
.PHONY: test test-integration test-coverage

## Run all unit tests
test:
	@echo "Running all unit tests..."
	@go test ./...

## Run integration tests (requires real API calls)
test-integration:
	@echo "Running integration tests (this will make real API calls)..."
	@go test -tags=integration -v ./internal/providers/openmeteo
	@go test -tags=integration -v ./internal/providers/openstreetmap
	@go test -tags=integration -v ./internal/providers/nws

## Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	@go test ./... -coverprofile=coverage.out
	@go tool cover -html=coverage.out
	@echo "✓ Coverage report opened in browser"

# ============================================================================
# Project-wide Commands
# ============================================================================
.PHONY: build clean deps fmt lint

## Build all binaries
build: api-build demo-build
	@echo "✓ All binaries built successfully"

## Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -f coverage.out
	@go clean
	@echo "✓ Clean complete"

## Install and tidy dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "✓ Dependencies updated"

## Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "✓ Code formatted"

## Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@golangci-lint run
	@echo "✓ Linting complete"
