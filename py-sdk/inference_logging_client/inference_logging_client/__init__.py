"""
Inference Logging Client - Decode MPLog feature logs from proto, arrow, or parquet format.

This package provides functionality to:
1. Decode MPLog feature logs from various encoding formats (proto, arrow, parquet)
2. Fetch feature schemas from inference API
3. Convert decoded logs to Spark DataFrames

Main functions:
    - decode_mplog: Decode MPLog bytes to a Spark DataFrame
    - decode_mplog_dataframe: Decode MPLog features from a Spark DataFrame
    - get_mplog_metadata: Extract metadata from MPLog bytes
"""

import warnings
from typing import TYPE_CHECKING, Collection, Optional

if TYPE_CHECKING:
    from pyspark.sql import DataFrame as SparkDataFrame
    from pyspark.sql import SparkSession

# Check for zstandard availability at import time for clear error messages
try:
    import zstandard as zstd

    _ZSTD_AVAILABLE = True
except ImportError:
    _ZSTD_AVAILABLE = False
    zstd = None

from .exceptions import (
    DecodeError,
    FormatError,
    InferenceLoggingError,
    ProtobufError,
    SchemaFetchError,
    SchemaNotFoundError,
)
from .formats import (
    decode_arrow_format,
    decode_arrow_features,
    decode_parquet_format,
    decode_parquet_features,
    decode_proto_format,
    decode_proto_features,
)
from .io import clear_schema_cache, get_feature_schema, get_mplog_metadata, parse_mplog_protobuf
from .types import FORMAT_TYPE_MAP, DecodedMPLog, FeatureInfo, Format
from .utils import format_dataframe_floats, get_format_name, unpack_metadata_byte

__version__ = "0.3.9"

# Maximum supported schema version (4 bits = 0-15)
_MAX_SCHEMA_VERSION = 15

__all__ = [
    "decode_mplog",
    "decode_mplog_dataframe",
    "decode_mplog_proto_dataframe",
    "decode_mplog_proto_csv",
    "get_mplog_metadata",
    "get_feature_schema",
    "clear_schema_cache",
    "format_dataframe_floats",
    "Format",
    "FeatureInfo",
    "DecodedMPLog",
    "get_format_name",
    "unpack_metadata_byte",
    # Exceptions
    "InferenceLoggingError",
    "SchemaFetchError",
    "SchemaNotFoundError",
    "DecodeError",
    "FormatError",
    "ProtobufError",
]


def _decompress_zstd(data: bytes) -> bytes:
    """Decompress zstd-compressed data.

    Args:
        data: Potentially zstd-compressed bytes

    Returns:
        Decompressed bytes, or original data if not compressed or zstd unavailable

    Raises:
        ImportError: If data is zstd-compressed but zstandard is not installed
    """
    # Check for zstd magic number: 0x28 0xB5 0x2F 0xFD
    if len(data) >= 4 and data[:4] == b"\x28\xb5\x2f\xfd":
        if not _ZSTD_AVAILABLE:
            raise ImportError(
                "Data appears to be zstd-compressed but the 'zstandard' package is not installed. "
                "Install it with: pip install zstandard"
            )
        decompressor = zstd.ZstdDecompressor()
        return decompressor.decompress(data)
    return data


def _split_raw_proto_entities(raw: bytes) -> list:
    """Split v2 raw-proto wire format into per-entity byte chunks.

    Wire format: [{<proto bytes>}, {<proto bytes>}, ...]
    Separator between entities is b'},{' or b'}, {'.
    """
    if len(raw) < 4:
        return [raw] if raw else []

    if raw[0:1] == b'[' and raw[1:2] == b'{':
        inner = raw[2:]
        if inner.endswith(b'}]'):
            inner = inner[:-2]
        elif inner.endswith(b'}'):
            inner = inner[:-1]

        chunks = []
        start = 0
        i = 0
        while i < len(inner):
            if inner[i:i + 1] == b'}':
                rest = inner[i + 1:i + 4]
                if rest.startswith(b', {') or rest.startswith(b',{'):
                    chunks.append(inner[start:i])
                    skip = 3 if rest.startswith(b', ') else 2
                    start = i + 1 + skip
                    i = start
                    continue
            i += 1
        if start < len(inner):
            chunks.append(inner[start:])
        return [c for c in chunks if c]

    if raw[0:1] == b'{':
        inner = raw[1:]
        if inner.endswith(b'}'):
            inner = inner[:-1]
        return [inner] if inner else []

    return [raw]


def _is_raw_proto_wire_format(data) -> bool:
    """Detect v2 raw-proto: starts with '[{' followed by non-'"' byte."""
    if isinstance(data, str):
        return len(data) >= 3 and data[0] == '[' and data[1] == '{' and data[2] != '"'
    if isinstance(data, (bytes, bytearray, memoryview)):
        raw = bytes(data)
        return len(raw) >= 3 and raw[0:1] == b'[' and raw[1:2] == b'{' and raw[2:3] != b'"'
    return False


