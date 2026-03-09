//go:build linux
// +build linux

package fs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewWrapAppendFile(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_wrap_file.dat")

	config := FileConfig{
		Filename:          filename,
		MaxFileSize:       1024 * 1024, // 1MB
		FilePunchHoleSize: 64 * 1024,   // 64KB
		BlockSize:         4096,
	}

	waf, err := NewWrapAppendFile(config)
	if err != nil {
		t.Fatalf("Failed to create WrapAppendFile: %v", err)
	}
	defer cleanupWrapFile(waf)

	// Verify initial state
	if waf.MaxFileSize != config.MaxFileSize {
		t.Errorf("Expected MaxFileSize %d, got %d", config.MaxFileSize, waf.MaxFileSize)
	}
	if waf.FilePunchHoleSize != config.FilePunchHoleSize {
		t.Errorf("Expected FilePunchHoleSize %d, got %d", config.FilePunchHoleSize, waf.FilePunchHoleSize)
	}
	if waf.blockSize != config.BlockSize {
		t.Errorf("Expected BlockSize %d, got %d", config.BlockSize, waf.blockSize)
	}
	if waf.LogicalCurrentOffset != 0 {
		t.Errorf("Expected LogicalCurrentOffset 0, got %d", waf.LogicalCurrentOffset)
	}
	if waf.PhysicalWriteOffset != 0 {
		t.Errorf("Expected PhysicalWriteOffset 0, got %d", waf.PhysicalWriteOffset)
	}
	if waf.PhysicalStartOffset != 0 {
		t.Errorf("Expected PhysicalStartOffset 0, got %d", waf.PhysicalStartOffset)
	}
	if waf.wrapped {
		t.Errorf("Expected wrapped to be false initially")
	}
}

func TestNewWrapAppendFile_DefaultBlockSize(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_wrap_file.dat")

	config := FileConfig{
		Filename:          filename,
		MaxFileSize:       1024 * 1024,
		FilePunchHoleSize: 64 * 1024,
		BlockSize:         0, // Should default to BLOCK_SIZE
	}

	waf, err := NewWrapAppendFile(config)
	if err != nil {
		t.Fatalf("Failed to create WrapAppendFile: %v", err)
	}
	defer cleanupWrapFile(waf)

	if waf.blockSize != BLOCK_SIZE {
		t.Errorf("Expected default BlockSize %d, got %d", BLOCK_SIZE, waf.blockSize)
	}
}

func TestWrapAppendFile_Pwrite_Success(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_wrap_file.dat")

	config := FileConfig{
		Filename:          filename,
		MaxFileSize:       1024 * 1024,
		FilePunchHoleSize: 64 * 1024,
		BlockSize:         4096,
	}

	waf, err := NewWrapAppendFile(config)
	if err != nil {
		t.Fatalf("Failed to create WrapAppendFile: %v", err)
	}
	defer cleanupWrapFile(waf)

	// Create aligned buffer
	data := createAlignedBuffer(4096, 4096)
	for i := range data {
		data[i] = byte(i % 256)
	}

	offset, err := waf.Pwrite(data)
	if err != nil {
		t.Fatalf("Pwrite failed: %v", err)
	}

	if offset != int64(len(data)) {
		t.Errorf("Expected offset %d, got %d", len(data), offset)
	}

	if waf.PhysicalWriteOffset != int64(len(data)) {
		t.Errorf("Expected PhysicalWriteOffset %d, got %d", len(data), waf.PhysicalWriteOffset)
	}

	if waf.LogicalCurrentOffset != int64(len(data)) {
		t.Errorf("Expected LogicalCurrentOffset %d, got %d", len(data), waf.LogicalCurrentOffset)
	}

	if waf.Stat.WriteCount.Load() != 1 {
		t.Errorf("Expected WriteCount 1, got %d", waf.Stat.WriteCount.Load())
	}

	if waf.wrapped {
		t.Errorf("Expected wrapped to be false")
	}
}

