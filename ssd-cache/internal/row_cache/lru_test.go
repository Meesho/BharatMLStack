package rowcache

import (
	"reflect"
	"testing"
)

func TestNewLRUCache(t *testing.T) {
	config := LRUCacheConfig{
		Capacity:          100,
		EvictionThreshold: 0.5,
		SlabSizes:         []int{64, 128, 256, 512, 1024},
	}

	cache := NewLRUCache(config)

	if cache.Capacity != 100 {
		t.Errorf("Expected capacity 100, got %d", cache.Capacity)
	}
	if cache.EvictionThreshold != 0.5 {
		t.Errorf("Expected eviction threshold 0.5, got %f", cache.EvictionThreshold)
	}
	if cache.size != 0 {
		t.Errorf("Expected initial size 0, got %d", cache.size)
	}
	if cache.head != nil {
		t.Errorf("Expected head to be nil initially")
	}
	if cache.tail != nil {
		t.Errorf("Expected tail to be nil initially")
	}
	if cache.cache == nil {
		t.Errorf("Expected cache map to be initialized")
	}
	if len(cache.cache) != 0 {
		t.Errorf("Expected empty cache map initially")
	}
}

func TestLRUCache_SetAndGet_SingleItem(t *testing.T) {
	config := LRUCacheConfig{
		Capacity:          10,
		EvictionThreshold: 0.5,
		SlabSizes:         []int{64, 128, 256},
	}
	cache := NewLRUCache(config)

	key := "test_key"
	value := []byte("test_value")

	cache.Set(key, value)

	retrievedValue := cache.Get(key)
	if !reflect.DeepEqual(retrievedValue, value) {
		t.Errorf("Expected %v, got %v", value, retrievedValue)
	}

	if cache.size != 1 {
		t.Errorf("Expected size 1, got %d", cache.size)
	}
}

func TestLRUCache_Get_NonExistentKey(t *testing.T) {
	config := LRUCacheConfig{
		Capacity:          10,
		EvictionThreshold: 0.5,
		SlabSizes:         []int{64, 128, 256},
	}
	cache := NewLRUCache(config)

	retrievedValue := cache.Get("non_existent")
	if retrievedValue != nil {
		t.Errorf("Expected nil for non-existent key, got %v", retrievedValue)
	}
}

func TestLRUCache_SetAndGet_MultipleItems(t *testing.T) {
	config := LRUCacheConfig{
		Capacity:          10,
		EvictionThreshold: 0.5,
		SlabSizes:         []int{64, 128, 256},
	}
	cache := NewLRUCache(config)

	testData := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
		"key3": []byte("value3"),
	}

	// Set all values
	for key, value := range testData {
		cache.Set(key, value)
	}

	// Verify all values can be retrieved
	for key, expectedValue := range testData {
		retrievedValue := cache.Get(key)
		if !reflect.DeepEqual(retrievedValue, expectedValue) {
			t.Errorf("For key %s: expected %v, got %v", key, expectedValue, retrievedValue)
		}
	}

	if cache.size != int64(len(testData)) {
		t.Errorf("Expected size %d, got %d", len(testData), cache.size)
	}
}

func TestLRUCache_UpdateExistingKey(t *testing.T) {
	config := LRUCacheConfig{
		Capacity:          10,
		EvictionThreshold: 0.5,
		SlabSizes:         []int{64, 128, 256},
	}
	cache := NewLRUCache(config)

	key := "update_key"
	originalValue := []byte("original")
	updatedValue := []byte("updated")

	// Set original value
	cache.Set(key, originalValue)
	if cache.size != 1 {
		t.Errorf("Expected size 1 after first set, got %d", cache.size)
	}

	// Update with new value
	cache.Set(key, updatedValue)
	if cache.size != 1 {
		t.Errorf("Expected size 1 after update, got %d", cache.size)
	}

	// Verify updated value
	retrievedValue := cache.Get(key)
	if !reflect.DeepEqual(retrievedValue, updatedValue) {
		t.Errorf("Expected updated value %v, got %v", updatedValue, retrievedValue)
	}
}

func TestLRUCache_EvictionWhenCapacityReached(t *testing.T) {
	config := LRUCacheConfig{
		Capacity:          3,
		EvictionThreshold: 0.5, // Will evict 50% when capacity reached
		SlabSizes:         []int{64, 128, 256},
	}
	cache := NewLRUCache(config)

	// Fill cache to capacity
	cache.Set("key1", []byte("value1"))
	cache.Set("key2", []byte("value2"))
	cache.Set("key3", []byte("value3"))

	if cache.size != 3 {
		t.Errorf("Expected size 3, got %d", cache.size)
	}

	// Add one more item to trigger eviction
	cache.Set("key4", []byte("value4"))

	// Should have evicted 50% (1 item) when capacity was reached
	// Size should be 3 (3 original + 1 new - 1 evicted)
	if cache.size != 3 {
		t.Errorf("Expected size 3 after eviction, got %d", cache.size)
	}

	// The newest item should still be accessible
	retrievedValue := cache.Get("key4")
	if !reflect.DeepEqual(retrievedValue, []byte("value4")) {
		t.Errorf("Expected newest item to be accessible after eviction")
	}
}

func TestLRUCache_EvictionThresholdZero(t *testing.T) {
	config := LRUCacheConfig{
		Capacity:          2,
		EvictionThreshold: 0.0, // Should evict at least 1 item
		SlabSizes:         []int{64, 128, 256},
	}
	cache := NewLRUCache(config)

	cache.Set("key1", []byte("value1"))
	cache.Set("key2", []byte("value2"))

	// This should trigger eviction
	cache.Set("key3", []byte("value3"))

	// Should have evicted 1 item (minimum when threshold calculation gives 0)
	if cache.size != 2 {
		t.Errorf("Expected size 2 after eviction, got %d", cache.size)
	}
}

