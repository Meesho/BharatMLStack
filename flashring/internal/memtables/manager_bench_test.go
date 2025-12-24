package memtables

import (
	"fmt"
	"testing"
	"time"

	"github.com/Meesho/BharatMLStack/flashring/internal/fs"
)

// Helper function to create a test file for benchmarks
func createManagerBenchmarkFile(b *testing.B) *fs.WrapAppendFile {
	filename := fmt.Sprintf("/media/a0d00kc/freedom/tmp/bench_memtable_%d.dat", time.Now().UnixNano())

	config := fs.FileConfig{
		Filename:          filename,
		MaxFileSize:       20 * 1024 * 1024 * 1024, // 20GB for benchmarks
		FilePunchHoleSize: 1024 * 1024 * 1024,      // 1GB
		BlockSize:         fs.BLOCK_SIZE,
	}

	file, err := fs.NewWrapAppendFile(config)
	if err != nil {
		b.Fatalf("Failed to create benchmark file: %v", err)
	}
	return file
}

func Benchmark_Puts(b *testing.B) {
	file := createManagerBenchmarkFile(b)

	manager, err := NewMemtableManager(file, 1024*1024*1024)
	if err != nil {
		b.Fatalf("Failed to create memtable manager: %v", err)
	}

	buf16k := make([]byte, 16*1024)
	for j := range buf16k {
		buf16k[j] = byte(j % 256)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		memtable, _, _ := manager.GetMemtable()
		_, _, readyForFlush := memtable.Put(buf16k)
		if readyForFlush {
			manager.Flush()
		}
	}

	b.ReportMetric(float64(manager.stats.Flushes), "flushes")
	b.ReportMetric(float64(b.N*16*1024)/1024/1024, "MB/s")
	b.ReportAllocs()

}
