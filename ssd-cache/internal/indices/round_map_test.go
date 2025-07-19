package indices

import (
	"testing"
)

func TestRoundMap_AddGetMetaRemove(t *testing.T) {
	rm := NewTestRoundMap()

	t.Run("Basic Add and GetMeta", func(t *testing.T) {
		hash := uint32(0x123456)
		memTableId := uint32(100)
		offset := uint32(2048)
		length := uint16(512)
		fingerprint28 := uint32(0xABCDEF0)
		freq := uint32(5)
		ttl := uint32(3600)
		lastAccess := uint32(1640995200)

		// Add entry
		overwritten := rm.bitmap.Add(hash, memTableId, offset, length, fingerprint28, freq, ttl, lastAccess)
		if overwritten {
			t.Error("Expected no overwrite for new entry")
		}

		// Get and verify metadata
		found, retMemTableId, retOffset, retLength, retFingerprint28, retFreq, retTtl, retLastAccess := rm.bitmap.GetMeta(hash)
		if !found {
			t.Fatal("Expected entry to be found")
		}

		if retMemTableId != memTableId {
			t.Errorf("MemTableId: expected %d, got %d", memTableId, retMemTableId)
		}
		if retOffset != offset {
			t.Errorf("Offset: expected %d, got %d", offset, retOffset)
		}
		if retLength != length {
			t.Errorf("Length: expected %d, got %d", length, retLength)
		}
		if retFingerprint28 != fingerprint28 {
			t.Errorf("Fingerprint28: expected 0x%x, got 0x%x", fingerprint28, retFingerprint28)
		}
		if retFreq != freq {
			t.Errorf("Freq: expected %d, got %d", freq, retFreq)
		}
		if retTtl != ttl {
			t.Errorf("TTL: expected %d, got %d", ttl, retTtl)
		}
		if retLastAccess != lastAccess {
			t.Errorf("LastAccess: expected %d, got %d", lastAccess, retLastAccess)
		}

		// Clean up
		rm.bitmap.Remove(hash)
	})

	t.Run("Remove Entry", func(t *testing.T) {
		hash := uint32(0x789ABC)
		memTableId := uint32(200)
		offset := uint32(4096)
		length := uint16(1024)
		fingerprint28 := uint32(0x1234567)
		freq := uint32(10)
		ttl := uint32(7200)
		lastAccess := uint32(1640995300)

		// Add entry
		rm.bitmap.Add(hash, memTableId, offset, length, fingerprint28, freq, ttl, lastAccess)

		// Verify it exists
		found, _, _, _, _, _, _, _ := rm.bitmap.GetMeta(hash)
		if !found {
			t.Fatal("Expected entry to be found before removal")
		}

		// Remove entry
		rm.bitmap.Remove(hash)

		// Verify it's gone
		found, _, _, _, _, _, _, _ = rm.bitmap.GetMeta(hash)
		if found {
			t.Error("Expected entry to be removed")
		}
	})

	t.Run("Same Hash Same Fingerprint", func(t *testing.T) {
		hash := uint32(0xDEF123)
		memTableId1 := uint32(300)
		memTableId2 := uint32(400)
		offset := uint32(8192)
		length := uint16(256)
		fingerprint28 := uint32(0xFEDCBA9) // Same fingerprint
		freq := uint32(1)
		ttl := uint32(1800)
		lastAccess := uint32(1640995400)

		// Add first entry
		overwritten1 := rm.bitmap.Add(hash, memTableId1, offset, length, fingerprint28, freq, ttl, lastAccess)
		if overwritten1 {
			t.Error("Expected no overwrite for first entry")
		}

		// Add second entry with same hash and fingerprint (updating same key, not a collision)
		overwritten2 := rm.bitmap.Add(hash, memTableId2, offset, length, fingerprint28, freq, ttl, lastAccess)
		if overwritten2 {
			t.Error("Expected no overwrite when updating same key (same hash + same fingerprint)")
		}

		// Verify the second entry is stored (last write wins)
		found, retMemTableId, _, _, _, _, _, _ := rm.bitmap.GetMeta(hash)
		if !found {
			t.Fatal("Expected entry to be found")
		}
		if retMemTableId != memTableId2 {
			t.Errorf("Expected memTableId %d (second write), got %d", memTableId2, retMemTableId)
		}
	})

	t.Run("Same Hash Different Fingerprint", func(t *testing.T) {
		hash := uint32(0x456789)
		memTableId1 := uint32(500)
		memTableId2 := uint32(600)
		offset := uint32(16384)
		length := uint16(128)
		fingerprint28_1 := uint32(0x1111111)
		fingerprint28_2 := uint32(0x2222222) // Different fingerprint
		freq := uint32(2)
		ttl := uint32(900)
		lastAccess := uint32(1640995500)

		// Add first entry
		overwritten1 := rm.bitmap.Add(hash, memTableId1, offset, length, fingerprint28_1, freq, ttl, lastAccess)
		if overwritten1 {
			t.Error("Expected no overwrite for first entry")
		}

		// Add second entry with same hash but different fingerprint (collision - different key)
		overwritten2 := rm.bitmap.Add(hash, memTableId2, offset, length, fingerprint28_2, freq, ttl, lastAccess)
		if !overwritten2 {
			t.Error("Expected overwrite when collision occurs (same hash + different fingerprint)")
		}

		// Verify the second entry is stored
		found, retMemTableId, _, _, retFingerprint28, _, _, _ := rm.bitmap.GetMeta(hash)
		if !found {
			t.Fatal("Expected entry to be found")
		}
		if retMemTableId != memTableId2 {
			t.Errorf("Expected memTableId %d (collision overwrite), got %d", memTableId2, retMemTableId)
		}
		if retFingerprint28 != fingerprint28_2 {
			t.Errorf("Expected fingerprint 0x%x (collision overwrite), got 0x%x", fingerprint28_2, retFingerprint28)
		}
	})

	t.Run("Boundary Values", func(t *testing.T) {
		testCases := []struct {
			name          string
			hash          uint32
			memTableId    uint32
			offset        uint32
			length        uint16
			fingerprint28 uint32
			freq          uint32
			ttl           uint32
			lastAccess    uint32
		}{
			{
				"Maximum values",
				0xFFFFFF,
				0xFFFFFFFF,
				0xFFFFFFFF,
				0xFFFF,
				0xFFFFFFF,
				0xFFFFF,
				0xFFFFFFFF,
				0xFFFFFFFF,
			},
			{
				"Minimum values",
				0,
				0,
				0,
				0,
				0,
				0,
				0,
				0,
			},
			{
				"Mixed boundaries",
				0x123456,
				0x80000000,
				0x40000000,
				0x8000,
				0x8000000,
				0x80000,
				0x80000000,
				0x40000000,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Add entry
				overwritten := rm.bitmap.Add(tc.hash, tc.memTableId, tc.offset, tc.length, tc.fingerprint28, tc.freq, tc.ttl, tc.lastAccess)
				if overwritten {
					t.Error("Expected no overwrite for new entry")
				}

				// Get and verify metadata
				found, retMemTableId, retOffset, retLength, retFingerprint28, retFreq, retTtl, retLastAccess := rm.bitmap.GetMeta(tc.hash)
				if !found {
					t.Fatal("Expected entry to be found")
				}

				// Verify all fields (accounting for potential bit masking)
				if retMemTableId != tc.memTableId {
					t.Errorf("MemTableId: expected %d, got %d", tc.memTableId, retMemTableId)
				}
				if retOffset != tc.offset {
					t.Errorf("Offset: expected %d, got %d", tc.offset, retOffset)
				}
				if retLength != tc.length {
					t.Errorf("Length: expected %d, got %d", tc.length, retLength)
				}
				// Fingerprint should be masked to 28 bits
				expectedFingerprint := tc.fingerprint28 & _LO_28BIT_IN_32BIT
				if retFingerprint28 != expectedFingerprint {
					t.Errorf("Fingerprint28: expected 0x%x, got 0x%x", expectedFingerprint, retFingerprint28)
				}
				// Freq should be masked to 20 bits
				expectedFreq := tc.freq & _LO_20BIT_IN_32BIT
				if retFreq != expectedFreq {
					t.Errorf("Freq: expected %d, got %d", expectedFreq, retFreq)
				}
				if retTtl != tc.ttl {
					t.Errorf("TTL: expected %d, got %d", tc.ttl, retTtl)
				}
				if retLastAccess != tc.lastAccess {
					t.Errorf("LastAccess: expected %d, got %d", tc.lastAccess, retLastAccess)
				}

				// Clean up
				rm.bitmap.Remove(tc.hash)
			})
		}
	})

	t.Run("Multiple Entries Different Hashes", func(t *testing.T) {
		entries := []struct {
			hash          uint32
			memTableId    uint32
			offset        uint32
			length        uint16
			fingerprint28 uint32
			freq          uint32
			ttl           uint32
			lastAccess    uint32
		}{
			{0x111111, 100, 1000, 100, 0x1111111, 1, 3600, 1640995200},
			{0x222222, 200, 2000, 200, 0x2222222, 2, 7200, 1640995300},
			{0x333333, 300, 3000, 300, 0x3333333, 3, 1800, 1640995400},
			{0x444444, 400, 4000, 400, 0x4444444, 4, 900, 1640995500},
		}

		// Add all entries
		for _, entry := range entries {
			overwritten := rm.bitmap.Add(entry.hash, entry.memTableId, entry.offset, entry.length, entry.fingerprint28, entry.freq, entry.ttl, entry.lastAccess)
			if overwritten {
				t.Errorf("Expected no overwrite for hash 0x%x", entry.hash)
			}
		}

		// Verify all entries exist and have correct data
		for _, entry := range entries {
			found, retMemTableId, retOffset, retLength, retFingerprint28, retFreq, retTtl, retLastAccess := rm.bitmap.GetMeta(entry.hash)
			if !found {
				t.Errorf("Expected entry with hash 0x%x to be found", entry.hash)
				continue
			}

			if retMemTableId != entry.memTableId {
				t.Errorf("Hash 0x%x - MemTableId: expected %d, got %d", entry.hash, entry.memTableId, retMemTableId)
			}
			if retOffset != entry.offset {
				t.Errorf("Hash 0x%x - Offset: expected %d, got %d", entry.hash, entry.offset, retOffset)
			}
			if retLength != entry.length {
				t.Errorf("Hash 0x%x - Length: expected %d, got %d", entry.hash, entry.length, retLength)
			}
			if retFingerprint28 != entry.fingerprint28 {
				t.Errorf("Hash 0x%x - Fingerprint28: expected 0x%x, got 0x%x", entry.hash, entry.fingerprint28, retFingerprint28)
			}
			if retFreq != entry.freq {
				t.Errorf("Hash 0x%x - Freq: expected %d, got %d", entry.hash, entry.freq, retFreq)
			}
			if retTtl != entry.ttl {
				t.Errorf("Hash 0x%x - TTL: expected %d, got %d", entry.hash, entry.ttl, retTtl)
			}
			if retLastAccess != entry.lastAccess {
				t.Errorf("Hash 0x%x - LastAccess: expected %d, got %d", entry.hash, entry.lastAccess, retLastAccess)
			}
		}

		// Remove entries one by one and verify others remain
		for i, entryToRemove := range entries {
			rm.bitmap.Remove(entryToRemove.hash)

			// Verify removed entry is gone
			found, _, _, _, _, _, _, _ := rm.bitmap.GetMeta(entryToRemove.hash)
			if found {
				t.Errorf("Expected entry with hash 0x%x to be removed", entryToRemove.hash)
			}

			// Verify remaining entries still exist
			for j, remainingEntry := range entries {
				if j <= i {
					continue // This entry should be removed
				}
				found, _, _, _, _, _, _, _ := rm.bitmap.GetMeta(remainingEntry.hash)
				if !found {
					t.Errorf("Expected remaining entry with hash 0x%x to still exist", remainingEntry.hash)
				}
			}
		}
	})

	t.Run("Collision vs Same Key Update", func(t *testing.T) {
		hash := uint32(0xABCDEF)
		fingerprint1 := uint32(0x1111111)
		fingerprint2 := uint32(0x2222222)

		// First entry
		overwritten1 := rm.bitmap.Add(hash, 100, 1000, 100, fingerprint1, 1, 3600, 1000)
		if overwritten1 {
			t.Error("Expected no overwrite for first entry")
		}

		// Update same key (same hash + same fingerprint) - should NOT be collision
		overwritten2 := rm.bitmap.Add(hash, 200, 2000, 200, fingerprint1, 2, 7200, 2000)
		if overwritten2 {
			t.Error("Expected no collision when updating same key (same hash + same fingerprint)")
		}

		// Different key collision (same hash + different fingerprint) - should BE collision
		overwritten3 := rm.bitmap.Add(hash, 300, 3000, 300, fingerprint2, 3, 1800, 3000)
		if !overwritten3 {
			t.Error("Expected collision when different key maps to same hash (same hash + different fingerprint)")
		}

		// Verify final state (should have the collision winner - fingerprint2)
		found, retMemTableId, retOffset, retLength, retFingerprint, retFreq, retTtl, retLastAccess := rm.bitmap.GetMeta(hash)
		if !found {
			t.Fatal("Expected entry to be found")
		}
		if retMemTableId != 300 || retOffset != 3000 || retLength != 300 || retFingerprint != fingerprint2 || retFreq != 3 || retTtl != 1800 || retLastAccess != 3000 {
			t.Errorf("Expected collision winner data (300, 3000, 300, 0x%x, 3, 1800, 3000), got (%d, %d, %d, 0x%x, %d, %d, %d)",
				fingerprint2, retMemTableId, retOffset, retLength, retFingerprint, retFreq, retTtl, retLastAccess)
		}

		// Clean up
		rm.bitmap.Remove(hash)
	})

	t.Run("GetMeta Non-Existent Entry", func(t *testing.T) {
		hash := uint32(0x999999)

		// Try to get metadata for non-existent entry
		found, memTableId, offset, length, fingerprint28, freq, ttl, lastAccess := rm.bitmap.GetMeta(hash)
		if found {
			t.Error("Expected no entry to be found for non-existent hash")
		}

		// Verify all returned values are zero
		if memTableId != 0 || offset != 0 || length != 0 || fingerprint28 != 0 || freq != 0 || ttl != 0 || lastAccess != 0 {
			t.Error("Expected all return values to be zero for non-existent entry")
		}
	})
}

