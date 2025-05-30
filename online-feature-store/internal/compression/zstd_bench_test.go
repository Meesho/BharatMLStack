package compression

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/rs/zerolog"
)

func BenchmarkZStdEncoder_Encode(b *testing.B) {
	// Set logging level to error
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)

	// Populate test data
	data := populateNormFP64Bytes(100)

	// Define different configurations for CPU count and iterations
	configurations := []struct {
		numCPUs    int
		iterations int
	}{
		{4, 1000},
	}

	// Run each configuration
	for _, config := range configurations {
		runtime.GOMAXPROCS(config.numCPUs)
		b.Run(
			fmt.Sprintf("CPUs_%d_Iterations_%d", config.numCPUs, config.iterations),
			func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < config.iterations; i++ {
					b.StartTimer()
					enc := NewZStdEncoder()
					cdata := enc.Encode(data)
					b.StopTimer()
					if len(cdata) == 0 || len(cdata) > len(data) {
						b.Errorf("Invalid compressed data length: %d", len(cdata))
					}
				}
			})
	}
}

func BenchmarkZStdDecoder_Decode(b *testing.B) {
	// Set logging level to error
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)

	// Populate test data
	data := populateNormFP64Bytes(1000)

	// Encode the test data
	enc := NewZStdEncoder()
	cdata := enc.Encode(data)

	// Define different configurations for CPU count and iterations
	configurations := []struct {
		numCPUs    int
		iterations int
	}{
		{1, 1000},
		{2, 1000},
		{4, 1000},
		{1, 2000},
		{2, 2000},
		{4, 2000},
		{1, 4000},
		{2, 4000},
		{4, 4000},
	}

	// Run each configuration
	for _, config := range configurations {
		runtime.GOMAXPROCS(config.numCPUs)
		b.Run(
			fmt.Sprintf("CPUs_%d_Iterations_%d", config.numCPUs, config.iterations),
			func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < config.iterations; i++ {
					b.StartTimer()
					dec := NewZStdDecoder()
					ddata, err := dec.Decode(cdata)
					b.StopTimer()
					if err != nil {
						b.Errorf("Error decoding compressed data: %v", err)
					}
					if len(ddata) == 0 || len(ddata) < len(cdata) {
						b.Errorf("Invalid decompressed data length: %d", len(ddata))
					}
				}
			})
	}
}
