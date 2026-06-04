"""Go<->Python decode parity tests.

Each fixture below is the EXACT byte output of the Go encoder
(model-proxy pkg/utils ConvertStringToType, which mirrors go-core
datatypeconverter) for a known input value. The Python decoder must invert
those bytes back to the original value for every supported type.

Encoding facts these tests pin down (the depth-level parity points):
  * scalars  -> fixed-width little-endian binary (struct-exact)
  * FP16 scalar -> bfloat16 (top 16 bits of float32: uint16(Float32bits>>16))
  * vectors  -> JSON text (json.Marshal of the typed slice)
  * uint8 vector -> base64 JSON string (Go marshals []uint8 as base64)
  * binary (feature-store / byte-column) vectors -> packed LE elements,
    FP16 elements are true IEEE-754 half
"""

import math

from inference_logging_client.decoder import (
    decode_scalar_value,
    decode_vector_or_string,
)
from inference_logging_client.utils import is_sized_type


def _decode(hex_bytes: str, feature_type: str):
    b = bytes.fromhex(hex_bytes)
    if is_sized_type(feature_type):
        return decode_vector_or_string(b, feature_type)
    return decode_scalar_value(b, feature_type)


# (feature_type, go_encoded_hex, expected_value)
STRING_PATH_FIXTURES = [
    ("DataTypeInt8", "fb", -5),
    ("DataTypeInt16", "d4fe", -300),
    ("DataTypeInt32", "90eefeff", -70000),
    ("DataTypeInt64", "15cd5b0700000000", 123456789),
    ("DataTypeUint8", "c8", 200),
    ("DataTypeUint16", "60ea", 60000),
    ("DataTypeUint32", "00286bee", 4000000000),
    ("DataTypeUint64", "000008c5a1d8ccf9", 18000000000000000000),
    ("DataTypeFP16", "c03f", 1.5),
    ("DataTypeFP32", "d00f4940", 3.14159),
    ("DataTypeFP64", "9b91048b0abf0540", 2.718281828),
    ("DataTypeBool", "01", True),
    ("DataTypeString", "68656c6c6f", "hello"),
    ("DataTypeInt32Vector", "5b312c322c335d", [1, 2, 3]),
    ("DataTypeInt64Vector", "5b31302c32305d", [10, 20]),
    ("DataTypeFP32Vector", "5b312e352c322e355d", [1.5, 2.5]),
    ("DataTypeFP64Vector", "5b312e312c322e325d", [1.1, 2.2]),
    ("DataTypeFP16Vector", "5b312e352c322e355d", [1.5, 2.5]),
    ("DataTypeUint32Vector", "5b372c385d", [7, 8]),
    ("DataTypeBoolVector", "5b747275652c66616c73655d", [True, False]),
    ("DataTypeStringVector", "5b2261222c2262225d", ["a", "b"]),
    ("DataTypeInt8Vector", "5b312c2d325d", [1, -2]),
    ("DataTypeUint8Vector", "224151493d22", [1, 2]),
]


def _equal(actual, expected):
    if isinstance(expected, float):
        return isinstance(actual, (int, float)) and math.isclose(actual, expected, rel_tol=1e-3, abs_tol=1e-3)
    if isinstance(expected, list):
        return (
            isinstance(actual, list)
            and len(actual) == len(expected)
            and all(_equal(a, e) for a, e in zip(actual, expected))
        )
    return actual == expected


def test_string_path_parity_all_types():
    """Every Go ConvertStringToType output decodes back to its input value."""
    for feature_type, hex_bytes, expected in STRING_PATH_FIXTURES:
        got = _decode(hex_bytes, feature_type)
        assert _equal(got, expected), f"{feature_type}: got {got!r}, expected {expected!r}"


def test_fp16_scalar_is_bfloat16():
    # Go writes FP16 scalars as bfloat16 (uint16(Float32bits(1.5)>>16) = 0x3FC0).
    assert _equal(decode_scalar_value(bytes.fromhex("c03f"), "DataTypeFP16"), 1.5)


def test_fp8_decode_matches_go():
    # Ported from go-core float8.FP8E4M3ToFP32Value / FP8E5M2ToFP32Value.
    assert _equal(decode_scalar_value(bytes([0x38]), "DataTypeFP8E4M3"), 1.0)
    assert _equal(decode_scalar_value(bytes([0x3C]), "DataTypeFP8E4M3"), 1.5)
    assert _equal(decode_scalar_value(bytes([0x3C]), "DataTypeFP8E5M2"), 1.0)
    assert _equal(decode_scalar_value(bytes([0x40]), "DataTypeFP8E5M2"), 2.0)
    # vector form
    assert _equal(
        decode_vector_or_string(bytes([0x38, 0x3C]), "DataTypeFP8E4M3Vector"), [1.0, 1.5]
    )


def test_float_values_are_full_precision():
    # Go keeps the exact float32 value; we must not round to 6 decimals.
    v = decode_scalar_value(bytes.fromhex("d00f4940"), "DataTypeFP32")
    assert abs(v - 3.1415901184) < 1e-9, v


def test_binary_byte_column_path_no_regression():
    # Feature-store / byte-column vectors are packed binary, NOT JSON.
    # FP16 elements are true IEEE-754 half: 1.5 -> 0x3E00, 2.5 -> 0x4100 (LE).
    fp16_vec = bytes.fromhex("003e0041")
    assert _equal(decode_vector_or_string(fp16_vec, "DataTypeFP16Vector"), [1.5, 2.5])

    # Raw (non-base64) uint8 vector elements decode as raw bytes.
    assert decode_vector_or_string(bytes([1, 2]), "DataTypeUint8Vector") == [1, 2]
