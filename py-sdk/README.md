# BharatML Stack Python SDKs

![Build Status](https://github.com/Meesho/BharatMLStack/actions/workflows/py-sdk.yml/badge.svg)
[![Discord](https://img.shields.io/badge/Discord-Join%20Chat-7289da?style=flat&logo=discord&logoColor=white)](https://discord.gg/XkT7XsV2AU)

A collection of independent Python packages for interacting with [BharatMLStack](https://github.com/Meesho/BharatMLStack) components. 🚀 

## 📦 Separated Packages

The BharatML Stack Python SDK has been **separated into 3 independent packages** for better modularity and focused dependencies:

| Package | Purpose | PyPI |
|---------|---------|------|
| **[bharatml_commons](https://pypi.org/project/bharatml_commons/)** | Common utilities and protobuf definitions | [![PyPI package](https://img.shields.io/pypi/v/bharatml_commons?label=pypi-package&color=light%20green)](https://pypi.org/project/bharatml_commons/) |
| **[spark_feature_push_client](https://pypi.org/project/spark_feature_push_client/)** | Apache Spark-based data pipeline client | [![PyPI](https://img.shields.io/pypi/v/spark_feature_push_client?label=pypi-package&color=light%20green)](https://pypi.org/project/spark_feature_push_client/) |
| **[grpc_feature_client](https://pypi.org/project/grpc_feature_client/)** | High-performance gRPC client for real-time operations | [![PyPI](https://img.shields.io/pypi/v/grpc_feature_client?label=pypi-package&color=light%20green)](https://pypi.org/project/grpc_feature_client/) |

## Key Features

- Feature metadata retrieval
- Protobuf serialization of feature values and produce to Apache Kafka
- Support for features of different various data types:
  - Scalar types (FP32, FP64, Int32, Int64, UInt32, UInt64, String, Bool)
  - Vector types (Vectors of each of the above Scalar Types)
- Kafka integration with configurable settings

## 🚀 Quick Installation

**Install all packages:**
```bash
pip install bharatml_commons spark_feature_push_client grpc_feature_client
```

**Install only what you need:**
```bash
# For Spark-based data pipelines
pip install bharatml_commons spark_feature_push_client

# For real-time gRPC operations
pip install bharatml_commons grpc_feature_client

# For common utilities only
pip install bharatml_commons
```

## 📖 Documentation

- **[bharatml_commons README](./bharatml_commons/README.md)**: Common utilities and protobuf definitions
- **[spark_feature_push_client README](./spark_feature_push_client/README.md)**: Spark-based data pipeline client
- **[grpc_feature_client README](./grpc_feature_client/README.md)**: High-performance gRPC client  


## Prerequisites

- Python 3.7+ (tested on Python 3.7, 3.8, 3.9, 3.10, 3.11, 3.12)
- (Optional) Apache Spark 3.0+ & spark-sql-kafka for Spark-based functionality

## Quick Start

### 1. bharatml_commons

**Foundation package** with shared utilities, protobuf definitions, and base classes.

```python
from bharatml_commons import FeatureMetadataClient, clean_column_name
from bharatml_commons.proto.persist.persist_pb2 import Query

# HTTP client for metadata operations
client = FeatureMetadataClient(url, job_id, token)
metadata = client.get_feature_metadata(["user_features"])

# Utility functions
clean_name = clean_column_name("feature@name#1")
```

### 2. spark_feature_push_client

**Apache Spark client** for batch data pipelines - reading from data sources and pushing to Kafka.

```python
from spark_feature_push_client import OnlineFeatureStorePyClient

# Initialize client for data pipeline
client = OnlineFeatureStorePyClient(metadata_url, job_id, job_token)

# Process Spark DataFrame → Protobuf → Kafka
proto_df = client.generate_df_with_protobuf_messages(spark_df)
client.write_protobuf_df_to_kafka(proto_df, kafka_servers, topic)
```

### 3. grpc_feature_client

**High-performance gRPC client** for real-time feature operations with direct API access.

```python
from grpc_feature_client import GRPCFeatureClient, GRPCClientConfig

# Configure for real-time operations
config = GRPCClientConfig(server_address, job_id, job_token)
client = GRPCFeatureClient(config)

# Direct API operations
result = client.persist_features(entity_label, keys_schema, feature_groups, data)
features = client.retrieve_decoded_features(entity_label, feature_groups, keys, entity_keys)
```

## Usage

### Spark Feature Push Client Usage

```python
from spark_feature_push_client import OnlineFeatureStorePyClient

# Initialize the client
client = OnlineFeatureStorePyClient(
    features_metadata_source_url="your_features_metadata_source_url",
    job_id="your_job_id",
    job_token="your_job_token"
)

# Get feature details
(
    offline_src_type_columns,
    offline_col_to_default_values_map,
    entity_column_names
) = client.get_features_details()
```


### Push Feature Values from Offline sources to online-feature-store  via Spark -> Kafka

#### Supported Offline Sources
1. Table (Hive/Delta)
2. Parquet folder stored in Cloud Storage (AWS/GCS/ADLS)
3. Delta folder stored in Cloud Storage (AWS/GCS/ADLS)


Refer to the [examples](https://github.com/Meesho/BharatMLStack/tree/main/online-feature-store/examples/notebook) for detailed examples of how to configure a job and push the feature values

Following is a simple flow / outline of the steps involved in above example:

```python
# create a new onlineFeatureStore client
opy_client = OnlineFeatureStorePyClient(features_metadata_source_url, job_id, job_token) 

# get the features details
feature_mapping, offline_col_to_default_values_map, onfs_fg_to_onfs_feat_map, onfs_fg_to_ofs_feat_map, fg_to_datatype_map, entity_label, entity_column_names = opy_client.get_features_details(fgs_to_consider)

# read the data from different sources
df = get_features_from_all_sources(spark, entity_column_names, feature_mapping, offline_col_to_default_values_map)

# serialize of protobuf binary
proto_df = opy_client.generate_df_with_protobuf_messages(df, intra_batch_size=20) 

# Produce data to kafka so that consumers write features to Online Feature Store
opy_client.write_protobuf_df_to_kafka(proto_df, kafka_bootstrap_servers, kafka_topic, additional_options)
```

## 🏗️ SDK Architecture

The multi-SDK architecture provides:

```
py-sdk/
├── src/
│   ├── spark_feature_push_client/    # Spark-based data pipeline
│   │   ├── utils/helpers.py          # Spark-specific utilities
│   │   ├── __init__.py
│   │   └── client.py                 # Batch ETL operations
│   ├── grpc_feature_client/          # gRPC real-time operations
│   │   ├── config.py                 # gRPC configuration
│   │   ├── client.py                 # Real-time API operations
│   │   ├── README.md                 # gRPC documentation
│   │   └── __init__.py
│   ├── bharatml_common/              # Shared utilities & protobuf
│   │   ├── proto/                    # ✅ Protobuf definitions
│   │   │   ├── persist.proto         # Persist operation schema
│   │   │   ├── retrieve.proto        # Retrieve operation schema
│   │   │   ├── persist/persist_pb2.py # Generated Python files
│   │   │   ├── retrieve/retrieve_pb2.py
│   │   │   └── generate_proto.py     # Code generation script
│   │   ├── http_client.py            # HTTP client utilities
│   │   ├── feature_metadata_client.py # ✅ Feature metadata client
│   │   ├── column_utils.py           # Column processing
│   │   ├── feature_utils.py          # Feature processing
│   │   ├── sdk_template.py           # Template for new SDKs
│   │   └── __init__.py
├── README.md
└── pyproject.toml
```

## 🛠️ Creating New SDKs

To create a new SDK in this project:

1. **Create the SDK directory structure:**
   ```
   src/your_new_sdk/
   ├── __init__.py           # Main exports
   ├── client.py             # Main client class
   ├── config.py             # Configuration classes
   └── utils/                # SDK-specific utilities
   ```

2. **Use shared utilities:**
   ```python
   from bharatml_common.http_client import BharatMLHTTPClient
   from bharatml_common.sdk_template import BaseSDKClient
   ```

3. **Update pyproject.toml:**
   ```toml
   [tool.hatch.build.targets.wheel]
   packages = [
       "src/spark_feature_push_client",
       "src/bharatml_common",
       "src/your_new_sdk"  # Add your new SDK
   ]
   ```

## Development

### Setting up Development Environment

```bash
# Clone the repository
git clone https://github.com/Meesho/BharatMLStack.git
cd BharatMLStack/py-sdk

# Install in development mode
pip install -e .

# Install development dependencies
pip install build pytest flake8 black isort mypy
```

### Running Tests

```bash
# Run all tests
pytest tests/ -v

# Run tests with coverage
pytest tests/ --cov=src --cov-report=html

# Run specific test file
pytest tests/test_client.py -v
```

### Code Quality

```bash
# Format code with black
black src/

# Sort imports with isort
isort src/

# Lint with flake8
flake8 src/

# Type checking with mypy
mypy src/ --ignore-missing-imports
```

### Building the Package

```bash
# Build the package
python -m build

# Check package metadata
pip install twine
twine check dist/*
```

### Code Style Guidelines

- Follow [PEP 8](https://pep8.org/) style guidelines
- Use [Black](https://black.readthedocs.io/) for code formatting
- Use [isort](https://pycqa.github.io/isort/) for import sorting
- Add type hints where possible
- Write docstrings for public functions and classes
- Keep line length to 88 characters (Black default)

## Complete Architecture Example

Here's how the Spark and gRPC clients work together in a complete ML feature pipeline:

```python
# 1. BATCH PIPELINE (Daily ETL Job)
from spark_feature_push_client import OnlineFeatureStorePyClient

# Process batch data with Spark
spark_client = OnlineFeatureStorePyClient(
    features_metadata_source_url="https://metadata.example.com",
    job_id="daily-batch-etl",
    job_token="pipeline-token"
)

# Read from data warehouse, transform, and push to Kafka
feature_details = spark_client.get_features_details()
historical_df = spark.sql("SELECT * FROM feature_warehouse.user_features")
proto_df = spark_client.generate_df_with_protobuf_messages(historical_df)
spark_client.write_protobuf_df_to_kafka(proto_df, kafka_brokers, "features.batch")

# 2. REAL-TIME SERVICE (Model Inference API)
from grpc_feature_client import GRPCFeatureClient, GRPCClientConfig

# Configure gRPC client for real-time operations
grpc_config = GRPCClientConfig(
    server_address="feature-store.example.com:50051",
    job_id="predator",
    job_token="api-token"
)

grpc_client = GRPCFeatureClient(grpc_config)

# Persist real-time features from user interactions
grpc_client.persist_features(
    entity_label="user_interaction",
    keys_schema=["user_id", "session_id"],
    feature_group_schemas=[{"label": "realtime_features", "feature_labels": ["click_count", "page_views"]}],
    data_rows=[{"user_id": "u123", "session_id": "s456", "click_count": 5, "page_views": 3}]
)

# Retrieve features for ML model inference
features = grpc_client.retrieve_decoded_features(
    entity_label="user_interaction", 
    feature_groups=[{"label": "user_features", "feature_labels": ["age", "location"]}],
    keys_schema=["user_id"],
    entity_keys=[["u123"], ["u456"]]
)

# Use features in ML model
model_input = prepare_features(features)
prediction = ml_model.predict(model_input)

# 3. FEATURE METADATA CLIENT (For REST API access)
from bharatml_common import FeatureMetadataClient

# Use metadata client for feature metadata operations
metadata_client = FeatureMetadataClient("https://api.example.com", "http-job", "http-token")
metadata = metadata_client.get_feature_metadata(["user_features"])
health = metadata_client.health_check()
```

## 🎯 When to Use Which Package

| Use Case | Package | Why |
|----------|---------|-----|
| **Daily ETL Jobs** | `spark_feature_push_client` | Distributed processing, handles large datasets efficiently |
| **Historical Backfill** | `spark_feature_push_client` | Batch processing from data warehouses/lakes |
| **Real-time Inference** | `grpc_feature_client` | Low latency, direct API access |
| **Feature Store Updates** | `grpc_feature_client` | Direct persist/retrieve operations |
| **Model Training** | `spark_feature_push_client` | Process training datasets at scale |
| **Model Serving** | `grpc_feature_client` | Real-time feature retrieval for predictions |
| **Metadata Operations** | `bharatml_commons` | HTTP-based metadata queries |
| **Utility Functions** | `bharatml_commons` | Column cleaning, feature processing |

## 📖 Documentation

- **[bharatml_commons README](./bharatml_commons/README.md)**: Common utilities and protobuf definitions
- **[spark_feature_push_client README](./spark_feature_push_client/README.md)**: Spark-based data pipeline client
- **[grpc_feature_client README](./grpc_feature_client/README.md)**: High-performance gRPC client
- **[Migration Guide](./MIGRATION_GUIDE.md)**: Migrating from the old unified package

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](https://github.com/Meesho/BharatMLStack/blob/main/CONTRIBUTION.md) for details.

## 📄 License

Licensed under the BharatMLStack Business Source License 1.1. See [LICENSE](https://github.com/Meesho/BharatMLStack/blob/main/LICENSE.md) for details.

## 🔗 Links

- **[BharatML Stack](https://github.com/Meesho/BharatMLStack)**: Main repository
- **[PyPI Packages](https://pypi.org/search/?q=bharatml)**: All published packages
- **[Issues](https://github.com/Meesho/BharatMLStack/issues)**: Bug reports and feature requests
- **[Discussions](https://github.com/Meesho/BharatMLStack/discussions)**: Community discussions



---

<div align="center">
  <strong>Built with ❤️ for the ML community from Meesho</strong>
</div>
<div align="center">
  <strong>If you find this useful, ⭐️ the repo — your support means the world to us!</strong>
</div>
