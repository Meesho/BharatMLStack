package indices

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

// BenchmarkStats tracks collision and hit/miss statistics during benchmarks
type BenchmarkStats struct {
	TotalAdds     int64
	Collisions    int64
	TotalGets     int64
	Hits          int64
	Misses        int64
	CollisionRate float64
	HitRate       float64
}

func (s *BenchmarkStats) RecordAdd(collision bool) {
	s.TotalAdds++
	if collision {
		s.Collisions++
	}
}

func (s *BenchmarkStats) RecordGet(hit bool) {
	s.TotalGets++
	if hit {
		s.Hits++
	} else {
		s.Misses++
	}
}

func (s *BenchmarkStats) Calculate() {
	if s.TotalAdds > 0 {
		s.CollisionRate = float64(s.Collisions) / float64(s.TotalAdds) * 100
	}
	if s.TotalGets > 0 {
		s.HitRate = float64(s.Hits) / float64(s.TotalGets) * 100
	}
}

func (s *BenchmarkStats) Print() {
	s.Calculate()
	fmt.Printf("=== Benchmark Statistics ===\n")
	fmt.Printf("Add Operations: %d (Collisions: %d, Rate: %.2f%%)\n", s.TotalAdds, s.Collisions, s.CollisionRate)
	fmt.Printf("Get Operations: %d (Hits: %d, Misses: %d, Hit Rate: %.2f%%)\n", s.TotalGets, s.Hits, s.Misses, s.HitRate)
	fmt.Printf("============================\n")
}

func BenchmarkRoundMap_Add(b *testing.B) {
	rm := NewTestRoundMap()
	stats := &BenchmarkStats{}

	// Pre-generate test data
	hashes := make([]uint32, b.N)
	memTableIds := make([]uint32, b.N)
	offsets := make([]uint32, b.N)
	lengths := make([]uint16, b.N)
	fingerprints := make([]uint32, b.N)
	freqs := make([]uint32, b.N)
	ttls := make([]uint32, b.N)
	lastAccesses := make([]uint32, b.N)

	rand.Seed(42) // Fixed seed for reproducible results
	for i := 0; i < b.N; i++ {
		hashes[i] = rand.Uint32() & _24BIT_MASK
		memTableIds[i] = rand.Uint32()
		offsets[i] = rand.Uint32()
		lengths[i] = uint16(rand.Intn(65536))
		fingerprints[i] = rand.Uint32() & _LO_28BIT_IN_32BIT
		freqs[i] = rand.Uint32() & _LO_20BIT_IN_32BIT
		ttls[i] = rand.Uint32()
		lastAccesses[i] = rand.Uint32()
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		collision := rm.bitmap.Add(
			hashes[i],
			memTableIds[i],
			offsets[i],
			lengths[i],
			fingerprints[i],
			freqs[i],
			ttls[i],
			lastAccesses[i],
		)
		stats.RecordAdd(collision)
	}

	b.StopTimer()
	stats.Print()
}

