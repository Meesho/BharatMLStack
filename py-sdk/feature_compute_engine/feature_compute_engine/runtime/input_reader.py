"""
Reads input DataFrames for asset execution.

Handles:
- Reading from artifact store (prior asset outputs)
- Reading from external Delta tables (raw data sources)
- Reading at specific delta versions (for reproducibility)
"""
from __future__ import annotations

import logging
from typing import Any, Dict, Optional

logger = logging.getLogger(__name__)


class InputReader:
    """
    Reads input DataFrames based on execution plan bindings.

    Supports two engines:
    - Spark (reads Delta tables via SparkSession)
    - Polars (reads Delta tables via deltalake/polars)

    Engine is auto-detected from available imports at runtime.
    """

    def __init__(self, engine: str = "spark"):
        self._engine = engine
        self._cache: Dict[str, Any] = {}

    def read(
        self,
        input_name: str,
        path_or_version: str,
        partition: Optional[str] = None,
    ) -> Any:
        """
        Read a single input DataFrame.

        For internal assets: reads from artifact store path.
        For external sources: reads from Delta table at specific version.
        """
        if input_name in self._cache:
            logger.info("Cache hit for input '%s' (injected)", input_name)
            return self._cache[input_name]

        cache_key = f"{input_name}:{path_or_version}:{partition}"
        if cache_key in self._cache:
            logger.info("Cache hit for input '%s'", input_name)
            return self._cache[cache_key]

        logger.info("Reading input '%s' from '%s'", input_name, path_or_version)

        if self._engine == "spark":
            df = self._read_spark(path_or_version, partition)
        elif self._engine == "polars":
            df = self._read_polars(path_or_version, partition)
        else:
            raise ValueError(f"Unsupported engine: {self._engine}")

        self._cache[cache_key] = df
        return df

    def read_shared(
        self,
        shared_inputs: Dict[str, str],
    ) -> Dict[str, Any]:
        """
        Read all shared inputs for a notebook invocation.
        These are read ONCE and shared across all assets in the invocation.
        """
        result = {}
        for input_name, path_or_version in shared_inputs.items():
            result[input_name] = self.read(input_name, path_or_version)
        return result

    def inject_output(self, asset_name: str, df: Any) -> None:
        """
        Inject a computed asset's output into the cache.
        Allows downstream assets in the same notebook invocation
        to read it directly (in-memory) instead of from artifact store.
        """
        self._cache[asset_name] = df

    def _read_spark(self, path_or_version: str, partition: Optional[str]) -> Any:
        """Read via Spark. Expects active SparkSession."""
        from pyspark.sql import SparkSession

        spark = SparkSession.getActiveSession()
        if spark is None:
            raise RuntimeError("No active SparkSession found")

        if (
            path_or_version.startswith("/")
            or path_or_version.startswith("s3://")
            or path_or_version.startswith("gs://")
        ):
            df = spark.read.format("delta").load(path_or_version)
        elif "@v" in path_or_version:
            table, version = path_or_version.rsplit("@v", 1)
            df = (
                spark.read.format("delta")
                .option("versionAsOf", int(version))
                .table(table)
            )
        else:
            df = spark.read.format("delta").table(path_or_version)

        if partition:
            from pyspark.sql import functions as F

            df = df.filter(F.col("ds") == partition)

        return df

    def _read_polars(self, path_or_version: str, partition: Optional[str]) -> Any:
        """Read via Polars + deltalake."""
        import polars as pl

        df = pl.read_delta(path_or_version)
        if partition:
            df = df.filter(pl.col("ds") == partition)
        return df
