package float8

import (
	"fmt"
	"math"
	"testing"
)

func BenchmarkFP8E4M3FromFP32Value(b *testing.B) {
	b.ReportAllocs() // force the benchmark to report allocation stats even if they are minimal

	// Test cases
	tests := []struct {
		name  string
		value float32
	}{
		{"Normal positive", float32(math.Pi)},
		{"Normal negative", -2.25879},
		{"Subnormal positive", 0.0078125},
		{"Subnormal negative", -0.0078125},
		{"Positive infinity", float32(math.Inf(1))},
		{"Negative infinity", float32(math.Inf(-1))},
		{"NaN", float32(math.NaN())},
		{"Zero", 0},
	}

	for _, tt := range tests {
		// Run a sub-benchmark for each test case
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				FP8E4M3FromFP32Value(tt.value)
			}
		})
	}
}

type testCaseFP8E4M3 struct {
	name string
	fp8  Float8e4m3
}

func BenchmarkFP8E4M3ToFP32Value(b *testing.B) {
	b.ReportAllocs() // force the benchmark to report allocation stats even if they are minimal

	tests := make([]testCaseFP8E4M3, 256) // Preallocate a slice with 256 elements

	for i := 0; i < 256; i++ {
		tests[i] = testCaseFP8E4M3{
			name: fmt.Sprintf("Test %d", i+1), // Names as "1" to "256"
			fp8:  Float8e4m3(i),               // FP8 values as 0 to 255
		}
	}

	for _, tt := range tests {
		// Run the benchmark for each test case
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				FP8E4M3ToFP32Value(Float8e4m3(tt.fp8))
			}
		})
	}
}
