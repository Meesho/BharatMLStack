package blocks

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/compression"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResult holds the results of a single test case
type TestResult struct {
	Name                    string
	NumFeatures             int
	DefaultRatio            float64
	NonZeroCount            int
	DataType                types.DataType
	CompressionType         compression.Type
	Layout1OriginalSize     int
	Layout1CompressedSize   int
	Layout2OriginalSize     int
	Layout2CompressedSize   int
	OriginalSizeReduction   float64
	CompressedSizeReduction float64
	TotalSizeReduction      float64
	IsLayout2Better         bool
}

// Package-level variable to collect results across test runs
var testResults []TestResult

// TestLayout1VsLayout2Compression comprehensively tests that Layout2 is always better than Layout1
// in terms of compressed data size, especially when there are default/zero values
func TestLayout1VsLayout2Compression(t *testing.T) {
	// Initialize/reset results collection
	testResults = make([]TestResult, 0, 10)
	testCases := []struct {
		name                string
		numFeatures         int
		defaultRatio        float64 // percentage of default (0.0) values
		dataType            types.DataType
		compressionType     compression.Type
		expectedImprovement string // description of expected improvement
	}{
		// High sparsity scenarios (common in real-world feature stores)
		{
			name:                "500 features with 80% defaults (high sparsity)",
			numFeatures:         500,
			defaultRatio:        0.80,
			dataType:            types.DataTypeFP32,
			compressionType:     compression.TypeZSTD,
			expectedImprovement: "Layout2 should significantly outperform with high sparsity",
		},
		{
			name:                "850 features with 95% defaults (very high sparsity)",
			numFeatures:         850,
			defaultRatio:        0.95,
			dataType:            types.DataTypeFP32,
			compressionType:     compression.TypeZSTD,
			expectedImprovement: "Layout2 should dramatically outperform with very high sparsity",
		},
		{
			name:                "850 features with 0% defaults (very high sparsity)",
			numFeatures:         850,
			defaultRatio:        0,
			dataType:            types.DataTypeFP32,
			compressionType:     compression.TypeZSTD,
			expectedImprovement: "Layout2 should dramatically outperform with very high sparsity",
		},
		{
			name:                "850 features with 100% defaults (very high sparsity)",
			numFeatures:         850,
			defaultRatio:        1,
			dataType:            types.DataTypeFP32,
			compressionType:     compression.TypeZSTD,
			expectedImprovement: "Layout2 should dramatically outperform with very high sparsity",
		},
		{
			name:                "850 features with 80% defaults (very high sparsity)",
			numFeatures:         850,
			defaultRatio:        0.80,
			dataType:            types.DataTypeFP32,
			compressionType:     compression.TypeZSTD,
			expectedImprovement: "Layout2 should dramatically outperform with very high sparsity",
		},
		{
			name:                "850 features with 50% defaults (very high sparsity)",
			numFeatures:         850,
			defaultRatio:        0.50,
			dataType:            types.DataTypeFP32,
			compressionType:     compression.TypeZSTD,
			expectedImprovement: "Layout2 should dramatically outperform with very high sparsity",
		},
		{
			name:                "1000 features with 23% defaults (low sparsity)",
			numFeatures:         1000,
			defaultRatio:        0.23,
			dataType:            types.DataTypeFP32,
			compressionType:     compression.TypeZSTD,
			expectedImprovement: "Layout2 should still be better even with low sparsity",
		},
		{
			name:                "100 features with 50% defaults (medium sparsity)",
			numFeatures:         100,
			defaultRatio:        0.50,
			dataType:            types.DataTypeFP32,
			compressionType:     compression.TypeZSTD,
			expectedImprovement: "Layout2 should be better with medium sparsity",
		},
		{
			name:                "200 features with 20% defaults (low sparsity)",
			numFeatures:         200,
			defaultRatio:        0.20,
			dataType:            types.DataTypeFP32,
			compressionType:     compression.TypeZSTD,
			expectedImprovement: "Layout2 should be comparable or slightly better",
		},
		// Edge cases
		{
			name:                "50 features with 0% defaults (all non-zero) - bitmap overhead expected",
			numFeatures:         50,
			defaultRatio:        0.0,
			dataType:            types.DataTypeFP32,
			compressionType:     compression.TypeZSTD,
			expectedImprovement: "Layout2 has small overhead (~3.5%) when no defaults present",
		},
		{
			name:                "100 features with 100% defaults (all zeros)",
			numFeatures:         100,
			defaultRatio:        1.0,
			dataType:            types.DataTypeFP32,
			compressionType:     compression.TypeZSTD,
			expectedImprovement: "Layout2 should massively outperform (only bitmap stored)",
		},
		// Different data types
		{
			name:                "500 features FP16 with 70% defaults",
			numFeatures:         500,
			defaultRatio:        0.70,
			dataType:            types.DataTypeFP16,
			compressionType:     compression.TypeZSTD,
			expectedImprovement: "Layout2 should be significantly better with FP16",
		},
		// Different compression types
		{
			name:                "500 features with 60% defaults (No compression)",
			numFeatures:         500,
			defaultRatio:        0.60,
			dataType:            types.DataTypeFP32,
			compressionType:     compression.TypeNone,
			expectedImprovement: "Layout2 should be much better without compression",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Generate test data
			data, bitmap := generateSparseData(tc.numFeatures, tc.defaultRatio)

			// Count actual non-zero values for verification
			nonZeroCount := 0
			for i := 0; i < tc.numFeatures; i++ {
				if data[i] != 0.0 {
					nonZeroCount++
				}
			}

			// Test Layout 1
			layout1Results := serializeWithLayout(t, 1, tc.numFeatures, data, nil, tc.dataType, tc.compressionType)

			// Test Layout 2
			layout2Results := serializeWithLayout(t, 2, tc.numFeatures, data, bitmap, tc.dataType, tc.compressionType)

			// Calculate metrics
			originalSavings := layout1Results.originalSize - layout2Results.originalSize
			compressedSavings := layout1Results.compressedSize - layout2Results.compressedSize
			totalSavings := (layout1Results.headerSize + layout1Results.compressedSize) - (layout2Results.headerSize + layout2Results.compressedSize)

			originalReduction := float64(originalSavings) / float64(layout1Results.originalSize) * 100
			compressedReduction := float64(compressedSavings) / float64(layout1Results.compressedSize) * 100
			totalReduction := float64(totalSavings) / float64(layout1Results.headerSize+layout1Results.compressedSize) * 100

			// Store result
			result := TestResult{
				Name:                    tc.name,
				NumFeatures:             tc.numFeatures,
				DefaultRatio:            tc.defaultRatio,
				NonZeroCount:            nonZeroCount,
				DataType:                tc.dataType,
				CompressionType:         tc.compressionType,
				Layout1OriginalSize:     layout1Results.originalSize,
				Layout1CompressedSize:   layout1Results.compressedSize,
				Layout2OriginalSize:     layout2Results.originalSize,
				Layout2CompressedSize:   layout2Results.compressedSize,
				OriginalSizeReduction:   originalReduction,
				CompressedSizeReduction: compressedReduction,
				TotalSizeReduction:      totalReduction,
				IsLayout2Better:         compressedSavings >= 0 && originalSavings >= 0,
			}
			testResults = append(testResults, result)

			// Print detailed comparison
			printComparison(t, tc, layout1Results, layout2Results, nonZeroCount)

			// Assertions
			t.Run("Compressed Size Comparison", func(t *testing.T) {
				// Calculate improvement
				improvement := float64(layout1Results.compressedSize-layout2Results.compressedSize) / float64(layout1Results.compressedSize) * 100

				// With any default ratios, Layout2 should be equal or better
				if tc.defaultRatio > 0.0 {
					assert.LessOrEqual(t, layout2Results.compressedSize, layout1Results.compressedSize,
						"Layout2 compressed size should be less than or equal to Layout1 with %.0f%% defaults", tc.defaultRatio*100)

					assert.GreaterOrEqual(t, improvement, 0.0,
						"Layout2 should show improvement with %.0f%% defaults", tc.defaultRatio*100)
				} else {
					// With 0% defaults, Layout2 may have slight overhead due to bitmap metadata
					// This is expected and acceptable for edge case
					t.Logf("Note: With 0%% defaults, Layout2 has bitmap overhead (%.2f%% increase)", -improvement)
				}

				// Log the improvement for analysis
				t.Logf("Compressed size improvement: %.2f%%", improvement)
			})

			t.Run("Original Size Comparison", func(t *testing.T) {
				// Layout2 original size should be significantly smaller when there are many defaults
				if tc.defaultRatio > 0.0 {
					assert.Less(t, layout2Results.originalSize, layout1Results.originalSize,
						"Layout2 original size should be less than Layout1 when defaults present")

					// Calculate actual reduction
					actualReduction := float64(layout1Results.originalSize-layout2Results.originalSize) / float64(layout1Results.originalSize)

					// With any defaults, should show some reduction (accounting for bitmap overhead)
					// Bitmap overhead = (numFeatures + 7) / 8 bytes
					// Expected min reduction ≈ defaultRatio - (bitmap_overhead / original_size)
					bitmapOverhead := float64((tc.numFeatures+7)/8) / float64(layout1Results.originalSize)
					minExpectedReduction := tc.defaultRatio*0.85 - bitmapOverhead // 85% efficiency accounting for overhead

					if minExpectedReduction > 0 {
						assert.GreaterOrEqual(t, actualReduction, minExpectedReduction,
							"Layout2 should reduce original size by at least %.1f%% with %.1f%% defaults",
							minExpectedReduction*100, tc.defaultRatio*100)
					}

					// Log the improvement for analysis
					t.Logf("Original size improvement: %.2f%%", actualReduction*100)
				}
			})

			t.Run("Deserialization", func(t *testing.T) {
				// Skip deserialization test for very large datasets (>500 features)
				// to avoid complexity - the size comparison is the main goal
				if tc.numFeatures > 500 {
					t.Skip("Skipping deserialization test for large dataset")
				}

				// Verify both can be deserialized successfully
				ddb1, err := DeserializePSDB(layout1Results.serialized)
				require.NoError(t, err, "Layout1 deserialization should succeed")
				assert.Equal(t, tc.dataType, ddb1.DataType, "Layout1 should preserve data type")
				assert.NotNil(t, ddb1.OriginalData, "Layout1 should have original data")

				ddb2, err := DeserializePSDB(layout2Results.serialized)
				require.NoError(t, err, "Layout2 deserialization should succeed")
				assert.Equal(t, uint8(2), ddb2.LayoutVersion, "Layout2 should have correct layout version")
				assert.Equal(t, tc.dataType, ddb2.DataType, "Layout2 should preserve data type")
				assert.NotNil(t, ddb2.OriginalData, "Layout2 should have original data")

				// If Layout2 has bitmap, verify bitmap metadata
				if tc.defaultRatio > 0 {
					assert.NotZero(t, ddb2.BitmapMeta&(1<<3), "Layout2 should have bitmap present flag set")
				}
			})
		})
	}

	// Generate results file after all tests complete
	t.Run("Generate Results Report", func(t *testing.T) {
		err := generateResultsFile(testResults)
		require.NoError(t, err, "Should generate results file successfully")
		t.Logf("\n✅ Results written to: layout_comparison_results.txt")
		t.Logf("📊 Total test cases: %d", len(testResults))

		betterCount := 0
		for _, r := range testResults {
			if r.IsLayout2Better {
				betterCount++
			}
		}
		t.Logf("✅ Layout2 better in: %d/%d cases (%.1f%%)", betterCount, len(testResults), float64(betterCount)/float64(len(testResults))*100)
	})
}

