---
title: Benchmarks
sidebar_position: 3
---
# Serialization Performance Benchmarks

## Summary

This report presents comprehensive benchmark results comparing three serialization formats for the BharatML Online Feature Store:

- **PSDB (Permanent Storage Data Block)** - Our custom format
- **Protocol Buffers v3** - Google's binary serialization
- **Apache Arrow** - Columnar in-memory analytics format

**Key Findings:**
- 🏆 **PSDB excels at small-to-medium scales** (100-1,000 features)
- ⚡ **35% faster** than Proto3, but **67% slower** than Arrow (for 100k features)
- 📦 **18% smaller** than Proto3, comparable to Arrow
- 🧠 **93% fewer allocations** than Arrow (4 vs 66 allocs/op)

## Test Methodology

### Environment
- **Platform**: macOS ARM64 (Apple Silicon)
- **Go Version**: 1.22.12
- **Test Date**: January 2025
- **Compression**: Disabled for fair comparison (`compression.TypeNone`)

### Test Data
- **Data Type**: Int32 arrays
- **Feature Group Sizes**: 100, 1,000, 10,000, 100,000 features
- **Test Iterations**: Variable (Go benchmark auto-scaling)
- **Pool Optimization**: PSDB uses object pooling for memory efficiency

## Performance Results

### Serialization Speed (Lower is Better)

| Feature Count | PSDB (ns/op) | Proto3 (ns/op) | Arrow (ns/op) | PSDB vs Proto3 | PSDB vs Arrow |
|---------------|--------------|----------------|---------------|----------------|---------------|
| 100           | 625          | 696            | 3,831         | **10% faster** | **84% faster** |
| 1,000         | 4,056        | 6,004          | 5,191         | **32% faster** | **22% faster** |
| 10,000        | 37,357       | 57,674         | 23,173        | **35% faster** | **38% slower** |
| 100,000       | 359,932      | 556,541        | 118,489       | **35% faster** | **67% slower** |

### Serialized Size (Lower is Better)

| Feature Count | Raw Size (bytes) | PSDB (bytes) | Proto3 (bytes) | Arrow (bytes) | PSDB Ratio | Proto3 Ratio | Arrow Ratio |
|---------------|------------------|--------------|----------------|---------------|------------|--------------|-------------|
| 100           | 400              | 409          | 490            | 680           | **102.2%** | 122.5%       | 170.0%      |
| 1,000         | 4,000            | 4,009        | 4,881          | 4,280         | **100.2%** | 122.0%       | 107.0%      |
| 10,000        | 40,000           | 40,009       | 48,717         | 40,280        | **100.0%** | 121.8%       | 100.7%      |
| 100,000       | 400,000          | 400,009      | 487,225        | 400,280       | **100.0%** | 121.8%       | 100.1%      |

### Memory Efficiency (Lower is Better)

| Feature Count | PSDB (B/op) | Proto3 (B/op) | Arrow (B/op) | PSDB (allocs/op) | Proto3 (allocs/op) | Arrow (allocs/op) |
|---------------|-------------|---------------|--------------|------------------|-------------------|-------------------|
| 100           | 461         | 768           | 7,032        | **4**            | **2**             | 66                |
| 1,000         | 4,143       | 5,632         | 15,544       | **4**            | **2**             | 66                |
| 10,000        | 41,029      | 49,408        | 122,617      | **4**            | **2**             | 66                |
| 100,000       | 401,814     | 491,776       | 957,948      | **4**            | **2**             | 66                |

### Throughput (Higher is Better)

| Format | Throughput (MB/s) | Relative Performance |
|--------|-------------------|---------------------|
| **PSDB** | **975.31** | Baseline (100%) |
| Proto3 | 666.12 | 68% of PSDB |
| Arrow | 768.25 | 79% of PSDB |

## Detailed Analysis

### PSDB Advantages
1. **Minimal Overhead**: Only 9-byte header + raw data
2. **Optimal Packing**: No padding or metadata bloat
3. **Memory Pooling**: Reuses objects to minimize allocations
4. **Native Optimization**: Designed specifically for feature store use cases

### Protocol Buffers Analysis
- **Consistent Overhead**: ~22% size penalty across all scales
- **Moderate Speed**: Reasonable serialization performance
- **Low Allocations**: Only 2 allocations per operation
- **Varint Encoding**: Efficient for smaller integers

### Apache Arrow Analysis
- **High Setup Cost**: Complex object creation (66 allocations)
- **Good Large-Scale**: Better relative performance with more data
- **Size Efficient**: Approaches raw data size for large datasets
- **Memory Intensive**: Significant memory overhead per operation

## Scaling Characteristics

### Small Datasets (100-1,000 features)
- **PSDB**: Consistent low overhead
- **Proto3**: Moderate overhead, stable performance
- **Arrow**: High setup cost dominates

### Large Datasets (10,000+ features)
- **PSDB**: Linear scaling, maintains efficiency
- **Proto3**: Good scaling but with consistent 22% size penalty
- **Arrow**: Better amortization of setup costs


