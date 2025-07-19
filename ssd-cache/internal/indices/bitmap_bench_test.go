package indices

import (
	"math/rand"
	"testing"
	"time"
)

// Basic operation benchmarks
func BenchmarkHBM24L4_Put_Sequential(b *testing.B) {
	hbm := NewHBM24L4(true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hbm.Put(uint32(i) & _24BIT_MASK)
	}
}

func BenchmarkHBM24L4_Put_Random(b *testing.B) {
	hbm := NewHBM24L4(true)
	hashes := make([]uint32, b.N)
	rand.Seed(42) // Fixed seed for reproducible results
	for i := 0; i < b.N; i++ {
		hashes[i] = rand.Uint32() & _24BIT_MASK
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hbm.Put(hashes[i])
	}
}

func BenchmarkHBM24L4_Get_Sequential_Hit(b *testing.B) {
	hbm := NewHBM24L4(true)
	// Pre-populate with sequential data
	for i := 0; i < 100000; i++ {
		hbm.Put(uint32(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hbm.Get(uint32(i % 100000))
	}
}

func BenchmarkHBM24L4_Get_Random_Hit(b *testing.B) {
	hbm := NewHBM24L4(true)
	hashes := make([]uint32, 100000)
	rand.Seed(42)
	for i := 0; i < 100000; i++ {
		hashes[i] = rand.Uint32() & _24BIT_MASK
		hbm.Put(hashes[i])
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hbm.Get(hashes[i%100000])
	}
}

func BenchmarkHBM24L4_Get_Random_Miss(b *testing.B) {
	hbm := NewHBM24L4(true)
	// Pre-populate with some data
	rand.Seed(42)
	for i := 0; i < 50000; i++ {
		hbm.Put(rand.Uint32() & _24BIT_MASK)
	}

	// Generate different random numbers for misses
	rand.Seed(123)
	hashes := make([]uint32, b.N)
	for i := 0; i < b.N; i++ {
		hashes[i] = rand.Uint32() & _24BIT_MASK
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hbm.Get(hashes[i])
	}
}

func BenchmarkHBM24L4_Remove_Sequential(b *testing.B) {
	hbm := NewHBM24L4(true)
	// Pre-populate
	for i := 0; i < b.N; i++ {
		hbm.Put(uint32(i) & _24BIT_MASK)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hbm.Remove(uint32(i) & _24BIT_MASK)
	}
}

func BenchmarkHBM24L4_Remove_Random(b *testing.B) {
	hbm := NewHBM24L4(true)
	hashes := make([]uint32, b.N)
	rand.Seed(42)
	for i := 0; i < b.N; i++ {
		hashes[i] = rand.Uint32() & _24BIT_MASK
		hbm.Put(hashes[i])
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hbm.Remove(hashes[i])
	}
}

// Mixed workload benchmarks
func BenchmarkHBM24L4_Mixed_70Put_20Get_10Remove(b *testing.B) {
	hbm := NewHBM24L4(true)
	rand.Seed(42)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hash := rand.Uint32() & _24BIT_MASK
		op := rand.Float32()

		if op < 0.7 {
			hbm.Put(hash)
		} else if op < 0.9 {
			hbm.Get(hash)
		} else {
			hbm.Remove(hash)
		}
	}
}

func BenchmarkHBM24L4_Mixed_50Put_40Get_10Remove(b *testing.B) {
	hbm := NewHBM24L4(true)
	rand.Seed(42)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hash := rand.Uint32() & _24BIT_MASK
		op := rand.Float32()

		if op < 0.5 {
			hbm.Put(hash)
		} else if op < 0.9 {
			hbm.Get(hash)
		} else {
			hbm.Remove(hash)
		}
	}
}

// Allocation strategy benchmarks
func BenchmarkHBM24L4_LazyVsEager_SmallWorkload(b *testing.B) {
	b.Run("Lazy", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			hbm := NewHBM24L4(true)
			for j := 0; j < 100; j++ {
				hbm.Put(uint32(j))
			}
		}
	})

	b.Run("Eager", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			hbm := NewHBM24L4(false)
			for j := 0; j < 100; j++ {
				hbm.Put(uint32(j))
			}
		}
	})
}

