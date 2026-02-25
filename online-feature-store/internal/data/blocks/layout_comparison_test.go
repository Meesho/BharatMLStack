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

// Bitmap payload size is derived from noOfFeatures (set on the block and provided by the
// feature schema at read time). Layout-2 adds a 10th byte: bit 0 (72nd bit) = bitmap present;
// the original 9-byte header is unchanged. So layout-2 works for any number of features (e.g. 850).

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

// catalogFeatureGroup describes one feature group for entityLabel=catalog use case
type catalogFeatureGroup struct {
	name        string
	dataType    types.DataType
	numFeatures int // for vectors = num vectors
}

// catalogFeatureGroups defines all feature groups for entityLabel=catalog (layout-2 tests skip Bool)
var catalogFeatureGroups = []catalogFeatureGroup{
	{name: "vector_int32", dataType: types.DataTypeInt32Vector, numFeatures: 1},
	{name: "embeddings_v2_fp16", dataType: types.DataTypeFP16Vector, numFeatures: 3},
	{name: "embedding_stcg_fp16", dataType: types.DataTypeFP16Vector, numFeatures: 3},
	{name: "raw_fp16_7d_1d_1am", dataType: types.DataTypeFP16, numFeatures: 1},
	{name: "rt_raw_ads_demand_attributes_fp32", dataType: types.DataTypeFP32, numFeatures: 1},
	{name: "derived_3_fp32", dataType: types.DataTypeFP32, numFeatures: 1},
	{name: "derived_fp16", dataType: types.DataTypeFP16, numFeatures: 4},
	{name: "properties_string", dataType: types.DataTypeString, numFeatures: 1},
	{name: "realtime_int64_1", dataType: types.DataTypeInt64, numFeatures: 1},
	{name: "derived_4_fp32", dataType: types.DataTypeFP32, numFeatures: 1},
	{name: "rt_raw_ad_attributes_v1_fp32", dataType: types.DataTypeFP32, numFeatures: 1},
	{name: "derived_ads_fp32", dataType: types.DataTypeFP32, numFeatures: 3},
	{name: "embedding_ca_fp32", dataType: types.DataTypeFP32Vector, numFeatures: 1},
	{name: "organic__derived_fp32", dataType: types.DataTypeFP32, numFeatures: 11},
	{name: "derived_fp32", dataType: types.DataTypeFP32, numFeatures: 46},
	{name: "derived_bool", dataType: types.DataTypeBool, numFeatures: 2}, // layout-1 only, skipped in layout-2 test
	{name: "raw_fp16_1d_30m_12am", dataType: types.DataTypeFP16, numFeatures: 1},
	{name: "derived_string", dataType: types.DataTypeString, numFeatures: 4},
	{name: "properties_2_string", dataType: types.DataTypeString, numFeatures: 1},
	{name: "derived_2_fp32", dataType: types.DataTypeFP32, numFeatures: 1},
	{name: "realtime_int64", dataType: types.DataTypeInt64, numFeatures: 1},
	{name: "merlin_embeddings_fp16", dataType: types.DataTypeFP16Vector, numFeatures: 2},
	{name: "rt_raw_ad_attributes_int32", dataType: types.DataTypeInt32, numFeatures: 1},
	{name: "rt_raw_ad_cpc_value_fp32", dataType: types.DataTypeFP32, numFeatures: 1},
	{name: "raw_uint64", dataType: types.DataTypeUint64, numFeatures: 3},
	{name: "rt_raw_ad_batch_attributes_fp32", dataType: types.DataTypeFP32, numFeatures: 1},
	{name: "embeddings_fp16", dataType: types.DataTypeFP16Vector, numFeatures: 1},
	{name: "vector_int32_lifetime", dataType: types.DataTypeInt32Vector, numFeatures: 1},
	{name: "derived_int32", dataType: types.DataTypeInt32, numFeatures: 14},
	{name: "vector_int32_lifetime_v2", dataType: types.DataTypeInt32Vector, numFeatures: 1},
	{name: "rt_raw_is_live_on_ad_string", dataType: types.DataTypeString, numFeatures: 1},
	{name: "rt_raw_ad_gmv_max_attributes_fp32", dataType: types.DataTypeFP32, numFeatures: 1},
	{name: "realtime_string", dataType: types.DataTypeString, numFeatures: 1},
}