def decode_mplog(
    log_data: bytes,
    model_proxy_id: str,
    version: int,
    spark: "SparkSession",
    format_type: Optional[Format] = None,
    inference_host: Optional[str] = None,
    decompress: bool = True,
    schema: Optional[list] = None,
    needed_columns: Optional[Collection[str]] = None,
    go_string: bool = True,
) -> "SparkDataFrame":
    """
    Main function to decode MPLog bytes to a Spark DataFrame.

    Args:
        log_data: The MPLog bytes (possibly compressed)
        model_proxy_id: The model proxy config ID
        version: The schema version (0-15)
        spark: The SparkSession to use for creating DataFrames
        format_type: The encoding format (proto, arrow, parquet). If None, auto-detect from metadata.
        inference_host: The inference service host URL. If None, reads from INFERENCE_HOST env var.
        decompress: Whether to attempt zstd decompression
        schema: Optional pre-fetched schema (list of FeatureInfo). If provided, skips schema fetch.
        needed_columns: Optional set or list of feature names to include. If provided, only these
            columns are returned (reduces memory and output size). If None, all schema columns are returned.

    Returns:
        Spark DataFrame with entity_id as first column and features as remaining columns

    Raises:
        ValueError: If version is out of valid range (0-15)
        ImportError: If data is zstd-compressed but zstandard is not installed
        FormatError: If format is unsupported or data cannot be parsed

    Example:
        >>> from pyspark.sql import SparkSession
        >>> import inference_logging_client
        >>> spark = SparkSession.builder.appName("decode").getOrCreate()
        >>> with open("log.bin", "rb") as f:
        ...     data = f.read()
        >>> df = inference_logging_client.decode_mplog(
        ...     log_data=data,
        ...     model_proxy_id="my-model",
        ...     version=1,
        ...     spark=spark
        ... )
        >>> df.show()
    """
    import os

    # Validate version range
    if not (0 <= version <= _MAX_SCHEMA_VERSION):
        raise ValueError(
            f"Version {version} is out of valid range (0-{_MAX_SCHEMA_VERSION}). "
            f"Version is encoded in 4 bits of the metadata byte."
        )

    # Read from environment variable if not provided
    if inference_host is None:
        inference_host = os.getenv("INFERENCE_HOST", "http://localhost:8082")

    # Attempt decompression if enabled
    working_data = log_data
    if decompress:
        working_data = _decompress_zstd(log_data)

    # If format_type is None, parse the protobuf to get format from metadata
    detected_format = format_type
    if detected_format is None:
        # Parse protobuf to extract metadata and detect format
        parsed = parse_mplog_protobuf(working_data)
        if parsed.format_type in FORMAT_TYPE_MAP:
            detected_format = FORMAT_TYPE_MAP[parsed.format_type]
        else:
            # Default to proto if format type is unknown
            detected_format = Format.PROTO

    # Use provided schema or fetch from inference service
    if schema is None:
        schema = get_feature_schema(model_proxy_id, version, inference_host)

    # go_string=True (default) yields exact go-core BytesToString output for
    # every format: proto threads the flag; arrow/parquet go through
    # decode_feature_value which defaults to the same go-core port.
    # Decode based on format
    if detected_format == Format.PROTO:
        entity_ids, decoded_rows = decode_proto_format(
            working_data, schema, needed_columns=needed_columns, go_string=go_string
        )
    elif detected_format == Format.ARROW:
        entity_ids, decoded_rows = decode_arrow_format(
            working_data, schema, needed_columns=needed_columns
        )
    elif detected_format == Format.PARQUET:
        entity_ids, decoded_rows = decode_parquet_format(
            working_data, schema, needed_columns=needed_columns
        )
    else:
        raise FormatError(f"Unsupported format: {detected_format}")

    # Restrict to needed_columns when provided (smaller output schema and rows)
    output_schema = schema
    if needed_columns is not None:
        needed_set = set(needed_columns)
        output_schema = [f for f in schema if f.name in needed_set]

    if not decoded_rows:
        # Return empty DataFrame with correct schema
        from pyspark.sql.types import StringType, StructField, StructType

        # Build empty schema with entity_id + feature columns (only output_schema)
        fields = [StructField("entity_id", StringType(), True)]
        for f in output_schema:
            fields.append(StructField(f.name, StringType(), True))
        empty_schema = StructType(fields)
        return spark.createDataFrame([], empty_schema)

    # Build rows: format decoders already return only needed_columns when set
    rows = []
    for entity_id, row_data in zip(entity_ids, decoded_rows):
        row = {"entity_id": entity_id}
        row.update({k: v for k, v in row_data.items() if k != "entity_id"})
        rows.append(row)

    # Create Spark DataFrame from list of dicts
    return spark.createDataFrame(rows)


def _extract_metadata_byte(metadata_data, json_module, base64_module) -> int:
    """Extract metadata byte from JSON array with base64-encoded string.

    Expected format: JSON array with single base64-encoded string, e.g., '["BQ=="]'
    """
    if metadata_data is None:
        return 0
    # Handle pandas NA/NaN
    try:
        if hasattr(metadata_data, "isna") and metadata_data.isna():
            return 0
    except (TypeError, ValueError):
        pass
    # Handle bytes/bytearray from BinaryType cast
    if isinstance(metadata_data, (bytes, bytearray)):
        try:
            metadata_data = metadata_data.decode("utf-8")
        except (UnicodeDecodeError, ValueError):
            return 0
    if isinstance(metadata_data, str):
        try:
            parsed = json_module.loads(metadata_data)
            if isinstance(parsed, list) and len(parsed) > 0:
                decoded = base64_module.b64decode(parsed[0])
                if len(decoded) > 0:
                    return decoded[0]
        except (json_module.JSONDecodeError, ValueError, TypeError):
            pass
        return 0
    if isinstance(metadata_data, list) and len(metadata_data) > 0:
        first_item = metadata_data[0]
        if isinstance(first_item, str):
            try:
                decoded = base64_module.b64decode(first_item)
                if len(decoded) > 0:
                    return decoded[0]
            except (ValueError, TypeError):
                pass
        return 0
    return 0


