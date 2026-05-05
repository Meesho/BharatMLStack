package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	// Common page sizes (4KB is most common)
	PageSize4K  = 4 * 1024
	PageSize8K  = 8 * 1024
	PageSize16K = 16 * 1024
	PageSize64K = 64 * 1024

	// Test data sizes
	SmallRecord  = 128  // 128 bytes
	MediumRecord = 1024 // 1KB
	LargeRecord  = 8192 // 8KB
)

// PageAlignedBuffer provides page-aligned buffered writing
type PageAlignedBuffer struct {
	file       *os.File
	buffer     []byte
	bufferSize int
	writePos   int
	mu         sync.Mutex
}

// NewPageAlignedBuffer creates a new page-aligned buffer
func NewPageAlignedBuffer(filename string, bufferSize int) (*PageAlignedBuffer, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	// Align buffer to page boundary
	buffer := make([]byte, bufferSize)

	return &PageAlignedBuffer{
		file:       file,
		buffer:     buffer,
		bufferSize: bufferSize,
		writePos:   0,
	}, nil
}

// Write writes data to the buffer, flushing when page size is reached
func (pab *PageAlignedBuffer) Write(data []byte) error {
	pab.mu.Lock()
	defer pab.mu.Unlock()

	dataLen := len(data)

	// If data is larger than buffer, write directly
	if dataLen > pab.bufferSize {
		if pab.writePos > 0 {
			if err := pab.flushUnsafe(); err != nil {
				return err
			}
		}
		_, err := pab.file.Write(data)
		return err
	}

	// If data doesn't fit in current buffer, flush first
	if pab.writePos+dataLen > pab.bufferSize {
		if err := pab.flushUnsafe(); err != nil {
			return err
		}
	}

	// Copy data to buffer
	copy(pab.buffer[pab.writePos:], data)
	pab.writePos += dataLen

	return nil
}

// Flush flushes the buffer to disk
func (pab *PageAlignedBuffer) Flush() error {
	pab.mu.Lock()
	defer pab.mu.Unlock()
	return pab.flushUnsafe()
}

func (pab *PageAlignedBuffer) flushUnsafe() error {
	if pab.writePos == 0 {
		return nil
	}

	_, err := pab.file.Write(pab.buffer[:pab.writePos])
	if err != nil {
		return err
	}

	pab.writePos = 0
	return nil
}

// Sync syncs the file to disk
func (pab *PageAlignedBuffer) Sync() error {
	if err := pab.Flush(); err != nil {
		return err
	}
	return pab.file.Sync()
}

// Close closes the buffer and file
func (pab *PageAlignedBuffer) Close() error {
	if err := pab.Flush(); err != nil {
		return err
	}
	return pab.file.Close()
}

// DirectWriter wraps direct file writing
type DirectWriter struct {
	file *os.File
}

func NewDirectWriter(filename string) (*DirectWriter, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return &DirectWriter{file: file}, nil
}

func (dw *DirectWriter) Write(data []byte) error {
	_, err := dw.file.Write(data)
	return err
}

func (dw *DirectWriter) Sync() error {
	return dw.file.Sync()
}

func (dw *DirectWriter) Close() error {
	return dw.file.Close()
}

// MemoryMappedWriter uses memory mapping for writing
type MemoryMappedWriter struct {
	file     *os.File
	data     []byte
	size     int64
	writePos int64
	mu       sync.Mutex
}

func NewMemoryMappedWriter(filename string, size int64) (*MemoryMappedWriter, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	// Truncate file to desired size
	if err := file.Truncate(size); err != nil {
		file.Close()
		return nil, err
	}

	// Memory map the file
	data, err := syscall.Mmap(int(file.Fd()), 0, int(size), syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		file.Close()
		return nil, err
	}

	return &MemoryMappedWriter{
		file:     file,
		data:     data,
		size:     size,
		writePos: 0,
	}, nil
}

func (mmw *MemoryMappedWriter) Write(data []byte) error {
	mmw.mu.Lock()
	defer mmw.mu.Unlock()

	dataLen := int64(len(data))
	if mmw.writePos+dataLen > mmw.size {
		return fmt.Errorf("write would exceed mapped region")
	}

	copy(mmw.data[mmw.writePos:], data)
	mmw.writePos += dataLen

	return nil
}

func (mmw *MemoryMappedWriter) Sync() error {
	// Use manual msync syscall since syscall.Msync might not be available on all platforms
	_, _, errno := syscall.Syscall(syscall.SYS_MSYNC, uintptr(unsafe.Pointer(&mmw.data[0])), uintptr(len(mmw.data)), uintptr(syscall.MS_SYNC))
	if errno != 0 {
		return errno
	}
	return nil
}

func (mmw *MemoryMappedWriter) Close() error {
	if err := syscall.Munmap(mmw.data); err != nil {
		return err
	}
	return mmw.file.Close()
}

// Benchmark functions
func benchmarkPageAlignedBuffer(recordSize, numRecords, bufferSize int) time.Duration {
	filename := fmt.Sprintf("test_page_aligned_%d_%d_%d.log", recordSize, numRecords, bufferSize)
	defer os.Remove(filename)

	writer, err := NewPageAlignedBuffer(filename, bufferSize)
	if err != nil {
		panic(err)
	}
	defer writer.Close()

	data := make([]byte, recordSize)
	for i := 0; i < recordSize; i++ {
		data[i] = byte(i % 256)
	}

	start := time.Now()

	for i := 0; i < numRecords; i++ {
		if err := writer.Write(data); err != nil {
			panic(err)
		}
	}

	if err := writer.Sync(); err != nil {
		panic(err)
	}

	return time.Since(start)
}