## Technical Implementation Notes

### PSDB Optimizations
```go
// Object pooling for zero allocations
var psdbPool = GetPSDBPool()

// Direct buffer allocation
headerSize := PSDBLayout1LengthBytes  // 9 bytes
dataSize := len(data) * 4            // 4 bytes per int32

// No compression for maximum speed
compressionType = compression.TypeNone
```

### Memory Layout Comparison
```
PSDB Layout:    [9-byte header][raw data]
Proto3 Layout:  [varint lengths][encoded data][padding]
Arrow Layout:   [schema][metadata][buffers][padding]
```

## Conclusion

**The optimal format depends on your use case and scale**:

### **PSDB: Best for Small-Medium Scale (≤1,000 features)**
- **Excellent speed**: Up to 83% faster than Arrow for small datasets
- **Optimal size efficiency**: Closest to raw data size (100.0-102.2%)
- **Memory efficiency**: Only 4 allocations per operation
- **Low overhead**: Minimal 9-byte header

### **Apache Arrow: Best for Large Scale (≥10,000 features)**  
- **Superior large-scale performance**: 67% faster than PSDB at 100k features
- **Efficient scaling**: Better amortization of setup costs
- **Size competitive**: Approaches raw data size for large datasets

### **Protocol Buffers: Balanced Middle Ground**
- **Consistent performance**: Moderate speed across all scales
- **Standard tooling**: Wide ecosystem support
- **Predictable overhead**: ~22% size penalty but stable

**Recommendation**: For the Online Feature Store's typical use patterns with **sub-1,000 feature requests**, **PSDB is the optimal choice** for production deployments.

## Raw Benchmark Output [Uncompressed Data]

