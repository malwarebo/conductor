.PHONY: test test-unit test-integration test-coverage test-race test-verbose clean diagram

test:
	@echo "Running all tests..."
	go test ./...

test-unit:
	@echo "Running unit tests..."
	go test -short ./...

test-integration:
	@echo "Running integration tests..."
	go test -run Integration ./...

test-coverage:
	@echo "Generating coverage report..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-race:
	@echo "Running tests with race detector..."
	go test -race ./...

test-verbose:
	@echo "Running tests with verbose output..."
	go test -v ./...

test-security:
	@echo "Running security tests..."
	go test -v ./security/...

test-utils:
	@echo "Running utils tests..."
	go test -v ./utils/...

test-api:
	@echo "Running API tests..."
	go test -v ./api/...

clean:
	@echo "Cleaning test cache and coverage files..."
	go clean -testcache
	rm -f coverage.out coverage.html

build:
	@echo "Building application..."
	go build -o conductor .

run:
	@echo "Running application..."
	go run main.go

diagram:
	@echo "Launching architecture diagram..."
	go run ./cmd/diagram

security-scan:
	@echo "Running security scan..."
	./scripts/security-check.sh

help:
	@echo "Available targets:"
	@echo "  test              - Run all tests"
	@echo "  test-unit         - Run unit tests only"
	@echo "  test-integration  - Run integration tests"
	@echo "  test-coverage     - Generate coverage report"
	@echo "  test-race         - Run tests with race detector"
	@echo "  test-verbose      - Run tests with verbose output"
	@echo "  test-security     - Run security component tests"
	@echo "  test-utils        - Run utility tests"
	@echo "  test-api          - Run API tests"
	@echo "  clean             - Clean test cache and coverage files"
	@echo "  build             - Build the application"
	@echo "  run               - Run the application"
	@echo "  diagram           - Launch interactive architecture diagram"
	@echo "  security-scan     - Run security vulnerability scan"

