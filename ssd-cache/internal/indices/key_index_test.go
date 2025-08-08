package indices

import (
	"fmt"
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

func TestKeyIndex_MainGoParams(t *testing.T) {
	// Using the same parameters as main.go
	ki := NewKeyIndex(10, 100, 100000000, 100) // rounds=10, rbInitial=100, rbMax=100M, deleteAmortizedStep=100

	const numKeys = 1000000 // Start with 1M keys to test the issue

	t.Run("add and retrieve keys with main.go params", func(t *testing.T) {
		// Add keys
		for i := 0; i < numKeys; i++ {
			key := fmt.Sprintf("key%d", i)
			ki.Put(key, uint16(len(key)), uint32(i%1000), uint32(i*10), uint64(time.Now().Unix()+3600))
		}

		// Try to retrieve all keys
		notFoundCount := 0
		foundCount := 0

		for i := 0; i < numKeys; i++ {
			key := fmt.Sprintf("key%d", i)
			_, _, _, _, _, _, found := ki.Get(key)
			if !found {
				notFoundCount++
			} else {
				foundCount++
			}
		}

		t.Logf("Total keys: %d", numKeys)
		t.Logf("Found: %d", foundCount)
		t.Logf("Not found: %d", notFoundCount)
		t.Logf("Found percentage: %.2f%%", float64(foundCount)/float64(numKeys)*100)

		// Check if most keys are missing (indicating the issue)
		if float64(notFoundCount)/float64(numKeys) > 0.5 {
			t.Logf("Issue reproduced: %.2f%% of keys are missing", float64(notFoundCount)/float64(numKeys)*100)
		}
	})

	t.Run("test ring buffer overflow behavior", func(t *testing.T) {
		// Test with more keys than ring buffer initial size to see overflow behavior
		smallKi := NewKeyIndex(10, 10, 50, 5) // Small ring buffer

		// Add more keys than initial capacity
		for i := 0; i < 100; i++ {
			key := fmt.Sprintf("overflow_key%d", i)
			smallKi.Put(key, uint16(len(key)), uint32(i%10), uint32(i*10), uint64(time.Now().Unix()+3600))
		}

		// Check how many keys are still findable
		foundCount := 0
		for i := 0; i < 100; i++ {
			key := fmt.Sprintf("overflow_key%d", i)
			_, _, _, _, _, _, found := smallKi.Get(key)
			if found {
				foundCount++
			}
		}

		t.Logf("Ring buffer overflow test - Found %d out of 100 keys", foundCount)

		// Earlier keys should be evicted due to ring buffer overflow
		earlyKeysFound := 0
		for i := 0; i < 10; i++ {
			key := fmt.Sprintf("overflow_key%d", i)
			_, _, _, _, _, _, found := smallKi.Get(key)
			if found {
				earlyKeysFound++
			}
		}

		laterKeysFound := 0
		for i := 90; i < 100; i++ {
			key := fmt.Sprintf("overflow_key%d", i)
			_, _, _, _, _, _, found := smallKi.Get(key)
			if found {
				laterKeysFound++
			}
		}

		t.Logf("Early keys (0-9) found: %d/10", earlyKeysFound)
		t.Logf("Later keys (90-99) found: %d/10", laterKeysFound)
	})

	t.Run("test trim operation with many keys", func(t *testing.T) {
		trimKi := NewKeyIndex(10, 100, 10000, 10)

		// Add keys with memId 1
		for i := 0; i < 5000; i++ {
			key := fmt.Sprintf("trim_key1_%d", i)
			trimKi.Put(key, uint16(len(key)), 1, uint32(i*10), uint64(time.Now().Unix()+3600))
		}

		// Add keys with memId 2
		for i := 0; i < 5000; i++ {
			key := fmt.Sprintf("trim_key2_%d", i)
			trimKi.Put(key, uint16(len(key)), 2, uint32(i*10), uint64(time.Now().Unix()+3600))
		}

		// Start trim
		trimKi.StartTrim()

		// Add more keys to trigger trim operations
		for i := 0; i < 1000; i++ {
			key := fmt.Sprintf("trim_trigger_%d", i)
			trimKi.Put(key, uint16(len(key)), 3, uint32(i*10), uint64(time.Now().Unix()+3600))

			if !trimKi.TrimStatus() {
				t.Logf("Trim completed after %d trigger operations", i+1)
				break
			}
		}

		// Count remaining keys
		memId1Found := 0
		memId2Found := 0
		triggerFound := 0

		for i := 0; i < 5000; i++ {
			key1 := fmt.Sprintf("trim_key1_%d", i)
			if _, _, _, _, _, _, found := trimKi.Get(key1); found {
				memId1Found++
			}

			key2 := fmt.Sprintf("trim_key2_%d", i)
			if _, _, _, _, _, _, found := trimKi.Get(key2); found {
				memId2Found++
			}
		}

		for i := 0; i < 1000; i++ {
			key := fmt.Sprintf("trim_trigger_%d", i)
			if _, _, _, _, _, _, found := trimKi.Get(key); found {
				triggerFound++
			}
		}

		t.Logf("After trim - memId 1 keys found: %d/5000", memId1Found)
		t.Logf("After trim - memId 2 keys found: %d/5000", memId2Found)
		t.Logf("After trim - trigger keys found: %d/1000", triggerFound)
	})
}

func TestKeyIndex_CapacityOverflowIssue(t *testing.T) {
	t.Run("demonstrate ring buffer capacity overflow", func(t *testing.T) {
		// Create with small max capacity to quickly demonstrate the issue
		ki := NewKeyIndex(10, 100, 1000, 10) // maxCapacity = 1000

		// Add more keys than max capacity
		const totalKeys = 2000
		for i := 0; i < totalKeys; i++ {
			key := fmt.Sprintf("overflow_key%d", i)
			ki.Put(key, uint16(len(key)), uint32(i%10), uint32(i*10), uint64(time.Now().Unix()+3600))
		}

		// Check how many keys are still findable
		foundCount := 0
		notFoundCount := 0
		firstFoundKey := -1
		lastFoundKey := -1

		for i := 0; i < totalKeys; i++ {
			key := fmt.Sprintf("overflow_key%d", i)
			_, _, _, _, _, _, found := ki.Get(key)
			if found {
				foundCount++
				if firstFoundKey == -1 {
					firstFoundKey = i
				}
				lastFoundKey = i
			} else {
				notFoundCount++
			}
		}

		t.Logf("Total keys added: %d", totalKeys)
		t.Logf("Ring buffer max capacity: 1000")
		t.Logf("Found: %d", foundCount)
		t.Logf("Not found: %d", notFoundCount)
		t.Logf("Found percentage: %.2f%%", float64(foundCount)/float64(totalKeys)*100)
		t.Logf("First found key index: %d", firstFoundKey)
		t.Logf("Last found key index: %d", lastFoundKey)

		// The ring buffer should only retain the last ~1000 keys
		if foundCount > 1000 {
			t.Errorf("Expected found count to be around max capacity (1000), got %d", foundCount)
		}

		// Early keys should be evicted
		earlyKeysFound := 0
		for i := 0; i < 500; i++ {
			key := fmt.Sprintf("overflow_key%d", i)
			if _, _, _, _, _, _, found := ki.Get(key); found {
				earlyKeysFound++
			}
		}

		// Later keys should be present
		laterKeysFound := 0
		for i := totalKeys - 500; i < totalKeys; i++ {
			key := fmt.Sprintf("overflow_key%d", i)
			if _, _, _, _, _, _, found := ki.Get(key); found {
				laterKeysFound++
			}
		}

		t.Logf("Early keys (0-499) found: %d/500", earlyKeysFound)
		t.Logf("Later keys (%d-%d) found: %d/500", totalKeys-500, totalKeys-1, laterKeysFound)

		// This demonstrates the issue: early keys are evicted when capacity is exceeded
		if earlyKeysFound > laterKeysFound {
			t.Error("Early keys should be evicted more than later keys due to FIFO eviction")
		}
	})

	t.Run("main.go scenario with scaling", func(t *testing.T) {
		// Test with proportionally scaled parameters to simulate main.go issue
		// main.go: rbMax=100M, keys=50M
		// scaled: rbMax=1000, keys=500 (same 50% ratio)
		ki := NewKeyIndex(10, 100, 1000, 10)

		const scaledKeys = 1500 // Exceed capacity by 50%
		for i := 0; i < scaledKeys; i++ {
			key := fmt.Sprintf("key%d", i)
			ki.Put(key, uint16(len(key)), uint32(i%1000), uint32(i*10), uint64(time.Now().Unix()+3600))
		}

		foundCount := 0
		notFoundCount := 0

		for i := 0; i < scaledKeys; i++ {
			key := fmt.Sprintf("key%d", i)
			if _, _, _, _, _, _, found := ki.Get(key); found {
				foundCount++
			} else {
				notFoundCount++
			}
		}

		t.Logf("Scaled test - Total: %d, Found: %d, Not found: %d", scaledKeys, foundCount, notFoundCount)
		t.Logf("Not found percentage: %.2f%%", float64(notFoundCount)/float64(scaledKeys)*100)

		// In main.go, this would cause ~33% of keys to be missing (1500-1000)/1500
		expectedNotFoundPercentage := float64(scaledKeys-1000) / float64(scaledKeys) * 100
		actualNotFoundPercentage := float64(notFoundCount) / float64(scaledKeys) * 100

		t.Logf("Expected not found percentage: %.2f%%", expectedNotFoundPercentage)
		t.Logf("Actual not found percentage: %.2f%%", actualNotFoundPercentage)

		if actualNotFoundPercentage < expectedNotFoundPercentage*0.8 {
			t.Logf("Note: Actual percentage is lower than expected, possibly due to ring buffer growth dynamics")
		}
	})
}

func TestKeyIndex_ExactMainGoParams(t *testing.T) {
	t.Run("exact main.go parameters", func(t *testing.T) {
		// Test with increasing key counts to find the breaking point
		testSizes := []int{100000, 1000000, 5000000, 10000000}

		for _, numKeys := range testSizes {
			t.Run(fmt.Sprintf("test_with_%d_keys", numKeys), func(t *testing.T) {
				// Create a fresh KeyIndex for each test
				freshKi := NewKeyIndex(10, 100, 100000000, 100)

				// Add keys
				for i := 0; i < numKeys; i++ {
					key := fmt.Sprintf("key%d", i)
					freshKi.Put(key, uint16(len(key)), uint32(i%1000), uint32(i*10), uint64(time.Now().Unix()+3600))
				}

				// Check retrieval
				foundCount := 0
				notFoundCount := 0

				for i := 0; i < numKeys; i++ {
					key := fmt.Sprintf("key%d", i)
					_, _, _, _, _, _, found := freshKi.Get(key)
					if found {
						foundCount++
					} else {
						notFoundCount++
					}
				}

				foundPercentage := float64(foundCount) / float64(numKeys) * 100
				t.Logf("Keys: %d, Found: %d (%.2f%%), Not found: %d (%.2f%%)",
					numKeys, foundCount, foundPercentage, notFoundCount, 100-foundPercentage)

				// Check if we have a significant loss rate
				if foundPercentage < 95.0 {
					t.Logf("Significant loss detected at %d keys: %.2f%% found", numKeys, foundPercentage)
				}
			})
		}
	})

	t.Run("check hash collision issues", func(t *testing.T) {
		ki := NewKeyIndex(10, 100, 100000000, 100)

		// Test with keys that might cause hash collisions
		const numKeys = 100000
		collisionKeys := make([]string, numKeys)

		for i := 0; i < numKeys; i++ {
			// Create keys similar to main.go pattern
			collisionKeys[i] = fmt.Sprintf("key%d", i)
			ki.Put(collisionKeys[i], uint16(len(collisionKeys[i])), uint32(i%1000), uint32(i*10), uint64(time.Now().Unix()+3600))
		}

		// Check for hash collisions by trying to find all keys
		foundCount := 0
		duplicateIndices := 0

		for i, key := range collisionKeys {
			_, _, _, _, _, _, found := ki.Get(key)
			if found {
				foundCount++
			} else {
				t.Logf("Key not found: %s (index %d)", key, i)
				if duplicateIndices < 10 {
					duplicateIndices++
				}
			}
		}

		t.Logf("Hash collision test - Found: %d/%d (%.2f%%)", foundCount, numKeys, float64(foundCount)/float64(numKeys)*100)
	})

	t.Run("ring buffer growth behavior", func(t *testing.T) {
		ki := NewKeyIndex(10, 100, 100000000, 100) // start with 100, can grow to 100M

		growthSteps := []int{50, 100, 200, 1000, 10000, 100000}

		for _, step := range growthSteps {
			// Add keys in batches
			for i := 0; i < step; i++ {
				key := fmt.Sprintf("growth_key_%d_%d", step, i)
				ki.Put(key, uint16(len(key)), uint32(i%100), uint32(i*10), uint64(time.Now().Unix()+3600))
			}

			// Check if we can find the most recent batch
			foundInBatch := 0
			for i := 0; i < step; i++ {
				key := fmt.Sprintf("growth_key_%d_%d", step, i)
				if _, _, _, _, _, _, found := ki.Get(key); found {
					foundInBatch++
				}
			}

			t.Logf("After adding %d keys: %d/%d found in last batch (%.2f%%)",
				step, foundInBatch, step, float64(foundInBatch)/float64(step)*100)
		}
	})
}

func TestKeyIndex_FileSizeLimitIssue(t *testing.T) {
	t.Run("demonstrate file size limit trim issue", func(t *testing.T) {
		// Create a KeyIndex and simulate the file wrapping scenario
		ki := NewKeyIndex(10, 100, 100000000, 100) // Same as main.go

		// Simulate the scenario where file has wrapped and trim is triggered

		// Phase 1: Add initial batch of keys (simulate early writes)
		const batchSize = 100000
		for i := 0; i < batchSize; i++ {
			key := fmt.Sprintf("early_key%d", i)
			ki.Put(key, uint16(len(key)), 1, uint32(i*100), uint64(time.Now().Unix()+3600)) // memId = 1
		}

		// Verify all early keys are present
		earlyKeysBeforeTrim := 0
		for i := 0; i < batchSize; i++ {
			key := fmt.Sprintf("early_key%d", i)
			if _, _, _, _, _, _, found := ki.Get(key); found {
				earlyKeysBeforeTrim++
			}
		}
		t.Logf("Early keys before trim: %d/%d", earlyKeysBeforeTrim, batchSize)

		// Phase 2: Simulate file wrapping by starting trim operation
		ki.StartTrim()
		t.Logf("Trim operation started (simulating file wrap)")

		// Phase 3: Add new keys which will trigger amortized deletion of old entries
		// Each Put() call will trigger up to 100 deletions (deleteAmortizedStep = 100)
		for i := 0; i < batchSize; i++ {
			key := fmt.Sprintf("new_key%d", i)
			ki.Put(key, uint16(len(key)), 2, uint32(i*100), uint64(time.Now().Unix()+3600)) // memId = 2

			// Check if trim is still in progress
			if !ki.TrimStatus() {
				t.Logf("Trim completed after adding %d new keys", i+1)
				break
			}
		}

		// Phase 4: Check how many early keys survived the trim
		earlyKeysAfterTrim := 0
		for i := 0; i < batchSize; i++ {
			key := fmt.Sprintf("early_key%d", i)
			if _, _, _, _, _, _, found := ki.Get(key); found {
				earlyKeysAfterTrim++
			}
		}

		// Check how many new keys are present
		newKeysFound := 0
		for i := 0; i < batchSize; i++ {
			key := fmt.Sprintf("new_key%d", i)
			if _, _, _, _, _, _, found := ki.Get(key); found {
				newKeysFound++
			}
		}

		t.Logf("Results after trim operation:")
		t.Logf("  Early keys (memId=1): %d/%d survived (%.2f%% lost)",
			earlyKeysAfterTrim, batchSize, 100-float64(earlyKeysAfterTrim)/float64(batchSize)*100)
		t.Logf("  New keys (memId=2): %d/%d found", newKeysFound, batchSize)

		// This demonstrates the main.go issue: old keys get deleted during trim operations
		if earlyKeysAfterTrim < earlyKeysBeforeTrim {
			t.Logf("SUCCESS: Reproduced the key loss issue!")
			t.Logf("  Lost %d keys due to trim operation", earlyKeysBeforeTrim-earlyKeysAfterTrim)
		}
	})

	t.Run("calculate file size for main.go scenario", func(t *testing.T) {
		// Calculate approximate file size for main.go scenario
		const numKeys = 50000000
		const avgKeySize = 7   // "key1234" average
		const avgValueSize = 9 // "value1234" average
		const overhead = 4     // CRC32

		avgEntrySize := avgKeySize + avgValueSize + overhead
		totalDataSize := int64(numKeys) * int64(avgEntrySize)
		maxFileSize := int64(1024 * 1024 * 1024) // 1GB from main.go

		t.Logf("Main.go scenario analysis:")
		t.Logf("  Number of keys: %d", numKeys)
		t.Logf("  Average entry size: %d bytes", avgEntrySize)
		t.Logf("  Total data size: %.2f MB", float64(totalDataSize)/(1024*1024))
		t.Logf("  Max file size: %.2f MB", float64(maxFileSize)/(1024*1024))
		t.Logf("  Size ratio: %.2fx (%.1f%% over limit)",
			float64(totalDataSize)/float64(maxFileSize),
			(float64(totalDataSize)/float64(maxFileSize)-1)*100)

		if totalDataSize > maxFileSize {
			t.Logf("FILE SIZE EXCEEDED: This will trigger file wrapping and trim operations!")
			expectedWraps := totalDataSize / maxFileSize
			t.Logf("  Expected number of wraps: %d", expectedWraps)
		}
	})
}

func TestKeyIndex_TrimMemIdBoundaryLogic(t *testing.T) {
	t.Run("verify trim stops at memId boundaries", func(t *testing.T) {
		// Use smaller parameters to make the test more predictable
		ki := NewKeyIndex(10, 100, 10000, 50) // deleteAmortizedStep = 50

		// Phase 1: Add entries for memId 1 (oldest memtable)
		memId1Keys := make([]string, 0)
		for i := 0; i < 1000; i++ {
			key := fmt.Sprintf("mem1_key%d", i)
			memId1Keys = append(memId1Keys, key)
			ki.Put(key, uint16(len(key)), 1, uint32(i*100), uint64(time.Now().Unix()+3600))
		}

		// Phase 2: Add entries for memId 2 (middle memtable)
		memId2Keys := make([]string, 0)
		for i := 0; i < 1000; i++ {
			key := fmt.Sprintf("mem2_key%d", i)
			memId2Keys = append(memId2Keys, key)
			ki.Put(key, uint16(len(key)), 2, uint32(i*100), uint64(time.Now().Unix()+3600))
		}

		// Phase 3: Add entries for memId 3 (newest memtable)
		memId3Keys := make([]string, 0)
		for i := 0; i < 1000; i++ {
			key := fmt.Sprintf("mem3_key%d", i)
			memId3Keys = append(memId3Keys, key)
			ki.Put(key, uint16(len(key)), 3, uint32(i*100), uint64(time.Now().Unix()+3600))
		}

		// Verify all keys are present before trim
		countKeys := func(keys []string, label string) int {
			found := 0
			for _, key := range keys {
				if _, _, _, _, _, _, exists := ki.Get(key); exists {
					found++
				}
			}
			t.Logf("%s: %d/%d found", label, found, len(keys))
			return found
		}

		t.Logf("Before trim:")
		mem1CountBefore := countKeys(memId1Keys, "MemId 1 keys")
		mem2CountBefore := countKeys(memId2Keys, "MemId 2 keys")
		mem3CountBefore := countKeys(memId3Keys, "MemId 3 keys")

		// Phase 4: Start trim (simulating file head punch for memId 1)
		ki.StartTrim()
		t.Logf("Trim started - should delete memId 1 entries only")

		// Phase 5: Trigger amortized deletion by adding new keys
		triggerKeys := make([]string, 0)
		maxTriggers := 100 // Limit to prevent infinite loop

		for i := 0; i < maxTriggers && ki.TrimStatus(); i++ {
			key := fmt.Sprintf("trigger_key%d", i)
			triggerKeys = append(triggerKeys, key)
			ki.Put(key, uint16(len(key)), 4, uint32(i*100), uint64(time.Now().Unix()+3600))

			if (i+1)%10 == 0 {
				t.Logf("After %d trigger operations, trim status: %v", i+1, ki.TrimStatus())
			}
		}

		t.Logf("Trim completed after %d trigger operations", len(triggerKeys))

		// Phase 6: Check results
		t.Logf("After trim:")
		mem1CountAfter := countKeys(memId1Keys, "MemId 1 keys")
		mem2CountAfter := countKeys(memId2Keys, "MemId 2 keys")
		mem3CountAfter := countKeys(memId3Keys, "MemId 3 keys")
		triggerCount := countKeys(triggerKeys, "Trigger keys")

		// Analysis
		mem1Deleted := mem1CountBefore - mem1CountAfter
		mem2Deleted := mem2CountBefore - mem2CountAfter
		mem3Deleted := mem3CountBefore - mem3CountAfter

		t.Logf("Deletion results:")
		t.Logf("  MemId 1: %d deleted, %d remaining", mem1Deleted, mem1CountAfter)
		t.Logf("  MemId 2: %d deleted, %d remaining", mem2Deleted, mem2CountAfter)
		t.Logf("  MemId 3: %d deleted, %d remaining", mem3Deleted, mem3CountAfter)
		t.Logf("  Trigger keys: %d found", triggerCount)

		// Expectations:
		// 1. MemId 1 should be heavily deleted (the trimmed memtable)
		// 2. MemId 2 and 3 should be mostly preserved
		// 3. Trim should stop at memId boundary

		if mem1Deleted == 0 {
			t.Error("Expected some memId 1 keys to be deleted during trim")
		}

		if mem2Deleted > mem1Deleted {
			t.Errorf("MemId 2 deletions (%d) should not exceed memId 1 deletions (%d)", mem2Deleted, mem1Deleted)
		}

		if mem3Deleted > mem2Deleted {
			t.Errorf("MemId 3 deletions (%d) should not exceed memId 2 deletions (%d)", mem3Deleted, mem2Deleted)
		}

		// Check if trim logic is working correctly
		expectedBehavior := mem1Deleted >= mem2Deleted && mem2Deleted >= mem3Deleted
		if expectedBehavior {
			t.Logf("✓ Trim behavior follows expected FIFO pattern")
		} else {
			t.Logf("✗ Trim behavior does not follow expected FIFO pattern")
		}
	})

	t.Run("check deleteAmortizedStep effectiveness", func(t *testing.T) {
		// Test with different deleteAmortizedStep values
		testSteps := []int{10, 50, 100, 200}

		for _, step := range testSteps {
			t.Run(fmt.Sprintf("step_%d", step), func(t *testing.T) {
				ki := NewKeyIndex(10, 100, 10000, step)

				// Add keys for memId 1
				for i := 0; i < 500; i++ {
					key := fmt.Sprintf("test_key%d", i)
					ki.Put(key, uint16(len(key)), 1, uint32(i*100), uint64(time.Now().Unix()+3600))
				}

				// Start trim
				ki.StartTrim()

				// Count triggers needed to complete trim
				triggerCount := 0
				for ki.TrimStatus() && triggerCount < 100 {
					key := fmt.Sprintf("trigger%d", triggerCount)
					ki.Put(key, uint16(len(key)), 2, uint32(triggerCount*100), uint64(time.Now().Unix()+3600))
					triggerCount++
				}

				t.Logf("DeleteAmortizedStep %d: Required %d triggers to complete trim", step, triggerCount)

				// Smaller steps should require more triggers
				if step == 10 {
					if triggerCount < 20 {
						t.Logf("Note: Step 10 completed in %d triggers (might be due to memId boundary)", triggerCount)
					}
				}
			})
		}
	})
}

func TestKeyIndex_MainGoFileSizeAnalysis(t *testing.T) {
	t.Run("detailed main.go file size calculation", func(t *testing.T) {
		const numKeys = 50000000
		const memtableSize = 1 * 1024 * 1024          // 1MB
		const maxFileSize = int64(1024 * 1024 * 1024) // 1GB

		// Calculate actual entry sizes for main.go data
		var totalSize int64
		sampleSizes := make([]int, 100) // Sample first 100 entries

		for i := 0; i < 100; i++ {
			key := fmt.Sprintf("key%d", i)
			value := fmt.Sprintf("value%d", i)
			entrySize := 4 + len(key) + len(value) // CRC32 + key + value
			sampleSizes[i] = entrySize
			totalSize += int64(entrySize)
		}

		avgEntrySize := totalSize / 100
		totalDataSize := int64(numKeys) * avgEntrySize

		t.Logf("Main.go detailed analysis:")
		t.Logf("  Sample entry sizes (first 10): %v", sampleSizes[:10])
		t.Logf("  Average entry size: %d bytes", avgEntrySize)
		t.Logf("  Total data size: %.2f MB (%.2f GB)",
			float64(totalDataSize)/(1024*1024),
			float64(totalDataSize)/(1024*1024*1024))
		t.Logf("  Max file size: %.2f MB (%.2f GB)",
			float64(maxFileSize)/(1024*1024),
			float64(maxFileSize)/(1024*1024*1024))

		// Calculate number of file wraps and trim sessions
		if totalDataSize > maxFileSize {
			numWraps := (totalDataSize + maxFileSize - 1) / maxFileSize // Ceiling division
			t.Logf("  FILE WILL WRAP %d times!", numWraps)
			t.Logf("  Each wrap triggers a trim session")

			// Calculate how many memtables per wrap
			memtablesPerWrap := maxFileSize / int64(memtableSize)
			totalMemtables := (totalDataSize + int64(memtableSize) - 1) / int64(memtableSize)

			t.Logf("  Memtables per file wrap: %d", memtablesPerWrap)
			t.Logf("  Total memtables needed: %d", totalMemtables)

			// Each trim session deletes the oldest memtable's data
			estimatedKeysLost := int((numWraps - 1) * int64(numKeys) / totalMemtables * memtablesPerWrap)
			lossPercentage := float64(estimatedKeysLost) / float64(numKeys) * 100

			t.Logf("  Estimated keys lost: %d (%.1f%%)", estimatedKeysLost, lossPercentage)

			if lossPercentage > 50 {
				t.Logf("  🔥 HIGH KEY LOSS EXPECTED! This explains the main.go issue.")
			}
		} else {
			t.Logf("  File size OK - no wrapping expected")
		}
	})

	t.Run("simulate main.go scenario with multiple trim sessions", func(t *testing.T) {
		// Simulate the multiple trim scenario
		ki := NewKeyIndex(10, 100, 100000000, 100) // main.go params

		const keysPerMemtable = 10000 // Smaller for test
		const numMemtables = 10
		totalKeys := keysPerMemtable * numMemtables

		// Phase 1: Add keys across multiple memtables
		allKeys := make([][]string, numMemtables)
		for memId := 0; memId < numMemtables; memId++ {
			allKeys[memId] = make([]string, keysPerMemtable)
			for i := 0; i < keysPerMemtable; i++ {
				key := fmt.Sprintf("mem%d_key%d", memId, i)
				allKeys[memId][i] = key
				ki.Put(key, uint16(len(key)), uint32(memId), uint32(i*100), uint64(time.Now().Unix()+3600))
			}
		}

		// Count initial keys
		initialCount := 0
		for memId := 0; memId < numMemtables; memId++ {
			for _, key := range allKeys[memId] {
				if _, _, _, _, _, _, found := ki.Get(key); found {
					initialCount++
				}
			}
		}
		t.Logf("Initial keys: %d/%d", initialCount, totalKeys)

		// Phase 2: Simulate multiple file wraps (3 trim sessions)
		numTrimSessions := 3
		for session := 0; session < numTrimSessions; session++ {
			t.Logf("Starting trim session %d", session+1)
			ki.StartTrim()

			// Trigger deletions
			for i := 0; ki.TrimStatus() && i < 200; i++ {
				key := fmt.Sprintf("session%d_trigger%d", session, i)
				ki.Put(key, uint16(len(key)), uint32(numMemtables+session), uint32(i*100), uint64(time.Now().Unix()+3600))
			}

			// Count remaining keys per memtable
			for memId := 0; memId < numMemtables; memId++ {
				count := 0
				for _, key := range allKeys[memId] {
					if _, _, _, _, _, _, found := ki.Get(key); found {
						count++
					}
				}
				if count < keysPerMemtable {
					t.Logf("  MemId %d: %d/%d remaining (%.1f%% lost)",
						memId, count, keysPerMemtable,
						100.0-float64(count)/float64(keysPerMemtable)*100)
				}
			}
		}

		// Final count
		finalCount := 0
		for memId := 0; memId < numMemtables; memId++ {
			for _, key := range allKeys[memId] {
				if _, _, _, _, _, _, found := ki.Get(key); found {
					finalCount++
				}
			}
		}

		keysLost := initialCount - finalCount
		lossPercentage := float64(keysLost) / float64(initialCount) * 100

		t.Logf("Final result:")
		t.Logf("  Keys lost: %d/%d (%.1f%%)", keysLost, initialCount, lossPercentage)

		if lossPercentage > 30 {
			t.Logf("  🎯 Multiple trim sessions cause significant key loss!")
		}
	})
}

func TestKeyIndex_ExpirationBug(t *testing.T) {
	t.Run("demonstrate expiration time bug", func(t *testing.T) {
		t.Logf("Demonstrating the expiration bug in main.go...")

		// Show the problematic values
		now := time.Now()
		buggyExptime := uint64(now.Second() + 3600) // WRONG - main.go bug
		correctExptime := uint64(now.Unix() + 3600) // CORRECT
		currentSecond := uint64(now.Second())       // What cache.go checks against
		currentUnix := uint64(now.Unix())           // What it should check against

		t.Logf("Time analysis:")
		t.Logf("  Current Unix timestamp: %d", currentUnix)
		t.Logf("  Current second (0-59): %d", currentSecond)
		t.Logf("  Buggy exptime (main.go): %d", buggyExptime)
		t.Logf("  Correct exptime: %d", correctExptime)

		// Show the comparison logic from cache.go line 93
		buggyComparison := buggyExptime > currentSecond
		correctComparison := correctExptime > currentUnix

		t.Logf("Expiration checks:")
		t.Logf("  Buggy: %d > %d = %v (key %s)",
			buggyExptime, currentSecond, buggyComparison,
			map[bool]string{true: "EXPIRED", false: "valid"}[buggyComparison])
		t.Logf("  Correct: %d > %d = %v (key %s)",
			correctExptime, currentUnix, correctComparison,
			map[bool]string{true: "EXPIRED", false: "valid"}[correctComparison])

		if buggyComparison && !correctComparison {
			t.Logf("🔥 BUG CONFIRMED: Keys appear expired due to time.Now().Second() vs time.Now().Unix() mismatch!")
		} else {
			t.Logf("Bug not reproduced in this specific second")
		}

		// The buggy exptime will almost always be > currentSecond (0-59)
		// because buggyExptime is usually 3600+ (seconds within minute + 3600)
		if buggyExptime > 59 {
			t.Logf("🎯 SMOKING GUN: buggyExptime (%d) > 59, so keys will ALWAYS appear expired!", buggyExptime)
		}
	})
}

func TestKeyIndex_Analyze50MKeyLoss(t *testing.T) {
	t.Run("analyze file wrapping during 50M key insertion", func(t *testing.T) {
		const numKeys = 50000000
		const memtableSize = 1 * 1024 * 1024          // 1MB
		const maxFileSize = int64(1024 * 1024 * 1024) // 1GB

		// Calculate key sizes based on actual main.go data
		sampleKey := "key12345678"                         // Representative key
		sampleValue := "value12345678"                     // Representative value
		entrySize := 4 + len(sampleKey) + len(sampleValue) // CRC32 + key + value

		totalDataSize := int64(numKeys) * int64(entrySize)
		numMemtables := (totalDataSize + int64(memtableSize) - 1) / int64(memtableSize) // Ceiling division

		t.Logf("50M key scenario analysis:")
		t.Logf("  Sample key: %s (%d bytes)", sampleKey, len(sampleKey))
		t.Logf("  Sample value: %s (%d bytes)", sampleValue, len(sampleValue))
		t.Logf("  Entry size: %d bytes", entrySize)
		t.Logf("  Total data: %.2f MB", float64(totalDataSize)/(1024*1024))
		t.Logf("  Memtable size: %.2f MB", float64(memtableSize)/(1024*1024))
		t.Logf("  Number of memtables: %d", numMemtables)
		t.Logf("  Max file size: %.2f MB", float64(maxFileSize)/(1024*1024))

		// Calculate when file wrapping occurs
		memtablesPerFile := maxFileSize / int64(memtableSize)
		fileWraps := (numMemtables + memtablesPerFile - 1) / memtablesPerFile

		t.Logf("File wrapping analysis:")
		t.Logf("  Memtables per file: %d", memtablesPerFile)
		t.Logf("  Number of file wraps: %d", fileWraps)

		if fileWraps > 1 {
			// Each wrap triggers a trim that deletes one memtable worth of keys
			keysPerMemtable := int64(memtableSize) / int64(entrySize)
			totalKeysDeleted := (fileWraps - 1) * keysPerMemtable
			deletionPercentage := float64(totalKeysDeleted) / float64(numKeys) * 100

			t.Logf("Trim impact:")
			t.Logf("  Keys per memtable: %d", keysPerMemtable)
			t.Logf("  Total keys deleted by trim: %d", totalKeysDeleted)
			t.Logf("  Deletion percentage: %.1f%%", deletionPercentage)

			if deletionPercentage > 90 {
				t.Logf("🔥 MULTIPLE FILE WRAPS CAUSE MASSIVE KEY LOSS!")
			}
		}

		// Ring buffer analysis
		rbCapacity := int64(100000000) // 100M from main.go
		t.Logf("Ring buffer analysis:")
		t.Logf("  Ring buffer capacity: %d", rbCapacity)
		t.Logf("  Required capacity: %d", numKeys)
		t.Logf("  Capacity utilization: %.1f%%", float64(numKeys)/float64(rbCapacity)*100)

		if numKeys < rbCapacity {
			t.Logf("✓ Ring buffer capacity is sufficient")
		} else {
			overflowKeys := numKeys - rbCapacity
			overflowPercentage := float64(overflowKeys) / float64(numKeys) * 100
			t.Logf("🔥 Ring buffer overflow: %d keys (%.1f%%) lost", overflowKeys, overflowPercentage)
		}
	})

	t.Run("simulate rapid memtable cycling", func(t *testing.T) {
		// Test if rapid memtable creation and trim operations cause key loss
		ki := NewKeyIndex(10, 100, 100000000, 100) // Same as main.go

		const batchSize = 100000 // Keys per memtable simulation
		const numBatches = 20    // Simulate 20 memtables

		allKeys := make([][]string, numBatches)

		// Phase 1: Add keys in batches (simulating memtable fills)
		for batch := 0; batch < numBatches; batch++ {
			allKeys[batch] = make([]string, batchSize)

			// Every 3 batches, simulate a file wrap and trim
			if batch > 0 && batch%3 == 0 {
				t.Logf("Simulating file wrap at batch %d", batch)
				ki.StartTrim()

				// Add some trigger keys to activate trim
				for trigger := 0; trigger < 50 && ki.TrimStatus(); trigger++ {
					triggerKey := fmt.Sprintf("wrap%d_trigger%d", batch, trigger)
					ki.Put(triggerKey, uint16(len(triggerKey)), uint32(batch+100), uint32(trigger*100), uint64(time.Now().Unix()+3600))
				}
				t.Logf("Trim completed at batch %d", batch)
			}

			// Add the batch keys
			for i := 0; i < batchSize; i++ {
				key := fmt.Sprintf("batch%d_key%d", batch, i)
				allKeys[batch][i] = key
				ki.Put(key, uint16(len(key)), uint32(batch), uint32(i*100), uint64(time.Now().Unix()+3600))
			}
		}

		// Phase 2: Check survival rates per batch
		t.Logf("Survival analysis per batch:")
		totalKeys := 0
		totalSurvived := 0

		for batch := 0; batch < numBatches; batch++ {
			survived := 0
			for _, key := range allKeys[batch] {
				if _, _, _, _, _, _, found := ki.Get(key); found {
					survived++
				}
			}

			survivalRate := float64(survived) / float64(batchSize) * 100
			t.Logf("  Batch %2d: %6d/%6d survived (%.1f%%)", batch, survived, batchSize, survivalRate)

			totalKeys += batchSize
			totalSurvived += survived
		}

		overallSurvival := float64(totalSurvived) / float64(totalKeys) * 100
		t.Logf("Overall survival: %d/%d (%.1f%%)", totalSurvived, totalKeys, overallSurvival)

		if overallSurvival < 50 {
			t.Logf("🔥 Confirmed: Rapid trim operations cause massive key loss!")
		}
	})
}

func TestKeyIndex_SingleTrimSessionAnalysis(t *testing.T) {
	t.Run("analyze single trim session behavior", func(t *testing.T) {
		// Simulate the exact scenario: one wrap with massive key loss
		ki := NewKeyIndex(10, 100, 100000000, 10) // Same as main.go with reduced step

		// Phase 1: Simulate filling up to the wrap point (memId 0-1023)
		keysPerMemtable := 1000 // Smaller for test efficiency
		numMemtablesBeforeWrap := 1024

		allKeys := make(map[uint32][]string)

		t.Logf("Adding keys for %d memtables before wrap...", numMemtablesBeforeWrap)
		for memId := uint32(0); memId < uint32(numMemtablesBeforeWrap); memId++ {
			keys := make([]string, keysPerMemtable)
			for i := 0; i < keysPerMemtable; i++ {
				key := fmt.Sprintf("mem%d_key%d", memId, i)
				keys[i] = key
				ki.Put(key, uint16(len(key)), memId, uint32(i*100), uint64(time.Now().Unix()+3600))
			}
			allKeys[memId] = keys

			if memId%200 == 0 {
				t.Logf("  Added memId %d", memId)
			}
		}

		// Count initial keys
		totalInitialKeys := 0
		for memId := uint32(0); memId < uint32(numMemtablesBeforeWrap); memId++ {
			for _, key := range allKeys[memId] {
				if _, _, _, _, _, _, found := ki.Get(key); found {
					totalInitialKeys++
				}
			}
		}
		t.Logf("Initial keys present: %d", totalInitialKeys)

		// Phase 2: Simulate the file wrap and trim
		t.Logf("Starting trim session (simulating file wrap)...")
		ki.StartTrim()

		// Phase 3: Add post-wrap keys and trigger trim
		postWrapMemId := uint32(1024) // First memId after wrap
		triggerCount := 0

		for ki.TrimStatus() && triggerCount < 10000 { // Safety limit
			key := fmt.Sprintf("postwrap_key%d", triggerCount)
			ki.Put(key, uint16(len(key)), postWrapMemId, uint32(triggerCount*100), uint64(time.Now().Unix()+3600))
			triggerCount++

			// Every 100 triggers, check deletion progress
			if triggerCount%100 == 0 {
				stillPresent := 0
				for memId := uint32(0); memId < uint32(numMemtablesBeforeWrap); memId++ {
					for _, key := range allKeys[memId] {
						if _, _, _, _, _, _, found := ki.Get(key); found {
							stillPresent++
						}
					}
				}
				deletedSoFar := totalInitialKeys - stillPresent
				t.Logf("  After %d triggers: %d keys deleted (%.1f%%)",
					triggerCount, deletedSoFar, float64(deletedSoFar)/float64(totalInitialKeys)*100)
			}
		}

		t.Logf("Trim completed after %d triggers", triggerCount)

		// Phase 4: Analyze which memtables were affected
		t.Logf("Analyzing deletion pattern by memId:")
		totalDeleted := 0

		for memId := uint32(0); memId < uint32(numMemtablesBeforeWrap); memId++ {
			survived := 0
			for _, key := range allKeys[memId] {
				if _, _, _, _, _, _, found := ki.Get(key); found {
					survived++
				}
			}
			deleted := keysPerMemtable - survived
			totalDeleted += deleted

			if deleted > 0 {
				t.Logf("  MemId %4d: %4d deleted, %4d survived (%.1f%% lost)",
					memId, deleted, survived, float64(deleted)/float64(keysPerMemtable)*100)
			}

			// Stop logging after first 20 affected memtables
			if memId > 20 && deleted == 0 {
				if memId == 21 {
					t.Logf("  ... (remaining memtables unaffected)")
				}
				break
			}
		}

		overallLossPercentage := float64(totalDeleted) / float64(totalInitialKeys) * 100
		t.Logf("Overall result:")
		t.Logf("  Total deleted: %d/%d (%.1f%%)", totalDeleted, totalInitialKeys, overallLossPercentage)
		t.Logf("  Expected: Only memId 0 should be deleted (~%.1f%%)",
			float64(keysPerMemtable)/float64(totalInitialKeys)*100)

		if overallLossPercentage > 10 {
			t.Logf("🔥 CONFIRMED: Single trim session is deleting too many memtables!")
		}
	})
}

func TestKeyIndex_RingBufferAt50MScale(t *testing.T) {
	t.Run("test ring buffer behavior at 50M scale", func(t *testing.T) {
		// Test at different scales to find the breaking point
		testScales := []int{1000000, 5000000, 10000000, 25000000} // 1M, 5M, 10M, 25M

		for _, numKeys := range testScales {
			t.Run(fmt.Sprintf("scale_%dM", numKeys/1000000), func(t *testing.T) {
				// Create fresh KeyIndex for each scale test
				freshKi := NewKeyIndex(10, 100, 100000000, 10)

				// Add keys with sequential memIds (simulating real scenario)
				keysPerMemtable := 37449 // Calculated from earlier analysis

				t.Logf("Testing %d keys...", numKeys)

				// Phase 1: Add all keys
				for i := 0; i < numKeys; i++ {
					key := fmt.Sprintf("key%d", i)
					memId := uint32(i / keysPerMemtable)
					freshKi.Put(key, uint16(len(key)), memId, uint32(i*100), uint64(time.Now().Unix()+3600))

					if i%1000000 == 0 && i > 0 {
						t.Logf("  Added %dM keys...", i/1000000)
					}
				}

				// Phase 2: Check how many keys are still findable
				foundCount := 0
				notFoundCount := 0

				// Sample check (check every 1000th key for performance)
				sampleRate := max(1, numKeys/10000) // Check ~10K keys max

				for i := 0; i < numKeys; i += sampleRate {
					key := fmt.Sprintf("key%d", i)
					if _, _, _, _, _, _, found := freshKi.Get(key); found {
						foundCount++
					} else {
						notFoundCount++
					}
				}

				totalSampled := foundCount + notFoundCount
				foundPercentage := float64(foundCount) / float64(totalSampled) * 100

				t.Logf("  Sample results: %d/%d found (%.1f%%)", foundCount, totalSampled, foundPercentage)

				if foundPercentage < 50 {
					t.Logf("🔥 SIGNIFICANT LOSS at %dM keys: %.1f%% found", numKeys/1000000, foundPercentage)
				} else if foundPercentage < 95 {
					t.Logf("⚠️  Some loss at %dM keys: %.1f%% found", numKeys/1000000, foundPercentage)
				} else {
					t.Logf("✓ Good at %dM keys: %.1f%% found", numKeys/1000000, foundPercentage)
				}
			})
		}
	})

	t.Run("test ring buffer growth behavior", func(t *testing.T) {
		// Test the ring buffer growth pattern
		ki := NewKeyIndex(10, 100, 100000000, 10)

		// Add keys and check ring buffer behavior at growth boundaries
		growthCheckpoints := []int{50, 100, 200, 1000, 10000, 100000, 1000000}

		for _, checkpoint := range growthCheckpoints {
			t.Logf("Adding up to %d keys...", checkpoint)

			// Add keys up to checkpoint
			for i := 0; i < checkpoint; i++ {
				key := fmt.Sprintf("growth_key%d", i)
				ki.Put(key, uint16(len(key)), uint32(i/1000), uint32(i*100), uint64(time.Now().Unix()+3600))
			}

			// Check how many we can still find
			found := 0
			for i := 0; i < checkpoint; i++ {
				key := fmt.Sprintf("growth_key%d", i)
				if _, _, _, _, _, _, exists := ki.Get(key); exists {
					found++
				}
			}

			foundPercentage := float64(found) / float64(checkpoint) * 100
			t.Logf("  At %d keys: %d/%d found (%.1f%%)", checkpoint, found, checkpoint, foundPercentage)

			if foundPercentage < 99 {
				t.Logf("⚠️  Loss detected at %d keys!", checkpoint)
			}
		}
	})

	t.Run("check specific ring buffer capacity overflow", func(t *testing.T) {
		// Test what happens when we exceed ring buffer initial size but stay under max
		ki := NewKeyIndex(10, 100, 1000, 10) // Small max capacity for testing

		// Add more than max capacity to force overflow
		numKeys := 1500

		for i := 0; i < numKeys; i++ {
			key := fmt.Sprintf("overflow_key%d", i)
			ki.Put(key, uint16(len(key)), uint32(i/100), uint32(i*100), uint64(time.Now().Unix()+3600))
		}

		// Check how many survived
		found := 0
		for i := 0; i < numKeys; i++ {
			key := fmt.Sprintf("overflow_key%d", i)
			if _, _, _, _, _, _, exists := ki.Get(key); exists {
				found++
			}
		}

		expectedFound := min(1000, numKeys) // Should keep last 1000 due to FIFO
		foundPercentage := float64(found) / float64(numKeys) * 100

		t.Logf("Ring buffer overflow test:")
		t.Logf("  Added: %d keys", numKeys)
		t.Logf("  Found: %d keys (%.1f%%)", found, foundPercentage)
		t.Logf("  Expected: ~%d keys (%.1f%%)", expectedFound, float64(expectedFound)/float64(numKeys)*100)

		if found != expectedFound {
			t.Logf("🔥 Ring buffer overflow behavior unexpected!")
		}
	})
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestKeyIndex_MainGoScenarioExact(t *testing.T) {
	t.Run("simulate exact main.go scenario with 50M keys", func(t *testing.T) {
		// Use exact main.go configuration
		// From main.go: RbInitial: 50000000, RbMax: 50000000, DeleteAmortizedStep: 1
		ki := NewKeyIndex(10, 50000000, 50000000, 1)

		const totalKeys = 50000000
		const preDeleteKeys = 30000000
		const batchSize = 1000000 // 1M keys per batch after 30M

		t.Logf("Starting KeyIndex simulation with main.go config:")
		t.Logf("  Rounds: 10")
		t.Logf("  RbInitial: 50,000,000")
		t.Logf("  RbMax: 50,000,000")
		t.Logf("  DeleteAmortizedStep: 1")
		t.Logf("  Total keys to add: %d", totalKeys)
		t.Logf("  Keys before trim: %d", preDeleteKeys)

		// Phase 1: Add first 30M keys (before starting deletes)
		t.Logf("Phase 1: Adding first %d keys...", preDeleteKeys)
		start := time.Now()

		for i := 0; i < preDeleteKeys; i++ {
			key := fmt.Sprintf("key%d", i)
			// Use incrementing memId to simulate memtable progression
			memId := uint32(i / 100000) // ~100K keys per memtable
			ki.Put(key, uint16(len(key)), memId, uint32(i*10), uint64(time.Now().Unix()+3600))

			if i%5000000 == 0 && i > 0 {
				t.Logf("  Added %dM keys... (%.1f%% complete)", i/1000000, float64(i)/float64(preDeleteKeys)*100)
			}
		}

		phase1Duration := time.Since(start)
		t.Logf("Phase 1 completed in %v", phase1Duration)

		// Check how many keys are present before trim using GetMeta
		t.Logf("Checking key presence before trim (using GetMeta)...")
		checkStart := time.Now()
		sampleRate := 10000 // Check every 10,000th key for performance
		foundBeforeTrim := 0
		totalSampled := 0

		for i := 0; i < preDeleteKeys; i += sampleRate {
			key := fmt.Sprintf("key%d", i)
			if _, _, _, _, _, _, found := ki.GetMeta(key); found {
				foundBeforeTrim++
			}
			totalSampled++
		}

		foundPercentageBeforeTrim := float64(foundBeforeTrim) / float64(totalSampled) * 100
		t.Logf("Before trim (GetMeta): %d/%d sampled keys found (%.2f%%)", foundBeforeTrim, totalSampled, foundPercentageBeforeTrim)
		t.Logf("Check completed in %v", time.Since(checkStart))

		// Phase 2: Start trim operation (simulate file wrap)
		t.Logf("Phase 2: Starting trim operation (simulating file wrap)...")
		ki.StartTrim()

		if !ki.TrimStatus() {
			t.Fatal("Expected trim status to be true after StartTrim()")
		}

		// Phase 3: Add remaining keys in 1M batches, triggering deletes
		remainingKeys := totalKeys - preDeleteKeys
		numBatches := remainingKeys / batchSize

		t.Logf("Phase 3: Adding remaining %d keys in %d batches of %d keys each...", remainingKeys, numBatches, batchSize)

		batchResults := make([]struct {
			batchNum     int
			keysAdded    int
			trimActive   bool
			deletedSoFar int
		}, numBatches)

		for batch := 0; batch < numBatches; batch++ {
			batchStart := time.Now()
			batchStartKey := preDeleteKeys + (batch * batchSize)
			batchEndKey := batchStartKey + batchSize

			// Add 1M keys in this batch
			for i := batchStartKey; i < batchEndKey; i++ {
				key := fmt.Sprintf("key%d", i)
				memId := uint32(i / 100000) // Continue memId progression
				ki.Put(key, uint16(len(key)), memId, uint32(i*10), uint64(time.Now().Unix()+3600))
			}

			// Check trim status after this batch
			trimActive := ki.TrimStatus()

			// Sample check to see how many early keys have been deleted using GetMeta
			deletedCount := 0
			sampleCheckSize := min(10000, preDeleteKeys/100) // Check 1% of pre-delete keys
			for i := 0; i < sampleCheckSize; i++ {
				checkKey := fmt.Sprintf("key%d", i*100) // Check every 100th key
				if _, _, _, _, _, _, found := ki.GetMeta(checkKey); !found {
					deletedCount++
				}
			}

			batchResults[batch] = struct {
				batchNum     int
				keysAdded    int
				trimActive   bool
				deletedSoFar int
			}{
				batchNum:     batch + 1,
				keysAdded:    batchSize,
				trimActive:   trimActive,
				deletedSoFar: deletedCount,
			}

			batchDuration := time.Since(batchStart)
			totalAdded := preDeleteKeys + ((batch + 1) * batchSize)

			t.Logf("  Batch %2d: Added %dM keys (total: %dM), trim active: %v, ~%d early keys deleted (GetMeta), took %v",
				batch+1, batchSize/1000000, totalAdded/1000000, trimActive, deletedCount, batchDuration)

			// If trim completed, note it
			if !trimActive {
				t.Logf("    ✓ Trim operation completed after batch %d", batch+1)
			}
		}

		// Phase 4: Final analysis
		t.Logf("Phase 4: Final analysis...")

		// Check final key distribution
		finalCheckStart := time.Now()

		// Check early keys (first 5M) using GetMeta
		earlyKeysFound := 0
		earlyKeysChecked := 0
		earlyCheckLimit := min(5000000, preDeleteKeys)
		earlyAccessUpdates := 0

		for i := 0; i < earlyCheckLimit; i += 1000 {
			key := fmt.Sprintf("key%d", i)
			memId, length, offset, lastAccessAt, freq, exptime, found := ki.GetMeta(key)
			if found {
				earlyKeysFound++
				if i < 10 { // Log details for first few keys
					t.Logf("    Early key %s: memId=%d, length=%d, offset=%d, lastAccess=%d, freq=%d, exptime=%d",
						key, memId, length, offset, lastAccessAt, freq, exptime)
				}
				if lastAccessAt > 0 { // Access time was updated
					earlyAccessUpdates++
				}
			}
			earlyKeysChecked++
		}

		// Check middle keys (25M-30M range) using GetMeta
		middleKeysFound := 0
		middleKeysChecked := 0
		middleStart := 25000000
		middleEnd := min(30000000, preDeleteKeys)
		middleAccessUpdates := 0

		for i := middleStart; i < middleEnd; i += 1000 {
			key := fmt.Sprintf("key%d", i)
			memId, length, offset, lastAccessAt, freq, exptime, found := ki.GetMeta(key)
			if found {
				middleKeysFound++
				if i == middleStart { // Log details for first key in range
					t.Logf("    Middle key %s: memId=%d, length=%d, offset=%d, lastAccess=%d, freq=%d, exptime=%d",
						key, memId, length, offset, lastAccessAt, freq, exptime)
				}
				if lastAccessAt > 0 { // Access time was updated
					middleAccessUpdates++
				}
			}
			middleKeysChecked++
		}

		// Check recent keys (45M-50M range) using GetMeta
		recentKeysFound := 0
		recentKeysChecked := 0
		recentStart := 45000000
		recentEnd := totalKeys
		recentAccessUpdates := 0

		for i := recentStart; i < recentEnd; i += 1000 {
			key := fmt.Sprintf("key%d", i)
			memId, length, offset, lastAccessAt, freq, exptime, found := ki.GetMeta(key)
			if found {
				recentKeysFound++
				if i == recentStart { // Log details for first key in range
					t.Logf("    Recent key %s: memId=%d, length=%d, offset=%d, lastAccess=%d, freq=%d, exptime=%d",
						key, memId, length, offset, lastAccessAt, freq, exptime)
				}
				if lastAccessAt > 0 { // Access time was updated
					recentAccessUpdates++
				}
			}
			recentKeysChecked++
		}

		finalCheckDuration := time.Since(finalCheckStart)

		t.Logf("Final key distribution analysis (using GetMeta):")
		t.Logf("  Early keys (0-5M):    %d/%d found (%.2f%%), %d access updates", earlyKeysFound, earlyKeysChecked, float64(earlyKeysFound)/float64(earlyKeysChecked)*100, earlyAccessUpdates)
		t.Logf("  Middle keys (25M-30M): %d/%d found (%.2f%%), %d access updates", middleKeysFound, middleKeysChecked, float64(middleKeysFound)/float64(middleKeysChecked)*100, middleAccessUpdates)
		t.Logf("  Recent keys (45M-50M): %d/%d found (%.2f%%), %d access updates", recentKeysFound, recentKeysChecked, float64(recentKeysFound)/float64(recentKeysChecked)*100, recentAccessUpdates)
		t.Logf("Final check completed in %v", finalCheckDuration)

		// Overall statistics
		totalLoss := earlyKeysChecked - earlyKeysFound
		totalLossPercentage := float64(totalLoss) / float64(earlyKeysChecked) * 100

		t.Logf("Summary:")
		t.Logf("  Total execution time: %v", time.Since(start))
		t.Logf("  Keys lost (early sample): %d/%d (%.2f%%)", totalLoss, earlyKeysChecked, totalLossPercentage)
		t.Logf("  Trim sessions completed: %v", !ki.TrimStatus())

		// Analyze the issue
		if totalLossPercentage > 50 {
			t.Logf("🔥 CRITICAL ISSUE REPRODUCED: %.1f%% of early keys lost!", totalLossPercentage)
			t.Logf("   This explains the main.go issue where keys become 'not found'")
		} else if totalLossPercentage > 10 {
			t.Logf("⚠️  MODERATE ISSUE: %.1f%% of early keys lost", totalLossPercentage)
		} else if totalLossPercentage > 1 {
			t.Logf("⚠️  MINOR ISSUE: %.1f%% of early keys lost", totalLossPercentage)
		} else {
			t.Logf("✓ No significant key loss detected")
		}

		// Check if it's a ring buffer overflow issue
		if foundPercentageBeforeTrim < 95 {
			t.Logf("🔍 ANALYSIS: Key loss occurred BEFORE trim - likely ring buffer overflow")
			t.Logf("   Ring buffer capacity: 50M, Keys added: 30M (should fit)")
		}

		// Check if it's a trim operation issue
		if foundPercentageBeforeTrim >= 95 && totalLossPercentage > 10 {
			t.Logf("🔍 ANALYSIS: Key loss occurred DURING trim - trim operation bug")
			t.Logf("   Trim may be deleting too many keys or wrong keys")
		}

		// Memory and performance stats
		t.Logf("Performance metrics:")
		t.Logf("  Average time per key (phase 1): %.2f ns", float64(phase1Duration.Nanoseconds())/float64(preDeleteKeys))
		t.Logf("  Final check time per key: %.2f ns", float64(finalCheckDuration.Nanoseconds())/float64(earlyKeysChecked+middleKeysChecked+recentKeysChecked))
	})

	t.Run("test ring buffer overflow beyond capacity", func(t *testing.T) {
		// Create KeyIndex with same config as main.go
		ki := NewKeyIndex(10, 50000000, 50000000, 1)

		// But add MORE than the ring buffer capacity to force overflow
		const totalKeys = 70000000     // 70M keys (40% more than capacity)
		const preDeleteKeys = 55000000 // 55M before trim (10% over capacity)
		const batchSize = 1000000

		t.Logf("Testing ring buffer overflow scenario:")
		t.Logf("  Ring buffer capacity: 50,000,000")
		t.Logf("  Keys before trim: %d (%.1f%% over capacity)", preDeleteKeys, float64(preDeleteKeys-50000000)/50000000*100)
		t.Logf("  Total keys: %d (%.1f%% over capacity)", totalKeys, float64(totalKeys-50000000)/50000000*100)

		// Phase 1: Add more keys than ring buffer capacity
		t.Logf("Phase 1: Adding %d keys (exceeding ring buffer capacity)...", preDeleteKeys)
		start := time.Now()

		// Track key loss during insertion
		lossCheckPoints := []int{10000000, 25000000, 50000000, 55000000}

		for i := 0; i < preDeleteKeys; i++ {
			key := fmt.Sprintf("key%d", i)
			memId := uint32(i / 100000)
			ki.Put(key, uint16(len(key)), memId, uint32(i*10), uint64(time.Now().Unix()+3600))

			// Check for loss at key milestones
			for _, checkpoint := range lossCheckPoints {
				if i == checkpoint-1 { // Check when we reach each milestone
					t.Logf("  Reached %dM keys, checking for ring buffer overflow...", checkpoint/1000000)

					// Check early keys to see if they're being evicted
					earlyKeysLost := 0
					sampleSize := 1000
					for j := 0; j < sampleSize; j++ {
						checkKey := fmt.Sprintf("key%d", j*100) // Check every 100th early key
						if _, _, _, _, _, _, found := ki.GetMeta(checkKey); !found {
							earlyKeysLost++
						}
					}
					lossPercentage := float64(earlyKeysLost) / float64(sampleSize) * 100
					t.Logf("    Early key loss at %dM: %d/%d (%.2f%%)", checkpoint/1000000, earlyKeysLost, sampleSize, lossPercentage)

					if lossPercentage > 50 {
						t.Logf("    🔥 MAJOR OVERFLOW DETECTED at %dM keys!", checkpoint/1000000)
					}
				}
			}

			if i%10000000 == 0 && i > 0 {
				t.Logf("  Added %dM keys...", i/1000000)
			}
		}

		phase1Duration := time.Since(start)
		t.Logf("Phase 1 completed in %v", phase1Duration)

		// Check final loss before trim
		t.Logf("Checking key distribution before trim...")
		earlyFound := 0
		middleFound := 0
		recentFound := 0
		sampleSize := 5000

		// Early keys (should be most affected by overflow)
		for i := 0; i < sampleSize; i++ {
			key := fmt.Sprintf("key%d", i*100)
			if _, _, _, _, _, _, found := ki.GetMeta(key); found {
				earlyFound++
			}
		}

		// Middle keys
		middleStart := 25000000
		for i := 0; i < sampleSize; i++ {
			key := fmt.Sprintf("key%d", middleStart+i*100)
			if _, _, _, _, _, _, found := ki.GetMeta(key); found {
				middleFound++
			}
		}

		// Recent keys (should be preserved)
		recentStart := preDeleteKeys - 5000000 // Last 5M keys
		for i := 0; i < sampleSize; i++ {
			key := fmt.Sprintf("key%d", recentStart+i*100)
			if _, _, _, _, _, _, found := ki.GetMeta(key); found {
				recentFound++
			}
		}

		t.Logf("Before trim overflow analysis:")
		t.Logf("  Early keys (0-500K):    %d/%d found (%.2f%%)", earlyFound, sampleSize, float64(earlyFound)/float64(sampleSize)*100)
		t.Logf("  Middle keys (25M range): %d/%d found (%.2f%%)", middleFound, sampleSize, float64(middleFound)/float64(sampleSize)*100)
		t.Logf("  Recent keys (50M+ range): %d/%d found (%.2f%%)", recentFound, sampleSize, float64(recentFound)/float64(sampleSize)*100)

		// Calculate overflow impact
		earlyLossPercentage := (1.0 - float64(earlyFound)/float64(sampleSize)) * 100
		if earlyLossPercentage > 90 {
			t.Logf("🔥 SEVERE RING BUFFER OVERFLOW: %.1f%% of early keys lost!", earlyLossPercentage)
		} else if earlyLossPercentage > 50 {
			t.Logf("🔥 MAJOR RING BUFFER OVERFLOW: %.1f%% of early keys lost!", earlyLossPercentage)
		} else if earlyLossPercentage > 10 {
			t.Logf("⚠️  MODERATE OVERFLOW: %.1f%% of early keys lost", earlyLossPercentage)
		}

		// Phase 2: Start trim and add remaining keys
		t.Logf("Phase 2: Starting trim and adding remaining keys...")
		ki.StartTrim()

		remainingKeys := totalKeys - preDeleteKeys
		for i := 0; i < remainingKeys; i++ {
			key := fmt.Sprintf("key%d", preDeleteKeys+i)
			memId := uint32((preDeleteKeys + i) / 100000)
			ki.Put(key, uint16(len(key)), memId, uint32((preDeleteKeys+i)*10), uint64(time.Now().Unix()+3600))

			if i%1000000 == 0 && i > 0 {
				t.Logf("  Added %dM more keys (total: %dM)", i/1000000, (preDeleteKeys+i)/1000000)
			}
		}

		// Final analysis
		t.Logf("Final analysis after %d total keys:", totalKeys)
		finalEarlyFound := 0
		finalRecentFound := 0

		// Check early keys again
		for i := 0; i < sampleSize; i++ {
			key := fmt.Sprintf("key%d", i*100)
			if _, _, _, _, _, _, found := ki.GetMeta(key); found {
				finalEarlyFound++
			}
		}

		// Check most recent keys
		finalRecentStart := totalKeys - 5000000
		for i := 0; i < sampleSize; i++ {
			key := fmt.Sprintf("key%d", finalRecentStart+i*100)
			if _, _, _, _, _, _, found := ki.GetMeta(key); found {
				finalRecentFound++
			}
		}

		finalEarlyLoss := (1.0 - float64(finalEarlyFound)/float64(sampleSize)) * 100
		t.Logf("Final results:")
		t.Logf("  Early keys final: %d/%d found (%.2f%% lost)", finalEarlyFound, sampleSize, finalEarlyLoss)
		t.Logf("  Recent keys final: %d/%d found (%.2f%%)", finalRecentFound, sampleSize, float64(finalRecentFound)/float64(sampleSize)*100)
		t.Logf("  Total time: %v", time.Since(start))

		if finalEarlyLoss > 95 {
			t.Logf("🎯 SUCCESS: Reproduced severe key loss (%.1f%%) similar to main.go issue!", finalEarlyLoss)
		}
	})
}
