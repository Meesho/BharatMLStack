package inmemorycache

import (
	"os"
	"sync"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestEnv sets up common test environment variables
func setupTestEnv(t *testing.T) func() {
	os.Setenv("APP_GC_PERCENTAGE", "20")
	viper.AutomaticEnv()

	return func() {
		os.Unsetenv("APP_GC_PERCENTAGE")
		viper.AutomaticEnv()
	}
}

// TestNewV1InMemoryCacheWithConf tests the newV1InMemoryCacheWithConf function
func TestNewV1InMemoryCacheWithConf(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	t.Run("creates cache with valid configuration", func(t *testing.T) {
		cache := newV1InMemoryCacheWithConf("test-cache", 10)

		require.NotNil(t, cache)

		// Verify cache size
		v1Cache := cache.(*V1)
		assert.Equal(t, 10, v1Cache.SizeInMb())

		// Test basic operations
		key := []byte("key1")
		value := []byte("value1")
		err := cache.Set(key, value)
		require.NoError(t, err)

		retrievedValue, err := cache.Get(key)
		require.NoError(t, err)
		assert.Equal(t, value, retrievedValue)
	})

	t.Run("creates cache with different sizes", func(t *testing.T) {
		cache1 := newV1InMemoryCacheWithConf("cache-1mb", 1)
		cache100 := newV1InMemoryCacheWithConf("cache-100mb", 100)
		cache500 := newV1InMemoryCacheWithConf("cache-500mb", 500)

		assert.Equal(t, 1, cache1.(*V1).SizeInMb())
		assert.Equal(t, 100, cache100.(*V1).SizeInMb())
		assert.Equal(t, 500, cache500.(*V1).SizeInMb())
	})

	t.Run("panics with zero size", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("newV1InMemoryCacheWithConf should panic with zero size")
			}
		}()
		newV1InMemoryCacheWithConf("test-cache", 0)
	})

	t.Run("panics with negative size", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("newV1InMemoryCacheWithConf should panic with negative size")
			}
		}()
		newV1InMemoryCacheWithConf("test-cache", -10)
	})

	t.Run("panics with empty cache name", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("newV1InMemoryCacheWithConf should panic with empty cache name")
			}
		}()
		newV1InMemoryCacheWithConf("", 10)
	})
}

// TestV1_CacheOperations tests basic cache operations
func TestV1_CacheOperations(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	t.Run("set and get operations", func(t *testing.T) {
		cache := newV1InMemoryCacheWithConf("test-cache", 10)

		key := []byte("test-key")
		value := []byte("test-value")

		// Set
		err := cache.Set(key, value)
		require.NoError(t, err)

		// Get
		retrievedValue, err := cache.Get(key)
		require.NoError(t, err)
		assert.Equal(t, value, retrievedValue)
	})

	t.Run("setex operations with expiry", func(t *testing.T) {
		cache := newV1InMemoryCacheWithConf("test-cache", 10)

		key := []byte("test-key-ex")
		value := []byte("test-value-ex")

		// SetEx with 10 seconds expiry
		err := cache.SetEx(key, value, 10)
		require.NoError(t, err)

		// Should be retrievable immediately
		retrievedValue, err := cache.Get(key)
		require.NoError(t, err)
		assert.Equal(t, value, retrievedValue)
	})

	t.Run("delete operations", func(t *testing.T) {
		cache := newV1InMemoryCacheWithConf("test-cache", 10)

		key := []byte("test-key-delete")
		value := []byte("test-value-delete")

		// Set
		err := cache.Set(key, value)
		require.NoError(t, err)

		// Delete
		deleted := cache.Delete(key)
		assert.True(t, deleted)

		// Should not be retrievable
		_, err = cache.Get(key)
		assert.Error(t, err)
	})

	t.Run("delete non-existent key returns false", func(t *testing.T) {
		cache := newV1InMemoryCacheWithConf("test-cache", 10)

		deleted := cache.Delete([]byte("non-existent"))
		assert.False(t, deleted)
	})

	t.Run("get non-existent key returns error", func(t *testing.T) {
		cache := newV1InMemoryCacheWithConf("test-cache", 10)

		_, err := cache.Get([]byte("non-existent"))
		assert.Error(t, err)
	})

	t.Run("overwrite existing key", func(t *testing.T) {
		cache := newV1InMemoryCacheWithConf("test-cache", 10)

		key := []byte("key")
		value1 := []byte("value1")
		value2 := []byte("value2")

		// Set first value
		err := cache.Set(key, value1)
		require.NoError(t, err)

		// Overwrite with second value
		err = cache.Set(key, value2)
		require.NoError(t, err)

		// Should get the second value
		retrieved, err := cache.Get(key)
		require.NoError(t, err)
		assert.Equal(t, value2, retrieved)
	})

	t.Run("multiple keys in same cache", func(t *testing.T) {
		cache := newV1InMemoryCacheWithConf("test-cache", 10)

		// Set multiple keys
		for i := 0; i < 10; i++ {
			key := []byte("key-" + string(rune('0'+i)))
			value := []byte("value-" + string(rune('0'+i)))
			err := cache.Set(key, value)
			require.NoError(t, err)
		}

		// Verify all keys
		for i := 0; i < 10; i++ {
			key := []byte("key-" + string(rune('0'+i)))
			expectedValue := []byte("value-" + string(rune('0'+i)))
			value, err := cache.Get(key)
			require.NoError(t, err)
			assert.Equal(t, expectedValue, value)
		}
	})
}

