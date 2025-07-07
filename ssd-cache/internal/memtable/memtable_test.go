package memtable

import (
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"
)

const (
	// File flags for testing
	O_DIRECT  = 0x4000
	O_WRONLY  = syscall.O_WRONLY
	O_APPEND  = syscall.O_APPEND
	O_CREAT   = syscall.O_CREAT
	O_DSYNC   = syscall.O_DSYNC
	FILE_MODE = 0644
)

// Test capacities in bytes
var testCapacities = []struct {
	name     string
	capacity int64
}{
	{"64KB", 64 * 1024},
	{"128KB", 128 * 1024},
	{"256KB", 256 * 1024},
	{"512KB", 512 * 1024},
	{"1MB", 1024 * 1024},
}

// Benchmark capacities including 2MB variant
var benchmarkCapacities = []struct {
	name     string
	capacity int64
	dsync    bool
}{
	{"64KB-NO-DSYNC", 64 * 1024, false},
	{"128KB-NO-DSYNC", 128 * 1024, false},
	{"256KB-NO-DSYNC", 256 * 1024, false},
	{"512KB-NO-DSYNC", 512 * 1024, false},
	{"1024KB-NO-DSYNC", 1024 * 1024, false},
	{"2048KB-NO-DSYNC", 2048 * 1024, false},
	{"4096KB-NO-DSYNC", 4096 * 1024, false},
	{"8192KB-NO-DSYNC", 8192 * 1024, false},
	{"16384KB-NO-DSYNC", 16384 * 1024, false},
	{"32768KB-NO-DSYNC", 32768 * 1024, false},
	{"65536KB-NO-DSYNC", 65536 * 1024, false},
	{"131072KB-NO-DSYNC", 131072 * 1024, false},
	{"262144KB-NO-DSYNC", 262144 * 1024, false},
	{"524288KB-NO-DSYNC", 524288 * 1024, false},
	{"1048576KB-NO-DSYNC", 1048576 * 1024, false},
	{"64KB-DSYNC", 64 * 1024, true},
	{"128KB-DSYNC", 128 * 1024, true},
	{"256KB-DSYNC", 256 * 1024, true},
	{"512KB-DSYNC", 512 * 1024, true},
	{"1024KB-DSYNC", 1024 * 1024, true},
	{"2048KB-DSYNC", 2048 * 1024, true},
	{"4096KB-DSYNC", 4096 * 1024, true},
	{"8192KB-DSYNC", 8192 * 1024, true},
	{"16384KB-DSYNC", 16384 * 1024, true},
	{"32768KB-DSYNC", 32768 * 1024, true},
	{"65536KB-DSYNC", 65536 * 1024, true},
	{"131072KB-DSYNC", 131072 * 1024, true},
	{"262144KB-DSYNC", 262144 * 1024, true},
	{"524288KB-DSYNC", 524288 * 1024, true},
	{"1048576KB-DSYNC", 1048576 * 1024, true},
}

// Helper function to create test file with specified flags
func createTestFile(t testing.TB, capacity int64) (*os.File, string) {
	t.Helper()

	// Create temporary directory
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_memtable.dat")

	// Open file with DIRECT_IO, WRITE_ONLY, APPEND_ONLY flags
	flags := O_DIRECT | O_WRONLY | O_APPEND | O_CREAT | O_DSYNC
	fd, err := syscall.Open(filename, flags, FILE_MODE)
	if err != nil {
		// If DIRECT_IO is not supported, fall back to regular flags
		t.Logf("DIRECT_IO not supported, falling back to regular flags: %v", err)
		flags = O_WRONLY | O_APPEND | O_CREAT | O_DSYNC
		fd, err = syscall.Open(filename, flags, FILE_MODE)
		if err != nil {
			t.Fatalf("Failed to open file: %v", err)
		}
	}

	file := os.NewFile(uintptr(fd), filename)
	if file == nil {
		t.Fatalf("Failed to create file from fd")
	}

	return file, filename
}