func TestPwrite_WrapAround(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_wrap_file.dat")

	config := FileConfig{
		Filename:          filename,
		MaxFileSize:       8192, // Small max size for easy wrapping
		FilePunchHoleSize: 4096,
		BlockSize:         4096,
	}

	waf, err := NewWrapAppendFile(config)
	if err != nil {
		t.Fatalf("Failed to create WrapAppendFile: %v", err)
	}
	defer cleanupWrapFile(waf)

	// Write first block
	data1 := createAlignedBuffer(4096, 4096)
	for i := range data1 {
		data1[i] = byte(1)
	}

	_, err = waf.Pwrite(data1)
	if err != nil {
		t.Fatalf("First Pwrite failed: %v", err)
	}

	if waf.wrapped {
		t.Errorf("Should not be wrapped after first write")
	}

	// Write second block - should trigger wrap
	data2 := createAlignedBuffer(4096, 4096)
	for i := range data2 {
		data2[i] = byte(2)
	}

	offset, err := waf.Pwrite(data2)
	if err != nil {
		t.Fatalf("Second Pwrite failed: %v", err)
	}

	// After wrapping, should be at PhysicalStartOffset
	if !waf.wrapped {
		t.Errorf("Should be wrapped after exceeding MaxFileSize")
	}

	if waf.PhysicalWriteOffset != waf.PhysicalStartOffset {
		t.Errorf("Expected PhysicalWriteOffset %d after wrap, got %d", waf.PhysicalStartOffset, waf.PhysicalWriteOffset)
	}

	if offset != waf.PhysicalStartOffset {
		t.Errorf("Expected return offset %d after wrap, got %d", waf.PhysicalStartOffset, offset)
	}

	if waf.LogicalCurrentOffset != int64(8192) {
		t.Errorf("Expected LogicalCurrentOffset %d, got %d", 8192, waf.LogicalCurrentOffset)
	}
}

func TestWrapAppendFile_Pwrite_BufferNotAligned(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_wrap_file.dat")

	config := FileConfig{
		Filename:          filename,
		MaxFileSize:       1024 * 1024,
		FilePunchHoleSize: 64 * 1024,
		BlockSize:         4096,
	}

	waf, err := NewWrapAppendFile(config)
	if err != nil {
		t.Fatalf("Failed to create WrapAppendFile: %v", err)
	}
	defer cleanupWrapFile(waf)

	// Only test if using DirectIO
	if waf.WriteDirectIO {
		// Create unaligned buffer
		data := make([]byte, 4097) // Not aligned to 4096

		_, err = waf.Pwrite(data)
		if err != ErrBufNoAlign {
			t.Errorf("Expected ErrBufNoAlign, got %v", err)
		}
	}
}

func TestPread_Success_NoWrap(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_wrap_file.dat")

	config := FileConfig{
		Filename:          filename,
		MaxFileSize:       1024 * 1024,
		FilePunchHoleSize: 64 * 1024,
		BlockSize:         4096,
	}

	waf, err := NewWrapAppendFile(config)
	if err != nil {
		t.Fatalf("Failed to create WrapAppendFile: %v", err)
	}
	defer cleanupWrapFile(waf)

	// Write some data first
	writeData := createAlignedBuffer(4096, 4096)
	for i := range writeData {
		writeData[i] = byte(i % 256)
	}

	_, err = waf.Pwrite(writeData)
	if err != nil {
		t.Fatalf("Pwrite failed: %v", err)
	}

	// Read the data back
	readData := createAlignedBuffer(4096, 4096)
	n, err := waf.Pread(0, readData)
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

	if waf.Stat.ReadCount.Load() != 1 {
		t.Errorf("Expected ReadCount 1, got %d", waf.Stat.ReadCount.Load())
	}
}

