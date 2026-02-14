package inmemorycache

import (
	"os"
	"sync"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetCacheInstances resets global cache instances for testing
func resetCacheInstances() {
	namedInstances = nil
	cacheOnce = sync.Once{}
}

// TestInitMultiInMemoryCacheWithConf tests the InitMultiInMemoryCacheWithConf function
func TestInitMultiInMemoryCacheWithConf(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	t.Run("initializes multiple caches with different sizes", func(t *testing.T) {
		resetCacheInstances()

		cacheDetails := []InMemoryCacheDetail{
			{Name: "user-cache", MemorySizeInMb: 100},
			{Name: "product-cache", MemorySizeInMb: 200},
			{Name: "session-cache", MemorySizeInMb: 50},
		}

		InitMultiInMemoryCacheWithConf(cacheDetails)

		// Verify all caches are initialized
		userCache, err := InstanceByName("user-cache")
		require.NoError(t, err)
		require.NotNil(t, userCache)

		productCache, err := InstanceByName("product-cache")
		require.NoError(t, err)
		require.NotNil(t, productCache)

		sessionCache, err := InstanceByName("session-cache")
		require.NoError(t, err)
		require.NotNil(t, sessionCache)

		// Verify each cache has correct size
		assert.Equal(t, 100, userCache.(*V1).SizeInMb())
		assert.Equal(t, 200, productCache.(*V1).SizeInMb())
		assert.Equal(t, 50, sessionCache.(*V1).SizeInMb())
	})

	t.Run("caches work independently with different data", func(t *testing.T) {
		resetCacheInstances()

		cacheDetails := []InMemoryCacheDetail{
			{Name: "cache-a", MemorySizeInMb: 10},
			{Name: "cache-b", MemorySizeInMb: 10},
		}

		InitMultiInMemoryCacheWithConf(cacheDetails)

		cacheA, err := InstanceByName("cache-a")
		require.NoError(t, err)

		cacheB, err := InstanceByName("cache-b")
		require.NoError(t, err)

		// Set different data in each cache
		keyA := []byte("key-a")
		valueA := []byte("value-a")
		err = cacheA.Set(keyA, valueA)
		require.NoError(t, err)

		keyB := []byte("key-b")
		valueB := []byte("value-b")
		err = cacheB.Set(keyB, valueB)
		require.NoError(t, err)

		// Verify data is isolated
		retrievedA, err := cacheA.Get(keyA)
		require.NoError(t, err)
		assert.Equal(t, valueA, retrievedA)

		retrievedB, err := cacheB.Get(keyB)
		require.NoError(t, err)
		assert.Equal(t, valueB, retrievedB)

		// Verify cross-cache isolation
		_, err = cacheA.Get(keyB)
		assert.Error(t, err, "cache-a should not have cache-b's data")

		_, err = cacheB.Get(keyA)
		assert.Error(t, err, "cache-b should not have cache-a's data")
	})

	t.Run("initializes single cache with config", func(t *testing.T) {
		resetCacheInstances()

		cacheDetails := []InMemoryCacheDetail{
			{Name: "single-cache", MemorySizeInMb: 150},
		}

		InitMultiInMemoryCacheWithConf(cacheDetails)

		cache, err := InstanceByName("single-cache")
		require.NoError(t, err)
		require.NotNil(t, cache)
		assert.Equal(t, 150, cache.(*V1).SizeInMb())
	})

	t.Run("handles empty cache details list", func(t *testing.T) {
		resetCacheInstances()

		cacheDetails := []InMemoryCacheDetail{}

		// Should not panic with empty list
		InitMultiInMemoryCacheWithConf(cacheDetails)

		// namedInstances should be initialized but empty
		assert.NotNil(t, namedInstances)
	})

	t.Run("panics with duplicate cache names", func(t *testing.T) {
		resetCacheInstances()

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("InitMultiInMemoryCacheWithConf should panic with duplicate cache names")
			}
		}()

		cacheDetails := []InMemoryCacheDetail{
			{Name: "duplicate-cache", MemorySizeInMb: 100},
			{Name: "duplicate-cache", MemorySizeInMb: 200},
		}

		InitMultiInMemoryCacheWithConf(cacheDetails)
	})

	t.Run("panics when called twice", func(t *testing.T) {
		resetCacheInstances()

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("InitMultiInMemoryCacheWithConf should panic when called twice")
			}
		}()

		cacheDetails1 := []InMemoryCacheDetail{
			{Name: "cache1", MemorySizeInMb: 100},
		}

		cacheDetails2 := []InMemoryCacheDetail{
			{Name: "cache2", MemorySizeInMb: 200},
		}

		InitMultiInMemoryCacheWithConf(cacheDetails1)

		// Second call should panic due to namedInstances != nil check
		InitMultiInMemoryCacheWithConf(cacheDetails2)
	})

	t.Run("panics if namedInstances already initialized", func(t *testing.T) {
		resetCacheInstances()

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("InitMultiInMemoryCacheWithConf should panic when namedInstances is already initialized")
			}
		}()

		// Initialize once
		cacheDetails1 := []InMemoryCacheDetail{
			{Name: "cache1", MemorySizeInMb: 100},
		}
		InitMultiInMemoryCacheWithConf(cacheDetails1)

		// Reset sync.Once but keep namedInstances initialized to simulate the check
		cacheOnce = sync.Once{}

		// Try to initialize again - should panic due to namedInstances != nil check
		cacheDetails2 := []InMemoryCacheDetail{
			{Name: "cache2", MemorySizeInMb: 200},
		}
		InitMultiInMemoryCacheWithConf(cacheDetails2)
	})

	t.Run("panics with zero cache size", func(t *testing.T) {
		resetCacheInstances()

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("InitMultiInMemoryCacheWithConf should panic with zero cache size")
			}
		}()

		cacheDetails := []InMemoryCacheDetail{
			{Name: "zero-cache", MemorySizeInMb: 0},
		}

		InitMultiInMemoryCacheWithConf(cacheDetails)
	})

	t.Run("panics with negative cache size", func(t *testing.T) {
		resetCacheInstances()

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("InitMultiInMemoryCacheWithConf should panic with negative cache size")
			}
		}()

		cacheDetails := []InMemoryCacheDetail{
			{Name: "negative-cache", MemorySizeInMb: -100},
		}

		InitMultiInMemoryCacheWithConf(cacheDetails)
	})

	t.Run("panics with empty cache name", func(t *testing.T) {
		resetCacheInstances()

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("InitMultiInMemoryCacheWithConf should panic with empty cache name")
			}
		}()

		cacheDetails := []InMemoryCacheDetail{
			{Name: "", MemorySizeInMb: 100},
		}

		InitMultiInMemoryCacheWithConf(cacheDetails)
	})
}

