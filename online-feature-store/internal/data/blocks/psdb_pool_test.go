package blocks

import (
	"testing"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/compression"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/system"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestGetPSDBPool(t *testing.T) {
	system.Init()
	// Get the pool instance
	psdbPool := GetPSDBPool()
	assert.NotNil(t, psdbPool, "PSDB pool should not be nil")

	// Get a PSDB from the pool
	psdb1 := psdbPool.Get()
	assert.NotNil(t, psdb1, "PSDB should not be nil")
	assert.NotNil(t, psdb1.Builder, "PSDB builder should not be nil")

	// Use the PSDB with correct types
	psdb1, err := psdb1.Builder.
		SetID(uint(1)).
		SetDataType(types.DataTypeInt64).
		SetCompressionB(compression.TypeZSTD).
		SetTTL(uint64(3600)).
		SetVersion(uint32(1)).
		SetScalarValues([]int64{1, 2, 3}, 3).
		Build()
	if err != nil {
		t.Fatalf("Failed to build PSDB: %v", err)
	}
	// Serialize to ensure it works
	_, err = psdb1.Serialize()
	assert.NoError(t, err, "Serialization should succeed")

	// Get another PSDB to verify pool is working
	psdb2 := psdbPool.Get()
	assert.NotNil(t, psdb2, "Second PSDB should not be nil")
	assert.NotEqual(t, psdb1, psdb2, "Different PSDBs should be returned")

	// Put first PSDB back and verify it's cleared
	psdbPool.Put(psdb1)
	assert.Zero(t, len(psdb1.originalData), "Data should be cleared")
	assert.Zero(t, len(psdb1.compressedData), "Compressed data should be cleared")
	assert.Zero(t, psdb1.originalDataLen, "Data length should be cleared")
	assert.Zero(t, psdb1.compressedDataLen, "Compressed data length should be cleared")
	assert.Equal(t, types.DataTypeUnknown, psdb1.dataType, "Data type should be cleared")
	assert.Zero(t, psdb1.layoutVersion, "ID should be cleared")
	assert.Zero(t, psdb1.featureSchemaVersion, "Version should be cleared")
	assert.Zero(t, psdb1.expiryAt, "TTL should be cleared")
	assert.Equal(t, compression.TypeNone, psdb1.compressionType, "Compression type should be cleared")
	assert.Equal(t, PSDBLayout1LengthBytes, len(psdb1.buf), "Buffer should be reset to prefix length")
	assert.Nil(t, psdb1.Data, "Data should be nil")

	// Put second PSDB back
	psdbPool.Put(psdb2)

	// Get a PSDB again and verify it's one of the cleared ones
	psdb3 := psdbPool.Get()
	assert.NotNil(t, psdb3, "Third PSDB should not be nil")
	assert.True(t, psdb3 == psdb1 || psdb3 == psdb2, "Pool should reuse cleared PSDBs")
}
