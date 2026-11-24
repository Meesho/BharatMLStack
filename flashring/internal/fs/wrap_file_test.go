//go:build linux
// +build linux

package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Meesho/BharatMLStack/flashring/internal/iouring"
	"github.com/Meesho/BharatMLStack/flashring/internal/testutil/mountfs"
)

func TestNewWrapAppendFile(t *testing.T) {
	tmpDir := mountfs.New(t)
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
	if waf.LogicalCurrentOffset.Load() != 0 {
		t.Errorf("Expected LogicalCurrentOffset 0, got %d", waf.LogicalCurrentOffset.Load())
	}
	if waf.PhysicalWriteOffset.Load() != 0 {
		t.Errorf("Expected PhysicalWriteOffset 0, got %d", waf.PhysicalWriteOffset.Load())
	}
	if waf.PhysicalStartOffset.Load() != 0 {
		t.Errorf("Expected PhysicalStartOffset 0, got %d", waf.PhysicalStartOffset.Load())
	}
	if waf.wrapped.Load() {
		t.Errorf("Expected wrapped to be false initially")
	}
}

func TestNewWrapAppendFile_DefaultBlockSize(t *testing.T) {
	tmpDir := mountfs.New(t)
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
	tmpDir := mountfs.New(t)
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

	if waf.PhysicalWriteOffset.Load() != int64(len(data)) {
		t.Errorf("Expected PhysicalWriteOffset %d, got %d", len(data), waf.PhysicalWriteOffset.Load())
	}

	if waf.LogicalCurrentOffset.Load() != int64(len(data)) {
		t.Errorf("Expected LogicalCurrentOffset %d, got %d", len(data), waf.LogicalCurrentOffset.Load())
	}

	if waf.Stat.WriteCount.Load() != 1 {
		t.Errorf("Expected WriteCount 1, got %d", waf.Stat.WriteCount.Load())
	}

	if waf.wrapped.Load() {
		t.Errorf("Expected wrapped to be false")
	}
}

func TestPwrite_WrapAround(t *testing.T) {
	tmpDir := mountfs.New(t)
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

	if waf.wrapped.Load() {
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
	if !waf.wrapped.Load() {
		t.Errorf("Should be wrapped after exceeding MaxFileSize")
	}

	if waf.PhysicalWriteOffset.Load() != waf.PhysicalStartOffset.Load() {
		t.Errorf("Expected PhysicalWriteOffset %d after wrap, got %d", waf.PhysicalStartOffset.Load(), waf.PhysicalWriteOffset.Load())
	}

	if offset != waf.PhysicalStartOffset.Load() {
		t.Errorf("Expected return offset %d after wrap, got %d", waf.PhysicalStartOffset.Load(), offset)
	}

	if waf.LogicalCurrentOffset.Load() != int64(8192) {
		t.Errorf("Expected LogicalCurrentOffset %d, got %d", 8192, waf.LogicalCurrentOffset.Load())
	}
}

func TestWrapAppendFile_Pwrite_BufferNotAligned(t *testing.T) {
	tmpDir := mountfs.New(t)
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
		data := createAlignedBuffer(8192, 4096)[1:4097] // one byte off a block boundary

		_, err = waf.Pwrite(data)
		if err != ErrBufNoAlign {
			t.Errorf("Expected ErrBufNoAlign, got %v", err)
		}
	}
}

