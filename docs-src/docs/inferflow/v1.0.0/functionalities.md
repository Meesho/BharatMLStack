---
title: Key Functionalities
sidebar_position: 2
---

# Inferflow - Key Functionalities

## Overview

Inferflow is a high-performance, config-driven ML inference orchestration engine built in **Go**. It provides **no-code feature retrieval**, **DAG-based execution**, and **multi-pattern model inference** — enabling ML teams to onboard new models through configuration changes alone.

---

## Core Capabilities

### Graph-Driven Feature Retrieval

Inferflow's defining feature is its ability to resolve entity relationships and retrieve features through configurable DAG topologies — no custom code required.

**How it works:**

1. A `model_config_id` maps to a pre-defined DAG of components
2. Context entity IDs (e.g., `userId`, `productIds`) are provided at request time
3. The DAG resolves intermediate entity relationships (e.g., extracting `category` from `product` to fetch `user x category` features)
4. Features are fetched in parallel from the Online Feature Store
5. A 2D feature matrix is assembled and passed to model scoring

**Impact:**
- New models require only a config update — no code changes
- Feature consistency is guaranteed across experiments
- Iteration cycles drop from days to minutes

### DAG Topology Executor

The execution engine uses **Kahn's algorithm** for topological ordering with **concurrent goroutine execution** at each level:

```
component_dependency: {
    "feature_initializer": ["fs_user", "fs_product"],
    "fs_user": ["ranker"],
    "fs_product": ["ranker"],
    "ranker": ["reranker"],
    "reranker": []
}
```

This config defines:
- `feature_initializer` runs first (zero in-degree)
- `fs_user` and `fs_product` run **in parallel** after init
- `ranker` runs after both feature components complete
- `reranker` runs after the ranker

**Key properties:**
- Cycle detection via in-degree analysis
- DAG topologies cached using Murmur3 hashing (Ristretto cache)
- Components are registered and resolved via a `ComponentProvider`

---

## Multi-Pattern Inference APIs

Inferflow supports three inference patterns via the **Predict API**, each designed for different ML use cases:

### PointWise Inference

Score each target independently against context features.

```protobuf
rpc InferPointWise(PredictRequest) returns (PredictResponse);
```

**Use cases:** Click-through rate prediction, fraud scoring, relevance ranking

**Input:** Context features + list of targets (e.g., products)
**Output:** Per-target scores

### PairWise Inference

Score pairs of targets relative to each other.

```protobuf
rpc InferPairWise(PredictRequest) returns (PredictResponse);
```

**Use cases:** Preference learning, comparison-based ranking

**Input:** Context features + targets + pair indices (first/second)
**Output:** Per-pair scores + optional per-target scores

### SlateWise Inference

Score groups (slates) of targets together, capturing inter-item effects.

```protobuf
rpc InferSlateWise(PredictRequest) returns (PredictResponse);
```

**Use cases:** Whole-page optimization, slate-level reranking, diversity-aware scoring

**Input:** Context features + targets + slate definitions (target indices per slate)
**Output:** Per-slate scores + optional per-target scores

---

## Entity & Legacy API

### RetrieveModelScore

The original Inferflow API for entity-based feature retrieval and scoring:

```protobuf
service Inferflow {
    rpc RetrieveModelScore(InferflowRequestProto) returns (InferflowResponseProto);
}
```

**Request structure:**

| Field | Description |
|-------|-------------|
| `entities` | List of entity types with their IDs and optional inline features |
| `model_config_id` | Identifies the model configuration (DAG, components, response format) |
| `tracking_id` | Request-level tracing identifier |

**Entity structure:**
- `entity`: Entity type label (e.g., `"user"`, `"product"`)
- `ids`: List of entity IDs
- `features`: Optional inline features (name + per-ID values)

---

## Component Types

### FeatureInitComponent

**Role:** Root DAG node — initializes the shared `ComponentMatrix`.

- Sets up rows from entity IDs
- Populates schema columns (string + byte) for all downstream components
- For slate APIs: initializes `SlateData` with `slate_target_indices`

### FeatureComponent

**Role:** Fetches features from the Online Feature Store (OnFS) for a specific entity type.