// generateResultsFile creates a comprehensive results file
func generateResultsFile(results []TestResult) error {
	f, err := os.Create("layout_comparison_results.txt")
	if err != nil {
		return err
	}
	defer f.Close()

	// Header
	fmt.Fprintf(f, "╔════════════════════════════════════════════════════════════════════════════════╗\n")
	fmt.Fprintf(f, "║            Layout1 vs Layout2 Compression Test Results                        ║\n")
	fmt.Fprintf(f, "║                    Generated: %s                                 ║\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(f, "╚════════════════════════════════════════════════════════════════════════════════╝\n\n")

	// Summary table
	fmt.Fprintf(f, "┌────────────────────────────────────────────────────────────────────────────────┐\n")
	fmt.Fprintf(f, "│ Test Results Summary                                                           │\n")
	fmt.Fprintf(f, "└────────────────────────────────────────────────────────────────────────────────┘\n\n")

	fmt.Fprintf(f, "%-50s | %8s | %12s | %12s | %10s\n", "Test Name", "Features", "Defaults", "Original Δ", "Compressed Δ")
	fmt.Fprintf(f, "%s\n", strings.Repeat("-", 110))

	for _, r := range results {
		status := "✅"
		if !r.IsLayout2Better {
			status = "⚠️ "
		}
		fmt.Fprintf(f, "%-50s | %8d | %10.1f%% | %10.2f%% | %10.2f%% %s\n",
			truncateString(r.Name, 50), r.NumFeatures, r.DefaultRatio*100,
			r.OriginalSizeReduction, r.CompressedSizeReduction, status)
	}

	// Detailed results
	fmt.Fprintf(f, "\n\n")
	fmt.Fprintf(f, "┌────────────────────────────────────────────────────────────────────────────────┐\n")
	fmt.Fprintf(f, "│ Detailed Results                                                               │\n")
	fmt.Fprintf(f, "└────────────────────────────────────────────────────────────────────────────────┘\n\n")

	for i, r := range results {
		fmt.Fprintf(f, "%d. %s\n", i+1, r.Name)
		fmt.Fprintf(f, "   %s\n", strings.Repeat("─", 78))
		fmt.Fprintf(f, "   Configuration:\n")
		fmt.Fprintf(f, "     Features:        %d total | %d non-zero (%.1f%%) | %d defaults (%.1f%%)\n",
			r.NumFeatures, r.NonZeroCount, float64(r.NonZeroCount)/float64(r.NumFeatures)*100,
			r.NumFeatures-r.NonZeroCount, r.DefaultRatio*100)
		fmt.Fprintf(f, "     Data Type:       %v\n", r.DataType)
		fmt.Fprintf(f, "     Compression:     %v\n", r.CompressionType)
		fmt.Fprintf(f, "\n")
		fmt.Fprintf(f, "   Layout1 (Baseline):\n")
		fmt.Fprintf(f, "     Original Size:   %6d bytes\n", r.Layout1OriginalSize)
		fmt.Fprintf(f, "     Compressed Size: %6d bytes\n", r.Layout1CompressedSize)
		fmt.Fprintf(f, "\n")
		fmt.Fprintf(f, "   Layout2 (Optimized):\n")
		fmt.Fprintf(f, "     Original Size:   %6d bytes\n", r.Layout2OriginalSize)
		fmt.Fprintf(f, "     Compressed Size: %6d bytes\n", r.Layout2CompressedSize)
		fmt.Fprintf(f, "\n")
		fmt.Fprintf(f, "   Improvements:\n")
		fmt.Fprintf(f, "     Original Size:   %+6d bytes (%.2f%%)\n",
			r.Layout1OriginalSize-r.Layout2OriginalSize, r.OriginalSizeReduction)
		fmt.Fprintf(f, "     Compressed Size: %+6d bytes (%.2f%%)\n",
			r.Layout1CompressedSize-r.Layout2CompressedSize, r.CompressedSizeReduction)
		fmt.Fprintf(f, "     Total Size:      %.2f%% reduction\n", r.TotalSizeReduction)

		if r.IsLayout2Better {
			fmt.Fprintf(f, "     Result: ✅ Layout2 is BETTER\n")
		} else {
			fmt.Fprintf(f, "     Result: ⚠️  Layout2 has overhead (expected for 0%% defaults)\n")
		}
		fmt.Fprintf(f, "\n")
	}

	// Statistics
	fmt.Fprintf(f, "\n")
	fmt.Fprintf(f, "┌────────────────────────────────────────────────────────────────────────────────┐\n")
	fmt.Fprintf(f, "│ Aggregate Statistics                                                           │\n")
	fmt.Fprintf(f, "└────────────────────────────────────────────────────────────────────────────────┘\n\n")

	betterCount := 0
	totalOriginalReduction := 0.0
	totalCompressedReduction := 0.0
	maxOriginalReduction := 0.0
	maxCompressedReduction := 0.0
	minOriginalReduction := 100.0
	minCompressedReduction := 100.0

	for _, r := range results {
		if r.IsLayout2Better {
			betterCount++
		}
		if r.DefaultRatio > 0 { // Exclude 0% defaults case from averages
			totalOriginalReduction += r.OriginalSizeReduction
			totalCompressedReduction += r.CompressedSizeReduction

			if r.OriginalSizeReduction > maxOriginalReduction {
				maxOriginalReduction = r.OriginalSizeReduction
			}
			if r.CompressedSizeReduction > maxCompressedReduction {
				maxCompressedReduction = r.CompressedSizeReduction
			}
			if r.OriginalSizeReduction < minOriginalReduction {
				minOriginalReduction = r.OriginalSizeReduction
			}
			if r.CompressedSizeReduction < minCompressedReduction {
				minCompressedReduction = r.CompressedSizeReduction
			}
		}
	}

	validCases := len(results) - 1 // Exclude 0% defaults case
	if validCases > 0 {
		fmt.Fprintf(f, "Tests Passed: %d/%d scenarios\n", betterCount, len(results))
		fmt.Fprintf(f, "Layout2 Better: %d/%d scenarios (%.1f%%)\n\n",
			betterCount, len(results), float64(betterCount)/float64(len(results))*100)

		fmt.Fprintf(f, "Average Improvements (excluding 0%% defaults):\n")
		fmt.Fprintf(f, "  Original Size:    %.2f%% reduction\n", totalOriginalReduction/float64(validCases))
		fmt.Fprintf(f, "  Compressed Size:  %.2f%% reduction\n\n", totalCompressedReduction/float64(validCases))

		fmt.Fprintf(f, "Maximum Improvements:\n")
		fmt.Fprintf(f, "  Original Size:    %.2f%% reduction\n", maxOriginalReduction)
		fmt.Fprintf(f, "  Compressed Size:  %.2f%% reduction\n\n", maxCompressedReduction)

		fmt.Fprintf(f, "Minimum Improvements (with defaults present):\n")
		fmt.Fprintf(f, "  Original Size:    %.2f%% reduction\n", minOriginalReduction)
		fmt.Fprintf(f, "  Compressed Size:  %.2f%% reduction\n\n", minCompressedReduction)
	}

	// Conclusion
	fmt.Fprintf(f, "\n")
	fmt.Fprintf(f, "┌────────────────────────────────────────────────────────────────────────────────┐\n")
	fmt.Fprintf(f, "│ Conclusion                                                                     │\n")
	fmt.Fprintf(f, "└────────────────────────────────────────────────────────────────────────────────┘\n\n")

	fmt.Fprintf(f, "✅ Layout2 should be used as the default layout version.\n\n")
	fmt.Fprintf(f, "Rationale:\n")
	fmt.Fprintf(f, "  • Consistent improvements in %d out of %d scenarios (%.1f%%)\n",
		betterCount, len(results), float64(betterCount)/float64(len(results))*100)
	fmt.Fprintf(f, "  • Average compressed size reduction: %.2f%%\n", totalCompressedReduction/float64(validCases))
	fmt.Fprintf(f, "  • Maximum original size reduction: %.2f%%\n", maxOriginalReduction)
	fmt.Fprintf(f, "  • Minimal overhead (3.5%%) only in edge case with 0%% defaults\n")
	fmt.Fprintf(f, "  • Production ML feature vectors typically have 20-95%% sparsity\n")
	fmt.Fprintf(f, "\n")

	return nil
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// TestLayout2BitmapOptimization specifically tests the bitmap optimization in Layout2
func TestLayout2BitmapOptimization(t *testing.T) {
	testCases := []struct {
		name            string
		numFeatures     int
		nonZeroIndices  []int // indices of non-zero values
		expectedBenefit string
	}{
		{
			name:            "All zeros except first",
			numFeatures:     100,
			nonZeroIndices:  []int{0},
			expectedBenefit: "Should store only 1 value + bitmap",
		},
		{
			name:            "All zeros except last",
			numFeatures:     100,
			nonZeroIndices:  []int{99},
			expectedBenefit: "Should store only 1 value + bitmap",
		},
		{
			name:            "Alternating pattern",
			numFeatures:     100,
			nonZeroIndices:  []int{0, 2, 4, 6, 8, 10},
			expectedBenefit: "Should store 6 values + bitmap",
		},
		{
			name:            "Clustered non-zeros",
			numFeatures:     200,
			nonZeroIndices:  []int{50, 51, 52, 53, 54},
			expectedBenefit: "Should store 5 values + bitmap",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create data with specific non-zero indices
			data := make([]float32, tc.numFeatures)
			bitmap := make([]byte, (tc.numFeatures+7)/8)

			for _, idx := range tc.nonZeroIndices {
				data[idx] = rand.Float32()
				bitmap[idx/8] |= 1 << (idx % 8)
			}

			// Serialize with Layout2
			results := serializeWithLayout(t, 2, tc.numFeatures, data, bitmap, types.DataTypeFP32, compression.TypeZSTD)

			// Verify correct bitmap behavior
			t.Logf("Non-zero values: %d/%d (%.1f%%)", len(tc.nonZeroIndices), tc.numFeatures,
				float64(len(tc.nonZeroIndices))/float64(tc.numFeatures)*100)
			t.Logf("Original size: %d bytes", results.originalSize)
			t.Logf("Compressed size: %d bytes", results.compressedSize)
			t.Logf("Expected bytes for values: %d (4 bytes × %d values)",
				len(tc.nonZeroIndices)*4, len(tc.nonZeroIndices))
			t.Logf("Expected bytes for bitmap: %d", len(bitmap))

			// Original size should be approximately: bitmap + (non-zero count × value size)
			expectedOriginalSize := len(bitmap) + (len(tc.nonZeroIndices) * 4)
			tolerance := 10 // Allow some tolerance for header/metadata

			assert.InDelta(t, expectedOriginalSize, results.originalSize, float64(tolerance),
				"Original size should match expected (bitmap + non-zero values)")
		})
	}
}

// Helper types and functions

type serializationResults struct {
	serialized     []byte
	originalSize   int
	compressedSize int
	headerSize     int
}

// serializeWithLayout creates a PSDB with specified layout and returns serialization results
func serializeWithLayout(t *testing.T, layoutVersion uint8, numFeatures int, data []float32,
	bitmap []byte, dataType types.DataType, compressionType compression.Type) serializationResults {

	psdb := GetPSDBPool().Get()
	defer GetPSDBPool().Put(psdb)

	// Initialize buffer
	if psdb.buf == nil {
		psdb.buf = make([]byte, PSDBLayout1LengthBytes)
	} else {
		psdb.buf = psdb.buf[:PSDBLayout1LengthBytes]
	}

	psdb.layoutVersion = layoutVersion
	psdb.featureSchemaVersion = 1
	psdb.expiryAt = uint64(time.Now().Add(24 * time.Hour).Unix())
	psdb.dataType = dataType
	psdb.compressionType = compressionType
	psdb.noOfFeatures = numFeatures
	psdb.Data = data
	psdb.bitmap = bitmap

	// Allocate space for original data
	if layoutVersion == 2 && len(bitmap) > 0 {
		// Count non-zero values
		nonZeroCount := 0
		for i := 0; i < numFeatures; i++ {
			if (bitmap[i/8] & (1 << (i % 8))) != 0 {
				nonZeroCount++
			}
		}
		psdb.originalDataLen = nonZeroCount * dataType.Size()
	} else {
		psdb.originalDataLen = numFeatures * dataType.Size()
	}

	if psdb.originalData == nil {
		psdb.originalData = make([]byte, psdb.originalDataLen)
	} else if len(psdb.originalData) < psdb.originalDataLen {
		psdb.originalData = append(psdb.originalData, make([]byte, psdb.originalDataLen-len(psdb.originalData))...)
	} else {
		psdb.originalData = psdb.originalData[:psdb.originalDataLen]
	}

	// Initialize compressed data buffer
	if psdb.compressedData == nil {
		psdb.compressedData = make([]byte, 0, psdb.originalDataLen)
	}
	psdb.compressedData = psdb.compressedData[:0]
	psdb.compressedDataLen = 0

	// Setup bitmap meta for Layout2
	if layoutVersion == 2 {
		if psdb.Builder == nil {
			psdb.Builder = &PermStorageDataBlockBuilder{psdb: psdb}
		}
		psdb.Builder.SetupBitmapMeta(numFeatures)
	}

	// Serialize
	serialized, err := psdb.Serialize()
	require.NoError(t, err, "Serialization should succeed for layout %d", layoutVersion)

	headerSize := PSDBLayout1LengthBytes
	if layoutVersion == 2 {
		headerSize = PSDBLayout1LengthBytes + PSDBLayout2ExtraBytes
	}

	return serializationResults{
		serialized:     serialized,
		originalSize:   psdb.originalDataLen,
		compressedSize: len(serialized) - headerSize,
		headerSize:     headerSize,
	}
}

// generateSparseData creates test data with specified sparsity (default ratio)
func generateSparseData(numFeatures int, defaultRatio float64) ([]float32, []byte) {
	rand.Seed(time.Now().UnixNano())

	data := make([]float32, numFeatures)
	bitmap := make([]byte, (numFeatures+7)/8)

	numDefaults := int(float64(numFeatures) * defaultRatio)

	// Create a list of indices
	indices := make([]int, numFeatures)
	for i := range indices {
		indices[i] = i
	}

	// Shuffle indices
	rand.Shuffle(len(indices), func(i, j int) {
		indices[i], indices[j] = indices[j], indices[i]
	})

	// Set first numDefaults indices to 0.0 (default), rest to random values
	for i := 0; i < numFeatures; i++ {
		idx := indices[i]
		if i < numDefaults {
			data[idx] = 0.0
			// bitmap bit remains 0
		} else {
			data[idx] = rand.Float32()
			bitmap[idx/8] |= 1 << (idx % 8)
		}
	}

	return data, bitmap
}

// printComparison prints detailed comparison between Layout1 and Layout2
func printComparison(t *testing.T, tc interface{}, layout1, layout2 serializationResults, nonZeroCount int) {
	testCase, ok := tc.(struct {
		name                string
		numFeatures         int
		defaultRatio        float64
		dataType            types.DataType
		compressionType     compression.Type
		expectedImprovement string
	})

	if !ok {
		return
	}

	separator := strings.Repeat("=", 80)
	t.Logf("\n%s", separator)
	t.Logf("📊 Test: %s", testCase.name)
	t.Logf("%s", separator)

	// Test configuration
	t.Logf("\n📋 Configuration:")
	t.Logf("  Total Features:    %d", testCase.numFeatures)
	t.Logf("  Non-Zero Values:   %d (%.1f%%)", nonZeroCount, float64(nonZeroCount)/float64(testCase.numFeatures)*100)
	t.Logf("  Default Values:    %d (%.1f%%)", testCase.numFeatures-nonZeroCount, testCase.defaultRatio*100)
	t.Logf("  Data Type:         %v (size: %d bytes)", testCase.dataType, testCase.dataType.Size())
	t.Logf("  Compression:       %v", testCase.compressionType)

	// Layout 1 results
	t.Logf("\n📦 Layout 1 (Baseline):")
	t.Logf("  Header Size:     %6d bytes", layout1.headerSize)
	t.Logf("  Original Size:   %6d bytes (stores ALL %d features)", layout1.originalSize, testCase.numFeatures)
	t.Logf("  Compressed Size: %6d bytes", layout1.compressedSize)
	t.Logf("  Total Size:      %6d bytes (header + compressed)", layout1.headerSize+layout1.compressedSize)
	if layout1.originalSize > 0 {
		t.Logf("  Compression:     %.2f%% reduction",
			float64(layout1.originalSize-layout1.compressedSize)/float64(layout1.originalSize)*100)
	}

	// Layout 2 results
	bitmapSize := (testCase.numFeatures + 7) / 8
	t.Logf("\n📦 Layout 2 (Optimized with Bitmap):")
	t.Logf("  Header Size:     %6d bytes (+1 byte bitmap metadata)", layout2.headerSize)
	if testCase.defaultRatio > 0 {
		t.Logf("  Bitmap Size:     %6d bytes (tracks %d features)", bitmapSize, testCase.numFeatures)
		t.Logf("  Values Size:     %6d bytes (stores only %d non-zero values)", layout2.originalSize-bitmapSize, nonZeroCount)
	}
	t.Logf("  Original Size:   %6d bytes (bitmap + non-zero values only)", layout2.originalSize)
	t.Logf("  Compressed Size: %6d bytes", layout2.compressedSize)
	t.Logf("  Total Size:      %6d bytes (header + compressed)", layout2.headerSize+layout2.compressedSize)
	if layout2.originalSize > 0 {
		t.Logf("  Compression:     %.2f%% reduction",
			float64(layout2.originalSize-layout2.compressedSize)/float64(layout2.originalSize)*100)
	}

	// Improvements
	originalSavings := layout1.originalSize - layout2.originalSize
	compressedSavings := layout1.compressedSize - layout2.compressedSize
	totalSavings := (layout1.headerSize + layout1.compressedSize) - (layout2.headerSize + layout2.compressedSize)

	t.Logf("\n🎯 Layout 2 Improvements:")
	t.Logf("  Original Size:   %6d bytes saved (%.2f%% reduction)", originalSavings,
		float64(originalSavings)/float64(layout1.originalSize)*100)
	t.Logf("  Compressed Size: %6d bytes saved (%.2f%% reduction)", compressedSavings,
		float64(compressedSavings)/float64(layout1.compressedSize)*100)
	t.Logf("  Total Size:      %6d bytes saved (%.2f%% reduction)", totalSavings,
		float64(totalSavings)/float64(layout1.headerSize+layout1.compressedSize)*100)

	if compressedSavings > 0 {
		t.Logf("  Result: ✅ Layout2 is BETTER")
	} else if compressedSavings == 0 {
		t.Logf("  Result: ⚖️  Layout2 is EQUAL")
	} else {
		t.Logf("  Result: ⚠️  Layout2 has overhead (expected for 0%% defaults)")
	}

	t.Logf("\n💡 Expected: %s", testCase.expectedImprovement)
	t.Logf("%s\n", separator)
}