func TestPread_Success_WithWrap(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_wrap_file.dat")

	config := FileConfig{
		Filename:          filename,
		MaxFileSize:       8192, // Small for easy wrapping
		FilePunchHoleSize: 4096,
		BlockSize:         4096,
	}

	waf, err := NewWrapAppendFile(config)
	if err != nil {
		t.Fatalf("Failed to create WrapAppendFile: %v", err)
	}
	defer cleanupWrapFile(waf)

	// Fill the file to cause wrapping
	data1 := createAlignedBuffer(4096, 4096)
	for i := range data1 {
		data1[i] = byte(1)
	}
	_, err = waf.Pwrite(data1)
	if err != nil {
		t.Fatalf("First Pwrite failed: %v", err)
	}

	data2 := createAlignedBuffer(4096, 4096)
	for i := range data2 {
		data2[i] = byte(2)
	}
	_, err = waf.Pwrite(data2)
	if err != nil {
		t.Fatalf("Second Pwrite failed: %v", err)
	}

	// Now write more to wrap around
	data3 := createAlignedBuffer(4096, 4096)
	for i := range data3 {
		data3[i] = byte(3)
	}
	_, err = waf.Pwrite(data3)
	if err != nil {
		t.Fatalf("Third Pwrite failed: %v", err)
	}

	if !waf.wrapped {
		t.Errorf("Expected wrapped to be true")
	}

	// Read from valid regions after wrap
	// Region 1: [PhysicalStartOffset, MaxFileSize) - should contain data2
	readData := createAlignedBuffer(4096, 4096)
	n, err := waf.Pread(4096, readData)
	if err != nil {
		t.Fatalf("Pread from high region failed: %v", err)
	}
	if n != 4096 {
		t.Errorf("Expected read length 4096, got %d", n)
	}

	// Region 2: [0, PhysicalWriteOffset) - should contain data3
	readData2 := createAlignedBuffer(4096, 4096)
	n, err = waf.Pread(0, readData2)
	if err != nil {
		t.Fatalf("Pread from low region failed: %v", err)
	}
	if n != 4096 {
		t.Errorf("Expected read length 4096, got %d", n)
	}

	// Verify data3 in wrapped position
	for i := range readData2 {
		if readData2[i] != byte(3) {
			t.Errorf("Data mismatch in wrapped region at index %d: expected %d, got %d", i, 3, readData2[i])
		}
	}
}

func TestPread_FileOffsetOutOfRange_NoWrap(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_wrap_file.dat")

	config := FileConfig{
		Filename:          filename,
		MaxFileSize:       1024 * 1024,
		FilePunchHoleSize: 64 * 1024,
		BlockSize:         4096,
	}

	waf, err := NewWrapAppendFile(config)
	if err != nil {
		t.Fatalf("Failed to create WrapAppendFile: %v", err)
	}
	defer cleanupWrapFile(waf)

	// Try to read without writing anything
	readData := createAlignedBuffer(4096, 4096)
	_, err = waf.Pread(0, readData)
	if err != ErrFileOffsetOutOfRange {
		t.Errorf("Expected ErrFileOffsetOutOfRange, got %v", err)
	}

	// Write some data
	writeData := createAlignedBuffer(4096, 4096)
	_, err = waf.Pwrite(writeData)
	if err != nil {
		t.Fatalf("Pwrite failed: %v", err)
	}

	// Try to read beyond written data
	_, err = waf.Pread(4096, readData)
	if err != ErrFileOffsetOutOfRange {
		t.Errorf("Expected ErrFileOffsetOutOfRange, got %v", err)
	}
}

