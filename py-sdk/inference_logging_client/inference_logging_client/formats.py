"""Format-specific decoders for proto, arrow, and parquet formats."""

import io
import sys
import warnings
from typing import Any, Collection, Optional

import pyarrow as pa
import pyarrow.ipc as ipc
import pyarrow.parquet as pq

from .types import Format, FeatureInfo
from .io import parse_mplog_protobuf
from .decoder import ByteReader, decode_scalar_value, decode_vector_or_string, decode_feature_value
from .utils import is_sized_type, get_scalar_size
from .exceptions import FormatError


def decode_arrow_features(
    encoded_bytes: bytes,
    schema: list[FeatureInfo],
    needed_columns: Optional[Collection[str]] = None,
) -> dict[str, Any]:
    """
    Decode Arrow IPC-encoded features for a single entity.

    Arrow encoding for single entity:
    - Arrow IPC stream with a single row
    - Columns named by feature index ("0", "1", ...)
    - Each column contains raw feature value bytes

    When needed_columns is set, only those features are decoded and returned.
    """
    needed_set = set(needed_columns) if needed_columns is not None else None
    if len(encoded_bytes) == 0:
        if needed_set is not None:
            return {f.name: None for f in schema if f.name in needed_set}
        return {f.name: None for f in schema}

    try:
        reader = pa.ipc.open_stream(io.BytesIO(encoded_bytes))
        table = reader.read_all()
    except Exception as e:
        raise FormatError(f"Failed to read Arrow IPC data: {e}")

    if table.num_rows == 0:
        if needed_set is not None:
            return {f.name: None for f in schema if f.name in needed_set}
        return {f.name: None for f in schema}

    result = {}
    for feature in schema:
        if needed_set is not None and feature.name not in needed_set:
            continue
        col_name = str(feature.index)

        if col_name not in table.column_names:
            result[feature.name] = None
            continue

        column = table.column(col_name)

        if column.is_null()[0].as_py():
            result[feature.name] = None
            continue

        value_bytes = column[0].as_py()
        if value_bytes is None or len(value_bytes) == 0:
            result[feature.name] = None
        else:
            result[feature.name] = decode_feature_value(value_bytes, feature.feature_type)

    return result


def decode_parquet_features(
    encoded_bytes: bytes,
    schema: list[FeatureInfo],
    needed_columns: Optional[Collection[str]] = None,
) -> dict[str, Any]:
    """
    Decode Parquet-encoded features for a single entity.

    When needed_columns is set, only those features are decoded and returned.
    """
    needed_set = set(needed_columns) if needed_columns is not None else None
    if len(encoded_bytes) == 0:
        if needed_set is not None:
            return {f.name: None for f in schema if f.name in needed_set}
        return {f.name: None for f in schema}

    try:
        table = pq.read_table(io.BytesIO(encoded_bytes))
    except Exception as e:
        raise FormatError(f"Failed to read Parquet data: {e}")

    if table.num_rows == 0:
        if needed_set is not None:
            return {f.name: None for f in schema if f.name in needed_set}
        return {f.name: None for f in schema}

    result = {}

    def include(feature: FeatureInfo) -> bool:
        return needed_set is None or feature.name in needed_set

    if "Features" in table.column_names:
        features_col = table.column("Features")
        feature_data = features_col[0].as_py()

        if feature_data is None:
            if needed_set is not None:
                return {f.name: None for f in schema if f.name in needed_set}
            return {f.name: None for f in schema}

        if isinstance(feature_data, dict):
            for feature in schema:
                if not include(feature):
                    continue
                value_bytes = feature_data.get(feature.index)
                if value_bytes is None or len(value_bytes) == 0:
                    result[feature.name] = None
                else:
                    result[feature.name] = decode_feature_value(value_bytes, feature.feature_type)
        elif isinstance(feature_data, list):
            if len(feature_data) > 0 and isinstance(feature_data[0], tuple):
                feature_map = {k: v for k, v in feature_data}
                for feature in schema:
                    if not include(feature):
                        continue
                    value_bytes = feature_map.get(feature.index)
                    if value_bytes is None or len(value_bytes) == 0:
                        result[feature.name] = None
                    else:
                        result[feature.name] = decode_feature_value(value_bytes, feature.feature_type)
            else:
                for feature in schema:
                    if not include(feature):
                        continue
                    if feature.index < len(feature_data):
                        value_bytes = feature_data[feature.index]
                        if value_bytes is None or (isinstance(value_bytes, (bytes, bytearray)) and len(value_bytes) == 0):
                            result[feature.name] = None
                        else:
                            result[feature.name] = decode_feature_value(value_bytes, feature.feature_type)
                    else:
                        result[feature.name] = None
        else:
            if needed_set is not None:
                return {f.name: None for f in schema if f.name in needed_set}
            return {f.name: None for f in schema}
    else:
        for feature in schema:
            if not include(feature):
                continue
            col_name = str(feature.index)
            if col_name in table.column_names:
                column = table.column(col_name)
                if column.is_null()[0].as_py():
                    result[feature.name] = None
                else:
                    value_bytes = column[0].as_py()
                    if value_bytes is None or len(value_bytes) == 0:
                        result[feature.name] = None
                    else:
                        result[feature.name] = decode_feature_value(value_bytes, feature.feature_type)
            else:
                result[feature.name] = None

    return result


