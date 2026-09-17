package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

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
