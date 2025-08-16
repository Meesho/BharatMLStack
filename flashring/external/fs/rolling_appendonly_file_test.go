//go:build linux
// +build linux

package fs

import (
	"os"
	"path/filepath"
	"testing"
	"unsafe"
)

// Helper function to create aligned buffers for DirectIO
func createAlignedBuffer(size, alignment int) []byte {
	// Allocate more memory than needed to ensure we can find an aligned address
	buf := make([]byte, size+alignment)

	// Find the aligned address
	addr := uintptr(unsafe.Pointer(&buf[0]))
	alignedAddr := (addr + uintptr(alignment-1)) &^ uintptr(alignment-1)

	// Calculate the offset
	offset := alignedAddr - addr

	// Return the aligned slice
	return buf[offset : offset+uintptr(size)]
}

func TestNewRollingAppendFile(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_rolling_file.dat")

	config := FileConfig{
		Filename:          filename,
		MaxFileSize:       1024 * 1024, // 1MB
		FilePunchHoleSize: 64 * 1024,   // 64KB
		BlockSize:         4096,
	}

	raf, err := NewRollingAppendFile(config)
	if err != nil {
		t.Fatalf("Failed to create RollingAppendFile: %v", err)
	}
	defer cleanup(raf)

	// Verify initial state
	if raf.MaxFileSize != config.MaxFileSize {
		t.Errorf("Expected MaxFileSize %d, got %d", config.MaxFileSize, raf.MaxFileSize)
	}
	if raf.FilePunchHoleSize != config.FilePunchHoleSize {
		t.Errorf("Expected FilePunchHoleSize %d, got %d", config.FilePunchHoleSize, raf.FilePunchHoleSize)
	}
	if raf.blockSize != config.BlockSize {
		t.Errorf("Expected BlockSize %d, got %d", config.BlockSize, raf.blockSize)
	}
	if raf.CurrentLogicalOffset != 0 {
		t.Errorf("Expected CurrentLogicalOffset 0, got %d", raf.CurrentLogicalOffset)
	}
	if raf.CurrentPhysicalOffset != 0 {
		t.Errorf("Expected CurrentPhysicalOffset 0, got %d", raf.CurrentPhysicalOffset)
	}
}

func TestNewRollingAppendFile_DefaultBlockSize(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_rolling_file.dat")

	config := FileConfig{
		Filename:          filename,
		MaxFileSize:       1024 * 1024,
		FilePunchHoleSize: 64 * 1024,
		BlockSize:         0, // Should default to BLOCK_SIZE
	}

	raf, err := NewRollingAppendFile(config)
	if err != nil {
		t.Fatalf("Failed to create RollingAppendFile: %v", err)
	}
	defer cleanup(raf)

	if raf.blockSize != BLOCK_SIZE {
		t.Errorf("Expected default BlockSize %d, got %d", BLOCK_SIZE, raf.blockSize)
	}
}

func TestPwrite_Success(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_rolling_file.dat")

	config := FileConfig{
		Filename:          filename,
		MaxFileSize:       1024 * 1024,
		FilePunchHoleSize: 64 * 1024,
		BlockSize:         4096,
	}

	raf, err := NewRollingAppendFile(config)
	if err != nil {
		t.Fatalf("Failed to create RollingAppendFile: %v", err)
	}
	defer cleanup(raf)

	// Create aligned buffer
	data := createAlignedBuffer(4096, 4096)
	for i := range data {
		data[i] = byte(i % 256)
	}

	offset, err := raf.Pwrite(data)
	if err != nil {
		t.Fatalf("Pwrite failed: %v", err)
	}

	if offset != int64(len(data)) {
		t.Errorf("Expected offset %d, got %d", len(data), offset)
	}

	if raf.CurrentPhysicalOffset != int64(len(data)) {
		t.Errorf("Expected CurrentPhysicalOffset %d, got %d", len(data), raf.CurrentPhysicalOffset)
	}

	if raf.Stat.WriteCount != 1 {
		t.Errorf("Expected WriteCount 1, got %d", raf.Stat.WriteCount)
	}
}

func TestPwrite_FileSizeExceeded(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_rolling_file.dat")

	config := FileConfig{
		Filename:          filename,
		MaxFileSize:       1024, // Small max size
		FilePunchHoleSize: 512,
		BlockSize:         4096,
	}

	raf, err := NewRollingAppendFile(config)
	if err != nil {
		t.Fatalf("Failed to create RollingAppendFile: %v", err)
	}
	defer cleanup(raf)

	// Try to write more than max file size
	data := make([]byte, 2048)

	_, err = raf.Pwrite(data)
	if err != ErrFileSizeExceeded {
		t.Errorf("Expected ErrFileSizeExceeded, got %v", err)
	}
}

