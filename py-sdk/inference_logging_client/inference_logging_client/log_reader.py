"""Read and decode .log files produced by asyncloguploader.

Supports local file paths, ``gs://bucket/key`` GCS URIs, and open binary
file-like objects. Each .log file is a sequence of frames; each frame holds
length-prefixed records of ``[timestamp_ns][MPLog protobuf]``.

Frame layout (all little-endian):

    ┌─────────────────────────────────────────────────────────────┐
    │ Frame header (8 bytes)                                      │
    │   capacity (uint32)  + valid_data_bytes (uint32)            │
    ├─────────────────────────────────────────────────────────────┤
    │ Frame body (capacity - 8 bytes)                             │
    │   First valid_data_bytes contain records; rest is padding.  │
    │   Record: [4B length][8B timestamp_ns][MPLog payload]       │
    └─────────────────────────────────────────────────────────────┘
"""

from __future__ import annotations

import os
import warnings
from contextlib import contextmanager
from pathlib import Path
from typing import IO, TYPE_CHECKING, Any, BinaryIO, Collection, Iterator, List, Optional, Tuple, Union

from .exceptions import FormatError
from .formats import decode_arrow_features, decode_parquet_features, decode_proto_features
from .io import get_feature_schema, parse_mplog_protobuf
from .types import FORMAT_TYPE_MAP, FeatureInfo, Format

if TYPE_CHECKING:
    from pyspark.sql import DataFrame as SparkDataFrame
    from pyspark.sql import SparkSession


_HEADER_SIZE = 8
_LENGTH_PREFIX_SIZE = 4
_TIMESTAMP_SIZE = 8

LogSource = Union[str, Path, BinaryIO, IO[bytes]]


def _is_gcs_uri(source: Any) -> bool:
    return isinstance(source, str) and source.startswith("gs://")


def _parse_gcs_uri(uri: str) -> Tuple[str, str]:
    """Split ``gs://bucket/key`` into ``(bucket, key)``."""
    if not uri.startswith("gs://"):
        raise ValueError(f"Not a GCS URI: {uri!r}")
    rest = uri[len("gs://") :]
    if "/" not in rest:
        raise ValueError(f"GCS URI missing object key: {uri!r}")
    bucket, key = rest.split("/", 1)
    if not bucket or not key:
        raise ValueError(f"GCS URI must be gs://<bucket>/<key>: {uri!r}")
    return bucket, key


@contextmanager
def _open_source(source: LogSource):
    """Yield a binary, seekable-where-possible reader for ``source``.

    Local paths and GCS objects are opened here and closed on exit. File-like
    objects are yielded as-is (caller retains ownership).
    """
    if _is_gcs_uri(source):
        try:
            from google.cloud import storage  # type: ignore
        except ImportError as exc:
            raise ImportError(
                "Reading from gs:// requires the 'google-cloud-storage' package. "
                "Install it with: pip install google-cloud-storage"
            ) from exc

        bucket_name, blob_name = _parse_gcs_uri(source)  # type: ignore[arg-type]
        client = storage.Client()
        blob = client.bucket(bucket_name).blob(blob_name)
        handle = blob.open("rb")
        try:
            yield handle
        finally:
            handle.close()
        return

    if isinstance(source, (str, Path)):
        handle = open(source, "rb")
        try:
            yield handle
        finally:
            handle.close()
        return

    # File-like
    if hasattr(source, "read"):
        yield source
        return

    raise TypeError(
        f"Unsupported source type: {type(source).__name__}. "
        "Expected a path, gs:// URI, or binary file-like object."
    )


def _read_exact(stream: BinaryIO, n: int) -> bytes:
    """Read exactly ``n`` bytes or return whatever was available (possibly empty)."""
    chunks: List[bytes] = []
    remaining = n
    while remaining > 0:
        chunk = stream.read(remaining)
        if not chunk:
            break
        chunks.append(chunk)
        remaining -= len(chunk)
    return b"".join(chunks)


def _iter_records_from_block(block: bytes, valid_data_bytes: int) -> Iterator[Tuple[int, bytes]]:
    """Yield (timestamp_ns, mplog_payload) records from a single frame body."""
    offset = 0
    limit = min(valid_data_bytes, len(block))
    while offset + _LENGTH_PREFIX_SIZE <= limit:
        record_length = int.from_bytes(block[offset : offset + _LENGTH_PREFIX_SIZE], "little")
        offset += _LENGTH_PREFIX_SIZE

        if record_length == 0:
            # Zero-length sentinel — keep scanning.
            continue
        if offset + record_length > limit:
            # Record claims to extend past valid data; stop this frame.
            break
        if record_length < _TIMESTAMP_SIZE:
            # Can't contain a timestamp; skip but keep moving.
            offset += record_length
            continue

        ts_ns = int.from_bytes(block[offset : offset + _TIMESTAMP_SIZE], "little")
        payload = block[offset + _TIMESTAMP_SIZE : offset + record_length]
        yield ts_ns, payload
        offset += record_length


