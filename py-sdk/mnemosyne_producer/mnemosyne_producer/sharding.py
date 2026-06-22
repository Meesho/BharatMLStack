"""
Sharding & key construction.

  shard_id = crc32_ieee(key_bytes) % num_shards

Python's `binascii.crc32` and Spark SQL's `crc32(col)` and Go's
`hash/crc32.ChecksumIEEE` all use the IEEE 802.3 polynomial, so the shard_id is
identical across Databricks (Spark SQL), the Python encoder, and the Go/Rust
serving stack.

Key convention: composite keys are pipe-joined and UTF-8 encoded — matches the
existing online-feature-store handler `getKeyString` in
`internal/handler/feature/retrieve.go`.
"""
from __future__ import annotations

import binascii
from typing import Iterable, Sequence


KEY_SEP = "|"


def crc32_ieee(b: bytes) -> int:
    """CRC32 (IEEE polynomial), unsigned 32-bit."""
    return binascii.crc32(b) & 0xFFFFFFFF


def make_key(key_columns: Sequence[str], row: dict, sep: str = KEY_SEP) -> str:
    """Build the composite entity key for an entity row. Null/missing → empty string."""
    parts = []
    for c in key_columns:
        v = row.get(c)
        parts.append("" if v is None else str(v))
    return sep.join(parts)


def shard_id(key_str: str, num_shards: int) -> int:
    if num_shards <= 0:
        raise ValueError(f"num_shards must be positive, got {num_shards}")
    return crc32_ieee(key_str.encode("utf-8")) % num_shards
