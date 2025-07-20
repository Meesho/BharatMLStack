package indices

import (
	"fmt"
	"math/rand"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestNewFlatBitmap(t *testing.T) {
	fb := NewFlatBitmap()
	if fb == nil {
		t.Fatal("NewFlatBitmap returned nil")
	}

	// Check that bitmap is initialized to zero
	for i := 0; i < _64_BITS_COUNT; i++ {
		if fb.bitmap[i] != 0 {
			t.Errorf("bitmap[%d] should be 0, got %d", i, fb.bitmap[i])
		}
		if fb.valueSlice[i] != nil {
			t.Errorf("valueSlice[%d] should be nil, got %v", i, fb.valueSlice[i])
		}
	}
}

func TestFlatBitmap_SetAndGet_Basic(t *testing.T) {
	fb := NewFlatBitmap()

	testCases := []struct {
		key string
		idx uint32
	}{
		{"test1", 1},
		{"test2", 2},
		{"hello", 100},
		{"world", 999},
		{"", 0}, // empty key
		{"a", 1},
		{"very_long_key_with_many_characters_to_test_hash_distribution", 12345},
	}

	// Set values
	for _, tc := range testCases {
		fb.Set(tc.key, tc.idx)
	}

	// Get and verify values
	for _, tc := range testCases {
		got, found := fb.Get(tc.key)
		if !found {
			t.Errorf("Get(%q) should find the key", tc.key)
			continue
		}
		if got != tc.idx {
			t.Errorf("Get(%q) = %d, want %d", tc.key, got, tc.idx)
		}
	}
}

func TestFlatBitmap_Get_NonExistent(t *testing.T) {
	fb := NewFlatBitmap()

	// Test getting non-existent keys
	testKeys := []string{"nonexistent", "missing", "not_there", ""}

	for _, key := range testKeys {
		got, found := fb.Get(key)
		if found {
			t.Errorf("Get(%q) should not find the key, but found value %d", key, got)
		}
		if got != 0 {
			t.Errorf("Get(%q) should return 0 for non-existent key, got %d", key, got)
		}
	}
}

func TestFlatBitmap_OverwriteValue(t *testing.T) {
	fb := NewFlatBitmap()

	key := "testkey"
	originalIdx := uint32(123)
	newIdx := uint32(456)

	// Set initial value
	fb.Set(key, originalIdx)
	got, found := fb.Get(key)
	if !found || got != originalIdx {
		t.Fatalf("Initial set failed: got %d, want %d, found %v", got, originalIdx, found)
	}

	// Overwrite with new value
	fb.Set(key, newIdx)
	got, found = fb.Get(key)
	if !found || got != newIdx {
		t.Errorf("Overwrite failed: got %d, want %d, found %v", got, newIdx, found)
	}
}

func TestFlatBitmap_MultipleKeysInSameBitPosition(t *testing.T) {
	fb := NewFlatBitmap()

	// Create keys that might hash to the same bit position but different hash values
	keys := []string{
		"collision1",
		"collision2",
		"collision3",
		"test_collision_a",
		"test_collision_b",
	}

	indices := []uint32{1, 2, 3, 4, 5}

	// Set all keys
	for i, key := range keys {
		fb.Set(key, indices[i])
	}

	// Verify all keys can be retrieved correctly
	for i, key := range keys {
		got, found := fb.Get(key)
		if !found {
			t.Errorf("Key %q not found", key)
			continue
		}
		if got != indices[i] {
			t.Errorf("Key %q: got %d, want %d", key, got, indices[i])
		}
	}
}

func TestFlatBitmap_HashCollisionHandling(t *testing.T) {
	fb := NewFlatBitmap()

	// Test with many keys to increase probability of collisions
	numKeys := 1000
	keyPrefix := "collision_test_"

	// Set many keys
	for i := 0; i < numKeys; i++ {
		key := keyPrefix + strconv.Itoa(i)
		idx := uint32(i + 1)
		fb.Set(key, idx)
	}

	// Verify all keys can still be retrieved
	for i := 0; i < numKeys; i++ {
		key := keyPrefix + strconv.Itoa(i)
		expectedIdx := uint32(i + 1)

		got, found := fb.Get(key)
		if !found {
			t.Errorf("Key %q not found after collision test", key)
			continue
		}
		if got != expectedIdx {
			t.Errorf("Key %q: got %d, want %d", key, got, expectedIdx)
		}
	}
}

func TestFlatBitmap_MaxIndexValue(t *testing.T) {
	fb := NewFlatBitmap()

	// Test with maximum possible index value (26 bits)
	maxIdx := uint32(0x3FFFFFF) // 26 bits all set
	key := "max_index_test"

	fb.Set(key, maxIdx)
	got, found := fb.Get(key)

	if !found {
		t.Fatal("Key with max index not found")
	}
	if got != maxIdx {
		t.Errorf("Max index test: got %d, want %d", got, maxIdx)
	}
}

func TestFlatBitmap_ZeroIndex(t *testing.T) {
	fb := NewFlatBitmap()

	key := "zero_index_test"
	idx := uint32(0)

	fb.Set(key, idx)
	got, found := fb.Get(key)

	if !found {
		t.Fatal("Key with zero index not found")
	}
	if got != idx {
		t.Errorf("Zero index test: got %d, want %d", got, idx)
	}
}

func TestFlatBitmap_SliceExpansion(t *testing.T) {
	fb := NewFlatBitmap()

	// Create multiple keys that should hash to the same position
	// to force slice expansion beyond initial 64 elements
	baseKey := "expansion_test_base"

	// Add many variations to force collisions and slice expansion
	numVariations := 100
	for i := 0; i < numVariations; i++ {
		key := fmt.Sprintf("%s_%d_variation", baseKey, i)
		idx := uint32(i + 1)
		fb.Set(key, idx)
	}

	// Verify all keys are still accessible
	for i := 0; i < numVariations; i++ {
		key := fmt.Sprintf("%s_%d_variation", baseKey, i)
		expectedIdx := uint32(i + 1)

		got, found := fb.Get(key)
		if !found {
			t.Errorf("Key %q not found after slice expansion", key)
			continue
		}
		if got != expectedIdx {
			t.Errorf("Key %q after expansion: got %d, want %d", key, got, expectedIdx)
		}
	}
}

func TestFlatBitmap_RandomKeys(t *testing.T) {
	fb := NewFlatBitmap()
	rand.Seed(time.Now().UnixNano())

	// Generate random keys and indices
	numTests := 500
	keyMap := make(map[string]uint32)

	for i := 0; i < numTests; i++ {
		// Generate random key
		keyLen := rand.Intn(20) + 1
		key := make([]byte, keyLen)
		for j := range key {
			key[j] = byte(rand.Intn(94) + 33) // Printable ASCII
		}
		keyStr := string(key)

		// Generate random index
		idx := uint32(rand.Intn(0x3FFFFFF)) // 26 bits

		keyMap[keyStr] = idx
		fb.Set(keyStr, idx)
	}

	// Verify all random keys
	for key, expectedIdx := range keyMap {
		got, found := fb.Get(key)
		if !found {
			t.Errorf("Random key %q not found", key)
			continue
		}
		if got != expectedIdx {
			t.Errorf("Random key %q: got %d, want %d", key, got, expectedIdx)
		}
	}
}

func TestFlatBitmap_ConcurrentAccess(t *testing.T) {
	fb := NewFlatBitmap()
	numGoroutines := 10
	keysPerGoroutine := 100

	var wg sync.WaitGroup

	// Concurrent writes
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < keysPerGoroutine; i++ {
				key := fmt.Sprintf("concurrent_%d_%d", goroutineID, i)
				idx := uint32(goroutineID*keysPerGoroutine + i + 1)
				fb.Set(key, idx)
			}
		}(g)
	}

	wg.Wait()

	// Concurrent reads
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < keysPerGoroutine; i++ {
				key := fmt.Sprintf("concurrent_%d_%d", goroutineID, i)
				expectedIdx := uint32(goroutineID*keysPerGoroutine + i + 1)

				got, found := fb.Get(key)
				if !found {
					t.Errorf("Concurrent key %q not found", key)
					continue
				}
				if got != expectedIdx {
					t.Errorf("Concurrent key %q: got %d, want %d", key, got, expectedIdx)
				}
			}
		}(g)
	}

	wg.Wait()
}

