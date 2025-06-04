# DeserializedPSDB: Low-Level Design

## Overview
`DeserializedPSDB` is a structured representation of data decoded from a Permanent Storage Data Block (PSDB). It contains metadata extracted from the PSDB header along with the decompressed original data. This structure allows efficient retrieval of scalar and vector features of various types.

## Structure
```go
type DeserializedPSDB struct {
    ExpiryAt             uint64
    Header               []byte
    CompressedData       []byte
    OriginalData         []byte

    FeatureSchemaVersion uint16
    LayoutVersion        uint8
    CompressionType      compression.Type
    DataType             types.DataType

    NegativeCache        bool
    Expired              bool
}
```
- **Header**: Stores the original PSDB header (first 9 bytes)
- **OriginalData**: The uncompressed data block, ready for feature extraction
- **CompressedData**: Retained for reference or re-serialization if needed
- **NegativeCache**: Marks blocks with no data for a given FG ID
- **Expired**: Indicates whether the data is past its expiry timestamp

## Deserialization Flow

### `DeserializePSDB(data []byte)`
Entry point to deserialize a PSDB into `DeserializedPSDB`. It extracts the layout version and delegates to layout-specific handlers.

### Layout Version Handling
Currently supports:
- **LayoutVersion 1** → handled via `deserializePSDBForLayout1`

### Header Parsing
- Extracts layout version from byte 7 (upper 4 bits)
- Parses `FeatureSchemaVersion`, expiry timestamp, compression type, and data type
- Extracts PSDB header, compressed and uncompressed data accordingly

### Decompression
- Compression type is determined from the header
- If compression is active, uses appropriate decoder to get `OriginalData`
- If no compression, `OriginalData` is a direct slice from the input buffer

## Feature Access APIs
The struct provides methods for accessing individual or grouped features. These include:

### String Accessors
- `GetStringScalarFeature`
- `GetStringVectorFeature`

These handle the Pascal-style string encoding and vector string layout used during serialization.

### Numeric Accessors
- `GetNumericScalarFeature`
- `GetNumericVectorFeature`

Used for accessing data types like `int32`, `float32`, `uint8`, etc., using their declared size and position in the byte array.

### Boolean Accessors
- `GetBoolScalarFeature`
- `GetBoolVectorFeature`

Uses bitwise manipulation to extract single or multiple boolean values packed in bytes.

## Helper Methods for Typed Conversion
A large set of utility functions are provided to convert byte slices into Go native types. These include:
- Scalar helpers (e.g., `HelperScalarFeatureToTypeFloat32`, `HelperScalarFeatureToTypeInt64`)
- Vector helpers (e.g., `HelperVectorFeatureToTypeInt32`, `HelperVectorFeatureToTypeFloat8E5M2`)

These are grouped by type categories:
- **Floating Point**: Handles FP16, FP32, FP64, FP8E4M3, FP8E5M2
- **Integer**: Signed and unsigned variants from int8 to int64
- **Boolean**: Converts 1-byte or bit-packed data to bool
- **String**: Decodes Pascal-style encoded strings from feature bytes

These helpers are used only when typed data is required by downstream systems (e.g., ML model inputs, analytics tools).

## Efficiency Advantages
`DeserializedPSDB` is designed for performance and efficiency at every step:

- **Zero-Copy Header Access**: Operates directly on slices without reallocating memory.
- **Compact Layout**: Bit-packing (especially for booleans) and Pascal-style encoding reduce memory footprint.
- **Fast Feature Lookup**: Index-based and offset-based access patterns eliminate the need for scanning.

Together, these optimizations significantly reduce both latency and memory overhead when working with large-scale, high-throughput systems like online feature stores.

## Negative Caching Support
- `NegativeCacheDeserializePSDB()` creates a PSDB structure marked as negative cache.
- Used when a feature group ID has no associated data but is being tracked to prevent redundant lookups.

## Copy Semantics
- `Copy()` creates a deep copy of the `DeserializedPSDB` including header, compressed and original data slices.
- Ensures safe reuse or mutation across goroutines or layers.

## Summary
`DeserializedPSDB` is the central decoded representation for PSDB content. It offers a uniform interface to extract data regardless of compression or layout and is extensible to support future layout versions. Typed access is delegated to helper methods, ensuring clean separation between byte-level parsing and data interpretation.

Its memory-efficient design, combined with direct access patterns, enables high performance and scalability in ML and real-time inference pipelines.

