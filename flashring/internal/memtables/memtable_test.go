package memtables

import (
	"path/filepath"
	"testing"

	"github.com/Meesho/BharatMLStack/flashring/internal/fs"
	"github.com/Meesho/BharatMLStack/flashring/internal/iouring"
)

// Helper function to create a mock file for testing
func createTestFile(t *testing.T) *fs.WrapAppendFile {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_memtable.dat")

	config := fs.FileConfig{
		Filename:          filename,
		MaxFileSize:       1024 * 1024, // 1MB
		FilePunchHoleSize: 64 * 1024,   // 64KB
		BlockSize:         fs.BLOCK_SIZE,
	}

	file, err := fs.NewWrapAppendFile(config)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	writeRing, err := iouring.NewIoUringWriter(32, 0)
	if err != nil {
		t.Fatalf("Failed to create io_uring write ring: %v", err)
	}
	file.WriteRing = writeRing
	return file
}

// Helper function to create a test page
func createTestPage(size int) *fs.AlignedPage {
	page, err := fs.NewAlignedPage(size)
	if err != nil {
		panic(err)
	}
	return page
}

// Helper function to cleanup resources
func cleanup(file *fs.WrapAppendFile, page *fs.AlignedPage) {
	if file != nil {
		if file.WriteRing != nil {
			file.WriteRing.Close()
		}
		file.Close()
	}
	if page != nil {
		fs.Unmap(page)
	}
}

func TestNewMemtable_Success(t *testing.T) {
	capacity := fs.BLOCK_SIZE * 2 // 8192 bytes
	file := createTestFile(t)
	page := createTestPage(capacity)
	defer cleanup(file, page)

	config := MemtableConfig{
		capacity: capacity,
		id:       1,
		page:     page,
		file:     file,
	}

	memtable, err := NewMemtable(config)
	if err != nil {
		t.Fatalf("NewMemtable failed: %v", err)
	}

	if memtable.Id != 1 {
		t.Errorf("Expected Id 1, got %d", memtable.Id)
	}
	if memtable.capacity != capacity {
		t.Errorf("Expected capacity %d, got %d", capacity, memtable.capacity)
	}
	if memtable.currentOffset != 0 {
		t.Errorf("Expected currentOffset 0, got %d", memtable.currentOffset)
	}
	if memtable.readyForFlush != false {
		t.Errorf("Expected readyForFlush false, got %v", memtable.readyForFlush)
	}
}

func TestNewMemtable_CapacityNotAligned(t *testing.T) {
	capacity := fs.BLOCK_SIZE + 100 // Not aligned to block size
	file := createTestFile(t)
	page := createTestPage(capacity)
	defer cleanup(file, page)

	config := MemtableConfig{
		capacity: capacity,
		id:       1,
		page:     page,
		file:     file,
	}

	_, err := NewMemtable(config)
	if err != ErrCapacityNotAligned {
		t.Errorf("Expected ErrCapacityNotAligned, got %v", err)
	}
}

func TestNewMemtable_PageNotProvided(t *testing.T) {
	capacity := fs.BLOCK_SIZE
	file := createTestFile(t)
	defer cleanup(file, nil)

	config := MemtableConfig{
		capacity: capacity,
		id:       1,
		page:     nil,
		file:     file,
	}

	_, err := NewMemtable(config)
	if err != ErrPageNotProvided {
		t.Errorf("Expected ErrPageNotProvided, got %v", err)
	}
}

func TestNewMemtable_FileNotProvided(t *testing.T) {
	capacity := fs.BLOCK_SIZE
	page := createTestPage(capacity)
	defer cleanup(nil, page)

	config := MemtableConfig{
		capacity: capacity,
		id:       1,
		page:     page,
		file:     nil,
	}

	_, err := NewMemtable(config)
	if err != ErrFileNotProvided {
		t.Errorf("Expected ErrFileNotProvided, got %v", err)
	}
}

func TestNewMemtable_PageBufferCapacityMismatch(t *testing.T) {
	capacity := fs.BLOCK_SIZE
	file := createTestFile(t)
	page := createTestPage(capacity * 2) // Wrong size
	defer cleanup(file, page)

	config := MemtableConfig{
		capacity: capacity,
		id:       1,
		page:     page,
		file:     file,
	}

	_, err := NewMemtable(config)
	if err != ErrPageBufferCapacityMismatch {
		t.Errorf("Expected ErrPageBufferCapacityMismatch, got %v", err)
	}
}

