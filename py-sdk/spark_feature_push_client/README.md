# spark_feature_push_client

[![PyPI version](https://badge.fury.io/py/spark_feature_push_client.svg)](https://badge.fury.io/py/spark_feature_push_client)
[![Python versions](https://img.shields.io/pypi/pyversions/spark_feature_push_client.svg)](https://pypi.org/project/spark_feature_push_client/)
[![Build Status](https://github.com/Meesho/BharatMLStack/actions/workflows/py-sdk.yml/badge.svg)](https://github.com/Meesho/BharatMLStack/actions/workflows/py-sdk.yml)
[![Python 3.7+](https://img.shields.io/badge/python-3.7+-blue.svg)](https://www.python.org/downloads/)
[![License](https://img.shields.io/badge/License-BharatMLStack%20BSL%201.1-blue.svg)](https://github.com/Meesho/BharatMLStack/blob/main/LICENSE.md)

Apache Spark-based client for pushing ML features from offline batch sources to the BharatML Stack Online Feature Store via Kafka. This client is designed for **data pipeline operations** - reading from batch sources and publishing to Kafka for online consumption.

## Installation

```bash
pip install spark_feature_push_client
```

## Dependencies

This package depends on:
- **[bharatml_commons](https://pypi.org/project/bharatml_commons/)**: Common utilities and protobuf definitions
- **PySpark 3.0+**: For distributed data processing

## Architecture Role

```
┌─────────────────┐    ┌──────────────────────┐    ┌─────────────┐    ┌─────────────────┐
│   Batch Sources │───▶│ Spark Feature Push   │───▶│   Kafka     │───▶│ Online Feature  │
│ • Tables        │    │      Client          │    │             │    │     Store       │
│ • Parquet       │    │ • Read & Transform   │    │             │    │                 │
│ • Delta         │    │ • Protobuf Serialize │    │             │    │                 │
│ • S3/GCS/ADLS   │    │ • Batch Processing   │    │             │    │                 │
└─────────────────┘    └──────────────────────┘    └─────────────┘    └─────────────────┘
                                                                                ▲
                                                                                │
                                                                      ┌─────────────────┐
                                                                      │ grpc_feature_   │
                                                                      │ client          │
                                                                      │ (Real-time)     │
                                                                      └─────────────────┘
```

## Features

- **Batch Source Integration**: Read from Tables (Hive/Delta), Parquet, and Delta files on cloud storage
- **Spark Processing**: Leverage Apache Spark for distributed data processing
- **Protobuf Serialization**: Convert feature data to protobuf format using bharatml_commons schemas
- **Kafka Publishing**: Push serialized features to Kafka topics for online consumption
- **Metadata Integration**: Fetch feature schemas and configurations via REST API
- **Data Type Support**: Handle scalar and vector types (strings, numbers, booleans, arrays)
- **Batch Optimization**: Configurable batch sizes for optimal Kafka throughput

## When to Use This Client

**Use spark_feature_push_client for:**
- 🔄 **Batch ETL Pipelines**: Scheduled feature computation and publishing
- 📊 **Historical Data Backfill**: Loading historical features into online store
- 🏗️ **Data Engineering**: Spark-based feature transformations
- 📈 **Large Scale Processing**: Processing millions of records efficiently
- ⚡ **Offline-to-Online**: Bridge between batch and real-time systems

**Use grpc_feature_client for:**
- 🚀 **Real-time Operations**: Direct persist/retrieve operations
- 🔍 **Interactive Queries**: Low-latency feature lookups
- 🎯 **API Integration**: Service-to-service communication
- 💨 **Single Records**: Persisting individual feature records

## Quick Start

```python
from spark_feature_push_client import OnlineFeatureStorePyClient

# Initialize client with metadata source
client = OnlineFeatureStorePyClient(
    features_metadata_source_url="https://api.example.com/metadata",
    job_id="feature-pipeline-job",
    job_token="your-auth-token"
)

# Get feature configuration 
feature_details = client.get_features_details()

# Process your Spark DataFrame
proto_df = client.generate_df_with_protobuf_messages(your_spark_df)

# Push to Kafka
client.write_protobuf_df_to_kafka(
    proto_df,
    kafka_bootstrap_servers="localhost:9092",
    kafka_topic="features.user_features"
)
```

## Related Packages

This package is part of the BharatML Stack ecosystem:

- **[bharatml_commons](https://pypi.org/project/bharatml_commons/)**: Common utilities and protobuf definitions (required dependency)
- **[grpc_feature_client](https://pypi.org/project/grpc_feature_client/)**: High-performance gRPC client for real-time operations

## License

Licensed under the BharatMLStack Business Source License 1.1. See [LICENSE](https://github.com/Meesho/BharatMLStack/blob/main/LICENSE.md) for details.

## Contributing

We welcome contributions! Please see our [Contributing Guide](https://github.com/Meesho/BharatMLStack/blob/main/CONTRIBUTION.md) for details.

## Prerequisites

- **Apache Spark 3.0+**: For distributed processing
- **Kafka Connector**: `spark-sql-kafka` for Kafka integration
- **Java 8/11**: Required by Spark
- **bharatml_common**: For protobuf schemas

```bash
# Example Spark session setup
spark = SparkSession.builder \
    .appName("FeaturePipeline") \
    .config("spark.jars.packages", "org.apache.spark:spark-sql-kafka-0-10_2.12:3.4.0") \
    .getOrCreate()
```

## Supported Data Sources

### 1. Database Tables
```python
# Hive/Delta tables
df = spark.sql("SELECT * FROM feature_db.user_features")
```

### 2. Cloud Storage - Parquet
```python
# AWS S3
df = spark.read.parquet("s3a://bucket/path/to/features/")

# Google Cloud Storage  
df = spark.read.parquet("gs://bucket/path/to/features/")

# Azure Data Lake
df = spark.read.parquet("abfss://container@account.dfs.core.windows.net/path/")
```

### 3. Cloud Storage - Delta
```python
# Delta format on cloud storage
df = spark.read.format("delta").load("s3a://bucket/delta-table/")
```

## Configuration Examples

### Basic Pipeline
```python
from pyspark.sql import SparkSession
from spark_feature_push_client import OnlineFeatureStorePyClient

# Create Spark session
spark = SparkSession.builder \
    .appName("FeatureETL") \
    .config("spark.jars.packages", "org.apache.spark:spark-sql-kafka-0-10_2.12:3.4.0") \
    .getOrCreate()

# Initialize client
client = OnlineFeatureStorePyClient(
    features_metadata_source_url="https://metadata-service.example.com/api/v1/features",
    job_id="daily-feature-pipeline",
    job_token="pipeline-secret-token",
    fgs_to_consider=["user_demographics", "user_behavior"]  # Optional: filter feature groups
)

# Get metadata and column mappings
(
    offline_src_type_columns,
    offline_col_to_default_values_map, 
    entity_column_names
) = client.get_features_details()

print(f"Entity columns: {entity_column_names}")
print(f"Feature mappings: {offline_src_type_columns}")
```

### Reading from Multiple Sources
```python
def get_features_from_all_sources(spark, entity_columns, feature_mapping, default_values):
    """
    Read and combine features from multiple offline sources
    """
    dataframes = []
    
    for source_info in feature_mapping:
        table_name, source_type, feature_list = source_info
        
        if source_type == "TABLE":
            # Read from Hive/Delta table
            df = spark.table(table_name)
            
        elif source_type.startswith("PARQUET_"):
            # Read from Parquet files
            df = spark.read.parquet(table_name)
            
        elif source_type.startswith("DELTA_"):
            # Read from Delta files
            df = spark.read.format("delta").load(table_name)
        
        # Select and rename columns
        select_cols = entity_columns.copy()
        for original_col, renamed_col in feature_list:
            if original_col in df.columns:
                df = df.withColumnRenamed(original_col, renamed_col)
                select_cols.append(renamed_col)
        
        df = df.select(select_cols)
        dataframes.append(df)
    
    # Union all dataframes
    if dataframes:
        combined_df = dataframes[0]
        for df in dataframes[1:]:
            combined_df = combined_df.unionByName(df, allowMissingColumns=True)
        
        # Fill missing values with defaults
        for col, default_val in default_values.items():
            if col in combined_df.columns:
                combined_df = combined_df.fillna({col: default_val})
        
        return combined_df
    
    return None

# Use the function
df = get_features_from_all_sources(
    spark, 
    entity_column_names, 
    offline_src_type_columns, 
    offline_col_to_default_values_map
)
```

### Protobuf Serialization & Kafka Publishing
```python
# Convert DataFrame to protobuf messages
# This creates binary protobuf messages suitable for Kafka
proto_df = client.generate_df_with_protobuf_messages(
    df, 
    intra_batch_size=20  # Batch size for serialization
)

# The proto_df has schema: [value: binary, intra_batch_id: long]
proto_df.printSchema()
# root
#  |-- value: binary (nullable = false)  
#  |-- intra_batch_id: long (nullable = false)

# Write to Kafka with batching for better throughput
client.write_protobuf_df_to_kafka(
    proto_df,
    kafka_bootstrap_servers="broker1:9092,broker2:9092,broker3:9092",
    kafka_topic="features.user_features",
    additional_options={
        "kafka.acks": "all",
        "kafka.retries": "3",
        "kafka.compression.type": "snappy"
    },
    kafka_num_batches=4  # Split into 4 parallel Kafka writes
)
```

## Data Type Handling

The client automatically handles the protobuf data type mappings:

### Scalar Types
```python
# Example DataFrame with different types
data = [
    ("user123", 25, 185.5, True, "premium"),      # int, float, bool, string
    ("user456", 30, 170.0, False, "basic")
]
df = spark.createDataFrame(data, ["user_id", "age", "height", "is_premium", "tier"])

# Automatically mapped to protobuf:
# age -> int32_values
# height -> fp32_values  
# is_premium -> bool_values
# tier -> string_values
```

### Vector Types
```python
# Example with vector/array features
from pyspark.sql.functions import array, lit

df = spark.createDataFrame([
    ("user123", [0.1, 0.2, 0.3], ["tech", "sports"], [1, 2, 3])
], ["user_id", "embeddings", "interests", "scores"])

# Automatically mapped to protobuf vectors:
# embeddings -> fp32_values in Vector
# interests -> string_values in Vector
# scores -> int32_values in Vector
```

## Production Pipeline Example

```python
def run_feature_pipeline():
    """
    Complete feature pipeline from batch sources to Kafka
    """
    
    # 1. Initialize Spark
    spark = SparkSession.builder \
        .appName("DailyFeaturePipeline") \
        .config("spark.sql.adaptive.enabled", "true") \
        .config("spark.jars.packages", "org.apache.spark:spark-sql-kafka-0-10_2.12:3.4.0") \
        .getOrCreate()
    
    try:
        # 2. Initialize feature client
        client = OnlineFeatureStorePyClient(
            features_metadata_source_url=os.getenv("METADATA_URL"),
            job_id=os.getenv("JOB_ID"),
            job_token=os.getenv("JOB_TOKEN")
        )
        
        # 3. Get feature configuration
        feature_mapping, default_values, entity_columns = client.get_features_details()
        
        # 4. Read and process data
        df = get_features_from_all_sources(spark, entity_columns, feature_mapping, default_values)
        
        if df is None or df.count() == 0:
            raise ValueError("No data found in sources")
        
        # 5. Convert to protobuf
        proto_df = client.generate_df_with_protobuf_messages(df, intra_batch_size=50)
        
        # 6. Publish to Kafka
        client.write_protobuf_df_to_kafka(
            proto_df,
            kafka_bootstrap_servers=os.getenv("KAFKA_BROKERS"),
            kafka_topic=os.getenv("KAFKA_TOPIC"),
            additional_options={
                "kafka.acks": "all",
                "kafka.compression.type": "snappy",
                "kafka.max.request.size": "10485760"  # 10MB
            },
            kafka_num_batches=int(os.getenv("KAFKA_BATCHES", "4"))
        )
        
        print(f"✅ Successfully processed {df.count()} records")
        
    finally:
        spark.stop()

if __name__ == "__main__":
    run_feature_pipeline()
```

## Configuration Options

### Client Configuration
```python
client = OnlineFeatureStorePyClient(
    features_metadata_source_url="https://api.example.com/metadata",  # Required
    job_id="pipeline-job-001",                                       # Required  
    job_token="secret-token-123",                                    # Required
    fgs_to_consider=["user_features", "item_features"]               # Optional: filter feature groups
)
```

### Protobuf Serialization Options
```python
proto_df = client.generate_df_with_protobuf_messages(
    df,
    intra_batch_size=20  # Records per protobuf message (default: 20)
)
```

### Kafka Publishing Options
```python
client.write_protobuf_df_to_kafka(
    proto_df,
    kafka_bootstrap_servers="localhost:9092",
    kafka_topic="features.topic",
    additional_options={
        "kafka.acks": "all",                    # Acknowledgment level
        "kafka.retries": "3",                   # Retry attempts
        "kafka.compression.type": "snappy",     # Compression
        "kafka.batch.size": "16384",            # Batch size
        "kafka.linger.ms": "100",               # Batching delay
        "kafka.max.request.size": "10485760"    # Max message size
    },
    kafka_num_batches=1  # Number of parallel Kafka writers (default: 1)
)
```

## Performance Tuning

### Spark Optimizations
```python
spark = SparkSession.builder \
    .appName("FeaturePipeline") \
    .config("spark.sql.adaptive.enabled", "true") \
    .config("spark.sql.adaptive.coalescePartitions.enabled", "true") \
    .config("spark.sql.adaptive.skewJoin.enabled", "true") \
    .config("spark.serializer", "org.apache.spark.serializer.KryoSerializer") \
    .getOrCreate()
```

### Memory Management
```python
# For large datasets, consider:
df = df.repartition(200)  # Optimal partition count
df.cache()  # Cache if reused multiple times
```

### Kafka Throughput
```python
# For high-throughput scenarios:
client.write_protobuf_df_to_kafka(
    proto_df,
    kafka_bootstrap_servers="brokers",
    kafka_topic="topic", 
    kafka_num_batches=8,  # Increase parallel writers
    additional_options={
        "kafka.batch.size": "65536",      # Larger batches
        "kafka.linger.ms": "100",         # Allow batching delay
        "kafka.compression.type": "lz4"   # Fast compression
    }
)
```

## Monitoring & Debugging

### DataFrame Inspection
```python
# Check data before processing
print(f"Records: {df.count()}")
print(f"Columns: {df.columns}")
df.printSchema()
df.show(5)

# Check protobuf output
proto_df.show(5, truncate=False)
print(f"Protobuf messages: {proto_df.count()}")
```

### Error Handling
```python
try:
    proto_df = client.generate_df_with_protobuf_messages(df)
    client.write_protobuf_df_to_kafka(proto_df, brokers, topic)
    
except Exception as e:
    print(f"Pipeline failed: {e}")
    # Log to monitoring system
    # Send alerts
    raise
```

## Integration with Other SDKs

### With gRPC Feature Client
```python
# Spark client pushes features to Kafka
spark_client = OnlineFeatureStorePyClient(...)
spark_client.write_protobuf_df_to_kafka(proto_df, brokers, topic)

# gRPC client retrieves features in real-time
from grpc_feature_client import GRPCFeatureClient
grpc_client = GRPCFeatureClient(config)
features = grpc_client.retrieve_decoded_features(...)
```

### With HTTP Feature Client (bharatml_common)
```python
# Use HTTP client for metadata validation
from bharatml_common import HTTPFeatureClient
http_client = HTTPFeatureClient(base_url, job_id, token)
metadata = http_client.get_feature_metadata()

# Validate feature names using shared utilities
from bharatml_common import clean_column_name
clean_features = [clean_column_name(name) for name in feature_names]

# Process with Spark client
spark_client.generate_df_with_protobuf_messages(df)
```

## Common Use Cases

### 1. Daily Batch ETL
```bash
# Cron job: 0 2 * * * (daily at 2 AM)
spark-submit \
  --packages org.apache.spark:spark-sql-kafka-0-10_2.12:3.4.0 \
  --conf spark.sql.adaptive.enabled=true \
  daily_feature_pipeline.py
```

### 2. Historical Backfill
```python
# Backfill last 30 days
from datetime import datetime, timedelta

for i in range(30):
    date = datetime.now() - timedelta(days=i)
    df = spark.sql(f"""
        SELECT * FROM features 
        WHERE date = '{date.strftime('%Y-%m-%d')}'
    """)
    
    proto_df = client.generate_df_with_protobuf_messages(df)
    client.write_protobuf_df_to_kafka(proto_df, brokers, f"backfill.{date.strftime('%Y%m%d')}")
```

### 3. Real-time Streaming (Advanced)
```python
# Read from streaming source, process, and publish
streaming_df = spark.readStream \
    .format("kafka") \
    .option("kafka.bootstrap.servers", input_brokers) \
    .option("subscribe", input_topic) \
    .load()

# Process streaming DataFrame
processed_df = streaming_df.select(...)

# Write to output Kafka (requires structured streaming)
query = processed_df.writeStream \
    .format("kafka") \
    .option("kafka.bootstrap.servers", output_brokers) \
    .option("topic", output_topic) \
    .start()
```

## Troubleshooting

### Common Issues

1. **OutOfMemoryError**
   ```python
   # Increase driver memory or reduce partition size
   spark.conf.set("spark.sql.adaptive.coalescePartitions.minPartitionNum", "50")
   ```

2. **Kafka Connection Timeout**
   ```python
   # Check network connectivity and broker addresses
   additional_options = {
       "kafka.request.timeout.ms": "60000",
       "kafka.session.timeout.ms": "30000"
   }
   ```

3. **Protobuf Serialization Errors**
   ```python
   # Check data types and null values
   df = df.fillna({"string_col": "", "numeric_col": 0})
   ```

4. **Metadata API Errors**
   ```python
   # Verify job_id, job_token, and URL
   # Check API server logs
   ```

### Debug Mode
```python
import logging
logging.basicConfig(level=logging.DEBUG)

# Enable Spark SQL logging
spark.sparkContext.setLogLevel("INFO")
```

## Migration from Legacy Clients

If migrating from older versions:

```python
# Old import
# from online_feature_store_py_client import OnlineFeatureStorePyClient

# New import (same interface)
from spark_feature_push_client import OnlineFeatureStorePyClient

# API remains the same - no code changes needed!
```

## Best Practices

1. **Resource Management**: Always stop Spark sessions
2. **Error Handling**: Implement proper exception handling and retries
3. **Monitoring**: Add metrics and logging to your pipelines
4. **Testing**: Test with sample data before production runs
5. **Security**: Use secure Kafka configurations in production
6. **Performance**: Monitor Spark UI for optimization opportunities

The Spark Feature Push Client is your gateway from batch data sources to the real-time online feature store! 🚀 