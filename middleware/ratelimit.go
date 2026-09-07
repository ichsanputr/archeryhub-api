package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type ipLimiter struct {
	mu      sync.Mutex
	history map[string][]time.Time
	limit   int
	window  time.Duration
}

func newIPLimiter(limit int, window time.Duration) *ipLimiter {
	l := &ipLimiter{
		history: make(map[string][]time.Time),
		limit:   limit,
		window:  window,
	}

	// Periodically clean up stale records every 2 * window
	go func() {
		ticker := time.NewTicker(2 * window)
		for range ticker.C {
			l.mu.Lock()
			now := time.Now()
			for ip, times := range l.history {
				var valid []time.Time
				for _, t := range times {
					if now.Sub(t) <= l.window {
						valid = append(valid, t)
					}
				}
				if len(valid) == 0 {
					delete(l.history, ip)
				} else {
					l.history[ip] = valid
				}
			}
			l.mu.Unlock()
		}
	}()

	return l
}

func (l *ipLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	// Filter out expired timestamps
	var valid []time.Time
	for _, t := range l.history[ip] {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= l.limit {
		l.history[ip] = valid
		return false
	}

	l.history[ip] = append(valid, now)
	return true
}

// RateLimit creates a middleware that restricts requests per client IP within a sliding window
func RateLimit(limit int, window time.Duration) gin.HandlerFunc {
	limiter := newIPLimiter(limit, window)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !limiter.allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Terlalu banyak percobaan. Silakan coba beberapa saat lagi.",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
