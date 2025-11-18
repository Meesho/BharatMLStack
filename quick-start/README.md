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
- **Apache Kafka**: Message broker for feature ingestion pipeline

**Application Services:**
- **Online Feature Store gRPC API Server**: High-performance gRPC interface (runs on port 8089)
- **ONFS Consumer**: Kafka consumer service for real-time feature ingestion from message streams (runs on port 8090)
- **Horizon**: Backend API service (runs on port 8082)
- **Numerix**: Matrix operations service (runs on port 8083)
- **Trufflebox UI**: Frontend web interface (runs on port 3000)

**Management Tools:**
- **etcd Workbench**: etcd management interface (runs on port 8081)
- **Kafka UI**: Kafka cluster management interface (runs on port 8084)

All services are orchestrated using Docker Compose with pre-built images from GitHub Container Registry (GHCR).

## Quick Start

### Starting the System

The start script provides an interactive service selector that allows you to choose which application services to run:

```bash
./start.sh
```

**Interactive Options:**
1. **All Services** - Starts all application services (API Server, Consumer, Horizon, Numerix, TruffleBox UI)
2. **Custom Selection** - Choose individual services to start
3. **Exit** - Exit without starting

**Infrastructure services (ScyllaDB, MySQL, Redis, etcd, Kafka) and Management Tools (etcd-workbench, kafka-ui) are always started.**

### Service Independence

Services can run independently based on your needs:
- **ONFS API Server** - For direct gRPC feature operations
- **ONFS Consumer** - For real-time Kafka-based feature ingestion (independent of API Server)
- **Horizon + TruffleBox UI** - For web-based feature store management
- **Numerix** - For matrix operations

### Specifying Service Versions

You can specify versions for application services using environment variables:

```bash
# Specify individual service versions
ONFS_VERSION=v1.2.3 ./start.sh

# Specify consumer version
ONFS_CONSUMER_VERSION=v1.0.0-beta-d74137 ./start.sh

# Combine multiple versions
ONFS_VERSION=v1.2.0 HORIZON_VERSION=v2.1.0 TRUFFLEBOX_VERSION=v1.0.5 ./start.sh

# Start all services with specific versions
ONFS_VERSION=v1.2.0 ONFS_CONSUMER_VERSION=v1.0.0-beta-d74137 ./start.sh
```

**Available Environment Variables:**
- `ONFS_VERSION` - Online Feature Store API Server version
- `ONFS_CONSUMER_VERSION` - ONFS Consumer version
- `HORIZON_VERSION` - Horizon Backend version
- `NUMERIX_VERSION` - Numerix Matrix Operations version
- `TRUFFLEBOX_VERSION` - TruffleBox UI version

**Version Formats:**
- `latest` (default) - Latest stable release
- `main` - Latest development build  
- `v1.2.3` - Specific version tag
- `sha-abcd1234` - Specific commit SHA

**Non-interactive Mode:**
```bash
# Start all services without prompts
./start.sh --all

# Start with specific versions non-interactively
ONFS_VERSION=v1.2.0 ./start.sh --all
```

**Advanced: Direct docker-compose Usage**

You can also work directly with docker-compose in the workspace directory:

```bash
# Change to workspace directory
cd workspace

# Start specific service with version
ONFS_VERSION=main docker-compose up -d onfs-api-server

# Start multiple services with different versions
ONFS_VERSION=v1.2.0 HORIZON_VERSION=v2.1.0 docker-compose up -d onfs-api-server horizon

# Restart a service with a different version
ONFS_CONSUMER_VERSION=v1.0.0-beta-d74137 docker-compose up -d onfs-consumer

# View specific service
docker-compose ps onfs-api-server
```

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
- **ONFS Consumer**: http://localhost:8090 (health check)
- **etcd Workbench**: http://localhost:8081
- **Kafka UI**: http://localhost:8084

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
- **ONFS Consumer**: http://localhost:8090
  - Health check: http://localhost:8090/health/self
- **Numerix**: http://localhost:8083
  - Health check: http://localhost:8083/health

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

- **Kafka**:
  - Bootstrap Servers: localhost:9092
  - Kafka UI: http://localhost:8084
  - Default Topic: online-feature-store.feature_ingestion



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

## Feature Ingestion Flows

BharatMLStack Online Feature Store supports two primary methods for ingesting features:

### 1. Direct gRPC API Ingestion

Use the ONFS API Server for synchronous, request-response feature operations:

```bash
# Persist features directly via gRPC
grpcurl -plaintext -H "online-feature-store-caller-id: <caller-id>" \
  -H "online-feature-store-auth-token: <auth-token>" \
  -d '<request-body>' localhost:8089 persist.FeatureService/PersistFeatures
```

**When to use:**
- Real-time feature updates requiring immediate confirmation
- Low-latency synchronous operations
- Direct integration with applications

### 2. Kafka Consumer Ingestion

The ONFS Consumer service provides asynchronous, stream-based feature ingestion:

**Architecture:**
```
Producer → Kafka Topic → ONFS Consumer → Redis/Scylla
```

