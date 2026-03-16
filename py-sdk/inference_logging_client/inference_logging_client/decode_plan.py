"""Compiled decode plans for proto feature decoding (optimized hot path)."""

from __future__ import annotations

from typing import Any, Callable, Optional

from .decoder import decode_scalar_value, decode_vector_or_string
from .types import FeatureInfo
from .utils import (
    SCALAR_TYPE_SIZES,
    SIZED_TYPES,
    get_scalar_size,
    is_sized_type,
    normalize_type,
)

# Canonical (normalized) type names we support: scalars + sized (string/vector/bytes)
_CANONICAL_SCALAR = set(SCALAR_TYPE_SIZES.keys())
_CANONICAL_SIZED = {normalize_type(t) for t in SIZED_TYPES}


def normalize_feature_type(raw_type: str) -> str:
    """Canonicalize feature type (handles both 'FP32' and 'DataTypeFP32' conventions).

    Returns:
        Normalized uppercase string (no underscores, no DATATYPE prefix).
    Raises:
        ValueError: If raw_type is None/empty or maps to an unknown type.
    """
    if raw_type is None or (isinstance(raw_type, str) and not raw_type.strip()):
        raise ValueError("Feature type cannot be None or empty")
    canonical = raw_type.upper().replace("_", "").replace("DATATYPE", "")
    if not canonical:
        raise ValueError(f"Invalid feature type: {raw_type!r}")
    if canonical not in _CANONICAL_SCALAR and canonical not in _CANONICAL_SIZED:
        raise ValueError(f"Unknown feature type: {raw_type!r} (normalized: {canonical!r})")
    return canonical


def compile_skip_plan(
    schema: list[FeatureInfo],
    needed_columns: Optional[set[str]] = None,
) -> list[tuple]:
    """Build a decode plan with bound decoders and contiguous scalar-run collapsing.

    Each plan entry is:
    - ("decode", name, is_sized, fixed_size, decoder, feature_type): decode this feature.
    - ("skip_bytes", total_size): advance pointer by total_size (run of scalars not needed).
    - ("skip_sized",): skip one variable-length field (read 2-byte size, then skip that many bytes).

    If needed_columns is None, all features are decoded. Otherwise only names in needed_columns.
    """
    plan: list[tuple] = []
    decode_all = needed_columns is None
    run_skip_size = 0

    for f in schema:
        canonical = normalize_feature_type(f.feature_type)
        sized = is_sized_type(f.feature_type)
        fixed_size: Optional[int] = None if sized else get_scalar_size(f.feature_type)
        need = decode_all or (f.name in needed_columns)

        if sized:
            if run_skip_size > 0:
                plan.append(("skip_bytes", run_skip_size))
                run_skip_size = 0
            if need:
                decoder: Callable[[bytes], Any] = (
                    lambda b, ft=f.feature_type: decode_vector_or_string(b, ft)
                )
                plan.append(("decode", f.name, True, None, decoder, canonical))
            else:
                plan.append(("skip_sized",))
        else:
            if fixed_size is None:
                raise ValueError(f"Unknown scalar size for type {f.feature_type!r}")
            if need:
                if run_skip_size > 0:
                    plan.append(("skip_bytes", run_skip_size))
                    run_skip_size = 0
                dec = lambda b, ft=f.feature_type: decode_scalar_value(b, ft)
                plan.append(("decode", f.name, False, fixed_size, dec, canonical))
            else:
                run_skip_size += fixed_size

    if run_skip_size > 0:
        plan.append(("skip_bytes", run_skip_size))

    return plan


def compile_selective_plan(
    schema: list[FeatureInfo],
    needed_columns: Optional[set[str]] = None,
) -> list[tuple]:
    """Build a plan for proto_decoder.decode_proto_selective (scalar/var/skip_bytes entries).

    Each plan entry is:
    - ("scalar", name, fixed_size, decoder, should_decode, feature_type)
    - ("var", name, None, decoder, should_decode, feature_type)
    - ("skip_bytes", total_size, None, False, None)  # 5 elements

    All decisions (sizes, decode vs skip) are baked in; no is_sized_type/get_scalar_size at decode time.
    """
    plan: list[tuple] = []
    decode_all = needed_columns is None
    run_skip_size = 0

    for f in schema:
        canonical = normalize_feature_type(f.feature_type)
        sized = is_sized_type(f.feature_type)
        fixed_size: Optional[int] = None if sized else get_scalar_size(f.feature_type)
        need = decode_all or (f.name in needed_columns)

        if sized:
            if run_skip_size > 0:
                plan.append(("skip_bytes", run_skip_size, None, False, None))
                run_skip_size = 0
            decoder: Callable[[bytes], Any] = (
                lambda b, ft=f.feature_type: decode_vector_or_string(b, ft)
            )
            plan.append(("var", f.name, None, decoder, need, canonical))
        else:
            if fixed_size is None:
                raise ValueError(f"Unknown scalar size for type {f.feature_type!r}")
            if need:
                if run_skip_size > 0:
                    plan.append(("skip_bytes", run_skip_size, None, False, None))
                    run_skip_size = 0
                dec = lambda b, ft=f.feature_type: decode_scalar_value(b, ft)
                plan.append(("scalar", f.name, fixed_size, dec, True, canonical))
            else:
                run_skip_size += fixed_size

    if run_skip_size > 0:
        plan.append(("skip_bytes", run_skip_size, None, False, None))

    return plan


def try_build_fixed_plan(schema: list[FeatureInfo]) -> Optional[tuple]:
    """If all features are fixed-size scalars, return (offsets, sizes, names, types, decoders).

    offsets[i] = byte offset from start of entity payload (after 1-byte generated flag).
    sizes[i] = byte size of feature i.
    names[i], types[i], decoders[i] = name, canonical type, bound decoder for feature i.
    Returns None if any feature is variable-length (string/vector).
    """
    offsets: list[int] = []
    sizes: list[int] = []
    names: list[str] = []
    types: list[str] = []
    decoders: list[Callable[[bytes], Any]] = []
    pos = 1  # after generated flag
    for f in schema:
        canonical = normalize_feature_type(f.feature_type)
        if is_sized_type(f.feature_type):
            return None
        sz = get_scalar_size(f.feature_type)
        if sz is None:
            return None
        offsets.append(pos)
        sizes.append(sz)
        names.append(f.name)
        types.append(canonical)
        decoders.append(lambda b, ft=f.feature_type: decode_scalar_value(b, ft))
        pos += sz
    return (tuple(offsets), tuple(sizes), tuple(names), tuple(types), tuple(decoders))
