---
title: Release Notes
sidebar_position: 5
---

# Online Feature Store - Release Notes

## Version 1.0.0 🚀
**Release Date**: June 2025  
**Status**: General Availability (GA)

We're excited to announce the first stable release of the **BharatML Online Feature Store** - a high-performance, production-ready feature serving system designed for machine learning workloads.

---

## 🎯 **What's New**

### **Core Feature Store Engine**
- **Ultra-Low Latency**: Achieve sub-10ms P99 response times for real-time inference
- **High Throughput**: Tested and validated at 1M+ requests per second with 100 IDs per request
- **Multi-Entity Support**: Serve features for multiple entity types (users, transactions, products, etc.)
- **Batch Retrieval**: Efficient bulk feature fetching for training and inference workloads

### **Advanced Data Type Support**
Complete support for all ML-relevant data types:

| Data Type | Variants | Optimizations |
|-----------|----------|---------------|
| **Integers** | int8, int16, int32, int64 | Varint encoding, bit packing |
| **Floats** | float16, float32, float64 | IEEE 754 compliant storage |
| **Strings** | Variable length | Pascal string encoding |
| **Booleans** | Bit-packed | 8x memory compression |
| **Vectors** | All above types | Contiguous memory layout |

### **Multi-Database Architecture**
Flexible backend storage with optimized drivers:
- **🔥 Scylla DB**: Ultra-high performance NoSQL (recommended for production)
- **⚡ Dragonfly**: Modern Redis alternative with better memory efficiency
- **📊 Redis**: Standard in-memory store for development environments

## 🚀 **Performance & Optimization**

### **PSDB v2 Serialization Format**
Our proprietary **Permanent Storage Data Block** format delivers:
- **35% faster** serialization than Protocol Buffers
- **100.0-102.2%** size efficiency (near raw data size)
- **93% fewer allocations** than Apache Arrow (4 vs 66 allocs/op)
- **975 MB/s** throughput capacity

### **Memory Management**
- **Object Pooling**: Zero-allocation feature retrieval with PSDBPool
- **Connection Pooling**: Optimized database connection reuse
- **Buffer Management**: Pre-allocated buffers for serialization operations
- **Smart Caching**: Configurable TTL-based feature caching

### **Compression Support**
Intelligent compression with multiple algorithms:
- **LZ4**: Ultra-fast compression for latency-critical workloads
- **Snappy**: Balanced compression ratio and speed
- **ZSTD**: Maximum compression for bandwidth-constrained environments
- **Auto-Fallback**: Intelligent selection based on data characteristics

## 🛠️ **APIs & SDKs**

### **gRPC API**
High-performance, language-agnostic interface:
```protobuf
service FeatureStoreService {
    rpc RetrieveFeatures(Query) returns (QueryResult);
    rpc RetrieveDecodedFeatures(Query) returns (DecodedQueryResult);
    rpc PersistFeatures(PersistFeaturesRequest) returns (Result);
}
```

### **Go SDK v1.0.0**
Native Go client with enterprise features:
- **Type-Safe API**: Strongly typed interfaces and data structures
- **Connection Management**: Configurable timeouts, TLS, and pooling
- **Batch Processing**: Configurable batch sizes for bulk operations
- **Metrics Integration**: Built-in timing and count metrics
- **Authentication**: Caller ID and token-based security

### **Python SDK Collection v1.0.0**
Three specialized Python packages for different ML workflows:

**bharatml_commons** - Common utilities and protobuf definitions:
- **HTTP Client**: Feature metadata operations
- **Protobuf Support**: Generated Python definitions for all APIs
- **Utility Functions**: Column cleaning and feature processing

**spark_feature_push_client** - Apache Spark-based data pipeline:
- **Batch ETL**: Large-scale data processing with Spark
- **Kafka Integration**: Protobuf serialization and Kafka publishing
- **Multi-Source Support**: Hive, Delta, Parquet, Cloud Storage

**grpc_feature_client** - High-performance gRPC client:
- **Real-time Operations**: Direct persist/retrieve API access
- **Low Latency**: Optimized for model inference workflows
- **Type Safety**: Strongly typed Python interfaces

### **RESTful Interface**
HTTP API for web applications:
- **Health Endpoints**: Built-in monitoring and status checks

