# BharatMLStack

<div align="center">
  <img src="assets/bharatmlstack.jpg" alt="BharatMLStack Logo" width="400"/>
</div>

## What is BharatMLStack?

BharatMLStack is a comprehensive, production-ready machine learning infrastructure platform designed to democratize ML capabilities across India and beyond. Our mission is to provide a robust, scalable, and accessible ML stack that empowers organizations to build, deploy, and manage machine learning solutions at massive scale.

## Our Vision

🎯 **Democratize Machine Learning**: Make advanced ML infrastructure accessible to organizations of all sizes
🚀 **Scale Without Limits**: Built to handle millions of requests per second with enterprise-grade reliability
🇮🇳 **India-First Approach**: Optimized for Indian market needs while maintaining global standards
⚡ **Real-Time Intelligence**: Enable instant decision-making with sub-millisecond feature serving
🔧 **Developer-Friendly**: Intuitive APIs and interfaces that accelerate ML development cycles

## Running at Million Scale

BharatMLStack is battle-tested in production environments, powering:
- **1M+ predictions per second** across distributed deployments
- **Sub-10ms latency** for real-time feature retrieval
- **99.99% uptime** with auto-scaling and fault tolerance
- **Petabyte-scale** feature storage and processing
- **Multi-region deployments** with global load balancing

## Core Components

### 🚀 Horizon - Control Plane & Backend
The central control plane for BharatMLStack components, serving as the backend for Trufflebox UI.
- **Component orchestration**: Manages and coordinates all BharatMLStack services
- **API gateway**: Unified interface for all MLOps and workflows

### 🎨 Trufflebox UI - ML Management Console  
Modern web interface for managing ML models, features, and experiments. Currently it supports:
- **Feature Registry**: Centralized repository for feature definitions and metadata
- **Feature Cataloging**: Discovery and search capabilities for available features
- **Online Feature Store Control System**: Management interface for feature store operations
- **Approval Flows**: Workflow management for feature deployment and changes 

### 🗄️ Online Feature Store - Real-Time Features
High-performance feature store for real-time ML inference and training.
- **Real-time serving**: Sub-10ms feature retrieval at scale  
- **Streaming ingestion**: Process millions of feature updates per second
- **Feature Backward Compatible Versioning**: Track and manage feature evolution
- **Multi-source integration**: Push from stream, batch and real-time sources

## Key Differentiators

- ✨ **Production-Ready**: Battle-tested components used in high-traffic production systems
- 🌐 **Cloud Agnostic**: Kubernetes-native, so deply on the cloud you love
- 📊 **Observability**: Built-in monitoring, logging


## Quick Start

```bash
# Clone the repository
git clone https://github.com/bharatml/BharatMLStack.git
cd BharatMLStack

# Deploy the full stack
./scripts/deploy.sh --environment production

# Access the UI
open http://localhost:8080
```

## Architecture

BharatMLStack follows a microservices architecture designed for scalability and maintainability:

```
┌─────────────────┐
│  Trufflebox UI  │
│   (Frontend)    │
└─────────┬───────┘
          │
          ▼
┌─────────────────┐
│     Horizon     │ ◄── Control Plane & Backend
│ (Control Plane) │
└─────────┬───────┘
          │
          ├─────────────────────┬
          ▼                     ▼                     
┌─────────────────┐    ┌─────────────────┐
│ Feature Store   │    │ Feature Store   │
│  GRPC Server    │    │   Consumer      │
└─────────────────┘    └─────────────────┘
```

## 📚 Documentation

### Comprehensive Documentation Hub

For detailed technical documentation, architecture deep-dives, and implementation guides, visit our comprehensive documentation in:

📖 **[Online Feature Store Documentation](./online-feature-store/docs/README.md)**

### 🎯 What You'll Find

#### **Core Architecture & Design**
- **[System Architecture](./online-feature-store/docs/architecture.md)** - Detailed system design and component interactions
- **[Schema Management](./online-feature-store/docs/schema.md)** - Key-schema isolation and etcd mapping strategies
- **[Performance Benchmarks](./online-feature-store/docs/)** - Latency, throughput, and scalability metrics

#### **Developer Guides**
- **[API Documentation](./online-feature-store/docs/)** - Complete API reference and usage examples
- **[SDK Integration](./go-sdk/)** - Go SDK for seamless integration
- **[CLI Tools](./quick-start/CLI-README.md)** - Command-line interface for testing and management

#### **Deployment & Operations**
- **[Quick Start Guide](./quick-start/)** - Get up and running in minutes
- **[Production Deployment](./online-feature-store/docs/)** - Enterprise deployment patterns
- **[Monitoring & Observability](./online-feature-store/docs/)** - Comprehensive monitoring setup

#### **Use Cases & Examples**
- **[Real-time ML Pipelines](./online-feature-store/docs/)** - Production ML workflow examples
- **[Feature Engineering](./online-feature-store/docs/)** - Best practices for feature development
- **[Scaling Patterns](./online-feature-store/docs/)** - Handle millions of requests per second

### 🚀 Quick Navigation

| Component | Documentation | Quick Start |
|-----------|--------------|-------------|
| **Online Feature Store** | [Docs](./online-feature-store/docs/) | [Setup](./quick-start/) |
| **Go SDK** | [Docs](./go-sdk/README.md) | [Examples](./go-sdk/README.md) |
| **Python SDK** | [Docs](./py-sdk/README.md) | [Quickstart](./py-sdk/README.md) |

### 💡 Getting Started Resources

**New to BharatMLStack?** Start here:
1. 📖 Read the [System Overview](./online-feature-store/docs/README.md)
2. 🚀 Follow the [Quick Start Guide](./quick-start/)
3. 🔧 Try the [CLI Tutorial](./quick-start/CLI-README.md)
4. 🏗️ Explore [Architecture Details](./online-feature-store/docs/architecture.md)

**Ready for Production?** Check out:
- 🏭 [Production Deployment Guide](./online-feature-store/docs/)
- 📊 [Performance Tuning](./online-feature-store/docs/)
- 🔐 [Security & Authentication](./online-feature-store/docs/)
- 📈 [Monitoring & Alerting](./online-feature-store/docs/)

## Contributing

We welcome contributions from the community! Please see our [Contributing Guide](CONTRIBUTING.md) for details on how to get started.

## Community & Support

- 💬 **Discord**: Join our [community chat](https://discord.gg/XkT7XsV2AU)
- 🐛 **Issues**: Report bugs and request features on [GitHub Issues](https://github.com/Meesho/BharatMLStack/issues)
- 📧 **Email**: Contact us at [hello@bharatmlstack.com](mailto:ml-oss@meesho.com )

## License

BharatMLStack is open-source software licensed under the [BharatMLStack Business Source License 1.1](LICENSE.md).

---

<div align="center">
  <strong>Built with ❤️ for the ML community from Meesho</strong>
</div>