// Helper function to create test file with specified flags
func createTestFileWithoutDSYNC(t testing.TB, capacity int64) (*os.File, string) {
	t.Helper()

	// Create temporary directory
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_memtable.dat")

	// Open file with DIRECT_IO, WRITE_ONLY, APPEND_ONLY flags
	flags := O_DIRECT | O_WRONLY | O_APPEND | O_CREAT
	fd, err := syscall.Open(filename, flags, FILE_MODE)
	if err != nil {
		// If DIRECT_IO is not supported, fall back to regular flags
		t.Logf("DIRECT_IO not supported, falling back to regular flags: %v", err)
		flags = O_WRONLY | O_APPEND | O_CREAT
		fd, err = syscall.Open(filename, flags, FILE_MODE)
		if err != nil {
			t.Fatalf("Failed to open file: %v", err)
		}
	}

	file := os.NewFile(uintptr(fd), filename)
	if file == nil {
		t.Fatalf("Failed to create file from fd")
	}

	return file, filename
}
func TestNewMemtable(t *testing.T) {
	for _, tc := range testCapacities {
		t.Run(tc.name, func(t *testing.T) {
			file, _ := createTestFile(t, tc.capacity)
			defer file.Close()

			memtable := NewMemtable(file, 0, tc.capacity)

			if memtable == nil {
				t.Fatal("Expected memtable to be created, got nil")
			}

			if memtable.capacity != tc.capacity {
				t.Errorf("Expected capacity %d, got %d", tc.capacity, memtable.capacity)
			}

			if memtable.fileOffset != 0 {
				t.Errorf("Expected fileOffset 0, got %d", memtable.fileOffset)
			}

			if memtable.size != 0 {
				t.Errorf("Expected initial size 0, got %d", memtable.size)
			}

			if memtable.readyForFlush {
				t.Error("Expected readyForFlush to be false initially")
			}

			if memtable.page == nil {
				t.Error("Expected page to be allocated")
			}
		})
	}
}

func TestMemtable_Put(t *testing.T) {
	for _, tc := range testCapacities {
		t.Run(tc.name, func(t *testing.T) {
			file, _ := createTestFile(t, tc.capacity)
			defer file.Close()

			memtable := NewMemtable(file, 0, tc.capacity)

			// Test putting small data
			testData := []byte("hello world")
			offset, length := memtable.Put(testData)

			if offset != 0 {
				t.Errorf("Expected offset 0, got %d", offset)
			}

			if length != int64(len(testData)) {
				t.Errorf("Expected length %d, got %d", len(testData), length)
			}

			if memtable.size != int64(len(testData)) {
				t.Errorf("Expected size %d, got %d", len(testData), memtable.size)
			}
		})
	}
}

func TestMemtable_Get(t *testing.T) {
	for _, tc := range testCapacities {
		t.Run(tc.name, func(t *testing.T) {
			file, _ := createTestFile(t, tc.capacity)
			defer file.Close()

			memtable := NewMemtable(file, 0, tc.capacity)

			// Put some test data
			testData := []byte("hello world test data")
			offset, length := memtable.Put(testData)

			// Get the data back
			retrievedData := memtable.Get(offset, length)

			if len(retrievedData) != len(testData) {
				t.Errorf("Expected retrieved data length %d, got %d", len(testData), len(retrievedData))
			}

			for i, b := range testData {
				if retrievedData[i] != b {
					t.Errorf("Data mismatch at index %d: expected %v, got %v", i, b, retrievedData[i])
				}
			}
		})
	}
}

func TestMemtable_PutMultiple(t *testing.T) {
	for _, tc := range testCapacities {
		t.Run(tc.name, func(t *testing.T) {
			file, _ := createTestFile(t, tc.capacity)
			defer file.Close()

			memtable := NewMemtable(file, 0, tc.capacity)

			// Put multiple pieces of data
			testData1 := []byte("first data")
			testData2 := []byte("second data")
			testData3 := []byte("third data")

			offset1, length1 := memtable.Put(testData1)
			offset2, length2 := memtable.Put(testData2)
			offset3, length3 := memtable.Put(testData3)

			// Verify offsets are sequential
			if offset1 != 0 {
				t.Errorf("Expected first offset 0, got %d", offset1)
			}

			if offset2 != int64(len(testData1)) {
				t.Errorf("Expected second offset %d, got %d", len(testData1), offset2)
			}

			expectedOffset3 := int64(len(testData1) + len(testData2))
			if offset3 != expectedOffset3 {
				t.Errorf("Expected third offset %d, got %d", expectedOffset3, offset3)
			}

			// Verify we can retrieve all data correctly
			retrieved1 := memtable.Get(offset1, length1)
			retrieved2 := memtable.Get(offset2, length2)
			retrieved3 := memtable.Get(offset3, length3)

			if string(retrieved1) != string(testData1) {
				t.Errorf("First data mismatch: expected %s, got %s", testData1, retrieved1)
			}

			if string(retrieved2) != string(testData2) {
				t.Errorf("Second data mismatch: expected %s, got %s", testData2, retrieved2)
			}

			if string(retrieved3) != string(testData3) {
				t.Errorf("Third data mismatch: expected %s, got %s", testData3, retrieved3)
			}
		})
	}
}

