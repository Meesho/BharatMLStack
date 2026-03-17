"""Benchmarks: decode_proto_features vs decode_proto_selective vs decode_proto_fixed.

Requires package deps (pip install -e .). Run from py-sdk/inference_logging_client:
  python3 benchmarks/bench_proto_decoder.py
"""

import struct
import time
from typing import List

# Run from py-sdk/inference_logging_client: python benchmarks/bench_proto_decoder.py
# Or: python -m benchmarks.bench_proto_decoder (from py-sdk/inference_logging_client)
import sys
import os

_here = os.path.dirname(os.path.abspath(__file__))
_pkg_root = os.path.dirname(_here)  # py-sdk/inference_logging_client (contains inference_logging_client pkg)
if _pkg_root not in sys.path:
    sys.path.insert(0, _pkg_root)

from inference_logging_client.types import FeatureInfo
from inference_logging_client.formats import decode_proto_features
from inference_logging_client.decode_plan import compile_selective_plan, try_build_fixed_plan
from inference_logging_client.proto_decoder import decode_proto_selective, decode_proto_fixed


def _make_proto_bytes(schema: List[FeatureInfo], payload_sizes: List[int]) -> bytes:
    """Build proto-encoded bytes: 1 byte flag + features in order.
    For scalars we use payload_sizes[i] bytes; for var we use 2-byte size + payload_sizes[i] bytes.
    """
    from inference_logging_client.utils import is_sized_type, get_scalar_size

    chunks = [bytes([1])]  # generated flag
    for i, f in enumerate(schema):
        if is_sized_type(f.feature_type):
            size = payload_sizes[i] if i < len(payload_sizes) else 0
            chunks.append(struct.pack("<H", size))
            chunks.append(b"\x00" * size if size else b"")
        else:
            sz = get_scalar_size(f.feature_type) or 4
            chunks.append(b"\x00" * sz)
    return b"".join(chunks)


def _schema_120_mixed() -> List[FeatureInfo]:
    """120 features: 118 scalars (INT32/FP32/INT64) + 2 STRING at end (var)."""
    schema = []
    for i in range(118):
        if i % 3 == 0:
            schema.append(FeatureInfo(f"f{i}", "INT32", i))
        elif i % 3 == 1:
            schema.append(FeatureInfo(f"f{i}", "FP32", i))
        else:
            schema.append(FeatureInfo(f"f{i}", "INT64", i))
    schema.append(FeatureInfo("f118", "STRING", 118))
    schema.append(FeatureInfo("f119", "STRING", 119))
    return schema


def _schema_all_scalar(n: int = 60) -> List[FeatureInfo]:
    """All-scalar schema for fixed plan."""
    schema = []
    for i in range(n):
        if i % 3 == 0:
            schema.append(FeatureInfo(f"g{i}", "INT32", i))
        elif i % 3 == 1:
            schema.append(FeatureInfo(f"g{i}", "FP32", i))
        else:
            schema.append(FeatureInfo(f"g{i}", "INT64", i))
    return schema


def _run_bench(name: str, fn, *args, iterations: int = 50_000) -> float:
    # Warmup
    for _ in range(1000):
        fn(*args)
    start = time.perf_counter()
    for _ in range(iterations):
        fn(*args)
    elapsed = time.perf_counter() - start
    per_call_us = (elapsed / iterations) * 1e6
    print(f"  {name}: {per_call_us:.2f} µs/call ({iterations} iterations)")
    return per_call_us


def main() -> None:
    print("Proto decoder benchmarks (single-entity decode)")
    print("-" * 60)

    # 1) Mixed schema ~120 features, ~500 bytes
    schema_120 = _schema_120_mixed()
    payload_sizes_120 = [0] * 118 + [10, 12]  # two small strings
    data_120 = _make_proto_bytes(schema_120, payload_sizes_120)
    print(f"Mixed schema: {len(schema_120)} features, payload {len(data_120)} bytes")
    iterations = 30_000

    plan_all = compile_selective_plan(schema_120, needed_columns=None)
    needed_3 = {schema_120[0].name, schema_120[1].name, schema_120[2].name}
    plan_3 = compile_selective_plan(schema_120, needed_columns=needed_3)

    print("\n1) Old decode_proto_features (all columns)")
    _run_bench("decode_proto_features (120 cols)", decode_proto_features, data_120, schema_120, iterations=iterations)

    print("\n2) decode_proto_selective (all columns)")
    _run_bench("decode_proto_selective (120 cols)", decode_proto_selective, data_120, plan_all, iterations=iterations)

    print("\n3) decode_proto_selective (3 of 120 columns)")
    _run_bench("decode_proto_selective (3 cols)", decode_proto_selective, data_120, plan_3, iterations=iterations)

    # 4) All-scalar schema, fixed plan
    schema_scalar = _schema_all_scalar(60)
    data_scalar = _make_proto_bytes(schema_scalar, [])
    fixed_plan = try_build_fixed_plan(schema_scalar)
    assert fixed_plan is not None
    print(f"\nAll-scalar schema: {len(schema_scalar)} features, payload {len(data_scalar)} bytes")
    print("\n4) decode_proto_fixed (all-scalar)")
    _run_bench("decode_proto_fixed (60 scalars)", decode_proto_fixed, data_scalar, fixed_plan, iterations=iterations)

    # Sanity: same results
    out_old = decode_proto_features(data_120, schema_120)
    out_sel = decode_proto_selective(data_120, plan_all)
    for k in list(out_old.keys())[:5]:
        assert k in out_sel, k
    print("\nSanity: decode_proto_features vs decode_proto_selective (first 5 keys match).")


if __name__ == "__main__":
    main()