func BenchmarkHBM24L4_LazyVsEager_LargeWorkload(b *testing.B) {
	b.Run("Lazy", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			hbm := NewHBM24L4(true)
			for j := 0; j < 10000; j++ {
				hbm.Put(uint32(j))
			}
		}
	})

	b.Run("Eager", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			hbm := NewHBM24L4(false)
			for j := 0; j < 10000; j++ {
				hbm.Put(uint32(j))
			}
		}
	})
}

// Data pattern benchmarks
func BenchmarkHBM24L4_SparseData(b *testing.B) {
	hbm := NewHBM24L4(true)
	// Sparse pattern: every 1000th value
	for i := 0; i < b.N; i++ {
		hbm.Put(uint32(i*1000) & _24BIT_MASK)
	}
}

func BenchmarkHBM24L4_DenseData(b *testing.B) {
	hbm := NewHBM24L4(true)
	// Dense pattern: consecutive values
	for i := 0; i < b.N; i++ {
		hbm.Put(uint32(i) & _24BIT_MASK)
	}
}

func BenchmarkHBM24L4_ClusteredData(b *testing.B) {
	hbm := NewHBM24L4(true)
	// Clustered pattern: groups of consecutive values with gaps
	cluster := 0
	for i := 0; i < b.N; i++ {
		if i%100 == 0 {
			cluster += 1000 // Jump to next cluster
		}
		hbm.Put(uint32(cluster+i%100) & _24BIT_MASK)
	}
}

// Cache behavior benchmarks
func BenchmarkHBM24L4_HotSpot_Access(b *testing.B) {
	hbm := NewHBM24L4(true)
	// Pre-populate
	for i := 0; i < 100000; i++ {
		hbm.Put(uint32(i))
	}

	// 80% of accesses go to 20% of data (hot spot)
	hotSpotSize := 20000

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var hash uint32
		if rand.Float32() < 0.8 {
			// Hot spot access
			hash = uint32(rand.Intn(hotSpotSize))
		} else {
			// Cold access
			hash = uint32(hotSpotSize + rand.Intn(80000))
		}
		hbm.Get(hash)
	}
}

// Stress test benchmarks
func BenchmarkHBM24L4_HighContention(b *testing.B) {
	hbm := NewHBM24L4(true)

	b.RunParallel(func(pb *testing.PB) {
		localRand := rand.New(rand.NewSource(time.Now().UnixNano()))
		for pb.Next() {
			hash := localRand.Uint32() & _24BIT_MASK
			op := localRand.Float32()

			if op < 0.6 {
				hbm.Put(hash)
			} else if op < 0.9 {
				hbm.Get(hash)
			} else {
				hbm.Remove(hash)
			}
		}
	})
}

// Memory allocation benchmarks
func BenchmarkHBM24L4_MemoryAllocation(b *testing.B) {
	b.Run("Lazy_GrowthPattern", func(b *testing.B) {
		hbm := NewHBM24L4(true)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Trigger allocations by accessing different parts of the tree
			hash := uint32(i*64*64) & _24BIT_MASK // Force different L1 nodes
			hbm.Put(hash)
		}
	})

	b.Run("Eager_Preallocation", func(b *testing.B) {
		hbm := NewHBM24L4(false)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Use the pre-allocated structure
			hbm.Put(uint32(i) & _24BIT_MASK)
		}
	})

	b.Run("Lazy_Map", func(b *testing.B) {
		m := make(map[uint32]bool)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			hash := uint32(i) & _24BIT_MASK
			m[hash] = true
		}
	})

	b.Run("Eager_Map", func(b *testing.B) {
		m := make(map[uint32]bool, 10000000)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			hash := uint32(i) & _24BIT_MASK
			m[hash] = true
		}
	})
}

// Scalability benchmarks
func BenchmarkHBM24L4_Scalability_Put(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}

	for _, size := range sizes {
		b.Run(func() string {
			if size >= 1000000 {
				return "1M"
			} else if size >= 100000 {
				return "100K"
			} else if size >= 10000 {
				return "10K"
			}
			return "1K"
		}(), func(b *testing.B) {
			hashes := make([]uint32, size)
			rand.Seed(42)
			for i := 0; i < size; i++ {
				hashes[i] = rand.Uint32() & _24BIT_MASK
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				hbm := NewHBM24L4(true)
				for j := 0; j < size; j++ {
					hbm.Put(hashes[j])
				}
			}
		})
	}
}