def iter_log_records(
    source: LogSource,
    strict: bool = False,
) -> Iterator[Tuple[int, bytes]]:
    """Stream ``(timestamp_ns, mplog_payload_bytes)`` from a .log file.

    Args:
        source: Local path, ``gs://bucket/key`` URI, or open binary file-like.
        strict: If True, raise :class:`FormatError` on any malformed frame.
            If False (default), warn and skip the offending frame, advancing by
            ``capacity`` bytes when possible. Truncated trailing frames always
            stop iteration.

    Yields:
        Tuples of ``(timestamp_ns, mplog_payload_bytes)`` in file order. The
        payload is the raw MPLog protobuf — feed it to
        :func:`get_mplog_metadata` or the decode functions to extract features.
    """
    with _open_source(source) as stream:
        frame_index = 0
        while True:
            header = _read_exact(stream, _HEADER_SIZE)
            if len(header) == 0:
                return
            if len(header) < _HEADER_SIZE:
                if strict:
                    raise FormatError(
                        f"Truncated frame header at frame {frame_index} "
                        f"(read {len(header)} of {_HEADER_SIZE} bytes)"
                    )
                warnings.warn(
                    f"Truncated frame header at frame {frame_index}; stopping",
                    UserWarning,
                    stacklevel=2,
                )
                return

            capacity = int.from_bytes(header[0:4], "little")
            valid_data_bytes = int.from_bytes(header[4:8], "little")

            if capacity < _HEADER_SIZE:
                msg = (
                    f"Frame {frame_index}: capacity {capacity} is smaller than the "
                    f"{_HEADER_SIZE}-byte header"
                )
                if strict:
                    raise FormatError(msg)
                warnings.warn(msg + "; stopping", UserWarning, stacklevel=2)
                return

            data_size = capacity - _HEADER_SIZE

            if valid_data_bytes > capacity:
                msg = (
                    f"Frame {frame_index}: valid_data_bytes ({valid_data_bytes}) > "
                    f"capacity ({capacity})"
                )
                if strict:
                    raise FormatError(msg)
                warnings.warn(msg + "; skipping frame", UserWarning, stacklevel=2)
                # Best-effort: advance by data_size and continue.
                _read_exact(stream, data_size)
                frame_index += 1
                continue

            if valid_data_bytes == 0:
                # Empty frame — still consume the padding.
                if data_size > 0:
                    _read_exact(stream, data_size)
                frame_index += 1
                continue

            block = _read_exact(stream, data_size)
            if len(block) < data_size:
                # Truncated trailing frame.
                msg = (
                    f"Frame {frame_index}: expected {data_size} body bytes, "
                    f"got {len(block)} (EOF)"
                )
                if strict:
                    raise FormatError(msg)
                if len(block) < valid_data_bytes:
                    warnings.warn(msg + "; stopping", UserWarning, stacklevel=2)
                    return
                # We have enough for valid_data_bytes — fall through.

            yield from _iter_records_from_block(block, valid_data_bytes)
            frame_index += 1


def read_log_file(
    source: LogSource,
    strict: bool = False,
) -> List[Tuple[int, bytes]]:
    """Eagerly materialize all records from a .log file.

    Convenience wrapper around :func:`iter_log_records`. For large files
    prefer the iterator form to avoid loading every record into memory.
    """
    return list(iter_log_records(source, strict=strict))


# Columns we attach per record (in addition to entity_id and feature cols).
_RECORD_METADATA_COLUMNS = (
    "timestamp_ns",
    "mp_config_id",
    "version",
    "format_type",
    "user_id",
    "tracking_id",
    "parent_entity",
)


