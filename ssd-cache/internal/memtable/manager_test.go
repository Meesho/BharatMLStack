package memtable

import (
	"os"
	"testing"
	"time"

	"github.com/Meesho/BharatMLStack/ssd-cache/internal/allocator"
)

func createManagerTestFile(t *testing.T) *os.File {
	file, err := os.CreateTemp("", "memtable_manager_test_*.bin")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	return file
}

func createTestAllocator() *allocator.AlignedPageAllocator {
	return allocator.NewAlignedPageAllocator(allocator.AlignedPageAllocatorConfig{
		PageSizeAlignement: 4096,
		Multiplier:         1,
		MaxPages:           10,
	})
}

func TestNewMemtableManager(t *testing.T) {
	file := createManagerTestFile(t)
	defer os.Remove(file.Name())
	defer file.Close()

	allocator := createTestAllocator()
	capacity := int64(4096)

	manager := NewMemtableManager(file, capacity, allocator)

	// Verify manager initialization
	if manager.file != file {
		t.Error("File not set correctly")
	}
	if manager.capacity != capacity {
		t.Error("Capacity not set correctly")
	}
	if manager.allocator != allocator {
		t.Error("Allocator not set correctly")
	}
	if manager.nextFileOffset != 2*capacity {
		t.Errorf("Expected nextFileOffset to be %d, got %d", 2*capacity, manager.nextFileOffset)
	}
	if manager.nextId != 2 {
		t.Errorf("Expected nextId to be 2, got %d", manager.nextId)
	}

	// Verify memtables initialization
	if manager.memtable1 == nil {
		t.Error("memtable1 not initialized")
	}
	if manager.memtable2 == nil {
		t.Error("memtable2 not initialized")
	}
	if manager.activeMemtable != manager.memtable1 {
		t.Error("activeMemtable should be memtable1 initially")
	}

	// Verify file offsets
	if manager.memtable1.fileOffset != 0 {
		t.Errorf("Expected memtable1 fileOffset to be 0, got %d", manager.memtable1.fileOffset)
	}
	if manager.memtable2.fileOffset != capacity {
		t.Errorf("Expected memtable2 fileOffset to be %d, got %d", capacity, manager.memtable2.fileOffset)
	}

	// Verify IDs
	if manager.memtable1.Id != 0 {
		t.Errorf("Expected memtable1 ID to be 0, got %d", manager.memtable1.Id)
	}
	if manager.memtable2.Id != 1 {
		t.Errorf("Expected memtable2 ID to be 1, got %d", manager.memtable2.Id)
	}
}

func TestGetMemtable(t *testing.T) {
	file := createManagerTestFile(t)
	defer os.Remove(file.Name())
	defer file.Close()

	allocator := createTestAllocator()
	capacity := int64(4096)
	manager := NewMemtableManager(file, capacity, allocator)

	memtable, id, fileOffset := manager.GetMemtable()

	if memtable != manager.memtable1 {
		t.Error("Expected to get memtable1 as active")
	}
	if id != 0 {
		t.Errorf("Expected ID to be 0, got %d", id)
	}
	if fileOffset != 0 {
		t.Errorf("Expected fileOffset to be 0, got %d", fileOffset)
	}
}

func TestGetMemtableById(t *testing.T) {
	file := createManagerTestFile(t)
	defer os.Remove(file.Name())
	defer file.Close()

	allocator := createTestAllocator()
	capacity := int64(4096)
	manager := NewMemtableManager(file, capacity, allocator)

	// Test getting existing memtables
	memtable := manager.GetMemtableById(0)
	if memtable != manager.memtable1 {
		t.Error("Expected to get memtable1 for ID 0")
	}

	memtable = manager.GetMemtableById(1)
	if memtable != manager.memtable2 {
		t.Error("Expected to get memtable2 for ID 1")
	}

	// Test getting non-existent memtable
	memtable = manager.GetMemtableById(999)
	if memtable != nil {
		t.Error("Expected nil for non-existent ID")
	}
}

func TestFlush(t *testing.T) {
	file := createManagerTestFile(t)
	defer os.Remove(file.Name())
	defer file.Close()

	allocator := createTestAllocator()
	capacity := int64(4096)
	manager := NewMemtableManager(file, capacity, allocator)

	// Initial state
	initialActive := manager.activeMemtable
	initialActiveId := initialActive.Id

	// Make the memtable ready for flush
	initialActive.readyForFlush = true

	// Flush
	err := manager.Flush()
	if err != nil {
		t.Errorf("Flush failed: %v", err)
	}

	// Verify active memtable has swapped
	if manager.activeMemtable == initialActive {
		t.Error("Active memtable should have swapped")
	}
	if manager.activeMemtable != manager.memtable2 {
		t.Error("Active memtable should be memtable2 after flush")
	}

	// Wait for async flush to complete
	time.Sleep(100 * time.Millisecond)

	// Verify the flushed memtable got new ID and fileOffset
	if initialActive.Id == initialActiveId {
		t.Error("Flushed memtable should have new ID")
	}
	if initialActive.Id != 2 {
		t.Errorf("Expected flushed memtable ID to be 2, got %d", initialActive.Id)
	}
	if initialActive.fileOffset != 2*capacity {
		t.Errorf("Expected flushed memtable fileOffset to be %d, got %d", 2*capacity, initialActive.fileOffset)
	}
	if manager.nextFileOffset != 3*capacity {
		t.Errorf("Expected nextFileOffset to be %d, got %d", 3*capacity, manager.nextFileOffset)
	}
}

