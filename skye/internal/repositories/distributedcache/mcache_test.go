package distributedcache

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetFinalTTLWithJitter(t *testing.T) {
	for i := 0; i < 100; i++ {
		ttl := 100
		finalTTL := getFinalTTLWithJitter(ttl)
		// jitter is ±10%, so finalTTL should be in [90, 110]
		assert.GreaterOrEqual(t, finalTTL, 90)
		assert.LessOrEqual(t, finalTTL, 110)
	}
}

func TestGetFinalTTLWithJitter_SmallTTL(t *testing.T) {
	for i := 0; i < 100; i++ {
		ttl := 10
		finalTTL := getFinalTTLWithJitter(ttl)
		// jitter is ±10% of 10 = ±1, range [9, 11]; floor clamped to ttl if < 1
		assert.GreaterOrEqual(t, finalTTL, 1)
		assert.LessOrEqual(t, finalTTL, 11)
	}
}

func TestGetFinalTTLWithJitter_ZeroTTL(t *testing.T) {
	// ttl=0 means jitterRange=0, rand.Intn(1)=0, finalTTL=0 → clamped to ttl=0
	finalTTL := getFinalTTLWithJitter(0)
	assert.Equal(t, 0, finalTTL)
}