func TestPread_FileOffsetOutOfRange_WithWrap(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_wrap_file.dat")

	config := FileConfig{
		Filename:          filename,
		MaxFileSize:       8192,
		FilePunchHoleSize: 4096,
		BlockSize:         4096,
	}

	waf, err := NewWrapAppendFile(config)
	if err != nil {
		t.Fatalf("Failed to create WrapAppendFile: %v", err)
	}
	defer cleanupWrapFile(waf)

	// Write 2 blocks to cause wrapping (MaxFileSize=8192, block=4096)
	for i := 0; i < 2; i++ {
		data := createAlignedBuffer(4096, 4096)
		_, err = waf.Pwrite(data)
		if err != nil {
			t.Fatalf("Pwrite %d failed: %v", i, err)
		}
	}

	if !waf.wrapped {
		t.Errorf("Expected wrapped to be true")
	}

	// After 2 writes: PhysicalStartOffset=0, PhysicalWriteOffset=0 (wrapped).
	// TrimHead to advance PhysicalStartOffset to 4096, creating a gap at [0, 4096).
	err = waf.TrimHead()
	if err != nil {
		t.Fatalf("TrimHead failed: %v", err)
	}

	// State: PhysicalStartOffset=4096, PhysicalWriteOffset=0, wrapped=true
	// Valid regions: [4096, 8192) and [0, 0) (empty).
	// Gap: [0, 4096) — reading from offset 0 should fail.
	readData := createAlignedBuffer(4096, 4096)
	_, err = waf.Pread(0, readData)
	if err != ErrFileOffsetOutOfRange {
		t.Errorf("Expected ErrFileOffsetOutOfRange for gap read, got %v", err)
	}
}

func TestWrapAppendFile_Pread_OffsetNotAligned(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_wrap_file.dat")

	config := FileConfig{
		Filename:          filename,
		MaxFileSize:       1024 * 1024,
		FilePunchHoleSize: 64 * 1024,
		BlockSize:         4096,
	}

	waf, err := NewWrapAppendFile(config)
	if err != nil {
		t.Fatalf("Failed to create WrapAppendFile: %v", err)
	}
	defer cleanupWrapFile(waf)

	// Only test if using DirectIO
	if waf.ReadDirectIO {
		// Write some data first
		writeData := createAlignedBuffer(8192, 4096)
		_, err = waf.Pwrite(writeData)
		if err != nil {
			t.Fatalf("Pwrite failed: %v", err)
		}

		// Try to read from unaligned offset
		readData := createAlignedBuffer(4096, 4096)
		_, err = waf.Pread(100, readData) // Not aligned to 4096
		if err != ErrOffsetNotAligned {
			t.Errorf("Expected ErrOffsetNotAligned, got %v", err)
		}
	}
}

func TestPread_BufferNotAligned(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_wrap_file.dat")

	config := FileConfig{
		Filename:          filename,
		MaxFileSize:       1024 * 1024,
		FilePunchHoleSize: 64 * 1024,
		BlockSize:         4096,
	}

	waf, err := NewWrapAppendFile(config)
	if err != nil {
		t.Fatalf("Failed to create WrapAppendFile: %v", err)
	}
	defer cleanupWrapFile(waf)

	// Only test if using DirectIO
	if waf.ReadDirectIO {
		// Write some data first
		writeData := createAlignedBuffer(4096, 4096)
		_, err = waf.Pwrite(writeData)
		if err != nil {
			t.Fatalf("Pwrite failed: %v", err)
		}

		// Try to read with unaligned buffer
		readData := make([]byte, 4097) // Not aligned
		_, err = waf.Pread(0, readData)
		if err != ErrBufNoAlign {
			t.Errorf("Expected ErrBufNoAlign, got %v", err)
		}
	}
}

func TestWrapAppendFile_TrimHead_Success(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_wrap_file.dat")

	config := FileConfig{
		Filename:          filename,
		MaxFileSize:       1024 * 1024,
		FilePunchHoleSize: 4096, // One block
		BlockSize:         4096,
	}

	waf, err := NewWrapAppendFile(config)
	if err != nil {
		t.Fatalf("Failed to create WrapAppendFile: %v", err)
	}
	defer cleanupWrapFile(waf)

	// Write some data first
	writeData := createAlignedBuffer(8192, 4096) // 2 blocks
	_, err = waf.Pwrite(writeData)
	if err != nil {
		t.Fatalf("Pwrite failed: %v", err)
	}

	initialStartOffset := waf.PhysicalStartOffset

	// Trim head
	err = waf.TrimHead()
	if err != nil {
		t.Fatalf("TrimHead failed: %v", err)
	}

	// Verify state changes
	expectedStartOffset := initialStartOffset + int64(config.FilePunchHoleSize)
	if waf.PhysicalStartOffset != expectedStartOffset {
		t.Errorf("Expected PhysicalStartOffset %d, got %d", expectedStartOffset, waf.PhysicalStartOffset)
	}

	if waf.Stat.PunchHoleCount.Load() != 1 {
		t.Errorf("Expected PunchHoleCount 1, got %d", waf.Stat.PunchHoleCount.Load())
	}
}