func TestPread_Success_NoWrap(t *testing.T) {
	tmpDir := mountfs.New(t)
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
	tmpDir := mountfs.New(t)
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

	// Fill the file: data1 at [0,4096), data2 at [4096,8192).
	data1 := createAlignedBuffer(4096, 4096)
	for i := range data1 {
		data1[i] = byte(1)
	}
	if _, err = waf.Pwrite(data1); err != nil {
		t.Fatalf("First Pwrite failed: %v", err)
	}

	data2 := createAlignedBuffer(4096, 4096)
	for i := range data2 {
		data2[i] = byte(2)
	}
	if _, err = waf.Pwrite(data2); err != nil {
		t.Fatalf("Second Pwrite failed: %v", err)
	}

	// Second Pwrite reaches MaxFileSize → writer wraps to 0.
	if !waf.wrapped.Load() {
		t.Fatalf("Expected wrapped to be true after filling file")
	}
	if waf.PhysicalWriteOffset.Load() != 0 {
		t.Fatalf("Expected PhysicalWriteOffset=0 after wrap, got %d", waf.PhysicalWriteOffset.Load())
	}

	// W == S == 0 (full ring): both regions are readable.
	readData := createAlignedBuffer(4096, 4096)
	if _, err = waf.Pread(0, readData); err != nil {
		t.Fatalf("Pread [0,4096) in full ring failed: %v", err)
	}
	if _, err = waf.Pread(4096, readData); err != nil {
		t.Fatalf("Pread [4096,8192) in full ring failed: %v", err)
	}

	// TrimHead: punch [0,4096), S advances to 4096.
	if err = waf.TrimHead(); err != nil {
		t.Fatalf("TrimHead failed: %v", err)
	}
	if waf.PhysicalStartOffset.Load() != 4096 {
		t.Fatalf("Expected PhysicalStartOffset=4096 after trim, got %d", waf.PhysicalStartOffset.Load())
	}

	// Now W(0) < S(4096). Valid region: [4096, 8192).
	// Punched region [0,4096) must be rejected.
	if _, err = waf.Pread(0, readData); err != ErrFileOffsetOutOfRange {
		t.Errorf("Pread from punched region should fail, got %v", err)
	}
	n, err := waf.Pread(4096, readData)
	if err != nil {
		t.Fatalf("Pread from old tail failed: %v", err)
	}
	if n != 4096 {
		t.Errorf("Expected read length 4096, got %d", n)
	}
	for i := range readData {
		if readData[i] != byte(2) {
			t.Errorf("Old tail data mismatch at %d: expected 2, got %d", i, readData[i])
			break
		}
	}

	// Write data3 into the freed region [0,4096). W advances to 4096.
	data3 := createAlignedBuffer(4096, 4096)
	for i := range data3 {
		data3[i] = byte(3)
	}
	if _, err = waf.Pwrite(data3); err != nil {
		t.Fatalf("Third Pwrite failed: %v", err)
	}

	// W == S == 4096 (full ring again): both regions readable.
	readOld := createAlignedBuffer(4096, 4096)
	n, err = waf.Pread(4096, readOld)
	if err != nil {
		t.Fatalf("Pread old tail [4096,8192) after refill failed: %v", err)
	}
	if n != 4096 {
		t.Errorf("Expected 4096 bytes, got %d", n)
	}
	for i := range readOld {
		if readOld[i] != byte(2) {
			t.Errorf("Old tail mismatch at %d: expected 2, got %d", i, readOld[i])
			break
		}
	}

	readNew := createAlignedBuffer(4096, 4096)
	n, err = waf.Pread(0, readNew)
	if err != nil {
		t.Fatalf("Pread new data [0,4096) after refill failed: %v", err)
	}
	if n != 4096 {
		t.Errorf("Expected 4096 bytes, got %d", n)
	}
	for i := range readNew {
		if readNew[i] != byte(3) {
			t.Errorf("New data mismatch at %d: expected 3, got %d", i, readNew[i])
			break
		}
	}
}

func TestPread_FileOffsetOutOfRange_NoWrap(t *testing.T) {
	tmpDir := mountfs.New(t)
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
	tmpDir := mountfs.New(t)
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

	if !waf.wrapped.Load() {
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
	tmpDir := mountfs.New(t)
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
	tmpDir := mountfs.New(t)
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
		readData := createAlignedBuffer(8192, 4096)[1:4097] // one byte off a block boundary
		_, err = waf.Pread(0, readData)
		if err != ErrBufNoAlign {
			t.Errorf("Expected ErrBufNoAlign, got %v", err)
		}
	}
}

func TestWrapAppendFile_TrimHead_Success(t *testing.T) {
	tmpDir := mountfs.New(t)
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

	initialStartOffset := waf.PhysicalStartOffset.Load()

	// Trim head
	err = waf.TrimHead()
	if err != nil {
		t.Fatalf("TrimHead failed: %v", err)
	}

	// Verify state changes
	expectedStartOffset := initialStartOffset + int64(config.FilePunchHoleSize)
	if waf.PhysicalStartOffset.Load() != expectedStartOffset {
		t.Errorf("Expected PhysicalStartOffset %d, got %d", expectedStartOffset, waf.PhysicalStartOffset.Load())
	}

	if waf.Stat.PunchHoleCount.Load() != 1 {
		t.Errorf("Expected PunchHoleCount 1, got %d", waf.Stat.PunchHoleCount.Load())
	}
}

func TestTrimHead_WrapAround(t *testing.T) {
	tmpDir := mountfs.New(t)
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
	waf.PhysicalStartOffset.Store(4096)

	// Trim head - should wrap around to 0
	err = waf.TrimHead()
	if err != nil {
		t.Fatalf("TrimHead failed: %v", err)
	}

	// Should wrap to 0 since 4096 + 8192 >= 8192
	if waf.PhysicalStartOffset.Load() != 0 {
		t.Errorf("Expected PhysicalStartOffset to wrap to 0, got %d", waf.PhysicalStartOffset.Load())
	}
}