func BenchmarkRoundMap_GetMeta(b *testing.B) {
	rm := NewTestRoundMap()
	stats := &BenchmarkStats{}

	// Pre-populate with some data
	numEntries := 10000
	populatedHashes := make([]uint32, numEntries)

	rand.Seed(42)
	for i := 0; i < numEntries; i++ {
		hash := rand.Uint32() & _24BIT_MASK
		memTableId := rand.Uint32()
		offset := rand.Uint32()
		length := uint16(rand.Intn(65536))
		fingerprint := rand.Uint32() & _LO_28BIT_IN_32BIT
		freq := rand.Uint32() & _LO_20BIT_IN_32BIT
		ttl := rand.Uint32()
		lastAccess := rand.Uint32()

		rm.bitmap.Add(hash, memTableId, offset, length, fingerprint, freq, ttl, lastAccess)
		populatedHashes[i] = hash
	}

	// Generate test hashes (mix of hits and misses)
	testHashes := make([]uint32, b.N)
	for i := 0; i < b.N; i++ {
		if rand.Float32() < 0.7 { // 70% chance of hit
			testHashes[i] = populatedHashes[rand.Intn(numEntries)]
		} else { // 30% chance of miss
			testHashes[i] = rand.Uint32() & _24BIT_MASK
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		found, _, _, _, _, _, _, _ := rm.bitmap.GetMeta(testHashes[i])
		stats.RecordGet(found)
	}

	b.StopTimer()
	stats.Print()
}

func BenchmarkRoundMap_Mixed_Workload(b *testing.B) {
	rm := NewTestRoundMap()
	stats := &BenchmarkStats{}

	// Pre-populate with initial data
	initialEntries := 5000
	populatedHashes := make([]uint32, initialEntries)

	rand.Seed(42)
	for i := 0; i < initialEntries; i++ {
		hash := rand.Uint32() & _24BIT_MASK
		memTableId := rand.Uint32()
		offset := rand.Uint32()
		length := uint16(rand.Intn(65536))
		fingerprint := rand.Uint32() & _LO_28BIT_IN_32BIT
		freq := rand.Uint32() & _LO_20BIT_IN_32BIT
		ttl := rand.Uint32()
		lastAccess := rand.Uint32()

		rm.bitmap.Add(hash, memTableId, offset, length, fingerprint, freq, ttl, lastAccess)
		populatedHashes[i] = hash
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		operation := rand.Float32()

		if operation < 0.3 { // 30% Add operations
			hash := rand.Uint32() & _24BIT_MASK
			memTableId := rand.Uint32()
			offset := rand.Uint32()
			length := uint16(rand.Intn(65536))
			fingerprint := rand.Uint32() & _LO_28BIT_IN_32BIT
			freq := rand.Uint32() & _LO_20BIT_IN_32BIT
			ttl := rand.Uint32()
			lastAccess := rand.Uint32()

			collision := rm.bitmap.Add(hash, memTableId, offset, length, fingerprint, freq, ttl, lastAccess)
			stats.RecordAdd(collision)

		} else { // 70% GetMeta operations
			var hash uint32
			if rand.Float32() < 0.6 && len(populatedHashes) > 0 { // 60% chance of querying existing hash
				hash = populatedHashes[rand.Intn(len(populatedHashes))]
			} else { // 40% chance of random hash
				hash = rand.Uint32() & _24BIT_MASK
			}

			found, _, _, _, _, _, _, _ := rm.bitmap.GetMeta(hash)
			stats.RecordGet(found)
		}
	}

	b.StopTimer()
	stats.Print()
}

func BenchmarkRoundMap_High_Collision_Scenario(b *testing.B) {
	rm := NewTestRoundMap()
	stats := &BenchmarkStats{}

	// Use a limited hash space to force more collisions
	hashSpace := uint32(1000) // Limited hash space

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		hash := rand.Uint32() % hashSpace // Force collisions by limiting hash space
		memTableId := rand.Uint32()
		offset := rand.Uint32()
		length := uint16(rand.Intn(65536))
		fingerprint := rand.Uint32() & _LO_28BIT_IN_32BIT // Different fingerprints to cause collisions
		freq := rand.Uint32() & _LO_20BIT_IN_32BIT
		ttl := rand.Uint32()
		lastAccess := rand.Uint32()

		collision := rm.bitmap.Add(hash, memTableId, offset, length, fingerprint, freq, ttl, lastAccess)
		stats.RecordAdd(collision)
	}

	b.StopTimer()
	stats.Print()
}

