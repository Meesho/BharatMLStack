# 🚀 BharatML Stack Deployment Guide

This guide covers how to deploy the complete BharatML Stack using pre-built Docker images from GitHub Container Registry.

## 🏗️ Architecture Overview

```
┌─────────────────────┐    ┌─────────────────────┐    ┌─────────────────────┐
│   TruffleBox UI     │───▶│      Horizon        │───▶│  ONFS API Server    │
│   (Frontend :80)    │    │   (Backend :8082)   │    │     (:8089)         │
└─────────────────────┘    └─────────────────────┘    └─────────────────────┘
                                                               │
                                                               ▼
┌─────────────────────┐    ┌─────────────────────┐    ┌─────────────────────┐
│     ScyllaDB        │◄───│   ONFS Consumer     │◄───│      Kafka          │
│     (:9042)         │    │     (:8080)         │    │     (External)      │
└─────────────────────┘    └─────────────────────┘    └─────────────────────┘
```

## 📦 Pre-built Images

All components are automatically built and pushed to GitHub Container Registry:

- `ghcr.io/meesho/horizon:latest` - Backend API service
- `ghcr.io/meesho/onfs-api-server:latest` - Online Feature Store API
- `ghcr.io/meesho/onfs-consumer:latest` - Feature Store Consumer
- `ghcr.io/meesho/trufflebox-ui:latest` - Web Interface

## 🚀 Quick Start

### Prerequisites

- Docker and Docker Compose installed
- At least 4GB RAM available
- Ports 80, 8080-8089, 3306, 6379, 9042, 2379-2380 available

### 1. Clone and Deploy

```bash
# Clone the repository
git clone https://github.com/Meesho/BharatMLStack.git
cd BharatMLStack

# Start the complete stack
docker-compose up -d

# Check service status
docker-compose ps
```

### 2. Wait for Services to Start

```bash
# Monitor logs
docker-compose logs -f

# Check individual service health
docker-compose logs trufflebox-ui
docker-compose logs horizon
docker-compose logs onfs-api-server
docker-compose logs onfs-consumer
```

### 3. Access the Services

- **TruffleBox UI**: http://localhost
- **Horizon API**: http://localhost:8082/api/v1
- **Feature Store API**: http://localhost:8089
- **Feature Store Consumer**: http://localhost:8080
- **ETCD Workbench**: http://localhost:8081

## 🔧 Configuration

### Environment Variables

Create a `.env` file to customize the deployment:

```bash
# Database Configuration
MYSQL_ROOT_PASSWORD=your_root_password
MYSQL_USER=your_user
MYSQL_PASSWORD=your_password
MYSQL_DATABASE=bharatml_db

# Redis Configuration
REDIS_PASSWORD=your_redis_password

# Application Configuration
HORIZON_PORT=8082
ONFS_API_PORT=8089
ONFS_CONSUMER_PORT=8080
UI_PORT=80

# External Services
KAFKA_BROKERS=your_kafka_brokers
```

### Custom Configuration

For production deployments, override the docker-compose configuration:

```bash
# Create override file
cat > docker-compose.override.yml << EOF
version: '3.8'
services:
  horizon:
    environment:
      - ENV=production
      - DB_SSL_MODE=require
      - LOG_LEVEL=info
    
  onfs-api-server:
    environment:
      - LOG_LEVEL=info
      - METRICS_ENABLED=true
    
  trufflebox-ui:
    environment:
      - REACT_APP_API_BASE_URL=https://your-domain.com/api/v1
EOF

# Deploy with overrides
docker-compose -f docker-compose.yml -f docker-compose.override.yml up -d
```

## 🔄 Development Workflow

### Building Images Locally

If you need to build images locally instead of using pre-built ones:

```bash
# Build all images
docker-compose -f docker-compose.dev.yml build

# Build specific service
docker-compose -f docker-compose.dev.yml build horizon
```

### Using Specific Image Tags

```bash
# Use specific version
export HORIZON_TAG=v1.2.3
export ONFS_TAG=develop
docker-compose up -d
```

## 📊 Monitoring and Health Checks