// TestV1_SizeInMb tests the SizeInMb getter
func TestV1_SizeInMb(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	tests := []struct {
		name       string
		sizeMB     int
		expectedMB int
	}{
		{
			name:       "returns correct size for 1 MB",
			sizeMB:     1,
			expectedMB: 1,
		},
		{
			name:       "returns correct size for 100 MB",
			sizeMB:     100,
			expectedMB: 100,
		},
		{
			name:       "returns correct size for 500 MB",
			sizeMB:     500,
			expectedMB: 500,
		},
		{
			name:       "returns correct size for 1024 MB",
			sizeMB:     1024,
			expectedMB: 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := newV1InMemoryCacheWithConf("test-cache", tt.sizeMB)
			v1Cache := cache.(*V1)
			assert.Equal(t, tt.expectedMB, v1Cache.SizeInMb())
		})
	}
}

// TestV1_PublishMetric tests that publishMetric goroutine is started
func TestV1_PublishMetric(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	t.Run("publishMetric goroutine is started", func(t *testing.T) {
		// This test verifies the cache is created and goroutine starts without panic
		cache := newV1InMemoryCacheWithConf("metrics-cache", 10)

		require.NotNil(t, cache)

		// Perform some operations to ensure cache is working
		err := cache.Set([]byte("key"), []byte("value"))
		require.NoError(t, err)
	})
}

// TestInMemoryCacheDetail tests the InMemoryCacheDetail struct
func TestInMemoryCacheDetail(t *testing.T) {
	t.Run("creates valid cache detail", func(t *testing.T) {
		detail := InMemoryCacheDetail{
			Name:           "test-cache",
			MemorySizeInMb: 100,
		}

		assert.Equal(t, "test-cache", detail.Name)
		assert.Equal(t, 100, detail.MemorySizeInMb)
	})

	t.Run("creates multiple cache details", func(t *testing.T) {
		details := []InMemoryCacheDetail{
			{Name: "cache1", MemorySizeInMb: 100},
			{Name: "cache2", MemorySizeInMb: 200},
			{Name: "cache3", MemorySizeInMb: 300},
		}

		assert.Len(t, details, 3)
		assert.Equal(t, "cache1", details[0].Name)
		assert.Equal(t, 100, details[0].MemorySizeInMb)
		assert.Equal(t, "cache2", details[1].Name)
		assert.Equal(t, 200, details[1].MemorySizeInMb)
	})
}

