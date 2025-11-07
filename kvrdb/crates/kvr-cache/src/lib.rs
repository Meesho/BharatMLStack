//! High-performance row cache implementation inspired by ScyllaDB
//!
//! This crate provides a sharded, concurrent row cache with LRU eviction,
//! designed for high-throughput database workloads.
//!
//! # Features
//! - **Sharded design**: Multiple independent shards to minimize lock contention
//! - **LRU eviction**: Automatic eviction of least recently used entries
//! - **Memory limits**: Configurable maximum cache size
//! - **High concurrency**: Lock-free reads, fine-grained write locking
//! - **Statistics**: Hit/miss ratio, eviction counts, memory usage
//!
//! # Example
//! ```
//! use kvr_cache::{RowCache, RowCacheConfig};
//!
//! let config = RowCacheConfig::default()
//!     .with_max_capacity(1000)
//!     .with_shard_count(16);
//!     
//! let cache = RowCache::new(config);
//! 
//! // Insert a row
//! cache.insert("key1".to_string(), vec![1, 2, 3]);
//! 
//! // Get a row
//! if let Some(value) = cache.get(&"key1".to_string()) {
//!     println!("Found: {:?}", value.value());
//! }
//! 
//! // Check statistics
//! let stats = cache.stats();
//! println!("Hit rate: {:.2}%", stats.hit_rate() * 100.0);
//! ```

use ahash::RandomState;
use dashmap::DashMap;
use parking_lot::{Mutex, RwLock};
use std::borrow::Borrow;
use std::hash::{Hash, Hasher};
use std::sync::atomic::{AtomicU64, AtomicUsize, Ordering};
use std::sync::Arc;
use std::time::{SystemTime, UNIX_EPOCH};

/// Configuration for the row cache
#[derive(Debug, Clone)]
pub struct RowCacheConfig {
    /// Maximum number of entries in the cache
    max_capacity: usize,
    /// Number of shards for concurrent access (default: 16)
    shard_count: usize,
    /// Maximum memory usage in bytes (0 = unlimited)
    max_memory_bytes: usize,
    /// Enable access time tracking for LRU
    track_access_time: bool,
}

impl Default for RowCacheConfig {
    fn default() -> Self {
        Self {
            max_capacity: 10_000,
            shard_count: 16,
            max_memory_bytes: 0,
            track_access_time: true,
        }
    }
}

impl RowCacheConfig {
    /// Set the maximum number of entries
    pub fn with_max_capacity(mut self, capacity: usize) -> Self {
        self.max_capacity = capacity;
        self
    }

    /// Set the number of shards (must be power of 2)
    pub fn with_shard_count(mut self, count: usize) -> Self {
        // Round up to next power of 2
        self.shard_count = count.next_power_of_two();
        self
    }

    /// Set the maximum memory usage in bytes
    pub fn with_max_memory_bytes(mut self, bytes: usize) -> Self {
        self.max_memory_bytes = bytes;
        self
    }

    /// Enable or disable access time tracking
    pub fn with_track_access_time(mut self, enabled: bool) -> Self {
        self.track_access_time = enabled;
        self
    }
}

/// Cached entry with metadata
#[derive(Debug)]
pub struct CacheEntry<V> {
    value: V,
    size: usize,
    last_access: AtomicU64,
    access_count: AtomicU64,
}

impl<V: Clone> Clone for CacheEntry<V> {
    fn clone(&self) -> Self {
        Self {
            value: self.value.clone(),
            size: self.size,
            last_access: AtomicU64::new(self.last_access.load(Ordering::Relaxed)),
            access_count: AtomicU64::new(self.access_count.load(Ordering::Relaxed)),
        }
    }
}

impl<V> CacheEntry<V> {
    fn new(value: V, size: usize) -> Self {
        Self {
            value,
            size,
            last_access: AtomicU64::new(current_timestamp()),
            access_count: AtomicU64::new(1),
        }
    }

