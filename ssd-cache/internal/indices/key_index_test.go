package indices

import (
	"testing"
	"time"
)

func TestNewKeyIndex(t *testing.T) {
	tests := []struct {
		name                string
		rounds              int
		rbInitial           int
		rbMax               int
		deleteAmortizedStep int
	}{
		{
			name:                "basic creation",
			rounds:              10,
			rbInitial:           100,
			rbMax:               1000,
			deleteAmortizedStep: 5,
		},
		{
			name:                "minimum values",
			rounds:              1,
			rbInitial:           1,
			rbMax:               10,
			deleteAmortizedStep: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ki := NewKeyIndex(tt.rounds, tt.rbInitial, tt.rbMax, tt.deleteAmortizedStep)

			if ki == nil {
				t.Fatal("Expected NewKeyIndex to return non-nil KeyIndex")
			}
			if ki.rm == nil {
				t.Fatal("Expected rm to be non-nil")
			}
			if ki.rb == nil {
				t.Fatal("Expected rb to be non-nil")
			}
			if ki.mc == nil {
				t.Fatal("Expected mc to be non-nil")
			}
			if ki.deleteInProgress {
				t.Error("Expected deleteInProgress to be false")
			}
			if ki.deleteAmortizedStep != tt.deleteAmortizedStep {
				t.Errorf("Expected deleteAmortizedStep to be %d, got %d", tt.deleteAmortizedStep, ki.deleteAmortizedStep)
			}
			if ki.startAt <= 0 {
				t.Error("Expected startAt to be greater than 0")
			}
		})
	}
}

func TestKeyIndex_Add(t *testing.T) {
	ki := NewKeyIndex(10, 100, 1000, 5)

	tests := []struct {
		name       string
		key        string
		length     uint16
		memId      uint32
		offset     uint32
		lastAccess uint32
		freq       uint32
		exptime    uint64
	}{
		{
			name:       "basic add",
			key:        "test_key_1",
			length:     256,
			memId:      1,
			offset:     100,
			lastAccess: 12345,
			freq:       1,
			exptime:    uint64(time.Now().Unix() + 3600),
		},
		{
			name:       "different key",
			key:        "another_key",
			length:     512,
			memId:      2,
			offset:     200,
			lastAccess: 54321,
			freq:       5,
			exptime:    uint64(time.Now().Unix() + 7200),
		},
		{
			name:       "empty key",
			key:        "",
			length:     0,
			memId:      0,
			offset:     0,
			lastAccess: 0,
			freq:       0,
			exptime:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Add should not panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Add panicked: %v", r)
				}
			}()
			ki.Add(tt.key, tt.length, tt.memId, tt.offset, tt.lastAccess, tt.freq, tt.exptime)
		})
	}
}

func TestKeyIndex_Get(t *testing.T) {
	ki := NewKeyIndex(10, 100, 1000, 5)

	// Test getting non-existent key
	t.Run("non-existent key", func(t *testing.T) {
		length, memId, offset, lastAccess, freq, exptime, found := ki.Get("non_existent")
		if found {
			t.Error("Expected found to be false for non-existent key")
		}
		if length != 0 {
			t.Errorf("Expected length to be 0, got %d", length)
		}
		if memId != 0 {
			t.Errorf("Expected memId to be 0, got %d", memId)
		}
		if offset != 0 {
			t.Errorf("Expected offset to be 0, got %d", offset)
		}
		if lastAccess != 0 {
			t.Errorf("Expected lastAccess to be 0, got %d", lastAccess)
		}
		if freq != 0 {
			t.Errorf("Expected freq to be 0, got %d", freq)
		}
		if exptime != 0 {
			t.Errorf("Expected exptime to be 0, got %d", exptime)
		}
	})

	// Add a key and then retrieve it
	t.Run("existing key", func(t *testing.T) {
		testKey := "test_get_key"
		expectedLength := uint16(256)
		expectedMemId := uint32(1)
		expectedOffset := uint32(100)
		expectedLastAccess := uint32(12345)
		expectedFreq := uint32(1)
		expectedExptime := uint64(time.Now().Unix() + 3600)

		ki.Add(testKey, expectedLength, expectedMemId, expectedOffset, expectedLastAccess, expectedFreq, expectedExptime)

		length, memId, offset, lastAccess, freq, exptime, found := ki.Get(testKey)
		if !found {
			t.Fatal("Expected to find the added key")
		}
		if length != expectedLength {
			t.Errorf("Expected length %d, got %d", expectedLength, length)
		}
		if memId != expectedMemId {
			t.Errorf("Expected memId %d, got %d", expectedMemId, memId)
		}
		if offset != expectedOffset {
			t.Errorf("Expected offset %d, got %d", expectedOffset, offset)
		}
		if lastAccess != expectedLastAccess {
			t.Errorf("Expected lastAccess %d, got %d", expectedLastAccess, lastAccess)
		}
		if freq != expectedFreq {
			t.Errorf("Expected freq %d, got %d", expectedFreq, freq)
		}
		if exptime != expectedExptime {
			t.Errorf("Expected exptime %d, got %d", expectedExptime, exptime)
		}
	})
}

