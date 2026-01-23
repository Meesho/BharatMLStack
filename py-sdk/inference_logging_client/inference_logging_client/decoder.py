"""Core decoding utilities and byte readers."""

import struct
import json
from typing import Any

from .utils import (
    normalize_type, is_sized_type, get_scalar_size,
    format_float, SCALAR_TYPE_SIZES, SIZED_TYPES
)


class ByteReader:
    """Helper class to read bytes sequentially."""
    
    def __init__(self, data: bytes):
        self.data = data
        self.pos = 0
    
    def read(self, n: int) -> bytes:
        if self.pos + n > len(self.data):
            raise ValueError(f"Not enough bytes: need {n}, have {len(self.data) - self.pos}")
        result = self.data[self.pos:self.pos + n]
        self.pos += n
        return result
    
    def read_uint8(self) -> int:
        return self.read(1)[0]
    
    def read_int8(self) -> int:
        return struct.unpack('<b', self.read(1))[0]
    
    def read_uint16(self) -> int:
        return struct.unpack('<H', self.read(2))[0]
    
    def read_int16(self) -> int:
        return struct.unpack('<h', self.read(2))[0]
    
    def read_uint32(self) -> int:
        return struct.unpack('<I', self.read(4))[0]
    
    def read_int32(self) -> int:
        return struct.unpack('<i', self.read(4))[0]
    
    def read_uint64(self) -> int:
        return struct.unpack('<Q', self.read(8))[0]
    
    def read_int64(self) -> int:
        return struct.unpack('<q', self.read(8))[0]
    
    def read_float16(self) -> float:
        """Read IEEE 754 half-precision (FP16) float."""
        bits = self.read_uint16()
        
        # Extract IEEE 754 FP16 components
        sign = (bits >> 15) & 0x1
        exponent = (bits >> 10) & 0x1F
        mantissa = bits & 0x3FF
        
        if exponent == 0:
            if mantissa == 0:
                return -0.0 if sign else 0.0
            # Subnormal
            return ((-1) ** sign) * (2 ** -14) * (mantissa / 1024.0)
        elif exponent == 31:
            if mantissa == 0:
                return float('-inf') if sign else float('inf')
            return float('nan')
        else:
            return ((-1) ** sign) * (2 ** (exponent - 15)) * (1.0 + mantissa / 1024.0)
    
    def read_float32(self) -> float:
        return struct.unpack('<f', self.read(4))[0]
    
    def read_float64(self) -> float:
        return struct.unpack('<d', self.read(8))[0]
    
    def remaining(self) -> int:
        return len(self.data) - self.pos
    
    def has_more(self) -> bool:
        return self.pos < len(self.data)


def read_varint(reader: ByteReader) -> int:
    """Read a protobuf varint."""
    result = 0
    shift = 0
    while True:
        byte = reader.read_uint8()
        result |= (byte & 0x7f) << shift
        if not (byte & 0x80):
            break
        shift += 7
    return result


def skip_field(reader: ByteReader, wire_type: int):
    """Skip a protobuf field based on wire type."""
    if wire_type == 0:  # varint
        read_varint(reader)
    elif wire_type == 1:  # 64-bit
        reader.read(8)
    elif wire_type == 2:  # length-delimited
        length = read_varint(reader)
        reader.read(length)
    elif wire_type == 5:  # 32-bit
        reader.read(4)
    else:
        raise ValueError(f"Unknown wire type: {wire_type}")


def decode_ieee754_fp16(value_bytes: bytes) -> float:
    """
    Decode IEEE 754 half-precision (FP16) to float.
    
    IEEE 754 FP16 format:
    - 1 bit sign
    - 5 bits exponent (bias 15)
    - 10 bits mantissa
    
    This is the format used by feature stores for vector data.
    """
    if len(value_bytes) != 2:
        return 0.0
    bits = struct.unpack('<H', value_bytes)[0]
    
    # Extract components
    sign = (bits >> 15) & 0x1
    exponent = (bits >> 10) & 0x1F
    mantissa = bits & 0x3FF
    
    if exponent == 0:
        # Subnormal or zero
        if mantissa == 0:
            return -0.0 if sign else 0.0
        # Subnormal number
        result = ((-1) ** sign) * (2 ** -14) * (mantissa / 1024.0)
        return format_float(result)
    elif exponent == 31:
        # Infinity or NaN
        if mantissa == 0:
            return float('-inf') if sign else float('inf')
        return float('nan')
    else:
        # Normal number
        result = ((-1) ** sign) * (2 ** (exponent - 15)) * (1.0 + mantissa / 1024.0)
        return format_float(result)


