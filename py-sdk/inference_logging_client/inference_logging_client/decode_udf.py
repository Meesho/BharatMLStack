"""Partition-level decode UDF for mapPartitions: proto-only, broadcast plans, JVM-parsed JSON."""

from __future__ import annotations

import base64
import json
from typing import Any, Callable, Dict, List, Optional, Set, Tuple

from .proto_decoder import decode_proto_fixed, decode_proto_selective

_ZSTD_MAGIC = b"\x28\xb5\x2f\xfd"


def _convert_value(v: Any, stringify_features: bool) -> Any:
    """Convert decoded feature value for output row. No per-entity dict."""
    if v is None:
        return None
    if stringify_features:
        return str(v)
    if isinstance(v, (list, tuple)):
        return str(v)
    if isinstance(v, bytes):
        return v.hex()
    return v


def make_partition_decoder(
    broadcast_plans: Any,
    features_column: str = "features",
    metadata_cols_in_df: Optional[List[str]] = None,
    output_columns: Optional[List[str]] = None,
    feature_columns_ordered: Optional[List[str]] = None,
    stringify_features: bool = True,
    feature_type_lookup: Optional[Dict[str, str]] = None,
) -> Callable:
    """Returns a function for df.rdd.mapPartitions(fn). Proto-only, no metadata parsing in UDF."""

    metadata_cols_in_df = metadata_cols_in_df or []
    output_columns = output_columns or []
    feature_columns_ordered = feature_columns_ordered or []
    # feature_type_lookup accepted for API; typed pass-through does not need it in hot path

    def _decode_partition(partition):
        from pyspark.sql import Row

        plans = broadcast_plans.value
        cached_key: Optional[Tuple[str, int]] = None
        cached_plan = None

        try:
            import zstandard as zstd
            dctx = zstd.ZstdDecompressor()
        except ImportError:
            dctx = None

        OutputRow = Row(*output_columns)
        metadata_set = set(metadata_cols_in_df)

        for row in partition:
            mp_config_id = row["mp_config_id"]
            version = row["_schema_version"]
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

            # Use features_parsed (JVM from_json) if present; else fall back to Python json.loads
            features_parsed = row["features_parsed"]
            if features_parsed is None:
                raw = row[features_column]
                if raw is None:
                    continue
                try:
                    if isinstance(raw, str):
                        features_parsed = json.loads(raw)
                    else:
                        features_parsed = json.loads(raw) if hasattr(raw, "decode") else list(raw)
                except (ValueError, TypeError):
                    continue
            if not isinstance(features_parsed, list):
                continue

            # entities_parsed: native list[str] from JVM from_json; fallback to Python parse
            entities_parsed = row["entities_parsed"]
            if entities_parsed is None:
                try:
                    raw_ent = row["entities"]
                except (KeyError, TypeError):
                    raw_ent = None
                if raw_ent is not None:
                    try:
                        entities_parsed = json.loads(raw_ent) if isinstance(raw_ent, str) else (raw_ent if isinstance(raw_ent, list) else [])
                    except (ValueError, TypeError):
                        entities_parsed = []
                else:
                    entities_parsed = []
            if entities_parsed is None:
                entities_parsed = []

            # Metadata values once per input row (reuse for all entities)
            for i, feature_item in enumerate(features_parsed):
                if not isinstance(feature_item, dict):
                    continue
                encoded_b64 = feature_item.get("encoded_features", "")
                if not encoded_b64:
                    continue
                try:
                    encoded_bytes = base64.b64decode(encoded_b64)
                except (ValueError, TypeError):
                    continue
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

                # Build values list in output_columns order; no dict
                values = []
                for col in output_columns:
                    if col == "entity_id":
                        values.append(entity_id)
                    elif col in metadata_set:
                        values.append(row[col])
                    else:
                        v = decoded.get(col)
                        values.append(_convert_value(v, stringify_features))
                yield OutputRow(*values)

    return _decode_partition