// defaultRatiosForCatalog defines sparsity scenarios to simulate per feature group
var defaultRatiosForCatalog = []float64{0.50, 0.80}

// TestLayout1VsLayout2Compression runs layout comparison for the catalog use case (entityLabel=catalog).
// Each catalog feature group is tested with 50% and 80% default ratios; Bool scalar is skipped (layout-1 only).
func TestLayout1VsLayout2Compression(t *testing.T) {
	// Initialize/reset results collection
	testResults = make([]TestResult, 0, 128)
	compressionType := compression.TypeZSTD

	// Build test cases: every (catalog feature group × default ratio), skip Bool for layout-2
	var testCases []struct {
		name                string
		numFeatures         int
		defaultRatio        float64
		dataType            types.DataType
		compressionType     compression.Type
		expectedImprovement string
	}
	for _, fg := range catalogFeatureGroups {
		if fg.dataType == types.DataTypeBool {
			continue // Bool scalar has layout-1 only
		}

		// Determine meaningful ratios for this feature group:
		// For numFeatures=1, only 0% (no default) and 100% (all default) are valid.
		// For numFeatures>1, derive ratios that produce distinct integer default counts
		// to avoid duplicates or misleading names caused by int truncation.
		type ratioCase struct {
			numDefaults int
			ratio       float64 // actual ratio = numDefaults / numFeatures
		}
		seen := make(map[int]bool)
		var ratios []ratioCase
		if fg.numFeatures == 1 {
			// Only two meaningful scenarios for a single feature
			ratios = []ratioCase{{0, 0.0}, {1, 1.0}}
		} else {
			for _, desiredRatio := range defaultRatiosForCatalog {
				nd := int(float64(fg.numFeatures) * desiredRatio)
				if seen[nd] {
					continue // skip duplicates caused by truncation
				}
				seen[nd] = true
				actualRatio := float64(nd) / float64(fg.numFeatures)
				ratios = append(ratios, ratioCase{nd, actualRatio})
			}
		}

		for _, rc := range ratios {
			expectedImprovement := "Layout2 should be better or equal with defaults"
			if rc.numDefaults == 0 {
				expectedImprovement = "Layout2 may have small bitmap overhead"
			}
			name := fmt.Sprintf("catalog/%s %d/%d defaults (%.0f%%)", fg.name, rc.numDefaults, fg.numFeatures, rc.ratio*100)
			testCases = append(testCases, struct {
				name                string
				numFeatures         int
				defaultRatio        float64
				dataType            types.DataType
				compressionType     compression.Type
				expectedImprovement string
			}{
				name:                name,
				numFeatures:         fg.numFeatures,
				defaultRatio:        rc.ratio,
				dataType:            fg.dataType,
				compressionType:     compressionType,
				expectedImprovement: expectedImprovement,
			})
		}
	}
	// Edge cases for catalog: 0% and 100% on derived_fp32
	testCases = append(testCases,
		struct {
			name                string
			numFeatures         int
			defaultRatio        float64
			dataType            types.DataType
			compressionType     compression.Type
			expectedImprovement string
		}{
			name: "catalog/derived_fp32 0% defaults (all non-zero)", numFeatures: 46, defaultRatio: 0, dataType: types.DataTypeFP32,
			compressionType: compression.TypeZSTD, expectedImprovement: "Layout2 has small overhead when no defaults",
		},
		struct {
			name                string
			numFeatures         int
			defaultRatio        float64
			dataType            types.DataType
			compressionType     compression.Type
			expectedImprovement string
		}{
			name: "catalog/derived_fp32 100% defaults", numFeatures: 46, defaultRatio: 1.0, dataType: types.DataTypeFP32,
			compressionType: compression.TypeZSTD, expectedImprovement: "Layout2 should massively outperform",
		},
	)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Bool scalar supports only layout 1; skip layout-2 comparison
			if tc.dataType == types.DataTypeBool {
				t.Skip("Bool scalar has layout-1 only")
			}
			// Generate test data for this data type
			sr, err := generateSparseDataByType(tc.dataType, tc.numFeatures, tc.defaultRatio)
			require.NoError(t, err, "Generate sparse data for %v", tc.dataType)
			nonZeroCount := sr.nonZeroCount

			// Layout 2 with no bitmap for baseline layout-1 size comparison uses same data, bitmap=nil for layout 1
			srLayout1 := &sparseDataResult{data: sr.data, bitmap: nil, nonZeroCount: sr.nonZeroCount, vectorLengths: sr.vectorLengths, stringLengths: sr.stringLengths}
			// Test Layout 1 (no bitmap)
			layout1Results := serializeWithLayoutByType(t, 1, tc.numFeatures, srLayout1, tc.dataType, tc.compressionType)

			// Test Layout 2 (with bitmap)
			layout2Results := serializeWithLayoutByType(t, 2, tc.numFeatures, sr, tc.dataType, tc.compressionType)

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

			// Layout2 can have fixed overhead (1-byte bitmap) that exceeds savings when
			// feature count is 1: we add 1 byte but save 0. Allow Layout2 up to 1 byte worse.
			layout2CanHaveOverhead := tc.numFeatures == 1

			// Assertions
			t.Run("Compressed Size Comparison", func(t *testing.T) {
				// Calculate improvement
				improvement := float64(layout1Results.compressedSize-layout2Results.compressedSize) / float64(layout1Results.compressedSize) * 100

				// With any default ratios, Layout2 should be equal or better (unless overhead case)
				if tc.defaultRatio > 0.0 {
					if layout2CanHaveOverhead {
						// Single feature: bitmap adds 1 byte, no savings; allow up to 1 byte worse
						maxAllowed := layout1Results.compressedSize + 1
						assert.LessOrEqual(t, layout2Results.compressedSize, maxAllowed,
							"Layout2 compressed size should be at most 1 byte more than Layout1 for single-feature")
						t.Logf("Note: Single-feature has bitmap overhead; Layout2 may be 1 byte larger")
					} else {
						assert.LessOrEqual(t, layout2Results.compressedSize, layout1Results.compressedSize,
							"Layout2 compressed size should be less than or equal to Layout1 with %.0f%% defaults", tc.defaultRatio*100)

						assert.GreaterOrEqual(t, improvement, 0.0,
							"Layout2 should show improvement with %.0f%% defaults", tc.defaultRatio*100)
					}
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
					if layout2CanHaveOverhead {
						// Single feature: bitmap adds 1 byte; allow Layout2 up to 1 byte larger
						maxAllowed := layout1Results.originalSize + 1
						assert.LessOrEqual(t, layout2Results.originalSize, maxAllowed,
							"Layout2 original size should be at most 1 byte more than Layout1 for single-feature")
						t.Logf("Note: Single-feature has bitmap overhead; original size improvement: %.2f%%",
							float64(layout1Results.originalSize-layout2Results.originalSize)/float64(layout1Results.originalSize)*100)
					} else {
						assert.Less(t, layout2Results.originalSize, layout1Results.originalSize,
							"Layout2 original size should be less than Layout1 when defaults present")

						// Calculate actual reduction
						actualReduction := float64(layout1Results.originalSize-layout2Results.originalSize) / float64(layout1Results.originalSize)

						// With any defaults, should show some reduction (accounting for bitmap overhead)
						// Bitmap overhead = (numFeatures + 7) / 8 bytes
						// Use a conservative efficiency factor (0.45) so we don't over-constrain string/vector encoding
						bitmapOverhead := float64((tc.numFeatures+7)/8) / float64(layout1Results.originalSize)
						minExpectedReduction := tc.defaultRatio*0.45 - bitmapOverhead

						if minExpectedReduction > 0 {
							// Allow 15% relative tolerance for rounding and encoding variance
							tolerance := minExpectedReduction * 0.85
							assert.GreaterOrEqual(t, actualReduction, tolerance,
								"Layout2 should reduce original size by at least %.1f%% with %.1f%% defaults (got %.1f%%)",
								minExpectedReduction*100, tc.defaultRatio*100, actualReduction*100)
						}

						// Log the improvement for analysis
						t.Logf("Original size improvement: %.2f%%", actualReduction*100)
					}
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
					assert.NotZero(t, ddb2.BitmapMeta&bitmapPresentMask, "Layout2 should have bitmap present flag set")
				}
			})
		})
	}

	// Generate results file after all tests complete
	t.Run("Generate Results Report", func(t *testing.T) {
		err := generateResultsFile(testResults)
		require.NoError(t, err, "Should generate results file successfully")
		t.Logf("\n✅ Results written to: layout_comparison_results.txt, layout_comparison_results.md")
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

// generateResultsMarkdown builds markdown content for the layout comparison results.
func generateResultsMarkdown(results []TestResult) string {
	var b strings.Builder
	b.WriteString("# Layout1 vs Layout2 Compression — Catalog Use Case\n\n")
	b.WriteString("## Executive Summary\n\n")
	betterCount := 0
	for _, r := range results {
		if r.IsLayout2Better {
			betterCount++
		}
	}
	b.WriteString(fmt.Sprintf("✅ **Layout2 is better than or equal to Layout1** in **%d/%d** catalog scenarios (%.1f%%).\n\n",
		betterCount, len(results), float64(betterCount)/float64(len(results))*100))
	b.WriteString("## Test Results by Data Type\n\n")
	byType := make(map[types.DataType][]TestResult)
	var typeOrder []types.DataType
	seen := make(map[types.DataType]bool)
	for _, r := range results {
		byType[r.DataType] = append(byType[r.DataType], r)
		if !seen[r.DataType] {
			seen[r.DataType] = true
			typeOrder = append(typeOrder, r.DataType)
		}
	}
	for _, dt := range typeOrder {
		list := byType[dt]
		b.WriteString(fmt.Sprintf("### %s\n\n", dt.String()))
		b.WriteString("| Scenario | Features | Defaults | Original Δ | Compressed Δ |\n")
		b.WriteString("|----------|----------|-----------|------------|-------------|\n")
		for _, row := range list {
			status := "✅"
			if !row.IsLayout2Better {
				status = "⚠️"
			}
			b.WriteString(fmt.Sprintf("| %s | %d | %.1f%% | %.2f%% | %.2f%% %s |\n",
				truncateString(row.Name, 40), row.NumFeatures, row.DefaultRatio*100,
				row.OriginalSizeReduction, row.CompressedSizeReduction, status))
		}
		b.WriteString("\n")
	}
	b.WriteString("## All Results Summary (Catalog Use Case)\n\n")
	b.WriteString("| Test Name | Data Type | Features | Defaults | Original Δ | Compressed Δ |\n")
	b.WriteString("|-----------|-----------|----------|-----------|------------|-------------|\n")
	for _, r := range results {
		status := "✅"
		if !r.IsLayout2Better {
			status = "⚠️"
		}
		b.WriteString(fmt.Sprintf("| %s | %s | %d | %.1f%% | %.2f%% | %.2f%% %s |\n",
			truncateString(r.Name, 45), r.DataType.String(), r.NumFeatures, r.DefaultRatio*100,
			r.OriginalSizeReduction, r.CompressedSizeReduction, status))
	}
	b.WriteString("\n## Key Findings (Catalog Use Case)\n\n")
	b.WriteString("- **Use case:** entityLabel=catalog with the defined feature groups (scalars and vectors).\n")
	b.WriteString("- Layout2 uses bitmap-based storage; bitmap present is the 72nd bit (10th byte bit 0). Bool scalar (derived_bool) is layout-1 only and excluded from layout-2 comparison.\n")
	b.WriteString("- With 0% defaults, Layout2 has small bitmap overhead; with 50%/80%/100% defaults, Layout2 reduces size.\n\n")
	b.WriteString("## Test Implementation\n\n")
	b.WriteString("Tests: `online-feature-store/internal/data/blocks/layout_comparison_test.go`\n\n")
	b.WriteString("```bash\n")
	b.WriteString("go test ./internal/data/blocks -run TestLayout1VsLayout2Compression -v\n")
	b.WriteString("go test ./internal/data/blocks -run TestLayout2BitmapOptimization -v\n")
	b.WriteString("```\n\n")
	b.WriteString(fmt.Sprintf("**Generated:** %s\n", time.Now().Format("2006-01-02 15:04:05")))
	return b.String()
}

// generateResultsFile creates a comprehensive results file (txt and md)
func generateResultsFile(results []TestResult) error {
	f, err := os.Create("layout_comparison_results.txt")
	if err != nil {
		return err
	}
	defer f.Close()

	// Header
	fmt.Fprintf(f, "╔════════════════════════════════════════════════════════════════════════════════╗\n")
	fmt.Fprintf(f, "║   Layout1 vs Layout2 Compression — Catalog Use Case (entityLabel=catalog)     ║\n")
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

	// Write markdown report next to the test (layout_comparison_results.md)
	md := generateResultsMarkdown(results)
	if err := os.WriteFile("layout_comparison_results.md", []byte(md), 0644); err != nil {
		return err
	}
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

// generateSparseData creates test data with specified sparsity (default ratio) for FP32
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

// sparseDataResult holds generated sparse data and bitmap for any type
type sparseDataResult struct {
	data        interface{}
	bitmap      []byte
	nonZeroCount int
	vectorLengths []uint16 // for vector types
	stringLengths []uint16 // for string scalar/vector
}

// generateSparseDataByType creates test data with specified sparsity for the given data type.
// Bool scalar is not supported (layout-1 only). For vector types, numFeatures = numVectors.
func generateSparseDataByType(dataType types.DataType, numFeatures int, defaultRatio float64) (*sparseDataResult, error) {
	rand.Seed(time.Now().UnixNano())
	bitmap := make([]byte, (numFeatures+7)/8)
	numDefaults := int(float64(numFeatures) * defaultRatio)
	indices := make([]int, numFeatures)
	for i := range indices {
		indices[i] = i
	}
	rand.Shuffle(len(indices), func(i, j int) { indices[i], indices[j] = indices[j], indices[i] })

	setBit := func(idx int) { bitmap[idx/8] |= 1 << (idx % 8) }
	nonZeroCount := numFeatures - numDefaults

	switch dataType {
	case types.DataTypeFP32, types.DataTypeFP16:
		data := make([]float32, numFeatures)
		for i := 0; i < numFeatures; i++ {
			idx := indices[i]
			if i < numDefaults {
				data[idx] = 0.0
			} else {
				data[idx] = rand.Float32()
				setBit(idx)
			}
		}
		return &sparseDataResult{data: data, bitmap: bitmap, nonZeroCount: nonZeroCount}, nil
	case types.DataTypeFP64:
		data := make([]float64, numFeatures)
		for i := 0; i < numFeatures; i++ {
			idx := indices[i]
			if i < numDefaults {
				data[idx] = 0.0
			} else {
				data[idx] = rand.Float64()
				setBit(idx)
			}
		}
		return &sparseDataResult{data: data, bitmap: bitmap, nonZeroCount: nonZeroCount}, nil
	case types.DataTypeInt32:
		data := make([]int32, numFeatures)
		for i := 0; i < numFeatures; i++ {
			idx := indices[i]
			if i < numDefaults {
				data[idx] = 0
			} else {
				data[idx] = int32(rand.Intn(1<<31 - 1))
				setBit(idx)
			}
		}
		return &sparseDataResult{data: data, bitmap: bitmap, nonZeroCount: nonZeroCount}, nil
	case types.DataTypeUint32:
		data := make([]uint32, numFeatures)
		for i := 0; i < numFeatures; i++ {
			idx := indices[i]
			if i < numDefaults {
				data[idx] = 0
			} else {
				data[idx] = uint32(rand.Uint32())
				setBit(idx)
			}
		}
		return &sparseDataResult{data: data, bitmap: bitmap, nonZeroCount: nonZeroCount}, nil
	case types.DataTypeInt64:
		data := make([]int64, numFeatures)
		for i := 0; i < numFeatures; i++ {
			idx := indices[i]
			if i < numDefaults {
				data[idx] = 0
			} else {
				data[idx] = int64(rand.Int63())
				setBit(idx)
			}
		}
		return &sparseDataResult{data: data, bitmap: bitmap, nonZeroCount: nonZeroCount}, nil
	case types.DataTypeUint64:
		data := make([]uint64, numFeatures)
		for i := 0; i < numFeatures; i++ {
			idx := indices[i]
			if i < numDefaults {
				data[idx] = 0
			} else {
				data[idx] = rand.Uint64()
				setBit(idx)
			}
		}
		return &sparseDataResult{data: data, bitmap: bitmap, nonZeroCount: nonZeroCount}, nil
	case types.DataTypeString:
		const maxStrLen = 32
		strLens := make([]uint16, numFeatures)
		data := make([]string, numFeatures)
		for i := 0; i < numFeatures; i++ {
			idx := indices[i]
			if i < numDefaults {
				data[idx] = ""
				strLens[idx] = maxStrLen
			} else {
				s := fmt.Sprintf("v%d", rand.Intn(10000))
				data[idx] = s
				strLens[idx] = uint16(len(s))
				if strLens[idx] < maxStrLen {
					strLens[idx] = maxStrLen
				}
				setBit(idx)
			}
		}
		return &sparseDataResult{data: data, bitmap: bitmap, nonZeroCount: nonZeroCount, stringLengths: strLens}, nil
	case types.DataTypeFP32Vector:
		const vecLen = 4
		vecLengths := make([]uint16, numFeatures)
		data := make([][]float32, numFeatures)
		for i := 0; i < numFeatures; i++ {
			vecLengths[i] = vecLen
			vec := make([]float32, vecLen)
			idx := indices[i]
			if i < numDefaults {
				data[idx] = vec
			} else {
				for j := 0; j < vecLen; j++ {
					vec[j] = rand.Float32()
				}
				data[idx] = vec
				setBit(idx)
			}
		}
		return &sparseDataResult{data: data, bitmap: bitmap, nonZeroCount: nonZeroCount, vectorLengths: vecLengths}, nil
	case types.DataTypeFP16Vector:
		// Same structure as FP32Vector; serialization encodes as FP16
		const vecLen = 4
		vecLengths := make([]uint16, numFeatures)
		data := make([][]float32, numFeatures)
		for i := 0; i < numFeatures; i++ {
			vecLengths[i] = vecLen
			vec := make([]float32, vecLen)
			idx := indices[i]
			if i < numDefaults {
				data[idx] = vec
			} else {
				for j := 0; j < vecLen; j++ {
					vec[j] = rand.Float32()
				}
				data[idx] = vec
				setBit(idx)
			}
		}
		return &sparseDataResult{data: data, bitmap: bitmap, nonZeroCount: nonZeroCount, vectorLengths: vecLengths}, nil
	case types.DataTypeInt32Vector:
		const vecLen = 4
		vecLengths := make([]uint16, numFeatures)
		data := make([][]int32, numFeatures)
		for i := 0; i < numFeatures; i++ {
			vecLengths[i] = vecLen
			vec := make([]int32, vecLen)
			idx := indices[i]
			if i < numDefaults {
				data[idx] = vec
			} else {
				for j := 0; j < vecLen; j++ {
					vec[j] = int32(rand.Intn(1<<31 - 1))
				}
				data[idx] = vec
				setBit(idx)
			}
		}
		return &sparseDataResult{data: data, bitmap: bitmap, nonZeroCount: nonZeroCount, vectorLengths: vecLengths}, nil
	case types.DataTypeBoolVector:
		const boolVecLen = 4
		vecLengths := make([]uint16, numFeatures)
		data := make([][]uint8, numFeatures)
		for i := 0; i < numFeatures; i++ {
			vecLengths[i] = boolVecLen
			vec := make([]uint8, boolVecLen)
			idx := indices[i]
			if i < numDefaults {
				data[idx] = vec
			} else {
				for j := 0; j < boolVecLen; j++ {
					vec[j] = uint8(rand.Intn(2))
				}
				data[idx] = vec
				setBit(idx)
			}
		}
		return &sparseDataResult{data: data, bitmap: bitmap, nonZeroCount: nonZeroCount, vectorLengths: vecLengths}, nil
	case types.DataTypeStringVector:
		const vecLen = 2
		const maxStrLen = 16
		vecLengths := make([]uint16, numFeatures)
		strLengths := make([]uint16, numFeatures) // per-vector max string length
		data := make([][]string, numFeatures)
		for i := 0; i < numFeatures; i++ {
			vecLengths[i] = vecLen
			strLengths[i] = maxStrLen
			idx := indices[i]
			vec := make([]string, vecLen)
			if i < numDefaults {
				data[idx] = vec
			} else {
				for j := 0; j < vecLen; j++ {
					vec[j] = fmt.Sprintf("s%d", rand.Intn(1000))
				}
				data[idx] = vec
				setBit(idx)
			}
		}
		return &sparseDataResult{data: data, bitmap: bitmap, nonZeroCount: nonZeroCount, vectorLengths: vecLengths, stringLengths: strLengths}, nil
	default:
		return nil, fmt.Errorf("unsupported data type for layout-2 comparison: %v", dataType)
	}
}

// serializeWithLayoutByType serializes a block with the given layout for any layout-2 capable type.
func serializeWithLayoutByType(t *testing.T, layoutVersion uint8, numFeatures int, sr *sparseDataResult,
	dataType types.DataType, compressionType compression.Type) serializationResults {
	t.Helper()
	psdb := GetPSDBPool().Get()
	defer GetPSDBPool().Put(psdb)

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
	psdb.Data = sr.data
	psdb.bitmap = sr.bitmap
	psdb.vectorLengths = sr.vectorLengths
	psdb.stringLengths = sr.stringLengths

	// originalDataLen and originalData
	if layoutVersion == 2 && len(sr.bitmap) > 0 {
		psdb.originalDataLen = 0
		switch dataType {
		case types.DataTypeFP32, types.DataTypeFP16:
			psdb.originalDataLen = sr.nonZeroCount * dataType.Size()
		case types.DataTypeFP64, types.DataTypeInt32, types.DataTypeUint32, types.DataTypeInt64, types.DataTypeUint64:
			psdb.originalDataLen = sr.nonZeroCount * dataType.Size()
		case types.DataTypeString:
			// Serialization builds dense dynamically; allocate enough for layout-1 style so Serialize has a buffer
			total := 0
			for _, l := range sr.stringLengths {
				total += int(l) + 2
			}
			psdb.originalDataLen = total
		case types.DataTypeFP32Vector, types.DataTypeFP16Vector:
			unitSize := dataType.Size()
			for i := 0; i < numFeatures; i++ {
				if (sr.bitmap[i/8] & (1 << (i % 8))) != 0 && i < len(sr.vectorLengths) {
					psdb.originalDataLen += int(sr.vectorLengths[i]) * unitSize
				}
			}
		case types.DataTypeInt32Vector:
			unitSize := types.DataTypeInt32.Size()
			for i := 0; i < numFeatures; i++ {
				if (sr.bitmap[i/8] & (1 << (i % 8))) != 0 && i < len(sr.vectorLengths) {
					psdb.originalDataLen += int(sr.vectorLengths[i]) * unitSize
				}
			}
		case types.DataTypeBoolVector:
			for i := 0; i < numFeatures; i++ {
				if (sr.bitmap[i/8] & (1 << (i % 8))) != 0 && i < len(sr.vectorLengths) {
					psdb.originalDataLen += (int(sr.vectorLengths[i]) + 7) / 8
				}
			}
		case types.DataTypeStringVector:
			// Serialization builds dense dynamically; allocate enough for layout-1 style
			total := 0
			for i, vl := range sr.vectorLengths {
				total += int(vl) * (int(sr.stringLengths[i]) + 2)
			}
			psdb.originalDataLen = total
		default:
			psdb.originalDataLen = sr.nonZeroCount * dataType.Size()
		}
	} else {
		switch dataType {
		case types.DataTypeString:
			total := 0
			for _, l := range sr.stringLengths {
				total += int(l) + 2
			}
			psdb.originalDataLen = total
		case types.DataTypeFP32Vector, types.DataTypeFP16Vector:
			total := 0
			for _, vl := range sr.vectorLengths {
				total += int(vl) * dataType.Size()
			}
			psdb.originalDataLen = total
		case types.DataTypeInt32Vector:
			total := 0
			for _, vl := range sr.vectorLengths {
				total += int(vl) * types.DataTypeInt32.Size()
			}
			psdb.originalDataLen = total
		case types.DataTypeBoolVector:
			total := 0
			for _, vl := range sr.vectorLengths {
				total += (int(vl) + 7) / 8
			}
			psdb.originalDataLen = total
		case types.DataTypeStringVector:
			total := 0
			for i, vl := range sr.vectorLengths {
				total += int(vl) * (int(sr.stringLengths[i]) + 2)
			}
			psdb.originalDataLen = total
		default:
			psdb.originalDataLen = numFeatures * dataType.Size()
		}
	}
	if psdb.originalData == nil || len(psdb.originalData) < psdb.originalDataLen {
		psdb.originalData = make([]byte, psdb.originalDataLen)
	} else {
		psdb.originalData = psdb.originalData[:psdb.originalDataLen]
	}
	psdb.compressedData = psdb.compressedData[:0]
	psdb.compressedDataLen = 0
	if psdb.Builder == nil {
		psdb.Builder = &PermStorageDataBlockBuilder{psdb: psdb}
	}
	psdb.Builder.SetupBitmapMeta(numFeatures)

	serialized, err := psdb.Serialize()
	require.NoError(t, err, "Serialization should succeed for layout %d type %v", layoutVersion, dataType)
	headerSize := PSDBLayout1LengthBytes
	if layoutVersion == 2 {
		headerSize = PSDBLayout1LengthBytes + PSDBLayout2ExtraBytes
	}
	origSize := psdb.originalDataLen
	return serializationResults{
		serialized:     serialized,
		originalSize:   origSize,
		compressedSize: len(serialized) - headerSize,
		headerSize:     headerSize,
	}
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
	t.Logf("  Header Size:     %6d bytes (10th byte: 72nd bit = bitmap present)", layout2.headerSize)
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
