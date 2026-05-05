---
title: Configuration Guide
sidebar_position: 3
---

# Inferflow - Configuration Guide

Inferflow is fully config-driven. All model onboarding, feature retrieval logic, DAG topology, and inference behavior are controlled through configuration stored in **etcd** — with zero code changes required.

---

## Configuration Overview

Inferflow configuration is organized into two layers:

1. **Static config** — Environment variables loaded at startup (via Viper)
2. **Dynamic config** — Model configurations stored in etcd, hot-reloaded on change

---

## Static Configuration (Environment Variables)

These are set at deployment time and require a restart to change.

### Server

| Variable | Description | Example |
|----------|-------------|---------|
| `APP_PORT` | gRPC/HTTP server port | `50051` |
| `APP_ENV` | Environment name | `production` |

### etcd

| Variable | Description | Example |
|----------|-------------|---------|
| `ETCD_ENDPOINTS` | Comma-separated etcd endpoints | `etcd-0:2379,etcd-1:2379` |
| `ETCD_DIAL_TIMEOUT` | Connection timeout | `5s` |

### Online Feature Store (OnFS)

| Variable | Description | Example |
|----------|-------------|---------|
| `externalServiceOnFs_host` | OnFS gRPC host | `onfs-api:50051` |
| `externalServiceOnFs_callerId` | Caller ID for auth | `inferflow` |
| `externalServiceOnFs_callerToken` | Caller token for auth | `<token>` |
| `externalServiceOnFs_batchSize` | Batch size for feature retrieval | `100` |
| `externalServiceOnFs_deadline` | Request deadline | `200ms` |

### Predator (Model Serving)

| Variable | Description | Example |
|----------|-------------|---------|
| `externalServicePredator_defaultDeadline` | Default inference deadline | `100ms` |

### Numerix (Compute Engine)

| Variable | Description | Example |
|----------|-------------|---------|
| `numerixClientV1_host` | Numerix gRPC host | `numerix:50052` |
| `numerixClientV1_deadline` | Request deadline | `100ms` |

### Kafka (Inference Logging)

| Variable | Description | Example |
|----------|-------------|---------|
| `KafkaBootstrapServers` | Kafka broker addresses | `kafka-0:9092,kafka-1:9092` |
| `KafkaLoggingTopic` | Topic for inference logs | `inferflow-logs` |

### Metrics (StatsD / Telegraf)

| Variable | Description | Example |
|----------|-------------|---------|
| `TELEGRAF_HOST` | StatsD host | `telegraf` |
| `TELEGRAF_PORT` | StatsD port | `8125` |

### In-Memory Cache

| Variable | Description | Example |
|----------|-------------|---------|
| `CACHE_SIZE_MB` | Cache size in MB | `512` |
| `CACHE_TYPE` | Cache implementation | `freecache` |

---

## Dynamic Configuration (etcd Model Config)

Model configurations are stored in etcd and hot-reloaded. Each model is identified by a `model_config_id`.

### Config Structure

```json
{
    "model_config_id_example": {
        "dag_execution_config": {
            "component_dependency": {
                "feature_initializer": ["fs_user", "fs_product"],
                "fs_user": ["ranker_model"],
                "fs_product": ["ranker_model"],
                "ranker_model": []
            }
        },
        "component_config": {
            "feature_component_config": {
                "fs_user": { ... },
                "fs_product": { ... }
            },
            "predator_component_config": {
                "ranker_model": { ... }
            },
            "numerix_component_config": {},
            "cache_enabled": true,
            "cache_version": "v1",
            "cache_ttl": 300,
            "error_logging_percent": 10
        },
        "response_config": {
            "features": ["ranker_model:score"],
            "model_schema_perc": 100,
            "logging_perc": 5,
            "log_features": ["fs_user:profile:age", "ranker_model:score"],
            "log_batch_size": 100
        }
    }
}
```

---

### DAG Execution Config

Defines the component dependency graph.

```json
{
    "component_dependency": {
        "<parent_component>": ["<child_1>", "<child_2>"],
        "<child_1>": ["<grandchild>"],
        "<child_2>": ["<grandchild>"],
        "<grandchild>": []
    }
}
```

**Rules:**
- The graph must be a valid DAG (no cycles)
- Components with no parents (zero in-degree) execute first
- Components with empty dependency arrays `[]` are leaf nodes
- All component names must match registered components in the `ComponentConfig`

---

### Feature Component Config

Configures how features are fetched from the Online Feature Store.

