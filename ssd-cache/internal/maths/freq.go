package maths

import "math"

const (
	q8Scale   = 1 << 8 // 256
	q16Scale  = 1 << 16
	logFactor = 0.4343 // ≈ 1/ln(10)
	logBase   = 10.0
	maxQ20    = (1 << 20) - 1 // 0xFFFFF
	maskQ20   = uint32(0xFFFFF)
)

// IncrementLogFreqQ20 increments a 20-bit Q12.8 log-freq value (in lower bits of uint32).
// Upper 12 bits are preserved and returned untouched.
func IncrementLogFreqQ20(packed uint32) uint32 {
	raw := packed & maskQ20
	y := float64(raw) / q8Scale

	// Increment logic
	delta := logFactor / math.Pow(logBase, y)
	y += delta

	// Convert back to fixed-point
	newRaw := uint32(y * q8Scale)
	if newRaw > maxQ20 {
		newRaw = maxQ20
	}

	// Return packed with upper 12 bits preserved
	return (packed &^ maskQ20) | newRaw
}

// IncrementLogFreqQ4_16 increments the lower 20-bit Q4.16 log-frequency in-place.
// Upper 12 bits are preserved untouched.
func IncrementLogFreqQ4_16(val uint32) uint32 {
	raw := val & maskQ20                  // Extract Q4.16
	y := float64(raw) / float64(q16Scale) // Convert to float

	delta := logFactor / math.Pow(logBase, y)
	y += delta

	newRaw := uint32(y * q16Scale)
	if newRaw > maxQ20 {
		newRaw = maxQ20
	}
	return (val &^ maskQ20) | newRaw
}
