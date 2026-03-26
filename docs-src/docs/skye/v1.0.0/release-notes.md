---
title: Release Notes
sidebar_position: 3
---

# Skye - Release Notes

## v1.0.0

### Overview

Initial open-source release of Skye, BharatMLStack's vector similarity search platform. This release represents a complete re-architecture of the internal VSS (Vector Similarity Search) service, addressing scalability, resilience, and operational efficiency challenges from the previous generation.

### What's New

#### Architecture
- **Model-first hierarchy**: Models at the base level with variants nested within, eliminating embedding duplication across tenants
- **Entity-based data split**: Separate embedding and aggregator tables per entity type (catalog, product, user)
- **Event-driven admin flows**: Kafka-based model lifecycle management with SQL-backed state persistence
- **Pluggable vector DB support**: Generic vector database abstraction replacing vendor-specific tight coupling

#### Serving
- **Multi-layer caching**: In-memory cache + Redis distributed cache for low-latency similarity search
- **Indexed-only search**: `search_indexed_only` flag prevents brute-force fallback on partially indexed collections
- **Pagination support**: Service-level pagination for clients
- **Separate search/index embeddings**: Models can use different embedding spaces for search and indexing

#### Ingestion
- **Shared embeddings across variants**: Single ingestion per model with parallel variant processing
- **Generic RT consumer schema**: Simplified onboarding for new real-time data sources
- **Retry topic**: Automatic capture and reprocessing of failed ingestion events
- **EOF to all partitions**: Ensures complete data consumption before processing completion

#### Operations
- **API-based model onboarding**: Register models and variants via REST API (replaces manual Databricks-only flow)
- **Automated cluster provisioning**: Scripted setup for consistent vector DB cluster configurations
- **Experiment isolation**: Dedicated EKS and vector DB clusters for experiments
- **Comprehensive observability**: Per-model + per-variant metrics for latency, throughput, error rates, and cache effectiveness

### Improvements Over Previous Architecture

| Area | Before | After |
|---|---|---|
| Embedding storage | Duplicated per tenant | Shared per model |
| Vector DB coupling | Tightly coupled to Qdrant | Pluggable via generic interface |
| State management | In-pod synchronous thread | Event-driven with SQL backing |
| Consumer handling | Paused during ingestion | No pausing; concurrent writes |
| Cluster setup | Manual, error-prone | Automated, consistent |
| Experiment infra | Shared with production | Isolated clusters |
| Failure recovery | Manual intervention | Retry topics + snapshots |
| Observability | Generic alerts | Model + variant level metrics |

### Known Limitations

- Snapshot restore is currently supported for smaller indexes only
- Pagination is handled at the service level (not natively by the vector DB)
- Horizontal scaling of vector DB clusters requires running provisioning scripts

### Technology Stack

- **Language**: Go
- **Vector Database**: Qdrant (pluggable)
- **Storage**: ScyllaDB
- **Cache**: Redis + In-Memory
- **Message Queue**: Kafka
- **Configuration**: ZooKeeper / etcd
- **Orchestration**: Kubernetes (EKS)
