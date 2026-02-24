---
title: Functionalities
sidebar_position: 2
---

# Skye - Functionalities

## Core Capabilities

### 1. Vector Similarity Search

Skye provides real-time nearest-neighbor search across high-dimensional vector spaces. It supports:

- **Configurable distance functions**: DOT product, Cosine similarity, Euclidean distance
- **Configurable vector dimensions**: Per-model vector dimension settings
- **Indexed-only search**: Queries only search within fully indexed space, avoiding brute-force fallback on partially built indexes
- **Pagination support**: Service-level pagination for clients, even when the underlying vector DB does not natively support it

### 2. Pluggable Vector Database Support

The platform is designed to be vector DB agnostic:

- **Generic vector config**: A `vector_db_type` field and generic `vectordb_config` replace vendor-specific configurations
- **Current support**: Qdrant with official Go client
- **Extensibility**: New vector databases can be integrated by implementing the vector DB interface

### 3. Model and Variant Management

#### Model Registration
- Models are registered via API with entity type, embedding configuration, distance function, vector dimension, and training data path
- Each model is associated with a store ID mapping to specific embedding and aggregator tables

#### Variant Registration
- Variants represent different views/filters of the same model (e.g., organic, ad, commerce)
- Each variant has its own filter criteria, vector DB cluster, job frequency, and version tracking
- Variants share the same embeddings, eliminating data redundancy

#### Model Promotion
- Successful experiments can be promoted from experiment clusters to production clusters via API

### 4. Embedding Ingestion

#### Batch Ingestion (Reset/Delta Jobs)
- Triggered via Databricks jobs that read from GCS paths
- Supports separate index-space and search-space embeddings
- Per-variant `to_be_indexed` flags control which embeddings are indexed for each variant
- EOF markers sent to all Kafka partitions ensure complete data consumption

#### Real-Time Ingestion
- Generic Kafka schema for all real-time consumers
- Entity-based aggregation data (e.g., is_live_ad, out_of_stock) updates in real time
- During model resets, real-time consumers continue pushing data to the latest collection (no pausing)

### 5. Real-Time Data Aggregation

- Entity-wise (catalog, product, user) real-time aggregation via ScyllaDB
- Generic approach: aggregator tables are entity-level, not model/version-specific
- All real-time data is consistent across models sharing the same entity

### 6. Intelligent Caching

- **In-memory cache**: First layer, reduces load on distributed cache
- **Distributed cache (Redis)**: Second layer for cached similarity results
- Hit rate monitoring and cache effectiveness metrics per model

### 7. Embedded Storage

- Optional embedding storage with configurable TTL
- Enables embedding lookup APIs for downstream consumers
- Stored in ScyllaDB with efficient binary serialization

### 8. Retry and Fault Tolerance

- **Retry topic**: Failed ingestion events are published to a dedicated retry topic
- **Event-driven state management**: Model states persist in SQL DB, surviving pod restarts
- **Kafka-based admin**: Asynchronous processing with automatic re-consumption on failure

### 9. Experiment Isolation

- Dedicated EKS cluster (`skye-service-experiments`) for experiments
- Dedicated vector DB cluster for experiment workloads
- Clean separation from production: experiments do not impact production performance
- Promotion path from experiment to production after load analysis

### 10. Centralized Cluster Management

- Automated cluster provisioning via scripts (collaboration with DevOps)
- Consistent configurations across all clusters (eliminates consensus issues)
- Horizontal scaling support: generic scripts for adding nodes to existing clusters

---

## Onboarding Flow

### Step-by-step Process

1. **Data Scientist** provides a base GCS path where model embeddings will be pushed
2. **Register Model** via `POST /register-model` with entity type, column mappings, model config
3. **Register Variant(s)** via `POST /register-variant` with filter criteria, vector DB config, job frequency
4. **Schedule Databricks Job** to read data from GCS path and ingest into Skye platform
5. **Reset Model** via `POST /reset-model` to trigger the first full ingestion
6. **Trigger Model Machine** via `POST /trigger-model-machine` to start the indexing pipeline

### Extending to New Tenants

With the variant system, extending a model to a new tenant only requires registering a new variant with appropriate filters -- no re-ingestion of embeddings is needed.