def decode_proto_features(
    encoded_bytes: bytes,
    schema: list[FeatureInfo],
    needed_columns: Optional[Collection[str]] = None,
) -> dict[str, Any]:
    """
    Decode proto-encoded features for a single entity.

    Proto encoding format:
    - First byte: generated flag (1 = no generated values, 0 = has generated values)
    - For each feature in schema order:
      - Scalars: fixed size bytes based on type
      - Strings/Vectors: 2-byte little-endian size prefix + data bytes

    When needed_columns is set, only those features are decoded and returned;
    bytes for other features are skipped to reduce memory and CPU.
    """
    needed_set = set(needed_columns) if needed_columns is not None else None
    if len(encoded_bytes) == 0:
        if needed_set is not None:
            return {f.name: None for f in schema if f.name in needed_set}
        return {f.name: None for f in schema}

    reader = ByteReader(encoded_bytes)
    result = {}

    # Read generated flag (first byte)
    if reader.remaining() < 1:
        if needed_set is not None:
            return {f.name: None for f in schema if f.name in needed_set}
        return {f.name: None for f in schema}

    _generated_flag = reader.read_uint8()  # Prefixed with _ to indicate intentionally unused

    for feature in schema:
        if not reader.has_more():
            if needed_set is None or feature.name in needed_set:
                result[feature.name] = None
            continue

        skip = needed_set is not None and feature.name not in needed_set
        if skip:
            try:
                if is_sized_type(feature.feature_type):
                    if reader.remaining() < 2:
                        break
                    size = reader.read_uint16()
                    if reader.remaining() < size:
                        break
                    reader.read(size)
                else:
                    size = get_scalar_size(feature.feature_type) or 4
                    if reader.remaining() < size:
                        break
                    reader.read(size)
            except Exception:
                break
            continue

        try:
            if is_sized_type(feature.feature_type):
                if reader.remaining() < 2:
                    result[feature.name] = None
                    continue

                size = reader.read_uint16()
                if size == 0:
                    result[feature.name] = None
                    continue

                if reader.remaining() < size:
                    result[feature.name] = None
                    continue

                value_bytes = reader.read(size)
                result[feature.name] = decode_vector_or_string(value_bytes, feature.feature_type)
            else:
                size = get_scalar_size(feature.feature_type)
                if size is None:
                    result[feature.name] = None
                    continue

                if reader.remaining() < size:
                    result[feature.name] = None
                    continue

                value_bytes = reader.read(size)
                result[feature.name] = decode_scalar_value(value_bytes, feature.feature_type)
        except Exception as e:
            result[feature.name] = f"<decode_error: {e}>"

    return result