- Reads `FSKeys` from config to extract lookup keys from the matrix
- Batches unique entities and calls OnFS via gRPC
- Optional **in-memory caching** keyed by `model_id:version:component:entity`
- Writes binary feature values into matrix byte columns

**Column naming convention:** `entity_label:feature_group:feature_name`

### PredatorComponent

**Role:** Calls model serving endpoints for inference.

- Builds feature payloads from matrix columns with type conversion
- Supports **percentage-based traffic routing** across multiple model endpoints
- Handles **slate-level inference**: per-slate matrix → separate inference → scores to `SlateData`
- Configurable **calibration** and **batch sizing**

### NumerixComponent

**Role:** Calls the Numerix compute engine for operations like reranking.

- Uses `ScoreMapping` config to map matrix columns to compute inputs
- Writes a single score column back to the matrix
- Supports slate mode for per-slate compute operations

---

## Feature Retrieval Pipeline

### Key Resolution

Feature components use `FSKeys` configuration to dynamically resolve entity keys:

```json
{
    "FSKeys": {
        "schema": ["user_id"],
        "col": "user:profile:user_id"
    }
}
```

The component reads key values from the existing matrix columns, enabling **chained entity resolution** — e.g., fetch product entity first, extract category, then fetch user x category features.

### Batched Retrieval

- Features are fetched via `FeatureService.RetrieveFeatures` gRPC call
- Requests are batched by unique entity keys
- Configurable batch size and deadline per component
- Auth via `CALLER_ID` and `CALLER_TOKEN` metadata

### In-Memory Caching

Optional per-component caching reduces OnFS load:

- Cache key: `model_id:cache_version:component_name:entity_key`
- Configurable TTL per component
- Zero-GC-overhead cache implementation available
- Cache hit/miss metrics tracked via StatsD

---

## Data Types

Inferflow supports comprehensive ML data types for feature encoding and model input/output:

| Data Type | Variants | Usage |
|-----------|----------|-------|
| **Integers** | int8, int16, int32, int64 | Categorical encodings, counts, IDs |
| **Floats** | float8 (e4m3, e5m2), float16, float32, float64 | Continuous features, embeddings, scores |
| **Strings** | Variable length | Categories, metadata |
| **Booleans** | Bit-packed | Binary indicators |
| **Vectors** | All scalar types | Embeddings, feature arrays |

Type conversion is handled by the `datatypeconverter` package with optimized float8 implementations.

---

## Inference Logging

Inferflow supports async inference logging to Kafka for model monitoring and debugging:

### Serialization Formats

| Format | Use Case |
|--------|----------|
| **Proto** | Default, compact |
| **Arrow** | Columnar analytics |
| **Parquet** | Long-term storage, query-friendly |

### Sampling Controls

| Config | Description |
|--------|-------------|
| `LoggingPerc` | Percentage of requests to log (0-100) |
| `LogBatchSize` | Batch size for log message grouping |
| `LogFeatures` | Specific features to include in logs |

### Log Content

Each `InferflowLog` message includes:
- `user_id`, `tracking_id`, `model_config_id`
- Entity IDs and feature values
- Model scores and metadata

---

## Configuration Hot-Reload

Model configurations are stored in **etcd** and support **live updates without redeployment**:

1. Inferflow registers watchers on etcd config paths
2. On config change, watchers trigger `ReloadModelConfigMapAndRegisterComponents`
3. `ConfigMap` is updated in memory
4. Feature schemas are re-initialized
5. DAG components are re-registered

This enables:
- Adding new models in production without restarts
- A/B testing with different model configurations
- Instant rollback by reverting etcd config

---

## Performance Characteristics

### Concurrency Model
- DAG components at the same level execute concurrently in goroutines
- Feature retrieval is parallelized across entity types
- External gRPC calls use connection pooling

### Memory Efficiency
- Built in Go — significantly lower memory footprint than Java equivalents (~80% reduction)
- Object pooling for `ComponentMatrix` and serialization buffers
- In-memory cache with zero-GC-overhead option (freecache)

### Serialization
- gRPC with Proto3 for all external communication
- Binary feature encoding in the `ComponentMatrix` for minimal overhead
- Configurable compression for Kafka logging

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