func BenchmarkRoundMap_Cache_Simulation(b *testing.B) {
	rm := NewTestRoundMap()
	stats := &BenchmarkStats{}

	// Simulate cache workload with working set
	workingSetSize := 1000
	workingSet := make([]uint32, workingSetSize)

	// Initialize working set
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < workingSetSize; i++ {
		hash := rand.Uint32() & _24BIT_MASK
		memTableId := rand.Uint32()
		offset := rand.Uint32()
		length := uint16(rand.Intn(65536))
		fingerprint := rand.Uint32() & _LO_28BIT_IN_32BIT
		freq := rand.Uint32() & _LO_20BIT_IN_32BIT
		ttl := rand.Uint32()
		lastAccess := rand.Uint32()

		collision := rm.bitmap.Add(hash, memTableId, offset, length, fingerprint, freq, ttl, lastAccess)
		stats.RecordAdd(collision)
		workingSet[i] = hash
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		operation := rand.Float32()

		if operation < 0.1 { // 10% cache updates/inserts
			hash := rand.Uint32() & _24BIT_MASK
			memTableId := rand.Uint32()
			offset := rand.Uint32()
			length := uint16(rand.Intn(65536))
			fingerprint := rand.Uint32() & _LO_28BIT_IN_32BIT
			freq := rand.Uint32() & _LO_20BIT_IN_32BIT
			ttl := rand.Uint32()
			lastAccess := rand.Uint32()

			collision := rm.bitmap.Add(hash, memTableId, offset, length, fingerprint, freq, ttl, lastAccess)
			stats.RecordAdd(collision)

		} else { // 90% cache lookups
			var hash uint32
			if rand.Float32() < 0.8 { // 80% hit rate on working set
				hash = workingSet[rand.Intn(workingSetSize)]
			} else { // 20% miss rate
				hash = rand.Uint32() & _24BIT_MASK
			}

			found, _, _, _, _, _, _, _ := rm.bitmap.GetMeta(hash)
			stats.RecordGet(found)
		}
	}

	b.StopTimer()
	stats.Print()
}

// Scalability benchmarks with different data sizes
func BenchmarkRoundMap_Scalability(b *testing.B) {
	sizes := []int{1000, 10000, 100000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			rm := NewTestRoundMap()
			stats := &BenchmarkStats{}

			// Pre-populate
			hashes := make([]uint32, size)
			rand.Seed(42)

			for i := 0; i < size; i++ {
				hash := rand.Uint32() & _24BIT_MASK
				memTableId := rand.Uint32()
				offset := rand.Uint32()
				length := uint16(rand.Intn(65536))
				fingerprint := rand.Uint32() & _LO_28BIT_IN_32BIT
				freq := rand.Uint32() & _LO_20BIT_IN_32BIT
				ttl := rand.Uint32()
				lastAccess := rand.Uint32()

				collision := rm.bitmap.Add(hash, memTableId, offset, length, fingerprint, freq, ttl, lastAccess)
				stats.RecordAdd(collision)
				hashes[i] = hash
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// Mix of operations
				if rand.Float32() < 0.5 {
					// GetMeta operation
					hash := hashes[rand.Intn(size)]
					found, _, _, _, _, _, _, _ := rm.bitmap.GetMeta(hash)
					stats.RecordGet(found)
				} else {
					// Add operation (update existing)
					hash := hashes[rand.Intn(size)]
					memTableId := rand.Uint32()
					offset := rand.Uint32()
					length := uint16(rand.Intn(65536))
					fingerprint := rand.Uint32() & _LO_28BIT_IN_32BIT
					freq := rand.Uint32() & _LO_20BIT_IN_32BIT
					ttl := rand.Uint32()
					lastAccess := rand.Uint32()

					collision := rm.bitmap.Add(hash, memTableId, offset, length, fingerprint, freq, ttl, lastAccess)
					stats.RecordAdd(collision)
				}
			}

			b.StopTimer()
			stats.Print()
		})
	}
}

// RoundMapBenchmarkStats extends BenchmarkStats with memory tracking
type RoundMapBenchmarkStats struct {
	BenchmarkStats
	MemoryBytes int64
}

