package allocator

import (
	"fmt"
	"sort"

	"github.com/Meesho/BharatMLStack/ssd-cache/internal/pool"
)

// SlabAllocator manages pools of byte slices with different size classes
// This implementation is thread-safe and provides pool statistics
type SlabAllocator struct {
	pools            []*pool.CustomPool
	sizeClasses      []int
	sizeClassToIndex map[int]int // Map from size class to index for O(1) lookup
}

// SlabConfig represents configuration for the slab allocator
type SlabConfig struct {
	SizeClasses []int // Size classes in bytes (e.g., 1KB, 4KB, 16KB, 64KB, 256KB, 1MB)
}

// NewSlabAllocator creates a new slab allocator with given size classes
func NewSlabAllocator(config SlabConfig) *SlabAllocator {
	if len(config.SizeClasses) == 0 {
		// Default size classes: 1KB, 4KB, 16KB, 64KB, 256KB, 1MB, 4MB
		config.SizeClasses = []int{
			2,       // 2 bytes
			4,       // 4 bytes
			8,       // 8 bytes
			16,      // 16 bytes
			32,      // 32 bytes
			64,      // 64 bytes
			128,     // 128 bytes
			256,     // 256 bytes
			512,     // 512 bytes
			1024,    // 1KB
			2048,    // 2KB
			4096,    // 4KB
			8192,    // 8KB
			16384,   // 16KB
			32768,   // 32KB
			65536,   // 64KB
			131072,  // 128KB
			262144,  // 256KB
			524288,  // 512KB
			1048576, // 1MB
		}
	}

	// Sort size classes in ascending order
	sort.Ints(config.SizeClasses)

	allocator := &SlabAllocator{
		sizeClasses:      make([]int, len(config.SizeClasses)),
		pools:            make([]*pool.CustomPool, len(config.SizeClasses)),
		sizeClassToIndex: make(map[int]int),
	}

	copy(allocator.sizeClasses, config.SizeClasses)

	// Initialize custom pools for each size class
	for i, size := range allocator.sizeClasses {
		currentSize := size // Capture the current size for the closure
		allocator.pools[i] = pool.NewCustomPool(func() interface{} {
			return make([]byte, currentSize)
		})
		allocator.sizeClassToIndex[size] = i
	}

	return allocator
}

// GetBuffer retrieves a buffer of at least the requested size
// Returns the buffer and the actual size class used
func (sa *SlabAllocator) GetBuffer(size int) ([]byte, int) {
	// Find the appropriate size class
	sizeClassIndex := sa.findSizeClass(size)

	if sizeClassIndex == -1 {
		// Size is larger than the largest size class, allocate directly
		return make([]byte, size), size
	}

	// Get buffer from the appropriate pool
	buffer := sa.pools[sizeClassIndex].Get().([]byte)
	actualSize := sa.sizeClasses[sizeClassIndex]

	// Reset the buffer (clear existing data)
	for i := range buffer {
		buffer[i] = 0
	}

	return buffer[:size], actualSize
}

// PutBuffer returns a buffer to the appropriate pool
func (sa *SlabAllocator) PutBuffer(buffer []byte) {
	// Find the size class for the actual size
	sizeClassIndex := sa.findSizeClassExact(len(buffer))

	if sizeClassIndex == -1 {
		// Buffer was allocated directly (not from a pool), just let GC handle it
		return
	}

	// Return buffer to the appropriate pool
	sa.pools[sizeClassIndex].Put(buffer[0:])
}

// findSizeClass finds the smallest size class that can accommodate the requested size
func (sa *SlabAllocator) findSizeClass(size int) int {
	for i, classSize := range sa.sizeClasses {
		if size <= classSize {
			return i
		}
	}
	return -1 // Size is larger than all size classes
}

// findSizeClassExact finds the exact size class match
func (sa *SlabAllocator) findSizeClassExact(size int) int {
	index, exists := sa.sizeClassToIndex[size]
	if exists {
		return index
	}
	return -1 // No exact match found
}

// GetSizeClasses returns a copy of the configured size classes
func (sa *SlabAllocator) GetSizeClasses() []int {
	classes := make([]int, len(sa.sizeClasses))
	copy(classes, sa.sizeClasses)
	return classes
}

// Stats returns statistics about the slab allocator
type SlabStats struct {
	SizeClasses []int
	PoolStats   []int64 // Detailed statistics for each pool
}

// GetStats returns statistics about the current state of the slab allocator
func (sa *SlabAllocator) GetStats() SlabStats {
	stats := SlabStats{
		SizeClasses: make([]int, len(sa.sizeClasses)),
		PoolStats:   make([]int64, len(sa.pools)),
	}

	copy(stats.SizeClasses, sa.sizeClasses)

	// Get statistics from each custom pool
	for i, pool := range sa.pools {
		stats.PoolStats[i] = pool.Count()
	}

	return stats
}

// String returns a string representation of the slab allocator configuration
func (sa *SlabAllocator) String() string {
	return fmt.Sprintf("SlabAllocator{SizeClasses: %v}", sa.sizeClasses)
}

// PrintStats prints detailed statistics about the slab allocator
func (sa *SlabAllocator) PrintStats() {
	stats := sa.GetStats()
	fmt.Println("Slab Allocator Statistics:")
	fmt.Println("==========================")

	for i, sizeClass := range stats.SizeClasses {
		poolStats := stats.PoolStats[i]
		fmt.Printf("Size Class %d bytes:\n", sizeClass)
		fmt.Printf("  Objects in pool: %d\n", poolStats)
		fmt.Println()
	}
}