def _decode_one_record(
    payload: bytes,
    timestamp_ns: int,
    inference_host: Optional[str],
    decompress: bool,
    schema_override: Optional[List[FeatureInfo]],
    needed_columns: Optional[Collection[str]],
    schema_cache: dict,
    stringify_values: bool,
) -> Iterator[dict]:
    """Decode a single MPLog payload into per-entity dict rows."""
    # Optional zstd decompression of the outer payload.
    working = payload
    if decompress and len(working) >= 4 and working[:4] == b"\x28\xb5\x2f\xfd":
        try:
            import zstandard as zstd  # type: ignore
        except ImportError as exc:
            raise ImportError(
                "MPLog payload is zstd-compressed but the 'zstandard' package is not "
                "installed. Install it with: pip install zstandard"
            ) from exc
        working = zstd.ZstdDecompressor().decompress(working)

    parsed = parse_mplog_protobuf(working)
    encoded_per_entity: List[bytes] = list(getattr(parsed, "_encoded_features", []) or [])
    if not encoded_per_entity:
        return

    mp_config_id = parsed.model_proxy_config_id or ""
    version = int(parsed.version or 0)

    # Resolve schema: explicit override > cache > fetch.
    if schema_override is not None:
        schema = schema_override
    else:
        cache_key = (mp_config_id, version)
        schema = schema_cache.get(cache_key)
        if schema is None:
            if not mp_config_id:
                return
            try:
                schema = get_feature_schema(mp_config_id, version, inference_host)
            except Exception as exc:
                warnings.warn(
                    f"Failed to fetch schema for {cache_key}: {exc}",
                    UserWarning,
                    stacklevel=2,
                )
                return
            schema_cache[cache_key] = schema

    fmt = FORMAT_TYPE_MAP.get(parsed.format_type, Format.PROTO)
    if fmt == Format.ARROW:
        decoder = decode_arrow_features
    elif fmt == Format.PARQUET:
        decoder = decode_parquet_features
    else:
        decoder = decode_proto_features

    entities = list(parsed.entities or [])
    parents = list(parsed.parent_entity or [])

    base_row = {
        "timestamp_ns": timestamp_ns,
        "mp_config_id": mp_config_id,
        "version": version,
        "format_type": int(parsed.format_type or 0),
        "user_id": parsed.user_id or "",
        "tracking_id": parsed.tracking_id or "",
    }

    for i, enc in enumerate(encoded_per_entity):
        entity_id = str(entities[i]) if i < len(entities) else f"entity_{i}"
        try:
            decoded = decoder(enc, schema, needed_columns=needed_columns)
        except Exception as exc:
            warnings.warn(
                f"Decode failed for {mp_config_id}@v{version} entity#{i}: {exc}",
                UserWarning,
                stacklevel=2,
            )
            continue

        row = dict(base_row)
        row["entity_id"] = entity_id
        row["parent_entity"] = str(parents[i]) if i < len(parents) else ""
        for name, value in decoded.items():
            if not stringify_values:
                row[name] = value
                continue
            # Spark mode: stringify complex values so the inferred schema is stable.
            if value is None:
                row[name] = None
            elif isinstance(value, (list, tuple)):
                row[name] = str(list(value))
            elif isinstance(value, bytes):
                row[name] = value.hex()
            else:
                row[name] = value
        yield row


def iter_decoded_log_rows(
    source: LogSource,
    inference_host: Optional[str] = None,
    decompress: bool = True,
    schema: Optional[List[FeatureInfo]] = None,
    needed_columns: Optional[Collection[str]] = None,
    strict: bool = False,
    stringify_values: bool = False,
) -> Iterator[dict]:
    """Stream fully-decoded per-entity rows from a .log file.

    This is the single shared decode path used by every output sink
    (Spark, pandas, CSV, JSONL, text). Use it directly when you want to
    pipe rows somewhere custom without materializing the whole file.

    Args:
        source: Local path, ``gs://bucket/key`` URI, or open binary file-like.
        inference_host: Inference service URL. Defaults to ``INFERENCE_HOST`` env.
        decompress: Attempt zstd decompression on each MPLog payload.
        schema: Pre-fetched schema; bypasses per-record schema fetch.
        needed_columns: Optional subset of feature names to keep.
        strict: Propagated to :func:`iter_log_records`.
        stringify_values: If True, coerce list/tuple/bytes feature values to
            strings (used by the Spark sink to keep its inferred schema stable).

    Yields:
        Dict rows with keys ``entity_id, timestamp_ns, mp_config_id, version,
        format_type, user_id, tracking_id, parent_entity`` and one entry per
        decoded feature.
    """
    if inference_host is None:
        inference_host = os.getenv("INFERENCE_HOST", "http://localhost:8082")

    schema_cache: dict = {}
    for ts_ns, payload in iter_log_records(source, strict=strict):
        yield from _decode_one_record(
            payload,
            ts_ns,
            inference_host=inference_host,
            decompress=decompress,
            schema_override=schema,
            needed_columns=needed_columns,
            schema_cache=schema_cache,
            stringify_values=stringify_values,
        )


