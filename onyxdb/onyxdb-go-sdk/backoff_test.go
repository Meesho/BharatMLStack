package sdk

import (
	"math"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// The dial backoff (B1) and the unhealthy backoff (B2) share one schedule:
// base, doubled per failure in a row, capped, then spread ±20% so a fleet of
// clients that failed together does not come back together
// (docs/design/onyxdb-readserver-admission-and-sdk-redial.md §3).

func TestBackoffDelay_DoublesFromTheBaseUpToTheCap(t *testing.T) {
	ms := time.Millisecond
	cases := []struct {
		base, max time.Duration
		failures  int
		want      time.Duration
	}{
		{100 * ms, 5 * time.Second, 1, 100 * ms},
		{100 * ms, 5 * time.Second, 2, 200 * ms},
		{100 * ms, 5 * time.Second, 3, 400 * ms},
		{100 * ms, 5 * time.Second, 6, 3200 * ms},
		{100 * ms, 5 * time.Second, 7, 5 * time.Second},
		{100 * ms, 5 * time.Second, 1 << 20, 5 * time.Second},
		{time.Second, 30 * time.Second, 1, time.Second},
		{time.Second, 30 * time.Second, 5, 16 * time.Second},
		{time.Second, 30 * time.Second, 6, 30 * time.Second},
		{time.Second, time.Second, 4, time.Second},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, backoffDelay(tc.base, tc.max, tc.failures, 0.5),
			"base %v max %v after %d failures", tc.base, tc.max, tc.failures)
	}
}

func TestBackoffDelay_JitterStaysWithinTwentyPercent(t *testing.T) {
	for n := 1; n <= 8; n++ {
		nominal := backoffDelay(100*time.Millisecond, 5*time.Second, n, 0.5)
		lo := time.Duration(float64(nominal) * 0.8)
		hi := time.Duration(float64(nominal) * 1.2)

		assert.Equal(t, lo, backoffDelay(100*time.Millisecond, 5*time.Second, n, 0))
		top := backoffDelay(100*time.Millisecond, 5*time.Second, n, math.Nextafter(1, 0))
		assert.LessOrEqual(t, top, hi)
		assert.Greater(t, top, time.Duration(float64(nominal)*1.19))
		for i := 0; i < 1000; i++ {
			d := backoffDelay(100*time.Millisecond, 5*time.Second, n, rand.Float64())
			assert.GreaterOrEqual(t, d, lo)
			assert.LessOrEqual(t, d, hi)
		}
	}
}
