.PHONY: build test clean tools

# Build the API binary
build:
	@mkdir -p bin
	@go build -o bin/api ./cmd/api

# Run all tests
test:
	@go test ./...

# Run tests with coverage
test-coverage:
	@go test ./... -coverprofile=coverage.out
	@go tool cover -html=coverage.out

# Clean build artifacts
clean:
	@rm -rf bin/ docs/ coverage.out
	@go clean

# Install development tools (swag for swagger)
tools:
	@go install github.com/swaggo/swag/cmd/swag@v1.16.4
