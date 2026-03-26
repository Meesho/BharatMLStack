![Build Status](https://github.com/Meesho/BharatMLStack/actions/workflows/skye.yml/badge.svg)
![Static Badge](https://img.shields.io/badge/release-v1.0.0-blue?style=flat)
[![Discord](https://img.shields.io/badge/Discord-Join%20Chat-7289da?style=flat&logo=discord&logoColor=white)](https://discord.gg/XkT7XsV2AU)

# Skye

Vector similarity search platform for BharatML Stack.

Skye enables fast semantic retrieval by representing data as vectors and querying nearest matches in high-dimensional space. It is composed of three runnable components: **skye-admin**, **skye-consumers**, and **skye-serving**.

## ✨ Features

- 🔌 **Pluggable Vector Databases** — Support for multiple vector DB backends (Qdrant, NGT, Eigenix) via a generic abstraction layer
- 🏗️ **Shared Embeddings, Isolated Indexes** — Models are stored once but serve multiple tenants (variants), reducing data redundancy
- ⚡ **Event-Driven Administration** — Model lifecycle management through Kafka-based event flows for resilience and fault tolerance
- 💾 **Multi-Layer Caching** — In-memory (FreeCache) + distributed (Redis) caching for ultra-low-latency serving
- 🔍 **Similarity Search APIs** — gRPC APIs for similar-candidate search, bulk embedding retrieval, and dot-product computation
- 🔄 **Real-Time + Batch Ingestion** — Kafka consumers for both reset/delta batch jobs and real-time embedding updates
- 🎯 **Configurable Distance Functions** — DOT, Cosine, and Euclidean distance support
- 🛡️ **Resilience** — Circuit breakers, retry topics, and snapshot-based recovery

## 🏗️ Architecture

Skye is built around three components:

| Component | Role |
|-----------|------|
| **skye-serving** | Handles real-time similarity search queries with in-memory caching and vector DB lookups (gRPC, port 9090) |
| **skye-consumers** | Processes embedding ingestion (reset/delta jobs) and real-time aggregation events from Kafka (HTTP, port 8080) |
| **skye-admin** | Manages model lifecycle, onboarding, variant registration, and coordinates jobs (HTTP, port 8080) |

For detailed architecture and data flow diagrams, see the [Skye documentation](https://meesho.github.io/BharatMLStack/category/skye).

### External Dependencies

| Dependency | Purpose |
|------------|---------|
| **etcd** | Dynamic model/variant configuration |
| **Kafka** | Embedding ingestion events, model state machine |
| **Qdrant** | Vector database (pluggable) |
| **ScyllaDB** | Embedding storage + aggregator data |
| **Redis** | Distributed caching |

## 📡 gRPC APIs (skye-serving)

### Similar Candidate Service

| RPC | Description |
|-----|-------------|
| `GetSimilarCandidates(SkyeRequest)` | Find similar candidates using embeddings or candidate IDs |

### Embedding Service

| RPC | Description |
|-----|-------------|
| `GetEmbeddingsForCandidates(SkyeBulkEmbeddingRequest)` | Bulk embedding retrieval for candidate IDs |
| `GetDotProductOfCandidatesForEmbedding(EmbeddingDotProductRequest)` | Compute dot products between an embedding and candidates |

### HTTP Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /health` | Health check |

## 🔧 Admin HTTP APIs (skye-admin)

### Model Management

| Endpoint | Description |
|----------|-------------|
| `POST /api/v1/model/register-model` | Register a new model |
| `POST /api/v1/model/register-variant` | Register a variant for a model |
| `POST /api/v1/model/register-store` | Register a storage store |
| `POST /api/v1/model/register-frequency` | Register job frequency |
| `POST /api/v1/model/register-entity` | Register an entity type |

### Qdrant Operations

| Endpoint | Description |
|----------|-------------|
| `POST /api/v1/qdrant/create-collection` | Create a Qdrant collection |
| `POST /api/v1/qdrant/process-model` | Process a model (reset) |
| `POST /api/v1/qdrant/process-multi-variant` | Process multiple variants |
| `POST /api/v1/qdrant/promote-variant` | Promote variant to scale-up cluster |
| `POST /api/v1/qdrant/trigger-indexing` | Trigger indexing pipeline |

## 🧰 SDKs

- **[Go SDK](../go-sdk/pkg/clients/skye/)** — Client library for backend services

## 🚀 Quick Start

### Prerequisites

- Go 1.24 or later
- Docker and Docker Compose (for local development)
- `librdkafka-dev` (CGO dependency for Kafka client)
- A running BharatML Stack environment (etcd, Kafka, Qdrant, ScyllaDB, Redis)

### Using Docker Compose (Recommended)

The easiest way to run Skye with all its dependencies is via the BharatML Stack quick-start:

```bash
cd quick-start
./start.sh
```

This starts Skye alongside all required services. See the [Quick Start Guide](../quick-start/README.md) for details.

### Standalone

```bash
cd skye

# Build all components
go build -o bin/skye-admin ./cmd/admin
go build -o bin/skye-consumers ./cmd/consumers
go build -o bin/skye-serving ./cmd/serving

# Run (example: serving)
./bin/skye-serving
```

## ⚙️ Configuration

Skye is configured via environment variables (loaded through Viper). Dynamic model/variant configuration is managed via etcd.

### Application

| Variable | Description |
|----------|-------------|
| `app_name` | Application name |
| `app_env` | Environment (staging/production) |
| `port` | HTTP/gRPC server port |
| `auth_tokens` | Authentication tokens |

### etcd

| Variable | Description |
|----------|-------------|
| `etcd_server` | etcd server address |
| `etcd_username` | etcd username |
| `etcd_password` | etcd password |
| `etcd_watcher_enabled` | Enable config hot-reload |

### Kafka

| Variable | Description |
|----------|-------------|
| `kafka_broker` | Kafka broker address |
| `kafka_group_id` | Kafka consumer group ID |
| `kafka_topic` | Kafka topic |
| `embedding_consumer_kafka_ids` | Comma-separated embedding consumer IDs |
| `realtime_consumer_kafka_ids` | Comma-separated real-time consumer IDs |
| `realtime_producer_kafka_id` | Real-time producer ID |

### Redis

| Variable | Description |
|----------|-------------|
| `redis_addr` | Redis server address |
| `redis_password` | Redis password |
| `redis_db` | Redis database number |

### Storage

| Variable | Description |
|----------|-------------|
| `storage_aggregator_db_count` | Number of aggregator database connections |
| `storage_embedding_store_count` | Number of embedding store connections |

## 🐳 Docker

```bash
cd skye

# Build images
docker build -f cmd/admin/Dockerfile -t skye-admin:latest .
docker build -f cmd/consumers/Dockerfile -t skye-consumers:latest .
docker build -f cmd/serving/Dockerfile -t skye-serving:latest .

# Run (example: serving)
docker run -p 9090:9090 --env-file .env skye-serving:latest

# Run (example: admin)
docker run -p 8080:8080 --env-file .env skye-admin:latest
```

## 📚 Documentation

| Version | Link |
|---------|------|
| v1.0.0 | [Skye Documentation](https://meesho.github.io/BharatMLStack/category/skye) |

## 🤝 Contributing

Contributions are welcome! Please check our [Contribution Guide](../CONTRIBUTING.md) for details on how to get started.

We encourage you to:
- Join our [Discord community](https://discord.gg/XkT7XsV2AU) to discuss features, ideas, and questions
- Check existing issues before opening a new one
- Follow our coding guidelines and pull request process
- Participate in code reviews and discussions

## Community & Support

- 💬 **Discord**: Join our [community chat](https://discord.gg/XkT7XsV2AU)
- 🐛 **Issues**: Report bugs and request features on [GitHub Issues](https://github.com/Meesho/BharatMLStack/issues)
- 📧 **Email**: Contact us at [ml-oss@meesho.com](mailto:ml-oss@meesho.com)

## License

BharatMLStack is open-source software licensed under the [BharatMLStack Business Source License 1.1](LICENSE.md).

---

<div align="center">
  <strong>Built with ❤️ for the ML community from Meesho</strong>
</div>
<div align="center">
  <strong>If you find this useful, ⭐️ the repo — your support means the world to us!</strong>
</div>
