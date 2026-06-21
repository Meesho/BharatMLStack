package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func TestIPRateLimiter_Allow(t *testing.T) {
	// 1 event/sec, burst of 2: the first 2 calls pass, the 3rd is blocked.
	l := NewIPRateLimiter(rate.Every(time.Second), 2, time.Minute)

	if !l.Allow("1.2.3.4") {
		t.Fatal("call 1 should be allowed")
	}
	if !l.Allow("1.2.3.4") {
		t.Fatal("call 2 (within burst) should be allowed")
	}
	if l.Allow("1.2.3.4") {
		t.Fatal("call 3 should be rate limited")
	}
	// A different IP has its own independent bucket.
	if !l.Allow("5.6.7.8") {
		t.Fatal("different IP should have its own bucket")
	}
}

func TestIPRateLimiter_Middleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	l := NewIPRateLimiter(rate.Every(time.Hour), 1, time.Minute) // burst 1
	r := gin.New()
	r.POST("/login", l.Middleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	do := func() int {
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		req.RemoteAddr = "9.9.9.9:1234"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}

	if code := do(); code != http.StatusOK {
		t.Fatalf("first request: got %d, want 200", code)
	}
	if code := do(); code != http.StatusTooManyRequests {
		t.Fatalf("second request: got %d, want 429", code)
	}
}

// TestIPRateLimiter_Refill verifies that the bucket refills over time: once
// enough time has elapsed for a token to be replenished, a previously
// rate-limited client is allowed again.
func TestIPRateLimiter_Refill(t *testing.T) {
	// 100 events/sec (one token every 10ms), burst 1.
	l := NewIPRateLimiter(rate.Every(10*time.Millisecond), 1, time.Minute)
	defer l.Stop()

	const ip = "10.0.0.1"
	if !l.Allow(ip) {
		t.Fatal("first call should be allowed")
	}
	if l.Allow(ip) {
		t.Fatal("second immediate call should be limited")
	}

	// Wait for the bucket to refill, then it should pass again.
	time.Sleep(30 * time.Millisecond)
	if !l.Allow(ip) {
		t.Fatal("call after refill window should be allowed")
	}
}

// TestNewIPRateLimiter_NonPositiveTTL ensures a non-positive ttl does not panic
// (time.NewTicker panics on <= 0) and that the limiter is still usable.
func TestNewIPRateLimiter_NonPositiveTTL(t *testing.T) {
	for _, ttl := range []time.Duration{0, -time.Second} {
		l := NewIPRateLimiter(rate.Every(time.Second), 1, ttl)
		if !l.Allow("1.1.1.1") {
			t.Fatalf("ttl=%v: limiter should allow the first call", ttl)
		}
	}
}