// TestInitMultiInMemoryCacheWithConf_Integration tests real-world usage scenarios
func TestInitMultiInMemoryCacheWithConf_Integration(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	t.Run("simulates microservice with multiple cache types", func(t *testing.T) {
		resetCacheInstances()

		// Typical microservice cache configuration
		cacheDetails := []InMemoryCacheDetail{
			{Name: "auth-token-cache", MemorySizeInMb: 50},    // Small cache for auth tokens
			{Name: "user-profile-cache", MemorySizeInMb: 200}, // Medium cache for user profiles
			{Name: "product-cache", MemorySizeInMb: 500},      // Large cache for product data
			{Name: "api-response-cache", MemorySizeInMb: 100}, // Cache for API responses
		}

		InitMultiInMemoryCacheWithConf(cacheDetails)

		// Test auth token cache
		authCache, err := InstanceByName("auth-token-cache")
		require.NoError(t, err)
		err = authCache.SetEx([]byte("token:abc123"), []byte(`{"user_id":1}`), 3600)
		require.NoError(t, err)

		// Test user profile cache
		userCache, err := InstanceByName("user-profile-cache")
		require.NoError(t, err)
		err = userCache.Set([]byte("user:1"), []byte(`{"name":"Alice","email":"alice@example.com"}`))
		require.NoError(t, err)

		// Test product cache
		productCache, err := InstanceByName("product-cache")
		require.NoError(t, err)
		err = productCache.Set([]byte("product:123"), []byte(`{"id":123,"name":"Laptop","price":999.99}`))
		require.NoError(t, err)

		// Test API response cache
		apiCache, err := InstanceByName("api-response-cache")
		require.NoError(t, err)
		err = apiCache.SetEx([]byte("api:/users/1"), []byte(`{"status":"success"}`), 300)
		require.NoError(t, err)

		// Verify all data is accessible
		token, err := authCache.Get([]byte("token:abc123"))
		require.NoError(t, err)
		assert.Contains(t, string(token), "user_id")

		profile, err := userCache.Get([]byte("user:1"))
		require.NoError(t, err)
		assert.Contains(t, string(profile), "Alice")

		product, err := productCache.Get([]byte("product:123"))
		require.NoError(t, err)
		assert.Contains(t, string(product), "Laptop")

		apiResp, err := apiCache.Get([]byte("api:/users/1"))
		require.NoError(t, err)
		assert.Contains(t, string(apiResp), "success")
	})

	t.Run("handles large number of caches", func(t *testing.T) {
		resetCacheInstances()

		// Create 20 caches
		cacheDetails := make([]InMemoryCacheDetail, 20)
		for i := 0; i < 20; i++ {
			cacheDetails[i] = InMemoryCacheDetail{
				Name:           "cache-" + string(rune('a'+i)),
				MemorySizeInMb: 10 + i*5, // Varying sizes
			}
		}

		InitMultiInMemoryCacheWithConf(cacheDetails)

		// Verify a few random caches
		cache0, err := InstanceByName("cache-a")
		require.NoError(t, err)
		require.NotNil(t, cache0)

		cache10, err := InstanceByName("cache-k")
		require.NoError(t, err)
		require.NotNil(t, cache10)

		cache19, err := InstanceByName("cache-t")
		require.NoError(t, err)
		require.NotNil(t, cache19)
	})
}

