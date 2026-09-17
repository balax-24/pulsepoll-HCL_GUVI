package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"pulsepoll/backend/internal/middleware"
	"pulsepoll/backend/internal/response"

	"github.com/gin-gonic/gin"
)

type prodVerificationPayload struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// TestProductionEquivalent_CloudflareAndBodyLimit verifies the exact production behavior:
// 1. Cloudflare TrustedPlatform with rotating Anycast proxy IPs extracting stable CF-Connecting-IP
//    and rejecting request 6 with HTTP 429 after burst 5 is exhausted.
// 2. Oversized Content-Length (>2 MiB) immediately returning HTTP 413 PAYLOAD_TOO_LARGE.
// 3. Chunked/streamed body (>2 MiB) tripping MaxBytesReader and returning HTTP 413 PAYLOAD_TOO_LARGE.
func TestProductionEquivalent_CloudflareAndBodyLimit(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.TrustedPlatform = gin.PlatformCloudflare
	// Production trusted reverse proxies (Render / Cloudflare RFC1918 + loopback)
	_ = r.SetTrustedProxies([]string{"127.0.0.1", "::1", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"})

	// Production global body limit (2 MiB)
	r.Use(middleware.RequestBodyLimit(2 << 20))

	// Production auth rate limit: 5 requests/min, burst 5
	authLimiter := middleware.RateLimit(5.0/60.0, 5)

	r.POST("/api/auth/register", authLimiter, func(c *gin.Context) {
		var req prodVerificationPayload
		if !response.BindJSON(c, &req, "Request body must contain name, email, and password") {
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"client": c.ClientIP(),
		})
	})

	t.Run("Cloudflare_RotatingProxies_Burst5_Request6_429", func(t *testing.T) {
		const clientIP = "203.0.113.195"
		validBody := `{"name":"Test User","email":"test@example.com","password":"Password123!"}`

		// Requests 1–5: rotating proxy IPs in X-Forwarded-For and RemoteAddr, identical CF-Connecting-IP
		for i := 1; i <= 5; i++ {
			req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(validBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("CF-Connecting-IP", clientIP)
			// Rotating Anycast edge IP on each connection
			cfProxyIP := fmt.Sprintf("172.70.%d.1", i*10)
			req.Header.Set("X-Forwarded-For", fmt.Sprintf("%s, %s, 10.0.0.1", clientIP, cfProxyIP))
			req.RemoteAddr = fmt.Sprintf("10.0.0.1:%d", 40000+i)

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("request %d: expected HTTP 200, got %d: %s", i, w.Code, w.Body.String())
			}

			var resp map[string]string
			_ = json.Unmarshal(w.Body.Bytes(), &resp)
			if resp["client"] != clientIP {
				t.Errorf("request %d: expected client IP %s, got %s", i, clientIP, resp["client"])
			}
		}

		// Request 6: same client IP, new rotating proxy IP -> MUST be HTTP 429
		req6 := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(validBody))
		req6.Header.Set("Content-Type", "application/json")
		req6.Header.Set("CF-Connecting-IP", clientIP)
		req6.Header.Set("X-Forwarded-For", fmt.Sprintf("%s, 172.70.99.1, 10.0.0.1", clientIP))
		req6.RemoteAddr = "10.0.0.1:40099"

		w6 := httptest.NewRecorder()
		r.ServeHTTP(w6, req6)

		if w6.Code != http.StatusTooManyRequests {
			t.Fatalf("request 6: expected HTTP 429 Too Many Requests, got %d: %s", w6.Code, w6.Body.String())
		}

		if retryAfter := w6.Header().Get("Retry-After"); retryAfter == "" {
			t.Errorf("request 6: expected Retry-After header on 429 response")
		}

		var errResp response.ErrorResponse
		if err := json.Unmarshal(w6.Body.Bytes(), &errResp); err != nil {
			t.Fatalf("failed to decode 429 JSON response: %v", err)
		}
		if errResp.Error.Code != "RATE_LIMITED" {
			t.Errorf("request 6: expected error code RATE_LIMITED, got %s", errResp.Error.Code)
		}

		// Request from a DIFFERENT client IP: must still be allowed (isolated bucket)
		reqOther := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(validBody))
		reqOther.Header.Set("Content-Type", "application/json")
		reqOther.Header.Set("CF-Connecting-IP", "198.51.100.77")
		reqOther.Header.Set("X-Forwarded-For", "198.51.100.77, 172.70.1.1, 10.0.0.1")
		reqOther.RemoteAddr = "10.0.0.1:40001"

		wOther := httptest.NewRecorder()
		r.ServeHTTP(wOther, reqOther)

		if wOther.Code != http.StatusOK {
			t.Fatalf("separate client IP expected HTTP 200, got %d", wOther.Code)
		}
	})

	t.Run("ContentLength_GreaterThan_2MiB_Returns_413", func(t *testing.T) {
		// 2,097,226 bytes (exact size from the production diagnostic report)
		oversizedBytes := bytes.Repeat([]byte("x"), 2097226)
		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(oversizedBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("CF-Connecting-IP", "198.51.100.88")
		req.RemoteAddr = "10.0.0.1:40001"

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("expected HTTP 413 Request Entity Too Large, got %d: %s", w.Code, w.Body.String())
		}

		var errResp response.ErrorResponse
		if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
			t.Fatalf("failed to decode 413 error response: %v", err)
		}

		if errResp.Error.Code != "PAYLOAD_TOO_LARGE" {
			t.Errorf("expected error code PAYLOAD_TOO_LARGE, got %s", errResp.Error.Code)
		}
		if errResp.Error.Message != "Request payload exceeds maximum allowed size (2 MB)" {
			t.Errorf("unexpected error message: %s", errResp.Error.Message)
		}
	})

	t.Run("Chunked_GreaterThan_2MiB_Returns_413", func(t *testing.T) {
		// Chunked stream (>2 MiB) with ContentLength = -1
		largeValue := bytes.Repeat([]byte("A"), 2097226)
		bodyJSON := fmt.Sprintf(`{"name":%q,"email":"chunked@example.com","password":"secretpass123"}`, string(largeValue))
		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(bodyJSON))
		req.ContentLength = -1 // simulate Transfer-Encoding: chunked without known Content-Length
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("CF-Connecting-IP", "198.51.100.99")
		req.RemoteAddr = "10.0.0.1:40001"

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("expected HTTP 413 when chunked stream exceeds 2 MiB, got %d: %s", w.Code, w.Body.String())
		}

		var errResp response.ErrorResponse
		if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
			t.Fatalf("failed to decode 413 error response: %v", err)
		}

		if errResp.Error.Code != "PAYLOAD_TOO_LARGE" {
			t.Errorf("expected error code PAYLOAD_TOO_LARGE, got %s", errResp.Error.Code)
		}
		if errResp.Error.Message != "Request payload exceeds maximum allowed size (2 MB)" {
			t.Errorf("unexpected error message: %s", errResp.Error.Message)
		}
	})
}
