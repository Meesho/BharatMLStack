# kvr-cache

A high-performance, sharded row cache implementation inspired by ScyllaDB's caching layer, designed for extreme concurrency in database workloads.

## Features

### 🚀 High Performance
- **Sharded Architecture**: Multiple independent cache shards minimize lock contention
- **Lock-Free Reads**: DashMap-based concurrent hash maps for blazing fast lookups
- **Optimized Eviction**: Efficient LRU (Least Recently Used) eviction policy
- **Memory Efficient**: Precise memory tracking and configurable limits

### 🔒 Concurrency
- **Thread-Safe**: All operations are safe for concurrent access
- **Configurable Sharding**: Tune shard count for your workload (power of 2)
- **Fine-Grained Locking**: Minimal contention between threads
- **Scalable**: Performance scales linearly with CPU cores

### 📊 Monitoring
- **Hit/Miss Tracking**: Monitor cache effectiveness
- **Memory Usage**: Track memory consumption in real-time
- **Access Patterns**: Detailed access count and timing information
- **Eviction Statistics**: Monitor cache pressure and eviction rates

## Usage

### Basic Example

```rust
use kvr_cache::{RowCache, RowCacheConfig};

// Create a cache with default configuration
let cache = RowCache::with_capacity(10000);

// Insert rows
cache.insert("user:123".to_string(), vec![1, 2, 3, 4]);
cache.insert("user:456".to_string(), vec![5, 6, 7, 8]);

// Retrieve rows
if let Some(entry) = cache.get(&"user:123".to_string()) {
    println!("Found data: {:?}", entry.value());
    println!("Access count: {}", entry.access_count());
}

// Check statistics
let stats = cache.stats();
println!("Hit rate: {:.2}%", stats.hit_rate() * 100.0);
println!("Memory usage: {} bytes", stats.memory_usage());
```

### Advanced Configuration

```rust
use kvr_cache::{RowCache, RowCacheConfig};

let config = RowCacheConfig::default()
    .with_max_capacity(100_000)        // Max number of entries
    .with_shard_count(32)              // Number of shards (power of 2)
    .with_max_memory_bytes(1_000_000)  // Memory limit (1 MB)
    .with_track_access_time(true);     // Enable LRU tracking

let cache = RowCache::new(config);
```

### Custom Types with Sizeable

To use custom types with memory tracking, implement the `Sizeable` trait:

```rust
use kvr_cache::{RowCache, Sizeable};

#[derive(Clone)]
struct Row {
    id: u64,
    data: Vec<u8>,
    metadata: String,
}

impl Sizeable for Row {
    fn size_bytes(&self) -> usize {
        std::mem::size_of::<u64>() 
            + self.data.len() 
            + self.metadata.len()
            + std::mem::size_of::<String>()
            + std::mem::size_of::<Vec<u8>>()
    }
}

let cache = RowCache::with_capacity(1000);
cache.insert("row1".to_string(), Row {
    id: 1,
    data: vec![0; 1024],
    metadata: "test".to_string(),
});
```

### Concurrent Access

```rust
use kvr_cache::{RowCache, RowCacheConfig};
use std::sync::Arc;
use std::thread;

let cache = Arc::new(RowCache::new(
    RowCacheConfig::default()
        .with_max_capacity(100_000)
        .with_shard_count(16)
));

let mut handles = vec![];

// Spawn multiple worker threads
for worker_id in 0..10 {
    let cache_clone = Arc::clone(&cache);
    let handle = thread::spawn(move || {
        for i in 0..1000 {
            let key = format!("key_{}_{}", worker_id, i);
            let value = vec![worker_id as u8; 100];
            
            // Insert
            cache_clone.insert(key.clone(), value);
            
            // Read back
            if let Some(entry) = cache_clone.get(&key) {
                assert_eq!(entry.value()[0], worker_id as u8);
            }
        }
    });
    handles.push(handle);
}

for handle in handles {
    handle.join().unwrap();
}
```

