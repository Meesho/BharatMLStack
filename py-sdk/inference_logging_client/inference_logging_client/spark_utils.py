"""Spark-side utilities: version extraction (SQL expression), JSON parsing, repartitioning."""

from __future__ import annotations

from typing import TYPE_CHECKING, List, Optional, Tuple

if TYPE_CHECKING:
    from pyspark.sql import DataFrame as SparkDataFrame

# Max distinct (mp_config_id, version) pairs we allow before raising (data quality guard)
_MAX_DISTINCT_PAIRS = 1000


def add_version_column(
    df: "SparkDataFrame",
    metadata_column: str = "metadata",
) -> "SparkDataFrame":
    """Add _schema_version column using a pure Spark SQL expression (JVM-only, no Python UDF).

    metadata column is a plain base64 string.  The first decoded byte encodes:
      bits 2-5 → schema version (0-15)
    Expression: unbase64 → first byte → hex → decimal int → (>> 2) & 0x0F
    """
    from pyspark.sql import functions as F
    from pyspark.sql.types import IntegerType

    # unbase64 → first byte as binary → hex string (e.g. "04") → decimal int
    raw_byte = F.conv(
        F.hex(F.substring(F.unbase64(F.col(metadata_column)), 1, 1)),
        16,
        10,
    ).cast(IntegerType())
    version = F.shiftright(raw_byte, 2).bitwiseAND(F.lit(0x0F)).cast(IntegerType())
    return df.withColumn("_schema_version", version)


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
    """Repartition by (mp_config_id, _schema_version) for schema locality."""
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
    Rows where JVM parsing fails will have features_parsed=null; the Arrow UDF falls
    back to Python json.loads for those rows.
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

    return df
