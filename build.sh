#!/bin/bash

# WinRamp Build Script

set -e

echo "🎵 Building WinRamp..."

# Check prerequisites
echo "Checking prerequisites..."
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.22+"
    exit 1
fi

if ! command -v wails &> /dev/null; then
    echo "❌ Wails is not installed. Installing..."
    go install github.com/wailsapp/wails/v2/cmd/wails@latest
fi

if ! command -v npm &> /dev/null; then
    echo "❌ npm is not installed. Please install Node.js"
    exit 1
fi

# Clean previous builds
echo "Cleaning previous builds..."
rm -rf build/bin
rm -rf frontend/dist

# Install Go dependencies
echo "Installing Go dependencies..."
go mod download
go mod tidy

# Install frontend dependencies
echo "Installing frontend dependencies..."
cd frontend
npm install
cd ..

# Run tests
echo "Running tests..."
go test ./... -v

# Build the application
echo "Building application..."
wails build -platform windows/amd64 -clean

echo "✅ Build complete! Binary location: build/bin/winramp.exe"
echo ""
echo "To run the application:"
echo "  ./build/bin/winramp.exe"
echo ""
echo "To create an installer:"
echo "  make installer"