func TestMemtable_PutExactCapacity(t *testing.T) {
	for _, tc := range testCapacities {
		t.Run(tc.name, func(t *testing.T) {
			file, _ := createTestFile(t, tc.capacity)
			defer file.Close()

			memtable := NewMemtable(file, 0, tc.capacity)

			// Create data that exactly fills capacity (should NOT trigger flush)
			exactData := make([]byte, tc.capacity)
			for i := range exactData {
				exactData[i] = byte(i % 256)
			}

			offset, length := memtable.Put(exactData)

			// Should NOT trigger flush since size == capacity (not > capacity)
			if memtable.readyForFlush {
				t.Error("Expected readyForFlush to be false - exact capacity should not trigger flush")
			}

			// Verify the data was written
			if offset != 0 {
				t.Errorf("Expected offset 0, got %d", offset)
			}

			if length != int64(len(exactData)) {
				t.Errorf("Expected length %d, got %d", len(exactData), length)
			}

			if memtable.size != tc.capacity {
				t.Errorf("Expected size %d, got %d", tc.capacity, memtable.size)
			}
		})
	}
}

func TestMemtable_PutCapacityOverflow(t *testing.T) {
	for _, tc := range testCapacities {
		t.Run(tc.name, func(t *testing.T) {
			file, _ := createTestFile(t, tc.capacity)
			defer file.Close()

			memtable := NewMemtable(file, 0, tc.capacity)

			// Create data that will exceed capacity by 1 byte
			largeData := make([]byte, tc.capacity+1)
			for i := range largeData {
				largeData[i] = byte(i % 256)
			}

			// This should set readyForFlush = true and return (-1, -1)
			offset, length := memtable.Put(largeData)

			// Should have set readyForFlush = true when capacity was exceeded
			if !memtable.readyForFlush {
				t.Error("Expected readyForFlush to be true after capacity overflow")
			}

			// Should return (-1, -1) when capacity is exceeded
			if offset != -1 {
				t.Errorf("Expected offset -1, got %d", offset)
			}

			if length != -1 {
				t.Errorf("Expected length -1, got %d", length)
			}

			// Size should remain unchanged since Put failed
			if memtable.size != 0 {
				t.Errorf("Expected size to remain 0 after failed Put, got %d", memtable.size)
			}
		})
	}
}

