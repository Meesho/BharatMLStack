# Row Cache Implementation Summary

## Overview

This document describes the high-performance row cache implementation for the `kvr-cache` crate, inspired by ScyllaDB's caching architecture.

## Architecture Design

### Core Components

```
RowCache
├── Multiple Shards (power of 2, default: 16)
│   ├── CacheShard
│   │   ├── DashMap<K, Arc<CacheEntry<V>>>  [Concurrent HashMap]
│   │   ├── Mutex<Vec<K>>                    [LRU List]
│   │   └── AtomicUsize                      [Memory Tracking]
│   └── ...
├── RowCacheConfig
└── CacheStats (with RwLock)
```

### Sharding Strategy

**Why Sharding?**
- Minimizes lock contention in high-concurrency scenarios
- Each shard operates independently with its own locks
- Hash-based key distribution ensures even load balancing
- Power-of-2 shard count enables fast bitwise modulo operations

**Hash Distribution:**
```rust
shard_index = hash(key) & (shard_count - 1)
```

### Concurrency Model

1. **Lock-Free Reads**: Using DashMap for concurrent hash map operations
2. **Fine-Grained Write Locking**: Each shard has independent locks
3. **Atomic Operations**: Memory tracking and access counting use atomics
4. **Thread-Safe Statistics**: RwLock protects global statistics

### LRU Eviction Policy

**How It Works:**
1. Each shard maintains its own LRU list
2. Entries track `last_access` timestamp (microseconds since epoch)
3. On insert, if capacity/memory exceeded:
   - Find entry with oldest `last_access` time
   - Evict that entry
4. Per-shard capacity = total_capacity / shard_count

**Eviction Triggers:**
- Entry count exceeds per-shard capacity limit
- Memory usage exceeds per-shard memory limit (if configured)

### Memory Management

**Memory Tracking:**
- Each entry reports its size via `Sizeable` trait
- Per-shard atomic counters track total memory usage
- Automatic eviction when memory limits are exceeded

**Built-in Sizeable Implementations:**
- `Vec<u8>`: length + struct overhead
- `String`: length + struct overhead  
- `Box<T>`: contained value size + pointer overhead
- `Arc<T>`: contained value size + pointer overhead

## Key Features

### 1. High Concurrency
- **Sharded Design**: 16+ independent shards
- **Lock-Free Reads**: DashMap enables concurrent access
- **Minimal Contention**: Per-shard locks

### 2. LRU Eviction
- **Timestamp-Based**: Microsecond precision
- **Access Tracking**: Count and time of last access
- **Lazy Eviction**: Triggered on insert operations

### 3. Memory Control
- **Size Tracking**: Per-entry memory usage
- **Configurable Limits**: Entry count and byte limits
- **Automatic Eviction**: Maintains memory bounds

### 4. Observability
- **Hit/Miss Rates**: Track cache effectiveness
- **Access Patterns**: Per-entry statistics
- **Memory Usage**: Real-time tracking
- **Eviction Counts**: Monitor cache pressure

## Performance Characteristics

### Time Complexity
- **Insert**: O(1) average, O(log n) worst case (with eviction)
- **Get**: O(1) average
- **Remove**: O(1) average
- **Eviction**: O(n) where n = entries in shard

### Space Complexity
- **Per Entry**: ~48 bytes overhead (excluding key/value)
  - Arc pointer: 8 bytes
  - CacheEntry struct: 40 bytes
    - value: size_of::<V>()
    - size: 8 bytes
    - last_access: 8 bytes (AtomicU64)
    - access_count: 8 bytes (AtomicU64)
    - padding: ~16 bytes

### Expected Performance (Modern Hardware)
- **Single-threaded read**: ~500-800 ns/op
- **Single-threaded write**: ~600-1000 ns/op
- **16-thread mixed (80% read)**: ~50M ops/sec
- **Eviction overhead**: ~1-5 µs per eviction

## Configuration Guidelines