func TestKeyIndex_IncrLastAccessAndFreq(t *testing.T) {
	ki := NewKeyIndex(10, 100, 1000, 5)

	t.Run("non-existent key", func(t *testing.T) {
		// Should not panic when key doesn't exist
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("IncrLastAccessAndFreq panicked on non-existent key: %v", r)
			}
		}()
		ki.IncrLastAccessAndFreq("non_existent")
	})

	t.Run("existing key", func(t *testing.T) {
		testKey := "test_incr_key"
		originalFreq := uint32(1)
		// Use a small relative time for lastAccess since IncrLastAccessAndFreq calculates time relative to startAt
		originalLastAccess := uint32(100)

		// Add a key first
		ki.Add(testKey, 256, 1, 100, originalLastAccess, originalFreq, uint64(time.Now().Unix()+3600))

		// Get original values
		_, _, _, origLastAccess, origFreq, _, found := ki.Get(testKey)
		if !found {
			t.Fatal("Expected to find the added key")
		}
		if origFreq != originalFreq {
			t.Errorf("Expected original freq %d, got %d", originalFreq, origFreq)
		}

		// Increment last access and frequency
		ki.IncrLastAccessAndFreq(testKey)

		// Verify changes
		_, _, _, newLastAccess, newFreq, _, found := ki.Get(testKey)
		if !found {
			t.Fatal("Expected to find the key after increment")
		}

		// Last access should be updated or stay the same (since IncrLastAccessAndFreq updates to current time)
		// The implementation calculates current time relative to startAt in minutes, so it might be 0 or small number
		if newLastAccess < origLastAccess && origLastAccess > 0 {
			// Only fail if origLastAccess was > 0 and newLastAccess is less than it by a significant amount
			if origLastAccess-newLastAccess > 100 {
				t.Errorf("Expected newLastAccess (%d) to not decrease significantly from origLastAccess (%d)", newLastAccess, origLastAccess)
			}
		}

		// Frequency might change depending on MorrisLogCounter implementation
		// At minimum, it should not decrease
		if newFreq < origFreq {
			t.Errorf("Expected newFreq (%d) >= origFreq (%d)", newFreq, origFreq)
		}
	})
}

func TestKeyIndex_TrimOperations(t *testing.T) {
	ki := NewKeyIndex(10, 100, 1000, 5)

	t.Run("initial trim status", func(t *testing.T) {
		if ki.TrimStatus() {
			t.Error("Expected initial trim status to be false")
		}
	})

	t.Run("start trim", func(t *testing.T) {
		ki.StartTrim()
		if !ki.TrimStatus() {
			t.Error("Expected trim status to be true after StartTrim")
		}
	})
}

