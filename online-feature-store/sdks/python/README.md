# Online Feature Store Python Client

A lightweight Python client for interacting with [Online Feature Store](https://github.com/Meesho/BharatMLStack/tree/main/online-feature-store). This client provides functionality for feature metadata retrieval, protobuf serialization, and Kafka integration. 🚀 

This client helps in pushing ML model's features stored in offline sources (like tables, Cloud storage objects in parquet/delta format, etc) to Online Feature Store

## Key Features

- Feature metadata retrieval
- Protobuf serialization of feature values and produce to Apache Kafka
- Support for features of different various data types:
  - Scalar types (FP32, FP64, Int32, Int64, UInt32, UInt64, String, Bool)
  - Vector types (Vectors of each of the above Scalar Types)
- Kafka integration with configurable settings

## 📥 Installation

```bash
pip install online-feature-store-py-client==0.1.1
```

## Prerequisites

- Python 3.7+
- (Optional) Apache Spark 3.0+ & spark-sql-kafka for Kafka feature push functionality

## Usage

### Basic Usage

```python
from orion_py_client import OnlineFeatureStorePyClient

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
) = opy_client.get_features_details()
```


### Push Feature Values from Offline sources to online-feature-store  via Spark -> Kafka

#### Supported Offline Sources
1. Table (Hive/Delta)
2. Parquet folder stored in Cloud Storage (AWS/GCS/ADLS)
3. Delta folder stored in Cloud Storage (AWS/GCS/ADLS)


Refer to the [examples](https://github.com/Meesho/orion/tree/main/examples/notebook) for detailed example of how to configure a job and push the feature values

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

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Support

For support, please create an issue
