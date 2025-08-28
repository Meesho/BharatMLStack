package fs

import (
	"path/filepath"
	"testing"
)

func BenchmarkPwrite(b *testing.B) {
	tmpDir := b.TempDir()
	filename := filepath.Join(tmpDir, "bench_rolling_file.dat")

	config := FileConfig{
		Filename:          filename,
		MaxFileSize:       1024 * 1024 * 1024, // 1GB
		FilePunchHoleSize: 64 * 1024,
		BlockSize:         4096,
	}

	raf, err := NewRollingAppendFile(config)
	if err != nil {
		b.Fatalf("Failed to create RollingAppendFile: %v", err)
	}
	defer cleanup(raf)

	// Create aligned buffer for DirectIO
	data := createAlignedBuffer(4096, 4096)
	for i := 0; i < 4096; i++ {
		data[i] = byte(i % 256)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := raf.Pwrite(data)
		if err != nil {
			b.Fatalf("Pwrite failed: %v", err)
		}
	}
}

func BenchmarkPread(b *testing.B) {
	tmpDir := b.TempDir()
	filename := filepath.Join(tmpDir, "bench_rolling_file.dat")

	config := FileConfig{
		Filename:          filename,
		MaxFileSize:       1024 * 1024 * 1024, // 1GB
		FilePunchHoleSize: 64 * 1024,
		BlockSize:         4096,
	}

	raf, err := NewRollingAppendFile(config)
	if err != nil {
		b.Fatalf("Failed to create RollingAppendFile: %v", err)
	}
	defer cleanup(raf)

	// Pre-populate with data using aligned buffer
	writeData := createAlignedBuffer(4096, 4096)
	for i := 0; i < 4096; i++ {
		writeData[i] = byte(i % 256)
	}

	for i := 0; i < 200000; i++ {
		_, err := raf.Pwrite(writeData)
		if err != nil {
			b.Fatalf("Pwrite failed: %v", err)
		}
	}

	readData := createAlignedBuffer(4096, 4096)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		offset := int64((i % 200000) * 4096)
		_, err := raf.Pread(offset, readData)
		if err != nil {
			b.Fatalf("Pread failed: %v", err)
		}
	}
}

// Benchmarks
func BenchmarkWrapAppendFile_Pwrite(b *testing.B) {
	tmpDir := b.TempDir()
	filename := filepath.Join(tmpDir, "bench_wrap_file.dat")

	config := FileConfig{
		Filename:          filename,
		MaxFileSize:       1024 * 1024 * 1024, // 1GB
		FilePunchHoleSize: 64 * 1024,
		BlockSize:         4096,
	}

	waf, err := NewWrapAppendFile(config)
	if err != nil {
		b.Fatalf("Failed to create WrapAppendFile: %v", err)
	}
	defer cleanupWrapFile(waf)

	// Create aligned buffer for DirectIO
	data := createAlignedBuffer(4096, 4096)
	for i := 0; i < 4096; i++ {
		data[i] = byte(i % 256)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := waf.Pwrite(data)
		if err != nil {
			b.Fatalf("Pwrite failed: %v", err)
		}
	}
}

func BenchmarkWrapAppendFile_Pread(b *testing.B) {
	tmpDir := b.TempDir()
	filename := filepath.Join(tmpDir, "bench_wrap_file.dat")

	config := FileConfig{
		Filename:          filename,
		MaxFileSize:       1024 * 1024 * 1024, // 1GB
		FilePunchHoleSize: 64 * 1024,
		BlockSize:         4096,
	}

	waf, err := NewWrapAppendFile(config)
	if err != nil {
		b.Fatalf("Failed to create WrapAppendFile: %v", err)
	}
	defer cleanupWrapFile(waf)

	// Pre-populate with data using aligned buffer
	writeData := createAlignedBuffer(4096, 4096)
	for i := 0; i < 4096; i++ {
		writeData[i] = byte(i % 256)
	}

	for i := 0; i < 200000; i++ {
		_, err := waf.Pwrite(writeData)
		if err != nil {
			b.Fatalf("Pwrite failed: %v", err)
		}
	}

	readData := createAlignedBuffer(4096, 4096)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		offset := int64((i % 200000) * 4096)
		_, err := waf.Pread(offset, readData)
		if err != nil {
			b.Fatalf("Pread failed: %v", err)
		}
	}
}
