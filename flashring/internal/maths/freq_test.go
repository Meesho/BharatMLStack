package maths

import (
	"testing"
)

func TestNew(t *testing.T) {
	counter := New()
	if counter == nil {
		t.Fatal("New() returned nil")
	}
	if counter.expClamp != 15 {
		t.Errorf("expClamp = %v, want 15", counter.expClamp)
	}
	if len(counter.th) != 16 {
		t.Errorf("threshold table length = %v, want 16", len(counter.th))
	}
	if len(counter.pow2) != 16 {
		t.Errorf("pow2 table length = %v, want 16", len(counter.pow2))
	}
}

func TestPow2Table(t *testing.T) {
	counter := New()

	expected := []uint64{1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768}
	for i, exp := range expected {
		if counter.pow2[i] != exp {
			t.Errorf("pow2[%d] = %v, want %v", i, counter.pow2[i], exp)
		}
	}
}

func TestThresholdTable(t *testing.T) {
	counter := New()

	max32 := uint64(^uint32(0))

	for e := uint32(0); e <= 15; e++ {
		expected := uint32(max32 >> e)
		if counter.th[e] != expected {
			t.Errorf("th[%d] = %v, want %v", e, counter.th[e], expected)
		}
	}
}

func TestValue(t *testing.T) {
	counter := New()

	tests := []struct {
		name     string
		v        uint16
		expected uint64
	}{
		{
			name:     "mantissa 0, exponent 0",
			v:        0,
			expected: 0,
		},
		{
			name:     "mantissa 5, exponent 0",
			v:        5,
			expected: 5, // 5 << 0
		},
		{
			name:     "mantissa 3, exponent 1",
			v:        (1 << eShift) | 3,
			expected: 6, // 3 << 1
		},
		{
			name:     "mantissa 100, exponent 2",
			v:        (2 << eShift) | 100,
			expected: 400, // 100 << 2
		},
		{
			name:     "mantissa 4095, exponent 0",
			v:        4095,
			expected: 4095, // 4095 << 0
		},
		{
			name:     "mantissa 2048, exponent 1",
			v:        (1 << eShift) | 2048,
			expected: 4096, // 2048 << 1
		},
		{
			name:     "mantissa 2048, exponent 15",
			v:        (15 << eShift) | 2048,
			expected: 2048 << 15, // 67108864
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := counter.Value(tt.v)
			if result != tt.expected {
				t.Errorf("Value(%v) = %v, want %v", tt.v, result, tt.expected)
			}
		})
	}
}

func TestIncBasicBehavior(t *testing.T) {
	counter := New()

	// With e=0, th[0] = 0xFFFFFFFF, so any hlo will hit (uint32(hlo) < th[0])
	v := uint16(5) // m=5, e=0
	newV, hit := counter.Inc(v, 0)

	if !hit {
		t.Error("Inc() should always hit at e=0")
	}

	expectedV := uint16(6) // m=6, e=0
	if newV != expectedV {
		t.Errorf("Inc(%v) = %v, want %v", v, newV, expectedV)
	}
}

func TestIncMantissaOverflow(t *testing.T) {
	counter := New()

	// m=4095 (mOverflow-1), e=0 -> increment should cause overflow
	v := uint16(mOverflow - 1) // m=4095, e=0
	newV, hit := counter.Inc(v, 0)

	if !hit {
		t.Error("Inc() should always hit at e=0")
	}

	// On overflow: m becomes 4096>>1 = 2048, e becomes 1
	expectedM := uint16(mOverflow >> 1) // 2048
	expectedE := uint16(1)
	expectedV := (expectedE << eShift) | expectedM

	if newV != expectedV {
		t.Errorf("Inc(%v) = %v, want %v (m=2048, e=1)", v, newV, expectedV)
	}

	// Verify the decoded value is reasonable
	// Before: Value(4095) = 4095 << 0 = 4095
	// After:  Value(newV) = 2048 << 1 = 4096
	valBefore := counter.Value(v)
	valAfter := counter.Value(newV)
	if valAfter <= valBefore {
		t.Errorf("Value should increase after overflow: before=%v, after=%v", valBefore, valAfter)
	}
}

