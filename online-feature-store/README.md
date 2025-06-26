<p align="center">
  <img src="assets/logov2.webp" alt="Online-feature-store" width="500"/>
</p>

----
# BharatMLStack's Online Feature Store: A Hyper-Scalable Feature Store for Real-Time ML
 
![build status](https://github.com/Meesho/BharatMLStack/actions/workflows/online-feature-store.yml/badge.svg)
![Static Badge](https://img.shields.io/badge/release-v1.0.0-blue?style=flat)
[![Discord](https://img.shields.io/badge/Discord-Join%20Chat-7289da?style=flat&logo=discord&logoColor=white)](https://discord.gg/XkT7XsV2AU)

Online-feature-store is a high-performance, scalable, and production-grade feature store built for modern machine learning systems. It supports both **real-time** and **batch** workflows, with a strong emphasis on **developer experience**, **system observability**, and **low-latency feature retrieval**.

Designed for high-scale real-time inference workloads, Online-feature-store ensures:

- ⚡️ **Ultra-low latency** - Ultra-low retrieval times at scale
- 🚀 **High throughput** - Tested to serve millions of QPS with hundreds of entity IDs per request
- 🛡️ **High availability** - Designed for mission-critical ML systems with multi-layer storage
- 🧠 **Multi-format support** - Scalar, vector/embedding, and quantized feature types
- 📈 **Comprehensive observability** - Full metrics with Prometheus and Grafana
- 🔄 **Hybrid ingestion** - Seamless batch and real-time feature updates

## 🏗️ Architecture

Online-feature-store consists of several key components working together:

- **Feature Generation** - Ingest features from existing offline sources or custom feature pipelines
- **Kafka Layer** - Buffered streaming ingestion with flow control
- **Horizon Control Plane** - Configuration management with TruffleBox UI
- **Online-feature-store Consumer** - Ingestion workhorse for writes into online stores
- **Online-feature-store gRPC API Server** - Low-latency feature retrieval and high-consistency ingestion
- **Storage Backends** - Redis/Dragonfly for caching and ScyllaDB for persistence

![Online-feature-store Architecture](../docs-src/static/img/v1.0.0-onfs-arch.png)

## 🚀 Quick Start

For detailed setup instructions, see the [**Quick Start Guide**](quick-start/README.md).

The quickest way to get started with Online-feature-store is to use the quick start scripts with **flexible service selection**:

```bash
# Clone the repository
git clone https://github.com/Meesho/BharatMLStack.git
cd BharatMLStack/online-feature-store/quick-start

# Interactive mode - choose which application services to start
./start.sh

# Or start all services (non-interactive)
./start.sh --all
```

### 🎯 Service Selection Options:

**Infrastructure is always started** (ScyllaDB, MySQL, Redis, etcd, etcd-workbench)

Choose which **application services** to start:

1. **🚀 All Services**
   - Online Feature Store + Horizon + TruffleBox UI
2. **🎛️ Custom Selection**
   - Choose individual application services (ONFS API, Horizon, TruffleBox UI)

To stop all services:
```bash
./stop.sh
```

## 🧰 SDKs

Online-feature-store provides SDKs to interact with the feature store:

- **[Go SDK](sdks/go/README.md)** - For backend services and ML inference
- **[Python SDK](sdks/python/README.md)** - For feature ingestion and Spark jobs


## 📊 Use Cases

Online-feature-store is ideal for:

- **Real-time inference** - Low-latency feature serving for online ML models, like Ranking and Recommendation, Fraud detection, Search, Dynamic pricing and bidding, real-time notification etc real time 
- **Feature Catalogue** - Centralized feature registry for organizations
- **Flexible Data Ingestion** - Support for batch processing, streaming pipelines, and real-time data sources with unified ingestion APIs


## 📚 Documentation

| Version |  Link |
|---------|-------------------|
| v1.0.0  | [Documentation](https://meesho.github.io/BharatMLStack/online-feature-store/v1.0.0) |
| v1.0.0  | [User Guide](https://meesho.github.io/BharatMLStack/trufflebox-ui/v1.0.0/userguide) |

## Development

### Prerequisites

- Go 1.22 or later
- Database (PostgreSQL/MySQL)
- Docker (optional, for containerization)
- Start the infrastructure using quick-start interactive startup

### Getting Started

```bash
# Clone and navigate to the project
cd online-feature-store

# Install dependencies
go mod download

# Set up environment variables
# For api-server
cp env-api-server.example ./cmd/api-server/.env

# For consumer
cp env-consumer.example ./cmd/consumer/.env
# Edit .env with your configuration

# Run tests
go test -v ./...

# Build the application
go build -v ./cmd/api-server
go build -v ./cmd/consumer

# Run the service
# Run api-server
bash -c 'set -a; source ./cmd/api-server/.env; set +a; exec go run ./cmd/api-server'

# Run consumer
bash -c 'set -a; source ./cmd/consumer/.env; set +a; exec go run ./cmd/consumer'
```

### Configuration

Create a `.env` file or set environment variables. Configuration is organized into **common settings** (shared between api-server and consumer) and **service-specific settings**.

## Common Configuration (api-server + consumer)
These settings must be **identical** across both api-server and consumer for proper operation:

```bash
# APPLICATION CONFIGURATION - Basic app settings
APP_ENV=prod
APP_NAME=onfs
APP_METRIC_SAMPLING_RATE=1

# ETCD CONFIGURATION - Distributed configuration management
ETCD_SERVER=127.0.0.1:2379
ETCD_WATCHER_ENABLED=true

# REDIS FAILOVER CONFIGURATION - Distributed caching or redis storage layer
# Can be used for either distributed caching or as a redis storage layer
STORAGE_REDIS_FAILOVER_2_SENTINEL_ADDRESSES=localhost:26379
STORAGE_REDIS_FAILOVER_2_DB=0
STORAGE_REDIS_FAILOVER_2_DISABLE_IDENTITY=true
STORAGE_REDIS_FAILOVER_2_MASTER_NAME=mymaster
STORAGE_REDIS_FAILOVER_2_MAX_IDLE_CONN=32
STORAGE_REDIS_FAILOVER_2_MIN_IDLE_CONN=20
STORAGE_REDIS_FAILOVER_2_MAX_ACTIVE_CONN=32
STORAGE_REDIS_FAILOVER_2_MAX_RETRY=-1
STORAGE_REDIS_FAILOVER_2_POOL_FIFO=false
STORAGE_REDIS_FAILOVER_2_READ_TIMEOUT_IN_MS=3000
STORAGE_REDIS_FAILOVER_2_WRITE_TIMEOUT_IN_MS=3000
STORAGE_REDIS_FAILOVER_2_POOL_TIMEOUT_IN_MS=3000
STORAGE_REDIS_FAILOVER_2_POOL_SIZE=32
STORAGE_REDIS_FAILOVER_2_CONN_MAX_IDLE_TIMEOUT_IN_MINUTES=15
STORAGE_REDIS_FAILOVER_2_CONN_MAX_AGE_IN_MINUTES=30

# SCYLLA DATABASE CONFIGURATION - Primary persistent storage
STORAGE_SCYLLA_1_CONTACT_POINTS=127.0.01
STORAGE_SCYLLA_1_KEYSPACE=onfs
STORAGE_SCYLLA_1_NUM_CONNS=1
STORAGE_SCYLLA_1_PORT=9042
STORAGE_SCYLLA_1_TIMEOUT_IN_MS=300000
STORAGE_SCYLLA_1_PASSWORD=
STORAGE_SCYLLA_1_USERNAME=

# ACTIVE STORAGE CONFIGURATION - Which storage backends to use
STORAGE_REDIS_FAILOVER_ACTIVE_CONFIG_IDS=2
STORAGE_SCYLLA_ACTIVE_CONFIG_IDS=1
```

## API-Server Specific Configuration
Additional settings required only for the api-server:

```bash
# Service-specific settings
APP_LOG_LEVEL=DEBUG
APP_PORT=8089

# IN-MEMORY CACHE CONFIGURATION - Local caching layer
IN_MEM_CACHE_3_ENABLED=true
IN_MEM_CACHE_3_NAME=onfs
IN_MEM_CACHE_3_SIZE_IN_BYTES=100
IN_MEM_CACHE_ACTIVE_CONFIG_IDS=3

# KUBERNETES/DEPLOYMENT CONFIGURATION - Pod and node identification
POD_IP=127.0.0.1
NODE_IP=127.0.0.1
```

## Consumer Specific Configuration
Additional settings required only for the consumer:

```bash
# Service-specific settings
APP_LOG_LEVEL=ERROR
APP_PORT=8090

# KAFKA CONSUMER CONFIGURATION - Feature ingestion from Kafka streams
KAFKA_CONSUMERS_FEATURE_CONSUMER_AUTO_COMMIT_INTERVAL_MS=5000
KAFKA_CONSUMERS_FEATURE_CONSUMER_AUTO_OFFSET_RESET=latest
KAFKA_CONSUMERS_FEATURE_CONSUMER_BASIC_AUTH_CREDENTIAL_SOURCE=USER_INFO
KAFKA_CONSUMERS_FEATURE_CONSUMER_BATCH_SIZE=100
KAFKA_CONSUMERS_FEATURE_CONSUMER_BOOTSTRAP_SERVERS=lkc-r0yp97.domqgq513gn.asia-southeast1.gcp.confluent.cloud:9092
KAFKA_CONSUMERS_FEATURE_CONSUMER_CLIENT_ID=online-feature-store-consumer
KAFKA_CONSUMERS_FEATURE_CONSUMER_ENABLE_AUTO_COMMIT=true
KAFKA_CONSUMERS_FEATURE_CONSUMER_GROUP_ID=online-feature-store-consumer
KAFKA_CONSUMERS_FEATURE_CONSUMER_LISTENER_CONCURRENCY=2
KAFKA_CONSUMERS_FEATURE_CONSUMER_MAX_WORKERS=50
KAFKA_CONSUMERS_FEATURE_CONSUMER_POLL_TIMEOUT=1000
KAFKA_CONSUMERS_FEATURE_CONSUMER_SASL_MECHANISM=PLAIN
KAFKA_CONSUMERS_FEATURE_CONSUMER_SASL_PASSWORD=ssl_password
KAFKA_CONSUMERS_FEATURE_CONSUMER_SASL_USERNAME=sasl_user_name
KAFKA_CONSUMERS_FEATURE_CONSUMER_SECURITY_PROTOCOL=SASL_SSL
KAFKA_CONSUMERS_FEATURE_CONSUMER_TOPIC=online-feature-store.feature_ingestion
```

# For Consumer:
Before running the consumer service, you'll need to set up a Kafka environment. You have two options:

### Testing

```bash
# Clone and navigate to the project
cd online-feature-store

# Run all tests
go test -v ./...

# Run tests with coverage
go test -v -cover ./...

# Run specific test package
go test -v ./pkg/config/...

# Run integration tests
go test -v -tags=integration ./...
```

### Building - api-server

```bash
# Clone and navigate to the project
cd online-feature-store

# Build for current platform
go build -v ./cmd/api-server

# Build for production
make build

# Build for specific platform
GOOS=linux GOARCH=amd64 go build -v ./cmd/api-server
```

### Building - consumer

```bash
# Clone and navigate to the project
cd online-feature-store

# Build for current platform
go build -v ./cmd/consumer

# Build for production
make build

# Build for specific platform
GOOS=linux GOARCH=amd64 go build -v ./cmd/consumer
```

## Docker

### Building the Docker Image

```bash
# Clone and navigate to the project
cd online-feature-store

# Build Docker image - api-server
docker build -t onfs-api-server -f cmd/api-server/DockerFile .

# Build Docker image - consumer
docker build -t onfs-consumer -f cmd/consumer/DockerFile .

# Run container with environment variables
# For api-server
docker run -p 8080:8080 \
   --env-file ./cmd/api-server/.env \
  onfs-api-server
  
# For consumer
docker run -p 8080:8080 \
   --env-file ./cmd/consumer/.env \
  onfs-consumer
```

## Deployment

### Production with Docker

```bash
# Clone and navigate to the project
cd online-feature-store

# Build production image - api-server
docker build -t onfs-api-server:latest -f cmd/api-server/DockerFile .

# Build production image - consumer
docker build -t onfs-consumer:latest -f cmd/consumer/DockerFile .

# Run in production mode
# For API Server
docker run -d \
  --name onfs-api-server \
  -p 8080:8080 \
  --env-file ./cmd/api-server/.env \
  onfs-api-server:latest

# For Consumer
docker run -d \
  --name onfs-consumer \
  -p 8090:8090 \
  --env-file ./cmd/api-server/.env \
  onfs-consumer:latest
```

### Observability

Use [Grafana](./grafana.json) to visualize the monitoring data. The `json` file can be directly imported to Grafana.

## 🤝 Contributing

Contributions are welcome! Please check our [Contribution Guide](../CONTRIBUTING.md) for details on how to get started.

We encourage you to:
- Join our [Discord community](https://discord.gg/XkT7XsV2AU) to discuss features, ideas, and questions
- Check existing issues before opening a new one
- Follow our coding guidelines and pull request process
- Participate in code reviews and discussions

## 📫 Need Help?

There are several ways to get help with Online-feature-store:

- Join the [Online-feature-store Discord community](https://discord.gg/XkT7XsV2AU) for questions and discussions
- Open an issue in the repository for bug reports or feature requests
- Check the documentation for guides and examples
- Reach out to the Online-feature-store core team

Feedback and contributions are welcome!

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
