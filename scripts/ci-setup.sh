#!/bin/bash
set -e

echo "Setting up CI environment..."

# Ensure we're in the right directory
cd "$(dirname "$0")/.."

# Clean any existing modules
echo "Cleaning module cache..."
go clean -modcache

# Download dependencies
echo "Downloading dependencies..."
go mod download

# Verify dependencies
echo "Verifying dependencies..."
go mod verify

# Tidy up
echo "Tidying up modules..."
go mod tidy

# Test compilation
echo "Testing compilation..."
go build -v ./...

echo "CI setup completed successfully!"
