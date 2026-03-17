"""Unit tests for decode_plan module."""

import pytest

from inference_logging_client.decode_plan import (
    compile_skip_plan,
    normalize_feature_type,
    try_build_fixed_plan,
)
from inference_logging_client.types import FeatureInfo


# --- normalize_feature_type ---


def test_normalize_feature_type_both_conventions():
    """Both 'FP32' and 'DataTypeFP32' normalize to same canonical form."""
    assert normalize_feature_type("FP32") == "FP32"
    assert normalize_feature_type("DataTypeFP32") == "FP32"
    assert normalize_feature_type("DataTypeInt64") == "INT64"
    assert normalize_feature_type("INT64") == "INT64"
    assert normalize_feature_type("DataTypeString") == "STRING"
    assert normalize_feature_type("FP32_VECTOR") == "FP32VECTOR"
    assert normalize_feature_type("DataTypeFP32Vector") == "FP32VECTOR"


def test_normalize_feature_type_none_empty_raises():
    """None or empty string raises ValueError."""
    with pytest.raises(ValueError, match="cannot be None or empty"):
        normalize_feature_type(None)
    with pytest.raises(ValueError, match="cannot be None or empty"):
        normalize_feature_type("")
    with pytest.raises(ValueError, match="cannot be None or empty"):
        normalize_feature_type("   ")


def test_normalize_feature_type_unknown_raises():
    """Unknown type raises ValueError."""
    with pytest.raises(ValueError, match="Unknown feature type"):
        normalize_feature_type("UnknownType123")
    with pytest.raises(ValueError, match="Unknown feature type"):
        normalize_feature_type("DataTypeUnknown")


# --- compile_skip_plan: needed_columns subset ---


def test_compile_skip_plan_needed_columns_subset():
    """Only features in needed_columns are decoded; rest skipped or collapsed."""
    schema = [
        FeatureInfo("a", "INT32", 0),
        FeatureInfo("b", "FP32", 1),
        FeatureInfo("c", "INT64", 2),
        FeatureInfo("d", "STRING", 3),
        FeatureInfo("e", "INT64", 4),
    ]
    plan = compile_skip_plan(schema, needed_columns={"b", "d"})
    decode_names = [e[1] for e in plan if e[0] == "decode"]
    assert decode_names == ["b", "d"]
    skip_bytes_total = sum(e[1] for e in plan if e[0] == "skip_bytes")
    # a=4, c=8, e=8 -> 20
    assert skip_bytes_total == 20
    skip_sized_count = sum(1 for e in plan if e[0] == "skip_sized")
    assert skip_sized_count == 0  # d is needed so decoded


def test_compile_skip_plan_decode_all():
    """When needed_columns is None, all features are decoded."""
    schema = [
        FeatureInfo("x", "INT32", 0),
        FeatureInfo("y", "FP32", 1),
    ]
    plan = compile_skip_plan(schema, needed_columns=None)
    assert len(plan) == 2
    assert plan[0][0] == "decode" and plan[0][1] == "x"
    assert plan[1][0] == "decode" and plan[1][1] == "y"


# --- compile_skip_plan: contiguous scalar run collapsing ---


def test_compile_skip_plan_contiguous_scalar_run_collapsing():
    """Consecutive scalars not in needed_columns become one skip_bytes."""
    schema = [
        FeatureInfo("s1", "INT32", 0),
        FeatureInfo("s2", "INT32", 1),
        FeatureInfo("s3", "FP32", 2),
        FeatureInfo("wanted", "FP64", 3),
    ]
    plan = compile_skip_plan(schema, needed_columns={"wanted"})
    # One skip_bytes(4+4+4=12), then decode wanted
    skip_entries = [e for e in plan if e[0] == "skip_bytes"]
    assert len(skip_entries) == 1
    assert skip_entries[0][1] == 12
    decode_entries = [e for e in plan if e[0] == "decode"]
    assert len(decode_entries) == 1 and decode_entries[0][1] == "wanted"


# --- both naming conventions in same schema ---


def test_compile_skip_plan_both_naming_conventions():
    """Schema can mix DataTypeX and X; plan stores feature_type (no callables, picklable)."""
    from inference_logging_client.decoder import decode_scalar_value
    import struct

    schema = [
        FeatureInfo("f1", "DataTypeFP32", 0),
        FeatureInfo("f2", "INT64", 1),
        FeatureInfo("f3", "DataTypeInt32", 2),
        FeatureInfo("f4", "DataTypeString", 3),
    ]
    plan = compile_skip_plan(schema, needed_columns=None)
    decode_entries = [e for e in plan if e[0] == "decode"]
    assert len(decode_entries) == 4
    names = [e[1] for e in decode_entries]
    assert names == ["f1", "f2", "f3", "f4"]
    # f4 is sized
    sized = [e for e in decode_entries if e[2] is True]
    assert len(sized) == 1 and sized[0][1] == "f4"
    # Scalars: entry is (kind, name, is_sized, fixed_size, feature_type); decode via feature_type
    for entry in decode_entries:
        if entry[2]:  # sized
            continue
        _, name, _is_sized, fixed_size, feature_type = entry
        if name == "f1":
            out = decode_scalar_value(struct.pack("<f", 1.5), feature_type)
            assert out == 1.5
        elif name == "f2":
            out = decode_scalar_value(struct.pack("<q", -99), feature_type)
            assert out == -99
        elif name == "f3":
            out = decode_scalar_value(struct.pack("<i", 42), feature_type)
            assert out == 42


# --- try_build_fixed_plan ---


def test_try_build_fixed_plan_all_scalar_returns_tuple():
    """All-scalar schema returns (offsets, sizes, names, types) — no callables, picklable."""
    schema = [
        FeatureInfo("a", "INT32", 0),
        FeatureInfo("b", "FP32", 1),
        FeatureInfo("c", "INT64", 2),
    ]
    result = try_build_fixed_plan(schema)
    assert result is not None
    offsets, sizes, names, types = result
    assert offsets == (1, 5, 9)  # after 1-byte flag: 4, 4, 8
    assert sizes == (4, 4, 8)
    assert names == ("a", "b", "c")
    assert types == ("INT32", "FP32", "INT64")


def test_try_build_fixed_plan_any_sized_returns_none():
    """Schema with any string/vector returns None."""
    schema_all_scalar = [
        FeatureInfo("a", "INT32", 0),
        FeatureInfo("b", "FP32", 1),
    ]
    assert try_build_fixed_plan(schema_all_scalar) is not None

    schema_with_string = [
        FeatureInfo("a", "INT32", 0),
        FeatureInfo("b", "STRING", 1),
    ]
    assert try_build_fixed_plan(schema_with_string) is None

    schema_with_vector = [
        FeatureInfo("a", "FP32", 0),
        FeatureInfo("b", "FP32VECTOR", 1),
    ]
    assert try_build_fixed_plan(schema_with_vector) is None
