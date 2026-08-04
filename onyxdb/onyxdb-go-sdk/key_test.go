package sdk

import (
	"encoding/hex"
	"encoding/json"
	"hash/crc32"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testVector struct {
	EntityLabel    string  `json:"entity_label"`
	PKValues       []int64 `json:"pk_values"`
	ExpectedUTF8   string  `json:"expected_key_utf8"`
	ExpectedHex    string  `json:"expected_key_hex"`
	ExpectedShard3 uint32  `json:"expected_shard_3"`
	ExpectedShard64 uint32 `json:"expected_shard_64"`
}

// TestGoldenVectors verifies that BuildStringKey + crc32 shard assignment in Go
// matches the Python producer and Rust read server exactly. The vectors are shared
// across all three languages via key_test_vectors.json.
func TestGoldenVectors(t *testing.T) {
	data, err := os.ReadFile("key_test_vectors.json")
	require.NoError(t, err, "key_test_vectors.json must exist alongside the Go tests")

	var vectors []testVector
	require.NoError(t, json.Unmarshal(data, &vectors))
	require.NotEmpty(t, vectors)

	for _, v := range vectors {
		t.Run(v.ExpectedUTF8, func(t *testing.T) {
			key := BuildStringKey(v.EntityLabel, v.PKValues...)
			keyStr := string(key)
			keyHex := hex.EncodeToString(key)

			assert.Equal(t, v.ExpectedUTF8, keyStr, "UTF-8 key mismatch")
			assert.Equal(t, v.ExpectedHex, keyHex, "hex key mismatch")

			crc := crc32.ChecksumIEEE(key)
			assert.Equal(t, v.ExpectedShard3, crc%3, "shard%%3 mismatch for %s", keyStr)
			assert.Equal(t, v.ExpectedShard64, crc%64, "shard%%64 mismatch for %s", keyStr)
		})
	}
}
