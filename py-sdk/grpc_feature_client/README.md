# grpc_feature_client

[![PyPI version](https://badge.fury.io/py/grpc_feature_client.svg)](https://badge.fury.io/py/grpc_feature_client)
[![Python versions](https://img.shields.io/pypi/pyversions/grpc_feature_client.svg)](https://pypi.org/project/grpc_feature_client/)
[![Build Status](https://github.com/Meesho/BharatMLStack/actions/workflows/py-sdk.yml/badge.svg)](https://github.com/Meesho/BharatMLStack/actions/workflows/py-sdk.yml)
[![Python 3.7+](https://img.shields.io/badge/python-3.7+-blue.svg)](https://www.python.org/downloads/)
[![License](https://img.shields.io/badge/License-BharatMLStack%20BSL%201.1-blue.svg)](https://github.com/Meesho/BharatMLStack/blob/main/LICENSE.md)

High-performance gRPC client for BharatML Stack real-time feature operations with direct API access.

## Installation

```bash
pip install grpc_feature_client
```

## Dependencies

This package depends on:
- **[bharatml_commons](https://pypi.org/project/bharatml_commons/)**: Common utilities and protobuf definitions
- **grpcio>=1.50.0**: gRPC framework
- **grpcio-tools>=1.50.0**: gRPC tools for protobuf

## Features

- **Direct gRPC API**: persist, retrieve, retrieveDecoded operations
- **Go SDK Compatible**: Same authentication and API semantics
- **Batch Processing**: Automatic batching with parallel execution
- **Real-time Focus**: Low-latency feature persistence and retrieval
- **Context Management**: Timeout and metadata handling
- **Connection Pooling**: Efficient connection management

## Quick Start

```python
from grpc_feature_client import GRPCFeatureClient, GRPCClientConfig

# Configure for real-time operations
config = GRPCClientConfig(
    server_address="localhost:50051",
    job_id="realtime-service",
    job_token="api-token"
)

client = GRPCFeatureClient(config)

# Direct API operations
result = client.persist_features(entity_label, keys_schema, feature_groups, data)
features = client.retrieve_decoded_features(entity_label, feature_groups, keys, entity_keys)
```

## API Reference

### GRPCFeatureClient

```python
class GRPCFeatureClient:
    def __init__(self, config: GRPCClientConfig)
    
    def persist_features(
        self,
        entity_label: str,
        keys_schema: List[str],
        feature_group_schemas: List[Dict[str, Any]],
        data_rows: List[Dict[str, Any]],
        timeout: Optional[float] = None
    ) -> Dict[str, Any]
    
    def retrieve_features(
        self,
        entity_label: str,
        feature_groups: List[Dict[str, Any]],
        keys_schema: List[str],
        entity_keys: List[List[str]],
        timeout: Optional[float] = None
    ) -> Dict[str, Any]
    
    def retrieve_decoded_features(
        self,
        entity_label: str,
        feature_groups: List[Dict[str, Any]],
        keys_schema: List[str],
        entity_keys: List[List[str]],
        timeout: Optional[float] = None
    ) -> Dict[str, Any]
```

### GRPCClientConfig

```python
class GRPCClientConfig:
    def __init__(
        self,
        server_address: str,
        job_id: str,
        job_token: str,
        use_tls: bool = False,
        timeout_seconds: float = 30.0,
        metadata: Dict[str, str] = None,
        max_receive_message_length: int = 4 * 1024 * 1024,
        max_send_message_length: int = 4 * 1024 * 1024
    )
```

## Usage Examples

### Persisting Features

```python
from grpc_feature_client import GRPCFeatureClient, GRPCClientConfig

config = GRPCClientConfig(
    server_address="feature-store.example.com:50051",
    job_id="model-inference-service",
    job_token="api-token"
)

client = GRPCFeatureClient(config)

# Persist real-time features
result = client.persist_features(
    entity_label="user_interaction",
    keys_schema=["user_id", "session_id"],
    feature_group_schemas=[{
        "label": "realtime_features",
        "feature_labels": ["click_count", "page_views"]
    }],
    data_rows=[{
        "user_id": "u123",
        "session_id": "s456",
        "click_count": 5,
        "page_views": 3
    }]
)

print(f"Persist result: {result}")
```

### Retrieving Features

```python
# Retrieve features for ML model inference
features = client.retrieve_decoded_features(
    entity_label="user_interaction",
    feature_groups=[{
        "label": "user_features",
        "feature_labels": ["age", "location"]
    }],
    keys_schema=["user_id"],
    entity_keys=[["u123"], ["u456"]]
)

print(f"Retrieved features: {features}")
```

### With Context Management

```python
# Use client with automatic cleanup
with GRPCFeatureClient(config) as client:
    result = client.persist_features(...)
    features = client.retrieve_decoded_features(...)
# Connection automatically closed
```

## When to Use

**Use grpc_feature_client for:**
- 🚀 **Real-time Operations**: Direct persist/retrieve operations
- 🔍 **Interactive Queries**: Low-latency feature lookups
- 🎯 **API Integration**: Service-to-service communication
- 💨 **Single Records**: Persisting individual feature records
- 🔄 **Model Serving**: Feature retrieval for online inference

**Use spark_feature_push_client for:**
- 🔄 **Batch ETL Pipelines**: Scheduled feature computation and publishing
- 📊 **Historical Data Backfill**: Loading historical features into online store
- 🏗️ **Data Engineering**: Spark-based feature transformations
- 📈 **Large Scale Processing**: Processing millions of records efficiently

## Related Packages

This package is part of the BharatML Stack ecosystem:

- **[bharatml_commons](https://pypi.org/project/bharatml_commons/)**: Common utilities and protobuf definitions (required dependency)
- **[spark_feature_push_client](https://pypi.org/project/spark_feature_push_client/)**: Spark-based data pipeline client

## License

Licensed under the BharatMLStack Business Source License 1.1. See [LICENSE](https://github.com/Meesho/BharatMLStack/blob/main/LICENSE.md) for details.

## Contributing

We welcome contributions! Please see our [Contributing Guide](https://github.com/Meesho/BharatMLStack/blob/main/CONTRIBUTION.md) for details. 