// TestInstanceByName_WithConf tests InstanceByName after InitMultiInMemoryCacheWithConf
func TestInstanceByName_WithConf(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	t.Run("returns error for non-existent cache", func(t *testing.T) {
		resetCacheInstances()

		cacheDetails := []InMemoryCacheDetail{
			{Name: "existing-cache", MemorySizeInMb: 100},
		}

		InitMultiInMemoryCacheWithConf(cacheDetails)

		cache, err := InstanceByName("non-existent-cache")

		assert.Error(t, err)
		assert.Nil(t, cache)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("returns correct cache for existing name", func(t *testing.T) {
		resetCacheInstances()

		cacheDetails := []InMemoryCacheDetail{
			{Name: "my-cache", MemorySizeInMb: 100},
		}

		InitMultiInMemoryCacheWithConf(cacheDetails)

		cache, err := InstanceByName("my-cache")

		require.NoError(t, err)
		require.NotNil(t, cache)

		// Verify it's functional
		err = cache.Set([]byte("test"), []byte("value"))
		require.NoError(t, err)
	})
}

// TestInitMultiInMemoryCache_vs_InitMultiInMemoryCacheWithConf ensures they don't conflict
func TestInitMultiInMemoryCache_vs_InitMultiInMemoryCacheWithConf(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	t.Run("panics when trying to use both initialization methods", func(t *testing.T) {
		resetCacheInstances()

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("InitMultiInMemoryCacheWithConf should panic when namedInstances already initialized")
			}
		}()

		os.Setenv("IN_MEMORY_CACHE_SIZE_IN_BYTES", "1048576")
		defer os.Unsetenv("IN_MEMORY_CACHE_SIZE_IN_BYTES")
		viper.AutomaticEnv()

		// Initialize with env-based method
		InitMultiInMemoryCache([]string{"cache1"})

		// Try to initialize with config-based method - should panic
		cacheDetails := []InMemoryCacheDetail{
			{Name: "cache2", MemorySizeInMb: 100},
		}
		InitMultiInMemoryCacheWithConf(cacheDetails)
	})

	t.Run("panics when calling InitMultiInMemoryCache if already initialized via InitMultiInMemoryCacheWithConf", func(t *testing.T) {
		resetCacheInstances()

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("InitMultiInMemoryCache should panic when namedInstances is already initialized")
			}
		}()

		os.Setenv("IN_MEMORY_CACHE_SIZE_IN_BYTES", "1048576")
		defer os.Unsetenv("IN_MEMORY_CACHE_SIZE_IN_BYTES")
		viper.AutomaticEnv()

		// Initialize with config-based method first
		cacheDetails := []InMemoryCacheDetail{
			{Name: "cache1", MemorySizeInMb: 100},
		}
		InitMultiInMemoryCacheWithConf(cacheDetails)

		// Reset sync.Once but keep namedInstances
		cacheOnce = sync.Once{}

		// Try to use env-based method - should panic
		InitMultiInMemoryCache([]string{"cache2"})
	})
}