def decode_log_file(
    source: LogSource,
    spark: "SparkSession",
    inference_host: Optional[str] = None,
    decompress: bool = True,
    schema: Optional[List[FeatureInfo]] = None,
    needed_columns: Optional[Collection[str]] = None,
    strict: bool = False,
) -> "SparkDataFrame":
    """Decode an asyncloguploader .log file into a Spark DataFrame.

    Args:
        source: Local path, ``gs://bucket/key`` GCS URI, or open binary file-like.
        spark: SparkSession used to materialize the result DataFrame.
        inference_host: Inference service base URL. If ``None``, reads from the
            ``INFERENCE_HOST`` env var (falls back to ``http://localhost:8082``).
        decompress: Attempt zstd decompression of MPLog payloads.
        schema: Optional pre-fetched schema. When set, schema lookups are
            skipped (assumes a single schema across all records).
        needed_columns: Optional subset of feature names to keep.
        strict: If True, raise :class:`FormatError` on any malformed frame.

    Returns:
        A DataFrame with one row per ``(record, entity)``. Columns are
        ``entity_id`` followed by record metadata
        (``timestamp_ns``, ``mp_config_id``, ``version``, ``format_type``,
        ``user_id``, ``tracking_id``, ``parent_entity``) and then the decoded
        feature columns.
    """
    rows: List[dict] = list(
        iter_decoded_log_rows(
            source,
            inference_host=inference_host,
            decompress=decompress,
            schema=schema,
            needed_columns=needed_columns,
            strict=strict,
            stringify_values=True,
        )
    )

    if not rows:
        from pyspark.sql.types import LongType, StringType, StructField, StructType

        fields = [StructField("entity_id", StringType(), True)]
        for col in _RECORD_METADATA_COLUMNS:
            dtype = LongType() if col in ("timestamp_ns", "version", "format_type") else StringType()
            fields.append(StructField(col, dtype, True))
        return spark.createDataFrame([], StructType(fields))

    # Build ordered columns: entity_id, record metadata (in fixed order), then features.
    feature_names: List[str] = []
    seen = set(("entity_id",) + _RECORD_METADATA_COLUMNS)
    for row in rows:
        for k in row.keys():
            if k not in seen:
                feature_names.append(k)
                seen.add(k)

    ordered_cols = ["entity_id", *_RECORD_METADATA_COLUMNS, *feature_names]
    # Backfill so every row has every column (Spark needs uniform dict keys).
    for row in rows:
        for col in ordered_cols:
            row.setdefault(col, None)

    df = spark.createDataFrame(rows)
    return df.select(ordered_cols)


def _ordered_columns(rows: List[dict]) -> List[str]:
    """Return ``entity_id, <metadata>, <feature>...`` in stable order."""
    feature_names: List[str] = []
    seen = set(("entity_id",) + _RECORD_METADATA_COLUMNS)
    for row in rows:
        for k in row.keys():
            if k not in seen:
                feature_names.append(k)
                seen.add(k)
    return ["entity_id", *_RECORD_METADATA_COLUMNS, *feature_names]


def decode_log_file_to_pandas(
    source: LogSource,
    inference_host: Optional[str] = None,
    decompress: bool = True,
    schema: Optional[List[FeatureInfo]] = None,
    needed_columns: Optional[Collection[str]] = None,
    strict: bool = False,
):
    """Decode a .log file straight into a pandas DataFrame.

    Same row shape as :func:`decode_log_file` but skips Spark entirely.
    Best for files that fit comfortably in driver memory.

    Requires ``pandas`` to be importable (transitively available via the
    ``pyarrow`` dependency, but not declared explicitly — install it if your
    environment lacks it).
    """
    try:
        import pandas as pd  # type: ignore
    except ImportError as exc:
        raise ImportError(
            "decode_log_file_to_pandas requires pandas. Install it with: pip install pandas"
        ) from exc

    rows = list(
        iter_decoded_log_rows(
            source,
            inference_host=inference_host,
            decompress=decompress,
            schema=schema,
            needed_columns=needed_columns,
            strict=strict,
            stringify_values=False,
        )
    )
    if not rows:
        return pd.DataFrame(columns=["entity_id", *_RECORD_METADATA_COLUMNS])

    cols = _ordered_columns(rows)
    return pd.DataFrame(rows, columns=cols)