func TestNewMemtable_PageBufferNil(t *testing.T) {
	capacity := fs.BLOCK_SIZE
	file := createTestFile(t)
	defer cleanup(file, nil)

	// Create page with nil buffer
	page := &fs.AlignedPage{Buf: nil}

	config := MemtableConfig{
		capacity: capacity,
		id:       1,
		page:     page,
		file:     file,
	}

	_, err := NewMemtable(config)
	if err != ErrPageBufferCapacityMismatch {
		t.Errorf("Expected ErrPageBufferCapacityMismatch, got %v", err)
	}
}

func TestMemtable_Get_Success(t *testing.T) {
	capacity := fs.BLOCK_SIZE
	file := createTestFile(t)
	page := createTestPage(capacity)
	defer cleanup(file, page)

	config := MemtableConfig{
		capacity: capacity,
		id:       1,
		page:     page,
		file:     file,
	}

	memtable, err := NewMemtable(config)
	if err != nil {
		t.Fatalf("NewMemtable failed: %v", err)
	}

	// Write some test data to the page buffer
	testData := []byte("Hello, World!")
	copy(page.Buf[:len(testData)], testData)

	// Get the data
	result, err := memtable.Get(0, uint16(len(testData)))
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if string(result) != string(testData) {
		t.Errorf("Expected %s, got %s", testData, result)
	}
}

func TestMemtable_Get_OffsetOutOfBounds(t *testing.T) {
	capacity := fs.BLOCK_SIZE
	file := createTestFile(t)
	page := createTestPage(capacity)
	defer cleanup(file, page)

	config := MemtableConfig{
		capacity: capacity,
		id:       1,
		page:     page,
		file:     file,
	}

	memtable, err := NewMemtable(config)
	if err != nil {
		t.Fatalf("NewMemtable failed: %v", err)
	}

	// Try to get data beyond capacity
	_, err = memtable.Get(capacity-10, 20)
	if err != ErrOffsetOutOfBounds {
		t.Errorf("Expected ErrOffsetOutOfBounds, got %v", err)
	}
}

func TestMemtable_Put_Success(t *testing.T) {
	capacity := fs.BLOCK_SIZE
	file := createTestFile(t)
	page := createTestPage(capacity)
	defer cleanup(file, page)

	config := MemtableConfig{
		capacity: capacity,
		id:       1,
		page:     page,
		file:     file,
	}

	memtable, err := NewMemtable(config)
	if err != nil {
		t.Fatalf("NewMemtable failed: %v", err)
	}

	testData := []byte("Hello, World!")
	offset, length, readyForFlush := memtable.Put(testData)

	if offset != 0 {
		t.Errorf("Expected offset 0, got %d", offset)
	}
	if length != uint16(len(testData)) {
		t.Errorf("Expected length %d, got %d", len(testData), length)
	}
	if readyForFlush {
		t.Errorf("Expected readyForFlush false, got %v", readyForFlush)
	}
	if memtable.currentOffset != len(testData) {
		t.Errorf("Expected currentOffset %d, got %d", len(testData), memtable.currentOffset)
	}

	// Verify data was written to buffer
	result, err := memtable.Get(0, uint16(len(testData)))
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if string(result) != string(testData) {
		t.Errorf("Expected %s, got %s", testData, result)
	}
}

func TestMemtable_Put_ExceedsCapacity(t *testing.T) {
	capacity := fs.BLOCK_SIZE
	file := createTestFile(t)
	page := createTestPage(capacity)
	defer cleanup(file, page)

	config := MemtableConfig{
		capacity: capacity,
		id:       1,
		page:     page,
		file:     file,
	}

	memtable, err := NewMemtable(config)
	if err != nil {
		t.Fatalf("NewMemtable failed: %v", err)
	}

	// Fill the memtable to near capacity
	testData := make([]byte, capacity-100)
	_, _, _ = memtable.Put(testData)

	// Try to put data that exceeds capacity
	largeData := make([]byte, 200)
	offset, length, readyForFlush := memtable.Put(largeData)

	if offset != -1 {
		t.Errorf("Expected offset -1, got %d", offset)
	}
	if length != 0 {
		t.Errorf("Expected length 0, got %d", length)
	}
	if !readyForFlush {
		t.Errorf("Expected readyForFlush true, got %v", readyForFlush)
	}
	if !memtable.readyForFlush {
		t.Errorf("Expected memtable.readyForFlush true, got %v", memtable.readyForFlush)
	}
}

