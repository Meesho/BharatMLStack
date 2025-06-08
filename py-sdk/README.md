![Build Status](https://github.com/Meesho/BharatMLStack/actions/workflows/py-sdk.yml/badge.svg)

# BharatMLStack Python SDK

A lightweight Python SDK for interacting with [BharatMLStack](https://github.com/Meesho/BharatMLStack) components, specifically the [Online Feature Store](https://github.com/Meesho/BharatMLStack/tree/main/online-feature-store). This SDK provides functionality for feature metadata retrieval, protobuf serialization, and Kafka integration. 🚀 

This SDK helps in pushing ML model's features stored in offline sources (like tables, Cloud storage objects in parquet/delta format, etc) to Online Feature Store, making it easy to integrate your ML pipelines with BharatMLStack infrastructure.

## Key Features

- Feature metadata retrieval
- Protobuf serialization of feature values and produce to Apache Kafka
- Support for features of different various data types:
  - Scalar types (FP32, FP64, Int32, Int64, UInt32, UInt64, String, Bool)
  - Vector types (Vectors of each of the above Scalar Types)
- Kafka integration with configurable settings

## 📥 Installation

### From PyPI (Recommended)

```bash
pip install online-feature-store-py-client==0.1.1
```

### From Source

```bash
git clone https://github.com/Meesho/BharatMLStack.git
cd BharatMLStack/py-sdk
pip install -e .
```

## Prerequisites

- Python 3.8+ (tested on Python 3.8, 3.9, 3.10, 3.11, 3.12)
- (Optional) Apache Spark 3.0+ & spark-sql-kafka for Kafka feature push functionality

## Quick Start

```python
from online_feature_store_py_client import OnlineFeatureStorePyClient

# Initialize client
client = OnlineFeatureStorePyClient(
    features_metadata_source_url="horizon_metadata_url",
    job_id="your_job_id", 
    job_token="your_token"
)

# Get feature details and push to Online Feature Store
feature_details = client.get_features_details()
# ... process your data ...
# client.write_protobuf_df_to_kafka(proto_df, kafka_servers, topic, options)
```

## Usage

### Basic Usage

```python
from online_feature_store_py_client import OnlineFeatureStorePyClient

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

Followng is a simple flow / outline of the steps involved in above example

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
pytest tests/ --cov=src/online_feature_store_py_client --cov-report=html

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

## Support

For support, please:
- 📖 Check the [BharatMLStack documentation](https://github.com/Meesho/BharatMLStack)
- 🐛 [Create an issue](https://github.com/Meesho/BharatMLStack/issues) for bug reports
- 💡 [Start a discussion](https://github.com/Meesho/BharatMLStack/discussions) for questions

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE.md](../LICENSE.md) file for details.
