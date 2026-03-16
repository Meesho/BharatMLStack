# Low-Level Design (LLD): PSDB Layout V2 & V3
## Optimized Sparse Feature Storage Format

**Project:** BharatMLStack Online Feature Store
**Component:** Permanent Storage Data Block (PSDB) Serialization Layer
**Branch:** `poc/fs_layout`
**Status:** In Development (PR #266)
**Last Updated:** 2026-03-16

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Architecture Overview](#architecture-overview)
3. [System Context & Components](#system-context--components)
4. [Serialization Pipeline](#serialization-pipeline)
5. [Deserialization Pipeline](#deserialization-pipeline)
6. [Data Format Specification](#data-format-specification)
7. [Type System & Encoding](#type-system--encoding)
8. [Bitmap-Based Sparse Encoding](#bitmap-based-sparse-encoding)
9. [Algorithm Details](#algorithm-details)
10. [Error Handling & Edge Cases](#error-handling--edge-cases)
11. [Performance Characteristics](#performance-characteristics)
12. [Storage Integration](#storage-integration)
13. [Configuration & Metadata](#configuration--metadata)
14. [Testing Strategy](#testing-strategy)
15. [Migration & Backward Compatibility](#migration--backward-compatibility)
16. [Integration Points & APIs](#integration-points--apis)
17. [Usage Examples](#usage-examples)
18. [Known Issues & Limitations](#known-issues--limitations)
19. [Future Enhancements](#future-enhancements)
20. [References](#references)

---

## Executive Summary

The PSDB Layout V2 & V3 introduces bitmap-based sparse feature encoding to the online feature store's permanent storage layer. This design reduces storage footprint for datasets with missing or default-valued features by:

- **Bitmap Metadata:** Optional per-feature presence indicator
- **Sparse Payload:** Only non-default features are serialized
- **Compression Integration:** Works alongside existing compression algorithms
- **Type Safety:** Full support for all scalar and vector data types
- **Backward Compatibility:** Layout V1 support maintained through versioning

**Expected Benefits:**
- 20-60% storage reduction for sparse feature sets
- Minimal latency overhead for dense feature sets
- Transparent handling of missing features via defaults

---

## Architecture Overview

### 1. High-Level System Context

```
┌─────────────────────────────────────────────────────────────┐
│           Online Feature Store Application                   │
├─────────────────────────────────────────────────────────────┤
│                    Feature Request                           │
│                         │                                    │
│                         ▼                                    │
│  ┌──────────────────────────────────────┐                  │
│  │   Feature Retrieval Handler          │                  │
│  │   (online-feature-store/handler/)    │                  │
│  └────────────────┬─────────────────────┘                  │
│                   │                                         │
│  ┌────────────────▼─────────────────────┐                  │
│  │  PSDB Deserializer (Layout V1/V2)    │                  │
│  │  • Version Detection                 │                  │
│  │  • Decompression                     │                  │
│  │  • Bitmap-Aware Extraction           │                  │
│  │  • Type Conversion                   │                  │
│  └────────────────┬─────────────────────┘                  │
│                   │                                         │
│  ┌────────────────▼──────────────────────────────┐         │
│  │    Storage Repository Layer                    │         │
│  │  ┌──────────────┐  ┌────────────────────┐     │         │
│  │  │ Redis Store  │  │  Scylla Store      │     │         │
│  │  └──────────────┘  └────────────────────┘     │         │
│  └────────────────────────────────────────────────┘         │
│                   ▲                                         │
│  ┌────────────────┴─────────────────────┐                  │
│  │   Feature Persistence Handler        │                  │
│  │   (online-feature-store/handler/)    │                  │
│  └────────────────┬─────────────────────┘                  │
│                   │                                         │
│  ┌────────────────▼──────────────────────┐                 │
│  │  PSDB Serializer (Layout V1/V2/V3)    │                 │
│  │  • Bitmap Generation                 │                 │
│  │  • Layout Selection                  │                 │
│  │  • Type-Safe Encoding                │                 │
│  │  • Compression                       │                 │
│  └──────────────────────────────────────┘                 │
│                                                             │
│               (incoming feature data)                       │
│                         ▲                                   │
└─────────────────────────┼───────────────────────────────────┘
                          │
                  Feature Ingestion Point
```

### 2. Component Interaction Model

```
Serialization Flow:
───────────────────

Feature Data ([]float32, [][]bool, etc.)
         │
         ▼
┌─────────────────────────────────┐
│  1. Type Detection              │
│  (DataType enum)                │
└────────────┬────────────────────┘
             │
             ▼
┌─────────────────────────────────┐
│  2. Bitmap Generation           │
│  (Optional sparse encoding)     │
│  - Feature presence bits        │
│  - byteOrder = (n+7)/8          │
└────────────┬────────────────────┘
             │
             ▼
┌─────────────────────────────────┐
│  3. Header Construction         │
│  - Feature schema version       │
│  - Expiry timestamp             │
│  - Layout version (1/2/3)       │
│  - Compression type             │
│  - Data type encoding           │
│  - Bitmap presence flag (V2+)   │
│  - Bool last index (bool only)  │
└────────────┬────────────────────┘
             │
             ▼
┌─────────────────────────────────┐
│  4. Type-Specific Encoding      │
│  - Scalar: Direct bytes         │
│  - Vector: Packed format        │
│  - String: Pascal format (len+) │
│  - Bool: Bit-packed             │
└────────────┬────────────────────┘
             │
             ▼
┌─────────────────────────────────┐
│  5. Compression                 │
│  (Type, GZIP, etc.)            │
└────────────┬────────────────────┘
             │
             ▼
Serialized Bytes ([]byte)


Deserialization Flow:
─────────────────────

Serialized Bytes ([]byte)
         │
         ▼
┌─────────────────────────────────┐
│  1. Header Extraction           │
│  - Read bytes 0-9 (Layout V2)   │
│  - Detect version               │
│  - Extract metadata             │
└────────────┬────────────────────┘
             │
             ▼
┌─────────────────────────────────┐
│  2. Decompression               │
│  - Decompress payload           │
│  - Handle TypeNone (no-op)      │
└────────────┬────────────────────┘
             │
             ▼
┌─────────────────────────────────┐
│  3. Bitmap Extraction (if V2)   │
│  - Read bitmap section          │
│  - Calculate bitmap size        │
│  - Separate dense payload       │
└────────────┬────────────────────┘
             │
             ▼
┌─────────────────────────────────┐
│  4. Feature Extraction          │
│  - Query bitmap for presence    │
│  - Extract from dense/defaults  │
│  - Type-specific decoding       │
└────────────┬────────────────────┘
             │
             ▼
Feature Data ([]float32, [][]bool, etc.)
```

---

## System Context & Components

### Files Structure

```
online-feature-store/internal/data/blocks/
├── perm_storage_datablock_v2.go          # Layout V2 serialization
├── perm_storage_datablock_v3.go          # Layout V3 (future)
├── deserialized_psdb.go                  # Base V1 struct
├── deserialized_psdb_v2.go               # V1 deserialization
├── deserialized_psdb_layout2.go          # V2-specific deserialization
├── perm_storage_datablock_v2_test.go     # Serialization tests
├── deserialized_psdb_v2_test.go          # V2 deserialization tests
├── layout_comparison_test.go             # Benchmarking & comparison
├── layout_comparison_results.txt         # Test results snapshot
├── cache_storage_datablock_v2.go         # Memory cache variant
├── cache_storage_datablock_v2_test.go    # Cache tests
└── psdb_shadow_compare.go                # Validation framework

go-sdk/pkg/datatypeconverter/byteorder/
├── system.go                             # Type encoding/decoding
└── types.go                              # DataType definitions
```

### Key Entities

| Component | Responsibility | Key Files |
|-----------|-----------------|-----------|
| **PermStorageDataBlock** | Serialization builder & executor | perm_storage_datablock_v2.go |
| **DeserializedPSDB** | Layout V1 deserialization | deserialized_psdb_v2.go |
| **DeserializedPSDBLayout2** | Layout V2 deserialization with bitmap | deserialized_psdb_layout2.go |
| **ByteOrder** | Type-aware encoding/decoding | byteorder/system.go |
| **DataType** | Type enumeration & metadata | types.go |
| **PSDBBlock Interface** | Abstract serialization interface | Implemented by both structures |

---

## Serialization Pipeline

### 1. Overview

The serialization pipeline transforms in-memory feature data into a compact byte format optimized for sparse data.

### 2. Entry Point

```go
// From persist.go - Feature persistence flow
func (h *Handler) persistFeatures(ctx context.Context, features map[string]interface{}) error {
    // 1. Create PSDB builder
    builder := NewPermStorageDataBlockBuilder()

    // 2. Set data and metadata
    builder.SetData(features[dataType])
    builder.SetNoOfFeatures(len(features))
    builder.SetDataType(dataType)

    // 3. Optional: Set bitmap for sparse encoding
    if shouldEnableSparseMode(features) {
        builder.SetBitmap(generateBitmap(features))
    }

    // 4. Serialize
    psdb := builder.Build()
    serialized, err := psdb.Serialize()

    // 5. Persist to storage
    return h.store.Set(featureKey, serialized)
}
```

### 3. Serialization Steps (Detail)

[This section would contain detailed breakdown of each serialization step with code references and examples]

---

## Deserialization Pipeline

[Detailed deserialization steps and logic]

---

## Data Format Specification

### 1. Header Layout (10 bytes for Layout V2)

```
Byte 0-1:     Feature Schema Version (uint16)
              [15:0] = Schema Version

Byte 2-6:     Expiry Timestamp (40 bits)
              Encoded as 5 bytes in system byte order
              Can be decoded as uint64 with EncodeExpiry()

Byte 7:       Layout & Compression & DataType Bits
              [7:4] = Layout Version (4 bits) → 0/1/2/3
              [3:1] = Compression Type (3 bits)
              [0:0] = DataType High Bit (1 bit)

Byte 8:       DataType & Bool Index
              [7:4] = DataType Low 4 Bits (4 bits)
              [3:0] = Bool DType Last Index (4 bits, bool only)

Byte 9:       Layout V2 Metadata (Bitmap Presence)
              [0:0] = Bitmap Present Flag (1 bit)
              [7:1] = Reserved (7 bits)

Total:        10 bytes for Layout V2 (9 bytes + byte 9)
```

### 2. Payload Layout (Layout V2)

```
Layout V2 Payload Structure:
────────────────────────────

If BitmapMeta & 0x01 == 1:
    ┌──────────────────────────────────┐
    │ Bitmap Section                   │
    │ Size: (numFeatures + 7) / 8      │
    │ Bit i = 1 if feature i present   │
    └──────────────────────────────────┘
    ┌──────────────────────────────────┐
    │ Dense Payload                    │
    │ Only non-default features        │
    │ Order: ascending by feature idx  │
    └──────────────────────────────────┘

Else (no bitmap):
    ┌──────────────────────────────────┐
    │ All Features (dense)             │
    │ Layout V1 style                  │
    └──────────────────────────────────┘
```

### 3. Data Type Encoding Table

| DataType | Bytes | Layout | Notes |
|----------|-------|--------|-------|
| FP32 | 4 | Scalar, Vector | IEEE 754 single precision |
| FP64 | 8 | Scalar, Vector | IEEE 754 double precision |
| Int32 | 4 | Scalar, Vector | Two's complement |
| Int64 | 8 | Scalar, Vector | Two's complement |
| Bool | 1 bit | Scalar, Vector | Bit-packed in bytes |
| String | Variable | Scalar, Vector | Pascal format (2-byte len + data) |

---

## Type System & Encoding

[Detailed type encoding specifications]

---

## Bitmap-Based Sparse Encoding

[Complete bitmap algorithm documentation]

---

## Algorithm Details

[Pseudocode and detailed algorithm implementations]

---

## Error Handling & Edge Cases

[Error scenarios and handling strategies]

---

## Performance Characteristics

[Performance metrics and analysis]

---

## Storage Integration

[Storage layer integration details]

---

## Configuration & Metadata

[Configuration system documentation]

---

## Testing Strategy

[Test coverage and validation approach]

---

## Migration & Backward Compatibility

[Version compatibility and migration path]

---

## Integration Points & APIs

[Public API surface and integration contracts]

---

## Usage Examples

[Code examples for common operations]

---

## Known Issues & Limitations

[Current limitations and open issues from PR #266]

---

## Future Enhancements

[Proposed improvements and next steps]

---

## References

- PR #266: PSDB Layout V2 Implementation
- Branch: `poc/fs_layout`
- Related Issues: Feature sparse encoding optimization
- Design Document: (link to full LLD)

---

**Document Version:** 1.0
**Last Updated:** 2026-03-16
**Author:** Generated from branch analysis
**Status:** Draft - In Development