func TestMemtable_PutProgressiveCapacity(t *testing.T) {
	tc := testCapacities[0] // Use 64KB for this test

	file, _ := createTestFile(t, tc.capacity)
	defer file.Close()

	memtable := NewMemtable(file, 0, tc.capacity)

	// Fill to near capacity (capacity - 10 bytes)
	nearCapacityData := make([]byte, tc.capacity-10)
	for i := range nearCapacityData {
		nearCapacityData[i] = byte(i % 256)
	}

	offset1, length1 := memtable.Put(nearCapacityData)

	// Should not trigger flush
	if memtable.readyForFlush {
		t.Error("Expected readyForFlush to be false - near capacity should not trigger flush")
	}

	if memtable.size != int64(len(nearCapacityData)) {
		t.Errorf("Expected size %d, got %d", len(nearCapacityData), memtable.size)
	}

	// Add exactly the remaining capacity (10 bytes) - should still not flush
	exactRemainingData := make([]byte, 10)
	for i := range exactRemainingData {
		exactRemainingData[i] = byte(i % 256)
	}

	offset2, length2 := memtable.Put(exactRemainingData)

	// Should still not trigger flush since total size == capacity
	if memtable.readyForFlush {
		t.Error("Expected readyForFlush to be false - exact capacity should not trigger flush")
	}

	if memtable.size != tc.capacity {
		t.Errorf("Expected size %d (exact capacity), got %d", tc.capacity, memtable.size)
	}

	// Now add one more byte - this should set readyForFlush = true and return (-1, -1)
	oneMoreByte := []byte{42}
	offset3, length3 := memtable.Put(oneMoreByte)

	// Should have set readyForFlush = true when capacity was exceeded
	if !memtable.readyForFlush {
		t.Error("Expected readyForFlush to be true after capacity overflow")
	}

	// Should return (-1, -1) when capacity is exceeded
	if offset3 != -1 || length3 != -1 {
		t.Errorf("Expected offset=-1 and length=-1 for overflow data, got offset=%d, length=%d", offset3, length3)
	}

	// Size should remain at capacity since the overflow Put failed
	if memtable.size != tc.capacity {
		t.Errorf("Expected size to remain %d after failed Put, got %d", tc.capacity, memtable.size)
	}

	// Verify the original data can still be retrieved (data written before overflow)
	retrieved1 := memtable.Get(offset1, length1)
	retrieved2 := memtable.Get(offset2, length2)

	if len(retrieved1) != len(nearCapacityData) {
		t.Errorf("Retrieved data 1 length mismatch: expected %d, got %d", len(nearCapacityData), len(retrieved1))
	}

	if len(retrieved2) != len(exactRemainingData) {
		t.Errorf("Retrieved data 2 length mismatch: expected %d, got %d", len(exactRemainingData), len(retrieved2))
	}
}

func TestMemtable_FlushNotReady(t *testing.T) {
	for _, tc := range testCapacities {
		t.Run(tc.name, func(t *testing.T) {
			file, _ := createTestFile(t, tc.capacity)
			defer file.Close()

			memtable := NewMemtable(file, 0, tc.capacity)

			// Try to flush when not ready
			err := memtable.Flush()

			if err == nil {
				t.Error("Expected error when flushing memtable not ready for flush")
			}

			expectedErr := "memtable not ready for flush"
			if err.Error() != expectedErr {
				t.Errorf("Expected error message '%s', got '%s'", expectedErr, err.Error())
			}
		})
	}
}

func TestMemtable_FlushReady(t *testing.T) {
	for _, tc := range testCapacities {
		t.Run(tc.name, func(t *testing.T) {
			file, _ := createTestFile(t, tc.capacity)
			defer file.Close()

			memtable := NewMemtable(file, 0, tc.capacity)

			// Fill memtable exactly to capacity (should NOT trigger flush)
			exactCapacityData := make([]byte, tc.capacity)
			for i := range exactCapacityData {
				exactCapacityData[i] = byte(i % 256)
			}

			memtable.Put(exactCapacityData)

			// Should not have triggered flush since size == capacity (not > capacity)
			if memtable.readyForFlush {
				t.Error("Expected readyForFlush to be false - exact capacity should not trigger flush")
			}

			// Now try adding one more byte to trigger readyForFlush
			oneMoreByte := []byte{42}
			offset, length := memtable.Put(oneMoreByte)

			// Should have set readyForFlush = true when capacity was exceeded
			if !memtable.readyForFlush {
				t.Error("Expected readyForFlush to be true after capacity overflow")
			}

			// Should return (-1, -1) when capacity is exceeded
			if offset != -1 || length != -1 {
				t.Errorf("Expected offset=-1 and length=-1 for overflow data, got offset=%d, length=%d", offset, length)
			}

			// Test manual flush when ready (should succeed or fail gracefully)
			err := memtable.Flush()
			if err != nil {
				t.Logf("Flush failed due to file system constraints: %v", err)
				// If flush failed, readyForFlush should still be false (reset by flush attempt)
				if memtable.readyForFlush {
					t.Error("Expected readyForFlush to be false after failed flush attempt")
				}
			} else {
				t.Logf("Flush succeeded!")
				// If flush succeeded, readyForFlush should be false and size should be reset
				if memtable.readyForFlush {
					t.Error("Expected readyForFlush to be false after successful flush")
				}
				if memtable.size != 0 {
					t.Errorf("Expected size to be reset to 0 after successful flush, got %d", memtable.size)
				}
			}
		})
	}
}