// TestV1_CacheWithSpecialCharacters tests cache names with special characters
func TestV1_CacheWithSpecialCharacters(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	t.Run("cache name with hyphens and underscores", func(t *testing.T) {
		cache := newV1InMemoryCacheWithConf("cache-with_special-chars", 10)

		require.NotNil(t, cache)

		// Verify cache works
		err := cache.Set([]byte("key"), []byte("value"))
		require.NoError(t, err)

		value, err := cache.Get([]byte("key"))
		require.NoError(t, err)
		assert.Equal(t, []byte("value"), value)
	})

	t.Run("cache name with numbers", func(t *testing.T) {
		cache := newV1InMemoryCacheWithConf("cache123", 10)

		require.NotNil(t, cache)
		v1Cache := cache.(*V1)
		assert.Equal(t, "cache123", v1Cache.cacheName)
	})
}

// TestInit tests the Init function
func TestInit(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	t.Run("initializes cache with version 1", func(t *testing.T) {
		// Reset singleton
		instance = nil
		once = sync.Once{}

		os.Setenv("IN_MEMORY_CACHE_SIZE_IN_BYTES", "10485760") // 10MB
		defer os.Unsetenv("IN_MEMORY_CACHE_SIZE_IN_BYTES")
		viper.AutomaticEnv()

		Init(1)

		cache := Instance()
		require.NotNil(t, cache)

		// Verify it's functional
		err := cache.Set([]byte("key"), []byte("value"))
		require.NoError(t, err)

		value, err := cache.Get([]byte("key"))
		require.NoError(t, err)
		assert.Equal(t, []byte("value"), value)
	})

	t.Run("panics with invalid version", func(t *testing.T) {
		instance = nil
		once = sync.Once{}

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Init should panic with invalid version")
			}
		}()

		os.Setenv("IN_MEMORY_CACHE_SIZE_IN_BYTES", "10485760")
		defer os.Unsetenv("IN_MEMORY_CACHE_SIZE_IN_BYTES")
		viper.AutomaticEnv()

		Init(2) // Invalid version
	})

	t.Run("only initializes once with sync.Once", func(t *testing.T) {
		instance = nil
		once = sync.Once{}

		os.Setenv("IN_MEMORY_CACHE_SIZE_IN_BYTES", "10485760")
		defer os.Unsetenv("IN_MEMORY_CACHE_SIZE_IN_BYTES")
		viper.AutomaticEnv()

		Init(1)
		firstInstance := Instance()

		// Try to init again
		Init(1)
		secondInstance := Instance()

		// Should be same instance
		assert.Equal(t, firstInstance, secondInstance)
	})
}

// TestInitV1 tests the InitV1 convenience function
func TestInitV1(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	t.Run("initializes V1 cache", func(t *testing.T) {
		instance = nil
		once = sync.Once{}

		os.Setenv("IN_MEMORY_CACHE_SIZE_IN_BYTES", "10485760")
		defer os.Unsetenv("IN_MEMORY_CACHE_SIZE_IN_BYTES")
		viper.AutomaticEnv()

		InitV1()

		cache := Instance()
		require.NotNil(t, cache)

		// Verify cache is functional
		err := cache.Set([]byte("test"), []byte("data"))
		require.NoError(t, err)

		data, err := cache.Get([]byte("test"))
		require.NoError(t, err)
		assert.Equal(t, []byte("data"), data)
	})
}

// TestInstance tests the Instance function
func TestInstance(t *testing.T) {
	t.Run("panics when not initialized", func(t *testing.T) {
		instance = nil
		once = sync.Once{}

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Instance should panic when not initialized")
			}
		}()

		Instance()
	})

	t.Run("returns initialized cache", func(t *testing.T) {
		cleanup := setupTestEnv(t)
		defer cleanup()

		instance = nil
		once = sync.Once{}

		os.Setenv("IN_MEMORY_CACHE_SIZE_IN_BYTES", "10485760")
		defer os.Unsetenv("IN_MEMORY_CACHE_SIZE_IN_BYTES")
		viper.AutomaticEnv()

		InitV1()

		cache := Instance()
		require.NotNil(t, cache)

		// Multiple calls return same instance
		cache2 := Instance()
		assert.Equal(t, cache, cache2)
	})
}

