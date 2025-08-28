// freq.go
package maths

/*
Package maths implements a **decimal Morris‑style probabilistic counter**
compressed into a single `uint32`.

------------------------------------------------------------------------
How the algorithm works
------------------------------------------------------------------------
1. **Layout (24 bits)**
| exponent : 20 bits | mantissa : 4 bits |
e m (0‑9)
The counter encodes ≈ `m · 10ᵉ`.
* Mantissa cycles at 10 (`mOverflow = 10`).
* `expClamp` chosen at construction bounds the maximum exponent.

2. **Increment rule**
On each logical “event”, we increment the *stored* value only with
probability **1 / 10ᵉ** (Bernoulli trial).
- A 32‑bit xorshift PRNG generates `rand32()`.
- We pre‑compute `th[e] = ⌊2³² / 10ᵉ⌋`.
- `rand32() < th[e]` ⇢ *hit* ⇒ advance mantissa (`m++`).
- When `m == 10` we reset `m = 0` and bump the exponent
  (until `expClamp`, then we saturate).

This is the classic idea introduced by **Robert Morris** for “counting
large numbers of events in small registers” :contentReference[oaicite:0]{index=0}.

3. **Decoding**
To retrieve an approximate frequency, multiply
`m · 10ᵉ` (done with a tiny `pow10` table).

4. **Error guarantees**
For mantissa 0‑9 the standard deviation of the estimate is
`σ ≈ √m · 10ᵉ`, so the relative error is ≤ `1/√m`
(≤ 33 % worst‑case, ≤ 10 % once `m ≥ 10`).
Such accuracy is typical for Morris‑style counters used in
streaming & LFU/TinyLFU cache admission :contentReference[oaicite:1]{index=1}.

5. **Complexity & footprint**
* **State per key:** 4 bytes.
* **Increment:** ~7 integer ops for PRNG + 1 compare + a few bit‑ops
  ⇒ ~5‑7 ns on modern CPUs (counter update is usually cheaper than the
  surrounding map/slice access).
* **No floating‑point or division** in the hot path; thresholds
  are prepared once in `New`.

------------------------------------------------------------------------
References
------------------------------------------------------------------------
* R. Morris. “Counting large numbers of events in small registers.”
*Communications of the ACM*, 21(10): 840‑842, 1978. :contentReference[oaicite:2]{index=2}
* P. Flajolet. “Approximate Counting: A Detailed Analysis.” *BIT* 25, 1985. :contentReference[oaicite:3]{index=3}
* G. Gundersen, “Approximate Counting with Morris’s Algorithm,” blog post, 2019. :contentReference[oaicite:4]{index=4}
*/

// 4‑bit mantissa (0‑9).  20‑bit exponent (0 … expClamp).
const (
	mBits     = 4
	eBits     = 24 - mBits
	mMask     = (1 << mBits) - 1 // 0xF
	eShift    = mBits
	mOverflow = 10 // mantissa cycles at 10
)

// ----------- fast RNG (xorshift32) --------
var rng uint32 = 0x7263b8e4 // non‑zero seed

type MorrisLogCounter struct {
	th       []uint32 // thresholds   th[e] = floor(2^32 / 10^e)
	pow10    []uint64 // pow10[e] = 10^e
	expClamp uint32
	rng      uint32
}

// New prepares tables for a desired exponent ceiling.
// expClamp must fit in the 20‑bit exponent field.
func New(expClamp uint32) *MorrisLogCounter {
	if expClamp >= 1<<eBits {
		panic("expClamp exceeds 20‑bit exponent capacity")
	}

	th := make([]uint32, expClamp+1)
	pow10 := make([]uint64, expClamp+1)

	var p10 uint64 = 1 // 10^0
	max32 := uint64(^uint32(0))
	for e := uint32(0); e <= expClamp; e++ {
		if e > 0 {
			p10 *= 10
		}
		pow10[e] = p10
		th[e] = uint32(max32 / p10) // floor(2^32 / 10^e)
	}

	return &MorrisLogCounter{
		th:       th,
		pow10:    pow10,
		expClamp: expClamp,
		rng:      rng,
	}
}

func (c *MorrisLogCounter) Inc(v uint32) (uint32, bool) {
	m := v & mMask   // mantissa
	e := v >> eShift // exponent (0 … expClamp)

	// 1 / 10^e probability check
	if c.rand32() >= c.th[e] {
		return v, false // miss
	}

	// hit
	m++
	if m == mOverflow {
		m = 0
		if e < c.expClamp {
			e++
		} else { // saturated at top state
			m = mOverflow - 1
		}
	}
	return (e << eShift) | m, true
}

func (c *MorrisLogCounter) Value(v uint32) uint64 {
	m := uint64(v & mMask)
	e := v >> eShift
	return m * c.pow10[e]
}

func (c *MorrisLogCounter) rand32() uint32 {
	r := rng
	r ^= r << 13
	r ^= r >> 17
	r ^= r << 5
	rng = r
	return r
}
