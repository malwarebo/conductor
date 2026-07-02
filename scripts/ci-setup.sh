#!/bin/bash
set -e

echo "Setting up CI environment..."

# Ensure we're in the right directory
cd "$(dirname "$0")/.."

# Download dependencies
echo "Downloading dependencies..."
go mod download

# Verify dependencies
echo "Verifying dependencies..."
go mod verify

# Test compilation
echo "Testing compilation..."
go build -v ./...

echo "CI setup completed successfully!"