// TestSetMockInstance tests the SetMockInstance function
func TestSetMockInstance(t *testing.T) {
	t.Run("sets mock instance", func(t *testing.T) {
		instance = nil

		mockCache := &MockInMemoryCacheClient{}
		mockCache.On("Set", []byte("key"), []byte("value")).Return(nil)
		mockCache.On("Get", []byte("key")).Return([]byte("value"), nil)

		SetMockInstance(mockCache)

		cache := Instance()
		require.NotNil(t, cache)

		// Verify mock is used
		err := cache.Set([]byte("key"), []byte("value"))
		require.NoError(t, err)

		value, err := cache.Get([]byte("key"))
		require.NoError(t, err)
		assert.Equal(t, []byte("value"), value)

		mockCache.AssertExpectations(t)
	})

	t.Run("replaces existing instance with mock", func(t *testing.T) {
		cleanup := setupTestEnv(t)
		defer cleanup()

		instance = nil
		once = sync.Once{}

		os.Setenv("IN_MEMORY_CACHE_SIZE_IN_BYTES", "10485760")
		defer os.Unsetenv("IN_MEMORY_CACHE_SIZE_IN_BYTES")
		viper.AutomaticEnv()

		// Initialize real cache
		InitV1()
		realCache := Instance()
		require.NotNil(t, realCache)

		// Replace with mock
		mockCache := &MockInMemoryCacheClient{}
		mockCache.On("Set", []byte("mock-key"), []byte("mock-value")).Return(nil)

		SetMockInstance(mockCache)

		// Should now use mock
		cache := Instance()
		assert.NotEqual(t, realCache, cache)

		err := cache.Set([]byte("mock-key"), []byte("mock-value"))
		require.NoError(t, err)

		mockCache.AssertExpectations(t)
	})
}

// TestMockInMemoryCacheClient tests all mock methods
func TestMockInMemoryCacheClient(t *testing.T) {
	t.Run("mock Get returns value", func(t *testing.T) {
		mockCache := &MockInMemoryCacheClient{}
		key := []byte("test-key")
		expectedValue := []byte("test-value")

		mockCache.On("Get", key).Return(expectedValue, nil)

		value, err := mockCache.Get(key)

		require.NoError(t, err)
		assert.Equal(t, expectedValue, value)
		mockCache.AssertExpectations(t)
	})

	t.Run("mock Get returns error", func(t *testing.T) {
		mockCache := &MockInMemoryCacheClient{}
		key := []byte("missing-key")

		mockCache.On("Get", key).Return(nil, assert.AnError)

		value, err := mockCache.Get(key)

		require.Error(t, err)
		assert.Nil(t, value)
		mockCache.AssertExpectations(t)
	})

	t.Run("mock Set succeeds", func(t *testing.T) {
		mockCache := &MockInMemoryCacheClient{}
		key := []byte("key")
		value := []byte("value")

		mockCache.On("Set", key, value).Return(nil)

		err := mockCache.Set(key, value)

		require.NoError(t, err)
		mockCache.AssertExpectations(t)
	})

	t.Run("mock Set returns error", func(t *testing.T) {
		mockCache := &MockInMemoryCacheClient{}
		key := []byte("key")
		value := []byte("value")

		mockCache.On("Set", key, value).Return(assert.AnError)

		err := mockCache.Set(key, value)

		require.Error(t, err)
		mockCache.AssertExpectations(t)
	})

	t.Run("mock SetEx succeeds", func(t *testing.T) {
		mockCache := &MockInMemoryCacheClient{}
		key := []byte("key")
		value := []byte("value")
		expiry := 300

		mockCache.On("SetEx", key, value, expiry).Return(nil)

		err := mockCache.SetEx(key, value, expiry)

		require.NoError(t, err)
		mockCache.AssertExpectations(t)
	})

	t.Run("mock SetEx returns error", func(t *testing.T) {
		mockCache := &MockInMemoryCacheClient{}
		key := []byte("key")
		value := []byte("value")
		expiry := 300

		mockCache.On("SetEx", key, value, expiry).Return(assert.AnError)

		err := mockCache.SetEx(key, value, expiry)

		require.Error(t, err)
		mockCache.AssertExpectations(t)
	})

	t.Run("mock Delete returns true", func(t *testing.T) {
		mockCache := &MockInMemoryCacheClient{}
		key := []byte("key")

		mockCache.On("Delete", key).Return(true)

		deleted := mockCache.Delete(key)

		assert.True(t, deleted)
		mockCache.AssertExpectations(t)
	})

	t.Run("mock Delete returns false", func(t *testing.T) {
		mockCache := &MockInMemoryCacheClient{}
		key := []byte("non-existent")

		mockCache.On("Delete", key).Return(false)

		deleted := mockCache.Delete(key)

		assert.False(t, deleted)
		mockCache.AssertExpectations(t)
	})

	t.Run("mock supports all InMemoryCache interface methods", func(t *testing.T) {
		mockCache := &MockInMemoryCacheClient{}

		// Setup all mock calls
		mockCache.On("Set", []byte("k1"), []byte("v1")).Return(nil)
		mockCache.On("Get", []byte("k1")).Return([]byte("v1"), nil)
		mockCache.On("SetEx", []byte("k2"), []byte("v2"), 60).Return(nil)
		mockCache.On("Delete", []byte("k1")).Return(true)

		// Use all methods
		err := mockCache.Set([]byte("k1"), []byte("v1"))
		require.NoError(t, err)

		value, err := mockCache.Get([]byte("k1"))
		require.NoError(t, err)
		assert.Equal(t, []byte("v1"), value)

		err = mockCache.SetEx([]byte("k2"), []byte("v2"), 60)
		require.NoError(t, err)

		deleted := mockCache.Delete([]byte("k1"))
		assert.True(t, deleted)

		mockCache.AssertExpectations(t)
	})
}

