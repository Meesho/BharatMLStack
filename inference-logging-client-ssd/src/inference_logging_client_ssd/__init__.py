"""inference-logging-client-ssd

Parse asyncloguploader ``.log`` files (written to SSD / GCS) into pandas
DataFrames, using ``inference-logging-client`` for protobuf decoding and
feature schema resolution.

Basic usage::

    from inference_logging_client_ssd import parse_log_file

    df = parse_log_file("/path/to/file.log")

With explicit Horizon-v2 host::

    df = parse_log_file(
        "/path/to/file.log",
        inference_host="http://horizon-v2.prd.meesho.int",
    )

Decode only a subset of features::

    df = parse_log_file("/path/to/file.log", needed_columns=["feature_a", "feature_b"])

Environment variables (same as ``inference-logging-client``):

* ``INFERENCE_HOST``  — Horizon-v2 API base URL
* ``INFERENCE_PATH``  — schema API path
"""

from __future__ import annotations

from collections.abc import Collection
from pathlib import Path
from typing import Optional

import pandas as pd

from ._deframer import deframe
from ._parser import build_dataframe

# Re-export ilc exceptions so callers only need one import.
from inference_logging_client.exceptions import (
    DecodeError,
    InferenceLoggingError,
    SchemaFetchError,
    SchemaNotFoundError,
)

__version__ = "0.1.0"

__all__ = [
    "parse_log_file",
    # exceptions
    "InferenceLoggingError",
    "SchemaFetchError",
    "SchemaNotFoundError",
    "DecodeError",
]


def parse_log_file(
    path: str | Path,
    *,
    inference_host: Optional[str] = None,
    api_path: Optional[str] = None,
    needed_columns: Optional[Collection[str]] = None,
) -> pd.DataFrame:
    """Parse an asyncloguploader ``.log`` file into a pandas DataFrame.

    Each row represents one entity within one inference request.

    Args:
        path: Path to the ``.log`` file on local disk.
        inference_host: Horizon-v2 API host used to fetch feature schemas,
            e.g. ``"http://horizon-v2.prd.meesho.int"``.  When ``None`` the
            value of the ``INFERENCE_HOST`` environment variable is used;
            if that is also unset it falls back to ``"http://localhost:8082"``.
        api_path: Schema API path.  When ``None`` the value of the
            ``INFERENCE_PATH`` environment variable is used; if that is also
            unset the ``inference-logging-client`` default is used.
        needed_columns: An optional collection of feature names to decode.
            When supplied only those columns appear in the returned DataFrame
            (in addition to the standard metadata columns).  Pass ``None``
            (the default) to decode every feature in the schema.

    Returns:
        ``pandas.DataFrame`` with the following columns (in order):

        * ``timestamp_ns``   — log timestamp in Unix nanoseconds (int)
        * ``timestamp``      — log timestamp as a UTC-aware ``datetime``
        * ``user_id``        — user identifier (str)
        * ``tracking_id``    — request tracking identifier (str)
        * ``model_config_id``— model-proxy config identifier (str)
        * ``version``        — feature schema version (int)
        * ``format_type``    — encoded format (0=proto, 1=arrow, 2=parquet)
        * ``entity_id``      — entity identifier, e.g. product ID (str)
        * ``parent_entity``  — parent entity identifier (str)
        * ``<feature_name>`` — one column per decoded feature (type varies)

        Returns an empty DataFrame with the metadata column headers when the
        file contains no parseable records.

    Raises:
        FileNotFoundError: If ``path`` does not exist.
    """
    path = Path(path)
    if not path.exists():
        raise FileNotFoundError(f"Log file not found: {path}")

    records = deframe(path)
    return build_dataframe(
        records,
        inference_host=inference_host,
        api_path=api_path,
        needed_columns=needed_columns,
    )
