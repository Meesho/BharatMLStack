package maths

import (
	"math"
	"testing"
)

// Unit tests for IncrementLogFreqQ20 and IncrementLogFreqQ4_16 functions
//
// Key insights about the logarithmic quantized increment behavior:
//
// IncrementLogFreqQ20 (Q12.8 format):
// 1. Uses Q12.8 fixed-point format (12 integer bits, 8 fractional bits)
// 2. Logarithmic scaling means larger values get exponentially smaller increments
// 3. For very large values (>= 4.0 in Q12.8), increments can be quantized to zero
// 4. Upper 12 bits of the input uint32 are preserved untouched
// 5. Maximum value is capped at 0xFFFFF (20-bit mask)
//
// IncrementLogFreqQ4_16 (Q4.16 format):
// 1. Uses Q4.16 fixed-point format (4 integer bits, 16 fractional bits)
// 2. Higher precision than Q12.8, providing larger and more precise increments
// 3. Logarithmic quantization still occurs but at higher values (>= 8.0 in Q4.16)
// 4. Same upper 12 bits preservation and 20-bit mask as Q12.8 version
// 5. Better performance (~30ns vs ~45ns per operation)

func TestIncrementLogFreqQ20_BasicIncrement(t *testing.T) {
	// Test basic increment functionality
	initial := uint32(0x100) // Small initial value in Q12.8 format
	result := IncrementLogFreqQ20(initial)

	if result <= initial {
		t.Errorf("Expected increment, got result %d <= initial %d", result, initial)
	}

	// Extract the 20-bit log-freq value
	initialFreq := initial & maskQ20
	resultFreq := result & maskQ20

	if resultFreq <= initialFreq {
		t.Errorf("Expected frequency increment, got %d <= %d", resultFreq, initialFreq)
	}
}

func TestIncrementLogFreqQ20_UpperBitsPreservation(t *testing.T) {
	// Test that upper 12 bits are preserved
	testCases := []uint32{
		0x12300100, // Upper bits: 0x123
		0xFFF00200, // Upper bits: 0xFFF
		0x00000300, // Upper bits: 0x000
		0xABC00400, // Upper bits: 0xABC
	}

	for _, input := range testCases {
		result := IncrementLogFreqQ20(input)

		// Check upper 12 bits are preserved
		upperBitsInput := input >> 20
		upperBitsResult := result >> 20

		if upperBitsInput != upperBitsResult {
			t.Errorf("Upper bits not preserved: input upper=0x%X, result upper=0x%X",
				upperBitsInput, upperBitsResult)
		}
	}
}

func TestIncrementLogFreqQ20_ZeroValue(t *testing.T) {
	// Test increment from zero
	initial := uint32(0)
	result := IncrementLogFreqQ20(initial)

	if result == 0 {
		t.Error("Expected non-zero result when incrementing from zero")
	}

	// Verify it's a reasonable increment for y=0 case
	expectedDelta := logFactor // When y=0, math.Pow(logBase, 0) = 1
	expectedRaw := uint32(expectedDelta * q8Scale)
	actualRaw := result & maskQ20

	// Allow some tolerance for floating point precision
	tolerance := uint32(2)
	if actualRaw < expectedRaw-tolerance || actualRaw > expectedRaw+tolerance {
		t.Errorf("Zero increment mismatch: expected ~%d, got %d", expectedRaw, actualRaw)
	}
}

func TestIncrementLogFreqQ20_MaxValueCapping(t *testing.T) {
	// Test overflow protection at maximum value
	maxInput := uint32(0x12300000) | maxQ20 // Upper bits + max 20-bit value
	result := IncrementLogFreqQ20(maxInput)

	// Should still be at max
	resultFreq := result & maskQ20
	if resultFreq != maxQ20 {
		t.Errorf("Expected max value capping: got %d, expected %d", resultFreq, maxQ20)
	}

	// Upper bits should be preserved
	if result>>20 != 0x123 {
		t.Error("Upper bits not preserved at max value")
	}
}

func TestIncrementLogFreqQ20_NearMaxValueCapping(t *testing.T) {
	// Test values near maximum - they may or may not overflow depending on the logarithmic increment
	nearMaxInput := uint32(0xFFFE0) // Very close to maxQ20
	result := IncrementLogFreqQ20(nearMaxInput)

	resultFreq := result & maskQ20
	// For very large values, the logarithmic increment might not cause overflow
	// Just verify we don't exceed the maximum
	if resultFreq > maxQ20 {
		t.Errorf("Result exceeded maximum: got %d, max %d", resultFreq, maxQ20)
	}

	// Verify result is >= input (should increment unless at max)
	inputFreq := nearMaxInput & maskQ20
	if resultFreq < inputFreq {
		t.Errorf("Result decreased: got %d, input was %d", resultFreq, inputFreq)
	}
}