def decode_mplog_dataframe(
    df: "SparkDataFrame",
    spark: "SparkSession",
    inference_host: Optional[str] = None,
    decompress: bool = True,
    features_column: str = "features",
    metadata_column: str = "metadata",
    mp_config_id_column: str = "mp_config_id",
    num_partitions: Optional[int] = None,
    max_records_per_batch: Optional[int] = None,
    needed_columns: Optional[Collection[str]] = None,
) -> "SparkDataFrame":
    """
    Decode MPLog features from a Spark DataFrame with specific column structure.

    Supports two wire formats (auto-detected per row):
    - JSON envelope: [{"encoded_features": "base64..."}] (original format)
    - v2 raw-proto: [{<proto bytes>}, {<proto bytes>}, ...] (binary framing)

    Expected DataFrame columns:
    - prism_ingested_at, prism_extracted_at, created_at
    - entities, features, metadata
    - mp_config_id, parent_entity, tracking_id, user_id
    - year, month, day, hour

    Processing is done distributed via mapInPandas so that large DataFrames (millions
    of rows, multi-MB per row) are not collected to the driver. Each partition is
    decoded on workers; only decoded (small) rows are returned.

    Args:
        df: Input Spark DataFrame with MPLog data columns
        spark: The SparkSession to use for creating the result DataFrame
        inference_host: The inference service host URL. If None, reads from INFERENCE_HOST env var.
        decompress: Whether to attempt zstd decompression
        features_column: Name of the column containing encoded features (default: "features")
        metadata_column: Name of the column containing metadata byte (default: "metadata")
        mp_config_id_column: Name of the column containing model proxy config ID (default: "mp_config_id")
        num_partitions: Number of partitions for distributed decode. Default 10000 to keep
            partition size small when rows are large (3-5 MB each). Increase if rows are small.
        max_records_per_batch: Max rows per Arrow batch in mapInPandas. When set (default 200),
            applied temporarily during this call to limit memory per batch when rows are large.
        needed_columns: Optional set or list of feature names to include. If provided, only these
            columns are decoded and returned (reduces memory and output size). If None, all schema columns are returned.

    Returns:
        Spark DataFrame with decoded features. Each row from input becomes multiple rows
        (one per entity) with entity_id as first column and features as remaining columns.
        Original row metadata (prism_ingested_at, mp_config_id, etc.) is preserved.

    Example:
        >>> from pyspark.sql import SparkSession
        >>> import inference_logging_client
        >>> spark = SparkSession.builder.appName("decode").getOrCreate()
        >>> df = spark.read.parquet("logs.parquet")
        >>> decoded_df = inference_logging_client.decode_mplog_dataframe(df, spark)
        >>> decoded_df.show()
    """
    import base64
    import json
    import os

    # Read from environment variable if not provided
    if inference_host is None:
        inference_host = os.getenv("INFERENCE_HOST", "http://localhost:8082")

    # Check if DataFrame is empty (avoid full count: use limit(1))
    if df.limit(1).count() == 0:
        from pyspark.sql.types import StructType
        return spark.createDataFrame([], StructType([]))

    # Validate required columns
    required_columns = [features_column, metadata_column, mp_config_id_column]
    df_columns = df.columns
    missing_columns = [col for col in required_columns if col not in df_columns]
    if missing_columns:
        raise ValueError(f"Missing required columns: {missing_columns}")

    # Only collect distinct (mp_config_id, metadata) to get schema keys - small payload
    distinct_df = df.select(mp_config_id_column, metadata_column).distinct()
    distinct_rows = distinct_df.collect()

    schema_cache: dict[tuple[str, int], list[FeatureInfo]] = {}
    for row in distinct_rows:
        metadata_data = row[metadata_column]
        metadata_byte = _extract_metadata_byte(metadata_data, json, base64)
        _, version, _ = unpack_metadata_byte(metadata_byte)
        if not (0 <= version <= _MAX_SCHEMA_VERSION):
            continue
        mp_config_id = row[mp_config_id_column]
        if mp_config_id is None:
            continue
        mp_config_id = str(mp_config_id)
        cache_key = (mp_config_id, version)
        if cache_key not in schema_cache:
            try:
                schema_cache[cache_key] = get_feature_schema(mp_config_id, version, inference_host)
            except Exception as e:
                warnings.warn(f"Failed to pre-fetch schema for {cache_key}: {e}", UserWarning)

    row_metadata_columns = [
        "prism_ingested_at",
        "prism_extracted_at",
        "created_at",
        "mp_config_id",
        "parent_entity",
        "tracking_id",
        "user_id",
        "year",
        "month",
        "day",
        "hour",
    ]
    _reserved_columns = {"entity_id"} | {c for c in row_metadata_columns if c in df_columns}

    # Build full output schema: entity_id + metadata cols + (optionally restricted) feature names
    all_feature_names = set()
    for feat_list in schema_cache.values():
        for f in feat_list:
            all_feature_names.add(f.name)
    if needed_columns is not None:
        needed_set = set(needed_columns)
        all_feature_names = all_feature_names & needed_set
    metadata_cols_in_schema = [c for c in row_metadata_columns if c in df_columns]
    from pyspark.sql.types import StringType, StructField, StructType
    # Map input column names to their Spark types so we can preserve them in the output
    input_field_map = {field.name: field.dataType for field in df.schema.fields}
    schema_fields = [StructField("entity_id", StringType(), True)]
    for c in metadata_cols_in_schema:
        # Preserve the original type (LongType, TimestampType, etc.)
        original_type = input_field_map.get(c, StringType())
        schema_fields.append(StructField(c, original_type, True))
    for c in sorted(all_feature_names):
        schema_fields.append(StructField(c, StringType(), True))
    full_schema = StructType(schema_fields)
    all_columns_ordered = ["entity_id"] + metadata_cols_in_schema + sorted(all_feature_names)

    def _safe_get(row, col, default=None):
        try:
            val = row[col] if col in row.index else getattr(row, col, default)
            if hasattr(val, "isna") and val.isna():
                return default
            return val
        except (KeyError, AttributeError):
            return default

    def _decode_batch(iterator):
        import pandas as pd
        for pdf in iterator:
            out_rows = []
            for idx, row in pdf.iterrows():
                features_data = _safe_get(row, features_column)
                if features_data is None:
                    continue
                metadata_data = _safe_get(row, metadata_column)
                # Handle bytes from BinaryType cast
                if isinstance(metadata_data, (bytes, bytearray)):
                    try:
                        metadata_data = metadata_data.decode("utf-8")
                    except (UnicodeDecodeError, ValueError):
                        continue
                metadata_byte = _extract_metadata_byte(metadata_data, json, base64)
                _, version, _ = unpack_metadata_byte(metadata_byte)
                if not (0 <= version <= _MAX_SCHEMA_VERSION):
                    continue
                mp_config_id = _safe_get(row, mp_config_id_column)
                if mp_config_id is None:
                    continue
                mp_config_id = str(mp_config_id)
                cache_key = (mp_config_id, version)
                feature_schema = schema_cache.get(cache_key)
                if feature_schema is None:
                    try:
                        feature_schema = get_feature_schema(mp_config_id, version, inference_host)
                    except Exception:
                        continue

                # Parse entities
                entities_val = None
                if "entities" in df_columns:
                    entities_raw = _safe_get(row, "entities")
                    if entities_raw is not None:
                        if isinstance(entities_raw, str):
                            try:
                                entities_val = json.loads(entities_raw)
                            except (json.JSONDecodeError, ValueError):
                                entities_val = [entities_raw]
                        elif isinstance(entities_raw, list):
                            entities_val = entities_raw
                        else:
                            entities_val = [entities_raw]
                _, _, format_type_num = unpack_metadata_byte(metadata_byte)
                detected_format = FORMAT_TYPE_MAP.get(format_type_num, Format.PROTO)
                parent_entity_val = None
                if "parent_entity" in df_columns:
                    parent_val = _safe_get(row, "parent_entity")
                    if parent_val is not None:
                        if isinstance(parent_val, str):
                            try:
                                parent_val = json.loads(parent_val)
                            except (json.JSONDecodeError, ValueError):
                                parent_val = [parent_val]
                        if isinstance(parent_val, list):
                            parent_entity_val = parent_val[0] if len(parent_val) == 1 else str(parent_val) if len(parent_val) > 1 else None
                        else:
                            parent_entity_val = str(parent_val)

                # --- v2 raw-proto wire format: [{<proto bytes>}, ...] ---
                if _is_raw_proto_wire_format(features_data):
                    if isinstance(features_data, str):
                        features_data = features_data.encode("utf-8", errors="surrogateescape")
                    elif isinstance(features_data, memoryview):
                        features_data = bytes(features_data)
                    entity_chunks = _split_raw_proto_entities(features_data)
                    for i, chunk in enumerate(entity_chunks):
                        if decompress:
                            chunk = _decompress_zstd(chunk)
                        try:
                            decoded_features = decode_proto_features(
                                chunk, feature_schema, needed_columns=needed_columns
                            )
                        except Exception:
                            continue
                        entity_id = str(entities_val[i]) if entities_val and i < len(entities_val) else f"entity_{i}"
                        result_row = {"entity_id": entity_id}
                        for k, v in decoded_features.items():
                            if k in _reserved_columns:
                                continue
                            if v is None:
                                result_row[k] = None
                            elif isinstance(v, (list, tuple)):
                                result_row[k] = str(v)
                            elif isinstance(v, bytes):
                                result_row[k] = v.hex()
                            else:
                                result_row[k] = str(v)
                        for col in row_metadata_columns:
                            if col in df_columns:
                                result_row[col] = _safe_get(row, col)
                        if parent_entity_val is not None:
                            result_row["parent_entity"] = str(parent_entity_val)
                        for col in all_columns_ordered:
                            if col not in result_row:
                                result_row[col] = None
                        out_rows.append(result_row)
                    continue  # next row

                # --- JSON envelope format: [{"encoded_features": "base64..."}] ---
                if isinstance(features_data, (bytes, bytearray, memoryview)):
                    try:
                        features_data = bytes(features_data).decode("utf-8")
                    except UnicodeDecodeError:
                        continue
                if isinstance(features_data, str):
                    try:
                        features_list = json.loads(features_data)
                    except (json.JSONDecodeError, ValueError, TypeError):
                        continue
                else:
                    features_list = features_data
                if not isinstance(features_list, list):
                    continue
                for i, feature_item in enumerate(features_list):
                    if not isinstance(feature_item, dict):
                        continue
                    entity_id = str(entities_val[i]) if entities_val and i < len(entities_val) else f"entity_{i}"
                    encoded_features_b64 = feature_item.get("encoded_features", "")
                    if not encoded_features_b64:
                        continue
                    try:
                        encoded_bytes = base64.b64decode(encoded_features_b64)
                    except (ValueError, TypeError):
                        continue
                    if len(encoded_bytes) == 0:
                        continue
                    working_data = encoded_bytes
                    if decompress:
                        working_data = _decompress_zstd(encoded_bytes)
                    try:
                        if detected_format == Format.ARROW:
                            decoded_features = decode_arrow_features(
                                working_data, feature_schema, needed_columns=needed_columns
                            )
                        elif detected_format == Format.PARQUET:
                            decoded_features = decode_parquet_features(
                                working_data, feature_schema, needed_columns=needed_columns
                            )
                        else:
                            decoded_features = decode_proto_features(
                                working_data, feature_schema, needed_columns=needed_columns
                            )
                    except Exception:
                        continue
                    result_row = {"entity_id": entity_id}
                    # Convert all feature values to strings for schema compatibility
                    for k, v in decoded_features.items():
                        if k in _reserved_columns:
                            continue
                        if v is None:
                            result_row[k] = None
                        elif isinstance(v, (list, tuple)):
                            result_row[k] = str(v)
                        elif isinstance(v, bytes):
                            result_row[k] = v.hex()
                        else:
                            result_row[k] = str(v)
                    for col in row_metadata_columns:
                        if col in df_columns:
                            result_row[col] = _safe_get(row, col)
                    if parent_entity_val is not None:
                        result_row["parent_entity"] = str(parent_entity_val)
                    # Fill missing schema columns with None
                    for col in all_columns_ordered:
                        if col not in result_row:
                            result_row[col] = None
                    out_rows.append(result_row)
            if out_rows:
                out_pdf = pd.DataFrame(out_rows, columns=all_columns_ordered)
                yield out_pdf

    n_partitions = num_partitions if num_partitions is not None else 10000
    # Cast binary-payload columns to BinaryType so pyarrow does not UTF-8
    # validate them at the Arrow->pandas boundary.
    from pyspark.sql import functions as _F
    from pyspark.sql.types import BinaryType as _BinaryType
    for _col_name in (features_column, metadata_column):
        if not isinstance(input_field_map.get(_col_name), _BinaryType):
            df = df.withColumn(_col_name, _F.col(_col_name).cast(_BinaryType()))
    df_repart = df.repartition(n_partitions)

    batch_limit = max_records_per_batch if max_records_per_batch is not None else 200
    prev_max_records = spark.conf.get("spark.sql.execution.arrow.maxRecordsPerBatch")
    spark.conf.set("spark.sql.execution.arrow.maxRecordsPerBatch", str(batch_limit))
    try:
        result_df = df_repart.mapInPandas(_decode_batch, full_schema)
    finally:
        spark.conf.set("spark.sql.execution.arrow.maxRecordsPerBatch", prev_max_records or "10000")

    # Reorder columns: entity_id first, then metadata, then features
    result_columns = result_df.columns
    metadata_cols = ["entity_id"]
    for col in [
        "prism_ingested_at",
        "prism_extracted_at",
        "created_at",
        "mp_config_id",
        "parent_entity",
        "tracking_id",
        "user_id",
        "year",
        "month",
        "day",
        "hour",
    ]:
        if col in result_columns:
            metadata_cols.append(col)
    feature_cols = [c for c in result_columns if c not in metadata_cols]
    column_order = metadata_cols + feature_cols
    return result_df.select(column_order)


