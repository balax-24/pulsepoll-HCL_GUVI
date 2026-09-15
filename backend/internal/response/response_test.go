package response

import (
	"encoding/json"
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