func TestIncrementLogFreqQ20_LogarithmicProperty(t *testing.T) {
	// Test that smaller values get larger increments (logarithmic property)
	smallValue := uint32(0x100)  // Small initial value
	largeValue := uint32(0x8000) // Larger initial value

	smallResult := IncrementLogFreqQ20(smallValue)
	largeResult := IncrementLogFreqQ20(largeValue)

	smallIncrement := (smallResult & maskQ20) - (smallValue & maskQ20)
	largeIncrement := (largeResult & maskQ20) - (largeValue & maskQ20)

	// Smaller values should get larger increments due to logarithmic nature
	if smallIncrement <= largeIncrement {
		t.Errorf("Expected larger increment for smaller values: small=%d, large=%d",
			smallIncrement, largeIncrement)
	}
}

func TestIncrementLogFreqQ20_MultipleIncrements(t *testing.T) {
	// Test multiple successive increments
	value := uint32(0x200)

	// Apply multiple increments and verify each one
	for i := 0; i < 10; i++ {
		prev := value
		value = IncrementLogFreqQ20(value)

		if value <= prev {
			t.Errorf("Increment %d failed: %d <= %d", i, value, prev)
		}

		// Verify we don't exceed maximum
		if (value & maskQ20) > maxQ20 {
			t.Errorf("Increment %d exceeded maximum: %d > %d", i, value&maskQ20, maxQ20)
		}
	}
}

func TestIncrementLogFreqQ20_QuantizationBehavior(t *testing.T) {
	// Test quantization behavior with specific Q12.8 values
	testCases := []struct {
		name          string
		input         uint32
		expectedDelta uint32
		tolerance     uint32
	}{
		{"Q12.8 value 1.0", 0x100, 11, 1}, // 1.0 in Q12.8 gives ~11
		{"Q12.8 value 2.0", 0x200, 1, 1},  // 2.0 in Q12.8 gives ~1
		{"Q12.8 value 4.0", 0x400, 0, 0},  // 4.0 in Q12.8 gives 0 (quantized to zero)
		{"Q12.8 value 8.0", 0x800, 0, 0},  // 8.0 in Q12.8 gives 0 (quantized to zero)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IncrementLogFreqQ20(tc.input)
			delta := (result & maskQ20) - (tc.input & maskQ20)

			if delta < tc.expectedDelta-tc.tolerance || delta > tc.expectedDelta+tc.tolerance {
				t.Errorf("Increment mismatch: got %d, expected %d±%d", delta, tc.expectedDelta, tc.tolerance)
			}
		})
	}
}

func TestIncrementLogFreqQ20_FixedPointAccuracy(t *testing.T) {
	// Test the fixed-point arithmetic accuracy
	input := uint32(0x300) // 3.0 in Q12.8
	result := IncrementLogFreqQ20(input)

	// Calculate expected result manually
	y := float64(input&maskQ20) / q8Scale
	expectedDelta := logFactor / math.Pow(logBase, y)
	expectedY := y + expectedDelta
	expectedRaw := uint32(expectedY * q8Scale)

	actualRaw := result & maskQ20

	// Allow for rounding errors in fixed-point conversion
	tolerance := uint32(2)
	if actualRaw < expectedRaw-tolerance || actualRaw > expectedRaw+tolerance {
		t.Errorf("Fixed-point accuracy test failed: expected ~%d, got %d", expectedRaw, actualRaw)
	}
}

func TestIncrementLogFreqQ20_EdgeCases(t *testing.T) {
	// Test various edge cases
	edgeCases := []struct {
		name            string
		input           uint32
		expectIncrement bool
	}{
		{"Minimum non-zero", 0x1, true},
		{"Power of 2 boundary", 0x100, true},
		{"Mid-range value", 0x8000, false},     // Large values may have zero increment due to quantization
		{"High value", 0xF0000, false},         // Very large values likely have zero increment
		{"Just before max", maxQ20 - 1, false}, // Near max may have zero increment
	}

	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IncrementLogFreqQ20(tc.input)

			// Basic sanity checks
			if (result & maskQ20) > maxQ20 {
				t.Error("Result exceeded maximum allowed value")
			}

			inputFreq := tc.input & maskQ20
			resultFreq := result & maskQ20

			if tc.expectIncrement {
				if resultFreq <= inputFreq {
					t.Error("Expected increment for this input value")
				}
			} else {
				// For large values, increment may be zero due to logarithmic nature
				if resultFreq < inputFreq {
					t.Error("Result should not decrease")
				}
			}
		})
	}
}

// Tests for IncrementLogFreqQ4_16 function (Q4.16 fixed-point format)

