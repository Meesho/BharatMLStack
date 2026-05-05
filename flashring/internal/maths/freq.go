// freq.go
package maths

/*
Package maths implements a binary Morris-style probabilistic counter
compressed into a single uint16.

------------------------------------------------------------------------
How the algorithm works
------------------------------------------------------------------------

 1. Layout (16 bits)

    ┌─ exponent (4 bits) ─┬─ mantissa (12 bits) ─┐
    │       e (0–15)       │     m (0–4095)        │
    └──────────────────────┴───────────────────────┘

    The counter encodes the approximate value: m × 2ᵉ (equivalently m << e).

 2. Increment rule

    On each key access, the counter is incremented probabilistically:
      - Probability of increment = 1 / 2ᵉ.
      - The caller supplies an external hash (hlo). We compare its lower
        32 bits against a precomputed threshold: th[e] = (2³² - 1) >> e.
      - If uint32(hlo) < th[e] → hit → mantissa advances (m++).
      - If uint32(hlo) >= th[e] → miss → counter unchanged.


 3. Mantissa overflow

    When m reaches 4096 (overflows 12 bits), we halve the mantissa
    and bump the exponent:  m = 2048, e++.

    This preserves the approximate decoded value across the transition:
      Before: m=4095, e=0  →  Value = 4095 × 1  = 4095
      After:  m=2048, e=1  →  Value = 2048 × 2  = 4096

    At max exponent (e=15), the counter saturates at m=4095, e=15
    (decoded value = 4095 × 32768 = 134,184,960).

 4. Decoding

    Value = m << e

    Examples:
      Encoded (e=0, m=42)   → 42 << 0  = 42       (exact)
      Encoded (e=0, m=4000) → 4000 << 0 = 4000     (exact)
      Encoded (e=1, m=2048) → 2048 << 1 = 4096     (step size = 2)
      Encoded (e=2, m=2500) → 2500 << 2 = 10000    (step size = 4)

 5. Resolution

    At exponent e, the step between consecutive representable values is 2ᵉ:
      e=0: step 1, exact integers      0 – 4,095
      e=1: step 2, even numbers    4,096 – 8,190
      e=2: step 4                  8,192 – 16,380
      ...
      e=15: step 32,768          up to ~134 million

    For cache frequency tracking, most keys stay in e=0 (exact counts up
    to 4095) or e=1 (step of 2), giving ~6,600 distinct values under 10K
    compared to ~37 with the previous base-10 design.

 6. Complexity & footprint

    State per key: 2 bytes (uint16), stored inline in the index entry.
    Increment:     1 compare + a few bit-ops, no floating-point.
    Thresholds:    precomputed once in New() (16 entries).
*/

// 12-bit mantissa (0–4095). 4-bit exponent (0–15).
const (
	mBits     = 12
	mMask     = (1 << mBits) - 1 // 0x0FFF
	eShift    = mBits
	mOverflow = 1 << mBits // 4096
)

type MorrisLogCounter struct {
	th       []uint32 // th[e] = (2^32 - 1) >> e; increment probability = 1/2^e
	pow2     []uint64 // pow2[e] = 2^e; used for decoding
	expClamp uint32   // maximum exponent, fixed at 15
}

// New creates a MorrisLogCounter with precomputed threshold and power tables.
// The 4-bit exponent field supports exponents 0–15.
func New() *MorrisLogCounter {
	const maxExp = 15

	th := make([]uint32, maxExp+1)
	pow2 := make([]uint64, maxExp+1)

	max32 := uint64(^uint32(0)) // 2^32 - 1

	for e := 0; e <= maxExp; e++ {
		th[e] = uint32(max32 >> e)
		pow2[e] = 1 << uint(e)
	}

	return &MorrisLogCounter{
		th:       th,
		pow2:     pow2,
		expClamp: maxExp,
	}
}

// Inc probabilistically increments the counter. hlo is the lower 64 bits
// of the key's hash, used as the randomness source for the Bernoulli trial.
// Returns the (possibly updated) counter and whether an increment occurred.
func (c *MorrisLogCounter) Inc(v uint16, hlo uint64) (uint16, bool) {
	m := v & mMask
	e := v >> eShift

	// Bernoulli trial: increment with probability 1/2^e.
	// At e=0 this is ~100% (th[0] = 0xFFFFFFFF).
	// At e=1 this is ~50%, at e=2 ~25%, etc.
	if uint32(hlo) >= c.th[e] {
		return v, false
	}

	m++
	if m == mOverflow {
		// Mantissa overflowed 12 bits. Halve mantissa and bump exponent
		// to keep the decoded value approximately continuous.
		if e < 15 {
			m = m >> 1 // 4096 → 2048
			e++
		} else {
			// Saturate: can't increase exponent further.
			m = mOverflow - 1 // clamp at 4095
		}
	}
	return (e << eShift) | (m & mMask), true
}

// Value decodes the counter into an approximate frequency: m × 2^e.
func (c *MorrisLogCounter) Value(v uint16) uint64 {
	m := uint64(v & mMask)
	e := v >> eShift
	return m << e
}