func (s *RoundMapBenchmarkStats) PrintDetailed(benchName string, numRounds int, uniqueKeys int) {
	s.Calculate()
	fmt.Printf("\n=== %s ===\n", benchName)
	fmt.Printf("Configuration: numRounds=%d, uniqueKeys=%d\n", numRounds, uniqueKeys)
	fmt.Printf("Add Operations: %d (Collisions: %d, Rate: %.2f%%)\n", s.TotalAdds, s.Collisions, s.CollisionRate)
	fmt.Printf("Get Operations: %d (Hits: %d, Misses: %d, Hit Rate: %.2f%%)\n", s.TotalGets, s.Hits, s.Misses, s.HitRate)
	fmt.Printf("Memory Usage: %d bytes (%.2f MB)\n", s.MemoryBytes, float64(s.MemoryBytes)/(1024*1024))
	fmt.Printf("=====================================\n")
}

// Helper function to estimate memory usage of RoundMap
func estimateRoundMapMemory(rm *RoundMap, uniqueKeys int) int64 {
	// Each HBM24L4 bitmap has fixed memory overhead plus data entries
	// This is an approximation - for exact measurement you'd need runtime.MemStats
	bitmapOverhead := int64(1024) // Estimated overhead per bitmap
	entrySize := int64(24)        // Estimated bytes per entry (3 * uint64)

	totalMemory := int64(len(rm.bitmaps)) * bitmapOverhead
	totalMemory += int64(uniqueKeys) * entrySize

	return totalMemory
}

// Generate deterministic test keys
func generateTestKeys(count int) []string {
	keys := make([]string, count)
	rand.Seed(42) // Fixed seed for reproducible results

	for i := 0; i < count; i++ {
		keys[i] = fmt.Sprintf("key_%d_%d", i, rand.Uint32())
	}

	return keys
}

// Comprehensive RoundMap benchmarks for specific scenarios
func BenchmarkRoundMap_Scenario_1024_100K(b *testing.B) {
	runRoundMapScenario(b, 1024, 100000, "RoundMap_1024_100K")
}

func BenchmarkRoundMap_Scenario_1024_1M(b *testing.B) {
	runRoundMapScenario(b, 512, 1000000, "RoundMap_1024_1M")
	//runMapScenario(b, "Map_1024_1M")
}

func BenchmarkRoundMap_Scenario_1024_10M(b *testing.B) {
	runRoundMapScenario(b, 1024, 10000000, "RoundMap_1024_10M")
}

func BenchmarkRoundMap_Scenario_2048_100K(b *testing.B) {
	runRoundMapScenario(b, 2048, 100000, "RoundMap_2048_100K")
}

func BenchmarkRoundMap_Scenario_2048_1M(b *testing.B) {
	runRoundMapScenario(b, 2048, 1000000, "RoundMap_2048_1M")
}

func BenchmarkRoundMap_Scenario_2048_10M(b *testing.B) {
	runRoundMapScenario(b, 2048, 10000000, "RoundMap_2048_10M")
}

func BenchmarkRoundMap_Scenario_4096_50M(b *testing.B) {
	runRoundMapScenario(b, 4096, 50000000, "RoundMap_4096_50M")
}

func runRoundMapScenario(b *testing.B, numRounds int, uniqueKeys int, scenarioName string) {
	rm := NewRoundMap(numRounds)
	stats := &RoundMapBenchmarkStats{}

	// Prepare metadata for each key
	memTableIds := uint32(0)
	offsets := uint32(0)
	lengths := uint16(0)
	freqs := uint32(0)
	ttls := uint32(0)
	lastAccesses := uint32(0)

	// Phase 1: Add all unique keys
	b.Run(scenarioName+"_Add", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {

			collision := rm.Add(
				fmt.Sprintf("key_%d", i),
				memTableIds,
				offsets,
				lengths,
				freqs,
				ttls,
				lastAccesses,
			)
			stats.RecordAdd(collision)
		}

		stats.MemoryBytes = estimateRoundMapMemory(rm, uniqueKeys)
	})

	// Phase 2: Get operations (mix of hits and misses)
	b.Run(scenarioName+"_Get", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			found, _, _, _, _, _, _, _ := rm.Get(fmt.Sprintf("key_%d", i))
			stats.RecordGet(found)
		}
	})

	b.StopTimer()
	stats.MemoryBytes = estimateRoundMapMemory(rm, uniqueKeys)
	stats.PrintDetailed(scenarioName, numRounds, uniqueKeys)
}