def decode_log_file_to_csv(
    source: LogSource,
    output_path: Union[str, Path],
    inference_host: Optional[str] = None,
    decompress: bool = True,
    schema: Optional[List[FeatureInfo]] = None,
    needed_columns: Optional[Collection[str]] = None,
    strict: bool = False,
) -> int:
    """Stream a .log file into a single CSV file. Returns rows written.

    Uses two passes: first collects column names by buffering rows, then writes
    a single CSV with a stable header. Memory cost is bounded by the rows in
    one file (typical asyncloguploader files yield well under 1M rows).
    """
    import csv

    rows = list(
        iter_decoded_log_rows(
            source,
            inference_host=inference_host,
            decompress=decompress,
            schema=schema,
            needed_columns=needed_columns,
            strict=strict,
            stringify_values=True,
        )
    )
    output_path = str(output_path)
    if not rows:
        with open(output_path, "w", newline="") as fh:
            csv.writer(fh).writerow(["entity_id", *_RECORD_METADATA_COLUMNS])
        return 0

    cols = _ordered_columns(rows)
    with open(output_path, "w", newline="") as fh:
        writer = csv.DictWriter(fh, fieldnames=cols)
        writer.writeheader()
        for row in rows:
            writer.writerow({c: row.get(c) for c in cols})
    return len(rows)


def decode_log_file_to_jsonl(
    source: LogSource,
    output_path: Union[str, Path],
    inference_host: Optional[str] = None,
    decompress: bool = True,
    schema: Optional[List[FeatureInfo]] = None,
    needed_columns: Optional[Collection[str]] = None,
    strict: bool = False,
) -> int:
    """Stream a .log file into newline-delimited JSON. Returns rows written.

    Unlike CSV, rows are written as they are produced — memory cost is one row
    at a time.
    """
    import json

    output_path = str(output_path)
    count = 0
    with open(output_path, "w") as fh:
        for row in iter_decoded_log_rows(
            source,
            inference_host=inference_host,
            decompress=decompress,
            schema=schema,
            needed_columns=needed_columns,
            strict=strict,
            stringify_values=False,
        ):
            # bytes values -> hex hex strings so json can serialize them
            serializable = {
                k: (v.hex() if isinstance(v, (bytes, bytearray)) else v) for k, v in row.items()
            }
            fh.write(json.dumps(serializable, default=str))
            fh.write("\n")
            count += 1
    return count


