# Integration test Makefile

.PHONY: test build-server build-web clean help

# Default target
help:
	@echo "Available targets:"
	@echo "  test         - Run all integration tests"
	@echo "  test-server  - Run server startup tests only"
	@echo "  test-ws      - Run WebSocket tests only"
	@echo "  build-server - Build the Go server binary"
	@echo "  build-web    - Build the Web UI"
	@echo "  build-all    - Build both server and web UI"
	@echo "  clean        - Clean built artifacts"
	@echo "  help         - Show this help message"

# Build targets
build-server:
	@echo "Building Go server..."
	cd .. && go build -o echonet-list .

build-web:
	@echo "Building Web UI..."
	cd ../web && npm ci && npm run build

build-all: build-server build-web

# Test targets
test: build-all
	@echo "Running all integration tests..."
	cd .. && go test -v -tags=integration -timeout=5m ./integration/tests/...

test-server: build-all
	@echo "Running server startup tests..."
	cd .. && go test -v -tags=integration -timeout=5m -run "TestServer" ./integration/tests/...

test-ws: build-all
	@echo "Running WebSocket tests..."
	cd .. && go test -v -tags=integration -timeout=5m -run "TestWebSocket" ./integration/tests/...

# Clean target
clean:
	@echo "Cleaning built artifacts..."
	cd .. && rm -f echonet-list
	cd ../web && rm -rf bundle node_modules/.vite

# Convenience targets for development
watch:
	@echo "Watching for changes and running tests..."
	cd .. && find . -name "*.go" -o -name "*.ts" -o -name "*.tsx" | entr -r make -C integration test

.DEFAULT_GOAL := help