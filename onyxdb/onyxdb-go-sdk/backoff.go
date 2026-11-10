package sdk

import "time"

// backoffJitter is the spread of every redial and unhealthy-mark wait, as a
// fraction of the nominal wait either side
// (docs/design/onyxdb-readserver-admission-and-sdk-redial.md §3 B1, B2).
const backoffJitter = 0.2

// backoffDelay is the wait after the n-th failure in a row (n ≥ 1): base,
// doubled for each failure after the first, capped at max, then spread ±20% by
// u (uniform in [0,1)). The jitter keeps a fleet of clients that failed
// together from coming back together. The cap applies before the jitter, so a
// wait never exceeds 1.2 × max.
func backoffDelay(base, max time.Duration, n int, u float64) time.Duration {
	d := base
	for i := 1; i < n && d < max; i++ {
		d *= 2
	}
	d = min(d, max)
	return time.Duration(float64(d) * (1 - backoffJitter + 2*backoffJitter*u))
}