def write_parsed_log(
    source: LogSource,
    output_path: Union[str, Path],
    inference_host: Optional[str] = None,
    decompress: bool = True,
    schema: Optional[List[FeatureInfo]] = None,
    needed_columns: Optional[Collection[str]] = None,
    strict: bool = False,
) -> int:
    """Write a human-readable parsed log (asynclogparser ``.parsed.log`` format).

    The output mirrors the layout produced by ``asynclogparse.py``::

        === Record N ===
        Timestamp: <utc> (raw: <ns> ns)
        User ID: ...
        Tracking ID: ...
        Model Config ID: ...
        Schema Version: <int>
        Format Type: <n> (proto|arrow|parquet)
        Entities: <count>
        ...
        --- Entity 1 ---
        Entity ID: ...
        Features:
          <name>: <value>

    Returns the number of records written.
    """
    import datetime as _dt

    output_path = str(output_path)
    schema_cache: dict = {}

    if inference_host is None:
        inference_host = os.getenv("INFERENCE_HOST", "http://localhost:8082")

    record_count = 0
    with open(output_path, "w") as out:
        for ts_ns, payload in iter_log_records(source, strict=strict):
            record_count += 1

            # Re-run the decode for this single record (so the output groups
            # entities by record, like asynclogparser does).
            working = payload
            if decompress and len(working) >= 4 and working[:4] == b"\x28\xb5\x2f\xfd":
                try:
                    import zstandard as zstd  # type: ignore
                    working = zstd.ZstdDecompressor().decompress(working)
                except ImportError:
                    pass

            try:
                parsed = parse_mplog_protobuf(working)
            except Exception as e:
                out.write(f"=== Record {record_count}: PARSE ERROR ===\nError: {e}\n\n")
                continue

            out.write(f"=== Record {record_count} ===\n")
            if ts_ns:
                ts = _dt.datetime.fromtimestamp(ts_ns / 1e9, tz=_dt.timezone.utc)
                out.write(
                    f"Timestamp: {ts.strftime('%Y-%m-%d %H:%M:%S.%f')} UTC (raw: {ts_ns} ns)\n"
                )
            out.write(f"User ID: {parsed.user_id}\n")
            out.write(f"Tracking ID: {parsed.tracking_id}\n")
            out.write(f"Model Config ID: {parsed.model_proxy_config_id}\n")
            out.write(f"Schema Version: {parsed.version}\n")
            fmt_name = {0: "proto", 1: "arrow", 2: "parquet"}.get(
                parsed.format_type, f"unknown({parsed.format_type})"
            )
            out.write(f"Format Type: {parsed.format_type} ({fmt_name})\n")
            entities = list(parsed.entities or [])
            parents = list(parsed.parent_entity or [])
            encoded = list(getattr(parsed, "_encoded_features", []) or [])
            out.write(f"Entities: {len(entities)}\n")
            if entities:
                out.write(f"Entity IDs: {entities}\n")
            out.write(f"Parent Entities: {len(parents)}\n")
            if parents:
                out.write(f"Parent Entity IDs: {parents}\n")
            out.write(f"Encoded Features: {len(encoded)}\n\n")

            # Resolve schema once per record.
            if schema is not None:
                feat_schema = schema
            else:
                key = (parsed.model_proxy_config_id or "", int(parsed.version or 0))
                feat_schema = schema_cache.get(key)
                if feat_schema is None and key[0]:
                    try:
                        feat_schema = get_feature_schema(key[0], key[1], inference_host)
                        schema_cache[key] = feat_schema
                    except Exception as e:
                        warnings.warn(
                            f"Schema fetch failed for {key}: {e}", UserWarning, stacklevel=2
                        )
                        feat_schema = None

            fmt = FORMAT_TYPE_MAP.get(parsed.format_type, Format.PROTO)
            decoder = {
                Format.ARROW: decode_arrow_features,
                Format.PARQUET: decode_parquet_features,
            }.get(fmt, decode_proto_features)

            for i, enc in enumerate(encoded):
                out.write(f"--- Entity {i+1} ---\n")
                eid = entities[i] if i < len(entities) else ""
                if eid:
                    out.write(f"Entity ID: {eid}\n")
                pe = parents[i] if i < len(parents) else ""
                if pe:
                    out.write(f"Parent Entity: {pe}\n")
                out.write("Features:\n")
                if feat_schema is None:
                    out.write(f"  (encoded_features length: {len(enc)} bytes — no schema)\n")
                else:
                    try:
                        decoded = decoder(enc, feat_schema, needed_columns=needed_columns)
                        for k, v in decoded.items():
                            out.write(f"  {k}: {v}\n")
                    except Exception as e:
                        out.write(f"  (decode error: {e})\n")
                out.write("\n")
            out.write("\n")

    return record_count


def analyze_log_file(source: LogSource, sample_records: int = 5) -> dict:
    """Structural analysis of a .log file. No protobuf decoding, no schema fetch.

    Mirrors ``asynclogparser`` ``--analyze``: returns frame-by-frame structure,
    record counts, data utilization, and timestamp-prefix detection.

    Returns a dict with keys: ``file_bytes``, ``frames`` (list), ``totals``,
    ``timestamp_format`` (``True`` / ``False`` / ``None``).
    """
    import datetime as _dt

    frames: List[dict] = []
    total_valid = 0
    total_records = 0
    sample: List[bytes] = []

    with _open_source(source) as stream:
        idx = 0
        while True:
            header = _read_exact(stream, _HEADER_SIZE)
            if len(header) == 0:
                break
            if len(header) < _HEADER_SIZE:
                frames.append({"index": idx, "error": "truncated header"})
                break

            capacity = int.from_bytes(header[0:4], "little")
            valid = int.from_bytes(header[4:8], "little")
            info = {"index": idx, "capacity": capacity, "valid_data_bytes": valid, "records": 0}

            if capacity < _HEADER_SIZE or valid > capacity:
                info["error"] = f"invalid (cap={capacity}, valid={valid})"
                frames.append(info)
                break

            data = _read_exact(stream, capacity - _HEADER_SIZE)
            if len(data) < capacity - _HEADER_SIZE:
                info["error"] = f"truncated body ({len(data)}/{capacity - _HEADER_SIZE})"
                frames.append(info)
                break

            off = 0
            limit = min(valid, len(data))
            while off + _LENGTH_PREFIX_SIZE <= limit:
                rec_len = int.from_bytes(data[off : off + _LENGTH_PREFIX_SIZE], "little")
                off += _LENGTH_PREFIX_SIZE
                if rec_len == 0:
                    continue
                if off + rec_len > limit:
                    break
                info["records"] += 1
                if len(sample) < sample_records:
                    sample.append(data[off : off + rec_len])
                off += rec_len

            total_records += info["records"]
            total_valid += valid
            frames.append(info)
            idx += 1

    # Best-effort total file size (only works for seekable streams).
    file_bytes: Optional[int] = None
    try:
        with _open_source(source) as s2:
            s2.seek(0, 2)
            file_bytes = s2.tell()
    except Exception:
        pass

    # Detect whether records carry an 8-byte UnixNano prefix (post-2024 format).
    ts_format: Optional[bool] = None
    if sample:
        ts_format = True
        for rec in sample:
            if len(rec) < _TIMESTAMP_SIZE:
                ts_format = False
                break
            ts_val = int.from_bytes(rec[:_TIMESTAMP_SIZE], "little")
            # Reasonable UnixNano range (~2020-2033).
            if not (1_600_000_000_000_000_000 < ts_val < 2_000_000_000_000_000_000):
                ts_format = False
                break

    return {
        "file_bytes": file_bytes,
        "frames": frames,
        "totals": {
            "frames": len(frames),
            "valid_bytes": total_valid,
            "records": total_records,
            "utilization": (total_valid / file_bytes) if file_bytes else None,
        },
        "timestamp_format": ts_format,
    }