## 🔧 **Enterprise Features**

### **Production Readiness**
- **Health Checks**: `/health` and `/ready` endpoints for orchestration
- **Graceful Shutdown**: Clean resource cleanup with configurable timeouts
- **Structured Logging**: Formatted logs with configurable levels
- **Signal Handling**: SIGTERM/SIGINT support for container environments

### **Monitoring & Observability**
- **DataDog Integration**: Built-in metrics collection and reporting
- **Prometheus Compatibility**: Standard metrics format support
- **Custom Metrics**: Request rates, latencies, error rates, and business metrics
- **Distributed Tracing**: Request flow visibility across services

### **Data Management**
- **TTL Support**: Automatic feature expiration
- **Feature Versioning**: Schema evolution with backward compatibility
- **Bulk Operations**: Efficient batch read/write with configurable sizes

## 🏗️ **Deployment & Configuration**

### **Container Support**
- **Docker Images**: Multi-architecture support (amd64, arm64)
- **Kubernetes Manifests**: Production-ready deployment templates
- **Helm Charts**: Parameterized installation with best practices
- **Resource Requirements**: Optimized CPU and memory specifications


## 📊 **Benchmarks & Performance**

### **Serialization Performance**
Comprehensive benchmarking against industry standards:

| Format | Speed (ns/op) | Size Efficiency | Memory (allocs/op) | Throughput (MB/s) |
|--------|---------------|-----------------|-------------------|-------------------|
| **PSDB v2** | **359,932** | **100.0%** | **4** | **975.31** |
| Protocol Buffers | 556,541 | 121.8% | 2 | 666.12 |
| Apache Arrow | 118,489* | 100.1% | 66 | 768.25 |

*Arrow performs better at 100k+ features due to amortized setup costs

### **Scale Testing Results**
- **Latency**: P50: 2.1ms, P95: 6.7ms, P99: 9.8ms
- **Throughput**: 1.2M RPS sustained with 100 concurrent features per request
- **Memory**: 512MB baseline, linear scaling with dataset size
- **Storage**: 99.9% compression efficiency vs raw data

## 🔄 **Compatibility**

### **Supported Go Versions**
- **Minimum**: Go 1.22.0
- **Recommended**: Go 1.22.8+
- **Testing**: Automated CI/CD across Go 1.22, 1.23

### **Database Compatibility**
| Database | Version | Status | Notes |
|----------|---------|--------|-------|
| Scylla DB | 5.0+ | ✅ Recommended | Optimal performance |
| Dragonfly | 1.0+ | ✅ Supported | Memory efficient |
| Redis | 6.0+ | ✅ Development | Limited scale |

### **Protocol Compatibility**
- **gRPC**: Protocol Buffers v3 compatible
- **HTTP**: REST API with OpenAPI 3.0 specification
- **Serialization**: Forward/backward compatible PSDB format

## 🐛 **Known Issues**

### **Current Limitations**
1. **Large Vector Support**: Vectors >10MB may experience increased latency

### **Workarounds**
1. **Vector Chunking**: Split large vectors into smaller segments

## 🔜 **What's Next**

### **Upcoming Features (v1.1.0)**
- **Stream Processing**: Real-time feature updates via Kafka/Pulsar
- **Feature Store UI**: Web-based management console
- **Advanced Analytics**: Built-in feature drift detection
- **Multi-Region**: Cross-region replication and failover

### **Performance Roadmap**
- **Vector Search**: Approximate nearest neighbor search
- **GPU Acceleration**: CUDA support for large-scale operations
- **Edge Deployment**: ARM-optimized builds for edge computing

## 💾 **Download & Installation**

### **Container Images**
```bash
# Pull the latest image
docker pull ghcr.io/meesho/onfs-api-server:latest
docker pull ghcr.io/meesho/onfs-consumer:latest

```

### **Arch Supported**
- **Linux (amd64)**
- **Linux (arm64)**
- **macOS (Intel)**
- **macOS (Apple Silicon)**

Checkout [Packages](https://github.com/orgs/Meesho/packages?repo_name=BharatMLStack) 

### **Source Code**
```bash
git clone https://github.com/Meesho/BharatMLStack.git
cd BharatMLStack/online-feature-store
git checkout release/1.0.0
```

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
