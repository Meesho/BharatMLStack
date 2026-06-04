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


COVERAGE_FIXTURES = [
    ('DataTypeFP16', '003e', '1.5'),
    ('DataTypeFP16', '662e', '0.099975586'),
    ('DataTypeFP16', '00c1', '-2.5'),
    ('DataTypeFP16', '0000', '0'),
    ('DataTypeFP16', 'ef7b', '64992'),
    ('DataTypeFP32', '17b7d1b8', '-0.0001'),
    ('DataTypeFP32', '6420f147', '123456.78'),
    ('DataTypeFP64', '48afbc9af2d77a3e', '1e-07'),
    ('DataTypeBoolVector', '01000101', 'true,false,true,true'),
    ('DataTypeInt8Vector', '80007f', '-128,0,127'),
    ('DataTypeInt16Vector', '00800000ff7f', '-32768,0,32767'),
    ('DataTypeInt32Vector', 'ffffffff0000000040e20100', '-1,0,123456'),
    ('DataTypeInt64Vector', '0000000000000080ffffffffffffff7f', '-9223372036854775808,9223372036854775807'),
    ('DataTypeUint8Vector', '0001ff', '0,1,255'),
    ('DataTypeUint16Vector', '0000ffff', '0,65535'),
    ('DataTypeUint32Vector', '00000000ffffffff', '0,4294967295'),
    ('DataTypeUint64Vector', '0000000000000000ffffffffffffffff', '0,18446744073709551615'),
    ('DataTypeFP16Vector', '003e00c1662e0000', '1.5,-2.5,0.099975586,0'),
    ('DataTypeFP32Vector', '0000c03f000000bf20bcbe4c95bfd63300000000', '1.5,-0.5,1e+08,1e-07,0'),
    ('DataTypeFP64Vector', '9a9999999999f13f9a999999999901c048afbc9af2d77a3e', '1.1,-2.2,1e-07'),
    ('DataTypeFP8E4M3Vector', '3cb830', '1.5,-1,0.5'),
    ('DataTypeFP8E5M2Vector', '4038c4', '2,0.5,-4'),
]


def test_full_coverage_all_vector_and_fp16_types():
    """Every vector element type + FS FP16 scalars, byte-path go-core binary."""
    for dtype, hexb, expected in COVERAGE_FIXTURES:
        got = bytes_to_string(bytes.fromhex(hexb), dtype)
        assert got == expected, f"{dtype} {hexb}: got {got!r}, expected {expected!r}"