def _normalize_schema(schema) -> "list[FeatureInfo]":
    """Accept either a list[FeatureInfo], a list of raw dicts, or the
    inference-service JSON shape ``{"data": [...]}`` and return list[FeatureInfo].

    Raw dict items must carry ``feature_name`` and ``feature_type`` keys
    (matching the inference service response). Order is preserved and used
    to assign the ``index`` of each FeatureInfo, which is the proto field
    position used by the decoder.
    """
    if schema is None:
        raise ValueError("schema must not be None")

    # Unwrap {"data": [...]} JSON shape
    if isinstance(schema, dict):
        if "data" not in schema:
            raise ValueError("schema dict must contain a 'data' key")
        items = schema["data"]
    else:
        items = schema

    if not isinstance(items, list) or not items:
        raise ValueError("schema must be a non-empty list (or dict with non-empty 'data')")

    # Already FeatureInfo objects
    if all(isinstance(it, FeatureInfo) for it in items):
        return items

    normalized: list[FeatureInfo] = []
    for idx, item in enumerate(items):
        if isinstance(item, FeatureInfo):
            normalized.append(item)
            continue
        if not isinstance(item, dict):
            raise ValueError(
                f"schema item at index {idx} must be FeatureInfo or dict, got {type(item).__name__}"
            )
        name = item.get("feature_name") or item.get("name")
        feature_type = item.get("feature_type")
        if not name or not feature_type:
            raise ValueError(
                f"schema item at index {idx} missing 'feature_name'/'name' or 'feature_type'"
            )
        normalized.append(FeatureInfo(name=name, feature_type=feature_type, index=idx))
    return normalized


