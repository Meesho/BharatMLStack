---
title: Release Notes
sidebar_position: 4
---

# Inferflow - Release Notes

## Version 1.0.0
**Release Date**: June 2025
**Status**: General Availability (GA)

We're excited to announce the first stable release of **Inferflow** — a graph-driven feature retrieval and model inference orchestration engine, part of BharatMLStack.

---

## What's New

### Config-Driven DAG Executor
- **No-code feature retrieval**: Onboard new models with config changes only — no custom code required
- **DAG topology execution**: Define component dependency graphs that are executed concurrently using Kahn's algorithm
- **Hot reload**: Model configurations stored in etcd are watched and reloaded live — no redeployment needed
- **DAG caching**: Topologies are cached using Murmur3 hashing with Ristretto for minimal overhead

### Multi-Pattern Inference APIs
Three structured inference patterns via the Predict API:

| API | Pattern | Use Case |
|-----|---------|----------|
| `InferPointWise` | Score each target independently | CTR prediction, fraud scoring |
| `InferPairWise` | Score pairs of targets | Preference learning, comparison ranking |
| `InferSlateWise` | Score groups of targets together | Whole-page optimization, diversity-aware ranking |

Plus the entity-based `RetrieveModelScore` API for direct feature retrieval and scoring.

### Component System
Four built-in component types:
- **FeatureInitComponent** — Initializes the shared ComponentMatrix
- **FeatureComponent** — Fetches features from the Online Feature Store (OnFS)
- **PredatorComponent** — Calls model serving endpoints with percentage-based traffic routing
- **NumerixComponent** — Calls compute engine for operations like reranking

### Online Feature Store Integration
- gRPC-based feature retrieval via `FeatureService.RetrieveFeatures`
- Batched retrieval with configurable batch size and deadline
- Token-based authentication
- Dynamic key resolution from the ComponentMatrix

### In-Memory Feature Caching
- Optional per-component caching to reduce OnFS load
- Configurable TTL per component
- Zero-GC-overhead cache option (freecache)
- Cache hit/miss metrics

### Inference Logging
- Async logging to Kafka for model monitoring and debugging
- Three serialization formats: **Proto**, **Arrow**, **Parquet**
- Configurable sampling rate and feature selection
- Batched log message grouping

---

## Performance

### Built in Go
Inferflow is written entirely in Go, delivering:
- ~80% lower memory usage compared to equivalent Java services
- Lower CPU utilization
- Faster, more efficient deployments

### Concurrency
- DAG components at the same level execute concurrently in goroutines
- Feature retrieval parallelized across entity types
- Connection pooling for all external gRPC calls

### Serialization
- gRPC with Proto3 for all APIs
- Binary feature encoding in the ComponentMatrix
- Configurable compression for Kafka logging (ZSTD support)

---

## APIs & Protocols

### gRPC API

**Inferflow Service:**
```protobuf
service Inferflow {
    rpc RetrieveModelScore(InferflowRequestProto) returns (InferflowResponseProto);
}
```

**Predict Service:**
```protobuf
service PredictService {
    rpc InferPointWise(PredictRequest) returns (PredictResponse);
    rpc InferPairWise(PredictRequest) returns (PredictResponse);
    rpc InferSlateWise(PredictRequest) returns (PredictResponse);
}
```

### Data Types Supported

| Type | Variants |
|------|----------|
| Integers | int8, int16, int32, int64 |
| Floats | float8 (e4m3, e5m2), float16, float32, float64 |
| Strings | Variable length |
| Booleans | Bit-packed |
| Vectors | All scalar types |

---

## Enterprise Features

### Production Readiness
- **Health checks**: HTTP health endpoints via cmux
- **Graceful shutdown**: Clean resource cleanup
- **Structured logging**: JSON-formatted logs via zerolog
- **Signal handling**: SIGTERM/SIGINT support for container environments

### Monitoring & Observability
- **StatsD / Telegraf integration**: Request rates, latencies, error rates
- **Per-component metrics**: Execution time, feature counts, cache hit rates
- **External API metrics**: OnFS, Predator, Numerix call tracking
- **Kafka logging metrics**: Messages sent, errors

### Configuration Management
- **etcd-based**: All model configs stored in etcd
- **Watch & reload**: Live config updates without restart
- **Multi-model support**: Multiple `model_config_id` entries served concurrently

---

## Deployment

### Container Support
- **Docker image**: Multi-stage build (Go Alpine builder + Debian runtime)
- **Optional Kafka**: librdkafka support via build flag
- **Static binary**: Single binary deployment

### Supported Environments
- Kubernetes (K8s)
- Google Kubernetes Engine (GKE)
- Amazon EKS

---

## Compatibility

### Supported Go Versions
- **Minimum**: Go 1.19
- **Recommended**: Go 1.24+

### External Dependencies

| Service | Version | Protocol |
|---------|---------|----------|
| etcd | 3.5+ | gRPC |
| Online Feature Store (OnFS) | 1.0+ | gRPC |
| Predator (Helix) | 1.0+ | gRPC |
| Numerix | 1.0+ | gRPC |
| Kafka | 2.0+ | TCP |

---

## Download & Installation

### Source Code
```bash
git clone https://github.com/Meesho/BharatMLStack.git
cd BharatMLStack/inferflow
```

### Build
```bash
go build -o inferflow-server cmd/inferflow/main.go
```

### Docker
```bash
docker build -t inferflow:latest .
```

---

## Contributing

We welcome contributions from the community! Please see our [Contributing Guide](https://github.com/Meesho/BharatMLStack/blob/main/CONTRIBUTING.md) for details on how to get started.

## Community & Support

- **Discord**: Join our [community chat](https://discord.gg/XkT7XsV2AU)
- **Issues**: Report bugs and request features on [GitHub Issues](https://github.com/Meesho/BharatMLStack/issues)
- **Email**: Contact us at [ml-oss@meesho.com](mailto:ml-oss@meesho.com)

## License

BharatMLStack is open-source software licensed under the [BharatMLStack Business Source License 1.1](https://github.com/Meesho/BharatMLStack/blob/main/LICENSE.md).

---

<div align="center">
  <strong>Built with ❤️ for the ML community from Meesho</strong>
</div>
<div align="center">
  <strong>If you find this useful, ⭐️ the repo — your support means the world to us!</strong>
</div>
