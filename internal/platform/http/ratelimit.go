package http

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// ---- Rate Limiting --------------------------------------------------------
//
// Token-bucket limiter keyed by client IP. Used on sensitive endpoints
// (auth, partner requests) where abuse would be costly. Limits are per
// process — sufficient for the single-node deployment this service targets;
// a distributed deployment would swap in a Redis-backed limiter.

// ipLimiter wraps a rate.Limiter with a last-seen timestamp so idle entries
// can be evicted instead of growing without bound.
type ipLimiter struct {
	lim      *rate.Limiter
	lastSeen time.Time
}

// RateLimiter returns a middleware that allows events per second with a burst
// of burst per client IP. Requests beyond the budget get 429 with a
// Retry-After hint.
func RateLimiter(eventsPerSec float64, burst int) gin.HandlerFunc {
	var (
		mu       sync.Mutex
		limiters = make(map[string]*ipLimiter)
	)

	// Evict entries idle for over an hour, at most once a minute.
	go func() {
		for range time.Tick(time.Minute) {
			mu.Lock()
			for ip, l := range limiters {
				if time.Since(l.lastSeen) > time.Hour {
					delete(limiters, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()
		mu.Lock()
		l, ok := limiters[ip]
		if !ok {
			l = &ipLimiter{lim: rate.NewLimiter(rate.Limit(eventsPerSec), burst)}
			limiters[ip] = l
		}
		l.lastSeen = time.Now()
		mu.Unlock()

		if !l.lim.Allow() {
			c.Header("Retry-After", "1")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"code":    "rate_limited",
					"message": "too many requests, please slow down",
				},
			})
			return
		}
		c.Next()
	}
}