func TestLRUCache_EvictEmptyCache(t *testing.T) {
	config := LRUCacheConfig{
		Capacity:          10,
		EvictionThreshold: 0.5,
		SlabSizes:         []int{64, 128, 256},
	}
	cache := NewLRUCache(config)

	// Calling evict on empty cache should not panic
	cache.Evict()

	if cache.size != 0 {
		t.Errorf("Expected size to remain 0, got %d", cache.size)
	}
}

func TestLRUCache_LRUOrdering(t *testing.T) {
	config := LRUCacheConfig{
		Capacity:          3,
		EvictionThreshold: 0.34, // Evict ~33% (1 item) when capacity reached
		SlabSizes:         []int{64, 128, 256},
	}
	cache := NewLRUCache(config)

	// Add items
	cache.Set("key1", []byte("value1"))
	cache.Set("key2", []byte("value2"))
	cache.Set("key3", []byte("value3"))

	// Access key1 to make it most recently used
	cache.Get("key1")

	// Add new item to trigger eviction
	cache.Set("key4", []byte("value4"))

	// key1 should still be accessible (was most recently used)
	retrievedValue := cache.Get("key1")
	if retrievedValue == nil {
		t.Errorf("Expected key1 to still be accessible after eviction")
	}

	// key4 should be accessible (newest)
	retrievedValue = cache.Get("key4")
	if retrievedValue == nil {
		t.Errorf("Expected key4 to be accessible after insertion")
	}

	// key2 should have been evicted (was least recently used)
	retrievedValue = cache.Get("key2")
	if retrievedValue != nil {
		t.Errorf("Expected key2 to be evicted (was least recently used)")
	}
}

func TestNextPowerOfTwo(t *testing.T) {
	testCases := []struct {
		input    uint32
		expected uint32
	}{
		{0, 1},
		{1, 1},
		{2, 2},
		{3, 4},
		{4, 4},
		{5, 8},
		{8, 8},
		{9, 16},
		{15, 16},
		{16, 16},
		{17, 32},
		{31, 32},
		{32, 32},
		{33, 64},
		{100, 128},
		{1000, 1024},
		{1024, 1024},
		{1025, 2048},
	}

	for _, tc := range testCases {
		result := NextPowerOfTwo(tc.input)
		if result != tc.expected {
			t.Errorf("NextPowerOfTwo(%d): expected %d, got %d", tc.input, tc.expected, result)
		}
	}
}

func TestPrevPowerOfTwo(t *testing.T) {
	testCases := []struct {
		input    uint32
		expected uint32
	}{
		{0, 0},
		{1, 1},
		{2, 2},
		{3, 2},
		{4, 4},
		{5, 4},
		{7, 4},
		{8, 8},
		{9, 8},
		{15, 8},
		{16, 16},
		{17, 16},
		{31, 16},
		{32, 32},
		{33, 32},
		{63, 32},
		{64, 64},
		{100, 64},
		{1000, 512},
		{1024, 1024},
		{2000, 1024},
	}

	for _, tc := range testCases {
		result := PrevPowerOfTwo(tc.input)
		if result != tc.expected {
			t.Errorf("PrevPowerOfTwo(%d): expected %d, got %d", tc.input, tc.expected, result)
		}
	}
}

func TestLRUCache_SetSameSizeValue(t *testing.T) {
	config := LRUCacheConfig{
		Capacity:          10,
		EvictionThreshold: 0.5,
		SlabSizes:         []int{64, 128, 256},
	}
	cache := NewLRUCache(config)

	key := "test_key"
	value1 := []byte("hello")
	value2 := []byte("world") // Same length as value1

	cache.Set(key, value1)
	originalSize := cache.size

	cache.Set(key, value2)

	// Size should remain the same
	if cache.size != originalSize {
		t.Errorf("Expected size to remain %d, got %d", originalSize, cache.size)
	}

	// Value should be updated
	retrievedValue := cache.Get(key)
	if !reflect.DeepEqual(retrievedValue, value2) {
		t.Errorf("Expected updated value %v, got %v", value2, retrievedValue)
	}
}

func TestLRUCache_SetDifferentSizeValue(t *testing.T) {
	config := LRUCacheConfig{
		Capacity:          10,
		EvictionThreshold: 0.5,
		SlabSizes:         []int{64, 128, 256},
	}
	cache := NewLRUCache(config)

	key := "test_key"
	shortValue := []byte("hi")
	longValue := []byte("this is a much longer value")

	cache.Set(key, shortValue)
	cache.Set(key, longValue)

	// Value should be updated
	retrievedValue := cache.Get(key)
	if !reflect.DeepEqual(retrievedValue, longValue) {
		t.Errorf("Expected updated value %v, got %v", longValue, retrievedValue)
	}
}

func BenchmarkLRUCache_Set(b *testing.B) {
	config := LRUCacheConfig{
		Capacity:          1000,
		EvictionThreshold: 0.5,
		SlabSizes:         []int{64, 128, 256, 512, 1024},
	}
	cache := NewLRUCache(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := string(rune(i % 1000))
		value := []byte("benchmark_value")
		cache.Set(key, value)
	}
}

func BenchmarkLRUCache_Get(b *testing.B) {
	config := LRUCacheConfig{
		Capacity:          1000,
		EvictionThreshold: 0.5,
		SlabSizes:         []int{64, 128, 256, 512, 1024},
	}
	cache := NewLRUCache(config)

	// Pre-populate cache
	for i := 0; i < 100; i++ {
		key := string(rune(i))
		value := []byte("benchmark_value")
		cache.Set(key, value)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := string(rune(i % 100))
		cache.Get(key)
	}
}
