# Inference Logging Client

A Python package for decoding MPLog feature logs from proto, arrow, or parquet format.

## Features

- Decode MPLog feature logs from multiple encoding formats:
  - **Proto**: Custom binary encoding with generated flag + sequential features
  - **Arrow**: Arrow IPC format with binary columns
  - **Parquet**: Parquet format with feature map
- Automatic format detection from metadata
- Support for zstd compression
- Fetch feature schemas from inference API
- Convert decoded logs to pandas DataFrames
- Command-line interface for easy usage

## Installation

```bash
pip install inference-logging-client
```

## Quick Start

### Python API

```python
import inference_logging_client

# Decode MPLog from bytes
with open("log.bin", "rb") as f:
    data = f.read()

df = inference_logging_client.decode_mplog(
    log_data=data,
    model_proxy_id="my-model",
    version=1
)

print(df.head())
```

### Decode from DataFrame

```python
import pandas as pd
import inference_logging_client

# Read DataFrame with MPLog columns
df = pd.read_parquet("logs.parquet")

# Decode features
decoded_df = inference_logging_client.decode_mplog_dataframe(df)

print(decoded_df.head())
```

### Command Line Interface

```bash
# Decode with auto-detection
inference-logging-client --model-proxy-id my-model --version 1 input.bin

# Specify format explicitly
inference-logging-client --model-proxy-id my-model --version 1 --format proto input.bin

# Output to CSV
inference-logging-client --model-proxy-id my-model --version 1 input.bin -o output.csv

# Decode from stdin (base64)
echo "BASE64_DATA" | inference-logging-client --model-proxy-id my-model --version 1 --base64 -
```

## Configuration

The package uses environment variables for configuration:

- `INFERENCE_HOST`: Inference service host URL (default: `http://localhost:8082`)
- `INFERENCE_PATH`: API path for schema fetching (default: `/api/v1/inference/mp-config-registry/get_feature_schema`)

## API Reference

### `decode_mplog()`

Decode MPLog bytes to a pandas DataFrame.

**Parameters:**
- `log_data` (bytes): The MPLog bytes (possibly compressed)
- `model_proxy_id` (str): The model proxy config ID
- `version` (int): The schema version
- `format_type` (Format, optional): The encoding format. If None, auto-detect from metadata.
- `inference_host` (str, optional): The inference service host URL
- `decompress` (bool): Whether to attempt zstd decompression (default: True)
- `schema` (list, optional): Pre-fetched schema (list of FeatureInfo). If provided, skips schema fetch.

**Returns:**
- `pd.DataFrame`: DataFrame with `entity_id` as first column and features as remaining columns

### `decode_mplog_dataframe()`

Decode MPLog features from a DataFrame with specific column structure.

**Parameters:**
- `df` (pd.DataFrame): Input DataFrame with MPLog data columns
- `inference_host` (str, optional): The inference service host URL
- `decompress` (bool): Whether to attempt zstd decompression (default: True)
- `features_column` (str): Name of the column containing encoded features (default: "features")
- `metadata_column` (str): Name of the column containing metadata byte (default: "metadata")
- `mp_config_id_column` (str): Name of the column containing model proxy config ID (default: "mp_config_id")

**Returns:**
- `pd.DataFrame`: DataFrame with decoded features, one row per entity

### `get_mplog_metadata()`

Extract metadata from MPLog bytes without full decoding.

**Parameters:**
- `log_data` (bytes): The MPLog bytes (possibly compressed)
- `decompress` (bool): Whether to attempt zstd decompression (default: True)

**Returns:**
- `DecodedMPLog`: Object with metadata fields populated

## Supported Feature Types

### Scalar Types
- Integer: INT8, INT16, INT32, INT64, UINT8, UINT16, UINT32, UINT64
- Float: FP16, FP32, FP64, FP8E5M2, FP8E4M3
- Boolean: BOOL
- String: STRING

### Vector Types
- All scalar types can be vectors (e.g., FP32VECTOR, INT64VECTOR)
- Vectors can be binary-encoded or JSON-encoded

## Encoding Formats

### Proto Format
- First byte: generated flag
- Scalars: fixed size bytes based on type
- Strings/Vectors: 2-byte little-endian size prefix + data bytes

### Arrow Format
- Arrow IPC format with binary columns
- Column names are feature indices ("0", "1", ...)
- Each column contains raw feature value bytes

### Parquet Format
- Parquet file with Features column (`map[int][]byte`)
- Each row represents an entity

## Metadata Byte Layout

The metadata byte encodes:
- Bit 0: compression flag (0=disabled, 1=enabled)
- Bit 1: reserved
- Bits 2-5: version (4 bits, 0-15)
- Bits 6-7: format type (00=proto, 01=arrow, 10=parquet)

## Development

```bash
# Install in development mode
pip install -e .

# Install with dev dependencies
pip install -e ".[dev]"

# Run tests
pytest

# Format code
black src/

# Lint code
ruff check src/
```

## License

MIT License

## Repository

[https://github.com/Meesho/BharatMLStack](https://github.com/Meesho/BharatMLStack)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
