# LLD (Low-Level Design) Documentation Generation Prompt

## Overview
Generate comprehensive Low-Level Design documentation for the `poc/fs_layout` branch which implements **PSDB Layout Version 2** - an optimized data serialization and storage format for sparse feature data in the online feature store.

---

## Branch Analysis Summary
- **Branch:** `poc/fs_layout`
- **Total Commits:** 14 commits from main
- **Files Changed:** 42 files
- **Total Changes:** 5,187 insertions(+), 1,177 deletions(-)
- **Key Focus:** PSDB (Permanent Storage Data Block) layout optimization with bitmap-based sparse encoding

---

## Key Changes by Component

### 1. Core Serialization Layer (`perm_storage_datablock_v2.go`, `perm_storage_datablock_v3.go`)
- Bitmap-based sparse feature encoding (Layout V2)
- Extended support for all data types (Bool, numeric, vectors, strings)
- Compression integration with layout-aware encoding
- **Lines of Code:** ~745 new lines in v3

### 2. Deserialization Layer (`deserialized_psdb_layout2.go`, `deserialized_psdb_v2.go`)
- Layout V1 and Layout V2 dual support
- Bitmap-aware sparse feature extraction
- Bounds checking and error handling for corrupted data
- **Key Files:** Layout-specific implementations

### 3. Data Type System (`byteorder/system.go`)
- Support for FP8, FP16, FP32, FP64, Int8-64, Uint8-64, Bool, String vectors
- Type-aware default value handling
- **Lines Changed:** ~491 modifications

### 4. Storage Integration
- Redis and Scylla data repository updates
- Shadow comparison for validation (new)
- PSDB builder modifications

### 5. Configuration Management
- Slate-aware config generation
- Feature schema versioning
- Layout metadata encoding

### 6. Testing & Validation
- Layout comparison test suite
- Performance benchmarking
- Test data restructuring for feature count alignment

---

## LLD Documentation Template

### Document Structure
```
1. Architecture Overview
   - System Context Diagram
   - Component Interaction Model
   - Data Flow Patterns

2. Component-Level Design
   - Serialization Pipeline
   - Deserialization Pipeline
   - Type System
   - Storage Layer

3. Data Format Specification
   - Header Structure (Bytes 0-9)
   - Bitmap Encoding (Optional)
   - Dense Payload Format
   - Compression Strategy

4. Algorithm & Implementation Details
   - Bitmap Index Calculation
   - Feature Extraction Logic
   - Sparse vs Dense Tradeoffs
   - Default Value Handling

5. Error Handling & Edge Cases
   - Bounds Checking
   - Corrupted Data Detection
   - Version Compatibility
   - Type Mismatch Scenarios

6. Performance Characteristics
   - Space Optimization Metrics
   - Serialization/Deserialization Latency
   - Compression Ratios
   - Bitmap Overhead Analysis

7. Integration Points
   - Config Management Integration
   - Storage Repository Interface
   - Feature Retrieval Pipeline
   - Persistence Layer

8. Testing Strategy
   - Unit Test Coverage
   - Integration Test Scenarios
   - Benchmark Test Results
   - Shadow Comparison Validation

9. Migration & Backward Compatibility
   - Layout V1 Support
   - Version Detection Logic
   - Upgrade Path

10. Code Examples & Usage Patterns
    - Serialization Example
    - Deserialization Example
    - Bitmap Query Pattern
    - Type Conversion Example
```

---

## Detailed Prompt Instructions

### Section 1: Architecture Overview
**Analyze commits:** `a14f1df4`, `468d003f`, `a415a4da`

Generate:
- System context diagram showing how PSDB Layout V2/V3 fits in the online feature store
- Component interaction diagram (serializer → bitmap → compressor → storage)
- Data flow: Feature ingestion → Serialization → Storage → Retrieval → Deserialization
- Highlight sparse feature optimization strategy

### Section 2: Serialization Pipeline
**Analyze commits:** `8e364d05`, `468d003f`
**Key Files:** `perm_storage_datablock_v2.go`, `perm_storage_datablock_v3.go`

Generate:
- Step-by-step serialization process
- Data type-specific handling (scalars, vectors, bools, strings)
- Bitmap generation logic for sparse features
- Compression integration points
- Header layout (9-10 bytes with metadata bits)
- Payload encoding strategies

### Section 3: Deserialization Pipeline
**Analyze commits:** `a415a4da`, `8e364d05`
**Key Files:** `deserialized_psdb_layout2.go`, `deserialized_psdb_v2.go`

Generate:
- Version detection from header
- Decompression logic
- Bitmap-aware feature extraction
- Layout V1 fallback paths
- Default value substitution for missing features
- Bounds checking and error handling

### Section 4: Data Format Specification
**Reference commits:** All data block commits

Generate:
- Binary layout specification with byte offsets
- Bit packing details for metadata (schema version, expiry, layout version, compression type, data type)
- Bool value encoding (packed bits)
- Bitmap structure (optional, byte-aligned)
- Dense payload format variations by data type
- Compression metadata

### Section 5: Type System & Default Handling
**Analyze commits:** `468d003f` (extend to all types)
**Key Files:** `byteorder/system.go`

Generate:
- Type-to-encoder mapping
- Default value handling per data type
- Type mismatch scenarios and fixes (e.g., FP32 decoder issue)
- Vector vs scalar differences
- String encoding (Pascal string format with 2-byte length prefix)

### Section 6: Bitmap Algorithm
**Analyze commits:** `8e364d05` (bounds checking)
**Key Files:** `deserialized_psdb_layout2.go`

