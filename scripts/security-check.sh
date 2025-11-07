#!/bin/bash

set -e

export PATH=$PATH:$(go env GOPATH)/bin

echo "Running Security Compliance Checks..."

echo "Checking Go module dependencies..."
go mod verify
go mod download

echo "Running vulnerability scan..."
if command -v govulncheck &> /dev/null; then
    govulncheck ./...
else
    echo "Installing govulncheck..."
    go install golang.org/x/vuln/cmd/govulncheck@latest
    govulncheck ./...
fi

echo "Running security static analysis..."
if command -v gosec &> /dev/null; then
    gosec -fmt json -out gosec-report.json ./...
else
    echo "Installing gosec..."
    go install github.com/securego/gosec/v2/cmd/gosec@latest
    gosec -fmt json -out gosec-report.json ./...
fi

echo "Checking license compliance..."
if command -v go-licenses &> /dev/null; then
    go-licenses report ./... > licenses-report.txt
    echo "License report generated: licenses-report.txt"
else
    echo "Installing go-licenses..."
    go install github.com/google/go-licenses@latest
    go-licenses report ./... > licenses-report.txt
    echo "License report generated: licenses-report.txt"
fi

echo "Scanning Docker image for vulnerabilities..."
if command -v trivy &> /dev/null; then
    docker build -t conductor:security-scan .
    trivy image conductor:security-scan --format json --output trivy-report.json
    echo "Docker security scan completed: trivy-report.json"
else
    echo "Trivy not found. Install with: brew install aquasecurity/trivy/trivy"
fi

echo "Security compliance checks completed!"
echo "Reports generated:"
echo "  - gosec-report.json (static analysis)"
echo "  - licenses-report.txt (license compliance)"
echo "  - trivy-report.json (Docker vulnerabilities)"