def decode_mplog_proto_dataframe(
    df: "SparkDataFrame",
    spark: "SparkSession",
    schema,
    decompress: bool = True,
    features_column: str = "features",
    mp_config_id_column: str = "mp_config_id",
    num_partitions: Optional[int] = None,
    max_records_per_batch: Optional[int] = None,
    needed_columns: Optional[Collection[str]] = None,
) -> "SparkDataFrame":
    """
    Decode MPLog features from a Spark DataFrame using a caller-supplied schema.

    Format is always PROTO. No schema fetch is performed and no inference service
    is contacted. The caller is responsible for passing the correct schema for the
    encoded payloads in the DataFrame; all rows are decoded against the same schema.

    Supports both JSON envelope and v2 raw-proto wire formats (auto-detected).

    Expected DataFrame columns:
    - features (encoded payloads; JSON-array-of-base64 strings or raw-proto framed bytes)
    - mp_config_id
    - optional: entities, parent_entity
    - optional row-metadata: prism_ingested_at, prism_extracted_at, created_at,
      tracking_id, user_id, year, month, day, hour

    Args:
        df: Input Spark DataFrame.
        spark: The SparkSession to use for creating the result DataFrame.
        schema: Schema applied to all rows. Accepted shapes:
            - list[FeatureInfo]
            - list[dict] with keys 'feature_name' (or 'name') and 'feature_type'
            - dict {"data": [...]} matching the inference service JSON response
            Order is used to assign the proto field index; do not reorder.
        decompress: Whether to attempt zstd decompression on each encoded payload.
        features_column: Name of the column containing encoded features (default: "features").
        mp_config_id_column: Name of the column containing model proxy config ID
            (default: "mp_config_id"). Pass-through column; not used to look up schema.
        num_partitions: Number of partitions for distributed decode. Default 10000.
        max_records_per_batch: Max rows per Arrow batch in mapInPandas. Default 50.
        needed_columns: Optional set or list of feature names to include. If provided,
            only these columns are decoded and returned.

    Returns:
        Spark DataFrame with entity_id as first column, followed by available row-metadata
        columns, followed by feature columns.

    Example:
        >>> from pyspark.sql import SparkSession
        >>> from inference_logging_client import (
        ...     decode_mplog_proto_dataframe, get_feature_schema,
        ... )
        >>> spark = SparkSession.builder.appName("decode").getOrCreate()
        >>> df = spark.read.parquet("logs.parquet")
        >>> schema = get_feature_schema("my-model", 1)
        >>> decoded_df = decode_mplog_proto_dataframe(df, spark, schema=schema)
        >>> decoded_df.show()
    """
    import base64
    import json

    schema = _normalize_schema(schema)

    # Check if DataFrame is empty (avoid full count: use limit(1))
    if df.limit(1).count() == 0:
        from pyspark.sql.types import StructType
        return spark.createDataFrame([], StructType([]))

    # Validate required columns
    df_columns = df.columns
    required_columns = [features_column, mp_config_id_column]
    missing_columns = [c for c in required_columns if c not in df_columns]
    if missing_columns:
        raise ValueError(f"Missing required columns: {missing_columns}")

    row_metadata_columns = [
        "prism_ingested_at",
        "prism_extracted_at",
        "created_at",
        "mp_config_id",
        "parent_entity",
        "tracking_id",
        "user_id",
        "year",
        "month",
        "day",
        "hour",
    ]
    _reserved_columns = {"entity_id"} | {c for c in row_metadata_columns if c in df_columns}

    # Build output schema: entity_id + available metadata cols + feature names
    all_feature_names = {f.name for f in schema}
    if needed_columns is not None:
        all_feature_names = all_feature_names & set(needed_columns)
    metadata_cols_in_schema = [c for c in row_metadata_columns if c in df_columns]

    from pyspark.sql.types import StringType, StructField, StructType
    input_field_map = {field.name: field.dataType for field in df.schema.fields}
    schema_fields = [StructField("entity_id", StringType(), True)]
    for c in metadata_cols_in_schema:
        original_type = input_field_map.get(c, StringType())
        schema_fields.append(StructField(c, original_type, True))
    for c in sorted(all_feature_names):
        schema_fields.append(StructField(c, StringType(), True))
    full_schema = StructType(schema_fields)
    all_columns_ordered = ["entity_id"] + metadata_cols_in_schema + sorted(all_feature_names)

    # Project to only the columns we actually need on workers
    projected_cols = [
        c for c in (
            [features_column, mp_config_id_column, "entities"] + row_metadata_columns
        )
        if c in df_columns
    ]
    seen = set()
    projected_cols = [c for c in projected_cols if not (c in seen or seen.add(c))]

    # Cast features to BinaryType for safe Arrow serialization
    from pyspark.sql import functions as _F
    from pyspark.sql.types import BinaryType as _BinaryType
    if not isinstance(input_field_map.get(features_column), _BinaryType):
        df = df.withColumn(features_column, _F.col(features_column).cast(_BinaryType()))

    df_projected = df.select(*projected_cols)

    # Capture for closure
    feature_schema = schema

    # --- Hot-path precomputation (runs once per call, used by every worker
    # invocation of _decode_batch). Avoids redoing this work per row/entity. ---
    has_entities_col = "entities" in df_columns
    has_parent_entity_col = "parent_entity" in df_columns
    metadata_cols_present = [c for c in row_metadata_columns if c in df_columns]
    # Pre-built template dict avoids the "fill missing columns with None"
    # loop per entity. We copy it per output row and overwrite the cells
    # we actually have values for.
    row_template = {c: None for c in all_columns_ordered}

    def _decode_batch(iterator):
        # Imports inside the worker function — pyspark needs the function to
        # be self-contained for cloudpickle, and free-variable callable refs
        # tend to break pickling on some pyspark builds.
        import base64 as _base64
        import json as _json
        import pandas as pd

        _b64decode = _base64.b64decode
        _json_loads = _json.loads
        _decode_proto = decode_proto_features
        _decompress = _decompress_zstd

        for pdf in iterator:
            # Single conversion to list-of-dicts is dramatically faster than
            # pandas.iterrows() for wide+long DataFrames. iterrows materializes
            # a Series per row with per-cell type lookups; to_dict("records")
            # walks the underlying numpy arrays once.
            records = pdf.to_dict(orient="records")
            out_rows = []
            out_rows_append = out_rows.append  # local-bind for speed

            for row in records:
                features_data = row.get(features_column)
                if not features_data:
                    continue

                entities_val = None
                if has_entities_col:
                    entities_raw = row.get("entities")
                    if entities_raw:
                        if isinstance(entities_raw, str):
                            try:
                                parsed = _json_loads(entities_raw)
                                entities_val = parsed if isinstance(parsed, list) else [entities_raw]
                            except (ValueError, TypeError):
                                entities_val = [entities_raw]
                        elif isinstance(entities_raw, list):
                            entities_val = entities_raw
                        else:
                            entities_val = [entities_raw]

                parent_entity_val = None
                if has_parent_entity_col:
                    parent_val = row.get("parent_entity")
                    if parent_val:
                        if isinstance(parent_val, str):
                            try:
                                parent_val = _json_loads(parent_val)
                            except (ValueError, TypeError):
                                parent_val = [parent_val]
                        if isinstance(parent_val, list):
                            n_parents = len(parent_val)
                            if n_parents == 1:
                                parent_entity_val = parent_val[0]
                            elif n_parents > 1:
                                parent_entity_val = str(parent_val)
                        else:
                            parent_entity_val = str(parent_val)

                # Precompute the row-metadata snapshot once per input row —
                # every entity expansion below shares the same values.
                base_metadata = {c: row.get(c) for c in metadata_cols_present}
                if parent_entity_val is not None and "parent_entity" in metadata_cols_present:
                    base_metadata["parent_entity"] = str(parent_entity_val)

                entities_len = len(entities_val) if entities_val else 0

                # --- v2 raw-proto wire format ---
                if _is_raw_proto_wire_format(features_data):
                    if isinstance(features_data, str):
                        features_data = features_data.encode("utf-8", errors="surrogateescape")
                    elif isinstance(features_data, (memoryview,)):
                        features_data = bytes(features_data)
                    entity_chunks = _split_raw_proto_entities(features_data)
                    for i, chunk in enumerate(entity_chunks):
                        if decompress:
                            chunk = _decompress(chunk)
                        try:
                            decoded_features = _decode_proto(
                                chunk, feature_schema, needed_columns=needed_columns
                            )
                        except Exception:
                            continue
                        entity_id = (
                            str(entities_val[i]) if i < entities_len else f"entity_{i}"
                        )
                        result_row = row_template.copy()
                        result_row["entity_id"] = entity_id
                        if base_metadata:
                            result_row.update(base_metadata)
                        for k, v in decoded_features.items():
                            if k in _reserved_columns:
                                continue
                            if v is None:
                                result_row[k] = None
                            elif type(v) is str:
                                result_row[k] = v
                            elif isinstance(v, (list, tuple)):
                                result_row[k] = str(v)
                            elif isinstance(v, bytes):
                                result_row[k] = v.hex()
                            else:
                                result_row[k] = str(v)
                        out_rows_append(result_row)
                    continue

                # --- JSON envelope format ---
                if not isinstance(features_data, str):
                    continue
                try:
                    features_list = _json_loads(features_data)
                except (ValueError, TypeError):
                    continue
                if not isinstance(features_list, list):
                    continue

                for i, feature_item in enumerate(features_list):
                    if not isinstance(feature_item, dict):
                        continue
                    encoded_features_b64 = feature_item.get("encoded_features")
                    if not encoded_features_b64:
                        continue
                    try:
                        encoded_bytes = _b64decode(encoded_features_b64)
                    except (ValueError, TypeError):
                        continue
                    if not encoded_bytes:
                        continue

                    if decompress:
                        try:
                            working_data = _decompress(encoded_bytes)
                        except Exception:
                            continue
                    else:
                        working_data = encoded_bytes

                    try:
                        decoded_features = _decode_proto(
                            working_data, feature_schema, needed_columns=needed_columns
                        )
                    except Exception:
                        continue

                    entity_id = (
                        str(entities_val[i])
                        if i < entities_len
                        else f"entity_{i}"
                    )

                    # Copy the prebuilt template instead of building a fresh
                    # dict and then filling all 322 missing keys.
                    result_row = row_template.copy()
                    result_row["entity_id"] = entity_id
                    if base_metadata:
                        result_row.update(base_metadata)

                    # Stringify decoded values for output (output schema is
                    # all StringType for feature cols). Skip reserved cols.
                    for k, v in decoded_features.items():
                        if k in _reserved_columns:
                            continue
                        if v is None:
                            result_row[k] = None
                        elif type(v) is str:
                            result_row[k] = v
                        elif isinstance(v, (list, tuple)):
                            result_row[k] = str(v)
                        elif isinstance(v, bytes):
                            result_row[k] = v.hex()
                        else:
                            result_row[k] = str(v)

                    out_rows_append(result_row)

            if out_rows:
                yield pd.DataFrame(out_rows, columns=all_columns_ordered)

    n_partitions = num_partitions if num_partitions is not None else 10000
    df_repart = df_projected.repartition(n_partitions)

    batch_limit = max_records_per_batch if max_records_per_batch is not None else 50
    prev_max_records = spark.conf.get("spark.sql.execution.arrow.maxRecordsPerBatch")
    spark.conf.set("spark.sql.execution.arrow.maxRecordsPerBatch", str(batch_limit))
    try:
        result_df = df_repart.mapInPandas(_decode_batch, full_schema)
    finally:
        spark.conf.set("spark.sql.execution.arrow.maxRecordsPerBatch", prev_max_records or "10000")

    # Reorder columns: entity_id first, then metadata, then features
    result_columns = result_df.columns
    metadata_cols = ["entity_id"]
    for col in row_metadata_columns:
        if col in result_columns:
            metadata_cols.append(col)
    feature_cols = [c for c in result_columns if c not in metadata_cols]
    column_order = metadata_cols + feature_cols
    return result_df.select(column_order)


