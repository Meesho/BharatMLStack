package ycsb

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	rowcache "github.com/Meesho/BharatMLStack/ssd-cache/internal/row_cache"
)

// YCSBConfig holds configuration for the YCSB LRU cache adapter
type YCSBConfig struct {
	Capacity          int64
	EvictionThreshold float64
	SlabSizes         []int
}

// DefaultYCSBConfig returns a default configuration for YCSB testing
func DefaultYCSBConfig() YCSBConfig {
	return YCSBConfig{
		Capacity:          1000000, // 1M capacity
		EvictionThreshold: 0.7,     // 70% eviction threshold
		SlabSizes:         []int{64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384},
	}
}

// LRUCacheDB implements the YCSB database interface for our LRU cache
type LRUCacheDB struct {
	cache  *rowcache.LRUCache
	config YCSBConfig
}

// NewLRUCacheDB creates a new LRU cache database instance for YCSB
func NewLRUCacheDB(config YCSBConfig) (*LRUCacheDB, error) {
	// Create LRU cache configuration
	cacheConfig := rowcache.LRUCacheConfig{
		Capacity:          config.Capacity,
		EvictionThreshold: config.EvictionThreshold,
		SlabSizes:         config.SlabSizes,
	}

	// Create the cache
	cache := rowcache.NewLRUCache(cacheConfig)

	return &LRUCacheDB{
		cache:  cache,
		config: config,
	}, nil
}

// NewLRUCacheDBWithDefaults creates a new LRU cache database with default config
func NewLRUCacheDBWithDefaults() (*LRUCacheDB, error) {
	return NewLRUCacheDB(DefaultYCSBConfig())
}

// Initialize initializes the database (no-op for our in-memory cache)
func (db *LRUCacheDB) Initialize(ctx context.Context) error {
	return nil
}

// Close closes the database connection (no-op for our in-memory cache)
func (db *LRUCacheDB) Close() error {
	return nil
}

// Read reads a record from the database
func (db *LRUCacheDB) Read(ctx context.Context, table string, key string, fields []string) (map[string][]byte, error) {
	value := db.cache.Get(key)
	if value == nil {
		return nil, errors.New("record not found")
	}

	// YCSB expects field-based records, so we'll store the value in field0
	result := make(map[string][]byte)
	if len(fields) == 0 {
		// If no specific fields requested, return default field
		result["field0"] = value
	} else {
		// Return only requested fields (for simplicity, we only support field0)
		for _, field := range fields {
			if field == "field0" {
				result[field] = value
			}
		}
	}

	return result, nil
}

// Scan performs a range scan (not efficiently supported by LRU cache)
func (db *LRUCacheDB) Scan(ctx context.Context, table string, startKey string, count int, fields []string) ([]map[string][]byte, error) {
	// LRU cache doesn't support efficient scanning
	// This is a limitation we'll document
	return nil, errors.New("scan operation not supported by LRU cache")
}

// Update updates a record in the database
func (db *LRUCacheDB) Update(ctx context.Context, table string, key string, values map[string][]byte) error {
	// For LRU cache, update is the same as insert
	return db.Insert(ctx, table, key, values)
}

// Insert inserts a record into the database
func (db *LRUCacheDB) Insert(ctx context.Context, table string, key string, values map[string][]byte) error {
	// Extract the value from field0 (YCSB standard field)
	value, exists := values["field0"]
	if !exists {
		return errors.New("field0 not found in values")
	}

	// Insert into cache
	db.cache.Set(key, value)
	return nil
}

// Delete deletes a record from the database
func (db *LRUCacheDB) Delete(ctx context.Context, table string, key string) error {
	// Our LRU cache doesn't have explicit delete - items are evicted based on LRU policy
	// We could implement this by setting a tombstone value, but for now we'll return an error
	return errors.New("explicit delete not supported by LRU cache (items are evicted automatically)")
}

// GetStats returns cache statistics
func (db *LRUCacheDB) GetStats() rowcache.LRUCacheStats {
	return db.cache.Stats()
}

// parseSlabSizes parses comma-separated slab sizes string
func parseSlabSizes(slabSizesStr string) ([]int, error) {
	parts := strings.Split(slabSizesStr, ",")
	slabSizes := make([]int, len(parts))

	for i, part := range parts {
		size, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return nil, fmt.Errorf("invalid slab size '%s': %v", part, err)
		}
		slabSizes[i] = size
	}

	return slabSizes, nil
}

// DB interface registration for go-ycsb
func init() {
	// This would register our database with go-ycsb
	// RegisterDB("lru", func() DB { return &LRUCacheDB{} })
}