func runMapScenario(b *testing.B, scenarioName string) {
	rm := make(map[uint64]struct {
		meta1 uint64
		meta2 uint64
		meta3 uint64
	})
	stats := &RoundMapBenchmarkStats{}

	// Prepare metadata for each key
	memTableIds := uint32(0)
	offsets := uint32(0)
	lengths := uint16(0)
	freqs := uint32(0)
	ttls := uint32(0)
	lastAccesses := uint32(0)

	// Phase 1: Add all unique keys
	b.Run(scenarioName+"_Add", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			meta1, meta2, meta3 := CreateMeta(memTableIds, offsets, lengths, memTableIds, freqs, ttls, lastAccesses)
			rm[uint64(i)] = struct {
				meta1 uint64
				meta2 uint64
				meta3 uint64
			}{
				meta1: meta1,
				meta2: meta2,
				meta3: meta3,
			}
		}
	})

	// Phase 2: Get operations (mix of hits and misses)
	b.Run(scenarioName+"_Get", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			st := rm[uint64(i)]
			_, _, _, _, _, _, _ = ExtractMeta(st.meta1, st.meta2, st.meta3)
			stats.RecordGet(true)
		}
	})

	b.StopTimer()
}

// Specific latency benchmarks for Add and Get operations
func BenchmarkRoundMap_Add_Latency(b *testing.B) {
	scenarios := []struct {
		numRounds  int
		uniqueKeys int
		name       string
	}{
		{1024, 100000, "1024_100K"},
		{1024, 1000000, "1024_1M"},
		{1024, 10000000, "1024_10M"},
		{2048, 100000, "2048_100K"},
		{2048, 1000000, "2048_1M"},
		{2048, 10000000, "2048_10M"},
		{4096, 50000000, "4096_50M"},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			rm := NewRoundMap(scenario.numRounds)
			keys := generateTestKeys(scenario.uniqueKeys)

			rand.Seed(42)
			memTableId := rand.Uint32()
			offset := rand.Uint32()
			length := uint16(rand.Intn(65536))
			freq := rand.Uint32() & _LO_20BIT_IN_32BIT
			ttl := rand.Uint32()
			lastAccess := rand.Uint32()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				keyIndex := i % scenario.uniqueKeys
				rm.Add(keys[keyIndex], memTableId, offset, length, freq, ttl, lastAccess)
			}
		})
	}
}

func BenchmarkRoundMap_Get_Latency(b *testing.B) {
	scenarios := []struct {
		numRounds  int
		uniqueKeys int
		name       string
	}{
		{1024, 100000, "1024_100K"},
		{1024, 1000000, "1024_1M"},
		{1024, 10000000, "1024_10M"},
		{2048, 100000, "2048_100K"},
		{2048, 1000000, "2048_1M"},
		{2048, 10000000, "2048_10M"},
		{4096, 50000000, "4096_50M"},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			rm := NewRoundMap(scenario.numRounds)
			keys := generateTestKeys(scenario.uniqueKeys)

			// Pre-populate the map
			rand.Seed(42)
			for i := 0; i < scenario.uniqueKeys; i++ {
				memTableId := rand.Uint32()
				offset := rand.Uint32()
				length := uint16(rand.Intn(65536))
				freq := rand.Uint32() & _LO_20BIT_IN_32BIT
				ttl := rand.Uint32()
				lastAccess := rand.Uint32()

				rm.Add(keys[i], memTableId, offset, length, freq, ttl, lastAccess)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				keyIndex := i % scenario.uniqueKeys
				rm.Get(keys[keyIndex])
			}
		})
	}
}