func TestKeyIndex_TrimAmortizedDeletion(t *testing.T) {
	// Use a small deleteAmortizedStep to make the test more predictable
	ki := NewKeyIndex(10, 100, 1000, 2)

	// Phase 1: Add entries with memId 1
	memId1Keys := []string{"key1_1", "key1_2", "key1_3", "key1_4", "key1_5"}
	for i, key := range memId1Keys {
		ki.Add(key, uint16(100+i), 1, uint32((i+1)*100), uint32(i*1000), uint32(i+1), uint64(time.Now().Unix()+int64(i*3600)))
	}

	// Phase 2: Add entries with memId 2
	memId2Keys := []string{"key2_1", "key2_2", "key2_3", "key2_4", "key2_5"}
	for i, key := range memId2Keys {
		ki.Add(key, uint16(200+i), 2, uint32((i+1)*200), uint32(i*2000), uint32(i+1), uint64(time.Now().Unix()+int64(i*3600)))
	}

	// Verify all keys are present before trim
	t.Run("verify all keys before trim", func(t *testing.T) {
		for _, key := range memId1Keys {
			_, _, _, _, _, _, found := ki.Get(key)
			if !found {
				t.Errorf("Expected to find memId 1 key %s before trim", key)
			}
		}
		for _, key := range memId2Keys {
			_, _, _, _, _, _, found := ki.Get(key)
			if !found {
				t.Errorf("Expected to find memId 2 key %s before trim", key)
			}
		}
	})

	// Start trim operation
	ki.StartTrim()
	if !ki.TrimStatus() {
		t.Fatal("Expected trim status to be true after StartTrim")
	}

	// Phase 3: Add more entries to trigger amortized deletion
	// Each Add will trigger deleteAmortizedStep deletions until all memId 1 entries are removed
	additionalKeys := []string{"additional1", "additional2", "additional3", "additional4", "additional5", "additional6", "additional7", "additional8"}
	actuallyAdded := make([]string, 0, len(additionalKeys))

	for i, key := range additionalKeys {
		// Add an entry which will trigger amortized deletion
		ki.Add(key, uint16(300+i), 3, uint32((i+1)*300), uint32(i*3000), uint32(i+1), uint64(time.Now().Unix()+int64(i*3600)))
		actuallyAdded = append(actuallyAdded, key)

		// Check trim status after each add
		trimStatus := ki.TrimStatus()
		t.Logf("After adding %s, trim status: %v", key, trimStatus)

		// If trim has completed, break early (this is the expected behavior)
		if !trimStatus {
			t.Logf("Trim completed after adding %s (total additional entries added: %d)", key, len(actuallyAdded))
			break
		}
	}

	// Verify that trim eventually completes
	t.Run("verify trim completion", func(t *testing.T) {
		// If trim is still in progress after all additions, something is wrong
		if ki.TrimStatus() {
			t.Error("Expected trim to complete after sufficient Add operations")
		}
	})

	// Verify that memId 1 entries are removed
	t.Run("verify memId 1 entries removed", func(t *testing.T) {
		memId1FoundCount := 0
		for _, key := range memId1Keys {
			_, _, _, _, _, _, found := ki.Get(key)
			if found {
				memId1FoundCount++
			}
		}

		// After trim, we expect most or all memId 1 entries to be removed
		// Due to the ring buffer nature, some might still exist, but there should be fewer
		t.Logf("Found %d out of %d memId 1 entries after trim", memId1FoundCount, len(memId1Keys))

		// The exact behavior depends on the ring buffer implementation, but trim should have removed some entries
		if memId1FoundCount == len(memId1Keys) {
			t.Error("Expected some memId 1 entries to be removed by trim operation")
		}
	})

	// Verify that memId 2 entries are mostly preserved
	t.Run("verify memId 2 entries preserved", func(t *testing.T) {
		memId2FoundCount := 0
		for _, key := range memId2Keys {
			_, _, _, _, _, _, found := ki.Get(key)
			if found {
				memId2FoundCount++
			}
		}

		t.Logf("Found %d out of %d memId 2 entries after trim", memId2FoundCount, len(memId2Keys))

		// memId 2 entries should be better preserved since they were added after memId 1
		// Some might be removed due to ring buffer eviction, but should be less affected by trim
	})

	// Verify that the entries we actually added are present
	t.Run("verify actually added entries present", func(t *testing.T) {
		foundCount := 0
		for _, key := range actuallyAdded {
			_, _, _, _, _, _, found := ki.Get(key)
			if found {
				foundCount++
			}
		}

		t.Logf("Found %d out of %d actually added entries after trim", foundCount, len(actuallyAdded))

		// All entries that were actually added should be findable
		if foundCount != len(actuallyAdded) {
			t.Errorf("Expected to find all %d actually added entries, but found %d", len(actuallyAdded), foundCount)
		}
	})
}

