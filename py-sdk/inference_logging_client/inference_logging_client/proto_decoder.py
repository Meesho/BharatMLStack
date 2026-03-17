"""Optimized proto decoders using precompiled plans (zero-copy, no branching per feature)."""

from __future__ import annotations

import struct
from typing import Any

from .decoder import decode_scalar_value, decode_vector_or_string

# Pre-created Struct objects for unpack_from — avoids repeated format string parsing
_STRUCT_U16 = struct.Struct("<H")
_STRUCT_U8 = struct.Struct("<B")


def decode_proto_selective(data: bytes, plan: list[tuple]) -> dict[str, Any]:
    """Decode proto-encoded features using a precompiled plan. Uses memoryview for zero-copy slicing.

    Plan entries (no callables; feature_type is used to resolve decoder at decode time):
    - ("scalar", name, fixed_size, feature_type, should_decode)
    - ("var", name, None, feature_type, should_decode)
    - ("skip_bytes", total_size, None, False, None)

    Skips byte 0 (generated flag), starts at pos=1. On any bounds violation, sets remaining
    features to None and returns (does not raise).
    """
    if len(data) < 1:
        return _empty_result_from_plan(plan)

    mv = memoryview(data)
    pos = 1  # skip generated flag
    result: dict[str, Any] = {}
    decoded_names = _decoded_names_from_plan(plan)

    for entry in plan:
        kind = entry[0]
        if kind == "skip_bytes":
            total_size = entry[1]
            if pos + total_size > len(data):
                _fill_remaining_none(result, decoded_names)
                return result
            pos += total_size
            continue
        if kind == "scalar":
            name, fixed_size, feature_type, should_decode = entry[1], entry[2], entry[3], entry[4]
            if pos + fixed_size > len(data):
                _fill_remaining_none(result, decoded_names)
                return result
            if should_decode:
                result[name] = decode_scalar_value(
                    mv[pos : pos + fixed_size].tobytes(), feature_type
                )
            pos += fixed_size
            continue
        if kind == "var":
            name, feature_type, should_decode = entry[1], entry[3], entry[4]
            if pos + 2 > len(data):
                _fill_remaining_none(result, decoded_names)
                return result
            size = _STRUCT_U16.unpack_from(mv, pos)[0]
            pos += 2
            if pos + size > len(data):
                _fill_remaining_none(result, decoded_names)
                return result
            if should_decode:
                result[name] = decode_vector_or_string(
                    mv[pos : pos + size].tobytes(), feature_type
                )
            pos += size
            continue

    return result


def _empty_result_from_plan(plan: list[tuple]) -> dict[str, Any]:
    names = _decoded_names_from_plan(plan)
    return {n: None for n in names}


def _decoded_names_from_plan(plan: list[tuple]) -> list[str]:
    names: list[str] = []
    for entry in plan:
        kind = entry[0]
        if kind == "scalar" and entry[4] is True:
            names.append(entry[1])
        elif kind == "var" and entry[4] is True:
            names.append(entry[1])
    return names


def _fill_remaining_none(result: dict[str, Any], decoded_names: list[str]) -> None:
    for n in decoded_names:
        if n not in result:
            result[n] = None


def decode_proto_fixed(data: bytes, fixed_plan: tuple) -> dict[str, Any]:
    """Decode proto for all-scalar schemas using precomputed offsets. Zero branching per feature.

    fixed_plan: (offsets, sizes, names, types) from try_build_fixed_plan (no callables).
    Decoder resolved at decode time via decode_scalar_value(bytes, types[i]).
    """
    offsets, sizes, names, types = fixed_plan
    if not offsets:
        return {}
    min_length = 1 + offsets[-1] + sizes[-1]
    if len(data) < min_length:
        return {name: None for name in names}

    mv = memoryview(data)
    result: dict[str, Any] = {}
    for i in range(len(names)):
        o, sz = offsets[i], sizes[i]
        result[names[i]] = decode_scalar_value(mv[o : o + sz].tobytes(), types[i])
    return result
