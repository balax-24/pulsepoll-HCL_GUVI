package middleware

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"pulsepoll/backend/internal/response"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// maxLimiterEntries places a strict upper bound on in-memory rate-limiter entries
// to prevent memory exhaustion attacks from unbounded client keys.
const maxLimiterEntries = 10000

// RateLimiter provides flexible rate limiting using the token bucket algorithm.
type RateLimiter struct {
	limiters map[string]*rateLimiterEntry
	mu       sync.Mutex
	rate     rate.Limit
	burst    int
}

// IPRateLimiter is an alias for RateLimiter for backward compatibility.
type IPRateLimiter = RateLimiter

type rateLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewRateLimiter creates a RateLimiter with the specified requests-per-second rate and burst capacity.
func NewRateLimiter(rps rate.Limit, burst int) *RateLimiter {
	rl := &RateLimiter{
		limiters: make(map[string]*rateLimiterEntry),
		rate:     rps,
		burst:    burst,
	}

	// Background cleanup of stale entries to reclaim memory.
	go rl.cleanupLoop()

	return rl
}

// NewIPRateLimiter creates an IPRateLimiter (alias for NewRateLimiter).
func NewIPRateLimiter(rps rate.Limit, burst int) *RateLimiter {
	return NewRateLimiter(rps, burst)
}

// getLimiter retrieves or creates a rate limiter for the given key.
func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, exists := rl.limiters[key]
	if exists {
		entry.lastSeen = time.Now()
		return entry.limiter
	}

	// Memory protection: if map reaches capacity, evict stale entries (>5 min)
	if len(rl.limiters) >= maxLimiterEntries {
		now := time.Now()
		for k, v := range rl.limiters {
			if now.Sub(v.lastSeen) > 5*time.Minute {
				delete(rl.limiters, k)
			}
		}

		// If still at capacity, evict the oldest entry to strictly cap memory
		if len(rl.limiters) >= maxLimiterEntries {
			var oldestKey string
			var oldestTime time.Time
			for k, v := range rl.limiters {
				if oldestKey == "" || v.lastSeen.Before(oldestTime) {
					oldestKey = k
					oldestTime = v.lastSeen
				}
			}
			if oldestKey != "" {
				delete(rl.limiters, oldestKey)
			}
		}
	}

	limiter := rate.NewLimiter(rl.rate, rl.burst)
	rl.limiters[key] = &rateLimiterEntry{
		limiter:  limiter,
		lastSeen: time.Now(),
	}
	return limiter
}

// cleanupLoop periodically removes entries inactive for more than 10 minutes.
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, entry := range rl.limiters {
			if now.Sub(entry.lastSeen) > 10*time.Minute {
				delete(rl.limiters, key)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimit returns Gin middleware that enforces per-IP rate limiting.
func RateLimit(rps rate.Limit, burst int) gin.HandlerFunc {
	return RateLimitWithKey(func(c *gin.Context) string {
		return c.ClientIP()
	}, rps, burst)
}

// RateLimitWithKey returns Gin middleware that enforces rate limiting using a custom key extractor.
// When the rate limit is exceeded, it sets the Retry-After header and responds with HTTP 429.
func RateLimitWithKey(keyFunc func(c *gin.Context) string, rps rate.Limit, burst int) gin.HandlerFunc {
	limiter := NewRateLimiter(rps, burst)

	return func(c *gin.Context) {
		key := keyFunc(c)
		if !limiter.getLimiter(key).Allow() {
			slog.Warn("Rate limit exceeded",
				slog.String("key", key),
				slog.String("path", c.Request.URL.Path),
				slog.String("method", c.Request.Method),
			)
			c.Header("Retry-After", "60")
			response.Error(c, http.StatusTooManyRequests, "RATE_LIMITED", "Too many requests. Please try again later.")
			c.Abort()
			return
		}
		c.Next()
	}
}
