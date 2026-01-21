"""
Inference Logging Client - Decode MPLog feature logs from proto, arrow, or parquet format.

This package provides functionality to:
1. Decode MPLog feature logs from various encoding formats (proto, arrow, parquet)
2. Fetch feature schemas from inference API
3. Convert decoded logs to pandas DataFrames

Main functions:
    - decode_mplog: Decode MPLog bytes to a DataFrame
    - decode_mplog_dataframe: Decode MPLog features from a DataFrame
    - get_mplog_metadata: Extract metadata from MPLog bytes
"""

from typing import Optional

import pandas as pd

from .types import Format, FeatureInfo, DecodedMPLog, FORMAT_TYPE_MAP
from .io import get_feature_schema, parse_mplog_protobuf, get_mplog_metadata
from .formats import decode_proto_format, decode_arrow_format, decode_parquet_format
from .utils import format_dataframe_floats, get_format_name, unpack_metadata_byte

__version__ = "0.1.0"

__all__ = [
    "decode_mplog",
    "decode_mplog_dataframe",
    "get_mplog_metadata",
    "get_feature_schema",
    "Format",
    "FeatureInfo",
    "DecodedMPLog",
    "get_format_name",
    "unpack_metadata_byte",
]


def decode_mplog(
    log_data: bytes,
    model_proxy_id: str,
    version: int,
    format_type: Optional[Format] = None,
    inference_host: Optional[str] = None,
    decompress: bool = True
) -> pd.DataFrame:
    """
    Main function to decode MPLog bytes to a DataFrame.
    
    Args:
        log_data: The MPLog bytes (possibly compressed)
        model_proxy_id: The model proxy config ID
        version: The schema version
        format_type: The encoding format (proto, arrow, parquet). If None, auto-detect from metadata.
        inference_host: The inference service host URL. If None, reads from INFERENCE_HOST env var.
        decompress: Whether to attempt zstd decompression
    
    Returns:
        pandas DataFrame with entity_id as first column and features as remaining columns
    
    Example:
        >>> import inference_logging_client
        >>> with open("log.bin", "rb") as f:
        ...     data = f.read()
        >>> df = inference_logging_client.decode_mplog(
        ...     log_data=data,
        ...     model_proxy_id="my-model",
        ...     version=1
        ... )
        >>> print(df.head())
    """
    import os
    import zstandard as zstd
    
    # Read from environment variable if not provided
    if inference_host is None:
        inference_host = os.getenv("INFERENCE_HOST", "http://localhost:8082")
    
    # Attempt decompression if enabled
    working_data = log_data
    if decompress:
        try:
            # Try to detect if data is zstd compressed
            # zstd magic number: 0x28 0xB5 0x2F 0xFD
            if len(log_data) >= 4 and log_data[:4] == b'\x28\xB5\x2F\xFD':
                decompressor = zstd.ZstdDecompressor()
                working_data = decompressor.decompress(log_data)
        except Exception:
            # Not compressed or decompression failed, use original data
            working_data = log_data
    
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
    
    # Fetch schema from inference service
    schema = get_feature_schema(model_proxy_id, version, inference_host)
    
    # Decode based on format
    if detected_format == Format.PROTO:
        entity_ids, decoded_rows = decode_proto_format(working_data, schema)
    elif detected_format == Format.ARROW:
        entity_ids, decoded_rows = decode_arrow_format(working_data, schema)
    elif detected_format == Format.PARQUET:
        entity_ids, decoded_rows = decode_parquet_format(working_data, schema)
    else:
        raise ValueError(f"Unsupported format: {detected_format}")
    
    if not decoded_rows:
        # Return empty DataFrame with correct columns
        columns = ["entity_id"] + [f.name for f in schema]
        return pd.DataFrame(columns=columns)
    
    # Build DataFrame
    df = pd.DataFrame(decoded_rows)
    
    # Insert entity_id as first column
    df.insert(0, "entity_id", entity_ids)
    
    return df