func TestTrimHead_WrapAround(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_wrap_file.dat")

	config := FileConfig{
		Filename:          filename,
		MaxFileSize:       8192,
		FilePunchHoleSize: 8192, // Same as max file size
		BlockSize:         4096,
	}

	waf, err := NewWrapAppendFile(config)
	if err != nil {
		t.Fatalf("Failed to create WrapAppendFile: %v", err)
	}
	defer cleanupWrapFile(waf)

	// Set PhysicalStartOffset to near end
	waf.PhysicalStartOffset = 4096

	// Trim head - should wrap around to 0
	err = waf.TrimHead()
	if err != nil {
		t.Fatalf("TrimHead failed: %v", err)
	}

	// Should wrap to 0 since 4096 + 8192 >= 8192
	if waf.PhysicalStartOffset != 0 {
		t.Errorf("Expected PhysicalStartOffset to wrap to 0, got %d", waf.PhysicalStartOffset)
	}
}

func TestTrimHead_OffsetNotAligned(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_wrap_file.dat")

	config := FileConfig{
		Filename:          filename,
		MaxFileSize:       1024 * 1024,
		FilePunchHoleSize: 4096,
		BlockSize:         4096,
	}

	waf, err := NewWrapAppendFile(config)
	if err != nil {
		t.Fatalf("Failed to create WrapAppendFile: %v", err)
	}
	defer cleanupWrapFile(waf)

	// Only test if using DirectIO
	if waf.WriteDirectIO {
		// Set unaligned PhysicalStartOffset
		waf.PhysicalStartOffset = 100

		err = waf.TrimHead()
		if err != ErrOffsetNotAligned {
			t.Errorf("Expected ErrOffsetNotAligned, got %v", err)
		}
	}
}

func TestPwrite_WrapAndContinue(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_wrap_file.dat")

	config := FileConfig{
		Filename:          filename,
		MaxFileSize:       8192,
		FilePunchHoleSize: 4096,
		BlockSize:         4096,
	}

	waf, err := NewWrapAppendFile(config)
	if err != nil {
		t.Fatalf("Failed to create WrapAppendFile: %v", err)
	}
	defer cleanupWrapFile(waf)

	for i := 0; i < 2; i++ {
		data := createAlignedBuffer(4096, 4096)
		_, err = waf.Pwrite(data)
		if err != nil {
			t.Fatalf("Pwrite %d failed: %v", i, err)
		}
	}

	if !waf.wrapped {
		t.Errorf("Expected wrapped to be true")
	}

	// After wrapping, PhysicalWriteOffset resets to PhysicalStartOffset.
	// A subsequent write overwrites at that position (trim is done at
	// the shard level via DeleteManager, not by Pwrite itself).
	data := createAlignedBuffer(4096, 4096)
	for i := range data {
		data[i] = 0xAB
	}
	_, err = waf.Pwrite(data)
	if err != nil {
		t.Fatalf("Post-wrap Pwrite failed: %v", err)
	}

	if waf.Stat.WriteCount.Load() != 3 {
		t.Errorf("Expected WriteCount 3, got %d", waf.Stat.WriteCount.Load())
	}

	if waf.LogicalCurrentOffset != int64(3*4096) {
		t.Errorf("Expected LogicalCurrentOffset %d, got %d", 3*4096, waf.LogicalCurrentOffset)
	}
}

func TestClose(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_wrap_file.dat")

	config := FileConfig{
		Filename:          filename,
		MaxFileSize:       1024 * 1024,
		FilePunchHoleSize: 64 * 1024,
		BlockSize:         4096,
	}

	waf, err := NewWrapAppendFile(config)
	if err != nil {
		t.Fatalf("Failed to create WrapAppendFile: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Errorf("File should exist before Close")
	}

	// Close and verify cleanup
	waf.Close()

	// File should be removed
	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		t.Errorf("File should be removed after Close")
	}
}