func TestIncExponentSaturation(t *testing.T) {
	counter := New()

	// m=4095, e=15 (max exponent) -> should saturate
	v := (uint16(15) << eShift) | uint16(mOverflow-1) // m=4095, e=15
	newV, hit := counter.Inc(v, 0)

	if !hit {
		t.Error("Inc() should hit")
	}

	// Should saturate: m stays at 4095, e stays at 15
	if newV != v {
		t.Errorf("Inc(%v) = %v, want %v (saturated at max)", v, newV, v)
	}
}

func TestIncMissBehavior(t *testing.T) {
	counter := New()

	// At e=1, th[1] = 0xFFFFFFFF >> 1 = 0x7FFFFFFF
	// hlo with uint32 >= 0x7FFFFFFF should miss
	v := (uint16(1) << eShift) | 5 // m=5, e=1
	hlo := uint64(0xFFFFFFFF)      // uint32(hlo) = 0xFFFFFFFF >= th[1]

	newV, hit := counter.Inc(v, hlo)

	if hit {
		t.Error("Inc() should miss when uint32(hlo) >= th[e]")
	}
	if newV != v {
		t.Errorf("Inc() on miss should return original value: got %v, want %v", newV, v)
	}
}

func TestIncStatisticalBehavior(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping statistical test in short mode")
	}

	counter := New()

	// Test with e=0 (should hit ~100% of the time since th[0] = 0xFFFFFFFF)
	v := uint16(5)
	hits := 0
	trials := 1000

	for i := 0; i < trials; i++ {
		_, hit := counter.Inc(v, uint64(i))
		if hit {
			hits++
		}
	}

	hitRate := float64(hits) / float64(trials)
	if hitRate < 0.99 {
		t.Errorf("Hit rate for e=0 = %v, want ~1.0", hitRate)
	}

	// Test with e=1 (should hit approximately 50% of the time)
	// th[1] = 0x7FFFFFFF, so uint32(hlo) < th[1] means lower half hits
	v = (1 << eShift) | 5
	hits = 0

	for i := 0; i < trials; i++ {
		// Use Knuth multiplicative hash to spread uint32 values evenly
		hlo := uint64(uint32(i) * 2654435761)
		_, hit := counter.Inc(v, hlo)
		if hit {
			hits++
		}
	}

	hitRate = float64(hits) / float64(trials)
	if hitRate < 0.35 || hitRate > 0.65 {
		t.Errorf("Hit rate for e=1 = %v, want ~0.50 (0.35-0.65)", hitRate)
	}
}

func TestIntegrationCountingApproximation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	counter := New()

	v := uint16(0)
	totalEvents := 100000

	for i := 0; i < totalEvents; i++ {
		newV, hit := counter.Inc(v, uint64(i*2654435761)) // Knuth multiplicative hash for spread
		if hit {
			v = newV
		}
	}

	approxCount := counter.Value(v)

	// The approximation should be in the right ballpark
	ratio := float64(approxCount) / float64(totalEvents)
	if ratio < 0.1 || ratio > 10.0 {
		t.Errorf("Approximation ratio = %v, totalEvents = %v, approxCount = %v",
			ratio, totalEvents, approxCount)
	}
}

func TestBitPacking(t *testing.T) {
	counter := New()

	tests := []struct {
		mantissa uint16
		exponent uint16
	}{
		{0, 0},
		{4095, 0},
		{0, 15},
		{2048, 3},
		{100, 7},
	}

	for _, tt := range tests {
		v := (tt.exponent << eShift) | (tt.mantissa & mMask)

		extractedM := v & mMask
		extractedE := v >> eShift

		if extractedM != tt.mantissa&mMask {
			t.Errorf("Mantissa packing: got %v, want %v", extractedM, tt.mantissa&mMask)
		}
		if extractedE != tt.exponent {
			t.Errorf("Exponent packing: got %v, want %v", extractedE, tt.exponent)
		}

		decoded := counter.Value(v)
		expected := uint64(tt.mantissa&mMask) << tt.exponent
		if decoded != expected {
			t.Errorf("Value() = %v, want %v (m=%v, e=%v)", decoded, expected, tt.mantissa, tt.exponent)
		}
	}
}

func BenchmarkInc(b *testing.B) {
	counter := New()
	v := uint16(123)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v, _ = counter.Inc(v, uint64(i))
	}
}

func BenchmarkValue(b *testing.B) {
	counter := New()
	v := uint16(123)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = counter.Value(v)
	}
}
