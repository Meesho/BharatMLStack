# Inference Logging Client

A Python SDK for decoding MPLog feature logs from proto, arrow, or parquet format. This client enables you to decode binary-encoded feature data from machine learning inference logging pipelines into Spark DataFrames.

It also reads framed `.log` files produced by `asyncloguploader` — directly from a local path, an open file-like object, or a `gs://` GCS URI — and decodes every embedded MPLog record in one call.

---

## Table of Contents

- [Overview](#overview)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Core API Reference](#core-api-reference)
  - [decode_mplog()](#decode_mplog)
  - [decode_mplog_dataframe()](#decode_mplog_dataframe)
  - [Reading .log Files](#log-api-reference)
  - [get_mplog_metadata()](#get_mplog_metadata)
  - [get_feature_schema()](#get_feature_schema)
  - [clear_schema_cache()](#clear_schema_cache)
- [Data Types](#data-types)
  - [Format Enum](#format-enum)
  - [FeatureInfo](#featureinfo)
  - [DecodedMPLog](#decodedmplog)
- [Supported Feature Types](#supported-feature-types)
- [Encoding Formats Explained](#encoding-formats-explained)
- [Exception Handling](#exception-handling)
- [Command Line Interface](#command-line-interface)
- [Advanced Usage Examples](#advanced-usage-examples)
- [Architecture & Internals](#architecture--internals)
- [Troubleshooting](#troubleshooting)
- [Development](#development)

---

## Overview

The Inference Logging Client is designed to decode MPLog (Model Proxy Log) feature data that has been encoded for efficient storage and transmission. It supports three encoding formats:

| Format | Description | Use Case |
|--------|-------------|----------|
| **Proto** | Custom binary encoding with generated flag + sequential features | Default, most compact |
| **Arrow** | Arrow IPC format with binary columns | Columnar analytics |
| **Parquet** | Parquet format with feature map | Long-term storage |

In addition, the SDK reads the **framed `.log` container** emitted by `asyncloguploader`. Each `.log` file holds many MPLog records — the SDK deframes the container and decodes every embedded record (one row per entity) in a single call. See [Reading .log files from GCS or local disk](#reading-log-files-from-gcs-or-local-disk).

### Key Features

- **Multi-format support**: Decode Proto, Arrow, and Parquet encoded logs
- **Automatic format detection**: Detects encoding format from metadata byte
- **Zstd compression support**: Automatic decompression of zstd-compressed data
- **Schema fetching**: Retrieves feature schemas from inference API with caching
- **Spark integration**: Returns data as PySpark DataFrames
- **`.log` container reader**: Local path, file-like, or `gs://` URI — deframes and decodes in one call
- **CLI tool**: Command-line interface for quick decoding (auto-routes `.log` and `gs://` inputs)
- **Thread-safe caching**: LRU cache for schemas with thread-safe access

---

## Installation

### From PyPI

```bash
pip install inference-logging-client
```

### From Source

```bash
cd py-sdk/inference_logging_client
pip install -e .
```

### With Development Dependencies

```bash
pip install -e ".[dev]"
```

### With GCS Support (for `gs://` .log inputs)

```bash
pip install "inference-logging-client[gcs]"
# or from source:
pip install -e ".[gcs]"
```

### Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| pyspark | ==3.3.0 | Spark DataFrame operations (exact pin) |
| pyarrow | >=5.0.0 | Arrow/Parquet format support |
| zstandard | >=0.15.0 | Zstd decompression |
| google-cloud-storage | >=2.0.0 | **Optional** — only required to read `gs://` URIs |

---

## Quick Start

### Basic Decoding from Bytes

```python
from pyspark.sql import SparkSession
import inference_logging_client

# Create SparkSession
spark = SparkSession.builder \
    .appName("inference-decode") \
    .getOrCreate()

# Read binary MPLog data
with open("inference_log.bin", "rb") as f:
    data = f.read()

# Decode to Spark DataFrame
df = inference_logging_client.decode_mplog(
    log_data=data,
    model_proxy_id="product-ranking-model",
    version=1,
    spark=spark
)

# View the results
df.show()
#    entity_id  feature_price  feature_category  embedding_vector
# 0  prod_123          29.99                 5  [0.1, 0.2, ...]
# 1  prod_456          49.99                 3  [0.3, 0.4, ...]

# Stop SparkSession when done
spark.stop()
```

### Decoding from a Spark DataFrame

```python
from pyspark.sql import SparkSession
import inference_logging_client

# Create SparkSession
spark = SparkSession.builder \
    .appName("inference-decode") \
    .getOrCreate()

# Read parquet file containing MPLog data
df = spark.read.parquet("inference_logs.parquet")

# Expected columns: features, metadata, mp_config_id, entities, ...
print(df.columns)
# ['prism_ingested_at', 'features', 'metadata', 'mp_config_id', 'entities', ...]

# Decode features from each row
decoded_df = inference_logging_client.decode_mplog_dataframe(df, spark)

decoded_df.show()
#    entity_id  prism_ingested_at  mp_config_id  feature_1  feature_2
# 0  user_123   2024-01-15 10:30   my-model      42         3.14
# 1  user_456   2024-01-15 10:30   my-model      17         2.71

spark.stop()
```

### Reading `.log` Files (local disk or GCS)

`asyncloguploader` writes framed `.log` containers holding many MPLog records
per file, partitioned by model + date + hour:

```
gs://gcs-dsci-inferflow-async-logger-prd/
└── async-logger-gcs-flush/<model_config_id>/<YYYY-MM-DD>/<HH>/
    └── <pod>_<YYYY-MM-DD>_<HH-MM-SS>_<seq>.log
```

Decode one file — local path, file-like object, or `gs://` URI:

```python
import inference_logging_client as ilc

# Local
pdf = ilc.decode_log_file_to_pandas("./pod.log")

# GCS  (requires: pip install "inference-logging-client[gcs]")
uri = ("gs://gcs-dsci-inferflow-async-logger-prd/async-logger-gcs-flush/"
       "search-ad-head-prepaid/2026-06-30/18/"
       "search-ad-head-prepaid--prd-inferflow-search-ad-ssd-primary-"
       "54c577fb4d-csbhq_2026-06-30_18-29-03_3.log")
pdf = ilc.decode_log_file_to_pandas(uri)
```

Fan out across a partition (files share the same schema — fetch it once):

```python
from google.cloud import storage
client = storage.Client()
blobs = list(client.list_blobs(
    "gcs-dsci-inferflow-async-logger-prd",
    prefix="async-logger-gcs-flush/search-ad-head-prepaid/2026-06-30/18/",
))
schema = ilc.get_feature_schema("search-ad-head-prepaid", version=4)
for b in blobs:
    ilc.decode_log_file_to_csv(
        f"gs://{b.bucket.name}/{b.name}",
        f"./out/{b.name.rsplit('/', 1)[-1]}.csv",
        schema=schema,
    )
```

Or use the CLI — same code path, no boilerplate:

```bash
inference-logging-client ./pod.log --output-format csv -o pod.csv
inference-logging-client gs://.../pod.log --output-format analyze
```

Row schema, output-format table, and full API list live in
[Reading .log Files](#log-api-reference).

---

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `INFERENCE_HOST` | `http://localhost:8082` | Inference service base URL |
| `INFERENCE_PATH` | `/api/v1/inference/mp-config-registry/get_feature_schema` | Schema fetch API path |

### Setting Environment Variables

```bash
export INFERENCE_HOST="https://inference.prod.example.com"
export INFERENCE_PATH="/api/v1/inference/mp-config-registry/get_feature_schema"
```

### Programmatic Configuration

```python
from pyspark.sql import SparkSession
import inference_logging_client

spark = SparkSession.builder.appName("decode").getOrCreate()

# Pass host directly to functions
df = inference_logging_client.decode_mplog(
    log_data=data,
    model_proxy_id="my-model",
    version=1,
    spark=spark,
    inference_host="https://inference.staging.example.com"
)
```

---

## Core API Reference

### decode_mplog()

Main function to decode MPLog bytes to a Spark DataFrame.

```python
def decode_mplog(
    log_data: bytes,
    model_proxy_id: str,
    version: int,
    spark: SparkSession,
    format_type: Optional[Format] = None,
    inference_host: Optional[str] = None,
    decompress: bool = True,
    schema: Optional[list] = None
) -> pyspark.sql.DataFrame:
```

#### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `log_data` | `bytes` | Yes | - | The MPLog bytes (possibly zstd compressed) |
| `model_proxy_id` | `str` | Yes | - | The model proxy config ID for schema lookup |
| `version` | `int` | Yes | - | The schema version (0-15) |
| `spark` | `SparkSession` | Yes | - | The SparkSession to use for creating DataFrames |
| `format_type` | `Format` | No | `None` | Encoding format. If None, auto-detects from metadata |
| `inference_host` | `str` | No | `None` | Inference service URL. Falls back to `INFERENCE_HOST` env |
| `decompress` | `bool` | No | `True` | Whether to attempt zstd decompression |
| `schema` | `list` | No | `None` | Pre-fetched schema to skip API call |

#### Returns

`pyspark.sql.DataFrame` with:
- First column: `entity_id` - identifier for each entity
- Remaining columns: decoded feature values

#### Exceptions

| Exception | When Raised |
|-----------|-------------|
| `ValueError` | Version out of range (0-15) |
| `ImportError` | Data is zstd-compressed but `zstandard` not installed |
| `FormatError` | Unsupported format or parse error |
| `SchemaFetchError` | Failed to fetch schema from API |
| `SchemaNotFoundError` | No features in schema response |

#### Example: Basic Usage

```python
from pyspark.sql import SparkSession
import inference_logging_client

spark = SparkSession.builder.appName("decode").getOrCreate()

with open("log.bin", "rb") as f:
    data = f.read()

df = inference_logging_client.decode_mplog(
    log_data=data,
    model_proxy_id="recommendation-model",
    version=2,
    spark=spark
)

print(f"Decoded {df.count()} entities with {len(df.columns) - 1} features")
```

#### Example: Explicit Format

```python
from pyspark.sql import SparkSession
from inference_logging_client import decode_mplog, Format

spark = SparkSession.builder.appName("decode").getOrCreate()

df = decode_mplog(
    log_data=arrow_encoded_data,
    model_proxy_id="my-model",
    version=1,
    spark=spark,
    format_type=Format.ARROW  # Skip auto-detection
)
```

#### Example: Pre-fetched Schema (Performance Optimization)

```python
from pyspark.sql import SparkSession
from inference_logging_client import decode_mplog, get_feature_schema

spark = SparkSession.builder.appName("decode").getOrCreate()

# Fetch schema once
schema = get_feature_schema("my-model", 1, "https://inference.example.com")

# Decode multiple logs with same schema
for log_bytes in batch_of_logs:
    df = decode_mplog(
        log_data=log_bytes,
        model_proxy_id="my-model",
        version=1,
        spark=spark,
        schema=schema  # Reuse cached schema
    )
    process(df)
```

---

### decode_mplog_dataframe()

Decode MPLog features from a Spark DataFrame containing encoded feature data.

```python
def decode_mplog_dataframe(
    df: pyspark.sql.DataFrame,
    spark: SparkSession,
    inference_host: Optional[str] = None,
    decompress: bool = True,
    features_column: str = "features",
    metadata_column: str = "metadata",
    mp_config_id_column: str = "mp_config_id"
) -> pyspark.sql.DataFrame:
```

#### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `df` | `pyspark.sql.DataFrame` | Yes | - | Input Spark DataFrame with MPLog columns |
| `spark` | `SparkSession` | Yes | - | The SparkSession to use |
| `inference_host` | `str` | No | `None` | Inference service URL |
| `decompress` | `bool` | No | `True` | Attempt zstd decompression |
| `features_column` | `str` | No | `"features"` | Column containing encoded features |
| `metadata_column` | `str` | No | `"metadata"` | Column containing metadata byte |
| `mp_config_id_column` | `str` | No | `"mp_config_id"` | Column containing model proxy ID |

#### Expected Input DataFrame Columns

| Column | Type | Required | Description |
|--------|------|----------|-------------|
| `features` | `bytes/str` | Yes | Encoded feature bytes (raw, base64, or hex) |
| `metadata` | `int/bytes` | Yes | Metadata byte for version/format detection |
| `mp_config_id` | `str` | Yes | Model proxy config ID |
| `entities` | `list/str` | No | Entity IDs (JSON list or single value) |
| `prism_ingested_at` | `datetime` | No | Preserved in output |
| `prism_extracted_at` | `datetime` | No | Preserved in output |
| `created_at` | `datetime` | No | Preserved in output |
| `parent_entity` | `str/list` | No | Preserved in output |
| `tracking_id` | `str` | No | Preserved in output |
| `user_id` | `str` | No | Preserved in output |
| `year`, `month`, `day`, `hour` | `int` | No | Partition columns, preserved |

#### Returns

`pyspark.sql.DataFrame` with:
- `entity_id`: Entity identifier (one row per entity)
- Metadata columns (if present in input)
- Decoded feature columns

#### Example: Processing Parquet Logs

```python
from pyspark.sql import SparkSession
import inference_logging_client

spark = SparkSession.builder.appName("decode").getOrCreate()

# Read from data lake
df = spark.read.parquet("s3://bucket/inference-logs/dt=2024-01-15/")

# Decode all rows
decoded = inference_logging_client.decode_mplog_dataframe(df, spark)

# Analyze features
decoded.groupBy('mp_config_id').avg('feature_score').show()
```

#### Example: Custom Column Names

```python
from pyspark.sql import SparkSession
import inference_logging_client

spark = SparkSession.builder.appName("decode").getOrCreate()

# Your DataFrame has different column names
df = spark.read.csv("custom_logs.csv", header=True)

decoded = inference_logging_client.decode_mplog_dataframe(
    df,
    spark,
    features_column="encoded_data",      # Custom name
    metadata_column="meta",               # Custom name
    mp_config_id_column="model_id"        # Custom name
)
```

---

<a id="log-api-reference"></a>
### Reading .log Files

All `.log` reading routes through one shared iterator
(`iter_decoded_log_rows`), so every output sink emits the **same rows** —
only the container differs. Each function accepts a `source` that can be a
local path, an open binary file-like object, or a `gs://bucket/key` URI.

#### Available sinks

| Function | Returns / Writes | Best for |
|----------|------------------|----------|
| `decode_log_file(source, spark, ...)` | `pyspark.sql.DataFrame` | Big files, distributed downstream |
| `decode_log_file_to_pandas(source, ...)` | `pandas.DataFrame` | Notebooks, mid-sized files |
| `decode_log_file_to_csv(source, path, ...)` | rows written (int) | Handoff to BigQuery / Snowflake / Excel |
| `decode_log_file_to_jsonl(source, path, ...)` | rows written (int) | Streaming pipelines |
| `write_parsed_log(source, path, ...)` | records written (int) | Human-readable inspection (asynclogparser `.parsed.log` layout) |
| `analyze_log_file(source)` + `print_analysis(...)` | dict / prints | Structural diagnosis, **no schema fetch** |
| `iter_decoded_log_rows(source, ...)` | `Iterator[dict]` | Custom sinks (Kafka, ClickHouse, ...) |
| `iter_log_records(source, ...)` | `Iterator[(ts_ns, bytes)]` | Raw MPLog payloads for custom decoding |
| `read_log_file(source, ...)` | `list[(ts_ns, bytes)]` | Small files, tests |

#### Common keyword arguments

Every decoding function above accepts these — same defaults, same meaning:

| Argument | Default | Meaning |
|----------|---------|---------|
| `inference_host` | `INFERENCE_HOST` env, else `http://localhost:8082` | Schema API base URL |
| `decompress` | `True` | Attempt zstd decompression per MPLog payload |
| `schema` | `None` | Pre-fetched `list[FeatureInfo]`; skips per-record schema fetch |
| `needed_columns` | `None` | Subset of feature names to keep |
| `strict` | `False` | `True` raises `FormatError` on any malformed frame |

#### Row shape (every decoding sink)

```
entity_id, timestamp_ns, mp_config_id, version, format_type,
user_id, tracking_id, parent_entity, <feature_1>, <feature_2>, ...
```

One row per `(record, entity)`. The Spark sink stringifies `list`/`bytes`
values for schema stability; all other sinks preserve native Python types.

#### Frame layout

```
Frame header (8B, little-endian):  capacity (uint32) + valid_data_bytes (uint32)
Frame body (capacity − 8 bytes):   first valid_data_bytes carry records, rest is padding
Record:                            [4B length][8B timestamp_ns][MPLog protobuf]
```

Records are packed back-to-back with no per-record alignment. The MPLog
protobuf is the same payload `get_mplog_metadata()` and `decode_mplog()`
consume — its metadata byte still drives per-record format selection
(proto / arrow / parquet).

#### Examples

```python
import inference_logging_client as ilc

# Spark
from pyspark.sql import SparkSession
spark = SparkSession.builder.getOrCreate()
df = ilc.decode_log_file("gs://bucket/pod.log", spark)

# pandas (no Spark)
pdf = ilc.decode_log_file_to_pandas("./pod.log")

# Write to disk
ilc.decode_log_file_to_csv  ("./pod.log", "./out.csv")
ilc.decode_log_file_to_jsonl("./pod.log", "./out.jsonl")
ilc.write_parsed_log        ("./pod.log", "./out.parsed.log")   # asynclogparser format

# Structural diagnosis (no schema fetch, no network)
ilc.print_analysis(ilc.analyze_log_file("./pod.log"))

# CI fail-fast on corrupt frames
ilc.decode_log_file_to_pandas("./golden.log", strict=True)

# Custom sink via the shared iterator
for row in ilc.iter_decoded_log_rows("gs://bucket/pod.log"):
    producer.send("decoded", row)
```

#### Parity with `asynclogparser`

| asynclogparser | SDK equivalent |
|----------------|----------------|
| `python asynclogparse.py foo.log` | `write_parsed_log("foo.log", "foo.parsed.log")` |
| `python asynclogparse.py --analyze foo.log` | `analyze_log_file(...)` + `print_analysis(...)` |
| `deframe_log_file` | `iter_log_records(...)` — byte-level identical |
| Per-record protobuf parse | `parse_mplog_protobuf(...)` |
| Per-entity `decode_proto_features` | Auto-dispatched by `format_type` (proto/arrow/parquet) |

Additions the SDK carries over the standalone script:
zstd decompression of MPLog payloads, cross-file schema cache keyed by
`(mp_config_id, version)`, and `strict` mode for CI. The ~250 lines of
byte-scan recovery heuristics in `asynclogparser` are intentionally not
carried — use `strict=True` for fail-fast, or the default `strict=False` to
warn-and-skip on a bad frame.

---

### get_mplog_metadata()

Extract metadata from MPLog bytes without full decoding. Useful for inspecting format and version.

```python
def get_mplog_metadata(
    log_data: bytes,
    decompress: bool = True
) -> DecodedMPLog:
```

#### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `log_data` | `bytes` | Yes | - | The MPLog bytes |
| `decompress` | `bool` | No | `True` | Attempt zstd decompression |

#### Returns

`DecodedMPLog` dataclass with:
- `user_id`: User identifier
- `tracking_id`: Request tracking ID
- `model_proxy_config_id`: Model proxy config ID
- `entities`: List of entity IDs
- `parent_entity`: List of parent entity IDs
- `metadata_byte`: Raw metadata byte
- `compression_enabled`: Whether compression was enabled
- `version`: Schema version (0-15)
- `format_type`: Format type int (0=proto, 1=arrow, 2=parquet)

#### Example: Inspect Log Before Decoding

```python
import inference_logging_client

with open("unknown_log.bin", "rb") as f:
    data = f.read()

metadata = inference_logging_client.get_mplog_metadata(data)

print(f"Model: {metadata.model_proxy_config_id}")
print(f"Version: {metadata.version}")
print(f"Format: {inference_logging_client.get_format_name(metadata.format_type)}")
print(f"Compression: {'enabled' if metadata.compression_enabled else 'disabled'}")
print(f"Entities: {len(metadata.entities)}")
```

---

### get_feature_schema()

Fetch feature schema from the inference API with automatic caching.

```python
def get_feature_schema(
    model_config_id: str,
    version: int,
    inference_host: Optional[str] = None,
    api_path: Optional[str] = None
) -> list[FeatureInfo]:
```

#### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `model_config_id` | `str` | Yes | - | Model proxy config ID |
| `version` | `int` | Yes | - | Schema version |
| `inference_host` | `str` | No | `None` | Inference service URL |
| `api_path` | `str` | No | `None` | API path for schema endpoint |

#### Returns

`list[FeatureInfo]`: List of feature definitions with:
- `name`: Feature name
- `feature_type`: Feature data type (e.g., "FP32", "INT64", "FP32VECTOR")
- `index`: Feature index in encoding order

#### Caching Behavior

- Schemas are cached by `(model_config_id, version)` tuple
- Cache is thread-safe (uses threading.Lock)
- Maximum 100 cached schemas (LRU eviction)
- Host/path are NOT part of cache key (schemas are canonical)

#### Example: Manual Schema Fetch

```python
from inference_logging_client import get_feature_schema

schema = get_feature_schema(
    model_config_id="product-ranking",
    version=3,
    inference_host="https://inference.example.com"
)

for feature in schema:
    print(f"  {feature.index}: {feature.name} ({feature.feature_type})")
```

---

### clear_schema_cache()

Clear the internal schema cache. Useful for testing or when schemas have changed.

```python
def clear_schema_cache() -> None:
```

#### Example

```python
from inference_logging_client import clear_schema_cache, get_feature_schema

# Clear before testing
clear_schema_cache()

# This will fetch fresh from API
schema = get_feature_schema("my-model", 1)
```

---

## Data Types

### Format Enum

```python
from inference_logging_client import Format

class Format(Enum):
    PROTO = "proto"     # Custom binary encoding
    ARROW = "arrow"     # Arrow IPC format
    PARQUET = "parquet" # Parquet format
```

### FeatureInfo

```python
from inference_logging_client import FeatureInfo

@dataclass
class FeatureInfo:
    name: str           # Feature name (e.g., "user_embedding")
    feature_type: str   # Type string (e.g., "FP32VECTOR")
    index: int          # Position in encoded data
```

### DecodedMPLog

```python
from inference_logging_client import DecodedMPLog

@dataclass
class DecodedMPLog:
    user_id: str = ""
    tracking_id: str = ""
    model_proxy_config_id: str = ""
    entities: list[str] = field(default_factory=list)
    parent_entity: list[str] = field(default_factory=list)
    metadata_byte: int = 0
    compression_enabled: bool = False
    version: int = 0
    format_type: int = 0  # 0=proto, 1=arrow, 2=parquet
```

---

## Supported Feature Types

### Scalar Types

| Type Aliases | Size | Description |
|-------------|------|-------------|
| `INT8`, `I8` | 1 byte | Signed 8-bit integer |
| `INT16`, `I16`, `SHORT` | 2 bytes | Signed 16-bit integer |
| `INT32`, `I32`, `INT` | 4 bytes | Signed 32-bit integer |
| `INT64`, `I64`, `LONG` | 8 bytes | Signed 64-bit integer |
| `UINT8`, `U8` | 1 byte | Unsigned 8-bit integer |
| `UINT16`, `U16` | 2 bytes | Unsigned 16-bit integer |
| `UINT32`, `U32` | 4 bytes | Unsigned 32-bit integer |
| `UINT64`, `U64` | 8 bytes | Unsigned 64-bit integer |
| `FP16`, `FLOAT16`, `F16` | 2 bytes | IEEE 754 half-precision float |
| `FP32`, `FLOAT32`, `F32`, `FLOAT` | 4 bytes | IEEE 754 single-precision float |
| `FP64`, `FLOAT64`, `F64`, `DOUBLE` | 8 bytes | IEEE 754 double-precision float |
| `FP8E5M2`, `FP8E4M3` | 1 byte | 8-bit floating point (raw byte) |
| `BOOL`, `BOOLEAN` | 1 byte | Boolean value |

### String Types

| Type | Description |
|------|-------------|
| `STRING`, `STR` | UTF-8 encoded string |
| `BYTES` | Binary bytes with 2-byte length prefix |

### Vector Types

All scalar types have vector variants:

| Type Pattern | Description |
|--------------|-------------|
| `{TYPE}VECTOR` | e.g., `FP32VECTOR`, `INT64VECTOR` |
| `VECTOR_{TYPE}` | e.g., `VECTOR_FP32`, `VECTOR_INT64` |
| `DATATYPE{TYPE}VECTOR` | e.g., `DATATYPEFP32VECTOR` |

Vectors can be encoded as:
- **Binary**: Packed element bytes (most common for feature stores)
- **JSON**: JSON array string (fallback)

#### Example: Working with Vectors

```python
from pyspark.sql import SparkSession
import inference_logging_client

spark = SparkSession.builder.appName("decode").getOrCreate()

df = inference_logging_client.decode_mplog(data, "model", 1, spark)

# Vector columns contain arrays
df.select("entity_id", "user_embedding").show(truncate=False)

# Access vector elements with Spark SQL functions
from pyspark.sql import functions as F
df.select("entity_id", F.element_at("user_embedding", 1).alias("first_elem")).show()
```

---

## Encoding Formats Explained

### Proto Format

The default and most compact encoding format.

```
┌─────────────────────────────────────────────────────────────┐
│ Byte 0: Generated Flag (1 = no generated values)           │
├─────────────────────────────────────────────────────────────┤
│ Feature 0: [fixed bytes OR 2-byte size + data]              │
├─────────────────────────────────────────────────────────────┤
│ Feature 1: [fixed bytes OR 2-byte size + data]              │
├─────────────────────────────────────────────────────────────┤
│ ...                                                         │
└─────────────────────────────────────────────────────────────┘
```

- **Scalars**: Fixed size based on type (e.g., 4 bytes for FP32)
- **Strings/Vectors**: 2-byte little-endian size prefix + data

### Arrow Format

Uses Arrow IPC (Inter-Process Communication) format.

```
┌─────────────────────────────────────────────────────────────┐
│ Arrow IPC Stream                                            │
│ ├── Schema: columns "0", "1", "2", ... (binary type)        │
│ └── RecordBatch                                             │
│     ├── Column "0": [entity0_feature0_bytes, ...]           │
│     ├── Column "1": [entity0_feature1_bytes, ...]           │
│     └── ...                                                 │
└─────────────────────────────────────────────────────────────┘
```

- Column names are feature indices as strings ("0", "1", "2", ...)
- Each cell contains raw binary feature bytes
- All entities in a single IPC blob

### Parquet Format

Uses Parquet columnar format.

```
┌─────────────────────────────────────────────────────────────┐
│ Parquet File                                                │
│ └── Column "Features": map<int, binary>                     │
│     ├── Row 0: {0: bytes, 1: bytes, ...}                   │
│     ├── Row 1: {0: bytes, 1: bytes, ...}                   │
│     └── ...                                                 │
└─────────────────────────────────────────────────────────────┘
```

- Features column is a map from feature index to binary bytes
- Each row represents one entity
- Alternative: columnar format with index-named columns (like Arrow)

### Metadata Byte Layout

```
Bit Layout:
┌─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┐
│  7  │  6  │  5  │  4  │  3  │  2  │  1  │  0  │
├─────┴─────┼─────┴─────┴─────┴─────┼─────┼─────┤
│  Format   │       Version         │ Rsv │Comp │
│  (2 bits) │       (4 bits)        │     │     │
└───────────┴───────────────────────┴─────┴─────┘

Format Type:
  00 = Proto
  01 = Arrow
  10 = Parquet
  11 = Reserved

Version: 0-15 (4 bits)
Compression: 0 = disabled, 1 = enabled (zstd)
```

---

## Exception Handling

### Exception Hierarchy

```
InferenceLoggingError (base)
├── SchemaFetchError     # API request failed
├── SchemaNotFoundError  # No features in response
├── DecodeError          # Feature decoding failed
├── FormatError          # Invalid format or parse error
└── ProtobufError        # Protobuf parsing failed
```

### Example: Comprehensive Error Handling

```python
from pyspark.sql import SparkSession
from inference_logging_client import (
    decode_mplog,
    InferenceLoggingError,
    SchemaFetchError,
    SchemaNotFoundError,
    FormatError,
)

spark = SparkSession.builder.appName("decode").getOrCreate()

try:
    df = decode_mplog(data, "my-model", 1, spark)
except SchemaFetchError as e:
    print(f"Failed to fetch schema: {e}")
    # Check network, inference service availability
except SchemaNotFoundError as e:
    print(f"Schema not found: {e}")
    # Check model_proxy_id and version
except FormatError as e:
    print(f"Invalid data format: {e}")
    # Check data integrity, correct format
except ValueError as e:
    print(f"Invalid parameter: {e}")
    # Check version range (0-15)
except ImportError as e:
    print(f"Missing dependency: {e}")
    # Install zstandard if data is compressed
except InferenceLoggingError as e:
    print(f"Decode error: {e}")
    # Generic fallback
finally:
    spark.stop()
```

---

## Command Line Interface

### Basic Usage

The CLI accepts three kinds of input and auto-routes them:

| Input | Detected by | Path |
|-------|-------------|------|
| Raw MPLog bytes file or `-` (stdin) | default | `decode_mplog()` — needs `-m` / `-v` |
| `*.log` container (any local path) | `.log` extension | `decode_log_file()` |
| `gs://bucket/key` URI | `gs://` prefix | `decode_log_file()` |

```bash
# Raw MPLog bytes (the original mode)
inference-logging-client --model-proxy-id my-model --version 1 input.bin

# Asyncloguploader .log file — no -m/-v needed (per-record from metadata)
inference-logging-client ./pod-xyz_17-58-12.log -o decoded.csv

# Same, but straight from GCS
inference-logging-client gs://my-bucket/asynclogs/pod-xyz_17-58-12.log --json
```

### CLI Arguments

| Argument | Short | Required | Default | Description |
|----------|-------|----------|---------|-------------|
| `input` | - | Yes | - | MPLog bytes file, `*.log` path, `gs://...` URI, or `-` for stdin |
| `--model-proxy-id` | `-m` | Raw-bytes mode only | - | Model proxy config ID (ignored for `.log` / `gs://`) |
| `--version` | `-v` | Raw-bytes mode only | - | Schema version (ignored for `.log` / `gs://`) |
| `--format` | `-f` | No | `auto` | Format: `proto`, `arrow`, `parquet`, `auto` (raw-bytes mode only) |
| `--inference-host` | - | No | env/localhost | Inference service URL |
| `--hex` | - | No | - | Input is hex-encoded (raw-bytes mode only) |
| `--base64` | - | No | - | Input is base64-encoded (raw-bytes mode only) |
| `--no-decompress` | - | No | - | Skip zstd decompression |
| `--output` | `-o` | No | stdout | Output file path; CSV by default, JSON with `--json` |
| `--json` | - | No | - | Output as JSON |
| `--spark-master` | - | No | `local[*]` | Spark master URL |
| `--strict` | - | No | - | `.log` mode only: raise on malformed frames instead of skipping |
| `--output-format` | - | No | `spark` (in .log mode) | `.log`/`gs://` only: `spark`, `pandas`, `csv`, `jsonl`, `text`, or `analyze` |

### Examples

```bash
# Output to CSV directory
inference-logging-client -m my-model -v 1 input.bin -o output_dir

# Output as JSON
inference-logging-client -m my-model -v 1 input.bin --json

# Read from stdin (base64 encoded)
echo "BASE64_DATA" | inference-logging-client -m my-model -v 1 --base64 -

# Read from stdin (hex encoded)
cat hex_data.txt | inference-logging-client -m my-model -v 1 --hex -

# Explicit Arrow format
inference-logging-client -m my-model -v 1 --format arrow input.bin

# Custom inference host
inference-logging-client -m my-model -v 1 \
    --inference-host https://inference.prod.example.com \
    input.bin

# Custom Spark master
inference-logging-client -m my-model -v 1 \
    --spark-master spark://master:7077 \
    input.bin

# Skip decompression (for pre-decompressed data)
inference-logging-client -m my-model -v 1 --no-decompress input.bin

# --- .log / GCS examples -----------------------------------------

# Decode a local asyncloguploader .log file to CSV
inference-logging-client ./pod-xyz_17-58-12.log -o decoded.csv

# Decode straight from GCS (requires the [gcs] extra)
inference-logging-client gs://my-bucket/asynclogs/pod-xyz_17-58-12.log \
    --inference-host https://inference.prod.example.com \
    -o gs_decoded.csv

# CI-style gate: fail on any corrupt frame
inference-logging-client gs://my-bucket/asynclogs/golden.log --strict --json

# Pandas-style CSV print to stdout (no Spark)
inference-logging-client ./pod-xyz_17-58-12.log --output-format pandas

# Write CSV / JSONL directly (no Spark)
inference-logging-client ./pod-xyz_17-58-12.log --output-format csv  -o decoded.csv
inference-logging-client ./pod-xyz_17-58-12.log --output-format jsonl -o decoded.jsonl

# Asynclogparser-compatible human-readable text
inference-logging-client ./pod-xyz_17-58-12.log --output-format text -o decoded.parsed.log

# Structural diagnosis only (no schema fetch, no decoding)
inference-logging-client ./pod-xyz_17-58-12.log --output-format analyze
```

### CLI Output Format

```
+----------+----------+----------+----------+
| entity_id| feature_1| feature_2| feature_3|
+----------+----------+----------+----------+
| entity_0 |      1.5 |      2.5 |      3.5 |
| entity_1 |      4.5 |      5.5 |      6.5 |
+----------+----------+----------+----------+

--- Summary ---
Format: proto (from metadata)
Version: 1
Compression: disabled
Rows: 2
Columns: 4
Features: feature_1, feature_2, feature_3...
```

---

## Advanced Usage Examples

### Batch Processing with Schema Reuse

```python
import os
import glob
from pyspark.sql import SparkSession
import inference_logging_client

spark = SparkSession.builder.appName("batch-decode").getOrCreate()

# Pre-fetch schema once
schema = inference_logging_client.get_feature_schema(
    "batch-model", 2, "https://inference.example.com"
)

def process_file(filepath):
    with open(filepath, "rb") as f:
        data = f.read()
    
    return inference_logging_client.decode_mplog(
        log_data=data,
        model_proxy_id="batch-model",
        version=2,
        spark=spark,
        schema=schema  # Reuse cached schema
    )

# Process files sequentially
log_files = glob.glob("/data/logs/*.bin")
all_dfs = [process_file(f) for f in log_files]

# Union all DataFrames
from functools import reduce
all_data = reduce(lambda a, b: a.union(b), all_dfs)
print(f"Total entities: {all_data.count()}")

spark.stop()
```

### Feature Analysis Pipeline

```python
from pyspark.sql import SparkSession
from pyspark.sql import functions as F
import inference_logging_client

spark = SparkSession.builder.appName("analysis").getOrCreate()

# Decode logs
df = inference_logging_client.decode_mplog_dataframe(
    spark.read.parquet("logs.parquet"),
    spark
)

# Analyze vector features
embedding_col = "user_embedding"

# Get embedding statistics
df.select(
    F.size(F.col(embedding_col)).alias("dimension"),
    F.aggregate(F.col(embedding_col), F.lit(0.0), lambda acc, x: acc + x).alias("sum")
).show()

# Find entities with unusual embeddings (using array functions)
df.withColumn(
    "embedding_norm",
    F.sqrt(F.aggregate(
        F.col(embedding_col),
        F.lit(0.0),
        lambda acc, x: acc + x * x
    ))
).filter(F.col("embedding_norm") > 10.0).show()

spark.stop()
```

### Integration with Feature Store

```python
from pyspark.sql import SparkSession
import inference_logging_client

spark = SparkSession.builder.appName("feature-compare").getOrCreate()

# Decode inference logs
df = inference_logging_client.decode_mplog(data, "ranking-model", 1, spark)

# Compare with feature store values
from feature_store import FeatureStoreClient

fs = FeatureStoreClient()

# Collect for comparison (for small datasets)
for row in df.collect():
    entity_id = row['entity_id']
    
    # Get fresh features from store
    fresh_features = fs.get_features(entity_id, ["feature_a", "feature_b"])
    
    # Compare logged vs fresh
    for feature_name in ["feature_a", "feature_b"]:
        logged = row[feature_name]
        fresh = fresh_features[feature_name]
        
        if logged != fresh:
            print(f"Drift detected for {entity_id}.{feature_name}:")
            print(f"  Logged: {logged}")
            print(f"  Fresh:  {fresh}")

spark.stop()
```

### Custom Schema Source

```python
from pyspark.sql import SparkSession
from inference_logging_client import decode_mplog, FeatureInfo

spark = SparkSession.builder.appName("custom-schema").getOrCreate()

# Define schema manually (useful for testing or offline processing)
custom_schema = [
    FeatureInfo(name="user_age", feature_type="INT32", index=0),
    FeatureInfo(name="user_score", feature_type="FP32", index=1),
    FeatureInfo(name="user_embedding", feature_type="FP32VECTOR", index=2),
    FeatureInfo(name="user_category", feature_type="STRING", index=3),
]

df = decode_mplog(
    log_data=data,
    model_proxy_id="my-model",  # Not used when schema provided
    version=1,                   # Not used when schema provided
    spark=spark,
    schema=custom_schema
)

spark.stop()
```

---

## Architecture & Internals

### Module Structure

```
inference_logging_client/
├── __init__.py      # Public API exports, decode_mplog(), decode_mplog_dataframe()
├── __main__.py      # Module execution entry point
├── cli.py           # Command-line interface (auto-routes .log / gs://)
├── decoder.py       # Core byte decoding, type conversion
├── exceptions.py    # Exception classes
├── formats.py       # Proto/Arrow/Parquet format decoders
├── io.py            # Schema fetching, protobuf parsing
├── log_reader.py    # .log frame deframer + GCS reader + decode_log_file()
├── types.py         # Data type definitions (Format, FeatureInfo, DecodedMPLog)
└── utils.py         # Utility functions (type normalization, formatting)
```

### Decoding Flow

```
                    ┌──────────────────┐
                    │   MPLog Bytes    │
                    │  (compressed?)   │
                    └────────┬─────────┘
                             │
                    ┌────────▼─────────┐
                    │  Zstd Decompress │
                    │   (if enabled)   │
                    └────────┬─────────┘
                             │
                    ┌────────▼─────────┐
                    │ Parse Protobuf   │
                    │ (outer wrapper)  │
                    └────────┬─────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
      ┌───────▼───────┐ ┌────▼────┐ ┌───────▼───────┐
      │  Proto Format │ │  Arrow  │ │ Parquet Format│
      │    Decoder    │ │ Decoder │ │    Decoder    │
      └───────┬───────┘ └────┬────┘ └───────┬───────┘
              │              │              │
              └──────────────┼──────────────┘
                             │
                    ┌────────▼─────────┐
                    │  Feature Schema  │◄──── API Fetch
                    │    (cached)      │
                    └────────┬─────────┘
                             │
                    ┌────────▼─────────┐
                    │  Decode Features │
                    │  (by type)       │
                    └────────┬─────────┘
                             │
                    ┌────────▼─────────┐
                    │ Spark DataFrame  │
                    └──────────────────┘
```

### Schema Cache

```python
# Thread-safe LRU cache with max 100 entries
# Key: (model_config_id, version)
# Value: list[FeatureInfo]

# Cache is NOT keyed by host/path because schemas are canonical
# Same model+version = same schema regardless of which host serves it
```

---

## Troubleshooting

### Common Issues

#### "No features found in schema"

```
SchemaNotFoundError: No features found in schema for model_config_id=xxx, version=1
```

**Causes:**
- Incorrect `model_proxy_id`
- Wrong `version` number
- Schema not yet registered

**Solutions:**
1. Verify model_proxy_id matches exactly
2. Check available versions in inference service
3. Use `get_mplog_metadata()` to see the version in the data

#### "Version out of valid range"

```
ValueError: Version 16 is out of valid range (0-15)
```

**Cause:** Version is encoded in 4 bits (0-15 only)

**Solution:** Check the version number passed to decode functions

#### "Data appears to be zstd-compressed but zstandard not installed"

```
ImportError: Data appears to be zstd-compressed but the 'zstandard' package is not installed.
```

**Solution:**
```bash
pip install zstandard
```

#### "Failed to read Arrow IPC data"

**Causes:**
- Corrupted data
- Wrong format specified
- Incomplete data

**Solutions:**
1. Use `format_type=None` for auto-detection
2. Check data integrity
3. Try `get_mplog_metadata()` to inspect format

#### Empty DataFrame Returned

**Causes:**
- No entities in the log
- All features decoded as None
- Schema mismatch

**Solutions:**
1. Check `get_mplog_metadata()` to verify entity count
2. Verify schema matches data version
3. Check for decode warnings

### Debug Mode

```python
import warnings
import logging
from pyspark.sql import SparkSession

# Enable all warnings
warnings.simplefilter("always")

# Enable debug logging for HTTP requests
logging.basicConfig(level=logging.DEBUG)

# Create Spark session with verbose logging
spark = SparkSession.builder \
    .appName("debug") \
    .config("spark.driver.extraJavaOptions", "-Dlog4j.logger.org.apache.spark=DEBUG") \
    .getOrCreate()

# Inspect before decoding
import inference_logging_client

metadata = inference_logging_client.get_mplog_metadata(data)
print(f"Format: {metadata.format_type}")
print(f"Version: {metadata.version}")
print(f"Entities: {len(metadata.entities)}")
print(f"Model: {metadata.model_proxy_config_id}")
```

---

## Development

### Setup

```bash
# Clone repository
git clone https://github.com/Meesho/BharatMLStack.git
cd BharatMLStack/py-sdk/inference_logging_client

# Create virtual environment
python -m venv venv
source venv/bin/activate

# Install in editable mode with dev dependencies
pip install -e ".[dev]"
```

### Running Tests

```bash
pytest

# With coverage
pytest --cov=inference_logging_client --cov-report=html
```

### Code Formatting

```bash
# Format with black
black inference_logging_client/

# Lint with ruff
ruff check inference_logging_client/
```

### Building Package

```bash
python -m build
```

---

## License

MIT License

## Repository

[https://github.com/Meesho/BharatMLStack](https://github.com/Meesho/BharatMLStack)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request