def decode_scalar_value(value_bytes: bytes, feature_type: str) -> Any:
    """Decode a scalar value from bytes based on feature type."""
    normalized = normalize_type(feature_type)
    
    if len(value_bytes) == 0:
        return None
    
    try:
        if normalized in {"INT8", "I8"}:
            return struct.unpack('<b', value_bytes)[0]
        elif normalized in {"INT16", "I16", "SHORT"}:
            return struct.unpack('<h', value_bytes)[0]
        elif normalized in {"INT32", "I32", "INT"}:
            return struct.unpack('<i', value_bytes)[0]
        elif normalized in {"INT64", "I64", "LONG"}:
            return struct.unpack('<q', value_bytes)[0]
        elif normalized in {"UINT8", "U8"}:
            return value_bytes[0]
        elif normalized in {"UINT16", "U16"}:
            return struct.unpack('<H', value_bytes)[0]
        elif normalized in {"UINT32", "U32"}:
            return struct.unpack('<I', value_bytes)[0]
        elif normalized in {"UINT64", "U64"}:
            return struct.unpack('<Q', value_bytes)[0]
        elif normalized in {"FP8E5M2", "FP8E4M3"}:
            return value_bytes[0]  # Return raw byte
        elif normalized in {"FP16", "FLOAT16", "F16"}:
            # IEEE 754 half-precision (FP16)
            if len(value_bytes) == 2:
                return decode_ieee754_fp16(value_bytes)
            return None
        elif normalized in {"FP32", "FLOAT32", "F32", "FLOAT"}:
            result = struct.unpack('<f', value_bytes)[0]
            return format_float(result)
        elif normalized in {"FP64", "FLOAT64", "F64", "DOUBLE"}:
            result = struct.unpack('<d', value_bytes)[0]
            return format_float(result)
        elif normalized in {"BOOL", "BOOLEAN"}:
            return value_bytes[0] != 0
        else:
            return value_bytes.hex()  # Unknown type, return hex
    except struct.error:
        return None


def decode_binary_vector(value_bytes: bytes, feature_type: str) -> list:
    """
    Decode a binary-encoded vector based on element type.
    
    Binary vectors are packed element bytes in sequence.
    """
    normalized = normalize_type(feature_type)
    
    if len(value_bytes) == 0:
        return []
    
    result = []
    
    # Determine element type and size
    if "FP16" in normalized or "FLOAT16" in normalized:
        # FP16 vector: 2 bytes per element, IEEE 754 half-precision format
        elem_size = 2
        for i in range(0, len(value_bytes), elem_size):
            if i + elem_size <= len(value_bytes):
                elem_bytes = value_bytes[i:i + elem_size]
                result.append(decode_ieee754_fp16(elem_bytes))
    
    elif "FP32" in normalized or "FLOAT32" in normalized:
        # FP32 vector: 4 bytes per element
        elem_size = 4
        for i in range(0, len(value_bytes), elem_size):
            if i + elem_size <= len(value_bytes):
                result.append(format_float(struct.unpack('<f', value_bytes[i:i + elem_size])[0]))
    
    elif "FP64" in normalized or "FLOAT64" in normalized:
        # FP64 vector: 8 bytes per element
        elem_size = 8
        for i in range(0, len(value_bytes), elem_size):
            if i + elem_size <= len(value_bytes):
                result.append(format_float(struct.unpack('<d', value_bytes[i:i + elem_size])[0]))
    
    elif "INT8" in normalized and "UINT" not in normalized:
        # INT8 vector: 1 byte per element, signed
        for b in value_bytes:
            result.append(struct.unpack('<b', bytes([b]))[0])
    
    elif "INT16" in normalized and "UINT" not in normalized:
        # INT16 vector: 2 bytes per element, signed
        elem_size = 2
        for i in range(0, len(value_bytes), elem_size):
            if i + elem_size <= len(value_bytes):
                result.append(struct.unpack('<h', value_bytes[i:i + elem_size])[0])
    
    elif "INT32" in normalized and "UINT" not in normalized:
        # INT32 vector: 4 bytes per element, signed
        elem_size = 4
        for i in range(0, len(value_bytes), elem_size):
            if i + elem_size <= len(value_bytes):
                result.append(struct.unpack('<i', value_bytes[i:i + elem_size])[0])
    
    elif "INT64" in normalized and "UINT" not in normalized:
        # INT64 vector: 8 bytes per element, signed
        elem_size = 8
        for i in range(0, len(value_bytes), elem_size):
            if i + elem_size <= len(value_bytes):
                result.append(struct.unpack('<q', value_bytes[i:i + elem_size])[0])
    
    elif "UINT8" in normalized:
        # UINT8 vector: 1 byte per element, unsigned
        result = list(value_bytes)
    
    elif "UINT16" in normalized:
        # UINT16 vector: 2 bytes per element, unsigned
        elem_size = 2
        for i in range(0, len(value_bytes), elem_size):
            if i + elem_size <= len(value_bytes):
                result.append(struct.unpack('<H', value_bytes[i:i + elem_size])[0])
    
    elif "UINT32" in normalized:
        # UINT32 vector: 4 bytes per element, unsigned
        elem_size = 4
        for i in range(0, len(value_bytes), elem_size):
            if i + elem_size <= len(value_bytes):
                result.append(struct.unpack('<I', value_bytes[i:i + elem_size])[0])
    
    elif "UINT64" in normalized:
        # UINT64 vector: 8 bytes per element, unsigned
        elem_size = 8
        for i in range(0, len(value_bytes), elem_size):
            if i + elem_size <= len(value_bytes):
                result.append(struct.unpack('<Q', value_bytes[i:i + elem_size])[0])
    
    elif "BOOL" in normalized:
        # Bool vector: 1 byte per element
        result = [b != 0 for b in value_bytes]
    
    elif "FP8" in normalized:
        # FP8 vector: 1 byte per element (return raw bytes, no standard decoding)
        result = list(value_bytes)
    
    else:
        # Unknown vector type, return empty
        return []
    
    return result


