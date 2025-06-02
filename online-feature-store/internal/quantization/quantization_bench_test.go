package quantization

import (
	"fmt"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/types"
	"testing"
)

func BenchmarkQuantize(b *testing.B) {
	// Define source data types for columns
	columnSourceDatatypeMap := map[string]types.DataType{
		"col1":  types.DataTypeFP32,
		"col2":  types.DataTypeFP32,
		"col3":  types.DataTypeFP32,
		"col4":  types.DataTypeFP32,
		"col5":  types.DataTypeFP16,
		"col6":  types.DataTypeFP16,
		"col7":  types.DataTypeFP64,
		"col8":  types.DataTypeFP64,
		"col9":  types.DataTypeFP64,
		"col10": types.DataTypeFP64,
	}

	// Define requested data types for columns
	columnRequestedDatatypeMap := map[string]types.DataType{
		"col1":  types.DataTypeFP8E5M2,
		"col2":  types.DataTypeFP8E4M3,
		"col3":  types.DataTypeFP16,
		"col4":  types.DataTypeFP16,
		"col5":  types.DataTypeFP8E5M2,
		"col6":  types.DataTypeFP8E4M3,
		"col7":  types.DataTypeFP32,
		"col8":  types.DataTypeFP16,
		"col9":  types.DataTypeFP8E5M2,
		"col10": types.DataTypeFP8E4M3,
	}

	// Define matrix column indices
	matrixColumnIndexMap := map[string]int{
		"col1":  0,
		"col2":  1,
		"col3":  2,
		"col4":  3,
		"col5":  4,
		"col6":  5,
		"col7":  6,
		"col8":  7,
		"col9":  8,
		"col10": 9,
	}

	// Size of byte slices to test
	sliceSizes := []int{500, 1000, 2500, 5000, 10000, 25000, 50000, 75000, 90000, 100000}

	for _, size := range sliceSizes {
		// Define the size-specific benchmark
		b.Run(fmt.Sprintf("SliceSize-%d", size), func(b *testing.B) {
			// Generate reusable byte slice for the current size
			baseSlice := make([]byte, size)
			for i := range baseSlice {
				baseSlice[i] = byte(i % 256)
			}

			// Benchmark loop
			for i := 0; i < b.N; i++ {
				// Create a matrix with 500 rows and 100 columns
				matrix := make([][][]byte, 500)
				for r := 0; r < 500; r++ {
					row := make([][]byte, 100)
					for c := 0; c < 100; c++ {
						row[c] = append([]byte{}, baseSlice...) // Ensure a new copy for each element
					}
					matrix[r] = row
				}

				// Call the quantize function
				quantize(matrix, columnSourceDatatypeMap, columnRequestedDatatypeMap, matrixColumnIndexMap)
			}
		})
	}
}
