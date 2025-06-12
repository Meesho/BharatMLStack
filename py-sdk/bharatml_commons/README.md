# bharatml_commons

[![PyPI version](https://badge.fury.io/py/bharatml_commons.svg)](https://badge.fury.io/py/bharatml_commons)
[![Python versions](https://img.shields.io/pypi/pyversions/bharatml_commons.svg)](https://pypi.org/project/bharatml_commons/)
[![Build Status](https://github.com/Meesho/BharatMLStack/actions/workflows/py-sdk.yml/badge.svg)](https://github.com/Meesho/BharatMLStack/actions/workflows/py-sdk.yml)
[![Python 3.7+](https://img.shields.io/badge/python-3.7+-blue.svg)](https://www.python.org/downloads/)
[![License](https://img.shields.io/badge/License-BharatMLStack%20BSL%201.1-blue.svg)](https://github.com/Meesho/BharatMLStack/blob/main/LICENSE.md)

Common utilities and protobuf definitions for BharatML Stack.

## Overview

`bharatml_commons` provides shared utilities, protobuf definitions, and base classes used across all BharatML Stack Python SDKs. This package serves as the foundation for other BharatML Stack components.

## Features

- **Protobuf Schemas**: Shared persist.proto and retrieve.proto definitions
- **HTTP Clients**: REST API client utilities with authentication
- **Feature Metadata Client**: Complete REST API client for metadata operations
- **Data Processing**: Column and feature processing utilities  
- **Base SDK Classes**: Consistent API design across all clients
- **Code Generation**: Scripts for protobuf Python file generation

## Installation

```bash
pip install bharatml_commons
```

## Quick Start

### Feature Metadata Client

```python
from bharatml_commons import FeatureMetadataClient

# Initialize client
client = FeatureMetadataClient(
    metadata_url="https://api.example.com",
    job_id="my-job",
    job_token="my-token"
)

# Get feature metadata
metadata = client.get_feature_metadata(["user_features"])
features = client.get_features({"user_id": "123"}, ["user_features"])
```

### HTTP Client Utilities

```python
from bharatml_commons import BharatMLHTTPClient

# Low-level HTTP client
client = BharatMLHTTPClient()
response = client.get("https://api.example.com/health")
```

### Column and Feature Utilities

```python
from bharatml_commons import clean_column_name, get_fgs_to_feature_mappings

# Clean column names
clean_name = clean_column_name("feature@name#1")
# Output: "feature_name_1"

# Process feature mappings
mappings = get_fgs_to_feature_mappings(feature_groups_data)
```

### Protobuf Definitions

```python
from bharatml_commons.proto.persist.persist_pb2 import Query, Data
from bharatml_commons.proto.retrieve.retrieve_pb2 import RetrieveQuery

# Use protobuf messages
query = Query(entity_label="user", keys_schema=["user_id"])
```

## API Reference

### FeatureMetadataClient

The main client for interacting with feature metadata APIs.

```python
class FeatureMetadataClient:
    def __init__(self, metadata_url: str, job_id: str, job_token: str)
    def get_feature_metadata(self, feature_groups: List[str]) -> Dict
    def get_features(self, entity_keys: Dict, feature_groups: List[str]) -> Dict
    def health_check() -> Dict
```

### Utility Functions

- `clean_column_name(name: str) -> str`: Clean and normalize column names
- `generate_renamed_column(old_name: str, new_name: str) -> str`: Generate renamed columns
- `get_fgs_to_feature_mappings(data: Dict) -> Dict`: Process feature group mappings
- `extract_entity_info(metadata: Dict) -> Tuple`: Extract entity information

### HTTP Client

```python
class BharatMLHTTPClient:
    def get(self, url: str, headers: Dict = None) -> Dict
    def post(self, url: str, data: Dict, headers: Dict = None) -> Dict
    def put(self, url: str, data: Dict, headers: Dict = None) -> Dict
    def delete(self, url: str, headers: Dict = None) -> Dict
```

## Development

### Setting up Development Environment

```bash
git clone https://github.com/Meesho/BharatMLStack.git
cd BharatMLStack/py-sdk/bharatml_commons
pip install -e .
```

### Running Tests

```bash
pytest tests/ -v
```

### Generating Protobuf Files

```bash
python -m bharatml_commons.proto.generate_proto
```

## Related Packages

This package is part of the BharatML Stack ecosystem:

- **[spark_feature_push_client](https://pypi.org/project/spark_feature_push_client/)**: Spark-based data pipeline client
- **[grpc_feature_client](https://pypi.org/project/grpc_feature_client/)**: High-performance gRPC client for real-time operations

## License

Licensed under the BharatMLStack Business Source License 1.1. See [LICENSE](https://github.com/Meesho/BharatMLStack/blob/main/LICENSE.md) for details.

## Contributing

We welcome contributions! Please see our [Contributing Guide](https://github.com/Meesho/BharatMLStack/blob/main/CONTRIBUTION.md) for details. 