def decode_proto_format(
    mplog_data: bytes,
    schema: list[FeatureInfo],
    needed_columns: Optional[Collection[str]] = None,
) -> tuple[list[str], list[dict[str, Any]]]:
    """
    Decode proto format MPLog.

    When needed_columns is set, only those features are decoded per entity.
    """
    parsed = parse_mplog_protobuf(mplog_data)

    entity_ids = list(parsed.entities)
    encoded_features_list = getattr(parsed, "_encoded_features", [])

    decoded_rows = []
    for encoded_bytes in encoded_features_list:
        decoded = decode_proto_features(encoded_bytes, schema, needed_columns=needed_columns)
        decoded_rows.append(decoded)

    # Ensure entity_ids matches decoded_rows count
    original_entity_count = len(entity_ids)
    while len(entity_ids) < len(decoded_rows):
        entity_ids.append(f"entity_{len(entity_ids)}")

    if original_entity_count != len(decoded_rows):
        warnings.warn(
            f"Entity count mismatch: {original_entity_count} entity IDs for {len(decoded_rows)} rows. "
            f"Generated synthetic IDs for missing entities.",
            UserWarning,
        )

    return entity_ids[: len(decoded_rows)], decoded_rows


def decode_arrow_format(
    mplog_data: bytes,
    schema: list[FeatureInfo],
    needed_columns: Optional[Collection[str]] = None,
) -> tuple[list[str], list[dict[str, Any]]]:
    """
    Decode Arrow IPC format MPLog.

    When needed_columns is set, only those features are decoded per row.
    """
    parsed = parse_mplog_protobuf(mplog_data)

    entity_ids = list(parsed.entities)
    encoded_features_list = getattr(parsed, "_encoded_features", [])

    if not encoded_features_list:
        return [], []

    if len(encoded_features_list) > 1:
        warnings.warn(
            f"Arrow format contains {len(encoded_features_list)} encoded feature blobs, "
            f"but only the first will be processed. This may indicate unexpected data.",
            UserWarning,
        )

    arrow_bytes = encoded_features_list[0]

    if len(arrow_bytes) == 0:
        return entity_ids, []

    try:
        reader = pa.ipc.open_stream(io.BytesIO(arrow_bytes))
        table = reader.read_all()
    except Exception as e:
        raise FormatError(f"Failed to read Arrow IPC data: {e}")

    needed_set = set(needed_columns) if needed_columns is not None else None
    num_rows = table.num_rows
    decoded_rows = []

    for row_idx in range(num_rows):
        row_data = {}
        for feature in schema:
            if needed_set is not None and feature.name not in needed_set:
                continue
            col_name = str(feature.index)

            if col_name not in table.column_names:
                row_data[feature.name] = None
                continue

            column = table.column(col_name)

            if column.is_null()[row_idx].as_py():
                row_data[feature.name] = None
                continue

            value_bytes = column[row_idx].as_py()
            if value_bytes is None or len(value_bytes) == 0:
                row_data[feature.name] = None
            else:
                row_data[feature.name] = decode_feature_value(value_bytes, feature.feature_type)

        decoded_rows.append(row_data)

    # Ensure entity_ids matches decoded_rows count
    original_entity_count = len(entity_ids)
    while len(entity_ids) < len(decoded_rows):
        entity_ids.append(f"entity_{len(entity_ids)}")

    if original_entity_count != len(decoded_rows):
        warnings.warn(
            f"Entity count mismatch: {original_entity_count} entity IDs for {len(decoded_rows)} rows. "
            f"Generated synthetic IDs for missing entities.",
            UserWarning,
        )

    return entity_ids[: len(decoded_rows)], decoded_rows