func TestMemtable_Put_MultiplePuts(t *testing.T) {
	capacity := fs.BLOCK_SIZE
	file := createTestFile(t)
	page := createTestPage(capacity)
	defer cleanup(file, page)

	config := MemtableConfig{
		capacity: capacity,
		id:       1,
		page:     page,
		file:     file,
	}

	memtable, err := NewMemtable(config)
	if err != nil {
		t.Fatalf("NewMemtable failed: %v", err)
	}

	// Put multiple pieces of data
	data1 := []byte("First")
	data2 := []byte("Second")
	data3 := []byte("Third")

	offset1, length1, _ := memtable.Put(data1)
	offset2, length2, _ := memtable.Put(data2)
	offset3, length3, _ := memtable.Put(data3)

	if offset1 != 0 {
		t.Errorf("Expected offset1 0, got %d", offset1)
	}
	if offset2 != len(data1) {
		t.Errorf("Expected offset2 %d, got %d", len(data1), offset2)
	}
	if offset3 != len(data1)+len(data2) {
		t.Errorf("Expected offset3 %d, got %d", len(data1)+len(data2), offset3)
	}

	// Verify all data can be retrieved
	result1, err := memtable.Get(offset1, length1)
	if err != nil || string(result1) != string(data1) {
		t.Errorf("Failed to retrieve data1: %v", err)
	}

	result2, err := memtable.Get(offset2, length2)
	if err != nil || string(result2) != string(data2) {
		t.Errorf("Failed to retrieve data2: %v", err)
	}

	result3, err := memtable.Get(offset3, length3)
	if err != nil || string(result3) != string(data3) {
		t.Errorf("Failed to retrieve data3: %v", err)
	}
}

func TestMemtable_Flush_Success(t *testing.T) {
	capacity := fs.BLOCK_SIZE
	file := createTestFile(t)
	page := createTestPage(capacity)
	defer cleanup(file, page)

	config := MemtableConfig{
		capacity: capacity,
		id:       1,
		page:     page,
		file:     file,
	}

	memtable, err := NewMemtable(config)
	if err != nil {
		t.Fatalf("NewMemtable failed: %v", err)
	}

	// Fill the memtable to trigger ready for flush
	testData := make([]byte, capacity-100)
	memtable.Put(testData)

	// Put data that exceeds capacity to trigger ready for flush
	memtable.Put(make([]byte, 200))

	if !memtable.readyForFlush {
		t.Fatalf("Expected memtable to be ready for flush")
	}

	n, fileOffset, err := memtable.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if n != len(page.Buf) {
		t.Errorf("Expected n %d, got %d", len(page.Buf), n)
	}
	if fileOffset < 0 {
		t.Errorf("Expected positive fileOffset, got %d", fileOffset)
	}
	if memtable.readyForFlush {
		t.Errorf("Expected readyForFlush to be false after flush, got %v", memtable.readyForFlush)
	}
}

func TestMemtable_Flush_NotReadyForFlush(t *testing.T) {
	capacity := fs.BLOCK_SIZE
	file := createTestFile(t)
	page := createTestPage(capacity)
	defer cleanup(file, page)

	config := MemtableConfig{
		capacity: capacity,
		id:       1,
		page:     page,
		file:     file,
	}

	memtable, err := NewMemtable(config)
	if err != nil {
		t.Fatalf("NewMemtable failed: %v", err)
	}

	// Try to flush without being ready
	_, _, err = memtable.Flush()
	if err != ErrMemtableNotReadyForFlush {
		t.Errorf("Expected ErrMemtableNotReadyForFlush, got %v", err)
	}
}

func TestMemtable_Discard(t *testing.T) {
	capacity := fs.BLOCK_SIZE
	file := createTestFile(t)
	page := createTestPage(capacity)
	defer cleanup(file, page)

	config := MemtableConfig{
		capacity: capacity,
		id:       1,
		page:     page,
		file:     file,
	}

	memtable, err := NewMemtable(config)
	if err != nil {
		t.Fatalf("NewMemtable failed: %v", err)
	}

	memtable.Discard()

	if memtable.file != nil {
		t.Errorf("Expected file to be nil after discard")
	}
	if memtable.page != nil {
		t.Errorf("Expected page to be nil after discard")
	}
}

