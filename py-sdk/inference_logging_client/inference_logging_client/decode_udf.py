"""Arrow-based partition decoder: proto-only, broadcast plans, JVM-parsed JSON.

Uses df.mapInArrow for zero-copy columnar I/O (no Python Row serialization overhead).
Input RecordBatches are converted to per-column Python lists once per batch;
output is emitted as a RecordBatch per input batch.
"""

from __future__ import annotations

import base64
import json
from typing import Any, Callable, Dict, List, Optional, Tuple

from .proto_decoder import decode_proto_fixed, decode_proto_selective

_ZSTD_MAGIC = b"\x28\xb5\x2f\xfd"


def _convert_value(v: Any, stringify_features: bool) -> Any:
    """Convert decoded feature value for output row."""
    if v is None:
        return None
    if stringify_features:
        return str(v)
    if isinstance(v, (list, tuple)):
        return str(v)
    if isinstance(v, bytes):
        return v.hex()
    return v


def _build_arrow_schema(spark_schema: Any) -> "pyarrow.Schema":
    """Convert Spark StructType to a pyarrow schema for explicit RecordBatch construction."""
    import pyarrow as pa
    from pyspark.sql.types import (
        BooleanType,
        DateType,
        DoubleType,
        FloatType,
        IntegerType,
        LongType,
        StringType,
        TimestampType,
    )

    fields = []
    for f in spark_schema.fields:
        dt = f.dataType
        if isinstance(dt, LongType):
            at = pa.int64()
        elif isinstance(dt, IntegerType):
            at = pa.int32()
        elif isinstance(dt, FloatType):
            at = pa.float32()
        elif isinstance(dt, DoubleType):
            at = pa.float64()
        elif isinstance(dt, BooleanType):
            at = pa.bool_()
        elif isinstance(dt, TimestampType):
            at = pa.timestamp("us")
        elif isinstance(dt, DateType):
            at = pa.date32()
        else:
            at = pa.string()
        fields.append(pa.field(f.name, at, nullable=True))
    return pa.schema(fields)


def make_arrow_decoder(
    broadcast_plans: Any,
    output_schema: Any,
    features_column: str = "features",
    metadata_cols_in_df: Optional[List[str]] = None,
    output_columns: Optional[List[str]] = None,
    feature_columns_ordered: Optional[List[str]] = None,
    stringify_features: bool = True,
    feature_type_lookup: Optional[Dict[str, str]] = None,
) -> Callable:
    """Returns a function for df.mapInArrow(fn, output_schema).

    Differences from the old mapPartitions approach:
    - Input arrives as Arrow RecordBatch: no per-row Python Row deserialisation.
    - All columns converted to Python lists once per batch (O(batch) vs O(rows) dict lookups).
    - base64.b64decode(..., validate=False): skips alphabet check (data is trusted proto).
    - Output emitted as RecordBatch: no per-row Python Row serialisation.
    """
    metadata_cols_in_df = metadata_cols_in_df or []
    output_columns = output_columns or []
    metadata_col_set = set(metadata_cols_in_df)

    def _decode_arrow_batches(batches):
        import pyarrow as pa

        plans = broadcast_plans.value
        arrow_schema = _build_arrow_schema(output_schema)

        cached_key: Optional[Tuple[str, int]] = None
        cached_plan = None

        try:
            import zstandard as zstd
            dctx = zstd.ZstdDecompressor()
        except ImportError:
            dctx = None

        for batch in batches:
            n = batch.num_rows
            if n == 0:
                continue

            # Convert all columns to Python lists once per batch.
            # .to_pylist() is O(n) but avoids repeated .as_py() calls inside the row loop.
            batch_data: Dict[str, list] = {
                name: batch.column(i).to_pylist()
                for i, name in enumerate(batch.schema.names)
            }

            mp_config_ids = batch_data.get("mp_config_id", [None] * n)
            versions = batch_data.get("_schema_version", [None] * n)
            features_parsed_col = batch_data.get("features_parsed", [None] * n)
            features_raw_col = batch_data.get(features_column, [None] * n)
            entities_parsed_col = batch_data.get("entities_parsed", [None] * n)
            entities_raw_col = batch_data.get("entities", [None] * n)

            # Pre-extract metadata columns so we don't re-look up per entity
            row_meta_cols: Dict[str, list] = {
                col: batch_data.get(col, [None] * n) for col in metadata_cols_in_df
            }

            out_lists: Dict[str, list] = {col: [] for col in output_columns}

            for row_idx in range(n):
                mp_config_id = mp_config_ids[row_idx]
                version = versions[row_idx]
                if mp_config_id is None or version is None:
                    continue
                mp_config_id = str(mp_config_id)
                version = int(version)
                key = (mp_config_id, version)
                if cached_key != key:
                    cached_plan = plans.get(key)
                    cached_key = key
                if cached_plan is None:
                    continue

                kind, plan_payload, _schema = cached_plan

                # Use JVM-parsed features if available; fall back to Python json.loads
                features_parsed = features_parsed_col[row_idx]
                if features_parsed is None:
                    raw = features_raw_col[row_idx]
                    if raw is None:
                        continue
                    try:
                        features_parsed = json.loads(raw) if isinstance(raw, str) else list(raw)
                    except (ValueError, TypeError):
                        continue
                if not isinstance(features_parsed, list):
                    continue

                entities_parsed = entities_parsed_col[row_idx]
                if entities_parsed is None:
                    raw_ent = entities_raw_col[row_idx]
                    if raw_ent is not None:
                        try:
                            entities_parsed = (
                                json.loads(raw_ent)
                                if isinstance(raw_ent, str)
                                else (raw_ent if isinstance(raw_ent, list) else [])
                            )
                        except (ValueError, TypeError):
                            entities_parsed = []
                    else:
                        entities_parsed = []

                # Metadata values for this input row (reused for every entity in the row)
                row_meta = {col: row_meta_cols[col][row_idx] for col in metadata_cols_in_df}

                for i, feature_item in enumerate(features_parsed):
                    if not isinstance(feature_item, dict):
                        continue
                    encoded_b64 = feature_item.get("encoded_features", "")
                    if not encoded_b64:
                        continue
                    # validate=False skips base64 alphabet check; safe here (data is trusted)
                    encoded_bytes = base64.b64decode(encoded_b64, validate=False)
                    if len(encoded_bytes) == 0:
                        continue
                    if dctx and len(encoded_bytes) >= 4 and encoded_bytes[:4] == _ZSTD_MAGIC:
                        try:
                            working_data = dctx.decompress(encoded_bytes)
                        except Exception:
                            continue
                    else:
                        working_data = encoded_bytes

                    if kind == "fixed":
                        decoded = decode_proto_fixed(working_data, plan_payload)
                    else:
                        decoded = decode_proto_selective(working_data, plan_payload)

                    entity_id = (
                        str(entities_parsed[i])
                        if entities_parsed and i < len(entities_parsed)
                        else f"entity_{i}"
                    )

                    for col in output_columns:
                        if col == "entity_id":
                            out_lists[col].append(entity_id)
                        elif col in metadata_col_set:
                            out_lists[col].append(row_meta.get(col))
                        else:
                            v = decoded.get(col)
                            out_lists[col].append(_convert_value(v, stringify_features))

            if out_lists.get("entity_id"):
                arrays = [
                    pa.array(out_lists[col], type=arrow_schema.field(col).type)
                    for col in output_columns
                ]
                yield pa.RecordBatch.from_arrays(arrays, schema=arrow_schema)

    return _decode_arrow_batches
