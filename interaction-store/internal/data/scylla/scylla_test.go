package scylla

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetRetrieveQueryCacheKey(t *testing.T) {
	keyspace := "ks"
	tableName := "t1"
	columns := []string{"week_0", "week_1"}

	key := getRetrieveQueryCacheKey(keyspace, tableName, columns)

	assert.Contains(t, key, keyspace)
	assert.Contains(t, key, tableName)
	assert.Contains(t, key, "week_0,week_1")
	assert.Contains(t, key, "retrieve")
	assert.Equal(t, "ks_t1_week_0,week_1_retrieve", key)
}

func TestGetRetrieveQueryCacheKey_EmptyColumns(t *testing.T) {
	key := getRetrieveQueryCacheKey("ks", "t1", nil)
	assert.Equal(t, "ks_t1__retrieve", key)
}

func TestGetUpdateQueryCacheKey(t *testing.T) {
	keyspace := "ks"
	tableName := "t1"
	columns := []string{"week_0", "week_1"}

	key := getUpdateQueryCacheKey(keyspace, tableName, columns)

	assert.Contains(t, key, keyspace)
	assert.Contains(t, key, tableName)
	assert.Contains(t, key, "week_0,week_1")
	assert.Contains(t, key, "update")
	assert.Equal(t, "ks_t1_week_0,week_1_update", key)
}

func TestGetUpdateQueryCacheKey_EmptyColumns(t *testing.T) {
	key := getUpdateQueryCacheKey("ks", "t1", []string{})
	assert.Equal(t, "ks_t1__update", key)
}

func TestGetRetrieveQueryCacheKey_Deterministic(t *testing.T) {
	k1 := getRetrieveQueryCacheKey("ks", "t1", []string{"a", "b"})
	k2 := getRetrieveQueryCacheKey("ks", "t1", []string{"a", "b"})
	assert.Equal(t, k1, k2)
}

func TestGetUpdateQueryCacheKey_Deterministic(t *testing.T) {
	k1 := getUpdateQueryCacheKey("ks", "t1", []string{"a", "b"})
	k2 := getUpdateQueryCacheKey("ks", "t1", []string{"a", "b"})
	assert.Equal(t, k1, k2)
}
