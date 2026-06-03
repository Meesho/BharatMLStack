<p align="center">
  <img src="../assets/logov2.webp" alt="mNemo" width="400"/>
</p>

---

# mNemo: WORM Database Platform

![Build Status](https://github.com/Meesho/BharatMLStack/actions/workflows/mnemo.yml/badge.svg)
![Version](https://img.shields.io/badge/release-v0.1.0-blue?style=flat)
[![Discord](https://img.shields.io/badge/Discord-Join%20Chat-7289da?style=flat&logo=discord&logoColor=white)](https://discord.gg/XkT7XsV2AU)

mNemo is a multi-tenant **WORM (Write-Once-Read-Many)** database platform built for high-throughput immutable dataset serving at ML inference scale.

- **>2M RPS** with **<2ms p99** on 10 TB+ datasets
- **Sub-second rollback** via atomic version pointer flip (V=2)
- **Zero-downtime version updates** — in-flight reads never blocked
- **~$12K/mo per 10 TB tenant** via ZSTD 9.5× compression + KEDA autoscaling
- **Multi-tenant** — independent versioning and refresh schedules per store

## Architecture

```
Producer (Databricks/PySpark) → shard + version SSTs → GCS
Control Plane (Go + etcd)     → onboarding, version lifecycle, placement
Data Plane (per shard)        → Data Loader (Go sidecar) → RocksDB → Read Server (Rust, TCP)
Client SDK (Go)               → crc32%S → assignment map → warm pod → TCP
```

## Components

| Component | Language | Purpose |
|---|---|---|
| `schema/` | Go | Shared etcd key builders + data types |
| `controlplane/` | Go | REST API: onboarding, version lifecycle, sizing |
| `dataplane/readserver/` | Rust + Tokio | TCP binary server, RocksDB read-only engine, ArcSwap version flip |
| `dataplane/dataloader/` | Go | Sidecar: GCS multipart download, IPC to read server, etcd watch |
| `sdk/` | Go | Client: CRC32 routing, scatter-gather BatchGet, TCP connection pool |
| `py-producer/` | Python | PySpark pipeline: parquet → SST → QC → GCS → publish |

## Quick Start

```bash
# Start local etcd
make dev-etcd

# Run all tests
make test

# Check 100% coverage
make coverage-check

# Stop local services
make dev-stop
```

See [quick-start/](quick-start/) for the Docker Compose setup.

## Documentation

| Version | Link |
|---|---|
| v1.0.0 | [Documentation](https://meesho.github.io/BharatMLStack/mnemo/v1.0.0) |

Full implementation plan: [PLAN.md](PLAN.md)
