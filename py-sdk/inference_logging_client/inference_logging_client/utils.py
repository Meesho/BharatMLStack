"""Utility functions for type normalization and formatting."""

from typing import Optional

# Feature type to size mapping for scalar types
SCALAR_TYPE_SIZES = {
    "INT8": 1, "I8": 1,
    "INT16": 2, "I16": 2, "SHORT": 2,
    "INT32": 4, "I32": 4, "INT": 4,
    "INT64": 8, "I64": 8, "LONG": 8,
    "UINT8": 1, "U8": 1,
    "UINT16": 2, "U16": 2,
    "UINT32": 4, "U32": 4,
    "UINT64": 8, "U64": 8,
    "FP8E5M2": 1, "FP8E4M3": 1,
    "FP16": 2, "FLOAT16": 2, "F16": 2,
    "FP32": 4, "FLOAT32": 4, "F32": 4, "FLOAT": 4,
    "FP64": 8, "FLOAT64": 8, "F64": 8, "DOUBLE": 8,
    "BOOL": 1, "BOOLEAN": 1,
}

# Types that use 2-byte size prefix
SIZED_TYPES = {
    "STRING", "STR",
    "BYTES",
    # Vector types
    "FP8E5M2VECTOR", "FP8E4M3VECTOR",
    "FP16VECTOR", "FLOAT16VECTOR",
    "FP32VECTOR", "FLOAT32VECTOR",
    "FP64VECTOR", "FLOAT64VECTOR",
    "INT8VECTOR", "INT16VECTOR", "INT32VECTOR", "INT64VECTOR",
    "UINT8VECTOR", "UINT16VECTOR", "UINT32VECTOR", "UINT64VECTOR",
    "STRINGVECTOR", "BOOLVECTOR",
    # With underscore
    "VECTOR_FP8E5M2", "VECTOR_FP8E4M3",
    "VECTOR_FP16", "VECTOR_FLOAT16",
    "VECTOR_FP32", "VECTOR_FLOAT32",
    "VECTOR_FP64", "VECTOR_FLOAT64",
    "VECTOR_INT8", "VECTOR_INT16", "VECTOR_INT32", "VECTOR_INT64",
    "VECTOR_UINT8", "VECTOR_UINT16", "VECTOR_UINT32", "VECTOR_UINT64",
    "VECTOR_STRING", "VECTOR_BOOL",
    # DataType prefix variants
    "DATATYPESTRING",
    "DATATYPEBYTES",
    "DATATYPEFP8E5M2VECTOR", "DATATYPEFP8E4M3VECTOR",
    "DATATYPEFP16VECTOR", "DATATYPEFP32VECTOR", "DATATYPEFP64VECTOR",
    "DATATYPEINT8VECTOR", "DATATYPEINT16VECTOR", "DATATYPEINT32VECTOR", "DATATYPEINT64VECTOR",
    "DATATYPEUINT8VECTOR", "DATATYPEUINT16VECTOR", "DATATYPEUINT32VECTOR", "DATATYPEUINT64VECTOR",
    "DATATYPESTRINGVECTOR", "DATATYPEBOOLVECTOR",
}


def normalize_type(feature_type: str) -> str:
    """Normalize feature type string for consistent comparison."""
    return feature_type.upper().replace("_", "").replace("DATATYPE", "")


def is_sized_type(feature_type: str) -> bool:
    """Check if the feature type requires a 2-byte size prefix."""
    normalized = normalize_type(feature_type)
    return (normalized in {"STRING", "STR"} or 
            "VECTOR" in normalized.upper() or
            normalized in SIZED_TYPES)


def get_scalar_size(feature_type: str) -> Optional[int]:
    """Get the byte size for a scalar type."""
    normalized = normalize_type(feature_type)
    return SCALAR_TYPE_SIZES.get(normalized)


def format_float(value: float) -> float:
    """
    Format float to 6 decimal places without scientific notation.
    
    Returns the float value formatted to 6 decimal places. For special values
    (inf, -inf, nan), returns them as-is. For regular floats, rounds to 6 decimals.
    """
    import math
    if math.isnan(value) or math.isinf(value):
        return value
    # Round to 6 decimal places and convert back to float
    # This ensures no scientific notation in string representation
    return round(value, 6)


def format_dataframe_floats(df):
    """
    Format all float columns in DataFrame to 6 decimal places.
    This ensures no scientific notation in output.
    """
    import pandas as pd
    df = df.copy()
    for col in df.columns:
        if df[col].dtype == 'float64' or df[col].dtype == 'float32':
            df[col] = df[col].apply(lambda x: format_float(x) if pd.notna(x) else x)
    return df


def unpack_metadata_byte(metadata_byte: int) -> tuple[bool, int, int]:
    """
    Unpack compression flag, version, and format type from metadata byte.
    Bit layout:
      - Bits 0-1: compression flag (00=disabled, 01=enabled, 10/11=reserved)
      - Bits 2-5: version (4 bits, 0-15)
      - Bits 6-7: format type (00=proto, 01=arrow, 10=parquet)
    """
    compression_enabled = (metadata_byte & 0x01) == 1
    version = (metadata_byte >> 2) & 0x0F  # 4 bits for version (0-15)
    format_type = (metadata_byte >> 6) & 0x03  # 2 bits for format type
    return compression_enabled, version, format_type


def get_format_name(format_type: int) -> str:
    """Get human-readable format name from format type int."""
    from .types import FORMAT_TYPE_PROTO, FORMAT_TYPE_ARROW, FORMAT_TYPE_PARQUET
    names = {
        FORMAT_TYPE_PROTO: "proto",
        FORMAT_TYPE_ARROW: "arrow",
        FORMAT_TYPE_PARQUET: "parquet",
    }
    return names.get(format_type, f"unknown({format_type})")