Generate:
- Bitmap indexing formula: `byteIdx = pos / 8`, `bitIdx = pos % 8`
- Bitmap presence detection
- Count set bits before position
- Error handling for undersized bitmaps
- Performance implications of bitmap presence

### Section 7: Storage Integration
**Analyze commits:** Multiple storage updates
**Key Files:** `redis.go`, `scylla.go`, `store.go`

Generate:
- How PSDB blocks are stored in Redis and Scylla
- Serialized size impact on storage
- Retrieval flow through repositories
- Shadow comparison mechanism for validation

### Section 8: Configuration & Metadata
**Analyze commits:** `8c7cbc16`, `965f6c73`

Generate:
- Feature schema versioning
- Slate-aware configuration
- Layout metadata propagation
- Header field encoding

### Section 9: Error Handling & Validation
**Analyze commits:** `8e364d05` (critical fixes)

Generate:
- Bounds checking in bitmap access
- Partial byte handling in bool vectors
- Type mismatch detection
- Corrupted payload detection
- Graceful fallback to defaults

### Section 10: Performance Analysis
**Analyze commits:** `486d25dd`, `ff6223ed`, `272b8db9`
**Key Files:** `layout_comparison_test.go`, `layout_comparison_results.txt`

Generate:
- Space savings metrics (comparison with Layout V1)
- Compression efficiency by data type
- Serialization/deserialization latency
- Bitmap overhead calculations
- Dense vs sparse tradeoff analysis
- Test results summary with 9+ scenarios

### Section 11: Testing Strategy
**Analyze commits:** `7b8bc3d7`, `486d25dd`
**Key Files:** `*_test.go`, `*_bench_test.go`

Generate:
- Unit test organization
- Test coverage areas
- Benchmark test structure
- Shadow comparison validation approach
- Failure scenarios being tested

### Section 12: Migration & Compatibility
**Analyze key deserialization logic**

Generate:
- Version detection mechanism
- Layout V1 to V2 compatibility
- Dual-path implementation strategy
- Fallback to V1 deserializer when bitmap absent
- Forward compatibility considerations

---

## Generated Document Deliverables

1. **LLD_PSDB_Layout_V2_V3.md** (Main Document)
   - Complete design specification
   - Architecture diagrams (ASCII or detailed descriptions)
   - All 12 sections above
   - Code references with line numbers

2. **LLD_Data_Format_Reference.md** (Technical Reference)
   - Byte-level format specification
   - Bit packing details
   - Type encoding tables
   - Example byte sequences

3. **LLD_Algorithm_Guide.md** (Implementation Guide)
   - Serialization algorithm pseudocode
   - Deserialization algorithm pseudocode
   - Bitmap operations with examples
   - Type conversion procedures

4. **LLD_Integration_Points.md** (Integration Guide)
   - API contracts
   - Configuration requirements
   - Error handling expectations
   - Usage examples

---

## Key Questions to Address

1. **Bitmap Overhead:** When is bitmap presence justified vs not?
2. **Sparse Feature Strategy:** How are features selected for sparse encoding?
3. **Type Mismatch Recovery:** How are FP32/FP16 decoder mismatches prevented?
4. **Compression Integration:** How does compression interact with bitmap encoding?
5. **Performance Tradeoffs:** Space vs latency tradeoffs with bitmap presence?
6. **Version Detection:** How are layout versions detected from headers?
7. **Default Propagation:** How are default values stored and retrieved?
8. **Error Recovery:** What happens with corrupted bitmap data?
9. **Test Coverage:** Which scenarios are critical to validate?
10. **Backward Compatibility:** How is Layout V1 support maintained?

---

## Critical Commit References

| Commit | Description | Key Files |
|--------|-------------|-----------|
| a14f1df4 | Layout V2 introduction | perm_storage_datablock_v2.go |
| 468d003f | Extend layout to all types | All data type handlers |
| 272b8db9 | Test data restructuring | *_test.go files |
| ff6223ed | Default calculation fix | system.go, test files |
| 486d25dd | Result report restructure | layout_comparison_test.go |
| a415a4da | Separate layout 2 impl | deserialized_psdb_layout2.go |
| 8e364d05 | Critical bounds fixes | deserialized_psdb_layout2.go, system.go |

---

## Context Files to Analyze

- Bitmap initialization and structure
- Header byte packing logic
- Compression encoder/decoder integration
- Type-specific serialization functions
- Feature extraction with bitmap queries
- Default value handling per type
- Error handling patterns
- Test scenarios and validation logic

---

## Output Format Requirements

- **Markdown format** with proper heading hierarchy
- **Code blocks** with language specification for syntax highlighting
- **ASCII diagrams** for architecture and data structures
- **Tables** for type mappings, byte layouts, and metrics
- **Cross-references** between sections
- **Line number references** to source code (e.g., file.go:123-145)
- **Examples** with sample data and expected outputs
- **Warnings** for critical implementation details
- **Assumptions** stated explicitly
- **TODOs** for areas needing further research

---

## Success Criteria

✅ Complete coverage of all 14 commits' changes
✅ Detailed algorithm descriptions with pseudocode
✅ Data format fully specified at byte/bit level
✅ Integration points clearly documented
✅ 50+ inline code references with context
✅ Multiple architecture diagrams
✅ Comprehensive test strategy documentation
✅ Error handling and edge cases covered
✅ Performance analysis with metrics
✅ Backward compatibility clearly explained
