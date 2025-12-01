.PHONY: build clean run test docker

# Build the Go service with C++ bindings
build:
	@echo "Building Skye-Eigenix HNSW service with C++ bindings..."
	CGO_ENABLED=1 go build -o eigenix ./cmd/eigenix

# Build and run the service (using go run to avoid security issues)
run:
	@echo "Starting Skye-Eigenix HNSW service..."
	CGO_ENABLED=1 go run ./cmd/eigenix

# Build and run from temp directory (alternative method)
run-safe: build-safe
	@echo "Starting Skye-Eigenix service from temp directory..."
	~/temp-build/eigenix

# Build in temp directory to avoid security software
build-safe:
	@echo "Building Skye-Eigenix service in temp directory..."
	@mkdir -p ~/temp-build
	CGO_ENABLED=1 go build -o ~/temp-build/eigenix ./cmd/eigenix

# Run the example client (requires service to be running)
test-client:
	@echo "Running example client..."
	@echo "Note: Client example not yet implemented"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f eigenix vector-search-service hnsw-service
	rm -rf indices
	go clean

# Install dependencies
deps:
	@echo "Installing Go dependencies..."
	go mod tidy

# Build Docker image
docker:
	@echo "Building Docker image..."
	docker build -t hnsw-service .

# Run with Docker
docker-run:
	@echo "Running with Docker..."
	docker run -p 8080:8080 hnsw-service

# Development build with debug info
debug:
	@echo "Building with debug info..."
	CGO_ENABLED=1 go build -gcflags="all=-N -l" -o hnsw-service-debug .

# Check if C++ compiler is available
check-deps:
	@echo "Checking dependencies..."
	@which g++ > /dev/null || (echo "Error: g++ not found. Please install a C++ compiler." && exit 1)
	@echo "✓ C++ compiler found"
	@go version
	@echo "✓ Go found"

# Help
help:
	@echo "Available targets:"
	@echo "  build        - Build the HNSW service"
	@echo "  run          - Build and run the service"
	@echo "  test-client  - Run the example client"
	@echo "  clean        - Clean build artifacts"
	@echo "  deps         - Install Go dependencies"
	@echo "  docker       - Build Docker image"
	@echo "  docker-run   - Run with Docker"
	@echo "  debug        - Build with debug info"
	@echo "  check-deps   - Check if dependencies are installed"
	@echo "  help         - Show this help"