func TestRoundMap_MetadataFunctions(t *testing.T) {
	t.Run("CreateMeta and ExtractMeta Roundtrip", func(t *testing.T) {
		memTableId := uint32(12345)
		offset := uint32(67890)
		length := uint16(512)
		fingerprint28 := uint32(0xABCDEF0)
		freq := uint32(0x12345)
		ttl := uint32(3600)
		lastAccess := uint32(1640995200)

		// Create metadata
		meta1, meta2, meta3 := CreateMeta(memTableId, offset, length, fingerprint28, freq, ttl, lastAccess)

		// Extract metadata
		retMemTableId, retOffset, retLength, retFingerprint28, retFreq, retTtl, retLastAccess := ExtractMeta(meta1, meta2, meta3)

		// Verify roundtrip
		if retMemTableId != memTableId {
			t.Errorf("MemTableId: expected %d, got %d", memTableId, retMemTableId)
		}
		if retOffset != offset {
			t.Errorf("Offset: expected %d, got %d", offset, retOffset)
		}
		if retLength != length {
			t.Errorf("Length: expected %d, got %d", length, retLength)
		}
		expectedFingerprint := fingerprint28 & _LO_28BIT_IN_32BIT
		if retFingerprint28 != expectedFingerprint {
			t.Errorf("Fingerprint28: expected 0x%x, got 0x%x", expectedFingerprint, retFingerprint28)
		}
		expectedFreq := freq & _LO_20BIT_IN_32BIT
		if retFreq != expectedFreq {
			t.Errorf("Freq: expected %d, got %d", expectedFreq, retFreq)
		}
		if retTtl != ttl {
			t.Errorf("TTL: expected %d, got %d", ttl, retTtl)
		}
		if retLastAccess != lastAccess {
			t.Errorf("LastAccess: expected %d, got %d", lastAccess, retLastAccess)
		}
	})

	t.Run("UpdateLastAccess", func(t *testing.T) {
		// Create initial metadata
		meta1, meta2, meta3 := CreateMeta(100, 200, 300, 0x1234567, 500, 600, 700)

		// Update last access
		newLastAccess := uint32(999999)
		updatedMeta3 := UpdateLastAccess(newLastAccess, meta3)

		// Extract and verify
		_, _, _, _, _, retTtl, retLastAccess := ExtractMeta(meta1, meta2, updatedMeta3)

		if retLastAccess != newLastAccess {
			t.Errorf("LastAccess: expected %d, got %d", newLastAccess, retLastAccess)
		}
		// TTL should remain unchanged
		if retTtl != 600 {
			t.Errorf("TTL should remain 600, got %d", retTtl)
		}
	})

	t.Run("Fingureprint28Matcher", func(t *testing.T) {
		fingerprint1 := uint32(0x1234567)
		fingerprint2 := uint32(0x1234567)
		fingerprint3 := uint32(0x7654321)

		// Create meta2 values with same fingerprint
		_, meta2_1, _ := CreateMeta(100, 200, 300, fingerprint1, 400, 500, 600)
		_, meta2_2, _ := CreateMeta(111, 222, 333, fingerprint2, 444, 555, 666)
		_, meta2_3, _ := CreateMeta(777, 888, 999, fingerprint3, 111, 222, 333)

		// Test matching fingerprints
		if !Fingureprint28Matcher(meta2_1, meta2_2) {
			t.Error("Expected fingerprints to match")
		}

		// Test non-matching fingerprints
		if Fingureprint28Matcher(meta2_1, meta2_3) {
			t.Error("Expected fingerprints to not match")
		}
	})

	t.Run("IncrementFreq Function", func(t *testing.T) {
		// Test frequency increment (this uses the logarithmic increment)
		originalFreq := uint32(100)
		incrementedFreq := IncrementFreq(originalFreq)

		// The incremented frequency should be different (and typically larger in fixed-point representation)
		if incrementedFreq == originalFreq {
			t.Error("Expected frequency to be incremented")
		}

		// Test multiple increments
		freq := uint32(50)
		for i := 0; i < 10; i++ {
			newFreq := IncrementFreq(freq)
			if newFreq < freq {
				t.Errorf("Iteration %d: Expected frequency to increase, got %d from %d", i, newFreq, freq)
			}
			freq = newFreq
		}
	})
}