```
goos: darwin
goarch: arm64
pkg: github.com/Meesho/BharatMLStack/online-feature-store/internal/data/blocks
BenchmarkInt32SerializationPSDB/PSDB/Size-100-10                 1940238               625.3 ns/op             409.0 bytes       461 B/op          4 allocs/op
BenchmarkInt32SerializationPSDB/PSDB/Size-1000-10                 288300              4056 ns/op              4009 bytes        4143 B/op          4 allocs/op
BenchmarkInt32SerializationPSDB/PSDB/Size-10000-10                 32144             37357 ns/op             40009 bytes       41032 B/op          4 allocs/op
BenchmarkInt32SerializationPSDB/PSDB/Size-100000-10                 3244            359932 ns/op            400009 bytes      401572 B/op          4 allocs/op
BenchmarkInt32SerializationProto3/Proto3/Size-100-10             1703066               695.9 ns/op             486.0 bytes       768 B/op          2 allocs/op
BenchmarkInt32SerializationProto3/Proto3/Size-1000-10             194142              6004 ns/op              4885 bytes        5632 B/op          2 allocs/op
BenchmarkInt32SerializationProto3/Proto3/Size-10000-10             20937             57674 ns/op             48734 bytes       49408 B/op          2 allocs/op
BenchmarkInt32SerializationProto3/Proto3/Size-100000-10             2085            556541 ns/op            487263 bytes      491776 B/op          2 allocs/op
BenchmarkInt32SerializationArrow/Arrow/Size-100-10                302257              3831 ns/op               680.0 bytes      7032 B/op         66 allocs/op
BenchmarkInt32SerializationArrow/Arrow/Size-1000-10               228718              5191 ns/op              4280 bytes       15544 B/op         66 allocs/op
BenchmarkInt32SerializationArrow/Arrow/Size-10000-10               52482             23173 ns/op             40280 bytes      122617 B/op         66 allocs/op
BenchmarkInt32SerializationArrow/Arrow/Size-100000-10               9765            120081 ns/op            400280 bytes      957948 B/op         66 allocs/op
BenchmarkInt32SerializationComparison/Comparison/Size-100/PSDB-10        1919401               670.2 ns/op        409.0 bytes            461 B/op          4 allocs/op
BenchmarkInt32SerializationComparison/Comparison/Size-100/Proto3-10      1733599               693.2 ns/op        490.0 bytes            768 B/op          2 allocs/op
BenchmarkInt32SerializationComparison/Comparison/Size-100/Arrow-10        304066              3896 ns/op          680.0 bytes           7032 B/op         66 allocs/op
BenchmarkInt32SerializationComparison/Comparison/Size-1000/PSDB-10        290784              4074 ns/op         4009 bytes     4143 B/op          4 allocs/op
BenchmarkInt32SerializationComparison/Comparison/Size-1000/Proto3-10      196962              6034 ns/op         4882 bytes     5632 B/op          2 allocs/op
BenchmarkInt32SerializationComparison/Comparison/Size-1000/Arrow-10       227908              5240 ns/op         4280 bytes    15544 B/op         66 allocs/op
BenchmarkInt32SerializationComparison/Comparison/Size-10000/PSDB-10        31732             38064 ns/op        40009 bytes    41024 B/op          4 allocs/op
BenchmarkInt32SerializationComparison/Comparison/Size-10000/Proto3-10              20827             57670 ns/op         48745 bytes           49408 B/op          2 allocs/op
BenchmarkInt32SerializationComparison/Comparison/Size-10000/Arrow-10               52000             23557 ns/op         40280 bytes          122617 B/op         66 allocs/op
BenchmarkInt32SerializationComparison/Comparison/Size-100000/PSDB-10                3268            363817 ns/op        400009 bytes          401575 B/op          4 allocs/op
BenchmarkInt32SerializationComparison/Comparison/Size-100000/Proto3-10              2097            559621 ns/op        487247 bytes          491776 B/op          2 allocs/op
BenchmarkInt32SerializationComparison/Comparison/Size-100000/Arrow-10              10000            118489 ns/op        400280 bytes          957947 B/op         66 allocs/op
BenchmarkInt32SizeComparison/SizeOnly/Size-100-10                               1000000000               0.0000223 ns/op           680.0 arrow_bytes               170.0 arrow_ratio_pct           490.0 proto3_bytes         122.5 proto3_ratio_pct           409.0 psdb_bytes        102.2 psdb_ratio_pct            400.0 raw_bytes
BenchmarkInt32SizeComparison/SizeOnly/Size-1000-10                              1000000000               0.0000379 ns/op          4280 arrow_bytes         107.0 arrow_ratio_pct          4881 proto3_bytes        122.0 proto3_ratio_pct             4009 psdb_bytes          100.2 psdb_ratio_pct           4000 raw_bytes
BenchmarkInt32SizeComparison/SizeOnly/Size-10000-10                             1000000000               0.0001182 ns/op         40280 arrow_bytes         100.7 arrow_ratio_pct         48717 proto3_bytes        121.8 proto3_ratio_pct            40009 psdb_bytes          100.0 psdb_ratio_pct          40000 raw_bytes
BenchmarkInt32SizeComparison/SizeOnly/Size-100000-10                            1000000000               0.001034 ns/op         400280 arrow_bytes         100.1 arrow_ratio_pct        487225 proto3_bytes        121.8 proto3_ratio_pct           400009 psdb_bytes          100.0 psdb_ratio_pct         400000 raw_bytes
BenchmarkInt32MemoryEfficiency/Memory/Size-100/PSDB_Pooled-10                    1926676               622.4 ns/op       461 B/op          4 allocs/op
BenchmarkInt32MemoryEfficiency/Memory/Size-100/Proto3-10                         1713428               685.0 ns/op       768 B/op          2 allocs/op
BenchmarkInt32MemoryEfficiency/Memory/Size-100/Arrow-10                           312584              4029 ns/op        7032 B/op         66 allocs/op
BenchmarkInt32MemoryEfficiency/Memory/Size-1000/PSDB_Pooled-10                    290197              4189 ns/op        4143 B/op          4 allocs/op
BenchmarkInt32MemoryEfficiency/Memory/Size-1000/Proto3-10                         195694              6078 ns/op        5632 B/op          2 allocs/op
BenchmarkInt32MemoryEfficiency/Memory/Size-1000/Arrow-10                          224722              5190 ns/op       15544 B/op         66 allocs/op
BenchmarkInt32MemoryEfficiency/Memory/Size-10000/PSDB_Pooled-10                    31898             37684 ns/op       41029 B/op          4 allocs/op
BenchmarkInt32MemoryEfficiency/Memory/Size-10000/Proto3-10                         20840             58032 ns/op       49408 B/op          2 allocs/op
BenchmarkInt32MemoryEfficiency/Memory/Size-10000/Arrow-10                          51440             24049 ns/op      122617 B/op         66 allocs/op
BenchmarkInt32MemoryEfficiency/Memory/Size-100000/PSDB_Pooled-10                    3325            357690 ns/op      401814 B/op          4 allocs/op
BenchmarkInt32MemoryEfficiency/Memory/Size-100000/Proto3-10                         2158            559694 ns/op      491776 B/op          2 allocs/op
BenchmarkInt32MemoryEfficiency/Memory/Size-100000/Arrow-10                          9622            117515 ns/op      957948 B/op         66 allocs/op
BenchmarkInt32Throughput/Throughput/PSDB-10                                       290912              4101 ns/op     975.31 MB/s        4143 B/op          4 allocs/op
BenchmarkInt32Throughput/Throughput/Proto3-10                                     199087              6005 ns/op     666.12 MB/s        5632 B/op          2 allocs/op
BenchmarkInt32Throughput/Throughput/Arrow-10                                      229594              5207 ns/op     768.25 MB/s       15544 B/op         66 allocs/op
BenchmarkGetPSDBPoolWithoutPool-10                                              23836599                50.64 ns/op      192 B/op          1 allocs/op
BenchmarkGetPSDBPoolWithPool-10                                                 100000000               10.76 ns/op        0 B/op          0 allocs/op
PASS
ok      github.com/Meesho/BharatMLStack/online-feature-store/internal/data/blocks       58.891s
```

---

*Benchmarks run on Apple Silicon (ARM64) with Go 1.22.12. Results may vary on different architectures and Go versions.*