func TestWrapAppendFile_MultipleOperations(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_wrap_file.dat")

	config := FileConfig{
		Filename:          filename,
		MaxFileSize:       16384, // 4 blocks
		FilePunchHoleSize: 4096,
		BlockSize:         4096,
	}

	waf, err := NewWrapAppendFile(config)
	if err != nil {
		t.Fatalf("Failed to create WrapAppendFile: %v", err)
	}
	defer cleanupWrapFile(waf)

	// Write multiple blocks to test wrap behavior
	for i := 0; i < 6; i++ { // More than max file size / block size
		data := createAlignedBuffer(4096, 4096)
		for j := range data {
			data[j] = byte((i*256 + j) % 256)
		}

		_, err = waf.Pwrite(data)
		if err != nil {
			t.Fatalf("Pwrite %d failed: %v", i, err)
		}
	}

	// Should be wrapped
	if !waf.wrapped {
		t.Errorf("Expected wrapped to be true after writing 6 blocks")
	}

	// Verify logical offset continues to grow
	expectedLogicalOffset := int64(6 * 4096)
	if waf.LogicalCurrentOffset != expectedLogicalOffset {
		t.Errorf("Expected LogicalCurrentOffset %d, got %d", expectedLogicalOffset, waf.LogicalCurrentOffset)
	}

	// Verify statistics
	if waf.Stat.WriteCount.Load() != 6 {
		t.Errorf("Expected WriteCount 6, got %d", waf.Stat.WriteCount.Load())
	}
}

func TestWrapAppendFile_Statistics(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_wrap_file.dat")

	config := FileConfig{
		Filename:          filename,
		MaxFileSize:       1024 * 1024,
		FilePunchHoleSize: 4096,
		BlockSize:         4096,
	}

	waf, err := NewWrapAppendFile(config)
	if err != nil {
		t.Fatalf("Failed to create WrapAppendFile: %v", err)
	}
	defer cleanupWrapFile(waf)

	// Initial state
	if waf.Stat.WriteCount.Load() != 0 {
		t.Errorf("Expected initial WriteCount 0, got %d", waf.Stat.WriteCount.Load())
	}
	if waf.Stat.ReadCount.Load() != 0 {
		t.Errorf("Expected initial ReadCount 0, got %d", waf.Stat.ReadCount.Load())
	}
	if waf.Stat.PunchHoleCount.Load() != 0 {
		t.Errorf("Expected initial PunchHoleCount 0, got %d", waf.Stat.PunchHoleCount.Load())
	}

	// Perform operations and verify statistics
	data := createAlignedBuffer(4096, 4096)

	// Write operation
	_, err = waf.Pwrite(data)
	if err != nil {
		t.Fatalf("Pwrite failed: %v", err)
	}
	if waf.Stat.WriteCount.Load() != 1 {
		t.Errorf("Expected WriteCount 1, got %d", waf.Stat.WriteCount.Load())
	}

	// Read operation
	_, err = waf.Pread(0, data)
	if err != nil {
		t.Fatalf("Pread failed: %v", err)
	}
	if waf.Stat.ReadCount.Load() != 1 {
		t.Errorf("Expected ReadCount 1, got %d", waf.Stat.ReadCount.Load())
	}

	// Trim operation
	err = waf.TrimHead()
	if err != nil {
		t.Fatalf("TrimHead failed: %v", err)
	}
	if waf.Stat.PunchHoleCount.Load() != 1 {
		t.Errorf("Expected PunchHoleCount 1, got %d", waf.Stat.PunchHoleCount.Load())
	}
}

// Helper function to clean up resources for WrapAppendFile
func cleanupWrapFile(waf *WrapAppendFile) {
	if waf.WriteFile != nil {
		waf.WriteFile.Close()
	}
	if waf.ReadFile != nil {
		waf.ReadFile.Close()
	}
	if waf.WriteFile != nil {
		os.Remove(waf.WriteFile.Name())
	}
}