def decode_parquet_format(
    mplog_data: bytes,
    schema: list[FeatureInfo],
    needed_columns: Optional[Collection[str]] = None,
) -> tuple[list[str], list[dict[str, Any]]]:
    """
    Decode Parquet format MPLog.

    When needed_columns is set, only those features are decoded per row.
    """
    parsed = parse_mplog_protobuf(mplog_data)

    entity_ids = list(parsed.entities)
    encoded_features_list = getattr(parsed, "_encoded_features", [])

    if not encoded_features_list:
        return [], []

    if len(encoded_features_list) > 1:
        warnings.warn(
            f"Parquet format contains {len(encoded_features_list)} encoded feature blobs, "
            f"but only the first will be processed. This may indicate unexpected data.",
            UserWarning,
        )

    parquet_bytes = encoded_features_list[0]

    if len(parquet_bytes) == 0:
        return entity_ids, []

    try:
        table = pq.read_table(io.BytesIO(parquet_bytes))
    except Exception as e:
        raise FormatError(f"Failed to read Parquet data: {e}")

    needed_set = set(needed_columns) if needed_columns is not None else None
    num_rows = table.num_rows
    decoded_rows = []

    def include(feature: FeatureInfo) -> bool:
        return needed_set is None or feature.name in needed_set

    if "Features" not in table.column_names:
        for row_idx in range(num_rows):
            row_data = {}
            for feature in schema:
                if not include(feature):
                    continue
                col_name = str(feature.index)
                if col_name in table.column_names:
                    column = table.column(col_name)
                    if column.is_null()[row_idx].as_py():
                        row_data[feature.name] = None
                    else:
                        value_bytes = column[row_idx].as_py()
                        if value_bytes is None or len(value_bytes) == 0:
                            row_data[feature.name] = None
                        else:
                            row_data[feature.name] = decode_feature_value(
                                value_bytes, feature.feature_type
                            )
                else:
                    row_data[feature.name] = None
            decoded_rows.append(row_data)
    else:
        features_col = table.column("Features")

        for row_idx in range(num_rows):
            row_data = {}
            feature_data = features_col[row_idx].as_py()

            if feature_data is None:
                for feature in schema:
                    if include(feature):
                        row_data[feature.name] = None
            elif isinstance(feature_data, dict):
                for feature in schema:
                    if not include(feature):
                        continue
                    value_bytes = feature_data.get(feature.index)
                    if value_bytes is None or len(value_bytes) == 0:
                        row_data[feature.name] = None
                    else:
                        row_data[feature.name] = decode_feature_value(
                            value_bytes, feature.feature_type
                        )
            elif isinstance(feature_data, list):
                if len(feature_data) > 0:
                    if isinstance(feature_data[0], tuple) and len(feature_data[0]) == 2:
                        feature_map = {k: v for k, v in feature_data}
                        for feature in schema:
                            if not include(feature):
                                continue
                            value_bytes = feature_map.get(feature.index)
                            if value_bytes is None or len(value_bytes) == 0:
                                row_data[feature.name] = None
                            else:
                                row_data[feature.name] = decode_feature_value(
                                    value_bytes, feature.feature_type
                                )
                    else:
                        for feature in schema:
                            if not include(feature):
                                continue
                            if feature.index < len(feature_data):
                                value_bytes = feature_data[feature.index]
                                if value_bytes is None or (
                                    isinstance(value_bytes, (bytes, bytearray))
                                    and len(value_bytes) == 0
                                ):
                                    row_data[feature.name] = None
                                else:
                                    row_data[feature.name] = decode_feature_value(
                                        value_bytes, feature.feature_type
                                    )
                            else:
                                row_data[feature.name] = None
                else:
                    for feature in schema:
                        if include(feature):
                            row_data[feature.name] = None
            else:
                for feature in schema:
                    if include(feature):
                        row_data[feature.name] = None

            decoded_rows.append(row_data)

    # Ensure entity_ids matches decoded_rows count
    original_entity_count = len(entity_ids)
    while len(entity_ids) < len(decoded_rows):
        entity_ids.append(f"entity_{len(entity_ids)}")

    if original_entity_count != len(decoded_rows):
        warnings.warn(
            f"Entity count mismatch: {original_entity_count} entity IDs for {len(decoded_rows)} rows. "
            f"Generated synthetic IDs for missing entities.",
            UserWarning,
        )

    return entity_ids[: len(decoded_rows)], decoded_rows
