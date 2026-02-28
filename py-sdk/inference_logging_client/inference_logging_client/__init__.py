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
from .formats import decode_arrow_format, decode_parquet_format, decode_proto_format
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


def decode_mplog_dataframe(
    df: "SparkDataFrame",
    spark: "SparkSession",
    inference_host: Optional[str] = None,
    decompress: bool = True,
    features_column: str = "features",
    metadata_column: str = "metadata",
    mp_config_id_column: str = "mp_config_id",
) -> "SparkDataFrame":
    """
    Decode MPLog features from a Spark DataFrame with specific column structure.

    Expected DataFrame columns:
    - prism_ingested_at, prism_extracted_at, created_at
    - entities, features, metadata
    - mp_config_id, parent_entity, tracking_id, user_id
    - year, month, day, hour

    Note: This function collects the DataFrame to the driver for processing.
    For very large datasets, consider partitioning and processing in smaller batches.

    Args:
        df: Input Spark DataFrame with MPLog data columns
        spark: The SparkSession to use for creating the result DataFrame
        inference_host: The inference service host URL. If None, reads from INFERENCE_HOST env var.
        decompress: Whether to attempt zstd decompression
        features_column: Name of the column containing encoded features (default: "features")
        metadata_column: Name of the column containing metadata byte (default: "metadata")
        mp_config_id_column: Name of the column containing model proxy config ID (default: "mp_config_id")

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

    # Track decode errors for summary
    decode_errors = []

    # Check if DataFrame is empty
    if df.count() == 0:
        from pyspark.sql.types import StructType
        return spark.createDataFrame([], StructType([]))

    # Validate required columns
    required_columns = [features_column, metadata_column, mp_config_id_column]
    df_columns = df.columns
    missing_columns = [col for col in required_columns if col not in df_columns]
    if missing_columns:
        raise ValueError(f"Missing required columns: {missing_columns}")

    # Collect to driver for processing
    # Note: For large datasets, consider using mapInPandas or processing in partitions
    rows = df.collect()

    # Pre-fetch schemas for unique (mp_config_id, version) combinations to avoid
    # redundant HTTP requests during row iteration.
    # Key: (mp_config_id, version) only - host/path intentionally excluded as schemas are canonical
    schema_cache: dict[tuple[str, int], list[FeatureInfo]] = {}

    # First pass: collect unique (mp_config_id, version) pairs
    for row in rows:
        # Extract metadata byte to get version
        metadata_data = row[metadata_column]
        metadata_byte = 0
        if metadata_data is not None:
            if isinstance(metadata_data, (int, float)):
                metadata_byte = int(metadata_data)
            elif isinstance(metadata_data, bytes) and len(metadata_data) > 0:
                metadata_byte = metadata_data[0]
            elif isinstance(metadata_data, (bytearray, memoryview)) and len(metadata_data) > 0:
                metadata_byte = metadata_data[0]
            elif isinstance(metadata_data, str):
                try:
                    metadata_byte = int(metadata_data)
                except ValueError:
                    pass

        _, version, _ = unpack_metadata_byte(metadata_byte)

        # Skip invalid versions
        if not (0 <= version <= _MAX_SCHEMA_VERSION):
            continue

        # Extract mp_config_id
        mp_config_id = row[mp_config_id_column]
        if mp_config_id is None:
            continue
        mp_config_id = str(mp_config_id)

        cache_key = (mp_config_id, version)
        if cache_key not in schema_cache:
            # Pre-fetch schema and store in local cache
            try:
                schema_cache[cache_key] = get_feature_schema(mp_config_id, version, inference_host)
            except Exception as e:
                # Log warning but don't fail - will be caught again in main loop
                warnings.warn(f"Failed to pre-fetch schema for {cache_key}: {e}", UserWarning)

    all_decoded_rows = []

    for idx, row in enumerate(rows):
        # Extract features bytes
        features_data = row[features_column]
        if features_data is None:
            continue

        # Convert features to bytes (handle base64, hex, or raw bytes)
        features_bytes = None
        if isinstance(features_data, bytes):
            features_bytes = features_data
        elif isinstance(features_data, str):
            # Try base64 first
            try:
                features_bytes = base64.b64decode(features_data)
            except Exception:
                # Try hex
                try:
                    features_bytes = bytes.fromhex(features_data)
                except Exception:
                    # Try UTF-8 encoding
                    features_bytes = features_data.encode("utf-8")
        elif isinstance(features_data, (bytearray, memoryview)):
            features_bytes = bytes(features_data)
        else:
            continue

        if features_bytes is None or len(features_bytes) == 0:
            continue

        # Extract metadata byte
        metadata_data = row[metadata_column]
        metadata_byte = 0
        if metadata_data is not None:
            if isinstance(metadata_data, (int, float)):
                metadata_byte = int(metadata_data)
            elif isinstance(metadata_data, bytes) and len(metadata_data) > 0:
                metadata_byte = metadata_data[0]
            elif isinstance(metadata_data, (bytearray, memoryview)) and len(metadata_data) > 0:
                metadata_byte = metadata_data[0]
            elif isinstance(metadata_data, str):
                try:
                    metadata_byte = int(metadata_data)
                except ValueError:
                    pass

        # Extract version from metadata byte
        _, version, _ = unpack_metadata_byte(metadata_byte)

        # Validate version range
        if not (0 <= version <= _MAX_SCHEMA_VERSION):
            warnings.warn(
                f"Row {idx}: Version {version} extracted from metadata is out of valid range (0-{_MAX_SCHEMA_VERSION}). "
                f"This may indicate corrupted metadata.",
                UserWarning,
            )
            continue

        # Extract mp_config_id
        mp_config_id = row[mp_config_id_column]
        if mp_config_id is None:
            continue
        mp_config_id = str(mp_config_id)

        # Lookup cached schema
        cache_key = (mp_config_id, version)
        cached_schema = schema_cache.get(cache_key)

        # Decode this row's features
        try:
            # Attempt decompression if enabled
            working_data = features_bytes
            if decompress:
                working_data = _decompress_zstd(features_bytes)

            # Parse protobuf to get format from metadata
            parsed = parse_mplog_protobuf(working_data)
            if parsed.format_type in FORMAT_TYPE_MAP:
                detected_format = FORMAT_TYPE_MAP[parsed.format_type]
            else:
                detected_format = Format.PROTO

            # Use cached schema or fetch
            feature_schema = cached_schema
            if feature_schema is None:
                feature_schema = get_feature_schema(mp_config_id, version, inference_host)

            # Decode based on format
            if detected_format == Format.PROTO:
                entity_ids, decoded_feature_rows = decode_proto_format(working_data, feature_schema)
            elif detected_format == Format.ARROW:
                entity_ids, decoded_feature_rows = decode_arrow_format(working_data, feature_schema)
            elif detected_format == Format.PARQUET:
                entity_ids, decoded_feature_rows = decode_parquet_format(working_data, feature_schema)
            else:
                raise FormatError(f"Unsupported format: {detected_format}")

            # Add original row metadata to each decoded entity row
            if decoded_feature_rows:
                # Metadata columns to preserve
                metadata_columns = [
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

                # Get entities from row if available
                entities_val = None
                if "entities" in df_columns:
                    entities_val = row["entities"]
                    if entities_val is not None:
                        if isinstance(entities_val, str):
                            try:
                                entities_val = json.loads(entities_val)
                            except (json.JSONDecodeError, ValueError):
                                entities_val = [entities_val]
                        elif not isinstance(entities_val, list):
                            entities_val = [entities_val]

                # Process parent_entity
                parent_entity_val = None
                if "parent_entity" in df_columns and row["parent_entity"] is not None:
                    parent_val = row["parent_entity"]
                    if isinstance(parent_val, str):
                        try:
                            parent_val = json.loads(parent_val)
                        except (json.JSONDecodeError, ValueError):
                            parent_val = [parent_val]
                    if isinstance(parent_val, list):
                        if len(parent_val) == 1:
                            parent_entity_val = parent_val[0]
                        elif len(parent_val) > 1:
                            parent_entity_val = str(parent_val)
                        else:
                            parent_entity_val = None
                    else:
                        parent_entity_val = parent_val

                for i, (entity_id, feature_row) in enumerate(zip(entity_ids, decoded_feature_rows)):
                    result_row = {"entity_id": entity_id}
                    result_row.update(feature_row)

                    # Add metadata columns
                    for col in metadata_columns:
                        if col in df_columns:
                            result_row[col] = row[col]

                    # Override entity_id from entities column if available and matches count
                    if entities_val and len(entities_val) == len(entity_ids):
                        result_row["entity_id"] = entities_val[i]

                    # Set parent_entity
                    if parent_entity_val is not None:
                        result_row["parent_entity"] = parent_entity_val

                    all_decoded_rows.append(result_row)

        except Exception as e:
            # Track error but continue processing other rows
            decode_errors.append((idx, str(e)))
            warnings.warn(f"Failed to decode row {idx}: {e}", UserWarning)
            continue

    if not all_decoded_rows:
        from pyspark.sql.types import StructType
        return spark.createDataFrame([], StructType([]))

    # Create Spark DataFrame from all decoded rows
    result_df = spark.createDataFrame(all_decoded_rows)

    # Reorder columns: entity_id first, then metadata columns, then features
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

    feature_cols = [col for col in result_columns if col not in metadata_cols]
    column_order = metadata_cols + feature_cols

    return result_df.select(column_order)