### Health Check Endpoints

All services provide health check endpoints:

```bash
# Check all service health
curl http://localhost:8082/api/v1/health    # Horizon
curl http://localhost:8089/health           # ONFS API Server  
curl http://localhost:8080/health           # ONFS Consumer
curl http://localhost/health                # TruffleBox UI (via nginx)
```

### Service Dependencies

The services start in the correct order with health checks:

1. **Infrastructure**: ScyllaDB, MySQL, Redis, ETCD
2. **Core Services**: ONFS API Server
3. **Processing**: ONFS Consumer
4. **Backend**: Horizon
5. **Frontend**: TruffleBox UI

### Logs and Debugging

```bash
# View all logs
docker-compose logs

# Follow specific service logs
docker-compose logs -f horizon

# View last 100 lines
docker-compose logs --tail=100 onfs-api-server

# Search logs
docker-compose logs | grep ERROR
```

## 🔐 Security Considerations

### Production Deployment

For production deployments:

```bash
# Use strong passwords
export MYSQL_ROOT_PASSWORD=$(openssl rand -base64 32)
export MYSQL_PASSWORD=$(openssl rand -base64 32)
export REDIS_PASSWORD=$(openssl rand -base64 32)

# Enable SSL/TLS
export DB_SSL_MODE=require
export REDIS_TLS=true

# Configure proper CORS
export CORS_ORIGINS=https://your-domain.com
```

### Network Security

```bash
# Create custom network with encryption
docker network create bharatml-secure --driver overlay --opt encrypted

# Use secrets for sensitive data
echo "your_secret_password" | docker secret create mysql_password -
```

## 🔄 CI/CD Integration

### Automated Builds

Images are automatically built on:
- **Push to master/develop**: `latest` tag
- **Pull requests**: `pr-<number>` tag  
- **Git tags**: `v1.2.3` tag and `1.2` tag
- **Branch pushes**: `<branch>-<sha>` tag

### Using Specific Builds

```bash
# Use develop branch build
docker-compose pull
HORIZON_TAG=develop docker-compose up -d horizon

# Use specific commit
HORIZON_TAG=main-abc1234 docker-compose up -d horizon

# Use release version
HORIZON_TAG=v1.0.0 docker-compose up -d
```

## 🚨 Troubleshooting

### Common Issues

**Services failing to start:**
```bash
# Check resource usage
docker stats

# Verify port availability
netstat -tulpn | grep :8089

# Check service logs
docker-compose logs onfs-api-server
```

**Database connection issues:**
```bash
# Test database connectivity
docker-compose exec horizon sh -c "curl -f mysql:3306"

# Check database logs
docker-compose logs mysql
```

**Frontend not loading:**
```bash
# Check nginx configuration
docker-compose exec trufflebox-ui cat /etc/nginx/conf.d/default.conf

# Verify backend connectivity
curl http://localhost:8082/api/v1/health
```

### Reset and Clean Up

```bash
# Stop all services
docker-compose down

# Remove volumes (⚠️  This deletes all data)
docker-compose down -v

# Clean up images
docker system prune -f

# Complete reset
docker-compose down -v --rmi all
docker system prune -af
```

## 🎯 Next Steps

1. **Configure External Services**: Set up Kafka brokers for production
2. **Set up Monitoring**: Add Prometheus/Grafana for observability
3. **Configure SSL**: Enable HTTPS for production deployments
4. **Set up Backups**: Configure database backup strategies
5. **Load Testing**: Test with production-level traffic

## 📖 Additional Resources

- [TruffleBox UI Documentation](./trufflebox-ui/README.md)
- [Horizon API Documentation](./horizon/README.md)
- [Online Feature Store Documentation](./online-feature-store/README.md)
- [Python SDK Documentation](./py-sdk/README.md)

## 🤝 Support

- 📖 [Documentation](https://github.com/Meesho/BharatMLStack)
- 🐛 [Report Issues](https://github.com/Meesho/BharatMLStack/issues)
- 💬 [Discussions](https://github.com/Meesho/BharatMLStack/discussions) 