func TestIncrementLogFreqQ4_16_BasicIncrement(t *testing.T) {
	// Test basic increment functionality
	initial := uint32(0x10000) // 1.0 in Q4.16 format
	result := IncrementLogFreqQ4_16(initial)

	if result <= initial {
		t.Errorf("Expected increment, got result %d <= initial %d", result, initial)
	}

	// Extract the 20-bit log-freq value
	initialFreq := initial & maskQ20
	resultFreq := result & maskQ20

	if resultFreq <= initialFreq {
		t.Errorf("Expected frequency increment, got %d <= %d", resultFreq, initialFreq)
	}
}

func TestIncrementLogFreqQ4_16_UpperBitsPreservation(t *testing.T) {
	// Test that upper 12 bits are preserved
	testCases := []uint32{
		0x12310000, // Upper bits: 0x123, value 1.0 in Q4.16
		0xFFF20000, // Upper bits: 0xFFF, value 2.0 in Q4.16
		0x00030000, // Upper bits: 0x000, value 3.0 in Q4.16
		0xABC40000, // Upper bits: 0xABC, value 4.0 in Q4.16
	}

	for _, input := range testCases {
		result := IncrementLogFreqQ4_16(input)

		// Check upper 12 bits are preserved
		upperBitsInput := input >> 20
		upperBitsResult := result >> 20

		if upperBitsInput != upperBitsResult {
			t.Errorf("Upper bits not preserved: input upper=0x%X, result upper=0x%X",
				upperBitsInput, upperBitsResult)
		}
	}
}

func TestIncrementLogFreqQ4_16_ZeroValue(t *testing.T) {
	// Test increment from zero
	initial := uint32(0)
	result := IncrementLogFreqQ4_16(initial)

	if result == 0 {
		t.Error("Expected non-zero result when incrementing from zero")
	}

	// Verify it's a reasonable increment for y=0 case
	expectedDelta := logFactor // When y=0, math.Pow(logBase, 0) = 1
	expectedRaw := uint32(expectedDelta * q16Scale)
	actualRaw := result & maskQ20

	// Allow some tolerance for floating point precision
	tolerance := uint32(100) // Larger tolerance due to higher precision
	if actualRaw < expectedRaw-tolerance || actualRaw > expectedRaw+tolerance {
		t.Errorf("Zero increment mismatch: expected ~%d, got %d", expectedRaw, actualRaw)
	}
}

func TestIncrementLogFreqQ4_16_MaxValueCapping(t *testing.T) {
	// Test overflow protection at maximum value
	maxInput := uint32(0x12300000) | maxQ20 // Upper bits + max 20-bit value
	result := IncrementLogFreqQ4_16(maxInput)

	// Should still be at max
	resultFreq := result & maskQ20
	if resultFreq != maxQ20 {
		t.Errorf("Expected max value capping: got %d, expected %d", resultFreq, maxQ20)
	}

	// Upper bits should be preserved
	if result>>20 != 0x123 {
		t.Error("Upper bits not preserved at max value")
	}
}

func TestIncrementLogFreqQ4_16_QuantizationBehavior(t *testing.T) {
	// Test quantization behavior with specific Q4.16 values
	// Q4.16 has much higher precision than Q12.8, so increments should be larger
	testCases := []struct {
		name          string
		input         uint32
		expectedDelta uint32
		tolerance     uint32
	}{
		{"Q4.16 value 1.0", 0x10000, 2845, 50}, // 1.0 in Q4.16 gives ~2845
		{"Q4.16 value 2.0", 0x20000, 284, 20},  // 2.0 in Q4.16 gives ~284
		{"Q4.16 value 4.0", 0x40000, 2, 2},     // 4.0 in Q4.16 gives ~2
		{"Q4.16 value 8.0", 0x80000, 0, 0},     // 8.0 in Q4.16 gives 0
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IncrementLogFreqQ4_16(tc.input)
			delta := (result & maskQ20) - (tc.input & maskQ20)

			if delta < tc.expectedDelta-tc.tolerance || delta > tc.expectedDelta+tc.tolerance {
				t.Errorf("Increment mismatch: got %d, expected %d±%d", delta, tc.expectedDelta, tc.tolerance)
			}
		})
	}
}

func TestIncrementLogFreqQ4_16_LogarithmicProperty(t *testing.T) {
	// Test that smaller values get larger increments (logarithmic property)
	smallValue := uint32(0x10000) // 1.0 in Q4.16
	largeValue := uint32(0x40000) // 4.0 in Q4.16

	smallResult := IncrementLogFreqQ4_16(smallValue)
	largeResult := IncrementLogFreqQ4_16(largeValue)

	smallIncrement := (smallResult & maskQ20) - (smallValue & maskQ20)
	largeIncrement := (largeResult & maskQ20) - (largeValue & maskQ20)

	// Smaller values should get larger increments due to logarithmic nature
	if smallIncrement <= largeIncrement {
		t.Errorf("Expected larger increment for smaller values: small=%d, large=%d",
			smallIncrement, largeIncrement)
	}
}

