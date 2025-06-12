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

✨ **Production-Ready**: Battle-tested components used in high-traffic production systems
🔒 **Enterprise Security**: End-to-end encryption, audit logs, and compliance ready
🌐 **Cloud Native**: Kubernetes-native with multi-cloud support
📊 **Observability**: Built-in monitoring, logging, and distributed tracing
🔄 **GitOps Integration**: Infrastructure as code with automated deployments
🤖 **AI/ML Ops**: Complete MLOps lifecycle from experimentation to production

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
          ├─────────────────────┬─────────────────────┐
          ▼                     ▼                     ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ Feature Store   │    │ Model Serving   │    │  Data Pipeline  │
│  (Real-time)    │    │   (Inference)   │    │  (Processing)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Contributing

We welcome contributions from the community! Please see our [Contributing Guide](CONTRIBUTING.md) for details on how to get started.

## Community & Support

- 📖 **Documentation**: [docs.bharatmlstack.com](https://docs.bharatmlstack.com)
- 💬 **Discord**: Join our [community chat](https://discord.gg/bharatmlstack)
- 🐛 **Issues**: Report bugs and request features on [GitHub Issues](https://github.com/bharatml/BharatMLStack/issues)
- 📧 **Email**: Contact us at [hello@bharatmlstack.com](mailto:hello@bharatmlstack.com)

## License

BharatMLStack is open-source software licensed under the [Apache License 2.0](LICENSE.md).

---

<div align="center">
  <strong>Built with ❤️ for the ML community</strong>
</div>