def print_analysis(result: dict) -> None:
    """Pretty-print the output of :func:`analyze_log_file`."""
    fb = result.get("file_bytes")
    print("=" * 60)
    print(f"FILE ANALYSIS  ({fb:,} bytes / {fb/(1024*1024):.2f} MB)" if fb else "FILE ANALYSIS")
    print("=" * 60)
    t = result["totals"]
    print(f"  frames: {t['frames']}, records: {t['records']:,}, valid: {t['valid_bytes']:,} bytes")
    if t.get("utilization") is not None:
        print(f"  utilization: {t['utilization']*100:.1f}%")
    print()
    print("PER-FRAME:")
    for fi in result["frames"]:
        if "error" in fi:
            print(f"  frame {fi['index']:3d}  ERROR: {fi['error']}")
            continue
        cap_mb = fi["capacity"] / (1024 * 1024)
        valid_mb = fi["valid_data_bytes"] / (1024 * 1024)
        pct = (fi["valid_data_bytes"] / fi["capacity"] * 100) if fi["capacity"] else 0
        print(
            f"  frame {fi['index']:3d}  cap={cap_mb:7.2f}MB  valid={valid_mb:7.2f}MB "
            f"({pct:5.1f}%)  records={fi['records']:,}"
        )
    print()
    tsf = result.get("timestamp_format")
    if tsf is True:
        print("  records carry an 8-byte UnixNano timestamp prefix.")
    elif tsf is False:
        print("  WARNING: records do NOT carry the expected timestamp prefix.")


# ---------------------------------------------------------------------------
# Directory / prefix handling
# ---------------------------------------------------------------------------

def _looks_like_single_file(s: str, suffix: str) -> bool:
    return s.endswith(suffix)


def list_log_sources(
    source: Union[str, Path, List[Union[str, Path]]],
    suffix: str = ".log",
) -> List[str]:
    """Expand ``source`` into a concrete list of ``.log`` file paths / URIs.

    Rules:
      - A ``list`` / ``tuple`` — returned as-is (stringified).
      - A local path to a file ending with ``suffix`` — ``[source]``.
      - A local directory — non-recursive listing of files ending with ``suffix``.
      - A ``gs://bucket/key`` URI ending with ``suffix`` — ``[source]``.
      - A ``gs://bucket/prefix/`` URI (or any URI without ``suffix``) — every
        object under that prefix whose name ends with ``suffix``, returned as
        ``gs://bucket/<name>`` in lexicographic order.

    File-like objects can't be enumerated — pass them directly to the
    single-file APIs.
    """
    if isinstance(source, (list, tuple)):
        return [str(s) for s in source]

    if isinstance(source, Path):
        source = str(source)

    if not isinstance(source, str):
        raise TypeError(
            f"list_log_sources needs a path, gs:// URI, or list; got {type(source).__name__}"
        )

    # GCS
    if source.startswith("gs://"):
        if _looks_like_single_file(source, suffix):
            return [source]
        try:
            from google.cloud import storage  # type: ignore
        except ImportError as exc:
            raise ImportError(
                "Listing gs:// prefixes requires 'google-cloud-storage'. "
                "Install it with: pip install google-cloud-storage"
            ) from exc

        rest = source[len("gs://") :]
        if "/" in rest:
            bucket_name, prefix = rest.split("/", 1)
        else:
            bucket_name, prefix = rest, ""
        client = storage.Client()
        blobs = client.list_blobs(bucket_name, prefix=prefix)
        found = sorted(
            f"gs://{bucket_name}/{b.name}" for b in blobs if b.name.endswith(suffix)
        )
        return found

    # Local
    if os.path.isdir(source):
        entries = sorted(
            os.path.join(source, name)
            for name in os.listdir(source)
            if name.endswith(suffix) and os.path.isfile(os.path.join(source, name))
        )
        return entries

    if os.path.isfile(source):
        return [source]

    if _looks_like_single_file(source, suffix):
        # Not an existing file, but user clearly asked for one — let the caller
        # surface the FileNotFoundError.
        return [source]

    raise FileNotFoundError(
        f"list_log_sources: {source!r} is neither an existing file/dir nor a gs:// URI"
    )