func TestMemtable_Integration(t *testing.T) {
	capacity := fs.BLOCK_SIZE
	file := createTestFile(t)
	page := createTestPage(capacity)
	defer cleanup(file, page)

	config := MemtableConfig{
		capacity: capacity,
		id:       42,
		page:     page,
		file:     file,
	}

	memtable, err := NewMemtable(config)
	if err != nil {
		t.Fatalf("NewMemtable failed: %v", err)
	}

	// Test complete workflow: multiple puts, get, trigger flush, and flush
	testCases := [][]byte{
		[]byte("First entry"),
		[]byte("Second entry with more data"),
		[]byte("Third entry"),
	}

	var offsets []int
	var lengths []uint16

	// Put multiple entries
	for i, data := range testCases {
		offset, length, readyForFlush := memtable.Put(data)
		if readyForFlush {
			t.Logf("Memtable ready for flush after entry %d", i)
			break
		}
		offsets = append(offsets, offset)
		lengths = append(lengths, length)
	}

	// Verify all entries can be retrieved
	for i := range offsets {
		result, err := memtable.Get(offsets[i], lengths[i])
		if err != nil {
			t.Fatalf("Get failed for entry %d: %v", i, err)
		}
		if string(result) != string(testCases[i]) {
			t.Errorf("Entry %d mismatch: expected %s, got %s", i, testCases[i], result)
		}
	}

	// Fill up the memtable to trigger ready for flush
	for !memtable.readyForFlush {
		memtable.Put([]byte("filler"))
	}

	// Test flush
	n, fileOffset, err := memtable.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if n != capacity {
		t.Errorf("Expected flush size %d, got %d", capacity, n)
	}
	if fileOffset <= 0 {
		t.Errorf("Expected positive file offset, got %d", fileOffset)
	}
}

