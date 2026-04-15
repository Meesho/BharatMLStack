"""Decode (timestamp_ns, proto_bytes) pairs into a pandas DataFrame.

Orchestrates inference-logging-client calls:

  1. get_mplog_metadata(proto_bytes)
       → DecodedMPLog  (user_id, tracking_id, model_proxy_config_id,
                         entities, parent_entity, version, format_type,
                         _encoded_features: list[bytes])

  2. get_feature_schema(model_config_id, version, ...)
       → list[FeatureInfo]  (cached by ilc; one fetch per unique config+version)

  3. decode_proto_features(encoded_bytes, schema, needed_columns)
       → dict[feature_name, value]  (one call per entity per record)

Output: one DataFrame row per entity.
"""

from __future__ import annotations

import datetime
import logging
from collections.abc import Collection
from typing import Optional

import pandas as pd
from inference_logging_client import get_mplog_metadata, get_feature_schema
from inference_logging_client.formats import decode_proto_features
from inference_logging_client.exceptions import (
    SchemaFetchError,
    SchemaNotFoundError,
    DecodeError,
)
from inference_logging_client.types import FORMAT_TYPE_PROTO

logger = logging.getLogger(__name__)

# Columns that always appear first in the output DataFrame, in this order.
_META_COLUMNS = [
    "timestamp_ns",
    "timestamp",
    "user_id",
    "tracking_id",
    "model_config_id",
    "version",
    "format_type",
    "entity_id",
    "parent_entity",
]


def build_dataframe(
    records: list[tuple[int, bytes]],
    inference_host: Optional[str] = None,
    api_path: Optional[str] = None,
    needed_columns: Optional[Collection[str]] = None,
) -> pd.DataFrame:
    """Decode records into a pandas DataFrame.

    Args:
        records: ``(timestamp_ns, proto_bytes)`` pairs from the deframer.
        inference_host: Custodian API host passed to ``get_feature_schema``.
        api_path: API path passed to ``get_feature_schema``.
        needed_columns: Feature names to decode; ``None`` decodes everything.

    Returns:
        DataFrame with metadata columns followed by feature columns.
        Returns an empty DataFrame (with metadata column headers) when
        ``records`` is empty or all records fail to decode.
    """
    if not records:
        return pd.DataFrame(columns=_META_COLUMNS)

    rows: list[dict] = []

    for i, (timestamp_ns, proto_bytes) in enumerate(records):
        try:
            mplog = get_mplog_metadata(proto_bytes)
        except Exception as exc:
            logger.warning("Record %d: MPLog parse failed — %s", i, exc)
            continue

        encoded_features: list[bytes] = getattr(mplog, "_encoded_features", [])
        entities = mplog.entities
        parent_entities = mplog.parent_entity
        model_config_id = mplog.model_proxy_config_id
        version = mplog.version
        format_type = mplog.format_type

        base = {
            "timestamp_ns": timestamp_ns,
            "timestamp": datetime.datetime.fromtimestamp(
                timestamp_ns / 1e9, tz=datetime.timezone.utc
            ),
            "user_id": mplog.user_id,
            "tracking_id": mplog.tracking_id,
            "model_config_id": model_config_id,
            "version": version,
            "format_type": format_type,
        }

        if not encoded_features:
            # No feature data — emit one metadata-only row per entity (or one
            # row if the entity list is also empty).
            for idx in range(max(len(entities), 1)):
                rows.append({
                    **base,
                    "entity_id": entities[idx] if idx < len(entities) else "",
                    "parent_entity": parent_entities[idx] if idx < len(parent_entities) else "",
                })
            continue

        # Fetch schema — ilc caches by (model_config_id, version) so this is
        # effectively free after the first call for a given config+version.
        schema = None
        if format_type == FORMAT_TYPE_PROTO:
            try:
                schema = get_feature_schema(
                    model_config_id,
                    version,
                    inference_host=inference_host,
                    api_path=api_path,
                )
            except (SchemaFetchError, SchemaNotFoundError) as exc:
                logger.warning(
                    "Record %d: schema unavailable for %s v%d — features will be omitted (%s)",
                    i, model_config_id, version, exc,
                )
        else:
            logger.warning(
                "Record %d: format_type=%d is not proto — feature decoding skipped",
                i, format_type,
            )

        for idx, enc_bytes in enumerate(encoded_features):
            entity_id = entities[idx] if idx < len(entities) else ""
            parent_entity = parent_entities[idx] if idx < len(parent_entities) else ""

            features: dict = {}
            if schema and enc_bytes:
                try:
                    features = decode_proto_features(enc_bytes, schema, needed_columns)
                except (DecodeError, Exception) as exc:
                    logger.warning(
                        "Record %d entity %d (%s): feature decode failed — %s",
                        i, idx, entity_id, exc,
                    )

            rows.append({
                **base,
                "entity_id": entity_id,
                "parent_entity": parent_entity,
                **features,
            })

        if (i + 1) % 1000 == 0:
            logger.info("Decoded %d / %d records (%d rows so far)", i + 1, len(records), len(rows))

    if not rows:
        return pd.DataFrame(columns=_META_COLUMNS)

    df = pd.DataFrame(rows)

    # Guarantee stable column order: metadata columns first, then features.
    present_meta = [c for c in _META_COLUMNS if c in df.columns]
    feature_cols = [c for c in df.columns if c not in _META_COLUMNS]
    return df[present_meta + sorted(feature_cols)]
