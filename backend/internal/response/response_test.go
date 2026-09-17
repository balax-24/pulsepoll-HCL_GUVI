package response

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestErrorResponse(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Error(c, http.StatusBadRequest, "BAD_REQUEST", "invalid payload")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}

	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode JSON error response: %v", err)
	}

	if resp.Error.Code != "BAD_REQUEST" {
		t.Errorf("expected code BAD_REQUEST, got %s", resp.Error.Code)
	}
	if resp.Error.Message != "invalid payload" {
		t.Errorf("expected message 'invalid payload', got %s", resp.Error.Message)
	}
}

func TestJSONResponse(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	type SampleData struct {
		Greeting string `json:"greeting"`
	}

	JSON(c, http.StatusOK, SampleData{Greeting: "hello"})

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp SampleData
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}

	if resp.Greeting != "hello" {
		t.Errorf("expected greeting hello, got %s", resp.Greeting)
	}
}

func TestBindJSON_Success(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(`{"name":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	type Data struct {
		Name string `json:"name" binding:"required"`
	}
	var d Data
	ok := BindJSON(c, &d, "validation error")
	if !ok {
		t.Fatalf("expected BindJSON to return true")
	}
	if d.Name != "test" {
		t.Errorf("expected name 'test', got %s", d.Name)
	}
}

func TestBindJSON_MaxBytesError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// Create body that exceeds 10 bytes limit
	body := bytes.NewBufferString(`{"name":"this is a long string that exceeds limit"}`)
	req := httptest.NewRequest(http.MethodPost, "/test", http.MaxBytesReader(w, io.NopCloser(body), 10))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	type Data struct {
		Name string `json:"name"`
	}
	var d Data
	ok := BindJSON(c, &d, "validation error")
	if ok {
		t.Fatalf("expected BindJSON to return false for oversized body")
	}
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected HTTP 413, got %d", w.Code)
	}

	var errResp ErrorResponse
	_ = json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp.Error.Code != "PAYLOAD_TOO_LARGE" {
		t.Errorf("expected PAYLOAD_TOO_LARGE, got %s", errResp.Error.Code)
	}
}

func TestBindJSON_MalformedPayload(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(`{not-json}`))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	type Data struct {
		Name string `json:"name"`
	}
	var d Data
	ok := BindJSON(c, &d, "custom malformed message")
	if ok {
		t.Fatalf("expected BindJSON to return false for malformed JSON")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected HTTP 400, got %d", w.Code)
	}

	var errResp ErrorResponse
	_ = json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp.Error.Code != "INVALID_REQUEST_PAYLOAD" {
		t.Errorf("expected INVALID_REQUEST_PAYLOAD, got %s", errResp.Error.Code)
	}
	if errResp.Error.Message != "custom malformed message" {
		t.Errorf("expected 'custom malformed message', got %s", errResp.Error.Message)
	}
}