// TestInMemoryCacheDetail_FieldNames tests the struct field names
func TestInMemoryCacheDetail_FieldNames(t *testing.T) {
	t.Run("uses correct field names", func(t *testing.T) {
		detail := InMemoryCacheDetail{
			Name:           "test-cache",
			MemorySizeInMb: 100,
		}

		// Verify field values
		assert.Equal(t, "test-cache", detail.Name)
		assert.Equal(t, 100, detail.MemorySizeInMb)
	})

	t.Run("can create array of cache details", func(t *testing.T) {
		details := []InMemoryCacheDetail{
			{Name: "cache1", MemorySizeInMb: 10},
			{Name: "cache2", MemorySizeInMb: 20},
			{Name: "cache3", MemorySizeInMb: 30},
		}

		assert.Len(t, details, 3)
		for i, detail := range details {
			assert.NotEmpty(t, detail.Name)
			assert.Equal(t, (i+1)*10, detail.MemorySizeInMb)
		}
	})
}

// TestInitMultiInMemoryCache_AdditionalCases tests additional scenarios for InitMultiInMemoryCache
func TestInitMultiInMemoryCache_AdditionalCases(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	t.Run("panics if namedInstances already initialized", func(t *testing.T) {
		resetCacheInstances()

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("InitMultiInMemoryCache should panic when namedInstances already initialized")
			}
		}()

		os.Setenv("IN_MEMORY_CACHE_SIZE_IN_BYTES", "10485760")
		defer os.Unsetenv("IN_MEMORY_CACHE_SIZE_IN_BYTES")
		viper.AutomaticEnv()

		// Initialize first
		InitMultiInMemoryCache([]string{"cache1"})

		// Reset sync.Once but keep namedInstances to trigger the check
		cacheOnce = sync.Once{}

		// Try to initialize again - should panic at the namedInstances != nil check
		InitMultiInMemoryCache([]string{"cache2"})
	})

	t.Run("initializes single cache with env config", func(t *testing.T) {
		resetCacheInstances()

		os.Setenv("IN_MEMORY_CACHE_SIZE_IN_BYTES", "52428800") // 50MB
		defer os.Unsetenv("IN_MEMORY_CACHE_SIZE_IN_BYTES")
		viper.AutomaticEnv()

		InitMultiInMemoryCache([]string{"single-cache"})

		cache, err := InstanceByName("single-cache")
		require.NoError(t, err)
		require.NotNil(t, cache)

		// Verify cache works
		err = cache.Set([]byte("test"), []byte("data"))
		require.NoError(t, err)

		data, err := cache.Get([]byte("test"))
		require.NoError(t, err)
		assert.Equal(t, []byte("data"), data)
	})

	t.Run("each cache gets same size from env", func(t *testing.T) {
		resetCacheInstances()

		os.Setenv("IN_MEMORY_CACHE_SIZE_IN_BYTES", "20971520") // 20MB
		defer os.Unsetenv("IN_MEMORY_CACHE_SIZE_IN_BYTES")
		viper.AutomaticEnv()

		cacheNames := []string{"cache-1", "cache-2", "cache-3"}
		InitMultiInMemoryCache(cacheNames)

		// All caches should have same size from env
		for _, name := range cacheNames {
			cache, err := InstanceByName(name)
			require.NoError(t, err)
			assert.Equal(t, 20, cache.(*V1).SizeInMb())
		}
	})

	t.Run("initializes caches with special character names", func(t *testing.T) {
		resetCacheInstances()

		os.Setenv("IN_MEMORY_CACHE_SIZE_IN_BYTES", "10485760")
		defer os.Unsetenv("IN_MEMORY_CACHE_SIZE_IN_BYTES")
		viper.AutomaticEnv()

		cacheNames := []string{"cache-with-dash", "cache_with_underscore", "cache123"}
		InitMultiInMemoryCache(cacheNames)

		for _, name := range cacheNames {
			cache, err := InstanceByName(name)
			require.NoError(t, err)
			require.NotNil(t, cache)
		}
	})
}

