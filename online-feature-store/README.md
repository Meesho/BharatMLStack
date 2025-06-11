<p align="center">
  <img src="assets/logov2.webp" alt="Online-feature-store" width="500"/>
</p>

----
# BharatMLStack's Online Feature Store: A Hyper-Scalable Feature Store for Real-Time ML

![Static Badge](https://img.shields.io/badge/oss_lifecycle-active-green) 
![build status](https://github.com/Meesho/BharatMLStack/actions/workflows/online-feature-store.yml/badge.svg)
[![Discord](https://img.shields.io/badge/Discord-Join%20Chat-5865F2?logo=discord&logoColor=white)](https://discord.gg/474wHtfm)
[![Roadmap](https://img.shields.io/badge/Project-Roadmap-blue?style=flat&logo=github)](./roadmap/roadmap_2025_2026.md)

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

![Online-feature-store Architecture](assets/production-architecture.png)

## 🚀 Quick Start

For detailed setup instructions, see the [**Quick Start Guide**](quick-start/README.md).

The quickest way to get started with Online-feature-store is to use the quick start scripts:

```bash
# Clone the repository
git clone https://github.com/Meesho/BharatMLStack.git
cd BharatMLStack/online-feature-store/quick-start

# Start Online-feature-store and all dependencies
./start.sh
```

This will spin up a complete Online-feature-store environment with:
- ScyllaDB, MySQL, Redis, and etcd databases
- Horizon backend API (port 8082)
- TruffleBox UI (port 3000)
- Online-feature-store gRPC API Server (port 8089)

To stop all services:
```bash
./stop.sh
```

## 🧰 SDKs

Online-feature-store provides SDKs to interact with the feature store:

- **[Go SDK](sdks/go/README.md)** - For backend services and ML inference
- **[Python SDK](sdks/python/README.md)** - For feature ingestion and Spark jobs

## 💡 Key Features

- **Low-latency retrieval** - In-memory LRU caches and efficient binary formats
- **Efficient storage** - Optimized binary data blocks with PSDBBlocks and CSDBBlocks
- **Cache hierarchy** - Multi-level caching with Redis/Dragonfly and in-memory options
- **Schema versioning** - Safe evolution of feature schemas with backward compatibility
- **Operational simplicity** - Centralized configuration via TruffleBox UI
- **Comprehensive monitoring** - Deep visibility into system performance with Prometheus/Grafana

## 📊 Use Cases

Online-feature-store is ideal for:

- **Real-time inference** - Low-latency feature serving for online ML models, like Ranking and Recommendation, Fraud detection, Search, Dynamic pricing and bidding, real-time notification etc real time 
- **Feature Catalogue** - Centralized feature registry for organizations
- **Flexible Data Ingestion** - Support for batch processing, streaming pipelines, and real-time data sources with unified ingestion APIs

## 🖥️ UI and Management

TruffleBox provides a comprehensive UI for:

- Entity and feature group registration
- Feature discovery and metadata management
- TTL and caching configuration
- Request approval workflows

To access the UI after starting Online-feature-store:
```
http://localhost:3000
```

## 📚 Documentation

For more detailed information, check out:

- [Architecture Overview](docs/architecture.md)
- [Storage Design - PSDB](docs/psdb-design.md)
- [Storage Design - CSDB](docs/csdb-design.md)
- [Key-Schema Isolation](docs/schema.md)
- [TruffleBox UI Guide](docs/usage.md)
- [2025-2026 Roadmap](roadmap/roadmap_2025_2026.md)
- [Contribution Guide](docs/CONTRIBUTION.md)
- [Code of Conduct](docs/CODE_OF_CONDUCT.md)
- [License](LICENSE.md)

## 🤝 Contributing

Contributions are welcome! Please check our [Contribution Guide](docs/CONTRIBUTION.md) for details on how to get started.

We encourage you to:
- Join our [Discord community](https://discord.gg/474wHtfm) to discuss features, ideas, and questions
- Check existing issues before opening a new one
- Follow our coding guidelines and pull request process
- Participate in code reviews and discussions

## 📫 Need Help?

There are several ways to get help with Online-feature-store:

- Join the [Online-feature-store Discord community](https://discord.gg/474wHtfm) for questions and discussions
- Open an issue in the repository for bug reports or feature requests
- Check the documentation for guides and examples
- Reach out to the Online-feature-store core team

Feedback and contributions are welcome!

## ⚖️ License

This project is proprietary software of Meesho Technologies under the [Business Source License 1.1](LICENSE.md).