def decode_mplog_proto_csv(
    input_csv: str,
    output_csv: str,
    schema,
    decompress: bool = True,
    features_column: str = "features",
    mp_config_id_column: str = "mp_config_id",
    needed_columns: Optional[Collection[str]] = None,
) -> int:
    """
    Decode an MPLog CSV file directly to another CSV, without Spark.

    Reads the input CSV row-by-row, decodes each row's encoded entities using
    the caller-supplied PROTO schema, and writes one decoded row per entity to
    output_csv. Pure-Python; uses only csv/json/base64 + decode_proto_features.

    Expected input columns: features, mp_config_id, optionally entities,
    parent_entity, and the row-metadata columns (prism_ingested_at, etc).

    Args:
        input_csv: Path to the input CSV.
        output_csv: Path where the decoded CSV will be written.
        schema: Same shapes accepted by decode_mplog_proto_dataframe:
            list[FeatureInfo], list[dict], or {"data": [...]}.
        decompress: Attempt zstd decompression per encoded payload.
        features_column: Column with the encoded features JSON.
        mp_config_id_column: Pass-through column name.
        needed_columns: Optional set of feature names to keep.

    Returns:
        Number of decoded rows written.
    """
    import base64
    import csv as _csv
    import json
    import sys as _sys

    # MPLog features cells can be multi-MB; lift the csv field-size cap.
    try:
        _csv.field_size_limit(_sys.maxsize)
    except OverflowError:
        _csv.field_size_limit(2**31 - 1)

    schema_list = _normalize_schema(schema)
    needed_set = set(needed_columns) if needed_columns is not None else None

    feature_names = [f.name for f in schema_list]
    if needed_set is not None:
        feature_names = [n for n in feature_names if n in needed_set]

    row_metadata_columns = [
        "prism_ingested_at",
        "prism_extracted_at",
        "created_at",
        "mp_config_id",
        "parent_entity",
        "tracking_id",
        "user_id",
        "year",
        "month",
        "day",
        "hour",
    ]

    with open(input_csv, "r", newline="", encoding="utf-8") as f_in:
        reader = _csv.DictReader(f_in)
        if reader.fieldnames is None:
            raise ValueError(f"Input CSV {input_csv} has no header row")
        input_columns = set(reader.fieldnames)

        if features_column not in input_columns:
            raise ValueError(f"Missing required column: {features_column}")

        present_metadata_cols = [c for c in row_metadata_columns if c in input_columns]
        out_columns = ["entity_id"] + present_metadata_cols + sorted(feature_names)

        n_written = 0
        with open(output_csv, "w", newline="", encoding="utf-8") as f_out:
            writer = _csv.DictWriter(f_out, fieldnames=out_columns, extrasaction="ignore")
            writer.writeheader()

            for row in reader:
                features_data = row.get(features_column)
                if not features_data:
                    continue
                try:
                    features_list = json.loads(features_data)
                except (json.JSONDecodeError, ValueError, TypeError):
                    continue
                if not isinstance(features_list, list):
                    continue

                entities_val = None
                if "entities" in input_columns:
                    entities_raw = row.get("entities")
                    if entities_raw:
                        try:
                            parsed = json.loads(entities_raw)
                            entities_val = parsed if isinstance(parsed, list) else [parsed]
                        except (json.JSONDecodeError, ValueError):
                            entities_val = [entities_raw]

                parent_entity_val = None
                if "parent_entity" in input_columns:
                    parent_raw = row.get("parent_entity")
                    if parent_raw:
                        try:
                            parsed = json.loads(parent_raw)
                            if isinstance(parsed, list):
                                parent_entity_val = (
                                    parsed[0] if len(parsed) == 1
                                    else str(parsed) if len(parsed) > 1
                                    else None
                                )
                            else:
                                parent_entity_val = parsed
                        except (json.JSONDecodeError, ValueError):
                            parent_entity_val = parent_raw

                base_metadata = {c: row.get(c) for c in present_metadata_cols}

                for i, feature_item in enumerate(features_list):
                    if not isinstance(feature_item, dict):
                        continue
                    encoded_b64 = feature_item.get("encoded_features", "")
                    if not encoded_b64:
                        continue
                    try:
                        encoded_bytes = base64.b64decode(encoded_b64)
                    except (ValueError, TypeError):
                        continue
                    if not encoded_bytes:
                        continue

                    working_data = encoded_bytes
                    if decompress:
                        try:
                            working_data = _decompress_zstd(encoded_bytes)
                        except Exception:
                            continue

                    try:
                        decoded = decode_proto_features(
                            working_data, schema_list, needed_columns=needed_set
                        )
                    except Exception:
                        continue

                    entity_id = (
                        str(entities_val[i])
                        if entities_val and i < len(entities_val)
                        else f"entity_{i}"
                    )

                    out_row = {"entity_id": entity_id}
                    out_row.update(base_metadata)
                    if parent_entity_val is not None and "parent_entity" in present_metadata_cols:
                        out_row["parent_entity"] = parent_entity_val
                    for k, v in decoded.items():
                        if needed_set is not None and k not in needed_set:
                            continue
                        if v is None:
                            out_row[k] = ""
                        elif isinstance(v, (list, tuple)):
                            out_row[k] = str(v)
                        elif isinstance(v, bytes):
                            out_row[k] = v.hex()
                        else:
                            out_row[k] = v
                    writer.writerow(out_row)
                    n_written += 1

    return n_written
