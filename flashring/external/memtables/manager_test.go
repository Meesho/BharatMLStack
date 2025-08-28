package memtables

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Meesho/BharatMLStack/flashring/external/fs"
)

// Helper function to create a mock file for testing
func createTestFileForManager(t *testing.T) *fs.WrapAppendFile {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_memtable_manager.dat")

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
	return file
}

func TestNewMemtableManager_Success(t *testing.T) {
	capacity := int32(fs.BLOCK_SIZE * 2) // 8192 bytes
	file := createTestFileForManager(t)
	defer file.Close()

	manager, err := NewMemtableManager(file, capacity)
	if err != nil {
		t.Fatalf("NewMemtableManager failed: %v", err)
	}

	// Verify initial state
	if manager.file != file {
		t.Errorf("Expected file to be set correctly")
	}
	if manager.Capacity != capacity {
		t.Errorf("Expected capacity %d, got %d", capacity, manager.Capacity)
	}
	if manager.memtable1 == nil {
		t.Errorf("Expected memtable1 to be initialized")
	}
	if manager.memtable2 == nil {
		t.Errorf("Expected memtable2 to be initialized")
	}
	if manager.activeMemtable != manager.memtable1 {
		t.Errorf("Expected activeMemtable to be memtable1 initially")
	}
	if manager.nextFileOffset != 2*int64(capacity) {
		t.Errorf("Expected nextFileOffset to be %d, got %d", 2*int64(capacity), manager.nextFileOffset)
	}
	if manager.nextId != 2 {
		t.Errorf("Expected nextId to be 2, got %d", manager.nextId)
	}
	if cap(manager.semaphore) != 1 {
		t.Errorf("Expected semaphore capacity to be 1, got %d", cap(manager.semaphore))
	}

	// Verify memtable initial IDs
	if manager.memtable1.Id != 0 {
		t.Errorf("Expected memtable1 ID to be 0, got %d", manager.memtable1.Id)
	}
	if manager.memtable2.Id != 1 {
		t.Errorf("Expected memtable2 ID to be 1, got %d", manager.memtable2.Id)
	}
}

func TestNewMemtableManager_InvalidCapacity(t *testing.T) {
	// Test with capacity not aligned to block size
	capacity := int32(fs.BLOCK_SIZE + 1) // Should fail alignment check
	file := createTestFileForManager(t)
	defer file.Close()

	_, err := NewMemtableManager(file, capacity)
	if err == nil {
		t.Errorf("Expected NewMemtableManager to fail with invalid capacity")
	}
}

func TestNewMemtableManager_NilFile(t *testing.T) {
	capacity := int32(fs.BLOCK_SIZE * 2)

	_, err := NewMemtableManager(nil, capacity)
	if err == nil {
		t.Errorf("Expected NewMemtableManager to fail with nil file")
	}
}

func TestMemtableManager_GetMemtable(t *testing.T) {
	capacity := int32(fs.BLOCK_SIZE * 2)
	file := createTestFileForManager(t)
	defer file.Close()

	manager, err := NewMemtableManager(file, capacity)
	if err != nil {
		t.Fatalf("NewMemtableManager failed: %v", err)
	}

	memtable, id, offset := manager.GetMemtable()

	// Initially should return memtable1
	if memtable != manager.memtable1 {
		t.Errorf("Expected to get memtable1")
	}
	if id != 0 {
		t.Errorf("Expected ID 0, got %d", id)
	}
	expectedOffset := uint64(0) * uint64(capacity)
	if offset != expectedOffset {
		t.Errorf("Expected offset %d, got %d", expectedOffset, offset)
	}
}

func TestMemtableManager_GetMemtableById(t *testing.T) {
	capacity := int32(fs.BLOCK_SIZE * 2)
	file := createTestFileForManager(t)
	defer file.Close()

	manager, err := NewMemtableManager(file, capacity)
	if err != nil {
		t.Fatalf("NewMemtableManager failed: %v", err)
	}

	// Test getting memtable1 by ID
	memtable := manager.GetMemtableById(0)
	if memtable != manager.memtable1 {
		t.Errorf("Expected to get memtable1 for ID 0")
	}

	// Test getting memtable2 by ID
	memtable = manager.GetMemtableById(1)
	if memtable != manager.memtable2 {
		t.Errorf("Expected to get memtable2 for ID 1")
	}

	// Test getting non-existent memtable
	memtable = manager.GetMemtableById(999)
	if memtable != nil {
		t.Errorf("Expected nil for non-existent ID, got %v", memtable)
	}
}

func TestMemtableManager_Flush(t *testing.T) {
	capacity := int32(fs.BLOCK_SIZE * 2)
	file := createTestFileForManager(t)
	defer file.Close()

	manager, err := NewMemtableManager(file, capacity)
	if err != nil {
		t.Fatalf("NewMemtableManager failed: %v", err)
	}

	// Verify initial state
	originalActive := manager.activeMemtable
	originalNextId := manager.nextId

	// Perform flush
	err = manager.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	// Verify active memtable swapped
	if manager.activeMemtable == originalActive {
		t.Errorf("Expected active memtable to be swapped")
	}

	// Active should now be the other memtable
	if originalActive == manager.memtable1 {
		if manager.activeMemtable != manager.memtable2 {
			t.Errorf("Expected active memtable to be memtable2")
		}
	} else {
		if manager.activeMemtable != manager.memtable1 {
			t.Errorf("Expected active memtable to be memtable1")
		}
	}

	// Give time for background goroutine to complete
	time.Sleep(100 * time.Millisecond)

	// Verify nextId was incremented (this happens in background)
	if manager.nextId <= originalNextId {
		t.Errorf("Expected nextId to be incremented, got %d, expected > %d", manager.nextId, originalNextId)
	}
}

