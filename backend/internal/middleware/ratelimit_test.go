package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRateLimit_AllowsUpToBurst(t *testing.T) {
	r := gin.New()
	r.Use(RateLimit(1.0, 3))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("request %d expected HTTP 200, got %d", i+1, w.Code)
		}
	}

	// 4th request exceeds burst capacity
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected HTTP 429 on burst exceed, got %d", w.Code)
	}

	retryAfter := w.Header().Get("Retry-After")
	if retryAfter == "" {
		t.Errorf("expected Retry-After header on 429 response")
	}
}

func TestRateLimitWithKey_SeparatesKeys(t *testing.T) {
	r := gin.New()
	// Key by query param "poll"
	r.Use(RateLimitWithKey(func(c *gin.Context) string {
		return c.Query("poll")
	}, 1.0, 2))
	r.GET("/vote", func(c *gin.Context) {
		c.String(http.StatusOK, "VOTED")
	})

	// 2 requests for poll_1 should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/vote?poll=poll_1", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("poll_1 request %d expected 200, got %d", i+1, w.Code)
		}
	}

	// 3rd request for poll_1 should be rate limited
	reqExceed := httptest.NewRequest(http.MethodGet, "/vote?poll=poll_1", nil)
	wExceed := httptest.NewRecorder()
	r.ServeHTTP(wExceed, reqExceed)
	if wExceed.Code != http.StatusTooManyRequests {
		t.Fatalf("poll_1 request 3 expected 429, got %d", wExceed.Code)
	}

	// Request for poll_2 should STILL SUCCEED because it has its own key
	reqPoll2 := httptest.NewRequest(http.MethodGet, "/vote?poll=poll_2", nil)
	wPoll2 := httptest.NewRecorder()
	r.ServeHTTP(wPoll2, reqPoll2)
	if wPoll2.Code != http.StatusOK {
		t.Fatalf("poll_2 request expected 200, got %d", wPoll2.Code)
	}
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	rl := NewRateLimiter(rate.Limit(100), 50)
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				lim := rl.getLimiter("10.0.0.1")
				lim.Allow()
			}
		}(i)
	}

	wg.Wait()
}

func TestRateLimiter_CapacityBounds(t *testing.T) {
	rl := NewRateLimiter(rate.Limit(10), 10)

	// Simulate stale entries
	now := time.Now()
	rl.mu.Lock()
	for i := 0; i < maxLimiterEntries+10; i++ {
		rl.limiters[string(rune(i))] = &rateLimiterEntry{
			limiter:  rate.NewLimiter(10, 10),
			lastSeen: now.Add(-10 * time.Minute),
		}
	}
	rl.mu.Unlock()

	// Calling getLimiter should evict stale entries and keep map bounded
	_ = rl.getLimiter("new-key")

	rl.mu.Lock()
	count := len(rl.limiters)
	rl.mu.Unlock()

	if count > maxLimiterEntries {
		t.Errorf("expected map entries <= %d, got %d", maxLimiterEntries, count)
	}
}

func TestRateLimit_CloudflareTrustedPlatform_StableKey(t *testing.T) {
	r := gin.New()
	r.TrustedPlatform = gin.PlatformCloudflare
	_ = r.SetTrustedProxies([]string{"127.0.0.1", "::1", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"})

	// Matching production auth limiter: 5 requests/minute, burst 5
	r.POST("/api/auth/login", RateLimit(5.0/60.0, 5), func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	const cfClientIP = "203.0.113.195"

	// 5 requests within burst capacity from the same Cloudflare client IP
	// Simulate rotating Anycast egress proxies in X-Forwarded-For and RemoteAddr
	for i := 1; i <= 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
		req.Header.Set("CF-Connecting-IP", cfClientIP)
		req.Header.Set("X-Forwarded-For", fmt.Sprintf("%s, 172.70.%d.1, 10.0.0.%d", cfClientIP, i, i))
		req.RemoteAddr = fmt.Sprintf("10.0.0.%d:45678", i)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("request %d: expected HTTP 200, got %d", i, w.Code)
		}
	}

	// 6th request from the same Cloudflare client must be rejected with 429 after burst is exhausted
	req6 := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req6.Header.Set("CF-Connecting-IP", cfClientIP)
	req6.Header.Set("X-Forwarded-For", fmt.Sprintf("%s, 172.70.99.1, 10.0.0.99", cfClientIP))
	req6.RemoteAddr = "10.0.0.99:45678"

	w6 := httptest.NewRecorder()
	r.ServeHTTP(w6, req6)

	if w6.Code != http.StatusTooManyRequests {
		t.Fatalf("request 6: expected HTTP 429 after exhausting burst, got %d", w6.Code)
	}

	if retryAfter := w6.Header().Get("Retry-After"); retryAfter == "" {
		t.Errorf("request 6: expected Retry-After header on 429 response")
	}

	// Request from a DIFFERENT Cloudflare client IP should succeed (isolated bucket)
	reqOther := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	reqOther.Header.Set("CF-Connecting-IP", "198.51.100.42")
	reqOther.Header.Set("X-Forwarded-For", "198.51.100.42, 172.70.1.1, 10.0.0.1")
	reqOther.RemoteAddr = "10.0.0.1:45678"

	wOther := httptest.NewRecorder()
	r.ServeHTTP(wOther, reqOther)

	if wOther.Code != http.StatusOK {
		t.Fatalf("different client IP expected HTTP 200, got %d", wOther.Code)
	}
}
