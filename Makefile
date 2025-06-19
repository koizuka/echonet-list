.PHONY: all build test integration-test clean help

# Default target
all: build

# Build the server
build:
	@echo "Building server..."
	go build -o echonet-list

# Build web UI
build-web:
	@echo "Building web UI..."
	cd web && npm install && npm run build

# Build everything
build-all: build build-web

# Run unit tests
test:
	@echo "Running unit tests..."
	go test -v ./...

# Run integration tests
integration-test: build
	@echo "Running integration tests..."
	@if [ ! -d "web/bundle" ]; then \
		echo "Web bundle not found. Building..."; \
		$(MAKE) build-web; \
	fi
	go test -v -tags=integration -timeout=5m ./integration/tests/...

# Run all tests
test-all: test integration-test

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f echonet-list
	rm -f test-echonet-list.log
	rm -f test-echonet-list.pid
	rm -rf web/bundle
	rm -rf web/node_modules

# Format Go code
fmt:
	@echo "Formatting Go code..."
	go fmt ./...

# Run linter
lint:
	@echo "Running linter..."
	go vet ./...

# Run web linter
lint-web:
	@echo "Running web linter..."
	cd web && npm run lint

# Show help
help:
	@echo "Available targets:"
	@echo "  make build           - Build the server"
	@echo "  make build-web       - Build the web UI"
	@echo "  make build-all       - Build server and web UI"
	@echo "  make test            - Run unit tests"
	@echo "  make integration-test - Run integration tests"
	@echo "  make test-all        - Run all tests"
	@echo "  make fmt             - Format Go code"
	@echo "  make lint            - Run Go linter"
	@echo "  make lint-web        - Run web linter"
	@echo "  make clean           - Clean build artifacts"
	@echo "  make help            - Show this help"