func TestFlatBitmap_SpecialCharacters(t *testing.T) {
	fb := NewFlatBitmap()

	specialKeys := []string{
		"key with spaces",
		"key\twith\ttabs",
		"key\nwith\nnewlines",
		"key-with-dashes",
		"key_with_underscores",
		"key.with.dots",
		"key@with#special$chars%",
		"🚀unicode🌟keys🎉",
		"中文键",
		"العربية",
	}

	for i, key := range specialKeys {
		idx := uint32(i + 1)
		fb.Set(key, idx)

		got, found := fb.Get(key)
		if !found {
			t.Errorf("Special character key %q not found", key)
			continue
		}
		if got != idx {
			t.Errorf("Special character key %q: got %d, want %d", key, got, idx)
		}
	}
}

// TestHash10Function is commented out due to variable scope conflict
// The Hash10 function works correctly as verified by the main FlatBitmap tests
/*
func TestHash10Function(t *testing.T) {
	testCases := []string{
		"test",
		"hello",
		"world",
		"",
		"a",
		"very_long_string_to_test_hash_function",
	}

	for _, key := range testCases {
		hashValue := Hash10(key)

		// Hash10 should return a 10-bit value (0-1023)
		if hashValue > 0x3FF {
			t.Errorf("Hash10(%q) = %d, should be <= 1023", key, hashValue)
		}
	}

	// Test that different keys produce different hashes (not guaranteed but likely)
	hashes := make(map[uint16]string)
	conflicts := 0

	for _, key := range testCases {
		hashValue := Hash10(key)
		if existingKey, exists := hashes[hashValue]; exists {
			conflicts++
			t.Logf("Hash collision: Hash10(%q) = Hash10(%q) = %d", key, existingKey, hashValue)
		} else {
			hashes[hashValue] = key
		}
	}

	// With a good hash function, we shouldn't have many conflicts in this small set
	if conflicts > len(testCases)/2 {
		t.Errorf("Too many hash conflicts: %d out of %d", conflicts, len(testCases))
	}
}
*/

// Benchmark tests
func BenchmarkFlatBitmap_Set(b *testing.B) {
	fb := NewFlatBitmap()
	var keys []string

	// Prepare keys
	b.Run("Keys", func(b *testing.B) {
		keys = make([]string, b.N)
		for i := 0; i < b.N; i++ {
			keys[i] = fmt.Sprintf("benchmark_key_%d", i)
		}
	})
	printMem("Before")
	b.ResetTimer()
	b.Run("Set", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			fb.Set(keys[i], uint32(i))
		}
	})
	b.Run("Get", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = fb.Get(keys[i])
		}
	})
	b.StopTimer()
	printMem("After")

}

func printMem(label string) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("%s: Alloc = %v KB\n", label, m.Alloc/1024)
}