func TestMultipleFlushes(t *testing.T) {
	file := createManagerTestFile(t)
	defer os.Remove(file.Name())
	defer file.Close()

	allocator := createTestAllocator()
	capacity := int64(4096)
	manager := NewMemtableManager(file, capacity, allocator)

	// Track initial states
	memtable1 := manager.memtable1
	memtable2 := manager.memtable2

	// First flush (memtable1 -> memtable2)
	memtable1.readyForFlush = true
	err := manager.Flush()
	if err != nil {
		t.Errorf("First flush failed: %v", err)
	}
	if manager.activeMemtable != memtable2 {
		t.Error("Active should be memtable2 after first flush")
	}

	// Wait for async flush
	time.Sleep(100 * time.Millisecond)

	// Second flush (memtable2 -> memtable1)
	memtable2.readyForFlush = true
	err = manager.Flush()
	if err != nil {
		t.Errorf("Second flush failed: %v", err)
	}
	if manager.activeMemtable != memtable1 {
		t.Error("Active should be memtable1 after second flush")
	}

	// Wait for async flush
	time.Sleep(100 * time.Millisecond)

	// Verify IDs have incremented
	// After 2 flushes: memtable1 (initially ID 0) becomes ID 2, memtable2 (initially ID 1) becomes ID 3
	if memtable1.Id != 2 {
		t.Errorf("Expected memtable1 ID to be 2, got %d", memtable1.Id)
	}
	if memtable2.Id != 3 {
		t.Errorf("Expected memtable2 ID to be 3, got %d", memtable2.Id)
	}

	// Verify file offsets
	if memtable1.fileOffset != 2*capacity {
		t.Errorf("Expected memtable1 fileOffset to be %d, got %d", 2*capacity, memtable1.fileOffset)
	}
	if memtable2.fileOffset != 3*capacity {
		t.Errorf("Expected memtable2 fileOffset to be %d, got %d", 3*capacity, memtable2.fileOffset)
	}
}

func TestFileOffsetProgression(t *testing.T) {
	file := createManagerTestFile(t)
	defer os.Remove(file.Name())
	defer file.Close()

	allocator := createTestAllocator()
	capacity := int64(4096)
	manager := NewMemtableManager(file, capacity, allocator)

	// Verify initial file offsets
	expectedOffsets := []int64{0, capacity}
	actualOffsets := []int64{manager.memtable1.fileOffset, manager.memtable2.fileOffset}

	for i, expected := range expectedOffsets {
		if actualOffsets[i] != expected {
			t.Errorf("Initial offset %d: expected %d, got %d", i, expected, actualOffsets[i])
		}
	}

	// Perform multiple flushes and verify file offset progression
	for i := 0; i < 3; i++ {
		manager.activeMemtable.readyForFlush = true
		manager.Flush()
		time.Sleep(100 * time.Millisecond)
	}

	// After 3 flushes, check that file offsets have progressed correctly
	if manager.nextFileOffset != 5*capacity {
		t.Errorf("Expected nextFileOffset to be %d, got %d", 5*capacity, manager.nextFileOffset)
	}
}

func TestGetMemtableByIdAfterFlush(t *testing.T) {
	file := createManagerTestFile(t)
	defer os.Remove(file.Name())
	defer file.Close()

	allocator := createTestAllocator()
	capacity := int64(4096)
	manager := NewMemtableManager(file, capacity, allocator)

	// Initially, we should be able to get memtables by ID 0 and 1
	if manager.GetMemtableById(0) == nil {
		t.Error("Should be able to get memtable with ID 0")
	}
	if manager.GetMemtableById(1) == nil {
		t.Error("Should be able to get memtable with ID 1")
	}

	// After flush, old IDs should become invalid
	manager.activeMemtable.readyForFlush = true
	manager.Flush()
	time.Sleep(100 * time.Millisecond)

	// Now ID 0 should be invalid, but ID 1 and 2 should be valid
	if manager.GetMemtableById(0) != nil {
		t.Error("ID 0 should be invalid after flush")
	}
	if manager.GetMemtableById(1) == nil {
		t.Error("ID 1 should still be valid")
	}
	if manager.GetMemtableById(2) == nil {
		t.Error("ID 2 should be valid after flush")
	}
}
