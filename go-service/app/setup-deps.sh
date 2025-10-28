#!/bin/bash

# Script to download Go dependencies and generate go.sum
# This ensures the project has all required dependencies

echo "Downloading Go dependencies..."

# Ensure we're in the right directory
cd "$(dirname "$0")"

# Initialize go.sum if it doesn't exist
if [ ! -f "go.sum" ]; then
    echo "Generating go.sum..."
    go mod download
    go mod tidy
    go mod verify
fi

echo "✓ Dependencies downloaded successfully"
echo ""
echo "To build the project:"
echo "  go build -o hls-converter-api ."
echo ""
echo "To run tests:"
echo "  go test -v ./..."
echo ""
echo "To run the application:"
echo "  go run main.go"