def decode_logs(
    source: Union[LogSource, List[Union[str, Path]]],
    spark: "SparkSession",
    inference_host: Optional[str] = None,
    decompress: bool = True,
    schema: Optional[List[FeatureInfo]] = None,
    needed_columns: Optional[Collection[str]] = None,
    strict: bool = False,
    suffix: str = ".log",
) -> "SparkDataFrame":
    """Decode a single ``.log`` file **or** every ``.log`` file under a directory /
    GCS prefix / list, and return one unioned Spark DataFrame.

    Same row shape and semantics as :func:`decode_log_file`. Uses
    :func:`list_log_sources` to expand the source, then decodes each file and
    unions with ``unionByName(..., allowMissingColumns=True)`` so per-file
    schema drift (e.g. different feature subsets) doesn't break the merge.
    """
    # Fast path: caller passed a file-like directly.
    if hasattr(source, "read") and not isinstance(source, (str, Path, list, tuple)):
        return decode_log_file(
            source,  # type: ignore[arg-type]
            spark,
            inference_host=inference_host,
            decompress=decompress,
            schema=schema,
            needed_columns=needed_columns,
            strict=strict,
        )

    sources = list_log_sources(source, suffix=suffix)  # type: ignore[arg-type]
    if not sources:
        # Return an empty DataFrame with the standard column set.
        from pyspark.sql.types import LongType, StringType, StructField, StructType

        fields = [StructField("entity_id", StringType(), True)]
        for col in _RECORD_METADATA_COLUMNS:
            dtype = LongType() if col in ("timestamp_ns", "version", "format_type") else StringType()
            fields.append(StructField(col, dtype, True))
        return spark.createDataFrame([], StructType(fields))

    from functools import reduce

    dfs = [
        decode_log_file(
            s,
            spark,
            inference_host=inference_host,
            decompress=decompress,
            schema=schema,
            needed_columns=needed_columns,
            strict=strict,
        )
        for s in sources
    ]
    if len(dfs) == 1:
        return dfs[0]
    return reduce(lambda a, b: a.unionByName(b, allowMissingColumns=True), dfs)


def decode_logs_to_pandas(
    source: Union[LogSource, List[Union[str, Path]]],
    inference_host: Optional[str] = None,
    decompress: bool = True,
    schema: Optional[List[FeatureInfo]] = None,
    needed_columns: Optional[Collection[str]] = None,
    strict: bool = False,
    suffix: str = ".log",
):
    """Same as :func:`decode_logs` but returns a pandas DataFrame (no Spark).

    Files are decoded sequentially, then concatenated with
    ``pd.concat(..., ignore_index=True)``.
    """
    try:
        import pandas as pd  # type: ignore
    except ImportError as exc:
        raise ImportError(
            "decode_logs_to_pandas requires pandas. Install it with: pip install pandas"
        ) from exc

    if hasattr(source, "read") and not isinstance(source, (str, Path, list, tuple)):
        return decode_log_file_to_pandas(
            source,  # type: ignore[arg-type]
            inference_host=inference_host,
            decompress=decompress,
            schema=schema,
            needed_columns=needed_columns,
            strict=strict,
        )

    sources = list_log_sources(source, suffix=suffix)  # type: ignore[arg-type]
    if not sources:
        return pd.DataFrame(columns=["entity_id", *_RECORD_METADATA_COLUMNS])

    frames = [
        decode_log_file_to_pandas(
            s,
            inference_host=inference_host,
            decompress=decompress,
            schema=schema,
            needed_columns=needed_columns,
            strict=strict,
        )
        for s in sources
    ]
    return pd.concat(frames, ignore_index=True)
