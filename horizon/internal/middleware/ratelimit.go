package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// IPRateLimiter applies a token-bucket rate limit per client IP. It is intended
// for sensitive, unauthenticated endpoints such as /login where an attacker
// could otherwise attempt unlimited credential guesses.
type IPRateLimiter struct {
	mu      sync.Mutex
	clients map[string]*clientLimiter
	r       rate.Limit
	burst   int
	ttl     time.Duration
}

type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewIPRateLimiter creates a limiter allowing r events/second with the given
// burst. Idle per-IP entries are evicted after ttl to bound memory usage.
func NewIPRateLimiter(r rate.Limit, burst int, ttl time.Duration) *IPRateLimiter {
	l := &IPRateLimiter{
		clients: make(map[string]*clientLimiter),
		r:       r,
		burst:   burst,
		ttl:     ttl,
	}
	go l.cleanupLoop()
	return l
}

func (l *IPRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(l.ttl)
	defer ticker.Stop()
	for range ticker.C {
		l.mu.Lock()
		for ip, cl := range l.clients {
			if time.Since(cl.lastSeen) > l.ttl {
				delete(l.clients, ip)
			}
		}
		l.mu.Unlock()
	}
}

func (l *IPRateLimiter) get(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()
	cl, ok := l.clients[ip]
	if !ok {
		cl = &clientLimiter{limiter: rate.NewLimiter(l.r, l.burst)}
		l.clients[ip] = cl
	}
	cl.lastSeen = time.Now()
	return cl.limiter
}

// Allow reports whether a request from ip may proceed.
func (l *IPRateLimiter) Allow(ip string) bool {
	return l.get(ip).Allow()
}

// Middleware returns a gin handler that rejects requests exceeding the limit
// with HTTP 429. NOTE: behind a proxy/ingress, configure gin's TrustedProxies
// so that ClientIP() reflects the real client and cannot be spoofed via
// X-Forwarded-For.
func (l *IPRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.Allow(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests, please try again later",
			})
			return
		}
		c.Next()
	}
}
