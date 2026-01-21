# Package Structure

This document describes the refactored package structure.

## Directory Structure

```
inference-logging-client/
├── LICENSE
├── MANIFEST.in
├── pyproject.toml          # Package configuration and metadata
├── README.md               # Package documentation
├── .gitignore              # Git ignore rules
└── src/
    └── inference_logging_client/  # Main package (note: underscores for Python imports)
        ├── __init__.py     # Main API exports
        ├── __main__.py     # Package execution entry point
        ├── types.py        # Type definitions (enums, dataclasses)
        ├── utils.py        # Utility functions (type normalization, formatting)
        ├── decoder.py      # Core decoding utilities (ByteReader, value decoders)
        ├── io.py           # I/O operations (schema fetching, protobuf parsing)
        ├── formats.py      # Format-specific decoders (proto, arrow, parquet)
        └── cli.py          # Command-line interface
```

## Module Breakdown

### `types.py`
- `Format` enum (PROTO, ARROW, PARQUET)
- `FeatureInfo` dataclass
- `DecodedMPLog` dataclass
- Format type constants and mappings

### `utils.py`
- Type normalization functions
- Scalar size mappings
- Float formatting utilities
- Metadata byte unpacking
- DataFrame formatting

### `decoder.py`
- `ByteReader` class for sequential byte reading
- Protobuf varint reading
- Scalar value decoding
- Vector/string decoding
- Feature value decoding

### `io.py`
- Schema fetching from custodian API (with caching)
- Protobuf parsing
- Metadata extraction

### `formats.py`
- Proto format decoder
- Arrow format decoder
- Parquet format decoder

### `cli.py`
- Command-line interface
- Argument parsing
- Input/output handling

### `__init__.py`
- Main package API
- `decode_mplog()` function
- `decode_mplog_dataframe()` function
- `get_mplog_metadata()` function
- Public exports

## Installation

```bash
# Install in development mode
pip install -e .

# Build package
python -m build

# Install from built wheel
pip install dist/inference_logging_client-*.whl
```

## Usage

### As a Python Package

```python
import inference_logging_client

# Decode MPLog
df = inference_logging_client.decode_mplog(
    log_data=bytes_data,
    model_proxy_id="my-model",
    version=1
)
```

### As a CLI Tool

```bash
# After installation, the CLI is available as:
inference-logging-client --model-proxy-id my-model --version 1 input.bin

# Or as a module:
python -m inference_logging_client --model-proxy-id my-model --version 1 input.bin
```

## Next Steps

1. Update `pyproject.toml` with your actual author information and repository URLs
2. Add a LICENSE file with your chosen license
3. Test the package installation: `pip install -e .`
4. Build and test the package: `python -m build`
5. Publish to PyPI: `twine upload dist/*`