func BenchmarkHBM24L4_Scalability_Get(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}

	for _, size := range sizes {
		b.Run(func() string {
			if size >= 1000000 {
				return "1M"
			} else if size >= 100000 {
				return "100K"
			} else if size >= 10000 {
				return "10K"
			}
			return "1K"
		}(), func(b *testing.B) {
			hbm := NewHBM24L4(true)
			hashes := make([]uint32, size)
			rand.Seed(42)

			// Pre-populate
			for i := 0; i < size; i++ {
				hashes[i] = rand.Uint32() & _24BIT_MASK
				hbm.Put(hashes[i])
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				hbm.Get(hashes[i%size])
			}
		})
	}
}

// Worst-case scenario benchmarks
func BenchmarkHBM24L4_WorstCase_FullCollision(b *testing.B) {
	hbm := NewHBM24L4(true)

	// Fill up a single leaf word completely (worst case for that word)
	baseHash := uint32(0x123400) // Same i0, i1, i2
	for i := 0; i < 64; i++ {
		hbm.Put(baseHash | uint32(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Access the full word
		hbm.Get(baseHash | uint32(i%64))
	}
}

func BenchmarkHBM24L4_WorstCase_MaxDepth(b *testing.B) {
	hbm := NewHBM24L4(true)

	// Create scenario that uses maximum depth at every level
	// Fill different parts of each level
	for i0 := 0; i0 < 64; i0++ {
		for i1 := 0; i1 < 64; i1++ {
			hash := uint32(i0<<18 | i1<<12 | (i1%64)<<6 | (i0 % 64))
			hbm.Put(hash & _24BIT_MASK)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		i0 := i % 64
		i1 := (i / 64) % 64
		hash := uint32(i0<<18 | i1<<12 | (i1%64)<<6 | (i0 % 64))
		hbm.Get(hash & _24BIT_MASK)
	}
}

// Real-world usage simulation benchmarks
func BenchmarkHBM24L4_CacheSimulation_LRU(b *testing.B) {
	hbm := NewHBM24L4(true)
	cacheSize := 10000
	workingSet := make([]uint32, cacheSize)

	// Initialize working set
	rand.Seed(42)
	for i := 0; i < cacheSize; i++ {
		workingSet[i] = rand.Uint32() & _24BIT_MASK
		hbm.Put(workingSet[i])
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 90% hits, 10% misses (typical cache behavior)
		if rand.Float32() < 0.9 {
			// Cache hit
			hbm.Get(workingSet[rand.Intn(cacheSize)])
		} else {
			// Cache miss - evict old, add new
			newHash := rand.Uint32() & _24BIT_MASK
			oldIdx := rand.Intn(cacheSize)
			hbm.Remove(workingSet[oldIdx])
			hbm.Put(newHash)
			workingSet[oldIdx] = newHash
		}
	}
}

func BenchmarkHBM24L4_BloomFilter_Simulation(b *testing.B) {
	hbm := NewHBM24L4(true)

	// Simulate bloom filter usage - mostly queries with occasional inserts
	rand.Seed(42)

	// Pre-populate with some data
	for i := 0; i < 50000; i++ {
		hbm.Put(rand.Uint32() & _24BIT_MASK)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hash := rand.Uint32() & _24BIT_MASK
		op := rand.Float32()

		if op < 0.95 {
			// 95% queries (typical for bloom filter)
			hbm.Get(hash)
		} else {
			// 5% inserts
			hbm.Put(hash)
		}
	}
}

// Memory and allocation focused benchmarks
func BenchmarkHBM24L4_MemoryFootprint(b *testing.B) {
	b.Run("MinimalUsage", func(b *testing.B) {
		// Test memory usage with minimal data
		for i := 0; i < b.N; i++ {
			hbm := NewHBM24L4(true)
			hbm.Put(42)
			hbm.Get(42)
		}
	})

	b.Run("ModerateUsage", func(b *testing.B) {
		// Test memory usage with moderate data
		for i := 0; i < b.N; i++ {
			hbm := NewHBM24L4(true)
			for j := 0; j < 1000; j++ {
				hbm.Put(uint32(j))
			}
		}
	})

	b.Run("HeavyUsage", func(b *testing.B) {
		// Test memory usage with heavy data
		for i := 0; i < b.N; i++ {
			hbm := NewHBM24L4(true)
			for j := 0; j < 100000; j++ {
				hbm.Put(uint32(j))
			}
		}
	})
}