func TestTrimHead_OffsetNotAligned(t *testing.T) {
	tmpDir := mountfs.New(t)
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
		waf.PhysicalStartOffset.Store(100)

		err = waf.TrimHead()
		if err != ErrOffsetNotAligned {
			t.Errorf("Expected ErrOffsetNotAligned, got %v", err)
		}
	}
}

func TestPwrite_WrapAndContinue(t *testing.T) {
	tmpDir := mountfs.New(t)
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

	if !waf.wrapped.Load() {
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

	if waf.LogicalCurrentOffset.Load() != int64(3*4096) {
		t.Errorf("Expected LogicalCurrentOffset %d, got %d", 3*4096, waf.LogicalCurrentOffset.Load())
	}
}

func TestClose(t *testing.T) {
	tmpDir := mountfs.New(t)
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
	tmpDir := mountfs.New(t)
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
	if !waf.wrapped.Load() {
		t.Errorf("Expected wrapped to be true after writing 6 blocks")
	}

	// Verify logical offset continues to grow
	expectedLogicalOffset := int64(6 * 4096)
	if waf.LogicalCurrentOffset.Load() != expectedLogicalOffset {
		t.Errorf("Expected LogicalCurrentOffset %d, got %d", expectedLogicalOffset, waf.LogicalCurrentOffset.Load())
	}

	// Verify statistics
	if waf.Stat.WriteCount.Load() != 6 {
		t.Errorf("Expected WriteCount 6, got %d", waf.Stat.WriteCount.Load())
	}
}

func TestWrapAppendFile_Statistics(t *testing.T) {
	tmpDir := mountfs.New(t)
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

// newWrapFileWithRing builds a WrapAppendFile and attaches a tiny io_uring
// writer to it for PwriteBatch tests. The writer is cleaned up by the
// surrounding cleanupWrapFile helper.
//
// Uses mountfs.New(t) instead of t.TempDir() because these tests exercise the
// real O_DIRECT + fallocate path; the container's overlayfs rejects O_DIRECT
// and returns EINVAL from fallocate, which causes the io_uring writer
// completeLoop to hang on Close() (see wrap_file_test.go cleanup deadlock).
func newWrapFileWithRing(t *testing.T, maxFileSize int64) *WrapAppendFile {
	t.Helper()
	filename := filepath.Join(mountfs.New(t), "wrap_io.dat")
	waf, err := NewWrapAppendFile(FileConfig{
		Filename:          filename,
		MaxFileSize:       maxFileSize,
		FilePunchHoleSize: BLOCK_SIZE,
		BlockSize:         BLOCK_SIZE,
	})
	if err != nil {
		t.Fatalf("NewWrapAppendFile: %v", err)
	}
	w, err := iouring.NewIoUringWriter(8, 0)
	if err != nil {
		t.Fatalf("NewIoUringWriter: %v", err)
	}
	waf.WriteRing = w
	t.Cleanup(func() {
		w.Close()
		cleanupWrapFile(waf)
	})
	return waf
}

func TestPwriteBatch_SmallBuffer_RoundTrip(t *testing.T) {
	waf := newWrapFileWithRing(t, BLOCK_SIZE*8)
	buf := createAlignedBuffer(BLOCK_SIZE*2, BLOCK_SIZE)
	for i := range buf {
		buf[i] = byte(i % 256)
	}

	total, off, err := waf.PwriteBatch(buf, BLOCK_SIZE)
	if err != nil {
		t.Fatalf("PwriteBatch: %v", err)
	}
	if total != len(buf) {
		t.Fatalf("total = %d, want %d", total, len(buf))
	}
	if off != int64(len(buf)) {
		t.Fatalf("offset = %d, want %d", off, len(buf))
	}
	if waf.LogicalCurrentOffset.Load() != int64(len(buf)) {
		t.Fatalf("LogicalCurrentOffset = %d, want %d", waf.LogicalCurrentOffset.Load(), len(buf))
	}
	if waf.Stat.WriteCount.Load() != 2 {
		t.Fatalf("WriteCount = %d, want 2 (chunkSize=BLOCK_SIZE, len=2*BLOCK_SIZE)",
			waf.Stat.WriteCount.Load())
	}
}

func TestPwriteBatch_LargeBuffer_SpansMultipleSubmissions(t *testing.T) {
	waf := newWrapFileWithRing(t, BLOCK_SIZE*32)
	buf := createAlignedBuffer(BLOCK_SIZE*16, BLOCK_SIZE)

	total, _, err := waf.PwriteBatch(buf, BLOCK_SIZE)
	if err != nil {
		t.Fatalf("PwriteBatch: %v", err)
	}
	if total != len(buf) {
		t.Fatalf("total = %d, want %d", total, len(buf))
	}
}

func TestPwriteBatch_Wraps_PhysicalOffsetResets(t *testing.T) {
	waf := newWrapFileWithRing(t, BLOCK_SIZE*4)
	buf := createAlignedBuffer(BLOCK_SIZE*4, BLOCK_SIZE)

	_, _, err := waf.PwriteBatch(buf, BLOCK_SIZE)
	if err != nil {
		t.Fatalf("PwriteBatch: %v", err)
	}
	if !waf.wrapped.Load() {
		t.Fatal("wrapped not set after PwriteBatch reaches MaxFileSize")
	}
	if waf.PhysicalWriteOffset.Load() != 0 {
		t.Fatalf("PhysicalWriteOffset = %d, want 0 after wrap", waf.PhysicalWriteOffset.Load())
	}
}

func TestPwriteBatch_UnalignedBuffer_ReturnsError(t *testing.T) {
	waf := newWrapFileWithRing(t, BLOCK_SIZE*8)
	if !waf.WriteDirectIO {
		t.Skip("WriteDirectIO disabled (filesystem fallback); alignment check skipped")
	}
	// Force misalignment by slicing one byte into a larger backing array.
	backing := createAlignedBuffer(BLOCK_SIZE*2, BLOCK_SIZE)
	unaligned := backing[1 : BLOCK_SIZE+1]

	_, _, err := waf.PwriteBatch(unaligned, BLOCK_SIZE)
	if err != ErrBufNoAlign {
		t.Fatalf("PwriteBatch err = %v, want ErrBufNoAlign", err)
	}
}

func TestAdvanceWriteOffset_NoWrap(t *testing.T) {
	waf := newWrapFileWithRing(t, BLOCK_SIZE*8)

	waf.AdvanceWriteOffset(BLOCK_SIZE * 2)

	if waf.PhysicalWriteOffset.Load() != BLOCK_SIZE*2 {
		t.Fatalf("PhysicalWriteOffset = %d, want %d", waf.PhysicalWriteOffset.Load(), BLOCK_SIZE*2)
	}
	if waf.LogicalCurrentOffset.Load() != BLOCK_SIZE*2 {
		t.Fatalf("LogicalCurrentOffset = %d, want %d", waf.LogicalCurrentOffset.Load(), BLOCK_SIZE*2)
	}
	if waf.wrapped.Load() {
		t.Fatal("wrapped flipped without crossing MaxFileSize")
	}
}

func TestAdvanceWriteOffset_WrapsAtMaxFileSize(t *testing.T) {
	waf := newWrapFileWithRing(t, BLOCK_SIZE*4)

	waf.AdvanceWriteOffset(BLOCK_SIZE * 4)

	if !waf.wrapped.Load() {
		t.Fatal("wrapped not set after AdvanceWriteOffset >= MaxFileSize")
	}
	if waf.PhysicalWriteOffset.Load() != 0 {
		t.Fatalf("PhysicalWriteOffset = %d, want 0 after wrap", waf.PhysicalWriteOffset.Load())
	}
}

func TestTrimHeadIfNeeded_FreshFile_ReturnsFalse(t *testing.T) {
	waf := newWrapFileWithRing(t, BLOCK_SIZE*8)
	if waf.TrimHeadIfNeeded() {
		t.Fatal("TrimHeadIfNeeded = true on a fresh file")
	}
}

func TestTrimHeadIfNeeded_AfterWrapAndAlignment_ReturnsTrue(t *testing.T) {
	waf := newWrapFileWithRing(t, BLOCK_SIZE*4)
	// AdvanceWriteOffset(MaxFileSize) sets wrapped=true and PhysicalWriteOffset=0;
	// PhysicalStartOffset is already 0 → both pointers equal → trim needed.
	waf.AdvanceWriteOffset(BLOCK_SIZE * 4)

	if !waf.TrimHeadIfNeeded() {
		t.Fatal("TrimHeadIfNeeded = false when wrapped and write==start")
	}
}

func TestValidateReadOffset_Aligned_ReturnsPhysOffset(t *testing.T) {
	waf := newWrapFileWithRing(t, BLOCK_SIZE*8)
	buf := createAlignedBuffer(BLOCK_SIZE, BLOCK_SIZE)
	_, err := waf.Pwrite(buf)
	if err != nil {
		t.Fatalf("Pwrite: %v", err)
	}

	got, err := waf.ValidateReadOffset(0, BLOCK_SIZE)
	if err != nil {
		t.Fatalf("ValidateReadOffset: %v", err)
	}
	if got != 0 {
		t.Fatalf("phys offset = %d, want 0", got)
	}
}

func TestValidateReadOffset_OutOfRange_ReturnsError(t *testing.T) {
	waf := newWrapFileWithRing(t, BLOCK_SIZE*8)

	_, err := waf.ValidateReadOffset(0, BLOCK_SIZE)
	if err != ErrFileOffsetOutOfRange {
		t.Fatalf("err = %v, want ErrFileOffsetOutOfRange", err)
	}
}

func TestValidateReadOffset_UnalignedOffset_ReturnsError(t *testing.T) {
	waf := newWrapFileWithRing(t, BLOCK_SIZE*8)
	if !waf.ReadDirectIO {
		t.Skip("ReadDirectIO disabled")
	}

	_, err := waf.ValidateReadOffset(123, BLOCK_SIZE)
	if err != ErrOffsetNotAligned {
		t.Fatalf("err = %v, want ErrOffsetNotAligned", err)
	}
}

func TestValidateReadOffset_AfterWrap_AppliesModulo(t *testing.T) {
	waf := newWrapFileWithRing(t, BLOCK_SIZE*4)
	buf := createAlignedBuffer(BLOCK_SIZE*4, BLOCK_SIZE)
	_, err := waf.Pwrite(buf)
	if err != nil {
		t.Fatalf("Pwrite: %v", err)
	}
	if !waf.wrapped.Load() {
		t.Fatal("setup failed: not wrapped")
	}

	// Request offset BLOCK_SIZE*5 (logical) → phys = 5 % 4 = 1 block.
	got, err := waf.ValidateReadOffset(BLOCK_SIZE*5, BLOCK_SIZE)
	if err != nil {
		t.Fatalf("ValidateReadOffset: %v", err)
	}
	if got != BLOCK_SIZE {
		t.Fatalf("phys offset = %d, want %d", got, BLOCK_SIZE)
	}
}

func TestNewWrapAppendFile_BadPath_Errors(t *testing.T) {
	_, err := NewWrapAppendFile(FileConfig{
		Filename:          "/nonexistent/path/foo.dat",
		MaxFileSize:       BLOCK_SIZE,
		FilePunchHoleSize: BLOCK_SIZE,
		BlockSize:         BLOCK_SIZE,
	})
	if err == nil {
		t.Fatal("NewWrapAppendFile on bad path returned nil err")
	}
}

// TestNewWrapAppendFile_DirectoryPath_Errors covers the createReadFileDescriptor
// branch by passing a directory path: O_WRONLY against a directory fails
// before the read fd is opened.
func TestNewWrapAppendFile_DirectoryPath_Errors(t *testing.T) {
	dir := t.TempDir()
	_, err := NewWrapAppendFile(FileConfig{
		Filename:          dir,
		MaxFileSize:       BLOCK_SIZE,
		FilePunchHoleSize: BLOCK_SIZE,
		BlockSize:         BLOCK_SIZE,
	})
	if err == nil {
		t.Fatal("NewWrapAppendFile on directory returned nil err")
	}
}

// TestPwriteBatch_SubmitFailure_StillConsumesSlot: on SubmitWriteBatch
// error the write pointer must still end len(buf) past where it started.
// Readers locate a memtable at memId*capacity, so rolling the pointer back
// would land every later memtable one slot early.
func TestPwriteBatch_SubmitFailure_StillConsumesSlot(t *testing.T) {
	waf := newWrapFileWithRing(t, BLOCK_SIZE*8)

	// Warmup write moves offset off zero.
	warmup := createAlignedBuffer(BLOCK_SIZE, BLOCK_SIZE)
	if _, _, err := waf.PwriteBatch(warmup, BLOCK_SIZE); err != nil {
		t.Fatalf("warmup PwriteBatch: %v", err)
	}
	startOffset := waf.PhysicalWriteOffset.Load()

	// An invalid fd makes the io_uring write fail with EBADF.
	waf.WriteFd = -1

	buf := createAlignedBuffer(BLOCK_SIZE*2, BLOCK_SIZE)
	_, _, err := waf.PwriteBatch(buf, BLOCK_SIZE)
	if err == nil {
		t.Fatal("PwriteBatch err = nil, want error from failed SubmitWriteBatch")
	}
	if want := startOffset + int64(len(buf)); waf.PhysicalWriteOffset.Load() != want {
		t.Fatalf("PhysicalWriteOffset = %d after failed batch, want %d (slot must be consumed)",
			waf.PhysicalWriteOffset.Load(), want)
	}
}