func TestMemtable_FlushManualSetup(t *testing.T) {
	tc := testCapacities[0] // Use 64KB for this test

	file, _ := createTestFile(t, tc.capacity)
	defer file.Close()

	memtable := NewMemtable(file, 0, tc.capacity)

	// Put some data (not exceeding capacity)
	testData := []byte("test data for manual flush")
	_, _ = memtable.Put(testData)
	originalSize := memtable.size

	// Manually set readyForFlush to test the flush mechanism
	// This simulates a scenario where flush is needed for reasons other than capacity overflow
	memtable.readyForFlush = true

	if !memtable.readyForFlush {
		t.Error("Expected memtable to be ready for flush after manual setup")
	}

	// Test flush - should now work correctly after fixing the bug
	err := memtable.Flush()

	// After successful flush, readyForFlush should be false
	if memtable.readyForFlush {
		t.Error("Expected readyForFlush to be false after flush attempt")
	}

	// Check if flush succeeded or failed due to file system constraints
	if err != nil {
		t.Logf("Flush failed due to file system constraints: %v", err)
		// If flush failed, internal state should still be consistent
		if memtable.size != originalSize {
			t.Errorf("Expected size to remain %d after failed flush, got %d", originalSize, memtable.size)
		}
	} else {
		t.Logf("Flush succeeded!")
		// If flush succeeded, size should be reset to 0
		if memtable.size != 0 {
			t.Errorf("Expected size to be reset to 0 after successful flush, got %d", memtable.size)
		}
		// File offset should be updated
		if memtable.fileOffset != tc.capacity {
			t.Errorf("Expected fileOffset to be updated to %d after flush, got %d", tc.capacity, memtable.fileOffset)
		}
	}
}

func TestMemtable_LargeData(t *testing.T) {
	// Test with 1MB capacity and large data
	tc := testCapacities[4] // 1MB

	file, _ := createTestFile(t, tc.capacity)
	defer file.Close()

	memtable := NewMemtable(file, 0, tc.capacity)

	// Create large test data (half capacity)
	largeData := make([]byte, tc.capacity/2)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	offset, length := memtable.Put(largeData)

	if offset != 0 {
		t.Errorf("Expected offset 0, got %d", offset)
	}

	if length != int64(len(largeData)) {
		t.Errorf("Expected length %d, got %d", len(largeData), length)
	}

	// Verify we can retrieve the large data
	retrieved := memtable.Get(offset, length)

	if len(retrieved) != len(largeData) {
		t.Errorf("Retrieved data length mismatch: expected %d, got %d", len(largeData), len(retrieved))
	}

	// Check first and last few bytes
	for i := 0; i < 100 && i < len(largeData); i++ {
		if retrieved[i] != largeData[i] {
			t.Errorf("Data mismatch at index %d: expected %v, got %v", i, largeData[i], retrieved[i])
		}
	}
}

func TestMemtable_EdgeCases(t *testing.T) {
	tc := testCapacities[0] // 64KB

	file, _ := createTestFile(t, tc.capacity)
	defer file.Close()

	memtable := NewMemtable(file, 0, tc.capacity)

	// Test empty data
	emptyData := []byte{}
	offset, length := memtable.Put(emptyData)

	if offset != 0 {
		t.Errorf("Expected offset 0 for empty data, got %d", offset)
	}

	if length != 0 {
		t.Errorf("Expected length 0 for empty data, got %d", length)
	}

	// Test single byte
	singleByte := []byte{42}
	offset, length = memtable.Put(singleByte)

	if offset != 0 {
		t.Errorf("Expected offset 0 for single byte, got %d", offset)
	}

	if length != 1 {
		t.Errorf("Expected length 1 for single byte, got %d", length)
	}

	retrieved := memtable.Get(offset, length)
	if len(retrieved) != 1 || retrieved[0] != 42 {
		t.Errorf("Single byte retrieval failed: expected [42], got %v", retrieved)
	}
}