**How it works:**
1. Applications publish feature data to Kafka topic: `online-feature-store.feature_ingestion`
2. ONFS Consumer reads messages from Kafka in batches
3. Features are persisted to configured storage backends (Redis/Scylla)
4. Consumer handles retries and error scenarios automatically

**Configuration:**
- **Kafka Topic**: `online-feature-store.feature_ingestion`
- **Consumer Port**: 8090
- **Health Check**: http://localhost:8090/health/self
- **Bootstrap Servers**: `broker:29092` (internal), `localhost:9092` (external)

**Publishing Features to Kafka:**
```bash
# Using kafka-console-producer (for testing)
docker exec -it broker kafka-console-producer.sh \
  --bootstrap-server localhost:9092 \
  --topic online-feature-store.feature_ingestion

# Then paste your feature JSON payload
```

**Consumer Benefits:**
- Decouples feature ingestion from application logic
- High throughput batch processing
- Automatic retries and error handling
- Scales independently of API server
- Supports backpressure and flow control

**Monitoring Consumer:**
```bash
# View consumer logs
cd workspace && docker-compose logs -f onfs-consumer

# Check Kafka UI for consumer lag
open http://localhost:8084
```

**When to use:**
- High-volume feature ingestion
- Asynchronous batch processing
- Event-driven architectures
- When producer and consumer need to scale independently

## Managing Services

### Viewing Logs

```bash
# View logs for all services
cd workspace && docker-compose logs -f

# View logs for specific services
cd workspace && docker-compose logs -f horizon
cd workspace && docker-compose logs -f trufflebox-ui
cd workspace && docker-compose logs -f onfs-api-server
cd workspace && docker-compose logs -f onfs-consumer
cd workspace && docker-compose logs -f kafka

# View logs for multiple services
cd workspace && docker-compose logs -f onfs-api-server onfs-consumer
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

1. **Port conflicts**: The following ports must be available:
   
   | Port | Service |
   |------|---------|
   | 3000 | TruffleBox UI |
   | 8081 | etcd Workbench |
   | 8082 | Horizon API |
   | 8083 | Numerix |
   | 8084 | Kafka UI |
   | 8089 | ONFS gRPC API |
   | 8090 | ONFS Consumer |
   | 9092 | Kafka |
   | 9042 | ScyllaDB |
   | 3306 | MySQL |
   | 6379 | Redis |
   | 2379 | etcd |
   
   If any ports are in use, stop conflicting services or modify port mappings in `docker-compose.yml`.

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

5. **Kafka consumer not receiving messages**: 
   ```bash
   # Check if Kafka topic exists
   docker exec -it broker kafka-topics.sh --bootstrap-server localhost:9092 --list
   
   # Check consumer group status
   docker exec -it broker kafka-consumer-groups.sh --bootstrap-server localhost:9092 --group onfs-consumer-group --describe
   
   # Verify Kafka UI
   open http://localhost:8084
   ```

6. **Consumer lag issues**: Monitor consumer performance via Kafka UI (http://localhost:8084) to identify bottlenecks

7. [How to use Etcd Workbench ?](https://github.com/tzfun/etcd-workbench/blob/master/README.md)

### Service Dependencies

Services start in the following order:
1. Infrastructure services (ScyllaDB, MySQL, Redis, etcd, Kafka)
2. kafka-init (creates required Kafka topics)
3. Application services (can run independently):
   - **ONFS API Server** - depends on databases (ScyllaDB, MySQL, Redis, etcd)
   - **ONFS Consumer** - depends on Kafka and databases (independent of API Server)
   - **Horizon** - depends on databases and ScyllaDB
   - **Numerix** - depends on etcd
4. Trufflebox UI (depends on Horizon)

**Key Points:**
- ONFS Consumer can run without ONFS API Server
- Services can be started individually based on use case
- All services depend on their respective infrastructure components

If a service fails to start, check its dependencies are healthy first.

## Development

The workspace directory contains all runtime configuration:

- `workspace/docker-compose.yml` - Complete service orchestration
- `workspace/check_db_and_init.sh` - Database initialization script

You can modify environment variables in the docker-compose.yml file and restart services.

## Contributing

We welcome contributions from the community! Please see our [Contributing Guide](CONTRIBUTING.md) for details on how to get started.

## Community & Support

- 💬 **Discord**: Join our [community chat](https://discord.gg/XkT7XsV2AU)
- 🐛 **Issues**: Report bugs and request features on [GitHub Issues](https://github.com/Meesho/BharatMLStack/issues)
- 📧 **Email**: Contact us at [ml-oss@meesho.com](mailto:ml-oss@meesho.com )

## License

BharatMLStack is open-source software licensed under the [BharatMLStack Business Source License 1.1](LICENSE.md).

---

<div align="center">
  <strong>Built with ❤️ for the ML community from Meesho</strong>
</div>
<div align="center">
  <strong>If you find this useful, ⭐️ the repo — your support means the world to us!</strong>
</div>