func TestPwrite_BufferNotAligned(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_rolling_file.dat")

	config := FileConfig{
		Filename:          filename,
		MaxFileSize:       1024 * 1024,
		FilePunchHoleSize: 64 * 1024,
		BlockSize:         4096,
	}

	raf, err := NewRollingAppendFile(config)
	if err != nil {
		t.Fatalf("Failed to create RollingAppendFile: %v", err)
	}
	defer cleanup(raf)

	// Only test if using DirectIO
	if raf.WriteDirectIO {
		// Create unaligned buffer
		data := make([]byte, 4097) // Not aligned to 4096

		_, err = raf.Pwrite(data)
		if err != ErrBufNoAlign {
			t.Errorf("Expected ErrBufNoAlign, got %v", err)
		}
	}
}

func TestPread_Success(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_rolling_file.dat")

	config := FileConfig{
		Filename:          filename,
		MaxFileSize:       1024 * 1024,
		FilePunchHoleSize: 64 * 1024,
		BlockSize:         4096,
	}

	raf, err := NewRollingAppendFile(config)
	if err != nil {
		t.Fatalf("Failed to create RollingAppendFile: %v", err)
	}
	defer cleanup(raf)

	// Write some data first
	writeData := createAlignedBuffer(4096, 4096)
	for i := range writeData {
		writeData[i] = byte(i % 256)
	}

	_, err = raf.Pwrite(writeData)
	if err != nil {
		t.Fatalf("Pwrite failed: %v", err)
	}

	// Read the data back
	readData := createAlignedBuffer(4096, 4096)
	n, err := raf.Pread(0, readData)
	if err != nil {
		t.Fatalf("Pread failed: %v", err)
	}

	if n != int32(len(readData)) {
		t.Errorf("Expected read length %d, got %d", len(readData), n)
	}

	// Verify data matches
	for i := range readData {
		if readData[i] != writeData[i] {
			t.Errorf("Data mismatch at index %d: expected %d, got %d", i, writeData[i], readData[i])
		}
	}

	if raf.Stat.ReadCount != 1 {
		t.Errorf("Expected ReadCount 1, got %d", raf.Stat.ReadCount)
	}
}

func TestPread_FileOffsetOutOfRange(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_rolling_file.dat")

	config := FileConfig{
		Filename:          filename,
		MaxFileSize:       1024 * 1024,
		FilePunchHoleSize: 64 * 1024,
		BlockSize:         4096,
	}

	raf, err := NewRollingAppendFile(config)
	if err != nil {
		t.Fatalf("Failed to create RollingAppendFile: %v", err)
	}
	defer cleanup(raf)

	// Try to read without writing anything
	readData := createAlignedBuffer(4096, 4096)
	_, err = raf.Pread(0, readData)
	if err != ErrFileOffsetOutOfRange {
		t.Errorf("Expected ErrFileOffsetOutOfRange, got %v", err)
	}

	// Write some data
	writeData := createAlignedBuffer(4096, 4096)
	_, err = raf.Pwrite(writeData)
	if err != nil {
		t.Fatalf("Pwrite failed: %v", err)
	}

	// Try to read beyond written data
	_, err = raf.Pread(4096, readData)
	if err != ErrFileOffsetOutOfRange {
		t.Errorf("Expected ErrFileOffsetOutOfRange, got %v", err)
	}
}

func TestPread_OffsetNotAligned(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_rolling_file.dat")

	config := FileConfig{
		Filename:          filename,
		MaxFileSize:       1024 * 1024,
		FilePunchHoleSize: 64 * 1024,
		BlockSize:         4096,
	}

	raf, err := NewRollingAppendFile(config)
	if err != nil {
		t.Fatalf("Failed to create RollingAppendFile: %v", err)
	}
	defer cleanup(raf)

	// Only test if using DirectIO
	if raf.ReadDirectIO {
		// Write some data first
		writeData := createAlignedBuffer(8192, 4096)
		_, err = raf.Pwrite(writeData)
		if err != nil {
			t.Fatalf("Pwrite failed: %v", err)
		}

		// Try to read from unaligned offset
		readData := createAlignedBuffer(4096, 4096)
		_, err = raf.Pread(100, readData) // Not aligned to 4096
		if err != ErrOffsetNotAligned {
			t.Errorf("Expected ErrOffsetNotAligned, got %v", err)
		}
	}
}

func TestTrimHead_Success(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_rolling_file.dat")

	config := FileConfig{
		Filename:          filename,
		MaxFileSize:       1024 * 1024,
		FilePunchHoleSize: 4096, // One block
		BlockSize:         4096,
	}

	raf, err := NewRollingAppendFile(config)
	if err != nil {
		t.Fatalf("Failed to create RollingAppendFile: %v", err)
	}
	defer cleanup(raf)

	// Write some data first
	writeData := createAlignedBuffer(8192, 4096) // 2 blocks
	_, err = raf.Pwrite(writeData)
	if err != nil {
		t.Fatalf("Pwrite failed: %v", err)
	}

	// Trim head
	err = raf.TrimHead()
	if err != nil {
		t.Fatalf("TrimHead failed: %v", err)
	}

	// Verify state changes
	if raf.LogicalStartOffset != int64(config.FilePunchHoleSize) {
		t.Errorf("Expected LogicalStartOffset %d, got %d", config.FilePunchHoleSize, raf.LogicalStartOffset)
	}

	if raf.Stat.PunchHoleCount != 1 {
		t.Errorf("Expected PunchHoleCount 1, got %d", raf.Stat.PunchHoleCount)
	}
}

