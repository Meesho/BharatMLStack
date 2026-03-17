"""Compiled decode plans for proto feature decoding (optimized hot path)."""

from __future__ import annotations

from typing import Optional

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
    """Build a decode plan with feature_type only (no callables; picklable for broadcast).

    Each plan entry is:
    - ("decode", name, is_sized, fixed_size, feature_type): decode this feature.
    - ("skip_bytes", total_size): advance pointer by total_size.
    - ("skip_sized",): skip one variable-length field.

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
                plan.append(("decode", f.name, True, None, canonical))
            else:
                plan.append(("skip_sized",))
        else:
            if fixed_size is None:
                raise ValueError(f"Unknown scalar size for type {f.feature_type!r}")
            if need:
                if run_skip_size > 0:
                    plan.append(("skip_bytes", run_skip_size))
                    run_skip_size = 0
                plan.append(("decode", f.name, False, fixed_size, canonical))
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

    Plan entries store feature_type string only (no callables), so the plan is picklable
    for Spark broadcast. Decoder is resolved at decode time in proto_decoder.

    Each plan entry is:
    - ("scalar", name, fixed_size, feature_type, should_decode)
    - ("var", name, None, feature_type, should_decode)
    - ("skip_bytes", total_size, None, False, None)
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
            plan.append(("var", f.name, None, canonical, need))
        else:
            if fixed_size is None:
                raise ValueError(f"Unknown scalar size for type {f.feature_type!r}")
            if need:
                if run_skip_size > 0:
                    plan.append(("skip_bytes", run_skip_size, None, False, None))
                    run_skip_size = 0
                plan.append(("scalar", f.name, fixed_size, canonical, True))
            else:
                run_skip_size += fixed_size

    if run_skip_size > 0:
        plan.append(("skip_bytes", run_skip_size, None, False, None))

    return plan


def try_build_fixed_plan(schema: list[FeatureInfo]) -> Optional[tuple]:
    """If all features are fixed-size scalars, return (offsets, sizes, names, types).

    No callables stored so the plan is picklable for Spark broadcast. Decoder is
    resolved at decode time in proto_decoder via decode_scalar_value(bytes, types[i]).
    Returns None if any feature is variable-length (string/vector).
    """
    offsets: list[int] = []
    sizes: list[int] = []
    names: list[str] = []
    types: list[str] = []
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
        pos += sz
    return (tuple(offsets), tuple(sizes), tuple(names), tuple(types))
