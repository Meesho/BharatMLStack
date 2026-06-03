"""Parity tests for the go-core datatypeconverter port.

Fixtures are the exact (hex, string) output of go-core
typeconverter.BytesToString, so this asserts the Python port reproduces Go's
canonical byte->string conversion for every type, byte-for-byte.
"""

from inference_logging_client.go_datatypeconverter import bytes_to_string, go_format_float

# (dtype, hex_bytes, expected_go_string) — captured from go-core BytesToString.
FIXTURES = [
    ("DataTypeBool", "01", "true"),
    ("DataTypeBool", "00", "false"),
    ("DataTypeInt8", "80", "-128"),
    ("DataTypeInt8", "7f", "127"),
    ("DataTypeInt16", "0080", "-32768"),
    ("DataTypeInt16", "ff7f", "32767"),
    ("DataTypeInt32", "90eefeff", "-70000"),
    ("DataTypeInt32", "ffffff7f", "2147483647"),
    ("DataTypeInt64", "0000000000000080", "-9223372036854775808"),
    ("DataTypeInt64", "15cd5b0700000000", "123456789"),
    ("DataTypeUint8", "00", "0"),
    ("DataTypeUint8", "ff", "255"),
    ("DataTypeUint16", "ffff", "65535"),
    ("DataTypeUint32", "00286bee", "4000000000"),
    ("DataTypeUint64", "ffffffffffffffff", "18446744073709551615"),
    ("DataTypeFP32", "d00f4940", "3.14159"),
    ("DataTypeFP32", "0000003f", "-0.5".replace("-", "")),  # placeholder, replaced below
    ("DataTypeFP32", "20bcbe4c", "1e+08"),
    ("DataTypeFP32", "95bfd633", "1e-07"),
    ("DataTypeFP32", "00000000", "0"),
    ("DataTypeFP64", "9b91048b0abf0540", "2.718281828"),
    ("DataTypeFP16", "003e", "1.5"),
    ("DataTypeFP16", "662e", "0.099975586"),
    ("DataTypeFP16", "00c1", "-2.5"),
    ("DataTypeFP8E4M3", "3c", "1.5"),
    ("DataTypeFP8E4M3", "b8", "-1"),
    ("DataTypeFP8E5M2", "40", "2"),
    ("DataTypeFP8E5M2", "38", "0.5"),
    ("DataTypeBoolVector", "010001", "true,false,true"),
    ("DataTypeInt8Vector", "01fe03", "1,-2,3"),
    ("DataTypeInt32Vector", "010000000200000003000000", "1,2,3"),
    ("DataTypeUint8Vector", "010203", "1,2,3"),
    ("DataTypeFP16Vector", "003e0041", "1.5,2.5"),
    ("DataTypeFP32Vector", "0000c03f00002040cdcccc3d", "1.5,2.5,0.1"),
    ("DataTypeFP64Vector", "9a9999999999f13f9a99999999990140", "1.1,2.2"),
    ("DataTypeFP8E4M3Vector", "3c38", "1.5,1"),
    ("DataTypeFP8E5M2Vector", "4038", "2,0.5"),
    ("DataTypeString", "68656c6c6f20776f726c64", "hello world"),
]
# fix the -0.5 fixture (0xbf000000 LE = 000000bf)
FIXTURES[16] = ("DataTypeFP32", "000000bf", "-0.5")


def test_bytes_to_string_matches_go():
    for dtype, hexb, expected in FIXTURES:
        got = bytes_to_string(bytes.fromhex(hexb), dtype)
        assert got == expected, f"{dtype} {hexb}: got {got!r}, expected {expected!r}"


def test_go_shortest_float_formatting():
    # float64 %v shortest, incl. the %f<->%e cutover (exp<-4 or exp>=6)
    cases64 = {
        99999.0: "99999", 100000.0: "100000", 999999.0: "999999",
        1000000.0: "1e+06", 1234567.0: "1.234567e+06", 1e7: "1e+07",
        1e8: "1e+08", 1e20: "1e+20", 1e21: "1e+21", 1e-4: "0.0001",
        1e-5: "1e-05", 0.1: "0.1", 3.14159: "3.14159",
    }
    for v, exp in cases64.items():
        assert go_format_float(float(v), 64) == exp, (v, go_format_float(float(v), 64), exp)