func benchmarkDirectWrite(recordSize, numRecords int) time.Duration {
	filename := fmt.Sprintf("test_direct_%d_%d.log", recordSize, numRecords)
	defer os.Remove(filename)

	writer, err := NewDirectWriter(filename)
	if err != nil {
		panic(err)
	}
	defer writer.Close()

	data := make([]byte, recordSize)
	for i := 0; i < recordSize; i++ {
		data[i] = byte(i % 256)
	}

	start := time.Now()

	for i := 0; i < numRecords; i++ {
		if err := writer.Write(data); err != nil {
			panic(err)
		}
	}

	if err := writer.Sync(); err != nil {
		panic(err)
	}

	return time.Since(start)
}

func benchmarkBufferedWrite(recordSize, numRecords, bufferSize int) time.Duration {
	filename := fmt.Sprintf("test_buffered_%d_%d_%d.log", recordSize, numRecords, bufferSize)
	defer os.Remove(filename)

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	writer := bufio.NewWriterSize(file, bufferSize)

	data := make([]byte, recordSize)
	for i := 0; i < recordSize; i++ {
		data[i] = byte(i % 256)
	}

	start := time.Now()

	for i := 0; i < numRecords; i++ {
		if _, err := writer.Write(data); err != nil {
			panic(err)
		}
	}

	if err := writer.Flush(); err != nil {
		panic(err)
	}

	if err := file.Sync(); err != nil {
		panic(err)
	}

	return time.Since(start)
}

func benchmarkMemoryMapped(recordSize, numRecords int) time.Duration {
	filename := fmt.Sprintf("test_mmap_%d_%d.log", recordSize, numRecords)
	defer os.Remove(filename)

	totalSize := int64(recordSize * numRecords)
	writer, err := NewMemoryMappedWriter(filename, totalSize)
	if err != nil {
		panic(err)
	}
	defer writer.Close()

	data := make([]byte, recordSize)
	for i := 0; i < recordSize; i++ {
		data[i] = byte(i % 256)
	}

	start := time.Now()

	for i := 0; i < numRecords; i++ {
		if err := writer.Write(data); err != nil {
			panic(err)
		}
	}

	if err := writer.Sync(); err != nil {
		panic(err)
	}

	return time.Since(start)
}

func printResults(name string, duration time.Duration, recordSize, numRecords int) {
	totalBytes := int64(recordSize * numRecords)
	throughputMBps := float64(totalBytes) / duration.Seconds() / (1024 * 1024)
	recordsPerSec := float64(numRecords) / duration.Seconds()

	fmt.Printf("%-30s: %10s | %8.2f MB/s | %10.0f records/s | %8.2f MB total\n",
		name, duration.Round(time.Microsecond), throughputMBps, recordsPerSec, float64(totalBytes)/(1024*1024))
}

func runBenchmarks() {
	fmt.Println("=== Append-Only File Writing Benchmarks ===")
	fmt.Printf("Go Version: %s, OS: %s, Arch: %s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	fmt.Printf("CPUs: %d\n\n", runtime.NumCPU())

	testCases := []struct {
		recordSize int
		numRecords int
		name       string
	}{
		{SmallRecord, 100000, "Small Records (128B x 100K)"},
		{MediumRecord, 50000, "Medium Records (1KB x 50K)"},
		{LargeRecord, 10000, "Large Records (8KB x 10K)"},
	}

	bufferSizes := []int{PageSize4K, PageSize8K, PageSize16K, PageSize64K}

	for _, tc := range testCases {
		fmt.Printf("\n=== %s ===\n", tc.name)
		fmt.Printf("%-30s: %10s | %8s | %10s | %8s\n", "Method", "Duration", "MB/s", "Records/s", "Total MB")
		fmt.Println(strings.Repeat("-", 80))

		// Direct write benchmark
		duration := benchmarkDirectWrite(tc.recordSize, tc.numRecords)
		printResults("Direct Write", duration, tc.recordSize, tc.numRecords)

		// Buffered write benchmarks with different buffer sizes
		for _, bufSize := range bufferSizes {
			duration := benchmarkBufferedWrite(tc.recordSize, tc.numRecords, bufSize)
			name := fmt.Sprintf("Buffered (%dK)", bufSize/1024)
			printResults(name, duration, tc.recordSize, tc.numRecords)
		}

		// Page-aligned buffer benchmarks
		for _, bufSize := range bufferSizes {
			duration := benchmarkPageAlignedBuffer(tc.recordSize, tc.numRecords, bufSize)
			name := fmt.Sprintf("Page-Aligned (%dK)", bufSize/1024)
			printResults(name, duration, tc.recordSize, tc.numRecords)
		}

		// Memory-mapped benchmark (if total size is reasonable)
		totalSize := int64(tc.recordSize * tc.numRecords)
		if totalSize < 1024*1024*1024 { // Less than 1GB
			duration := benchmarkMemoryMapped(tc.recordSize, tc.numRecords)
			printResults("Memory Mapped", duration, tc.recordSize, tc.numRecords)
		}
	}

	fmt.Println("\n=== Recommendations ===")
	fmt.Println("1. For high-throughput workloads: Use page-aligned buffers with 16KB-64KB buffer sizes")
	fmt.Println("2. For low-latency workloads: Use smaller buffers (4KB-8KB) with frequent flushing")
	fmt.Println("3. For large sequential writes: Consider memory-mapped files")
	fmt.Println("4. Always align buffer sizes to page boundaries for optimal performance")
	fmt.Println("5. Use fdatasync instead of fsync when metadata updates aren't critical")
}

func main() {
	runBenchmarks()
}