func TestMemtable_EdgeCases(t *testing.T) {
	capacity := fs.BLOCK_SIZE
	file := createTestFile(t)
	page := createTestPage(capacity)
	defer cleanup(file, page)

	config := MemtableConfig{
		capacity: capacity,
		id:       1,
		page:     page,
		file:     file,
	}

	memtable, err := NewMemtable(config)
	if err != nil {
		t.Fatalf("NewMemtable failed: %v", err)
	}

	// Test zero-length put
	offset, length, readyForFlush := memtable.Put([]byte{})
	if offset != 0 || length != 0 || readyForFlush {
		t.Errorf("Zero-length put: offset=%d, length=%d, readyForFlush=%v", offset, length, readyForFlush)
	}

	// Test zero-length get
	result, err := memtable.Get(0, 0)
	if err != nil {
		t.Fatalf("Zero-length get failed: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("Expected zero-length result, got %d", len(result))
	}

	// Test get at exact capacity boundary with zero length (should succeed)
	result, err = memtable.Get(capacity, 0)
	if err != nil {
		t.Errorf("Expected no error for boundary get with zero length, got %v", err)
	}
	if len(result) != 0 {
		t.Errorf("Expected zero-length result for boundary get, got %d", len(result))
	}

	// Test get beyond capacity boundary
	_, err = memtable.Get(capacity, 1)
	if err != ErrOffsetOutOfBounds {
		t.Errorf("Expected ErrOffsetOutOfBounds for beyond boundary get, got %v", err)
	}

	// Test put that exactly fills capacity
	exactData := make([]byte, capacity)
	offset, length, readyForFlush = memtable.Put(exactData)
	if offset != 0 || length != uint16(capacity) || readyForFlush {
		t.Errorf("Exact capacity put: offset=%d, length=%d, readyForFlush=%v", offset, length, readyForFlush)
	}

	// Next put should trigger ready for flush
	offset, length, readyForFlush = memtable.Put([]byte("overflow"))
	if offset != -1 || length != 0 || !readyForFlush {
		t.Errorf("Overflow put: offset=%d, length=%d, readyForFlush=%v", offset, length, readyForFlush)
	}
}

func TestGetBufForAppend_HappyPath_ReturnsSlice(t *testing.T) {
	capacity := fs.BLOCK_SIZE
	file := createTestFile(t)
	page := createTestPage(capacity)
	defer cleanup(file, page)

	m, err := NewMemtable(MemtableConfig{
		capacity: capacity,
		id:       1,
		page:     page,
		file:     file,
	})
	if err != nil {
		t.Fatalf("NewMemtable: %v", err)
	}

	buf, off, length, ready := m.GetBufForAppend(64)
	if ready {
		t.Fatal("ready = true with capacity available")
	}
	if length != 64 || off != 0 {
		t.Fatalf("(off=%d, length=%d), want (0, 64)", off, length)
	}
	if len(buf) != 64 {
		t.Fatalf("len(buf) = %d, want 64", len(buf))
	}
	// Buf returned points at the page — writes through it must persist.
	buf[0] = 0xAA
	if m.page.Buf[0] != 0xAA {
		t.Fatal("write through GetBufForAppend slice did not reach page")
	}
}

func TestGetBufForAppend_Overflow_FlagsReady(t *testing.T) {
	capacity := fs.BLOCK_SIZE
	file := createTestFile(t)
	page := createTestPage(capacity)
	defer cleanup(file, page)

	m, err := NewMemtable(MemtableConfig{
		capacity: capacity,
		id:       1,
		page:     page,
		file:     file,
	})
	if err != nil {
		t.Fatalf("NewMemtable: %v", err)
	}
	m.currentOffset = capacity - 10

	buf, off, length, ready := m.GetBufForAppend(64)
	if !ready {
		t.Fatal("ready = false when overflow expected")
	}
	if buf != nil || off != -1 || length != 0 {
		t.Fatalf("unexpected return: buf=%v off=%d length=%d", buf, off, length)
	}
	if !m.readyForFlush {
		t.Fatal("readyForFlush not flipped on overflow")
	}
}

func TestGetBufForRead_HappyPath_ReturnsSlice(t *testing.T) {
	capacity := fs.BLOCK_SIZE
	file := createTestFile(t)
	page := createTestPage(capacity)
	defer cleanup(file, page)

	m, err := NewMemtable(MemtableConfig{
		capacity: capacity,
		id:       1,
		page:     page,
		file:     file,
	})
	if err != nil {
		t.Fatalf("NewMemtable: %v", err)
	}
	// Seed page with a pattern.
	page.Buf[10] = 0x42

	buf, ok := m.GetBufForRead(10, 1)
	if !ok {
		t.Fatal("ok = false for in-range slice")
	}
	if len(buf) != 1 || buf[0] != 0x42 {
		t.Fatalf("buf = %v, want [0x42]", buf)
	}
}

func TestGetBufForRead_OutOfRange_ReturnsFalse(t *testing.T) {
	capacity := fs.BLOCK_SIZE
	file := createTestFile(t)
	page := createTestPage(capacity)
	defer cleanup(file, page)

	m, err := NewMemtable(MemtableConfig{
		capacity: capacity,
		id:       1,
		page:     page,
		file:     file,
	})
	if err != nil {
		t.Fatalf("NewMemtable: %v", err)
	}

	buf, ok := m.GetBufForRead(capacity-1, 5)
	if ok {
		t.Fatal("ok = true when slice exceeds capacity")
	}
	if buf != nil {
		t.Fatalf("buf = %v, want nil", buf)
	}
}

// TestMemtable_Flush_PwriteBatchError covers the
// `if err != nil { return 0, 0, err }` branch in Flush by flipping the
// underlying file into WriteDirectIO mode and offsetting the flush by
// one byte so the slice handed to PwriteBatch is no longer
// block-aligned. fs.WrapAppendFile.PwriteBatch then returns
// fs.ErrBufNoAlign before any io_uring syscall is issued, exercising the
// memtable's error-propagation path without depending on a real kernel
// fault. The docker test loopback doesn't actually open the file with
// O_DIRECT (DIRECT_IO is unsupported on loopback ext4 and the driver
// falls back), but PwriteBatch's alignment check only inspects the
// WriteDirectIO flag — toggling it on the test fixture is enough.
func TestMemtable_Flush_PwriteBatchError(t *testing.T) {
	capacity := fs.BLOCK_SIZE * 2
	file := createTestFile(t)
	page := createTestPage(capacity)
	defer cleanup(file, page)

	m, err := NewMemtable(MemtableConfig{
		capacity: capacity,
		id:       7,
		page:     page,
		file:     file,
	})
	if err != nil {
		t.Fatalf("NewMemtable: %v", err)
	}

	// Force the alignment guard inside PwriteBatch to fire by:
	//   1. Turning DirectIO on so the guard runs at all.
	//   2. Shifting the flush start by one byte so buf[1:] hits the guard
	//      with a non-block-aligned slice.
	file.WriteDirectIO = true
	m.flushStartOffset = 1
	m.currentOffset = capacity
	m.readyForFlush = true

	n, fileOffset, err := m.Flush()
	if err == nil {
		t.Fatal("Flush err = nil, want fs.ErrBufNoAlign")
	}
	if err != fs.ErrBufNoAlign {
		t.Fatalf("Flush err = %v, want fs.ErrBufNoAlign", err)
	}
	if n != 0 || fileOffset != 0 {
		t.Fatalf("Flush = (%d, %d, _), want (0, 0, _) on error", n, fileOffset)
	}
	// The manager recycles the memtable under a new id after a failed
	// flush, so it must come back empty and its file slot consumed.
	if m.readyForFlush || m.currentOffset != 0 {
		t.Fatalf("memtable not reset after failed flush: readyForFlush=%v currentOffset=%d", m.readyForFlush, m.currentOffset)
	}
	if got := file.PhysicalWriteOffset.Load(); got != int64(capacity) {
		t.Fatalf("PhysicalWriteOffset = %d after failed flush, want %d (one full slot)", got, capacity)
	}
}

func TestMemtable_Flush_StaggeredStart_AdvancesWriteOffset(t *testing.T) {
	capacity := fs.BLOCK_SIZE * 2
	file := createTestFile(t)
	page := createTestPage(capacity)
	defer cleanup(file, page)

	m, err := NewMemtable(MemtableConfig{
		capacity: capacity,
		id:       1,
		page:     page,
		file:     file,
	})
	if err != nil {
		t.Fatalf("NewMemtable: %v", err)
	}

	// Simulate a stagger: skip the first block. After flush the file
	// pointer should have advanced by full capacity, not just by the data
	// portion.
	m.flushStartOffset = fs.BLOCK_SIZE
	m.readyForFlush = true
	m.currentOffset = capacity

	startPhys := file.PhysicalWriteOffset.Load()
	n, _, err := m.Flush()
	if err != nil {
		t.Fatalf("Flush: %v", err)
	}
	// n should be flushStartOffset + actual bytes written = capacity.
	if n != capacity {
		t.Fatalf("n = %d, want %d", n, capacity)
	}
	if file.PhysicalWriteOffset.Load()-startPhys != int64(capacity) {
		t.Fatalf("file advanced by %d, want %d",
			file.PhysicalWriteOffset.Load()-startPhys, capacity)
	}
	// flushStartOffset must reset to 0 after first flush.
	if m.flushStartOffset != 0 {
		t.Fatalf("flushStartOffset = %d, want 0 after first flush", m.flushStartOffset)
	}
}

// A sealed memtable takes no more entries, so the shard rotates it out on
// the next write and it flushes like a full one.
func TestMemtable_Seal_NextAppendAsksForFlush(t *testing.T) {
	capacity := fs.BLOCK_SIZE * 2
	file := createTestFile(t)
	page := createTestPage(capacity)
	defer cleanup(file, page)

	m, err := NewMemtable(MemtableConfig{capacity: capacity, id: 1, page: page, file: file})
	if err != nil {
		t.Fatalf("NewMemtable: %v", err)
	}
	if _, _, _, ready := m.GetBufForAppend(16); ready {
		t.Fatal("append into a fresh memtable asked for a flush")
	}
	m.Seal()
	if buf, _, _, ready := m.GetBufForAppend(16); !ready || buf != nil {
		t.Fatalf("append after Seal: readyForFlush = %v, buf = %v; want true and nil", ready, buf)
	}
	if n, _, err := m.Flush(); err != nil || n != capacity {
		t.Fatalf("Flush after Seal = (%d, _, %v), want (%d, _, nil)", n, err, capacity)
	}
}