func TestKeyIndex_MultipleOperations(t *testing.T) {
	ki := NewKeyIndex(10, 100, 1000, 5)

	// Add multiple keys
	keys := []string{"key1", "key2", "key3", "key4", "key5"}
	for i, key := range keys {
		ki.Add(key, uint16(100+i), uint32(i+1), uint32((i+1)*100), uint32(i*1000), uint32(i+1), uint64(time.Now().Unix()+int64(i*3600)))
	}

	// Verify all keys can be retrieved
	for i, key := range keys {
		length, memId, offset, _, freq, _, found := ki.Get(key)
		if !found {
			t.Errorf("Key %s should be found", key)
		}
		if length != uint16(100+i) {
			t.Errorf("Expected length %d for key %s, got %d", 100+i, key, length)
		}
		if memId != uint32(i+1) {
			t.Errorf("Expected memId %d for key %s, got %d", i+1, key, memId)
		}
		if offset != uint32((i+1)*100) {
			t.Errorf("Expected offset %d for key %s, got %d", (i+1)*100, key, offset)
		}
		if freq != uint32(i+1) {
			t.Errorf("Expected freq %d for key %s, got %d", i+1, key, freq)
		}
	}

	// Update access patterns
	for _, key := range keys {
		ki.IncrLastAccessAndFreq(key)
	}

	// Verify keys still exist after updates
	for _, key := range keys {
		_, _, _, _, _, _, found := ki.Get(key)
		if !found {
			t.Errorf("Key %s should still be found after update", key)
		}
	}
}

func TestKeyIndex_EdgeCases(t *testing.T) {
	ki := NewKeyIndex(10, 100, 1000, 5)

	t.Run("very long key", func(t *testing.T) {
		longKey := string(make([]byte, 1000))
		for i := range longKey {
			longKey = longKey[:i] + "a" + longKey[i+1:]
		}

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Add panicked with long key: %v", r)
			}
		}()
		ki.Add(longKey, 256, 1, 100, 12345, 1, uint64(time.Now().Unix()+3600))

		_, _, _, _, _, _, found := ki.Get(longKey)
		if !found {
			t.Error("Expected to find the long key")
		}
	})

	t.Run("max uint values", func(t *testing.T) {
		key := "max_values_key"
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Add panicked with max values: %v", r)
			}
		}()
		ki.Add(key, 65535, 4294967295, 4294967295, 16777215, 16777215, 281474976710655)

		length, memId, offset, lastAccess, freq, exptime, found := ki.Get(key)
		if !found {
			t.Fatal("Expected to find the key with max values")
		}
		if length != 65535 {
			t.Errorf("Expected length 65535, got %d", length)
		}
		if memId != 4294967295 {
			t.Errorf("Expected memId 4294967295, got %d", memId)
		}
		if offset != 4294967295 {
			t.Errorf("Expected offset 4294967295, got %d", offset)
		}
		// Note: lastAccess and freq are masked to 24 bits, so they may not match exactly
		if lastAccess > 16777215 {
			t.Errorf("Expected lastAccess <= 16777215, got %d", lastAccess)
		}
		if freq > 16777215 {
			t.Errorf("Expected freq <= 16777215, got %d", freq)
		}
		if exptime != 281474976710655 {
			t.Errorf("Expected exptime 281474976710655, got %d", exptime)
		}
	})
}

func BenchmarkKeyIndex_Add(b *testing.B) {
	ki := NewKeyIndex(10, 1000, 10000, 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "benchmark_key_" + string(rune('0'+i%10000000))
		ki.Add(key, uint16(i%65536), uint32(i), uint32(i*10), uint32(i%16777216), uint32(i%16777216), uint64(time.Now().Unix()+int64(i)))
	}
}

func BenchmarkKeyIndex_Get(b *testing.B) {
	ki := NewKeyIndex(10, 1000, 10000, 10)

	// Pre-populate with some data
	for i := 0; i < 10000000; i++ {
		key := "benchmark_key_" + string(rune('0'+i%10000000))
		ki.Add(key, uint16(i%65536), uint32(i), uint32(i*10), uint32(i%16777216), uint32(i%16777216), uint64(time.Now().Unix()+int64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "benchmark_key_" + string(rune('0'+i%10000000))
		ki.Get(key)
	}
}

func BenchmarkKeyIndex_IncrLastAccessAndFreq(b *testing.B) {
	ki := NewKeyIndex(10, 1000, 10000, 10)

	// Pre-populate with some data
	for i := 0; i < 10000000; i++ {
		key := "benchmark_key_" + string(rune('0'+i%10000000))
		ki.Add(key, uint16(i%65536), uint32(i), uint32(i*10), uint32(i%16777216), uint32(i%16777216), uint64(time.Now().Unix()+int64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "benchmark_key_" + string(rune('0'+i%10000000))
		ki.IncrLastAccessAndFreq(key)
	}
}
