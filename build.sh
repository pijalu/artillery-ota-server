#!/bin/bash

# Build script for artillery-ota-server with embed options

set -e

echo "Building artillery-ota-server..."

# Initialize and update git submodules if they're not already present
echo "Checking git submodules..."
if [ -d ".git" ] || [ -f ".git" ]; then
    if [ ! -d "artillery-m1-debs" ] || [ -z "$(ls -A artillery-m1-debs 2>/dev/null)" ]; then
        echo "Initializing and updating git submodules..."
        git submodule init
        git submodule update --recursive
    else
        echo "Git submodules appear to be present, skipping initialization"
    fi
else
    echo "Not in a git repository, assuming submodules are handled externally"
fi

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