func TestIsAlignedOffset(t *testing.T) {
	tests := []struct {
		name      string
		offset    int64
		alignment int
		expected  bool
	}{
		{"aligned_0", 0, 4096, true},
		{"aligned_4096", 4096, 4096, true},
		{"aligned_8192", 8192, 4096, true},
		{"unaligned_100", 100, 4096, false},
		{"unaligned_4097", 4097, 4096, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAlignedOffset(tt.offset, tt.alignment)
			if result != tt.expected {
				t.Errorf("isAlignedOffset(%d, %d) = %v, expected %v", tt.offset, tt.alignment, result, tt.expected)
			}
		})
	}
}

func TestMultipleOperations(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_rolling_file.dat")

	config := FileConfig{
		Filename:          filename,
		MaxFileSize:       1024 * 1024,
		FilePunchHoleSize: 4096,
		BlockSize:         4096,
	}

	raf, err := NewRollingAppendFile(config)
	if err != nil {
		t.Fatalf("Failed to create RollingAppendFile: %v", err)
	}
	defer cleanup(raf)

	// Write multiple blocks
	for i := 0; i < 5; i++ {
		data := createAlignedBuffer(4096, 4096)
		for j := range data {
			data[j] = byte((i*256 + j) % 256)
		}

		_, err = raf.Pwrite(data)
		if err != nil {
			t.Fatalf("Pwrite %d failed: %v", i, err)
		}
	}

	// Verify total written
	expectedPhysicalOffset := int64(5 * 4096)
	if raf.CurrentPhysicalOffset != expectedPhysicalOffset {
		t.Errorf("Expected CurrentPhysicalOffset %d, got %d", expectedPhysicalOffset, raf.CurrentPhysicalOffset)
	}

	// Read back data from different offsets
	for i := 0; i < 5; i++ {
		readData := createAlignedBuffer(4096, 4096)
		_, err = raf.Pread(int64(i*4096), readData)
		if err != nil {
			t.Fatalf("Pread %d failed: %v", i, err)
		}

		// Verify data integrity
		for j := range readData {
			expected := byte((i*256 + j) % 256)
			if readData[j] != expected {
				t.Errorf("Data mismatch at block %d, index %d: expected %d, got %d", i, j, expected, readData[j])
			}
		}
	}

	// Verify statistics
	if raf.Stat.WriteCount != 5 {
		t.Errorf("Expected WriteCount 5, got %d", raf.Stat.WriteCount)
	}
	if raf.Stat.ReadCount != 5 {
		t.Errorf("Expected ReadCount 5, got %d", raf.Stat.ReadCount)
	}
}

func TestStatistics(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_rolling_file.dat")

	config := FileConfig{
		Filename:          filename,
		MaxFileSize:       1024 * 1024,
		FilePunchHoleSize: 4096,
		BlockSize:         4096,
	}

	raf, err := NewRollingAppendFile(config)
	if err != nil {
		t.Fatalf("Failed to create RollingAppendFile: %v", err)
	}
	defer cleanup(raf)

	// Initial state
	if raf.Stat.WriteCount != 0 {
		t.Errorf("Expected initial WriteCount 0, got %d", raf.Stat.WriteCount)
	}
	if raf.Stat.ReadCount != 0 {
		t.Errorf("Expected initial ReadCount 0, got %d", raf.Stat.ReadCount)
	}
	if raf.Stat.PunchHoleCount != 0 {
		t.Errorf("Expected initial PunchHoleCount 0, got %d", raf.Stat.PunchHoleCount)
	}

	// Perform operations and verify statistics
	data := createAlignedBuffer(4096, 4096)

	// Write operation
	_, err = raf.Pwrite(data)
	if err != nil {
		t.Fatalf("Pwrite failed: %v", err)
	}
	if raf.Stat.WriteCount != 1 {
		t.Errorf("Expected WriteCount 1, got %d", raf.Stat.WriteCount)
	}

	// Read operation
	_, err = raf.Pread(0, data)
	if err != nil {
		t.Fatalf("Pread failed: %v", err)
	}
	if raf.Stat.ReadCount != 1 {
		t.Errorf("Expected ReadCount 1, got %d", raf.Stat.ReadCount)
	}

	// Trim operation
	err = raf.TrimHead()
	if err != nil {
		t.Fatalf("TrimHead failed: %v", err)
	}
	if raf.Stat.PunchHoleCount != 1 {
		t.Errorf("Expected PunchHoleCount 1, got %d", raf.Stat.PunchHoleCount)
	}
}

// Helper function to clean up resources
func cleanup(raf *RollingAppendFile) {
	if raf.WriteFile != nil {
		raf.WriteFile.Close()
	}
	if raf.ReadFile != nil {
		raf.ReadFile.Close()
	}
	if raf.WriteFile != nil {
		os.Remove(raf.WriteFile.Name())
	}
}
