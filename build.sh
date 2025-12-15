#!/bin/bash

# Build script for artillery-ota-server with embed options

set -e

echo "Building artillery-ota-server..."

# Generate embedded files if needed
echo "Generating embedded files..."
go run tools/generate-embed/main.go

# Create bin directory if it doesn't exist
mkdir -p bin

# Build the regular version (AMD64 Linux)
echo "Building AMD64 Linux version..."
GOOS=linux GOARCH=amd64 go build -o bin/ota-server-linux-amd64 .

# Build ARM Linux version
echo "Building ARM Linux version..."
GOOS=linux GOARCH=arm64 go build -o bin/ota-server-linux-arm64 .

# Build macOS version (M1/M2 - ARM)
echo "Building macOS ARM version..."
GOOS=darwin GOARCH=arm64 go build -o bin/ota-server-darwin-arm64 .

# Build Windows version
echo "Building Windows version..."
GOOS=windows GOARCH=amd64 go build -o bin/ota-server-windows-amd64.exe .

echo "Multi-platform builds completed!"

# Show binary information
echo ""
echo "Binaries built:"
ls -lah bin/