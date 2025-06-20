![Build Status](https://github.com/Meesho/BharatMLStack/actions/workflows/horizon.yml/badge.svg)
![Static Badge](https://img.shields.io/badge/release-v1.0.0-blue?style=flat)
[![Discord](https://img.shields.io/badge/Discord-Join%20Chat-7289da?style=flat&logo=discord&logoColor=white)](https://discord.gg/XkT7XsV2AU)

# Horizon

Backend API service for [TruffleBox UI](../trufflebox-ui) - the web interface for BharatML Stack. Provides REST APIs and data management capabilities to support the frontend dashboard.

## Overview

Horizon serves as the backend infrastructure for TruffleBox UI, offering:

- **REST API Layer**: Provides endpoints for the TruffleBox UI frontend
- **Data Management**: Handles feature store metadata and configuration
- **Authentication & Authorization**: Manages user access and permissions
- **Integration Layer**: Connects with BharatML Stack components
- **Monitoring & Analytics**: Provides insights and monitoring capabilities

## Architecture

```
┌─────────────────┐    HTTP/REST     ┌─────────────────┐
│  TruffleBox UI  │ ◄─────────────► │    Horizon      │
│   (Frontend)    │                 │   (Backend)     │
└─────────────────┘                 └─────────────────┘
                                             │
                                             ▼
                                    ┌─────────────────┐
                                    │ BharatML Stack  │
                                    │   Components    │
                                    └─────────────────┘
```

## Features

- **RESTful APIs**: Clean HTTP APIs for frontend integration
- **High Performance**: Built with Go for optimal performance
- **Scalable**: Designed for horizontal scaling
- **Database Integration**: Persistent storage for configuration and metadata
- **CORS Support**: Proper cross-origin resource sharing for web frontend
- **Monitoring**: Built-in observability and health checks
- **Docker Ready**: Containerized for easy deployment

## API Endpoints

Horizon provides various API endpoints to support TruffleBox UI functionality:

- `/api/v1/features` - Feature management APIs
- `/api/v1/models` - Model metadata and serving
- `/api/v1/health` - Health check endpoints
- `/api/v1/metrics` - Monitoring and analytics
- `/api/v1/auth` - Authentication endpoints

## Development

### Prerequisites

- Go 1.22 or later
- Database (PostgreSQL/MySQL)
- Docker (optional, for containerization)

### Getting Started

```bash
# Clone and navigate to the project
cd horizon

# Install dependencies
go mod download

# Set up environment variables
cp .env.example .env
# Edit .env with your configuration

# Run database migrations (if applicable)
make migrate

# Run tests
go test -v ./...

# Build the application
go build -v ./cmd/horizon

# Run the service
go run ./cmd/horizon
```

### Configuration

Create a `.env` file or set environment variables:

```bash
# Server Configuration
PORT=8080
HOST=localhost

# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_NAME=horizon
DB_USER=your_user
DB_PASSWORD=your_password

# CORS Configuration (for TruffleBox UI)
CORS_ORIGINS=http://localhost:3000,http://localhost:8080

# BharatML Stack Integration
FEATURE_STORE_URL=http://localhost:9000
```

### Testing

```bash
# Run all tests
go test -v ./...

# Run tests with coverage
go test -v -cover ./...

# Run specific test package
go test -v ./pkg/api/...

# Run integration tests
go test -v -tags=integration ./...
```

### Building

```bash
# Build for current platform
go build -v ./cmd/horizon

# Build for production
make build

# Build for specific platform
GOOS=linux GOARCH=amd64 go build -v ./cmd/horizon
```

## Docker

### Building the Docker Image

```bash
# Build Docker image
docker build -t horizon -f cmd/horizon/Dockerfile .

# Run container with environment variables
docker run -p 8080:8080 \
  -e DB_HOST=host.docker.internal \
  -e DB_NAME=horizon \
  horizon
```

### Docker Compose (Development)

```bash
# Start all services (includes database)
docker-compose up -d

# View logs
docker-compose logs -f horizon

# Stop services
docker-compose down
```

## Integration with TruffleBox UI

Horizon is designed to work seamlessly with TruffleBox UI:

1. **Start Horizon Backend**:
   ```bash
   cd horizon
   go run ./cmd/horizon
   # Backend runs on http://localhost:8080
   ```

2. **Start TruffleBox UI Frontend**:
   ```bash
   cd trufflebox-ui
   npm install
   npm start
   # Frontend runs on http://localhost:3000
   ```

3. **Configure API Base URL** in TruffleBox UI:
   ```javascript
   // trufflebox-ui/.env
   REACT_APP_API_BASE_URL=http://localhost:8080/api/v1
   ```

## API Documentation

### Health Check
```bash
curl http://localhost:8080/api/v1/health
```

### Features API
```bash
# Get all features
curl http://localhost:8080/api/v1/features

# Get specific feature
curl http://localhost:8080/api/v1/features/{feature_id}
```

### Authentication
```bash
# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "user", "password": "pass"}'
```

## Deployment

### Production with Docker

```bash
# Build production image
docker build -t horizon:latest -f cmd/horizon/Dockerfile .

# Run in production mode
docker run -d \
  --name horizon \
  -p 8080:8080 \
  -e ENV=production \
  -e DB_HOST=your-db-host \
  horizon:latest
```

### Kubernetes

```bash
# Apply Kubernetes manifests
kubectl apply -f k8s/

# Check deployment status
kubectl get pods -l app=horizon

# Port forward for testing
kubectl port-forward svc/horizon 8080:8080
```

### Environment Variables

Key environment variables for production:

```bash
ENV=production
PORT=8080
DB_HOST=production-db-host
DB_PORT=5432
DB_NAME=horizon_prod
DB_SSL_MODE=require
CORS_ORIGINS=https://your-trufflebox-ui-domain.com
JWT_SECRET=your-jwt-secret
LOG_LEVEL=info
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests for your changes
5. Ensure all tests pass (`go test -v ./...`)
6. Update API documentation if needed
7. Submit a pull request

## Related Projects

- **[TruffleBox UI](../trufflebox-ui)**: Frontend web interface that uses this backend
- **[Online Feature Store](../online-feature-store)**: Core feature store service
- **[Python SDK](../py-sdk)**: Python client libraries

## License

Licensed under the BharatMLStack Business Source License 1.1. See [LICENSE.md](../LICENSE.md) for details. 

---

<div align="center">
  <strong>Built with ❤️ for the ML community from Meesho</strong>
</div>
<div align="center">
  <strong>If you find this useful, ⭐️ the repo — your support means the world to us!</strong>
</div>