func TestIncrementLogFreqQ4_16_MultipleIncrements(t *testing.T) {
	// Test multiple successive increments
	value := uint32(0x8000) // 0.5 in Q4.16

	// Apply multiple increments and verify each one
	for i := 0; i < 10; i++ {
		prev := value
		value = IncrementLogFreqQ4_16(value)

		if value <= prev {
			t.Errorf("Increment %d failed: %d <= %d", i, value, prev)
		}

		// Verify we don't exceed maximum
		if (value & maskQ20) > maxQ20 {
			t.Errorf("Increment %d exceeded maximum: %d > %d", i, value&maskQ20, maxQ20)
		}
	}
}

func TestIncrementLogFreqQ4_16_FixedPointAccuracy(t *testing.T) {
	// Test the fixed-point arithmetic accuracy
	input := uint32(0x30000) // 3.0 in Q4.16
	result := IncrementLogFreqQ4_16(input)

	// Calculate expected result manually
	y := float64(input&maskQ20) / q16Scale
	expectedDelta := logFactor / math.Pow(logBase, y)
	expectedY := y + expectedDelta
	expectedRaw := uint32(expectedY * q16Scale)

	actualRaw := result & maskQ20

	// Allow for rounding errors in fixed-point conversion
	tolerance := uint32(10) // Higher precision format, smaller tolerance
	if actualRaw < expectedRaw-tolerance || actualRaw > expectedRaw+tolerance {
		t.Errorf("Fixed-point accuracy test failed: expected ~%d, got %d", expectedRaw, actualRaw)
	}
}

func TestIncrementLogFreqQ4_16_EdgeCases(t *testing.T) {
	// Test various edge cases
	edgeCases := []struct {
		name            string
		input           uint32
		expectIncrement bool
	}{
		{"Minimum non-zero", 0x1, true},
		{"Small fractional", 0x8000, true},     // 0.5 in Q4.16
		{"Q4.16 value 1.0", 0x10000, true},     // 1.0 in Q4.16
		{"Q4.16 value 8.0", 0x80000, false},    // 8.0 in Q4.16, may have zero increment
		{"Just before max", maxQ20 - 1, false}, // Near max may have zero increment
	}

	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IncrementLogFreqQ4_16(tc.input)

			// Basic sanity checks
			if (result & maskQ20) > maxQ20 {
				t.Error("Result exceeded maximum allowed value")
			}

			inputFreq := tc.input & maskQ20
			resultFreq := result & maskQ20

			if tc.expectIncrement {
				if resultFreq <= inputFreq {
					t.Error("Expected increment for this input value")
				}
			} else {
				// For large values, increment may be zero due to logarithmic nature
				if resultFreq < inputFreq {
					t.Error("Result should not decrease")
				}
			}
		})
	}
}

func TestIncrementLogFreqQ4_16_CompareWithQ20(t *testing.T) {
	// Compare behavior between Q4.16 and Q12.8 formats for equivalent values
	// Both should preserve upper bits and have similar logarithmic properties

	// Test value 1.0 in both formats
	valueQ20 := uint32(0x12300100)   // 1.0 in Q12.8 with upper bits 0x123
	valueQ4_16 := uint32(0x12310000) // 1.0 in Q4.16 with upper bits 0x123

	resultQ20 := IncrementLogFreqQ20(valueQ20)
	resultQ4_16 := IncrementLogFreqQ4_16(valueQ4_16)

	// Both should preserve upper bits
	if resultQ20>>20 != 0x123 || resultQ4_16>>20 != 0x123 {
		t.Error("Upper bits not preserved in comparison test")
	}

	// Both should show increments (though magnitudes will differ due to scale)
	if (resultQ20 & maskQ20) <= (valueQ20 & maskQ20) {
		t.Error("Q20 version should increment")
	}
	if (resultQ4_16 & maskQ20) <= (valueQ4_16 & maskQ20) {
		t.Error("Q4.16 version should increment")
	}
}

// Benchmark to verify performance characteristics
func BenchmarkIncrementLogFreqQ20(b *testing.B) {
	input := uint32(0x12345678) // Mix of upper bits and log-freq value

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = IncrementLogFreqQ20(input)
	}
}

func BenchmarkIncrementLogFreqQ4_16(b *testing.B) {
	input := uint32(0x12345678) // Mix of upper bits and log-freq value

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = IncrementLogFreqQ4_16(input)
	}
}
