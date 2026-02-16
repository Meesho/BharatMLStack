<div align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/bharatmlstack-logo.png" />
    <source media="(prefers-color-scheme: light)" srcset="assets/bharatmlstack-logo.png" />
    <img src="assets/bharatmlstack-logo.png" alt="BharatMLStack" width="100%"/>
  </picture>
</div>



<div align="center">

[![CI](https://github.com/Meesho/BharatMLStack/actions/workflows/ci.yml/badge.svg)](https://github.com/Meesho/BharatMLStack/actions/workflows/ci.yml)
![Static Badge](https://img.shields.io/badge/oss_lifecycle-active-green)
[![Discord](https://img.shields.io/badge/Discord-Join%20Chat-7289da?style=flat&logo=discord&logoColor=white)](https://discord.gg/XkT7XsV2AU)
[![Made in India](https://img.shields.io/badge/Made%20in-India-FF9933?style=flat)](https://en.wikipedia.org/wiki/India)

[![Meesho](https://img.shields.io/badge/Built%20by%20Meesho-%20%E2%9D%A4%20-white?style=flat&labelColor=white&color=450839)](https://meesho.com)


</div>

## What is BharatMLStack?

BharatMLStack is a production-ready, cloud-agnostic ML infrastructure platform that powers real-time feature serving, model inference, and embedding search at massive scale. Built and battle-tested at [Meesho](https://meesho.com), it is designed to help organizations ship ML to production faster, cheaper, and more reliably.

## Our Vision

BharatMLStack is built around **four core tenets**:

### Workflow Integration & Productivity
> Ship ML to production faster than ever.

- **3x faster** experiment-to-deployment cycles
- **95% reduction** in model onboarding time

### Cloud-Agnostic & Lock-In Free
> Run anywhere. Own your stack.

- Runs across **public cloud, on-prem, and edge**
- Kubernetes-native with zero vendor lock-in

### Economic Efficiency
> Do more with less.

- **60–70% lower** infrastructure costs vs hyperscaler managed services
- Optimized resource utilization across CPU and GPU workloads

### Availability & Scalability
> Enterprise-grade reliability at internet scale.

- **99.99% uptime** across clusters
- **1M+ QPS** with low latency

## Designed Truly for Bharat Scale

Built for the demands of one of the world's largest e-commerce platforms:

| Metric | Performance |
|--------|-------------|
| **Feature Store** | 2.4M QPS (batch of 100 id lookups) |
| **Model Inference** | 1M+ QPS |
| **Embedding Search** | 500K QPS |
| **Feature Retrieval Latency** | Sub-10ms |

## Core Components

| Component | Description | Version | Docs |
|-----------|-------------|---------|------|
| **[Horizon](./horizon/)** | Control plane that orchestrates all services and powers TruffleBox UI | `v1.3.0` | [Docs](https://meesho.github.io/BharatMLStack/) |
| **[TruffleBox UI](./trufflebox-ui/)** | Web console for feature registry, cataloging, and approval workflows | `v1.3.0` | [Docs](https://meesho.github.io/BharatMLStack/trufflebox-ui/v1.0.0/userguide) |
| **[Online Feature Store](./online-feature-store/)** | Sub-10ms feature retrieval at millions of QPS with streaming ingestion | `v1.2.0` | [Docs](https://meesho.github.io/BharatMLStack/category/online-feature-store) |
| **[Inferflow](./inferflow/)** | DAG-based real-time inference orchestration for composable ML pipelines | `v1.0.0` | [Docs](https://meesho.github.io/BharatMLStack/category/inferflow) |
| **[Numerix](./numerix/)** | Rust-powered math compute engine for high-performance matrix ops | `v1.0.0` | [Docs](https://meesho.github.io/BharatMLStack/category/numerix) |
| **[Skye](./skye/)** | Vector similarity search with pluggable backends | `v1.0.0` | [Docs](https://meesho.github.io/BharatMLStack/category/skye) |
| **[Interaction Store](./interaction-store/)** | ScyllaDB-backed store for user interaction signals at sub-10ms | — | — |
| **[Go SDK](./go-sdk/)** | Go client for Feature Store, Interaction Store, and logging | `v1.2.0` | [Docs](https://meesho.github.io/BharatMLStack/category/go-sdk) |
| **[Python SDK](./py-sdk/)** | Python client libraries for Feature Store and inference logging | `v1.0.1` | [Docs](https://meesho.github.io/BharatMLStack/category/python-sdk) |

> Full documentation at [meesho.github.io/BharatMLStack](https://meesho.github.io/BharatMLStack/) | [Blogs](https://meesho.github.io/BharatMLStack/blog)
- [All Blog Posts](https://meesho.github.io/BharatMLStack/blog)

## Quick Start

```bash
git clone https://github.com/Meesho/BharatMLStack.git
cd BharatMLStack/quick-start
ONFS_VERSION=v1.2.0 HORIZON_VERSION=v1.3.0 TRUFFLEBOX_VERSION=v1.3.0 NUMERIX_VERSION=v1.0.0 ./start.sh
```

For step-by-step setup, Docker Compose details, sample data, and health checks, see the full **[Quick Start Guide →](./quick-start/README.md)**.

## Architecture

<div align="center">
  <img src="assets/bharatmlstack-architecture.png" alt="BharatMLStack Architecture" width="1000"/>
</div>

## Use-Cases

BharatMLStack powers a wide range of ML-driven applications:

| Use-Case | What BharatMLStack Enables |
|----------|---------------------------|
| **Personalized Candidate Generation** | Retrieve and rank millions of candidates in real time using feature vectors and embedding similarity |
| **Personalized Ranking** | Serve user, item, and context features at ultra-low latency to power real-time ranking models |
| **Fraud & Risk Detection** | Stream interaction signals and features to detect anomalies and fraudulent patterns in milliseconds |
| **Image Search** | Run embedding search at 500K QPS to match visual queries against massive product catalogs |
| **LLM Recommender Systems** | Orchestrate LLM inference pipelines with feature enrichment for next-gen recommendation engines |
| **DL & LLM Deployments at Scale** | Deploy and scale deep learning and large language models across GPU clusters with Inferflow orchestration |

## Contributing

We welcome contributions from the community! Please see our [Contributing Guide](CONTRIBUTING.md) for details on how to get started.

## Community & Support

- **Discord**: Join our [community chat](https://discord.gg/XkT7XsV2AU)
- **Issues**: Report bugs and request features on [GitHub Issues](https://github.com/Meesho/BharatMLStack/issues)
- **Email**: Contact us at [ml-oss@meesho.com](mailto:ml-oss@meesho.com)

## License

BharatMLStack is open-source software licensed under the [BharatMLStack Business Source License 1.1](LICENSE.md).

---

<div align="center">
  <strong>Built with ❤️ for the ML community from Meesho</strong>
</div>
<div align="center">
  <strong>If you find this useful, ⭐️ the repo — your support means the world to us!</strong>
</div>
