"""
Public API: decode_single_config and decode_multi_config.

Orchestrate the full pipeline: JVM JSON parse, version column, plan build/broadcast,
mapPartitions decode, output schema. All complexity is internal; caller provides
a DataFrame and gets decoded output.
"""

from __future__ import annotations

import os
import warnings
from typing import Any, Optional, Set, Tuple, Union

from .decode_udf import make_partition_decoder
from .plan_builder import build_and_broadcast_plans, build_output_schema
from .spark_utils import (
    add_version_column,
    collect_distinct_pairs,
    parse_json_columns,
    prepare_partitions,
)

# Hardcoded internally; not exposed to caller
_MP_CONFIG_ID_COLUMN = "mp_config_id"
_DEFAULT_NUM_PARTITIONS = 50_000

ROW_METADATA_COLUMNS = [
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


def _decode_internal(
    df: Any,
    spark: Any,
    inference_host: Optional[str],
    needed_columns: Optional[Set[str]],
    num_partitions: Optional[int],
    stringify_features: bool,
    features_column: str,
    metadata_column: str,
) -> Union[Any, Tuple[Any, dict]]:
    """Single pipeline: parse JSON, add version, build/broadcast plans, mapPartitions decode."""
    inference_host = inference_host or os.getenv("INFERENCE_HOST", "http://localhost:8082")

    if df.limit(1).count() == 0:
        metadata_cols_empty = [c for c in ROW_METADATA_COLUMNS if c in df.columns]
        empty_schema = build_output_schema(df, metadata_cols_empty, set(), {}, stringify_features)
        empty_df = spark.createDataFrame([], empty_schema)
        return empty_df if stringify_features else (empty_df, {})

    df = parse_json_columns(df, features_column=features_column, entities_column="entities")
    df = add_version_column(df, metadata_column=metadata_column)

    try:
        pairs = collect_distinct_pairs(df, mp_config_id_column=_MP_CONFIG_ID_COLUMN)
    except ValueError as e:
        raise e

    if not pairs:
        warnings.warn(
            "No distinct (mp_config_id, _schema_version) pairs found. Returning empty DataFrame.",
            UserWarning,
            stacklevel=2,
        )
        from pyspark.sql.types import StructType
        empty = spark.createDataFrame([], StructType([]))
        return empty if stringify_features else (empty, {})

    broadcast_plans, output_feature_names, type_map = build_and_broadcast_plans(
        spark, pairs, inference_host,
        needed_columns=needed_columns,
        stringify_features=stringify_features,
    )
    plans = broadcast_plans.value
    if not plans:
        warnings.warn(
            "No valid decode plans built (schema fetch failed for all config/version pairs). Returning empty DataFrame.",
            UserWarning,
            stacklevel=2,
        )
        metadata_cols = [c for c in ROW_METADATA_COLUMNS if c in df.columns]
        empty_schema = build_output_schema(df, metadata_cols, set(), {}, stringify_features)
        empty_df = spark.createDataFrame([], empty_schema)
        return empty_df if stringify_features else (empty_df, type_map)

    metadata_cols = [c for c in ROW_METADATA_COLUMNS if c in df.columns]
    output_schema = build_output_schema(
        df, metadata_cols, output_feature_names, plans, stringify_features
    )
    output_columns = ["entity_id"] + metadata_cols + sorted(output_feature_names)
    feature_columns_ordered = sorted(output_feature_names)

    df = prepare_partitions(df, num_partitions or _DEFAULT_NUM_PARTITIONS, mp_config_id_column=_MP_CONFIG_ID_COLUMN)

    udf_fn = make_partition_decoder(
        broadcast_plans,
        features_column=features_column,
        metadata_cols_in_df=metadata_cols,
        output_columns=output_columns,
        feature_columns_ordered=feature_columns_ordered,
        stringify_features=stringify_features,
        feature_type_lookup=type_map,
    )
    result_rdd = df.rdd.mapPartitions(udf_fn)
    result_df = spark.createDataFrame(result_rdd, output_schema)

    # Reorder: entity_id first, then metadata, then features sorted
    reorder = ["entity_id"] + [c for c in ROW_METADATA_COLUMNS if c in result_df.columns] + sorted(output_feature_names)
    result_df = result_df.select([c for c in reorder if c in result_df.columns])

    if stringify_features:
        return result_df
    return result_df, type_map


def decode_single_config(
    df: Any,
    spark: Any,
    mp_config_id: str,
    inference_host: Optional[str] = None,
    needed_columns: Optional[Set[str]] = None,
    num_partitions: Optional[int] = None,
    stringify_features: bool = True,
    features_column: str = "features",
    metadata_column: str = "metadata",
) -> Union[Any, Tuple[Any, dict]]:
    """
    Decode MPLog features for a single model config. Filters by mp_config_id then runs the pipeline.

    Proto-only; decompression is always on. Version is taken from _schema_version (JVM-side);
    features/entities are parsed on the JVM via from_json when possible.

    Args:
        df: Input Spark DataFrame with features, metadata, mp_config_id (and optionally entities).
        spark: SparkSession.
        mp_config_id: Filter to rows with this mp_config_id only.
        inference_host: Base URL for schema API; defaults to INFERENCE_HOST env or http://localhost:8082.
        needed_columns: If set, only these feature names are decoded (narrow output).
        num_partitions: Repartition by (mp_config_id, _schema_version) to this count; default 50000.
        stringify_features: If True, all feature values are strings; if False, typed and (result_df, type_map) returned.
        features_column: Column name for JSON array of entity features (default "features").
        metadata_column: Column name for metadata byte (default "metadata").

    Returns:
        If stringify_features=True: Spark DataFrame with entity_id, metadata cols, and decoded features.
        If stringify_features=False: (Spark DataFrame, type_map) where type_map is feature_name -> type string.

    Raises:
        ValueError: If more than 1000 distinct (mp_config_id, version) pairs (data quality guard).

    Example (use case 1: single config, all features):
        >>> from pyspark.sql import SparkSession
        >>> from inference_logging_client.api import decode_single_config
        >>> spark = SparkSession.builder.appName("decode").getOrCreate()
        >>> df = spark.read.parquet("logs.parquet")
        >>> decoded = decode_single_config(df, spark, mp_config_id="my-model")
        >>> decoded.show()
    """
    from pyspark.sql import functions as F
    filtered = df.filter(F.col(_MP_CONFIG_ID_COLUMN) == mp_config_id)
    return _decode_internal(
        filtered, spark, inference_host, needed_columns, num_partitions,
        stringify_features, features_column, metadata_column,
    )


def decode_multi_config(
    df: Any,
    spark: Any,
    inference_host: Optional[str] = None,
    needed_columns: Optional[Set[str]] = None,
    num_partitions: Optional[int] = None,
    stringify_features: bool = True,
    features_column: str = "features",
    metadata_column: str = "metadata",
) -> Union[Any, Tuple[Any, dict]]:
    """
    Decode MPLog features for all model configs in the DataFrame. Runs the full pipeline.

    Proto-only; decompression always on. Version from _schema_version; features/entities
    parsed on JVM when possible. Distinct (mp_config_id, version) pairs are collected on
    the driver and schemas fetched once; plans are broadcast to executors.

    Args:
        df: Input Spark DataFrame with features, metadata, mp_config_id (and optionally entities).
        spark: SparkSession.
        inference_host: Base URL for schema API; defaults to INFERENCE_HOST env or http://localhost:8082.
        needed_columns: If set, only these feature names are decoded (narrow output).
        num_partitions: Repartition by (mp_config_id, _schema_version); default 50000.
        stringify_features: If True, all feature values are strings; if False, (result_df, type_map) returned.
        features_column: Column name for JSON features (default "features").
        metadata_column: Column name for metadata (default "metadata").

    Returns:
        If stringify_features=True: Spark DataFrame (entity_id, metadata cols, decoded features).
        If stringify_features=False: (Spark DataFrame, type_map).

    Raises:
        ValueError: If more than 1000 distinct (mp_config_id, version) pairs.

    Example (use case 2: multi config, all features):
        >>> from pyspark.sql import SparkSession
        >>> from inference_logging_client.api import decode_multi_config
        >>> spark = SparkSession.builder.appName("decode").getOrCreate()
        >>> df = spark.read.parquet("logs.parquet")
        >>> decoded = decode_multi_config(df, spark)
        >>> decoded.show()

    Example (use case 3: selective columns with typed output):
        >>> decoded, type_map = decode_multi_config(
        ...     df, spark,
        ...     needed_columns={"score", "age", "category"},
        ...     stringify_features=False,
        ... )
        >>> decoded.show()
        >>> # type_map e.g. {"score": "float", "age": "int32", "category": "string"}
    """
    return _decode_internal(
        df, spark, inference_host, needed_columns, num_partitions,
        stringify_features, features_column, metadata_column,
    )