// TestV1Builder_Build_ErrorPaths tests error returns from build method
func TestV1Builder_Build_ErrorPaths(t *testing.T) {
	t.Run("returns error when cache name is empty", func(t *testing.T) {
		// Create builder and manually set cacheName to empty
		builder := &V1Builder{
			v1: &V1{
				cacheName: "",
				sizeInMb:  100,
			},
		}

		cache, err := builder.build()

		require.Error(t, err)
		assert.Nil(t, cache)
		assert.Contains(t, err.Error(), "cache name is required")
	})

	t.Run("returns error when size is zero", func(t *testing.T) {
		builder := &V1Builder{
			v1: &V1{
				cacheName: "test-cache",
				sizeInMb:  0,
			},
		}

		cache, err := builder.build()

		require.Error(t, err)
		assert.Nil(t, cache)
		assert.Contains(t, err.Error(), "invalid cache size")
		assert.Contains(t, err.Error(), "0 MB")
	})

	t.Run("returns error when size is negative", func(t *testing.T) {
		builder := &V1Builder{
			v1: &V1{
				cacheName: "test-cache",
				sizeInMb:  -50,
			},
		}

		cache, err := builder.build()

		require.Error(t, err)
		assert.Nil(t, cache)
		assert.Contains(t, err.Error(), "invalid cache size")
		assert.Contains(t, err.Error(), "-50 MB")
	})

	t.Run("returns error when both name and size are invalid", func(t *testing.T) {
		builder := &V1Builder{
			v1: &V1{
				cacheName: "",
				sizeInMb:  0,
			},
		}

		cache, err := builder.build()

		require.Error(t, err)
		assert.Nil(t, cache)
		// Should fail on name check first
		assert.Contains(t, err.Error(), "cache name is required")
	})

	t.Run("succeeds with valid configuration", func(t *testing.T) {
		builder := &V1Builder{
			v1: &V1{
				cacheName: "valid-cache",
				sizeInMb:  100,
			},
		}

		cache, err := builder.build()

		require.NoError(t, err)
		require.NotNil(t, cache)
		assert.Equal(t, "valid-cache", cache.cacheName)
		assert.Equal(t, 100, cache.sizeInMb)
		assert.NotNil(t, cache.inMemCache)
	})
}
