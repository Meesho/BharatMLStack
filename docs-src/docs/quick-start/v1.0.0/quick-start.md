---
title: Quick Start
sidebar_position: 1
---

# BharatML Stack Quick Start Guide
[![Discord](https://img.shields.io/badge/Discord-Join%20Chat-7289da?style=flat&logo=discord&logoColor=white)](https://discord.gg/XkT7XsV2AU)

A quick way to get the BharatML Stack Online Feature Store platform up and running locally for development and testing.

## Prerequisites

- Docker and Docker Compose
- Go 1.22 or later
- `nc` (netcat) command for connectivity checks
- Bash shell
- `grpcurl` for testing gRPC API endpoints (install from [https://github.com/fullstorydev/grpcurl](https://github.com/fullstorydev/grpcurl))

## System Components

BharatMLStack's Online Feature Store consists of several interconnected services:

**Infrastructure Services:**
- **ScyllaDB**: NoSQL database for high-performance feature storage
- **MySQL**: Relational database for metadata and configuration
- **Redis**: In-memory data store for caching
- **etcd**: Distributed key-value store for service coordination

**Application Services:**
- **Horizon**: Backend API service (runs on port 8082)
- **Trufflebox UI**: Frontend web interface (runs on port 3000)
- **Online Feature Store gRPC API Server**: High-performance gRPC interface (runs on port 8089)
- **etcd Workbench**: etcd management interface (runs on port 8081)

All services are orchestrated using Docker Compose with pre-built images from GitHub Container Registry (GHCR).

## Quick Start

### Starting the System

Run the start script to set up your workspace and launch all services:

```bash
./start.sh
```

### Testing Different Versions

You can easily test different versions of the application services by setting environment variables:

```bash
# Test specific versions [Replace with actual versions]
ONFS_VERSION=v1.2.3 HORIZON_VERSION=v2.1.0 TRUFFLEBOX_VERSION=v1.0.5 ./start.sh

# Or set them in your workspace and run docker-compose directly
cd workspace
ONFS_VERSION=main docker-compose up -d onfs-api-server
```

Available version formats:
- `latest` (default) - Latest stable release
- `main` - Latest development build  
- `v1.2.3` - Specific version tag
- `sha-abcd1234` - Specific commit SHA

This will:
1. Check for Go installation (1.22+ required)
2. Create a workspace directory with configuration files
3. Pull and start all services using `docker-compose up -d`
4. Wait for services to become healthy
5. Initialize databases with required schemas
6. Display access information and helpful commands

Once complete, you can access:
- **Trufflebox UI**: http://localhost:3000
- **Horizon API**: http://localhost:8082
- **Online Feature Store gRPC API**: http://localhost:8089
- **etcd Workbench**: http://localhost:8081

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
- **URL**: http://localhost:3000
- **Default admin credentials**:
  - Email: `admin@admin.com`
  - Password: `admin`

### API Endpoints
- **Horizon API**: http://localhost:8082
  - Health check: http://localhost:8082/health
- **ONFS gRPC API**: http://localhost:8089
  - Health check: http://localhost:8089/health/self

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
  - Workbench: http://localhost:8081 


## Feature Store API Examples

### gRPC API Commands

Use the following `grpcurl` commands to interact with the Online Feature Store gRPC API:

**Persist Features:**
```bash
grpcurl -plaintext -H "online-feature-store-caller-id: <caller-id>" -H "online-feature-store-auth-token: <auth-token>" -d '<request-body>' localhost:8089 persist.FeatureService/PersistFeatures
```

**Retrieve Features (Decoded):**
```bash
grpcurl -plaintext -H "online-feature-store-caller-id: <caller-id>" -H "online-feature-store-auth-token: <auth-token>" -d '<request-body>' localhost:8089 retrieve.FeatureService/RetrieveDecodedResult
```

**Retrieve Features (Binary):**
```bash
grpcurl -plaintext -H "online-feature-store-caller-id: <caller-id>" -H "online-feature-store-auth-token: <auth-token>" -d '<request-body>' localhost:8089 retrieve.FeatureService/RetrieveFeatures
```

### Sample Request Bodies

**Single Feature Group Persist:**
```json
{
    "data": [{
        "key_values": ["10"],
        "feature_values": [{
            "values": {"fp32_values": [123.45]}
        }]
    }],
    "entity_label": "catalog",
    "feature_group_schema": [{
        "label": "int_fg",
        "feature_labels": ["id"]
    }],
    "keys_schema": ["catalog_id"]
}
```

**Single Feature Group Retrieve:**
```json
{
    "entity_label": "catalog",
    "feature_groups": [{
        "label": "int_fg",
        "feature_labels": ["id"]
    }],
    "keys_schema": ["catalog_id"],
    "keys": [{"cols": ["10"]}]
}
```

**Multiple Feature Groups Persist:**
```json
{
    "data": [
        {
            "key_values": ["1"],
            "feature_values": [
                {"values": {"fp32_values": [28.5]}},
                {"values": {"string_values": ["Bharat"]}}
            ]
        },
        {
            "key_values": ["2"],
            "feature_values": [
                {"values": {"fp32_values": [32.0]}},
                {"values": {"string_values": ["India"]}}
            ]
        }
    ],
    "entity_label": "catalog",
    "feature_group_schema": [
        {"label": "int_fg", "feature_labels": ["id"]},
        {"label": "string_fg", "feature_labels": ["name"]}
    ],
    "keys_schema": ["catalog_id"]
}
```

**Multiple Feature Groups Retrieve:**
```json
{
    "entity_label": "catalog",
    "feature_groups": [
        {"label": "int_fg", "feature_labels": ["id"]},
        {"label": "string_fg", "feature_labels": ["name"]}
    ],
    "keys_schema": ["catalog_id"],
    "keys": [
        {"cols": ["1"]},
        {"cols": ["2"]}
    ]
}
```

**Vector Feature Group Persist:**
```json
{
    "data": [{
        "key_values": ["123"],
        "feature_values": [{
            "values": {
                "vector": [{
                    "values": {"fp32_values": [1.0, 2.0, 3.0, 4.0]}
                }]
            }
        }]
    }],
    "entity_label": "catalog",
    "feature_group_schema": [{
        "label": "vector_fg",
        "feature_labels": ["embedding"]
    }],
    "keys_schema": ["catalog_id"]
}
```

**Vector Feature Group Retrieve:**
```json
{
    "entity_label": "catalog",
    "feature_groups": [{
        "label": "vector_fg",
        "feature_labels": ["embedding"]
    }],
    "keys_schema": ["catalog_id"],
    "keys": [{"cols": ["123"]}]
}
```

### Key Points

**Only one type per feature value block:**
- `feature_values` is a list, and each item in the list has only one value type populated
- For example: one item has only `fp32_values`, another has only `int64_values`

**Field Types:**
The following value types are supported:

- **fp32_values**: `float32[]`
- **fp64_values**: `float64[]`
- **int32_values**: `int32[]`
- **int64_values**: `string[]` (because JSON doesn't support 64-bit ints directly)
- **uint32_values**: `uint32[]`
- **uint64_values**: `string[]`
- **string_values**: `string[]`
- **bool_values**: `bool[]`
- **vector**: list of objects with nested values (used for embedded features)

### Response Format Differences

- **Retrieve Features (Binary)**: Returns data in binary format for optimal performance and reduced network overhead
- **Retrieve Features (Decoded)**: Returns data in human-readable string format for easier debugging and development purposes

## Managing Services

### Viewing Logs

```bash
# View logs for all services
cd workspace && docker-compose logs -f

# View logs for specific services
cd workspace && docker-compose logs -f horizon
cd workspace && docker-compose logs -f trufflebox-ui
cd workspace && docker-compose logs -f onfs-api-server
```

### Service Management

```bash
# Restart a specific service
cd workspace && docker-compose restart horizon

# Stop all services
cd workspace && docker-compose down

# Start services again
cd workspace && docker-compose up -d

# Check service status
cd workspace && docker-compose ps
```

## Troubleshooting

### Common Issues

1. **Port conflicts**: Ensure ports 3000, 8081, 8082, 8089, 9042, 3306, 6379, and 2379 are not in use by other applications.

2. **Docker network issues**: If containers can't communicate, try recreating:
   ```bash
   docker network rm onfs-network
   docker network create onfs-network
   ```

3. **Service health checks failing**: Check if all infrastructure services (databases) are running:
   ```bash
   cd workspace && docker-compose ps
   ```

4. **Image pull issues**: Ensure you have access to GitHub Container Registry:
   ```bash
   docker login ghcr.io
   ```
5. [How to use Etcd Workbench ?](https://github.com/tzfun/etcd-workbench/blob/master/README.md)

### Service Dependencies

Services start in the following order:
1. Infrastructure services (ScyllaDB, MySQL, Redis, etcd)
2. Online Feature Store gRPC API Server
3. Horizon (depends on databases + ONFS API)
4. Trufflebox UI (depends on Horizon)

If a service fails to start, check its dependencies are healthy first.

## Development

The workspace directory contains all runtime configuration:

- `workspace/docker-compose.yml` - Complete service orchestration
- `workspace/check_db_and_init.sh` - Database initialization script

You can modify environment variables in the docker-compose.yml file and restart services.

## Contributing

We welcome contributions from the community! Please see our [Contributing Guide](https://github.com/Meesho/BharatMLStack/blob/main/CONTRIBUTING.md) for details on how to get started.

## Community & Support

- 💬 **Discord**: Join our [community chat](https://discord.gg/XkT7XsV2AU)
- 🐛 **Issues**: Report bugs and request features on [GitHub Issues](https://github.com/Meesho/BharatMLStack/issues)
- 📧 **Email**: Contact us at [ml-oss@meesho.com](mailto:ml-oss@meesho.com )

## License

BharatMLStack is open-source software licensed under the [BharatMLStack Business Source License 1.1](https://github.com/Meesho/BharatMLStack/blob/main/LICENSE.md).

---

<div align="center">
  <strong>Built with ❤️ for the ML community from Meesho</strong>
</div>
<div align="center">
  <strong>If you find this useful, ⭐️ the repo — your support means the world to us!</strong>
</div>