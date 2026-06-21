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