func TestMemtableManager_FlushSwapsBetweenMemtables(t *testing.T) {
	capacity := int32(fs.BLOCK_SIZE * 2)
	file := createTestFileForManager(t)
	defer file.Close()

	manager, err := NewMemtableManager(file, capacity)
	if err != nil {
		t.Fatalf("NewMemtableManager failed: %v", err)
	}

	// Initially active is memtable1
	if manager.activeMemtable != manager.memtable1 {
		t.Fatalf("Expected initial active to be memtable1")
	}

	// First flush - should swap to memtable2
	err = manager.Flush()
	if err != nil {
		t.Fatalf("First flush failed: %v", err)
	}
	if manager.activeMemtable != manager.memtable2 {
		t.Errorf("Expected active to be memtable2 after first flush")
	}

	// Second flush - should swap back to memtable1
	err = manager.Flush()
	if err != nil {
		t.Fatalf("Second flush failed: %v", err)
	}
	if manager.activeMemtable != manager.memtable1 {
		t.Errorf("Expected active to be memtable1 after second flush")
	}
}

func TestMemtableManager_FlushConcurrency(t *testing.T) {
	capacity := int32(fs.BLOCK_SIZE * 2)
	file := createTestFileForManager(t)
	defer file.Close()

	manager, err := NewMemtableManager(file, capacity)
	if err != nil {
		t.Fatalf("NewMemtableManager failed: %v", err)
	}

	const numConcurrentFlushes = 10
	var wg sync.WaitGroup
	errors := make(chan error, numConcurrentFlushes)

	// Launch multiple concurrent flushes
	for i := 0; i < numConcurrentFlushes; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := manager.Flush(); err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent flush failed: %v", err)
	}

	// Give time for all background operations to complete
	time.Sleep(200 * time.Millisecond)

	// Verify manager is still in a valid state
	memtable, id, offset := manager.GetMemtable()
	if memtable == nil {
		t.Errorf("Active memtable should not be nil")
	}
	if id != memtable.Id {
		t.Errorf("Returned ID %d should match memtable ID %d", id, memtable.Id)
	}
	expectedOffset := uint64(memtable.Id) * uint64(capacity)
	if offset != expectedOffset {
		t.Errorf("Expected offset %d, got %d", expectedOffset, offset)
	}
}

func TestMemtableManager_GetMemtableAfterFlush(t *testing.T) {
	capacity := int32(fs.BLOCK_SIZE * 2)
	file := createTestFileForManager(t)
	defer file.Close()

	manager, err := NewMemtableManager(file, capacity)
	if err != nil {
		t.Fatalf("NewMemtableManager failed: %v", err)
	}

	// Get initial memtable
	initialMemtable, initialId, _ := manager.GetMemtable()

	// Perform flush
	err = manager.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	// Get memtable after flush
	newMemtable, newId, newOffset := manager.GetMemtable()

	// Should be different memtable
	if newMemtable == initialMemtable {
		t.Errorf("Expected different memtable after flush")
	}
	if newId == initialId {
		t.Errorf("Expected different ID after flush")
	}

	// Offset calculation should be correct
	expectedOffset := uint64(newId) * uint64(capacity)
	if newOffset != expectedOffset {
		t.Errorf("Expected offset %d, got %d", expectedOffset, newOffset)
	}
}

func TestMemtableManager_Integration(t *testing.T) {
	capacity := int32(fs.BLOCK_SIZE * 2)
	file := createTestFileForManager(t)
	defer file.Close()

	manager, err := NewMemtableManager(file, capacity)
	if err != nil {
		t.Fatalf("NewMemtableManager failed: %v", err)
	}

	// Test complete workflow: get memtable, put data, flush, repeat
	testData := []byte("Hello, MemtableManager!")

	// Get initial memtable and put some data
	memtable, id, _ := manager.GetMemtable()
	offset, length, readyForFlush := memtable.Put(testData)
	if readyForFlush {
		t.Errorf("Memtable should not be ready for flush after small put")
	}

	// Verify data can be retrieved
	data, err := memtable.Get(offset, length)
	if err != nil {
		t.Fatalf("Failed to get data: %v", err)
	}
	if string(data) != string(testData) {
		t.Errorf("Expected %s, got %s", testData, data)
	}

	// Verify GetMemtableById works
	retrievedMemtable := manager.GetMemtableById(id)
	if retrievedMemtable != memtable {
		t.Errorf("GetMemtableById should return the same memtable")
	}

	// Perform flush and verify state changes
	err = manager.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	// Get new active memtable
	newMemtable, newId, _ := manager.GetMemtable()
	if newMemtable == memtable {
		t.Errorf("Active memtable should have changed after flush")
	}
	if newId == id {
		t.Errorf("Active memtable ID should have changed after flush")
	}

	// Old memtable should still be retrievable by its original ID
	oldMemtable := manager.GetMemtableById(id)
	if oldMemtable != memtable {
		t.Errorf("Should still be able to retrieve old memtable by ID")
	}

	// Give background flush time to complete
	time.Sleep(100 * time.Millisecond)
}