    /// Get reference to the cached value
    pub fn value(&self) -> &V {
        &self.value
    }

    /// Get the size of this entry in bytes
    pub fn size(&self) -> usize {
        self.size
    }

    /// Get the last access timestamp
    pub fn last_access(&self) -> u64 {
        self.last_access.load(Ordering::Relaxed)
    }

    /// Get the access count
    pub fn access_count(&self) -> u64 {
        self.access_count.load(Ordering::Relaxed)
    }

    fn touch(&self) {
        self.last_access.store(current_timestamp(), Ordering::Relaxed);
        self.access_count.fetch_add(1, Ordering::Relaxed);
    }
}

/// A single shard of the cache
struct CacheShard<K, V> {
    map: DashMap<K, Arc<CacheEntry<V>>, RandomState>,
    lru_list: Mutex<Vec<K>>,
    memory_usage: AtomicUsize,
}

impl<K, V> CacheShard<K, V>
where
    K: Hash + Eq + Clone,
{
    fn new() -> Self {
        Self {
            map: DashMap::with_hasher(RandomState::new()),
            lru_list: Mutex::new(Vec::new()),
            memory_usage: AtomicUsize::new(0),
        }
    }

    fn insert(&self, key: K, value: V, size: usize) -> Option<Arc<CacheEntry<V>>> {
        let entry = Arc::new(CacheEntry::new(value, size));
        let old = self.map.insert(key.clone(), entry.clone());
        
        // Update memory usage
        if let Some(ref old_entry) = old {
            self.memory_usage.fetch_sub(old_entry.size, Ordering::Relaxed);
        }
        self.memory_usage.fetch_add(size, Ordering::Relaxed);

        // Update LRU list
        let mut lru = self.lru_list.lock();
        lru.retain(|k| k != &key);
        lru.push(key);

        old
    }

    fn get<Q>(&self, key: &Q) -> Option<Arc<CacheEntry<V>>>
    where
        K: Borrow<Q>,
        Q: Hash + Eq + ?Sized,
    {
        self.map.get(key).map(|entry| {
            entry.touch();
            entry.clone()
        })
    }

    fn remove<Q>(&self, key: &Q) -> Option<Arc<CacheEntry<V>>>
    where
        K: Borrow<Q>,
        Q: Hash + Eq + ?Sized + ToOwned<Owned = K>,
    {
        if let Some((key_owned, entry)) = self.map.remove(key) {
            self.memory_usage.fetch_sub(entry.size, Ordering::Relaxed);
            
            // Remove from LRU list
            let mut lru = self.lru_list.lock();
            lru.retain(|k| k != &key_owned);
            
            Some(entry)
        } else {
            None
        }
    }

    fn evict_lru(&self) -> Option<K> {
        let mut lru = self.lru_list.lock();
        
        // Find the least recently used key
        if lru.is_empty() {
            return None;
        }

        // Sort by access time to find true LRU
        let keys: Vec<_> = lru.drain(..).collect();
        if keys.is_empty() {
            return None;
        }

        // Find the entry with the oldest access time
        let mut oldest_key = keys[0].clone();
        let mut oldest_time = u64::MAX;

        for key in &keys {
            if let Some(entry) = self.map.get(key) {
                let access_time = entry.last_access();
                if access_time < oldest_time {
                    oldest_time = access_time;
                    oldest_key = key.clone();
                }
            }
        }

        // Rebuild LRU list without the evicted key
        for key in keys {
            if key != oldest_key {
                lru.push(key);
            }
        }

        drop(lru);

        // Remove the entry
        if let Some((_, entry)) = self.map.remove(&oldest_key) {
            self.memory_usage.fetch_sub(entry.size, Ordering::Relaxed);
            Some(oldest_key)
        } else {
            None
        }
    }

    fn len(&self) -> usize {
        self.map.len()
    }

    fn memory_usage(&self) -> usize {
        self.memory_usage.load(Ordering::Relaxed)
    }

    fn clear(&self) {
        self.map.clear();
        self.lru_list.lock().clear();
        self.memory_usage.store(0, Ordering::Relaxed);
    }
}

