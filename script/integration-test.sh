#!/bin/bash

# Integration test runner script
# This script builds both the Go server and Web UI, then runs integration tests

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Get script directory and project root
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." &> /dev/null && pwd )"

print_status "Starting integration test suite..."
print_status "Project root: $PROJECT_ROOT"

# Change to project root
cd "$PROJECT_ROOT"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    print_error "Go is not installed or not in PATH"
    exit 1
fi

# Check if Node.js is installed
if ! command -v node &> /dev/null; then
    print_error "Node.js is not installed or not in PATH"
    exit 1
fi

# Check if npm is installed
if ! command -v npm &> /dev/null; then
    print_error "npm is not installed or not in PATH"
    exit 1
fi

# Build Go server
print_status "Building Go server..."
if go build -o echonet-list .; then
    print_status "Go server built successfully"
else
    print_error "Failed to build Go server"
    exit 1
fi

# Build Web UI
print_status "Building Web UI..."
cd web

# Install dependencies if node_modules doesn't exist
if [ ! -d "node_modules" ]; then
    print_status "Installing npm dependencies..."
    if npm ci; then
        print_status "npm dependencies installed successfully"
    else
        print_error "Failed to install npm dependencies"
        exit 1
    fi
fi

# Build the Web UI
if npm run build; then
    print_status "Web UI built successfully"
else
    print_error "Failed to build Web UI"
    exit 1
fi

# Return to project root
cd "$PROJECT_ROOT"

# Check if web bundle was created
if [ ! -d "web/bundle" ]; then
    print_error "Web bundle directory not found after build"
    exit 1
fi

print_status "Bundle directory contents:"
ls -la web/bundle/

# Run integration tests
print_status "Running integration tests..."
if go test -v -tags=integration -timeout=5m ./integration/tests/...; then
    print_status "✅ All integration tests passed!"
else
    print_error "❌ Integration tests failed!"
    exit 1
fi

print_status "Integration test suite completed successfully!"