def is_likely_json(value_bytes: bytes) -> bool:
    """Check if bytes look like JSON (starts with [ or {)."""
    if len(value_bytes) == 0:
        return False
    first_byte = value_bytes[0]
    # ASCII [ = 91, { = 123
    return first_byte in (91, 123)


def decode_vector_or_string(value_bytes: bytes, feature_type: str) -> Any:
    """Decode a vector or string value from bytes."""
    normalized = normalize_type(feature_type)
    
    if len(value_bytes) == 0:
        return None
    
    if normalized in {"STRING", "STR"}:
        try:
            return value_bytes.decode('utf-8')
        except UnicodeDecodeError:
            return value_bytes.hex()
    
    if normalized in {"BYTES"}:
        # BYTES type: first 2 bytes are the length prefix, remaining bytes are the string content
        if len(value_bytes) < 2:
            return None
        # Read 2-byte little-endian length prefix
        length = struct.unpack('<H', value_bytes[:2])[0]
        string_bytes = value_bytes[2:]
        # Use the length to extract the string (or all remaining bytes if length exceeds available)
        actual_bytes = string_bytes[:length] if length <= len(string_bytes) else string_bytes
        try:
            return actual_bytes.decode('utf-8')
        except UnicodeDecodeError:
            # Fallback to hex if not valid UTF-8
            return actual_bytes.hex()
    
    # Check if this is a vector type
    is_vector = "VECTOR" in normalized
    
    if is_vector:
        # For vectors, check if it's JSON or binary encoded
        # JSON vectors start with '[' (0x5b)
        if is_likely_json(value_bytes):
            try:
                return json.loads(value_bytes.decode('utf-8'))
            except (json.JSONDecodeError, UnicodeDecodeError):
                pass
        
        # Try binary vector decoding
        decoded = decode_binary_vector(value_bytes, feature_type)
        if decoded:
            return decoded
        
        # Fallback to hex if binary decoding returned empty
        return value_bytes.hex()
    
    # For non-vector sized types, try JSON decode first
    if is_likely_json(value_bytes):
        try:
            return json.loads(value_bytes.decode('utf-8'))
        except (json.JSONDecodeError, UnicodeDecodeError):
            pass
    
    # Return hex representation for unknown binary data
    return value_bytes.hex()


def decode_feature_value(value_bytes: bytes, feature_type: str) -> Any:
    """Decode a feature value based on its type."""
    if value_bytes is None or len(value_bytes) == 0:
        return None
    
    if is_sized_type(feature_type):
        return decode_vector_or_string(value_bytes, feature_type)
    
    return decode_scalar_value(value_bytes, feature_type)
