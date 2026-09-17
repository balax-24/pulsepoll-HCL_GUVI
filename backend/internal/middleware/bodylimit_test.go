package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"pulsepoll/backend/internal/response"

	"github.com/gin-gonic/gin"
)

type testRegisterPayload struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func TestRequestBodyLimit_AcceptsSmallBody(t *testing.T) {
	r := gin.New()
	r.Use(RequestBodyLimit(1024)) // 1 KB
	r.POST("/upload", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}
		c.String(http.StatusOK, string(body))
	})

	smallPayload := bytes.Repeat([]byte("a"), 500)
	req := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(smallPayload))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", w.Code)
	}
}

func TestRequestBodyLimit_RejectsOversizedBody(t *testing.T) {
	r := gin.New()
	r.Use(RequestBodyLimit(1024)) // 1 KB
	r.POST("/upload", func(c *gin.Context) {
		_, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.String(http.StatusRequestEntityTooLarge, "body too large")
			return
		}
		c.String(http.StatusOK, "OK")
	})

	oversizedPayload := bytes.Repeat([]byte("a"), 2048) // 2 KB > 1 KB limit
	req := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(oversizedPayload))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected HTTP 413 for oversized payload, got %d", w.Code)
	}
}

func TestRequestBodyLimit_ContentLengthExceedsLimit(t *testing.T) {
	r := gin.New()
	r.Use(RequestBodyLimit(2 << 20)) // 2 MiB limit (2,097,152 bytes)

	handlerCalled := false
	r.POST("/api/auth/register", func(c *gin.Context) {
		handlerCalled = true
		var p testRegisterPayload
		if !response.BindJSON(c, &p, "Request body must contain name, email, and password") {
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 2,097,226 bytes payload (matches diagnostic test)
	const oversizedSize = (2 << 20) + 74
	oversizedBody := bytes.Repeat([]byte("x"), oversizedSize)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(oversizedBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected HTTP 413 for Content-Length > 2 MiB, got %d: %s", w.Code, w.Body.String())
	}

	if handlerCalled {
		t.Errorf("expected handler not to be invoked when Content-Length exceeds limit")
	}

	var errResp response.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != "PAYLOAD_TOO_LARGE" {
		t.Errorf("expected code PAYLOAD_TOO_LARGE, got %s", errResp.Error.Code)
	}
}

func TestRequestBodyLimit_MaxBytesReaderDuringJSONDecode(t *testing.T) {
	r := gin.New()
	r.Use(RequestBodyLimit(2 << 20)) // 2 MiB limit

	r.POST("/api/auth/register", func(c *gin.Context) {
		var p testRegisterPayload
		if !response.BindJSON(c, &p, "Request body must contain name, email, and password") {
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Chunked/streamed body exceeding 2 MiB without known Content-Length (ContentLength = -1)
	largeName := bytes.Repeat([]byte("A"), (2<<20)+500)
	bodyJSON := fmt.Sprintf(`{"name":%q,"email":"test@example.com","password":"secretpassword123"}`, string(largeName))
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(bodyJSON))
	req.ContentLength = -1 // simulate chunked/streaming request
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected HTTP 413 when MaxBytesReader trips during JSON decoding, got %d: %s", w.Code, w.Body.String())
	}

	var errResp response.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != "PAYLOAD_TOO_LARGE" {
		t.Errorf("expected code PAYLOAD_TOO_LARGE, got %s", errResp.Error.Code)
	}
}

func TestRequestBodyLimit_ValidJSONWorks(t *testing.T) {
	r := gin.New()
	r.Use(RequestBodyLimit(2 << 20))

	r.POST("/api/auth/register", func(c *gin.Context) {
		var p testRegisterPayload
		if !response.BindJSON(c, &p, "Request body must contain name, email, and password") {
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"name":   p.Name,
			"email":  p.Email,
		})
	})

	bodyJSON := `{"name":"Jane Doe","email":"jane@example.com","password":"SecurePassword123!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(bodyJSON))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 for valid JSON, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["name"] != "Jane Doe" || resp["email"] != "jane@example.com" {
		t.Errorf("unexpected response body: %v", resp)
	}
}

func TestRequestBodyLimit_MalformedAndMissingFieldsReturn400(t *testing.T) {
	r := gin.New()
	r.Use(RequestBodyLimit(2 << 20))

	r.POST("/api/auth/register", func(c *gin.Context) {
		var p testRegisterPayload
		if !response.BindJSON(c, &p, "Request body must contain name, email, and password") {
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	testCases := []struct {
		name        string
		body        string
		expectedMsg string
	}{
		{
			name:        "malformed syntax",
			body:        `{"name": "broken-json, `,
			expectedMsg: "Request body must contain name, email, and password",
		},
		{
			name:        "missing required password field",
			body:        `{"name": "Alice", "email": "alice@example.com"}`,
			expectedMsg: "Request body must contain name, email, and password",
		},
		{
			name:        "empty JSON object",
			body:        `{}`,
			expectedMsg: "Request body must contain name, email, and password",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected HTTP 400, got %d: %s", w.Code, w.Body.String())
			}

			var errResp response.ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
				t.Fatalf("failed to decode error response: %v", err)
			}

			if errResp.Error.Code != "INVALID_REQUEST_PAYLOAD" {
				t.Errorf("expected code INVALID_REQUEST_PAYLOAD, got %s", errResp.Error.Code)
			}
			if errResp.Error.Message != tc.expectedMsg {
				t.Errorf("expected message %q, got %q", tc.expectedMsg, errResp.Error.Message)
			}
		})
	}
}