```json
{
    "fs_user": {
        "fs_keys": {
            "schema": ["user_id"],
            "col": "context:user:user_id"
        },
        "fs_request": {
            "entity_label": "user",
            "feature_groups": [
                {
                    "label": "demographics",
                    "feature_labels": ["age", "location", "income_bracket"]
                },
                {
                    "label": "behavior",
                    "feature_labels": ["click_rate", "purchase_freq"]
                }
            ]
        },
        "fs_flatten_resp_keys": ["user_id"],
        "col_name_prefix": "user",
        "comp_cache_enabled": true,
        "comp_cache_ttl": 600,
        "composite_id": false
    }
}
```

| Field | Description |
|-------|-------------|
| `fs_keys` | How to extract lookup keys from the matrix. `schema` defines key column names; `col` references a matrix column |
| `fs_request` | OnFS query: entity label + feature groups with specific features |
| `fs_flatten_resp_keys` | Keys to flatten in response mapping |
| `col_name_prefix` | Prefix for matrix column names (e.g., `user:demographics:age`) |
| `comp_cache_enabled` | Enable in-memory caching for this component |
| `comp_cache_ttl` | Cache TTL in seconds |
| `composite_id` | Whether entity keys are composite |

---

### Predator Component Config

Configures model inference endpoints.

```json
{
    "ranker_model": {
        "model_name": "product_ranker_v3",
        "model_endpoint": "predator-ranker:8080",
        "model_end_points": {
            "predator-ranker-v3:8080": 80,
            "predator-ranker-v4:8080": 20
        },
        "deadline": 100,
        "batch_size": 50,
        "calibration": {
            "enabled": false
        },
        "inputs": {
            "feature_map": {
                "user:demographics:age": "INT32",
                "user:behavior:click_rate": "FP32",
                "product:attributes:category_id": "INT32"
            }
        },
        "outputs": {
            "score_columns": ["score", "confidence"]
        },
        "slate_component": false
    }
}
```

| Field | Description |
|-------|-------------|
| `model_name` | Model identifier on the serving platform |
| `model_endpoint` | Primary model serving endpoint |
| `model_end_points` | Multiple endpoints with percentage-based traffic routing |
| `deadline` | Inference timeout in milliseconds |
| `batch_size` | Max items per inference batch |
| `calibration` | Score calibration settings |
| `inputs.feature_map` | Map of matrix column → data type for model input |
| `outputs.score_columns` | Column names for model output scores |
| `slate_component` | If true, runs per-slate inference |

---

### Numerix Component Config

Configures compute operations (e.g., reranking).

```json
{
    "reranker": {
        "score_column": "final_score",
        "data_type": "FP32",
        "score_mapping": {
            "ranker_model:score": "FP32",
            "user:behavior:click_rate": "FP32"
        },
        "compute_id": "diversity_rerank_v1",
        "slate_component": false
    }
}
```

| Field | Description |
|-------|-------------|
| `score_column` | Output column name for the computed score |
| `data_type` | Output data type |
| `score_mapping` | Map of matrix columns to include as compute inputs |
| `compute_id` | Identifies the compute operation on Numerix |
| `slate_component` | If true, runs per-slate compute |

---

### Response Config

Controls what data is returned to the client and what is logged.

```json
{
    "features": ["ranker_model:score", "reranker:final_score"],
    "model_schema_perc": 100,
    "logging_perc": 5,
    "log_features": [
        "user:demographics:age",
        "ranker_model:score",
        "reranker:final_score"
    ],
    "log_batch_size": 100
}
```

| Field | Description |
|-------|-------------|
| `features` | Matrix columns to include in the gRPC response |
| `model_schema_perc` | Percentage of requests that include full schema in response |
| `logging_perc` | Percentage of requests to send to Kafka for logging |
| `log_features` | Specific features to include in log messages |
| `log_batch_size` | Batch size for grouped log messages |

---

### Service-Level Config

Global settings that apply across all models.

```json
{
    "v2_logging_type": "proto",
    "compression_enabled": false
}
```

| Field | Values | Description |
|-------|--------|-------------|
| `v2_logging_type` | `proto`, `arrow`, `parquet` | Serialization format for Kafka inference logs |
| `compression_enabled` | `true`, `false` | Enable compression for log messages |

---

## Example: Onboarding a New Model

To onboard a new ranking model, update the etcd config:

**Step 1:** Define the feature retrieval graph

```json
"component_dependency": {
    "feature_initializer": ["fs_user", "fs_product", "fs_user_x_category"],
    "fs_product": ["fs_user_x_category"],
    "fs_user": ["new_ranker"],
    "fs_user_x_category": ["new_ranker"],
    "new_ranker": []
}
```

Here `fs_user_x_category` depends on `fs_product` because it needs the category ID extracted from the product entity to resolve the user x category key.

**Step 2:** Configure each component (feature groups, model endpoints, etc.)

**Step 3:** Push the config to etcd — Inferflow picks it up automatically via watchers.

No code changes. No redeployment. The new model is live.

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