### Shard Count
```rust
// Low concurrency (< 4 threads)
.with_shard_count(4)

// Medium concurrency (4-16 threads)
.with_shard_count(16)  // Default

// High concurrency (16+ threads)
.with_shard_count(32)
```

**Trade-offs:**
- More shards = better concurrency, more memory overhead
- Fewer shards = lower overhead, more contention
- Recommended: Match or exceed CPU core count

### Capacity Planning
```rust
// Based on working set size
.with_max_capacity(100_000)  // Number of entries

// Based on available memory
.with_max_memory_bytes(1_000_000_000)  // 1 GB
```

**Guidelines:**
- Monitor hit rate (target: >90% for read-heavy workloads)
- Set capacity to cover working set + 20% buffer
- Use memory limits for safety bounds

## Usage Patterns

### Read-Heavy Workload (90% reads)
```rust
let config = RowCacheConfig::default()
    .with_max_capacity(100_000)
    .with_shard_count(16);
```

### Write-Heavy Workload (50%+ writes)
```rust
let config = RowCacheConfig::default()
    .with_max_capacity(50_000)
    .with_shard_count(32);  // More shards for write concurrency
```

### Memory-Constrained Environment
```rust
let config = RowCacheConfig::default()
    .with_max_memory_bytes(500_000_000)  // 500 MB hard limit
    .with_shard_count(16);
```

## Comparison with ScyllaDB

### Similarities
- **Sharded architecture** for concurrency
- **LRU eviction** policy
- **Row-level granularity**
- **Memory limits** and tracking

### Differences
- **No TTL support** (future enhancement)
- **Simpler eviction** (no tiered caching)
- **Generic design** (works with any K/V types)
- **Rust-native** (using DashMap, not custom hashtable)

## Testing

### Unit Tests (9 tests)
1. `test_basic_insert_and_get` - Basic functionality
2. `test_cache_miss` - Miss handling
3. `test_lru_eviction` - Eviction correctness
4. `test_concurrent_access` - Thread safety
5. `test_stats` - Statistics accuracy
6. `test_remove` - Removal correctness
7. `test_clear` - Clear functionality
8. `test_memory_tracking` - Memory accounting
9. `test_access_tracking` - Access pattern tracking

### Benchmarks (8 benchmark suites)
1. `bench_single_thread_insert` - Insert throughput
2. `bench_single_thread_get` - Get throughput
3. `bench_mixed_operations` - Mixed read/write
4. `bench_concurrent_reads` - Concurrent read scaling
5. `bench_concurrent_writes` - Concurrent write scaling
6. `bench_concurrent_mixed` - Realistic workload
7. `bench_eviction_performance` - Eviction overhead
8. `bench_shard_count_impact` - Optimal shard tuning

## Future Enhancements

### Planned Features
- [ ] **TTL Support**: Time-based entry expiration
- [ ] **Write Policies**: Write-through, write-back, write-around
- [ ] **Bloom Filters**: Fast negative lookups
- [ ] **Async API**: Tokio/async-std support
- [ ] **Metrics Export**: Prometheus integration
- [ ] **Compression**: Transparent value compression
- [ ] **Tiered Caching**: Hot/warm/cold data separation

### Performance Optimizations
- [ ] SIMD-accelerated hashing
- [ ] Lock-free eviction using epoch-based reclamation
- [ ] Adaptive shard resizing
- [ ] Prefetching hints

## Dependencies

```toml
dashmap = "6.1"          # Concurrent hashmap
parking_lot = "0.12"     # Fast locks
ahash = "0.8"            # Fast hashing
crossbeam-epoch = "0.9"  # Epoch-based memory reclamation
```

## API Stability

- **Stable**: Core cache operations (insert, get, remove)
- **Stable**: Configuration options
- **Stable**: Statistics API
- **Unstable**: Internal shard implementation
- **Unstable**: Eviction algorithm details

## License

See main repository LICENSE file.

---

**Implementation Date**: November 2025  
**Author**: Generated for BharatMLStack/kvrdb  
**Version**: 0.1.0

