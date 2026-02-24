package inmemorycache

import (
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

func TestInitPanicWhenEnvVarNotSet(t *testing.T) {
	// Ensure environment variables are not set
	os.Unsetenv("IN_MEMORY_CACHE_SIZE_IN_BYTES")
	os.Unsetenv("APP_GC_PERCENTAGE")
	viper.AutomaticEnv()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Init should panic when IN_MEMORY_CACHE_SIZE_IN_BYTES is not set")
		}
	}()
	_ = newV1InMemoryCache("test-cache")
}

func TestGetInstance(t *testing.T) {
	os.Setenv("IN_MEMORY_CACHE_SIZE_IN_BYTES", "1024") // 1KB
	os.Setenv("APP_GC_PERCENTAGE", "20")
	viper.AutomaticEnv()
	// Test Getting Cache Instance
	t.Run("Get cache instance", func(t *testing.T) {
		cache := newV1InMemoryCache("test-cache")
		require.NotNil(t, cache, "Cache instance should not be nil")
		//TODO: Add more assertions for new version of in-memory cache
	})

	// Clear environment variables after test
	os.Unsetenv("IN_MEMORY_CACHE_SIZE_IN_BYTES")
	os.Unsetenv("APP_GC_PERCENTAGE")
	viper.AutomaticEnv()
}

func TestSetAndGet(t *testing.T) {
	os.Setenv("IN_MEMORY_CACHE_SIZE_IN_BYTES", "1024") // 1KB
	os.Setenv("APP_GC_PERCENTAGE", "20")
	viper.AutomaticEnv()
	// Test Getting Cache Instance
	t.Run("Get cache instance", func(t *testing.T) {
		cache := newV1InMemoryCache("test-cache")
		require.NotNil(t, cache, "Cache instance should not be nil")
		key := []byte("key")
		val := []byte("value")
		err := cache.Set(key, val)
		require.Nil(t, err, "err should be nil on set")
		time.Sleep(5 * time.Second)
		testVal, err := cache.Get(key)
		require.Nil(t, err, "err should be nil on get")
		require.Equal(t, val, testVal, "value should be same as set value")
		err = cache.SetEx(key, val, 1)
		require.Nil(t, err, "err should be nil on set")
		time.Sleep(5 * time.Second)
		testVal, err = cache.Get(key)
		require.NotNil(t, err, "should get not found error")
		require.Nil(t, testVal, "value should be nil")
	})

	// Clear environment variables after test
	os.Unsetenv("IN_MEMORY_CACHE_SIZE_IN_BYTES")
	os.Unsetenv("APP_GC_PERCENTAGE")
	viper.AutomaticEnv()
}