def decode_mplog_dataframe(
    df: pd.DataFrame,
    inference_host: Optional[str] = None,
    decompress: bool = True,
    features_column: str = "features",
    metadata_column: str = "metadata",
    mp_config_id_column: str = "mp_config_id"
) -> pd.DataFrame:
    """
    Decode MPLog features from a DataFrame with specific column structure.
    
    Expected DataFrame columns:
    - prism_ingested_at, prism_extracted_at, created_at
    - entities, features, metadata
    - mp_config_id, parent_entity, tracking_id, user_id
    - year, month, day, hour
    
    Args:
        df: Input DataFrame with MPLog data columns
        inference_host: The inference service host URL. If None, reads from INFERENCE_HOST env var.
        decompress: Whether to attempt zstd decompression
        features_column: Name of the column containing encoded features (default: "features")
        metadata_column: Name of the column containing metadata byte (default: "metadata")
        mp_config_id_column: Name of the column containing model proxy config ID (default: "mp_config_id")
    
    Returns:
        pandas DataFrame with decoded features. Each row from input becomes multiple rows
        (one per entity) with entity_id as first column and features as remaining columns.
        Original row metadata (prism_ingested_at, mp_config_id, etc.) is preserved.
    
    Example:
        >>> import pandas as pd
        >>> import inference_logging_client
        >>> df = pd.read_parquet("logs.parquet")
        >>> decoded_df = inference_logging_client.decode_mplog_dataframe(df)
        >>> print(decoded_df.head())
    """
    import os
    import sys
    import json
    import base64
    
    # Read from environment variable if not provided
    if inference_host is None:
        inference_host = os.getenv("INFERENCE_HOST", "http://localhost:8082")
    
    if df.empty:
        return pd.DataFrame()
    
    # Validate required columns
    required_columns = [features_column, metadata_column, mp_config_id_column]
    missing_columns = [col for col in required_columns if col not in df.columns]
    if missing_columns:
        raise ValueError(f"Missing required columns: {missing_columns}")
    
    all_decoded_rows = []
    
    for idx, row in df.iterrows():
        # Extract features bytes
        features_data = row[features_column]
        if pd.isna(features_data):
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
                    features_bytes = features_data.encode('utf-8')
        elif isinstance(features_data, (bytearray, memoryview)):
            features_bytes = bytes(features_data)
        else:
            continue
        
        if features_bytes is None or len(features_bytes) == 0:
            continue
        
        # Extract metadata byte
        metadata_data = row[metadata_column]
        metadata_byte = 0
        if not pd.isna(metadata_data):
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
        
        # Extract mp_config_id
        mp_config_id = row[mp_config_id_column]
        if pd.isna(mp_config_id):
            continue
        mp_config_id = str(mp_config_id)
        
        # Decode this row's features
        try:
            decoded_df = decode_mplog(
                log_data=features_bytes,
                model_proxy_id=mp_config_id,
                version=version,
                format_type=None,  # Auto-detect from metadata
                inference_host=inference_host,
                decompress=decompress
            )
            
            # Add original row metadata to each decoded entity row
            if not decoded_df.empty:
                # Preserve original metadata columns
                metadata_columns = [
                    "prism_ingested_at", "prism_extracted_at", "created_at",
                    "mp_config_id", "parent_entity", "tracking_id", "user_id",
                    "year", "month", "day", "hour"
                ]
                
                for col in metadata_columns:
                    if col in df.columns:
                        decoded_df[col] = row[col]
                
                # Update entity_id from entities column if available and matches count
                if "entities" in df.columns and not pd.isna(row["entities"]):
                    # entities might be a list or string representation
                    entities_val = row["entities"]
                    if isinstance(entities_val, str):
                        try:
                            entities_val = json.loads(entities_val)
                        except (json.JSONDecodeError, ValueError):
                            entities_val = [entities_val]
                    elif not isinstance(entities_val, list):
                        entities_val = [entities_val]
                    
                    # Match entities with decoded rows (only if counts match)
                    if len(entities_val) == len(decoded_df):
                        decoded_df["entity_id"] = entities_val
                
                # Add parent_entity if it exists
                if "parent_entity" in df.columns and not pd.isna(row["parent_entity"]):
                    parent_val = row["parent_entity"]
                    if isinstance(parent_val, str):
                        try:
                            parent_val = json.loads(parent_val)
                        except (json.JSONDecodeError, ValueError):
                            parent_val = [parent_val]
                    if isinstance(parent_val, list):
                        # If list, use first element or join if multiple
                        if len(parent_val) == 1:
                            decoded_df["parent_entity"] = parent_val[0]
                        elif len(parent_val) > 1:
                            decoded_df["parent_entity"] = str(parent_val)
                        else:
                            decoded_df["parent_entity"] = None
                    else:
                        decoded_df["parent_entity"] = parent_val
                
                all_decoded_rows.append(decoded_df)
        except Exception as e:
            # Log error but continue processing other rows
            print(f"Warning: Failed to decode row {idx}: {e}", file=sys.stderr)
            continue
    
    if not all_decoded_rows:
        return pd.DataFrame()
    
    # Combine all decoded DataFrames
    result_df = pd.concat(all_decoded_rows, ignore_index=True)
    
    # Reorder columns: entity_id first, then metadata columns, then features
    metadata_cols = ["entity_id"]
    for col in ["prism_ingested_at", "prism_extracted_at", "created_at",
                "mp_config_id", "parent_entity", "tracking_id", "user_id",
                "year", "month", "day", "hour"]:
        if col in result_df.columns:
            metadata_cols.append(col)
    
    feature_cols = [col for col in result_df.columns if col not in metadata_cols]
    column_order = metadata_cols + feature_cols
    
    return result_df[column_order]