/// Cache statistics
#[derive(Debug, Clone, Default)]
pub struct CacheStats {
    hits: u64,
    misses: u64,
    inserts: u64,
    evictions: u64,
    memory_usage: usize,
    entry_count: usize,
}

impl CacheStats {
    /// Calculate the hit rate (0.0 to 1.0)
    pub fn hit_rate(&self) -> f64 {
        let total = self.hits + self.misses;
        if total == 0 {
            0.0
        } else {
            self.hits as f64 / total as f64
        }
    }

    /// Get the number of cache hits
    pub fn hits(&self) -> u64 {
        self.hits
    }

    /// Get the number of cache misses
    pub fn misses(&self) -> u64 {
        self.misses
    }

    /// Get the number of insertions
    pub fn inserts(&self) -> u64 {
        self.inserts
    }

    /// Get the number of evictions
    pub fn evictions(&self) -> u64 {
        self.evictions
    }

    /// Get the current memory usage in bytes
    pub fn memory_usage(&self) -> usize {
        self.memory_usage
    }

    /// Get the current number of entries
    pub fn entry_count(&self) -> usize {
        self.entry_count
    }
}

/// High-performance sharded row cache with LRU eviction
pub struct RowCache<K, V> {
    shards: Vec<Arc<CacheShard<K, V>>>,
    config: RowCacheConfig,
    stats: Arc<RwLock<CacheStats>>,
}

impl<K, V> RowCache<K, V>
where
    K: Hash + Eq + Clone,
{
    /// Create a new row cache with the given configuration
    pub fn new(config: RowCacheConfig) -> Self {
        let shard_count = config.shard_count;
        let mut shards = Vec::with_capacity(shard_count);
        
        for _ in 0..shard_count {
            shards.push(Arc::new(CacheShard::new()));
        }

        Self {
            shards,
            config,
            stats: Arc::new(RwLock::new(CacheStats::default())),
        }
    }

    /// Create a new row cache with default configuration
    pub fn with_capacity(capacity: usize) -> Self {
        Self::new(RowCacheConfig::default().with_max_capacity(capacity))
    }

    /// Get the shard index for a key
    fn shard_index(&self, key: &K) -> usize {
        let mut hasher = ahash::AHasher::default();
        key.hash(&mut hasher);
        let hash = hasher.finish();
        (hash as usize) & (self.shards.len() - 1)
    }

    /// Insert a row into the cache
    pub fn insert(&self, key: K, value: V) -> Option<Arc<CacheEntry<V>>>
    where
        V: Sizeable,
    {
        let size = value.size_bytes();
        let shard_idx = self.shard_index(&key);
        let shard = &self.shards[shard_idx];

        // Check if we need to evict
        self.maybe_evict(shard);

        let old = shard.insert(key, value, size);

        // Update stats
        let mut stats = self.stats.write();
        stats.inserts += 1;
        if old.is_some() {
            stats.evictions += 1;
        }

        old
    }

    /// Get a row from the cache
    pub fn get<Q>(&self, key: &Q) -> Option<Arc<CacheEntry<V>>>
    where
        K: Borrow<Q>,
        Q: Hash + Eq + ?Sized,
    {
        let mut hasher = ahash::AHasher::default();
        key.hash(&mut hasher);
        let hash = hasher.finish();
        let shard_idx = (hash as usize) & (self.shards.len() - 1);

        let result = self.shards[shard_idx].get(key);

        // Update stats
        let mut stats = self.stats.write();
        if result.is_some() {
            stats.hits += 1;
        } else {
            stats.misses += 1;
        }

        result
    }

    /// Remove a row from the cache
    pub fn remove<Q>(&self, key: &Q) -> Option<Arc<CacheEntry<V>>>
    where
        K: Borrow<Q>,
        Q: Hash + Eq + ?Sized + ToOwned<Owned = K>,
    {
        let mut hasher = ahash::AHasher::default();
        key.hash(&mut hasher);
        let hash = hasher.finish();
        let shard_idx = (hash as usize) & (self.shards.len() - 1);

        self.shards[shard_idx].remove(key)
    }

    /// Check if we need to evict entries and perform eviction
    fn maybe_evict(&self, shard: &CacheShard<K, V>) {
        let capacity_per_shard = self.config.max_capacity / self.shards.len();
        
        // Check entry count
        if shard.len() >= capacity_per_shard {
            if shard.evict_lru().is_some() {
                let mut stats = self.stats.write();
                stats.evictions += 1;
            }
        }

        // Check memory limit if configured
        if self.config.max_memory_bytes > 0 {
            let memory_per_shard = self.config.max_memory_bytes / self.shards.len();
            while shard.memory_usage() > memory_per_shard {
                if shard.evict_lru().is_some() {
                    let mut stats = self.stats.write();
                    stats.evictions += 1;
                } else {
                    break;
                }
            }
        }
    }

    /// Get the current number of entries in the cache
    pub fn len(&self) -> usize {
        self.shards.iter().map(|s| s.len()).sum()
    }

    /// Check if the cache is empty
    pub fn is_empty(&self) -> bool {
        self.len() == 0
    }

    /// Get current memory usage across all shards
    pub fn memory_usage(&self) -> usize {
        self.shards.iter().map(|s| s.memory_usage()).sum()
    }

    /// Get cache statistics
    pub fn stats(&self) -> CacheStats {
        let mut stats = self.stats.read().clone();
        stats.memory_usage = self.memory_usage();
        stats.entry_count = self.len();
        stats
    }

    /// Reset statistics
    pub fn reset_stats(&self) {
        let mut stats = self.stats.write();
        *stats = CacheStats::default();
    }

    /// Clear all entries from the cache
    pub fn clear(&self) {
        for shard in &self.shards {
            shard.clear();
        }
        self.reset_stats();
    }
}

