package memtables

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/Meesho/BharatMLStack/ssd-cache/internal/fs"
)

// Helper function to create a test file for memtable manager
func createManagerTestFile(t *testing.T) *fs.WrapAppendFile {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_manager.dat")

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

func TestNewMemtableManager(t *testing.T) {
	tests := []struct {
		name        string
		capacity    int32
		expectError bool
	}{
		{
			name:        "valid capacity aligned to block size",
			capacity:    fs.BLOCK_SIZE,
			expectError: false,
		},
		{
			name:        "valid larger capacity",
			capacity:    fs.BLOCK_SIZE * 2,
			expectError: false,
		},
		{
			name:        "invalid capacity not aligned",
			capacity:    fs.BLOCK_SIZE + 1,
			expectError: true,
		},
		{
			name:        "very small capacity",
			capacity:    1,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := createManagerTestFile(t)
			defer testFile.Close()

			manager, err := NewMemtableManager(testFile, tt.capacity)

			if tt.expectError {
				if err == nil {
					t.Errorf("NewMemtableManager() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("NewMemtableManager() unexpected error = %v", err)
				return
			}

			if manager == nil {
				t.Error("NewMemtableManager() returned nil manager")
				return
			}

			// Verify manager state
			if manager.capacity != tt.capacity {
				t.Errorf("capacity = %v, want %v", manager.capacity, tt.capacity)
			}

			if manager.file != testFile {
				t.Error("file not set correctly")
			}

			if manager.memtable1 == nil || manager.memtable2 == nil {
				t.Error("memtables not initialized")
			}

			if manager.activeMemtable != manager.memtable1 {
				t.Error("activeMemtable should initially be memtable1")
			}

			if manager.nextFileOffset != 2*int64(tt.capacity) {
				t.Errorf("nextFileOffset = %v, want %v", manager.nextFileOffset, 2*int64(tt.capacity))
			}

			if manager.nextId != 2 {
				t.Errorf("nextId = %v, want %v", manager.nextId, 2)
			}

			if manager.flushInProgress != false {
				t.Error("flushInProgress should initially be false")
			}

			// Verify memtables are initialized correctly
			if manager.memtable1.Id != 0 {
				t.Errorf("memtable1.Id = %v, want %v", manager.memtable1.Id, 0)
			}

			if manager.memtable2.Id != 1 {
				t.Errorf("memtable2.Id = %v, want %v", manager.memtable2.Id, 1)
			}

			// Clean up: close flush goroutine
			close(manager.flushChan)
		})
	}
}

func TestMemtableManager_GetMemtable(t *testing.T) {
	testFile := createManagerTestFile(t)
	defer testFile.Close()

	capacity := int32(fs.BLOCK_SIZE)
	manager, err := NewMemtableManager(testFile, capacity)
	if err != nil {
		t.Fatalf("NewMemtableManager() error = %v", err)
	}
	defer close(manager.flushChan)

	memtable, id, offset := manager.GetMemtable()

	if memtable != manager.activeMemtable {
		t.Error("GetMemtable() returned wrong memtable")
	}

	if id != manager.activeMemtable.Id {
		t.Errorf("GetMemtable() id = %v, want %v", id, manager.activeMemtable.Id)
	}

	expectedOffset := uint64(manager.activeMemtable.Id) * uint64(capacity)
	if offset != expectedOffset {
		t.Errorf("GetMemtable() offset = %v, want %v", offset, expectedOffset)
	}
}

func TestMemtableManager_GetMemtableById(t *testing.T) {
	testFile := createManagerTestFile(t)
	defer testFile.Close()

	capacity := int32(fs.BLOCK_SIZE)
	manager, err := NewMemtableManager(testFile, capacity)
	if err != nil {
		t.Fatalf("NewMemtableManager() error = %v", err)
	}
	defer close(manager.flushChan)

	tests := []struct {
		name      string
		id        uint32
		expectNil bool
		expectMem *Memtable
	}{
		{
			name:      "get memtable1 by id 0",
			id:        0,
			expectNil: false,
			expectMem: manager.memtable1,
		},
		{
			name:      "get memtable2 by id 1",
			id:        1,
			expectNil: false,
			expectMem: manager.memtable2,
		},
		{
			name:      "get non-existent memtable",
			id:        999,
			expectNil: true,
			expectMem: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.GetMemtableById(tt.id)

			if tt.expectNil {
				if result != nil {
					t.Errorf("GetMemtableById() = %v, want nil", result)
				}
			} else {
				if result != tt.expectMem {
					t.Errorf("GetMemtableById() = %v, want %v", result, tt.expectMem)
				}
			}
		})
	}
}

func TestMemtableManager_Flush(t *testing.T) {
	testFile := createManagerTestFile(t)
	defer testFile.Close()

	capacity := int32(fs.BLOCK_SIZE)
	manager, err := NewMemtableManager(testFile, capacity)
	if err != nil {
		t.Fatalf("NewMemtableManager() error = %v", err)
	}
	defer close(manager.flushChan)

	// Get initial active memtable
	initialActive := manager.activeMemtable
	var otherMemtable *Memtable
	if initialActive == manager.memtable1 {
		otherMemtable = manager.memtable2
	} else {
		otherMemtable = manager.memtable1
	}

	// Make the memtable ready for flush by filling it
	initialActive.readyForFlush = true

	// Perform flush
	err = manager.Flush()
	if err != nil {
		t.Errorf("Flush() error = %v", err)
	}

	// Verify that active memtable was swapped
	if manager.activeMemtable == initialActive {
		t.Error("Flush() should have swapped active memtable")
	}

	if manager.activeMemtable != otherMemtable {
		t.Error("Flush() should have swapped to the other memtable")
	}

	// Wait a bit for the flush goroutine to process
	time.Sleep(100 * time.Millisecond)

	// Verify flush was initiated
	if !manager.flushInProgress {
		t.Error("flushInProgress should be true during flush")
	}
}

func TestMemtableManager_FlushBasic(t *testing.T) {
	testFile := createManagerTestFile(t)
	defer testFile.Close()

	capacity := int32(fs.BLOCK_SIZE)
	manager, err := NewMemtableManager(testFile, capacity)
	if err != nil {
		t.Fatalf("NewMemtableManager() error = %v", err)
	}
	defer close(manager.flushChan)

	// Get initial state
	initialNextId := manager.nextId
	initialNextFileOffset := manager.nextFileOffset

	// Make memtable ready for flush
	manager.activeMemtable.readyForFlush = true

	// Trigger flush
	err = manager.Flush()
	if err != nil {
		t.Errorf("Flush() error = %v", err)
	}

	// Wait for flush to complete
	select {
	case <-manager.flushCompleteChan:
		// Flush completed successfully
	case <-time.After(2 * time.Second):
		t.Error("Flush did not complete within timeout")
		return
	}

	// Verify that nextId was incremented
	expectedNextId := initialNextId + 1
	if manager.nextId != expectedNextId {
		t.Errorf("nextId = %v, want %v", manager.nextId, expectedNextId)
	}

	// Verify that nextFileOffset was updated
	expectedOffset := initialNextFileOffset + int64(capacity)
	if manager.nextFileOffset != expectedOffset {
		t.Errorf("nextFileOffset = %v, want %v", manager.nextFileOffset, expectedOffset)
	}
}

func TestMemtableManager_SequentialFlush(t *testing.T) {
	testFile := createManagerTestFile(t)
	defer testFile.Close()

	capacity := int32(fs.BLOCK_SIZE)
	manager, err := NewMemtableManager(testFile, capacity)
	if err != nil {
		t.Fatalf("NewMemtableManager() error = %v", err)
	}
	defer close(manager.flushChan)

	// Get initial state
	initialNextId := manager.nextId

	// Perform first flush
	manager.activeMemtable.readyForFlush = true
	err = manager.Flush()
	if err != nil {
		t.Errorf("First flush error: %v", err)
	}

	// Wait for first flush to complete
	select {
	case <-manager.flushCompleteChan:
		// First flush completed
	case <-time.After(2 * time.Second):
		t.Error("First flush did not complete within timeout")
		return
	}

	// Perform second flush
	manager.activeMemtable.readyForFlush = true
	err = manager.Flush()
	if err != nil {
		t.Errorf("Second flush error: %v", err)
	}

	// Wait for second flush to complete
	select {
	case <-manager.flushCompleteChan:
		// Second flush completed
	case <-time.After(2 * time.Second):
		t.Error("Second flush did not complete within timeout")
		return
	}

	// Verify final state - should have incremented by 2
	expectedNextId := initialNextId + 2
	if manager.nextId != expectedNextId {
		t.Errorf("nextId = %v, want %v", manager.nextId, expectedNextId)
	}
}

// Note: TestMemtableManager_FlushWithError removed because we can't easily
// mock file errors with the real WrapAppendFile implementation

func TestMemtableManager_InitialState(t *testing.T) {
	testFile := createManagerTestFile(t)
	defer testFile.Close()

	capacity := int32(fs.BLOCK_SIZE)
	manager, err := NewMemtableManager(testFile, capacity)
	if err != nil {
		t.Fatalf("NewMemtableManager() error = %v", err)
	}
	defer close(manager.flushChan)

	// Test initial state invariants
	if manager.memtable1.Id == manager.memtable2.Id {
		t.Error("memtable1 and memtable2 should have different IDs")
	}

	if manager.activeMemtable != manager.memtable1 && manager.activeMemtable != manager.memtable2 {
		t.Error("activeMemtable should be one of the two memtables")
	}

	if cap(manager.flushChan) != 1 {
		t.Errorf("flushChan capacity = %v, want %v", cap(manager.flushChan), 1)
	}

	if cap(manager.flushCompleteChan) != 1 {
		t.Errorf("flushCompleteChan capacity = %v, want %v", cap(manager.flushCompleteChan), 1)
	}
}
