![Build Status](https://github.com/Meesho/BharatMLStack/actions/workflows/inferflow.yml/badge.svg)
![Static Badge](https://img.shields.io/badge/release-v1.0.0-blue?style=flat)
[![Discord](https://img.shields.io/badge/Discord-Join%20Chat-7289da?style=flat&logo=discord&logoColor=white)](https://discord.gg/XkT7XsV2AU)

# Inferflow

DAG-based real-time ML inference orchestration service for BharatML Stack.

## What is Inferflow?

Inferflow is the inference layer of BharatML Stack. It receives scoring requests, orchestrates feature retrieval, model execution, and post-processing through a configurable DAG (Directed Acyclic Graph) pipeline, and returns predictions — all in real time.

Each model gets its own DAG of components (feature fetch, model scoring, numeric computation) that execute concurrently where possible, making inference both flexible and fast.

## Features

- ⚡ **DAG-Based Execution** — Components run in parallel when independent; topological ordering ensures correctness
- 🧠 **PointWise, PairWise & SlateWise APIs** — gRPC APIs for per-target, pair-level, and slate-level inference
- 🔄 **Dynamic Model Configuration** — Model DAGs and configs are loaded from etcd and hot-reloaded without restarts
- 📦 **Feature Retrieval** — Fetches real-time features from the Online Feature Store (ONFS) via gRPC
- 🎯 **Model Scoring** — Calls Predator for ML model inference (supports multi-endpoint routing)
- 🔢 **Numeric Computation** — Delegates matrix operations to Numerix via gRPC
- 💾 **In-Memory Caching** — FreeCache-based feature caching for low-latency repeated lookups
- 📊 **Inference Logging** — Asynchronous logging to Kafka in proto, Arrow, or Parquet format
- 🔀 **Multiplexed Server** — gRPC and HTTP served on a single port via cmux

## Architecture

For detailed architecture and data flow diagrams, see the [Inferflow documentation](https://meesho.github.io/BharatMLStack/docs/inferflow).



### External Dependencies

| Dependency | Purpose |
|------------|---------|
| **etcd** | Dynamic model configuration |
| **ONFS** | Real-time feature retrieval |
| **Predator** | ML model inference |
| **Numerix** | Numeric / matrix computation |
| **Kafka** | Inference logging |

## gRPC APIs

### Predict Service (`server/proto/predict.proto`)

| RPC | Description |
|-----|-------------|
| `InferPointWise(PointWiseRequest)` | Per-target scoring — score each target independently |
| `InferPairWise(PairWiseRequest)` | Pair-level scoring — rank/score pairs of targets |
| `InferSlateWise(SlateWiseRequest)` | Slate-level scoring — score ordered groups of targets |

### Legacy Service (`server/proto/inferflow.proto`)

| RPC | Description |
|-----|-------------|
| `RetrieveModelScore(InferflowRequestProto)` | Entity-based model scoring (legacy API) |

### HTTP Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /health/self` | Health check |

For detailed API schemas, see [Predict APIs and Feature Logging](PREDICT_APIS_AND_FEATURE_LOGGING.md).

## 🧰 SDKs

Inferflow provides SDKs to interact with the feature store:

- **[Go SDK](sdks/go/README.md)** - For backend services and ML inference


## Quick Start

### Prerequisites

- Go 1.24 or later
- Docker and Docker Compose (for local development)
- A running BharatML Stack environment (etcd, ONFS, Predator, Numerix)

### Using Docker Compose (Recommended)

The easiest way to run Inferflow with all its dependencies is via the BharatML Stack quick-start:

```bash
cd quick-start
./start.sh
```

This starts Inferflow alongside all required services. See the [Quick Start Guide](../quick-start/README.md) for details.

### Standalone

```bash
cd inferflow

# Set up environment variables
cp cmd/inferflow/application.env .env
# Edit .env with your configuration

# Build
go build -o bin/inferflow cmd/inferflow/main.go

# Run
./bin/inferflow
```

## Configuration

Inferflow is configured via environment variables. Group reference:

### Application

| Variable | Description | Default |
|----------|-------------|---------|
| `APP_ENV` | Environment (dev/staging/prod) | `prod` |
| `APP_NAME` | Application name | `inferflow` |
| `APP_PORT` | Server port (gRPC + HTTP) | `8085` |
| `APP_LOG_LEVEL` | Log level (DEBUG/INFO/ERROR) | `INFO` |
| `APP_GC_PERCENTAGE` | Go GC target percentage | `1` |

### etcd

| Variable | Description | Default |
|----------|-------------|---------|
| `ETCD_SERVER` | etcd server address | `http://etcd:2379` |
| `ETCD_WATCHER_ENABLED` | Enable config hot-reload | `true` |

### Cache

| Variable | Description | Default |
|----------|-------------|---------|
| `IN_MEMORY_CACHE_SIZE_IN_BYTES` | Feature cache size | `6000000000` |
| `DAG_TOPOLOGY_CACHE_SIZE` | DAG topology cache entries | `500` |
| `DAG_TOPOLOGY_CACHE_TTL_SEC` | DAG cache TTL in seconds | `300` |

### Predator (ML Model Serving)

| Variable | Description | Default |
|----------|-------------|---------|
| `EXTERNAL_SERVICE_PREDATOR_PORT` | Predator gRPC port | `8090` |
| `EXTERNAL_SERVICE_PREDATOR_GRPC_PLAIN_TEXT` | Use plaintext gRPC | `true` |
| `EXTERNAL_SERVICE_PREDATOR_CALLER_ID` | Caller identifier | `inferflow` |
| `EXTERNAL_SERVICE_PREDATOR_CALLER_TOKEN` | Auth token | `inferflow` |
| `EXTERNAL_SERVICE_PREDATOR_DEADLINE` | Request deadline (ms) | `200` |

### Numerix (Matrix Operations)

| Variable | Description | Default |
|----------|-------------|---------|
| `NUMERIX_CLIENT_V1_HOST` | Numerix host | `numerix` |
| `NUMERIX_CLIENT_V1_PORT` | Numerix port | `8083` |
| `NUMERIX_CLIENT_V1_DEADLINE_MS` | Request deadline (ms) | `5000` |
| `NUMERIX_CLIENT_V1_PLAINTEXT` | Use plaintext gRPC | `true` |
| `NUMERIX_CLIENT_V1_AUTHTOKEN` | Auth token | `numerix` |
| `NUMERIX_CLIENT_V1_BATCHSIZE` | Batch size | `100` |

### ONFS (Online Feature Store)

| Variable | Description | Default |
|----------|-------------|---------|
| `EXTERNAL_SERVICE_ONFS_FS_HOST` | ONFS host | `onfs-api-server` |
| `EXTERNAL_SERVICE_ONFS_FS_PORT` | ONFS port | `8089` |
| `EXTERNAL_SERVICE_ONFS_FS_GRPC_PLAIN_TEXT` | Use plaintext gRPC | `true` |
| `EXTERNAL_SERVICE_ONFS_FS_CALLER_ID` | Caller identifier | `inferflow` |
| `EXTERNAL_SERVICE_ONFS_FS_CALLER_TOKEN` | Auth token | `inferflow` |
| `EXTERNAL_SERVICE_ONFS_FS_DEAD_LINE` | Request deadline (ms) | `200` |
| `EXTERNAL_SERVICE_ONFS_FS_BATCH_SIZE` | Batch size | `50` |

### Kafka (Inference Logging)

| Variable | Description | Default |
|----------|-------------|---------|
| `KAFKA_BOOTSTRAP_SERVERS` | Kafka broker addresses | `broker:29092` |
| `KAFKA_LOGGING_TOPIC` | Inference log topic | `inferflow_inference_logs` |

## Docker

```bash
# Build
docker build -f cmd/inferflow/Dockerfile -t inferflow:latest .

# Run
docker run -p 8085:8085 --env-file .env inferflow:latest
```

## Documentation

| Version | Link |
|---------|------|
| Latest | [Inferflow Documentation](https://meesho.github.io/BharatMLStack/docs/inferflow) |

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests for your changes
5. Ensure all tests pass (`go test ./...`)
6. Update documentation if needed
7. Submit a pull request

For more details, see [CONTRIBUTING.md](../CONTRIBUTING.md).

## Community & Support

- 💬 **Discord**: Join our [community chat](https://discord.gg/XkT7XsV2AU)
- 🐛 **Issues**: Report bugs and request features on [GitHub Issues](https://github.com/Meesho/BharatMLStack/issues)
- 📧 **Email**: Contact us at [ml-oss@meesho.com](mailto:ml-oss@meesho.com)

## License

BharatMLStack is open-source software licensed under the [BharatMLStack Business Source License 1.1](../LICENSE.md).

---

<div align="center">
  <strong>Built with ❤️ for the ML community from Meesho</strong>
</div>
<div align="center">
  <strong>If you find this useful, ⭐️ the repo — your support means the world to us!</strong>
</div>
