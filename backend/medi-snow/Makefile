.PHONY: build clean test test-integration run-api run-demo

# Build all binaries to bin/ directory
build:
	@echo "Building binaries..."
	@mkdir -p bin
	go build -o bin/api ./cmd/api
	go build -o bin/avalanche-demo ./cmd/avalanche-demo

# Run the API server
run-api:
	go run ./cmd/api

# Run the demo CLI
run-demo:
	go run ./cmd/avalanche-demo

# Run all tests (excluding integration tests)
test:
	go test ./...

# Run integration tests (requires real API calls)
test-integration:
	@echo "Running integration tests (this will make real API calls)..."
	go test -tags=integration -v ./internal/providers/openmeteo
	go test -tags=integration -v ./internal/providers/openstreetmap

# Run tests with coverage
test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out

# Install dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Run linter (requires golangci-lint)
lint:
	golangci-lint run

# Help command
help:
	@echo "Available commands:"
	@echo "  make build          - Build all binaries to bin/"
	@echo "  make run-api        - Run API server"
	@echo "  make run-demo       - Run avalanche demo CLI"
	@echo "  make test           - Run all tests (excludes integration)"
	@echo "  make test-integration - Run integration tests (makes real API calls)"
	@echo "  make test-coverage  - Run tests with coverage report"
	@echo "  make clean          - Remove build artifacts"
	@echo "  make deps           - Download and tidy dependencies"
	@echo "  make fmt            - Format code"
	@echo "  make lint           - Run linter"
