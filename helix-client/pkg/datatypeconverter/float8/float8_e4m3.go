package float8

import (
	"math"
	"math/bits"
)

const (
	fp8E4M3Max          uint32 = 1087 << 20
	E4M3DeNormMask      uint32 = 141 << 23
	E4M3ExponentBiasAug uint32 = 3288858623 // Adjusted exponent bias for FP8 format. Represents ((uint32_t)(7 - 127) << 23) + 0x7FFFF;
)

type Float8e4m3 uint8

// FP8E4M3FromFP32Value converts a 32-bit floating-point value (float32) to an 8-bit floating-point value (FP8 E4M3 format).
// The FP8 E4M3 format consists of 1 sign bit, 4 exponent bits, and 3 mantissa bits.
// Implementation based on the paper https://arxiv.org/pdf/2209.05433.pdf
//
// Parameters:
// - f: The 32-bit floating-point value to be converted.
//
// Returns:
// - uint8: The 8-bit floating-point representation of the input value.

func FP8E4M3FromFP32Value(f float32) Float8e4m3 {

	fBits := math.Float32bits(f)
	var result uint8

	sign := fBits & 0x80000000
	fBits ^= sign

	if fBits >= fp8E4M3Max {
		result = 0x7f
	} else {
		if fBits < (121 << 23) {
			fBits = math.Float32bits(math.Float32frombits(fBits) + math.Float32frombits(E4M3DeNormMask))
			result = uint8(fBits - E4M3DeNormMask)
		} else {
			mantissaOdd := (fBits >> 20) & 1
			fBits += E4M3ExponentBiasAug
			fBits += mantissaOdd
			result = uint8(fBits >> 20)
		}
	}

	result |= uint8(sign >> 24)

	return Float8e4m3(result)
}

// FP8E4M3ToFP32Value converts an 8-bit floating-point value (FP8 E4M3 format) to a 32-bit floating-point value (float32).
// The FP8 E4M3 format consists of 1 sign bit, 4 exponent bits, and 3 mantissa bits.
// NaN is represented when all exponent and mantissa bits are 1.
//
// Parameters:
// - input: The 8-bit floating-point value to be converted.
//
// Returns:
// - float32: The 32-bit floating-point representation of the input value.
func FP8E4M3ToFP32Value(input Float8e4m3) float32 {
	// In FP8E4M3FN format:
	// - Bits 7: Sign bit
	// - Bits 6-3: Exponent (4 bits)
	// - Bits 2-0: Mantissa (3 bits)
	// NaN is represented when all exponent and mantissa bits are 1

	// First, check if the input is NaN (0x7F = 0111 1111)
	if (input & 0x7F) == 0x7F {
		return float32(math.NaN())
	}

	// Extend to 32 bits and shift to upper part
	w := uint32(input) << 24

	// Extract sign bit
	sign := w & uint32(0x80000000)

	// Extract mantissa and biased exponent
	nonSign := w & uint32(0x7FFFFFFF)

	// Calculate renormalization shift
	var reNormShift uint32
	if nonSign != 0 {
		// Count leading zeros after the sign bit
		reNormShift = uint32(bits.LeadingZeros32(nonSign))
		if reNormShift > 4 {
			reNormShift -= 4
		} else {
			reNormShift = 0
		}
	}

	// Check for zero
	if nonSign == 0 {
		return math.Float32frombits(sign) // Preserve sign for ±0
	}

	// Handle normalized and denormalized numbers
	result := sign |
		((nonSign << reNormShift >> 4) + ((0x78 - reNormShift) << 23))

	return math.Float32frombits(result)
}