EDGE_FIXTURES = [
    ('DataTypeFP16', '0100', '5.9604645e-08'),
    ('DataTypeFP16', '0200', '1.1920929e-07'),
    ('DataTypeFP16', '0300', '1.7881393e-07'),
    ('DataTypeFP16', '007c', '+Inf'),
    ('DataTypeFP16', '017c', 'NaN'),
    ('DataTypeFP16', '0080', '-0'),
    ('DataTypeFP16', '00fc', '-Inf'),
    ('DataTypeFP8E5M2', '01', '1.5258789e-05'),
    ('DataTypeFP8E5M2', '02', '3.0517578e-05'),
    ('DataTypeFP8E5M2', '03', '4.5776367e-05'),
    ('DataTypeFP8E5M2', '7c', '+Inf'),
    ('DataTypeFP8E5M2', '7d', 'NaN'),
    ('DataTypeFP8E4M3', '7f', 'NaN'),
    ('DataTypeFP8E4M3', '80', '-0'),
    ('DataTypeFP8E5M2', '80', '-0'),
    ('DataTypeFP8E5M2', 'fc', '-Inf'),
    ('DataTypeFP32', '1e429e18', '4.0908804e-24'),
    ('DataTypeFP32', '2215aaee', '-2.6319e+28'),
    ('DataTypeFP32', '06a2d64b', '2.8132364e+07'),
    ('DataTypeFP32', '5da1e0ff', 'NaN'),
    ('DataTypeFP64', '8639235afdf3c731', '6.941166305652727e-69'),
    ('DataTypeFP64', '96893dd4ed57b784', '-6.132105145909716e-286'),
    ('DataTypeFP64', 'c5605185d954149a', '-4.7848780304174145e-183'),
    ('DataTypeFP64', 'e458e08a901ff5ff', 'NaN'),
    ('DataTypeFP16Vector', '261b9393091613867cc0', '0.003490448,-0.00092458725,0.0014734268,-9.268522e-05,-2.2421875'),
    ('DataTypeFP16Vector', '8c113a3e4b81b6c0f143', '0.00067710876,1.5566406,-1.9729137e-05,-2.3554688,3.9707031'),
    ('DataTypeFP16Vector', '668104426a6c9496', '-2.1338463e-05,3.0078125,4520,-0.0016059875'),
    ('DataTypeFP16Vector', '907d', 'NaN'),
    ('DataTypeFP32Vector', '0b9dd7eccb803dcc', '-2.0852853e+27,-4.96771e+07'),
    ('DataTypeFP32Vector', '28517e63a473a6fd9a9701f8', '4.691321e+21,-2.7656536e+37,-1.0513768e+34'),
    ('DataTypeFP32Vector', 'dabfbd8a', '-1.8272205e-32'),
    ('DataTypeFP32Vector', '1a03857f', 'NaN'),
    ('DataTypeFP64Vector', '5b9dc8b8e0c78a44824a97c4420ebd28b84c9ae9f4a93d4f57174dbe88460eaf87b370da561bb8f9', '1.5808577942289669e+22,1.8877873312250154e-112,5.24115628394067e+73,-4.9870401132054686e-82,-2.1366602376534176e+278'),
    ('DataTypeFP64Vector', 'be0f3a63644fb57f4d9b06fb00e3575b5f6a12edcbadf202', '1.4964479059459674e+307,1.0596803624902316e+132,1.827912411121787e-294'),
    ('DataTypeFP64Vector', '84327d49cf43fde8', '-5.468949910989888e+197'),
    ('DataTypeFP64Vector', '11521d29d9acfd7f', 'NaN'),
    ('DataTypeFP8E4M3Vector', '80', '-0'),
    ('DataTypeFP8E4M3Vector', 'ff', 'NaN'),
    ('DataTypeFP8E5M2Vector', '82ecfffb58a6', '-3.0517578e-05,-4096,NaN,-57344,128,-0.0234375'),
    ('DataTypeFP8E5M2Vector', '82974964ff', '-3.0517578e-05,-0.0017089844,10,1024,NaN'),
    ('DataTypeFP8E5M2Vector', '03', '4.5776367e-05'),
    ('DataTypeFP8E5M2Vector', '80', '-0'),
    ('DataTypeFP8E5M2Vector', '7c', '+Inf'),
    ('DataTypeFP8E5M2Vector', 'ff', 'NaN'),
    ('DataTypeFP8E5M2Vector', 'fc', '-Inf'),
    ('DataTypeFP8E5M2', '7c', '+Inf'),
    ('DataTypeFP8E5M2', '7d', 'NaN'),
    ('DataTypeFP8E5M2', '7e', 'NaN'),
    ('DataTypeFP8E4M3', '7f', 'NaN'),
    ('DataTypeFP8E5M2', '7f', 'NaN'),
    ('DataTypeFP8E4M3', '80', '-0'),
    ('DataTypeFP8E5M2', '80', '-0'),
    ('DataTypeFP8E5M2', 'fc', '-Inf'),
    ('DataTypeFP8E5M2', 'fd', 'NaN'),
    ('DataTypeFP8E5M2', 'fe', 'NaN'),
    ('DataTypeFP8E4M3', 'ff', 'NaN'),
    ('DataTypeFP8E5M2', 'ff', 'NaN'),
]


def test_float_edge_cases_match_go():
    """NaN, +/-Inf, negative zero, subnormals, scientific boundaries -- byte-for-byte."""
    for dtype, hexb, expected in EDGE_FIXTURES:
        got = bytes_to_string(bytes.fromhex(hexb), dtype)
        assert got == expected, f"{dtype} {hexb}: got {got!r}, expected {expected!r}"
