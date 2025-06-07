![Build Status](https://github.com/Meesho/BharatMLStack/workflows/CI%20Build%20and%20Test/badge.svg) ![Build](https://img.shields.io/badge/build-unknown-lightgrey) ![Version](https://img.shields.io/badge/version-unknown-lightgrey)

# Horizon

Model serving and inference service built with Go. Provides high-performance model serving capabilities for machine learning models.

## Features

- High-performance gRPC API for model inference
- Support for multiple model formats
- Horizontal scaling capabilities
- Monitoring and observability
- Docker containerization

## Development

### Prerequisites

- Go 1.19 or later
- Docker (optional, for containerization)

### Getting Started

```bash
# Install dependencies
go mod download

# Run tests
go test -v ./...

# Build the application
go build -v ./...

# Run the service
go run ./cmd/...
```

### Testing

```bash
# Run all tests
go test -v ./...

# Run tests with coverage
go test -v -cover ./...

# Run specific test package
go test -v ./pkg/...
```

### Building

```bash
# Build for current platform
go build -v ./...

# Build for specific platform
GOOS=linux GOARCH=amd64 go build -v ./...
```

## Docker

### Building the Docker Image

```bash
# Build Docker image
docker build -t horizon -f cmd/horizon/Dockerfile .

# Run container
docker run -p 8080:8080 horizon
```

### Docker Compose (Development)

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f horizon

# Stop services
docker-compose down
```

## Configuration

The service can be configured using environment variables or configuration files. See the example configuration files in the `configs/` directory.

## API Documentation

The gRPC API documentation is generated from the proto files. See the `api/` directory for proto definitions.

## Deployment

### Kubernetes

```bash
# Apply Kubernetes manifests
kubectl apply -f k8s/

# Check deployment status
kubectl get pods -l app=horizon
```

### Docker Swarm

```bash
# Deploy stack
docker stack deploy -c docker-compose.prod.yml horizon
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for your changes
5. Ensure all tests pass
6. Submit a pull request

## License

See [LICENSE.md](../LICENSE.md) for details. 