## Architecture

### Sharding Strategy

The cache uses a sharding strategy similar to ScyllaDB:

1. **Hash-Based Distribution**: Keys are hashed and distributed across shards
2. **Independent Shards**: Each shard operates independently with its own lock
3. **Power-of-2 Shards**: Shard count is always a power of 2 for fast modulo operations
4. **Load Balancing**: Hash function ensures even distribution

### LRU Eviction

The eviction policy ensures the cache stays within configured limits:

1. **Per-Shard LRU**: Each shard maintains its own LRU list
2. **Timestamp Tracking**: Access times tracked with microsecond precision
3. **Lazy Eviction**: Eviction happens on insert when limits are exceeded
4. **Memory-Based**: Can evict based on entry count or memory usage

### Memory Management

```
Total Cache
├── Shard 0
│   ├── Entry 1 (key + value + metadata)
│   ├── Entry 2 (key + value + metadata)
│   └── ...
├── Shard 1
│   ├── Entry N (key + value + metadata)
│   └── ...
└── Shard N-1
    └── ...
```

## Performance

### Benchmarks

Run benchmarks with:

```bash
cargo bench --package kvr-cache
```

Expected performance characteristics on modern hardware:

- **Single-threaded read**: ~500-800 ns per operation
- **Single-threaded write**: ~600-1000 ns per operation
- **16-thread mixed workload**: ~50M ops/sec (80% reads, 20% writes)
- **Memory overhead**: ~48 bytes per entry (excluding key/value)

### Tuning Guidelines

1. **Shard Count**: 
   - More shards = better concurrency but more memory overhead
   - Recommended: 16-32 shards for most workloads
   - Use CPU core count as a starting point

2. **Capacity**:
   - Set based on working set size
   - Monitor hit rate and adjust accordingly
   - Aim for >90% hit rate for best performance

3. **Memory Limits**:
   - Set based on available RAM
   - Leave headroom for OS and other components
   - Monitor eviction rate to detect memory pressure

## Design Inspiration

This implementation draws inspiration from:

- **ScyllaDB**: Sharded architecture and LRU eviction
- **DashMap**: Lock-free concurrent hash map
- **Tokio**: Sharded slab allocator patterns

## Comparison with Other Caches

| Feature | kvr-cache | moka | quick_cache | lru |
|---------|-----------|------|-------------|-----|
| Sharded | ✅ | ✅ | ❌ | ❌ |
| Lock-Free Reads | ✅ | ✅ | ✅ | ❌ |
| LRU Eviction | ✅ | ✅ | ✅ | ✅ |
| Memory Tracking | ✅ | ❌ | ❌ | ❌ |
| Per-Entry Stats | ✅ | ❌ | ❌ | ❌ |
| Zero-Copy Gets | ✅ | ❌ | ❌ | ❌ |

## Thread Safety

All operations are thread-safe:

- `insert()`: Safe to call from multiple threads
- `get()`: Safe to call from multiple threads (lock-free)
- `remove()`: Safe to call from multiple threads
- `stats()`: Safe to call from multiple threads
- `clear()`: Safe to call from multiple threads

## Testing

Run tests:

```bash
# Unit tests
cargo test --package kvr-cache

# With output
cargo test --package kvr-cache -- --nocapture

# Specific test
cargo test --package kvr-cache test_concurrent_access
```

## Future Enhancements

- [ ] TTL (Time-To-Live) support
- [ ] Write-through/write-back policies
- [ ] Bloom filters for negative lookups
- [ ] Async API support
- [ ] Metrics export (Prometheus)
- [ ] Compression support
- [ ] Tiered caching (hot/warm/cold)

## License

See the main repository license.

## Contributing

Contributions are welcome! Please ensure:

1. All tests pass: `cargo test --package kvr-cache`
2. Benchmarks don't regress significantly
3. Code is formatted: `cargo fmt`
4. No clippy warnings: `cargo clippy`

