"""End-to-end proto decode parity: a framed proto row of go-core bytes decoded
with go_string=True must yield exactly the strings go-core BytesToString emits."""

import struct

from inference_logging_client.types import FeatureInfo
from inference_logging_client.formats import decode_proto_features


def _sized(payload: bytes) -> bytes:
    return struct.pack("<H", len(payload)) + payload


def _build_row(features):
    """features: list of (FeatureInfo, raw_value_bytes, is_sized). Returns the
    proto per-entity blob: flag byte + per-feature framing."""
    out = bytearray([1])  # generated flag
    for _info, raw, is_sized in features:
        out += _sized(raw) if is_sized else raw
    return bytes(out)


def test_proto_go_string_full_row():
    # go-core canonical bytes (captured from go-core BytesToString fixtures)
    feats = [
        (FeatureInfo("i32", "DataTypeInt32", 0), bytes.fromhex("90eefeff"), False),     # -70000
        (FeatureInfo("f32", "DataTypeFP32", 1), bytes.fromhex("20bcbe4c"), False),       # 1e+08
        (FeatureInfo("f16", "DataTypeFP16", 2), bytes.fromhex("003e"), False),           # 1.5
        (FeatureInfo("b", "DataTypeBool", 3), bytes.fromhex("01"), False),               # true
        (FeatureInfo("f32v", "DataTypeFP32Vector", 4),
         bytes.fromhex("0000c03f00002040cdcccc3d"), True),                               # 1.5,2.5,0.1
        (FeatureInfo("u8v", "DataTypeUint8Vector", 5), bytes.fromhex("010203"), True),   # 1,2,3
        (FeatureInfo("s", "DataTypeString", 6), b"hello world", True),                   # hello world
    ]
    row = _build_row(feats)
    schema = [info for info, _, _ in feats]

    out = decode_proto_features(row, schema, go_string=True)
    assert out == {
        "i32": "-70000",
        "f32": "1e+08",
        "f16": "1.5",
        "b": "true",
        "f32v": "1.5,2.5,0.1",
        "u8v": "1,2,3",
        "s": "hello world",
    }, out


def test_proto_go_string_handles_string_path_json_vector():
    # model-proxy ConvertStringToType emits JSON for vectors; go_string must
    # reformat it into the same Go-canonical comma string.
    feats = [
        (FeatureInfo("f32v", "DataTypeFP32Vector", 0), b"[1.5,2.5,0.1]", True),
        (FeatureInfo("u8v", "DataTypeUint8Vector", 1), b'"AQID"', True),  # base64 of [1,2,3]
    ]
    row = _build_row(feats)
    schema = [info for info, _, _ in feats]
    out = decode_proto_features(row, schema, go_string=True)
    assert out["f32v"] == "1.5,2.5,0.1", out
    assert out["u8v"] == "1,2,3", out
