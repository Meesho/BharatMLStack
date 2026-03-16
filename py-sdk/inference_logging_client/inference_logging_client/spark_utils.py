"""Spark-side utilities: version extraction UDF, JSON parsing via from_json, repartitioning."""

from __future__ import annotations

import base64
import warnings
from typing import TYPE_CHECKING, List, Optional, Tuple

if TYPE_CHECKING:
    from pyspark.sql import DataFrame as SparkDataFrame

from .utils import unpack_metadata_byte

# Max distinct (mp_config_id, version) pairs we allow before raising (data quality guard)
_MAX_DISTINCT_PAIRS = 1000


def _metadata_to_version_impl(metadata_value: object) -> Optional[int]:
    """Extract schema version from metadata column value. Used inside Spark UDF.

    - If string: base64-decode then take first byte.
    - If bytes/bytearray: take first byte.
    - Unpack version from byte (bits 2-5) and return as int.
    - Returns None for null, empty, or decode failure.
    """
    if metadata_value is None:
        return None
    try:
        if isinstance(metadata_value, str):
            decoded = base64.b64decode(metadata_value)
        elif isinstance(metadata_value, (bytes, bytearray)):
            decoded = bytes(metadata_value)
        else:
            return None
        if len(decoded) < 1:
            return None
        _, version, _ = unpack_metadata_byte(decoded[0])
        return version
    except Exception:
        return None


def add_version_column(
    df: "SparkDataFrame",
    metadata_column: str = "metadata",
) -> "SparkDataFrame":
    """Add _schema_version column using a Python UDF (lightweight: single-byte decode per row).

    UDF: metadata value (string or bytes) -> base64 decode if string -> unpack_metadata_byte -> version (int).
    Runs on driver/executors; intended to run once before the heavy decode step.
    """
    from pyspark.sql import functions as F
    from pyspark.sql.types import IntegerType

    udf = F.udf(_metadata_to_version_impl, IntegerType())
    return df.withColumn("_schema_version", udf(F.col(metadata_column)))


def collect_distinct_pairs(
    df: "SparkDataFrame",
    mp_config_id_column: str = "mp_config_id",
) -> List[Tuple[str, int]]:
    """Collect distinct (mp_config_id, _schema_version) on the driver.

    df must already have _schema_version (e.g. from add_version_column).
    Returns list of (config_id_str, version_int). Drops rows where either is null.
    Raises ValueError if more than 1000 distinct pairs (data quality guard).
    """
    if "_schema_version" not in df.columns:
        raise ValueError(
            "DataFrame must have _schema_version column; call add_version_column first"
        )
    distinct_df = df.select(mp_config_id_column, "_schema_version").distinct()
    rows = distinct_df.collect()
    pairs: List[Tuple[str, int]] = []
    for row in rows:
        config_id = row[mp_config_id_column]
        version = row["_schema_version"]
        if config_id is not None and version is not None:
            pairs.append((str(config_id), int(version)))
    if len(pairs) > _MAX_DISTINCT_PAIRS:
        raise ValueError(
            f"Too many distinct (mp_config_id, schema_version) pairs: {len(pairs)}. "
            f"Maximum allowed is {_MAX_DISTINCT_PAIRS}. "
            "This may indicate inconsistent or corrupted metadata; check data quality."
        )
    return pairs


def prepare_partitions(
    df: "SparkDataFrame",
    num_partitions: int,
    mp_config_id_column: str = "mp_config_id",
) -> "SparkDataFrame":
    """Repartition by (mp_config_id, _schema_version) for schema locality.

    df must already have _schema_version. Returns repartitioned DataFrame.
    """
    from pyspark.sql import functions as F

    if "_schema_version" not in df.columns:
        raise ValueError(
            "DataFrame must have _schema_version column; call add_version_column first"
        )
    return df.repartition(
        num_partitions,
        F.col(mp_config_id_column),
        F.col("_schema_version"),
    )


def parse_json_columns(
    df: "SparkDataFrame",
    features_column: str = "features",
    entities_column: str = "entities",
) -> "SparkDataFrame":
    """Parse features and entities JSON columns on the JVM via from_json (vectorized).

    Adds features_parsed (array of map) and entities_parsed (array of string).
    If JVM parsing fails for some rows (non-null input -> null parsed), warns so the
    UDF can fall back to Python json.loads for those rows; UDF should use
    features_parsed if not null, else parse the raw features string in Python.
    """
    from pyspark.sql import functions as F
    from pyspark.sql.types import ArrayType, MapType, StringType

    features_schema = ArrayType(MapType(StringType(), StringType()))
    entities_schema = ArrayType(StringType())

    df = df.withColumn(
        "features_parsed",
        F.from_json(F.col(features_column), features_schema),
    )
    if entities_column in df.columns:
        df = df.withColumn(
            "entities_parsed",
            F.from_json(F.col(entities_column), entities_schema),
        )
    else:
        df = df.withColumn("entities_parsed", F.lit(None).cast(entities_schema))

    # Validation: any row where original is non-null but parsed is null (JVM parse failure)
    features_failures = df.filter(
        F.col(features_column).isNotNull() & F.col("features_parsed").isNull()
    ).limit(1).count()
    if features_failures > 0:
        warnings.warn(
            "Some rows had non-null features but JVM from_json returned null (parse failure). "
            "UDF should fall back to Python json.loads for those rows (use features_parsed if not null, else parse raw features).",
            UserWarning,
            stacklevel=2,
        )
    if entities_column in df.columns:
        entities_failures = df.filter(
            F.col(entities_column).isNotNull() & F.col("entities_parsed").isNull()
        ).limit(1).count()
        if entities_failures > 0:
            warnings.warn(
                "Some rows had non-null entities but JVM from_json returned null (parse failure). "
                "UDF should fall back to Python json.loads for those rows (use entities_parsed if not null, else parse raw entities).",
                UserWarning,
                stacklevel=2,
            )

    return df