// TestInitMultiInMemoryCacheWithConf_AdditionalCases tests additional scenarios
func TestInitMultiInMemoryCacheWithConf_AdditionalCases(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	t.Run("initializes with varying cache sizes", func(t *testing.T) {
		resetCacheInstances()

		cacheDetails := []InMemoryCacheDetail{
			{Name: "tiny", MemorySizeInMb: 1},
			{Name: "small", MemorySizeInMb: 10},
			{Name: "medium", MemorySizeInMb: 100},
			{Name: "large", MemorySizeInMb: 1000},
		}

		InitMultiInMemoryCacheWithConf(cacheDetails)

		// Verify sizes
		tiny, _ := InstanceByName("tiny")
		assert.Equal(t, 1, tiny.(*V1).SizeInMb())

		small, _ := InstanceByName("small")
		assert.Equal(t, 10, small.(*V1).SizeInMb())

		medium, _ := InstanceByName("medium")
		assert.Equal(t, 100, medium.(*V1).SizeInMb())

		large, _ := InstanceByName("large")
		assert.Equal(t, 1000, large.(*V1).SizeInMb())
	})

	t.Run("caches maintain independent state", func(t *testing.T) {
		resetCacheInstances()

		cacheDetails := []InMemoryCacheDetail{
			{Name: "cache-x", MemorySizeInMb: 10},
			{Name: "cache-y", MemorySizeInMb: 10},
		}

		InitMultiInMemoryCacheWithConf(cacheDetails)

		cacheX, _ := InstanceByName("cache-x")
		cacheY, _ := InstanceByName("cache-y")

		// Fill cache-x
		for i := 0; i < 100; i++ {
			key := []byte("key-x-" + string(rune('0'+i%10)))
			value := []byte("value-x-" + string(rune('0'+i%10)))
			cacheX.Set(key, value)
		}

		// cache-y should be empty
		_, err := cacheY.Get([]byte("key-x-0"))
		assert.Error(t, err, "cache-y should not have cache-x's data")

		// Add data to cache-y
		cacheY.Set([]byte("key-y"), []byte("value-y"))

		// Verify cache-y data
		value, err := cacheY.Get([]byte("key-y"))
		require.NoError(t, err)
		assert.Equal(t, []byte("value-y"), value)

		// cache-x should not have cache-y's data
		_, err = cacheX.Get([]byte("key-y"))
		assert.Error(t, err)
	})

	t.Run("works with single cache detail", func(t *testing.T) {
		resetCacheInstances()

		cacheDetails := []InMemoryCacheDetail{
			{Name: "solo-cache", MemorySizeInMb: 50},
		}

		InitMultiInMemoryCacheWithConf(cacheDetails)

		cache, err := InstanceByName("solo-cache")
		require.NoError(t, err)
		assert.Equal(t, 50, cache.(*V1).SizeInMb())
	})

	t.Run("all caches support all operations", func(t *testing.T) {
		resetCacheInstances()

		cacheDetails := []InMemoryCacheDetail{
			{Name: "ops-cache-1", MemorySizeInMb: 10},
			{Name: "ops-cache-2", MemorySizeInMb: 20},
		}

		InitMultiInMemoryCacheWithConf(cacheDetails)

		for _, detail := range cacheDetails {
			cache, err := InstanceByName(detail.Name)
			require.NoError(t, err)

			// Test Set
			err = cache.Set([]byte("key"), []byte("value"))
			require.NoError(t, err)

			// Test Get
			value, err := cache.Get([]byte("key"))
			require.NoError(t, err)
			assert.Equal(t, []byte("value"), value)

			// Test SetEx
			err = cache.SetEx([]byte("key-ex"), []byte("value-ex"), 10)
			require.NoError(t, err)

			// Test Delete
			deleted := cache.Delete([]byte("key"))
			assert.True(t, deleted)

			// Verify deleted
			_, err = cache.Get([]byte("key"))
			assert.Error(t, err)
		}
	})
}
