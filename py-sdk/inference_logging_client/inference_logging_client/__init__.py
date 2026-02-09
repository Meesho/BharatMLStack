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
from typing import TYPE_CHECKING, Optional

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

__version__ = "0.1.0"

# Maximum supported schema version (4 bits = 0-15)
_MAX_SCHEMA_VERSION = 15

__all__ = [
    "decode_mplog",
    "decode_mplog_dataframe",
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


def decode_mplog(
    log_data: bytes,
    model_proxy_id: str,
    version: int,
    spark: "SparkSession",
    format_type: Optional[Format] = None,
    inference_host: Optional[str] = None,
    decompress: bool = True,
    schema: Optional[list] = None,
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

    # Decode based on format
    if detected_format == Format.PROTO:
        entity_ids, decoded_rows = decode_proto_format(working_data, schema)
    elif detected_format == Format.ARROW:
        entity_ids, decoded_rows = decode_arrow_format(working_data, schema)
    elif detected_format == Format.PARQUET:
        entity_ids, decoded_rows = decode_parquet_format(working_data, schema)
    else:
        raise FormatError(f"Unsupported format: {detected_format}")

    if not decoded_rows:
        # Return empty DataFrame with correct schema
        from pyspark.sql.types import StringType, StructField, StructType

        # Build empty schema with entity_id + feature columns
        fields = [StructField("entity_id", StringType(), True)]
        for f in schema:
            fields.append(StructField(f.name, StringType(), True))
        empty_schema = StructType(fields)
        return spark.createDataFrame([], empty_schema)

    # Build rows with entity_id as first field
    rows = []
    for entity_id, row_data in zip(entity_ids, decoded_rows):
        row = {"entity_id": entity_id}
        row.update(row_data)
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
) -> "SparkDataFrame":
    """
    Decode MPLog features from a Spark DataFrame with specific column structure.

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
    # Build full output schema: entity_id + metadata cols + all feature names from all schemas
    all_feature_names = set()
    for feat_list in schema_cache.values():
        for f in feat_list:
            all_feature_names.add(f.name)
    metadata_cols_in_schema = [c for c in row_metadata_columns if c in df_columns]
    from pyspark.sql.types import StringType, StructField, StructType
    schema_fields = [StructField("entity_id", StringType(), True)]
    for c in metadata_cols_in_schema:
        schema_fields.append(StructField(c, StringType(), True))
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
                if isinstance(features_data, str):
                    try:
                        features_list = json.loads(features_data)
                    except (json.JSONDecodeError, ValueError, TypeError):
                        continue
                else:
                    features_list = features_data
                if not isinstance(features_list, list):
                    continue
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
                            parent_entity_val = parent_val
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
                            decoded_features = decode_arrow_features(working_data, feature_schema)
                        elif detected_format == Format.PARQUET:
                            decoded_features = decode_parquet_features(working_data, feature_schema)
                        else:
                            decoded_features = decode_proto_features(working_data, feature_schema)
                    except Exception:
                        continue
                    result_row = {"entity_id": entity_id}
                    result_row.update(decoded_features)
                    for col in row_metadata_columns:
                        if col in df_columns:
                            result_row[col] = _safe_get(row, col)
                    if parent_entity_val is not None:
                        result_row["parent_entity"] = parent_entity_val
                    # Fill missing schema columns with None
                    for col in all_columns_ordered:
                        if col not in result_row:
                            result_row[col] = None
                    out_rows.append(result_row)
            if out_rows:
                out_pdf = pd.DataFrame(out_rows, columns=all_columns_ordered)
                yield out_pdf

    n_partitions = num_partitions if num_partitions is not None else 10000
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