// High-performance benchmark: Write 16KB records until 100MB+16KB total
// This simulates realistic SSD cache workload patterns
func BenchmarkMemtable_Write16KBWorkload(b *testing.B) {
	disableFlushes := false
	const recordSize = 16 * 1024 // 16KB per record

	// Create 16KB test data
	testData := make([]byte, recordSize)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	for _, tc := range benchmarkCapacities {
		b.Run(tc.name, func(b *testing.B) {

			file, _ := createTestFile(b, tc.capacity)
			defer file.Close()
			memtable := NewMemtable(file, 0, tc.capacity)
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()

			b.ResetTimer()
			b.ReportAllocs()

			start := time.Now()
			totalData := int64(0)
			for i := 0; i < b.N; i++ {
				if totalData+int64(len(testData)) > tc.capacity && disableFlushes {
					break
				}
				_, length := memtable.Put(testData)
				totalData += length
			}

			elapsed := time.Since(start)

			// Calculate metrics
			recordsPerSecond := float64(b.N) / elapsed.Seconds()
			b.ReportMetric(float64(memtable.flushCount), "flushes")
			b.ReportMetric(float64(memtable.flushCount)/elapsed.Seconds(), "flushes/sec")
			b.ReportMetric(recordsPerSecond, "records/sec")
			b.ReportMetric(recordsPerSecond*16*1024/(1024*1024), "MB/sec")
			fileInfo, _ := file.Stat()
			b.ReportMetric(float64(fileInfo.Size()), "file_size")
		})
	}
}

// Benchmark single record latency
func BenchmarkMemtable_RecordLatency(b *testing.B) {
	const recordSize = 16 * 1024 // 16KB per record

	testData := make([]byte, recordSize)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	for _, tc := range benchmarkCapacities {
		b.Run(tc.name, func(b *testing.B) {
			file, _ := createTestFile(b, tc.capacity)
			defer file.Close()

			memtable := NewMemtable(file, 0, tc.capacity)
			recordCount := int64(0)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Reset memtable when approaching capacity to avoid flush overhead in latency test
				if memtable.size+recordSize > tc.capacity {
					memtable = NewMemtable(file, 0, tc.capacity)
					recordCount = 0
				}

				memtable.Put(testData)
				recordCount++
			}
		})
	}
}

// Benchmark Get latency for 16KB records
func BenchmarkMemtable_GetLatency(b *testing.B) {
	const recordSize = 16 * 1024

	for _, tc := range benchmarkCapacities {
		b.Run(tc.name, func(b *testing.B) {
			file, _ := createTestFile(b, tc.capacity)
			defer file.Close()

			memtable := NewMemtable(file, 0, tc.capacity)

			// Pre-populate with records
			testData := make([]byte, recordSize)
			for i := range testData {
				testData[i] = byte(i % 256)
			}

			var offsets []int64
			var lengths []int64

			// Fill up to 80% of capacity
			maxRecords := (tc.capacity * 80 / 100) / recordSize
			for i := int64(0); i < maxRecords; i++ {
				offset, length := memtable.Put(testData)
				offsets = append(offsets, offset)
				lengths = append(lengths, length)
			}

			if len(offsets) == 0 {
				b.Skip("Capacity too small for test data")
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				idx := i % len(offsets)
				_ = memtable.Get(offsets[idx], lengths[idx])
			}
		})
	}
}

// Benchmark flush behavior under different capacities
func BenchmarkMemtable_FlushBehavior(b *testing.B) {
	const recordSize = 16 * 1024

	testData := make([]byte, recordSize)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	for _, tc := range benchmarkCapacities {
		b.Run(tc.name, func(b *testing.B) {
			file, _ := createTestFile(b, tc.capacity)
			defer file.Close()

			recordsPerFlush := tc.capacity / recordSize
			b.Logf("Records per flush for %s: %d", tc.name, recordsPerFlush)

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				memtable := NewMemtable(file, 0, tc.capacity)

				// Fill to exactly trigger one flush
				for j := int64(0); j <= recordsPerFlush; j++ {
					memtable.Put(testData)
				}
			}
		})
	}
}
