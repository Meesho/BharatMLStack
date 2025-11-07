use criterion::{black_box, criterion_group, criterion_main, BenchmarkId, Criterion, Throughput};
use kvr_cache::{RowCache, RowCacheConfig};
use rand::Rng;
use std::sync::Arc;
use std::thread;

// Generate random keys for testing
fn generate_keys(count: usize) -> Vec<String> {
    (0..count).map(|i| format!("key_{}", i)).collect()
}

// Generate random values
fn generate_value(size: usize) -> Vec<u8> {
    let mut rng = rand::thread_rng();
    (0..size).map(|_| rng.r#gen()).collect()
}

fn bench_single_thread_insert(c: &mut Criterion) {
    let mut group = c.benchmark_group("single_thread_insert");
    
    for size in [100, 1000, 10000] {
        group.throughput(Throughput::Elements(size as u64));
        group.bench_with_input(BenchmarkId::from_parameter(size), &size, |b, &size| {
            let cache = RowCache::new(
                RowCacheConfig::default()
                    .with_max_capacity(size * 2)
                    .with_shard_count(16),
            );
            let keys = generate_keys(size);
            let value = generate_value(100);

            b.iter(|| {
                for key in &keys {
                    cache.insert(black_box(key.clone()), black_box(value.clone()));
                }
            });
        });
    }
    group.finish();
}

fn bench_single_thread_get(c: &mut Criterion) {
    let mut group = c.benchmark_group("single_thread_get");
    
    for size in [100, 1000, 10000] {
        group.throughput(Throughput::Elements(size as u64));
        group.bench_with_input(BenchmarkId::from_parameter(size), &size, |b, &size| {
            let cache = RowCache::new(
                RowCacheConfig::default()
                    .with_max_capacity(size * 2)
                    .with_shard_count(16),
            );
            let keys = generate_keys(size);
            let value = generate_value(100);

            // Pre-populate cache
            for key in &keys {
                cache.insert(key.clone(), value.clone());
            }

            b.iter(|| {
                for key in &keys {
                    let _ = cache.get(black_box(key));
                }
            });
        });
    }
    group.finish();
}

fn bench_mixed_operations(c: &mut Criterion) {
    let mut group = c.benchmark_group("mixed_operations");
    
    for size in [1000, 10000] {
        group.throughput(Throughput::Elements(size as u64));
        group.bench_with_input(BenchmarkId::from_parameter(size), &size, |b, &size| {
            let cache = RowCache::new(
                RowCacheConfig::default()
                    .with_max_capacity(size * 2)
                    .with_shard_count(16),
            );
            let keys = generate_keys(size);
            let value = generate_value(100);

            // Pre-populate 50% of cache
            for i in 0..size / 2 {
                cache.insert(keys[i].clone(), value.clone());
            }

            b.iter(|| {
                let mut rng = rand::thread_rng();
                for _ in 0..size {
                    let idx = rng.gen_range(0..size);
                    let key = &keys[idx];
                    
                    // 70% reads, 30% writes
                    if rng.gen_bool(0.7) {
                        let _ = cache.get(black_box(key));
                    } else {
                        cache.insert(black_box(key.clone()), black_box(value.clone()));
                    }
                }
            });
        });
    }
    group.finish();
}

fn bench_concurrent_reads(c: &mut Criterion) {
    let mut group = c.benchmark_group("concurrent_reads");
    
    for threads in [2, 4, 8, 16, 32, 64, 100] {
        group.throughput(Throughput::Elements(10000));
        group.bench_with_input(BenchmarkId::from_parameter(threads), &threads, |b, &threads| {
            let cache = Arc::new(RowCache::new(
                RowCacheConfig::default()
                    .with_max_capacity(20000)
                    .with_shard_count(64),  // More shards for high concurrency
            ));
            let keys = generate_keys(10000);
            let value = generate_value(100);

            // Pre-populate cache
            for key in &keys {
                cache.insert(key.clone(), value.clone());
            }

            b.iter(|| {
                let mut handles = vec![];
                let ops_per_thread = 10000 / threads;

                for _ in 0..threads {
                    let cache_clone = Arc::clone(&cache);
                    let keys_clone = keys.clone();
                    
                    let handle = thread::spawn(move || {
                        let mut rng = rand::thread_rng();
                        for _ in 0..ops_per_thread {
                            let idx = rng.gen_range(0..keys_clone.len());
                            let _ = cache_clone.get(&keys_clone[idx]);
                        }
                    });
                    handles.push(handle);
                }

                for handle in handles {
                    handle.join().unwrap();
                }
            });
        });
    }
    group.finish();
}

fn bench_concurrent_writes(c: &mut Criterion) {
    let mut group = c.benchmark_group("concurrent_writes");
    
    for threads in [2, 4, 8, 16, 32, 64, 100] {
        group.throughput(Throughput::Elements(10000));
        group.bench_with_input(BenchmarkId::from_parameter(threads), &threads, |b, &threads| {
            let cache = Arc::new(RowCache::new(
                RowCacheConfig::default()
                    .with_max_capacity(20000)
                    .with_shard_count(64),  // More shards for high concurrency
            ));
            let value = generate_value(100);

            b.iter(|| {
                let mut handles = vec![];
                let ops_per_thread = 10000 / threads;

                for thread_idx in 0..threads {
                    let cache_clone = Arc::clone(&cache);
                    let value_clone = value.clone();
                    
                    let handle = thread::spawn(move || {
                        for i in 0..ops_per_thread {
                            let key = format!("key_{}_{}", thread_idx, i);
                            cache_clone.insert(key, value_clone.clone());
                        }
                    });
                    handles.push(handle);
                }

                for handle in handles {
                    handle.join().unwrap();
                }
            });
        });
    }
    group.finish();
}

fn bench_concurrent_mixed(c: &mut Criterion) {
    let mut group = c.benchmark_group("concurrent_mixed");
    
    for threads in [2, 4, 8, 16, 32, 64, 100] {
        group.throughput(Throughput::Elements(10000));
        group.bench_with_input(BenchmarkId::from_parameter(threads), &threads, |b, &threads| {
            let cache = Arc::new(RowCache::new(
                RowCacheConfig::default()
                    .with_max_capacity(20000)
                    .with_shard_count(64),  // More shards for high concurrency
            ));
            let keys = generate_keys(10000);
            let value = generate_value(100);

            // Pre-populate 50% of cache
            for i in 0..5000 {
                cache.insert(keys[i].clone(), value.clone());
            }

            b.iter(|| {
                let mut handles = vec![];
                let ops_per_thread = 10000 / threads;

                for _ in 0..threads {
                    let cache_clone = Arc::clone(&cache);
                    let keys_clone = keys.clone();
                    let value_clone = value.clone();
                    
                    let handle = thread::spawn(move || {
                        let mut rng = rand::thread_rng();
                        for _ in 0..ops_per_thread {
                            let idx = rng.gen_range(0..keys_clone.len());
                            let key = &keys_clone[idx];
                            
                            // 80% reads, 20% writes (realistic workload)
                            if rng.gen_bool(0.8) {
                                let _ = cache_clone.get(key);
                            } else {
                                cache_clone.insert(key.clone(), value_clone.clone());
                            }
                        }
                    });
                    handles.push(handle);
                }

                for handle in handles {
                    handle.join().unwrap();
                }
            });
        });
    }
    group.finish();
}

fn bench_eviction_performance(c: &mut Criterion) {
    let mut group = c.benchmark_group("eviction");
    
    for capacity in [100, 1000, 5000] {
        group.throughput(Throughput::Elements(capacity as u64));
        group.bench_with_input(BenchmarkId::from_parameter(capacity), &capacity, |b, &capacity| {
            let cache = RowCache::new(
                RowCacheConfig::default()
                    .with_max_capacity(capacity)
                    .with_shard_count(16),
            );
            let value = generate_value(100);

            b.iter(|| {
                // Insert 2x capacity to trigger evictions
                for i in 0..capacity * 2 {
                    let key = format!("key_{}", i);
                    cache.insert(black_box(key), black_box(value.clone()));
                }
            });
        });
    }
    group.finish();
}

fn bench_shard_count_impact(c: &mut Criterion) {
    let mut group = c.benchmark_group("shard_count_impact");
    
    for shard_count in [1, 4, 8, 16, 32, 64] {
        group.bench_with_input(
            BenchmarkId::from_parameter(shard_count),
            &shard_count,
            |b, &shard_count| {
                let cache = Arc::new(RowCache::new(
                    RowCacheConfig::default()
                        .with_max_capacity(20000)
                        .with_shard_count(shard_count),
                ));
                let keys = generate_keys(10000);
                let value = generate_value(100);

                // Pre-populate
                for key in &keys {
                    cache.insert(key.clone(), value.clone());
                }

                b.iter(|| {
                    let mut handles = vec![];
                    let threads = 8;
                    let ops_per_thread = 1000;

                    for _ in 0..threads {
                        let cache_clone = Arc::clone(&cache);
                        let keys_clone = keys.clone();
                        let value_clone = value.clone();
                        
                        let handle = thread::spawn(move || {
                            let mut rng = rand::thread_rng();
                            for _ in 0..ops_per_thread {
                                let idx = rng.gen_range(0..keys_clone.len());
                                let key = &keys_clone[idx];
                                
                                if rng.gen_bool(0.8) {
                                    let _ = cache_clone.get(key);
                                } else {
                                    cache_clone.insert(key.clone(), value_clone.clone());
                                }
                            }
                        });
                        handles.push(handle);
                    }

                    for handle in handles {
                        handle.join().unwrap();
                    }
                });
            },
        );
    }
    group.finish();
}

criterion_group!(
    benches,
    bench_single_thread_insert,
    bench_single_thread_get,
    bench_mixed_operations,
    bench_concurrent_reads,
    bench_concurrent_writes,
    bench_concurrent_mixed,
    bench_eviction_performance,
    bench_shard_count_impact,
);

criterion_main!(benches);

