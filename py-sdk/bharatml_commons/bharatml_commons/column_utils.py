"""
Column and data processing utilities for BharatML Stack SDKs
"""

import re
from typing import List, Tuple


def clean_column_name(column_name: str) -> str:
    """
    Cleans a column name to be Delta Table compatible:
    - Replaces unsupported special characters with "_"
    - Ensures it doesn't start with a number
    - Converts to lowercase

    Args:
        column_name: Original column name to clean

    Returns:
        Cleaned column name compatible with Delta Table standards
    """
    # Define a regex pattern to match invalid characters
    invalid_chars = r"[^a-zA-Z0-9_]"
    cleaned_name = re.sub(invalid_chars, "_", column_name)  # Replace invalid chars with "_"

    # Ensure it doesn't start with a number
    if cleaned_name[0].isdigit():
        cleaned_name = "_" + cleaned_name

    return cleaned_name.lower()  # Convert to lowercase (Delta standard)


def generate_renamed_column(table_name: str, source_type: str, feature_col: str) -> str:
    """
    Generate renamed column based on source type and table name
    
    Args:
        table_name: Name of the source table/path
        source_type: Type of data source (TABLE, PARQUET_*, DELTA_*)
        feature_col: Original feature column name
        
    Returns:
        Renamed column following naming conventions
    """
    if source_type == "TABLE":
        rename_feature_col = table_name.split(".")[1] + "___" + feature_col
    elif source_type in [
        "PARQUET_GCS", "PARQUET_S3", "PARQUET_ADLS",  # Parquet sources
        "DELTA_GCS", "DELTA_S3", "DELTA_ADLS"         # Delta sources
    ]:
        # Extract the last part of the path and clean it
        path_parts = table_name.split("gs://")[1].strip("/ ").split("/")
        clean_table_name = clean_column_name(path_parts[-1])
        rename_feature_col = clean_table_name + "___" + feature_col
    else:
        raise ValueError(f"Unsupported source type: {source_type} for table: {table_name}")
    return rename_feature_col 

def test_function():
    print("test_function 23")
