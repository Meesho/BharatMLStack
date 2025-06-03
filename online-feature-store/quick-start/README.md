# Online Feature Store Quick Start Guide

A quick way to get the Online Feature Store platform up and running locally for development and testing.

## Prerequisites

- Docker and Docker Compose
- Go 1.22 or later
- `nc` (netcat) command for connectivity checks
- Bash shell

## System Components

BharatMLStack's Online Feature Store consists of several interconnected services:

- **ScyllaDB**: NoSQL database for high-performance storage
- **MySQL**: Relational database for structured data
- **Redis | Dragonfly**: In-memory data store for caching
- **etcd**: Distributed key-value store for configuration
- **Horizon**: Backend API service (runs on port 8082)
- **Trufflebox**: Frontend UI (runs on port 3000)
- **Online-feature-store gRPC API Server**: gRPC interface for Online Feature Store services (runs on port 8089)

## Quick Start

### Starting the System

Run the start script to set up your workspace and launch all services:

```bash
./start.sh
```

This will:
1. Check for Go installation (1.22+ required)
2. Create a workspace directory
3. Start all Docker services via docker-compose
4. Initialize databases with required schemas
5. Launch Horizon, Trufflebox, and Online Feature Store gRPC API server

Once complete, you can access:
- Trufflebox UI: http://localhost:3000
- Horizon API: http://localhost:8082
- Online Feature Store gRPC API Server: http://localhost:8089

### Stopping the System

To stop all services:

```bash
./stop.sh
```

To stop and completely purge all containers, volumes, and workspace:

```bash
./stop.sh --purge
```

## Accessing Services

### Frontend UI
- URL: http://localhost:3000
- Default admin credentials:
  - Email: admin@admin.com
  - Password: admin

### Database Access

- **MySQL**:
  - Host: localhost
  - Port: 3306
  - Username: root
  - Password: root
  - Database: testdb

- **ScyllaDB**:
  - Host: localhost
  - Port: 9042
  - Keyspace: onfs

- **Redis**:
  - Host: localhost
  - Port: 6379

- **etcd**:
  - Endpoint: http://localhost:2379

## Troubleshooting

### Viewing Container Logs

```bash
# View logs for specific containers
docker logs -f horizon
docker logs -f trufflebox
docker logs -f onfs-grpc-api-server
```

### Common Issues

1. **Port conflicts**: Ensure ports 3000, 8082, 8089, 9042, 3306, 6379, and 2379 are not in use by other applications.

2. **Docker network issues**: If containers can't communicate, try restarting Docker or recreating the network:
   ```bash
   docker network rm onfs-network
   docker network create onfs-network
   ```

3. **Database connectivity**: If services can't connect to databases, ensure the databases are running:
   ```bash
   docker ps | grep -E 'mysql|scylla|redis|etcd'
   ```

## Development

The workspace directory contains all runtime artifacts. You can modify configurations in the following files:

- `workspace/docker-compose.yml` - Docker Compose configuration
- Container environment variables in the boot scripts

## License

This project is proprietary software of Meesho Technologies.
