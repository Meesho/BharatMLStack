package float8

import (
	"math"
)

const (
	fp32Inf             uint32 = 255 << 23  // 32-bit representation of infinity (exponent all 1s)
	fp8E5M2Max          uint32 = 143 << 23  // Maximum FP8 normalized value in FP32 format
	E5M2DeNormMask      uint32 = 134 << 23  // Mask for creating denormalized values in FP8
	E5M2ExponentBiasAug uint32 = 3356491775 // Adjusted exponent bias for FP8 format. Represents ((uint32_t)(15 - 127) << 23) + 0xFFFFF;
)

type Float8e5m2 uint8

// FP8E5M2FromFP32Value converts a 32-bit floating-point value (float32) to an 8-bit floating-point value (FP8 E5M2 format).
// The FP8 E5M2 format consists of 1 sign bit, 5 exponent bits, and 2 mantissa bits.
/// Implementation based on the paper https://arxiv.org/pdf/2209.05433.pdf
//
// Parameters:
// - f: The 32-bit floating-point value to be converted.
//
// Returns:
// - uint8: The 8-bit floating-point representation of the input value

func FP8E5M2FromFP32Value(f float32) Float8e5m2 {

	// Convert float32 to IEEE 754 binary representation (uint32)
	fBits := math.Float32bits(f)
	var result uint8 = 0

	// Extract the sign bit and adjust fBits to hold only magnitude
	sign := fBits & 0x80000000 // Isolate sign bit
	fBits ^= sign              // Clear sign a bit from fBits

	// Handle special cases: NaN and infinity
	if fBits >= fp8E5M2Max {
		if fBits > fp32Inf {
			result = 0x7F // FP8 NaN representation
		} else {
			result = 0x7C // FP8 Infinity representation
		}
	} else {
		// Handle normal and deNormal values
		if fBits < (113 << 23) { // Threshold for denormalized FP8 values
			// Adjust denormalized values with a deNorm mask for proper rounding
			fBits = math.Float32bits(math.Float32frombits(fBits) + math.Float32frombits(E5M2DeNormMask))
			result = uint8(fBits - E5M2DeNormMask)
		} else {
			// Adjust for normalized values and rounding
			mantissaOdd := (fBits >> 21) & 1           // Check if mantissa is odd for rounding
			fBits += E5M2ExponentBiasAug + mantissaOdd // Adjust exponent bias and apply rounding
			result = uint8(fBits >> 21)                // Shift to get the final 8-bit result
		}
	}

	// Restore the sign bit to the final FP8 result
	result |= uint8(sign >> 24)

	return Float8e5m2(result)
}

// FP8E5M2ToFP32Value converts an 8-bit floating-point value (FP8 E5M2 format) to a 32-bit floating-point value (float32).
// The FP8 E5M2 format consists of 1 sign bit, 5 exponent bits, and 2 mantissa bits.
//
// Parameters:
// - fp8: The 8-bit floating-point value to be converted.
//
// Returns:
// - float32: The 32-bit floating-point representation of the input value

func FP8E5M2ToFP32Value(fp8 Float8e5m2) float32 {

	// Extract components
	sign := (fp8 >> 7) & 0x1      // Extract the sign bit (1 bit)
	exponent := (fp8 >> 2) & 0x1F // Extract the exponent (5 bits)
	mantissa := fp8 & 0x3         // Extract the mantissa (2 bits)

	// Handle special cases: NaN and infinity
	if exponent == 0x1F {
		if mantissa == 0 {
			if sign == 1 {
				return float32(math.Inf(-1)) // Negative infinity
			}
			return float32(math.Inf(1)) // Positive infinity
		}
		return float32(math.NaN()) // NaN
	}

	var value float32

	// Handle normal and subnormal numbers
	if exponent == 0 {
		// Subnormal numbers
		// For subnormal: value = (-1)^sign * 2^(-14) * (0.mantissa)
		mantissaValue := float64(mantissa) / 4.0          // Divide by 4 because it's 2 bits
		value = float32(mantissaValue * math.Pow(2, -14)) // -14 is the minimum exponent
	} else {
		// Normal numbers
		// For normals: value = (-1)^sign * 2^(exp-bias) * (1.mantissa)
		mantissaValue := 1.0 + float64(mantissa)/4.0 // Add implicit 1
		trueExponent := int(exponent) - 15           // Subtract bias (15)
		value = float32(mantissaValue * math.Pow(2, float64(trueExponent)))
	}

	// Apply sign
	if sign == 1 {
		value = -value
	}

	return value
}
