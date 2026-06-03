"""Faithful Python port of go-core datatypeconverter.BytesToString.

Go<->Python parity: validated byte-for-byte against go-core BytesToString for
every scalar/vector type incl. Go shortest-%g float formatting, IEEE-754 FP16,
and FP8 E4M3/E5M2.

Mirrors github.com/Meesho/go-core/datatypeconverter/typeconverter so the
Python decoder produces byte-for-byte identical strings to Go for every type.
"""
import math
import struct


def _f32_bits(f):
    return struct.unpack("<I", struct.pack("<f", f))[0]


def _shortest(f, bits):
    """Shortest decimal digits round-tripping to the float at `bits` precision.
    Returns (neg, digits, dp): value = digits * 10^(dp-len(digits))."""
    neg = math.copysign(1.0, f) < 0
    a = abs(f)
    if a == 0.0:
        return neg, "0", 1
    if bits == 32:
        target = _f32_bits(struct.unpack("<f", struct.pack("<f", a))[0])
        def rt(s):
            try:
                return _f32_bits(float(s)) == target
            except (ValueError, OverflowError):
                return False
    else:
        def rt(s):
            return float(s) == a
    s = None
    for p in range(1, 18):
        cand = f"{a:.{p - 1}e}"
        if rt(cand):
            s = cand
            break
    if s is None:
        s = f"{a:.17e}"
    mant, exp = s.split("e")
    exp = int(exp)
    mant = mant.replace(".", "")
    mant = mant.rstrip("0") or "0"
    return neg, mant, exp + 1  # dp = leading-digit power + 1


def go_format_float(f, bits):
    """Replicate Go strconv FormatFloat(f,'g',-1,bits) as used by fmt.Sprint."""
    if math.isnan(f):
        return "NaN"
    if math.isinf(f):
        return "+Inf" if f > 0 else "-Inf"
    neg, digits, dp = _shortest(f, bits)
    if digits == "0":
        return "0"
    sign = "-" if neg else ""
    exp = dp - 1
    # Go shortest-'g' (fmt.Sprint): exponential when exp < -4 or exp >= 6
    # (ftoa eprec=6 for shortest). Validated against go-core BytesToString.
    if exp < -4 or exp >= 6:
        return _fmt_e(sign, digits, exp)
    return _fmt_f(sign, digits, dp)


def _fmt_e(sign, digits, exp):
    mant = digits[0]
    if len(digits) > 1:
        mant += "." + digits[1:]
    return f"{sign}{mant}e{exp:+03d}"


def _fmt_f(sign, digits, dp):
    if dp <= 0:
        body = "0." + ("0" * (-dp)) + digits
    elif dp >= len(digits):
        body = digits + ("0" * (dp - len(digits)))
    else:
        body = digits[:dp] + "." + digits[dp:]
    return sign + body


# ---- scalar bytes -> value/string (mirrors go-core per-type functions) ----
def _u32_to_f32(bits):
    return struct.unpack("<f", struct.pack("<I", bits & 0xFFFFFFFF))[0]


def fp16_as_fp32(b):  # IEEE-754 half -> float (go-core Float16AsFP32)
    return struct.unpack("<e", b)[0]


def fp8_e4m3_as_fp32(byte):
    if (byte & 0x7F) == 0x7F:
        return float("nan")
    w = (byte << 24) & 0xFFFFFFFF
    sign = w & 0x80000000
    non_sign = w & 0x7FFFFFFF
    if non_sign == 0:
        return _u32_to_f32(sign)
    renorm = 32 - non_sign.bit_length()
    renorm = renorm - 4 if renorm > 4 else 0
    res = sign | ((((non_sign << renorm) & 0xFFFFFFFF) >> 4) + (((0x78 - renorm) << 23) & 0xFFFFFFFF))
    return _u32_to_f32(res)


def fp8_e5m2_as_fp32(byte):
    sign = (byte >> 7) & 0x1
    exponent = (byte >> 2) & 0x1F
    mantissa = byte & 0x3
    if exponent == 0x1F:
        if mantissa == 0:
            return float("-inf") if sign else float("inf")
        return float("nan")
    if exponent == 0:
        v = (mantissa / 4.0) * (2.0 ** -14)
    else:
        v = (1.0 + mantissa / 4.0) * (2.0 ** (exponent - 15))
    return -v if sign else v


_SCALAR = {
    "bool": (1, lambda b: "true" if b[0] != 0 else "false"),
    "int8": (1, lambda b: str(struct.unpack("<b", b)[0])),
    "int16": (2, lambda b: str(struct.unpack("<h", b)[0])),
    "int32": (4, lambda b: str(struct.unpack("<i", b)[0])),
    "int64": (8, lambda b: str(struct.unpack("<q", b)[0])),
    "uint8": (1, lambda b: str(b[0])),
    "uint16": (2, lambda b: str(struct.unpack("<H", b)[0])),
    "uint32": (4, lambda b: str(struct.unpack("<I", b)[0])),
    "uint64": (8, lambda b: str(struct.unpack("<Q", b)[0])),
    "fp32": (4, lambda b: go_format_float(struct.unpack("<f", b)[0], 32)),
    "fp64": (8, lambda b: go_format_float(struct.unpack("<d", b)[0], 64)),
    "fp16": (2, lambda b: go_format_float(fp16_as_fp32(b), 32)),
    "fp8e4m3": (1, lambda b: go_format_float(fp8_e4m3_as_fp32(b[0]), 32)),
    "fp8e5m2": (1, lambda b: go_format_float(fp8_e5m2_as_fp32(b[0]), 32)),
}


def _norm(dt):
    return dt.lower().replace("datatype", "")


def bytes_to_string(data, dtype):
    """Port of go-core BytesToString: bytes -> canonical comma/scalar string."""
    n = _norm(dtype)
    if n in ("string", "bytes", "stringvector"):
        return data.decode("utf-8", "replace")
    if n in _SCALAR:
        size, fn = _SCALAR[n]
        return fn(data)
    if n.endswith("vector"):
        base = n[:-6]
        if base in _SCALAR:
            size, fn = _SCALAR[base]
            return ",".join(fn(data[i:i + size]) for i in range(0, len(data), size))
    raise ValueError("unsupported data type: " + dtype)