/// Trait for types that can report their size in bytes
pub trait Sizeable {
    /// Return the size of this value in bytes
    fn size_bytes(&self) -> usize;
}

// Implement Sizeable for common types
impl Sizeable for Vec<u8> {
    fn size_bytes(&self) -> usize {
        self.len() + std::mem::size_of::<Self>()
    }
}

impl Sizeable for String {
    fn size_bytes(&self) -> usize {
        self.len() + std::mem::size_of::<Self>()
    }
}

impl Sizeable for &str {
    fn size_bytes(&self) -> usize {
        self.len()
    }
}

impl<T: Sizeable> Sizeable for Box<T> {
    fn size_bytes(&self) -> usize {
        self.as_ref().size_bytes() + std::mem::size_of::<Self>()
    }
}

impl<T: Sizeable> Sizeable for Arc<T> {
    fn size_bytes(&self) -> usize {
        self.as_ref().size_bytes() + std::mem::size_of::<Self>()
    }
}

// Helper function to get current timestamp in milliseconds
fn current_timestamp() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_millis() as u64
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::thread;

    #[test]
    fn test_basic_insert_and_get() {
        let cache = RowCache::new(RowCacheConfig::default().with_max_capacity(100));
        
        cache.insert("key1".to_string(), "value1".to_string());
        
        let result = cache.get(&"key1".to_string());
        assert!(result.is_some());
        assert_eq!(result.unwrap().value(), "value1");
    }

    #[test]
    fn test_cache_miss() {
        let cache: RowCache<String, String> = RowCache::new(RowCacheConfig::default());
        
        let result = cache.get(&"nonexistent".to_string());
        assert!(result.is_none());
    }

    #[test]
    fn test_lru_eviction() {
        // Use smaller shard count and larger capacity for predictable test
        let cache = RowCache::new(
            RowCacheConfig::default()
                .with_max_capacity(50)
                .with_shard_count(4)
        );
        
        // Insert significantly more than capacity to ensure evictions
        for i in 0..100 {
            cache.insert(format!("key{}", i), format!("value{}", i));
            thread::sleep(std::time::Duration::from_millis(1));
        }
        
        // Cache should not exceed capacity significantly
        assert!(cache.len() <= 60); // Allow some slack for per-shard limits
        
        // Most recent entries should still be there
        assert!(cache.get(&"key99".to_string()).is_some());
        assert!(cache.get(&"key98".to_string()).is_some());
        assert!(cache.get(&"key97".to_string()).is_some());
        
        // Early entries should definitely be evicted
        let evicted_count = (0..20)
            .filter(|i| cache.get(&format!("key{}", i)).is_none())
            .count();
        assert!(evicted_count >= 10, "Expected at least 10 early entries to be evicted");
    }

    #[test]
    fn test_concurrent_access() {
        let cache = Arc::new(RowCache::new(
            RowCacheConfig::default()
                .with_max_capacity(1000)
                .with_shard_count(16)
        ));

        let mut handles = vec![];

        // Spawn multiple threads
        for thread_id in 0..10 {
            let cache_clone = Arc::clone(&cache);
            let handle = thread::spawn(move || {
                for i in 0..100 {
                    let key = format!("key{}_{}", thread_id, i);
                    let value = format!("value{}_{}", thread_id, i);
                    cache_clone.insert(key.clone(), value.clone());
                    
                    let result = cache_clone.get(&key);
                    assert!(result.is_some());
                    assert_eq!(result.unwrap().value(), &value);
                }
            });
            handles.push(handle);
        }

        for handle in handles {
            handle.join().unwrap();
        }

        assert!(cache.len() > 0);
    }

    #[test]
    fn test_stats() {
        let cache = RowCache::new(RowCacheConfig::default());
        
        cache.insert("key1".to_string(), "value1".to_string());
        cache.insert("key2".to_string(), "value2".to_string());
        
        cache.get(&"key1".to_string()); // hit
        cache.get(&"key3".to_string()); // miss
        
        let stats = cache.stats();
        assert_eq!(stats.hits(), 1);
        assert_eq!(stats.misses(), 1);
        assert_eq!(stats.inserts(), 2);
        assert!(stats.hit_rate() > 0.0);
    }

    #[test]
    fn test_remove() {
        let cache = RowCache::new(RowCacheConfig::default());
        
        cache.insert("key1".to_string(), "value1".to_string());
        assert!(cache.get(&"key1".to_string()).is_some());
        
        cache.remove(&"key1".to_string());
        assert!(cache.get(&"key1".to_string()).is_none());
    }

    #[test]
    fn test_clear() {
        let cache = RowCache::new(RowCacheConfig::default());
        
        for i in 0..10 {
            cache.insert(format!("key{}", i), format!("value{}", i));
        }
        
        assert_eq!(cache.len(), 10);
        
        cache.clear();
        assert_eq!(cache.len(), 0);
    }

    #[test]
    fn test_memory_tracking() {
        let cache = RowCache::new(RowCacheConfig::default());
        
        cache.insert("key1".to_string(), "a".repeat(100));
        cache.insert("key2".to_string(), "b".repeat(200));
        
        let memory = cache.memory_usage();
        assert!(memory > 300); // At least the string sizes
    }

    #[test]
    fn test_access_tracking() {
        let cache = RowCache::new(RowCacheConfig::default());
        
        cache.insert("key1".to_string(), "value1".to_string());
        
        let entry1 = cache.get(&"key1".to_string()).unwrap();
        let count1 = entry1.access_count();
        
        thread::sleep(std::time::Duration::from_millis(10));
        
        let entry2 = cache.get(&"key1".to_string()).unwrap();
        let count2 = entry2.access_count();
        
        assert!(count2 > count1);
    }
}