func TestRoundMap_EdgeCases(t *testing.T) {
	rm := NewTestRoundMap()

	t.Run("24-bit Hash Masking", func(t *testing.T) {
		// Test that high bits in hash are ignored
		baseHash := uint32(0x123456)
		maskedHash1 := uint32(0x1123456)  // Same lower 24 bits
		maskedHash2 := uint32(0xFF123456) // Same lower 24 bits

		memTableId := uint32(100)
		offset := uint32(200)
		length := uint16(300)
		fingerprint28 := uint32(0x1111111)
		freq := uint32(400)
		ttl := uint32(500)
		lastAccess := uint32(600)

		// Add with base hash
		rm.bitmap.Add(baseHash, memTableId, offset, length, fingerprint28, freq, ttl, lastAccess)

		// Try to get with masked hashes (should find the same entry)
		found1, _, _, _, _, _, _, _ := rm.bitmap.GetMeta(maskedHash1)
		found2, _, _, _, _, _, _, _ := rm.bitmap.GetMeta(maskedHash2)

		if !found1 {
			t.Error("Expected to find entry with masked hash 1")
		}
		if !found2 {
			t.Error("Expected to find entry with masked hash 2")
		}

		// Clean up
		rm.bitmap.Remove(baseHash)
	})

	t.Run("Bit Masking in Metadata", func(t *testing.T) {
		hash := uint32(0x555555)

		// Use values that exceed the bit limits
		memTableId := uint32(0xFFFFFFFF)    // 32 bits - should be preserved
		offset := uint32(0xFFFFFFFF)        // 32 bits - should be preserved
		length := uint16(0xFFFF)            // 16 bits - should be preserved
		fingerprint28 := uint32(0x1FFFFFFF) // >28 bits - should be masked to 28 bits
		freq := uint32(0xFFFFFFFF)          // >20 bits - should be masked to 20 bits
		ttl := uint32(0xFFFFFFFF)           // 32 bits - should be preserved
		lastAccess := uint32(0xFFFFFFFF)    // 32 bits - should be preserved

		rm.bitmap.Add(hash, memTableId, offset, length, fingerprint28, freq, ttl, lastAccess)

		found, retMemTableId, retOffset, retLength, retFingerprint28, retFreq, retTtl, retLastAccess := rm.bitmap.GetMeta(hash)
		if !found {
			t.Fatal("Expected entry to be found")
		}

		// Check that values are properly masked
		if retMemTableId != memTableId {
			t.Errorf("MemTableId should be preserved: expected %d, got %d", memTableId, retMemTableId)
		}
		if retOffset != offset {
			t.Errorf("Offset should be preserved: expected %d, got %d", offset, retOffset)
		}
		if retLength != length {
			t.Errorf("Length should be preserved: expected %d, got %d", length, retLength)
		}

		expectedFingerprint := fingerprint28 & _LO_28BIT_IN_32BIT
		if retFingerprint28 != expectedFingerprint {
			t.Errorf("Fingerprint28 should be masked to 28 bits: expected 0x%x, got 0x%x", expectedFingerprint, retFingerprint28)
		}

		expectedFreq := freq & _LO_20BIT_IN_32BIT
		if retFreq != expectedFreq {
			t.Errorf("Freq should be masked to 20 bits: expected %d, got %d", expectedFreq, retFreq)
		}

		if retTtl != ttl {
			t.Errorf("TTL should be preserved: expected %d, got %d", ttl, retTtl)
		}
		if retLastAccess != lastAccess {
			t.Errorf("LastAccess should be preserved: expected %d, got %d", lastAccess, retLastAccess)
		}

		// Clean up
		rm.bitmap.Remove(hash)
	})
}
