"""
Writes asset output DataFrames to artifact store and optionally to serving tables.
"""
from __future__ import annotations

import logging
from typing import Any

logger = logging.getLogger(__name__)


class OutputWriter:
    """Writes computed asset outputs to the appropriate storage location."""

    def __init__(self, engine: str = "spark"):
        self._engine = engine

    def write_artifact(self, df: Any, artifact_path: str) -> None:
        """
        Write to artifact store. This ALWAYS happens for executed assets.
        Artifacts are immutable: path includes compute_key.
        """
        logger.info("Writing artifact to '%s'", artifact_path)

        if self._engine == "spark":
            df.write.format("delta").mode("overwrite").save(artifact_path)
        elif self._engine == "polars":
            import deltalake

            if hasattr(df, "collect"):
                df = df.collect()
            deltalake.write_deltalake(artifact_path, df.to_arrow(), mode="overwrite")
        else:
            raise ValueError(f"Unsupported engine: {self._engine}")

    def publish_to_serving(self, df: Any, serving_path: str) -> None:
        """
        Publish to serving table. Only for ACTIVE assets.
        Overwrites the partition in the serving table.
        """
        logger.info("Publishing to serving table at '%s'", serving_path)

        if self._engine == "spark":
            df.write.format("delta").mode("overwrite").save(serving_path)
        elif self._engine == "polars":
            import deltalake

            if hasattr(df, "collect"):
                df = df.collect()
            deltalake.write_deltalake(
                serving_path, df.to_arrow(), mode="overwrite"
            )
