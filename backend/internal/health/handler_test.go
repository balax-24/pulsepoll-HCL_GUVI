package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type mockPinger struct {
	err error
}

func (m *mockPinger) Ping(ctx context.Context) error {
	return m.err
}

func TestHealthHandler_AllHealthy(t *testing.T) {
	mongoMock := &mockPinger{err: nil}
	redisMock := &mockPinger{err: nil}

	h := NewHandler(mongoMock, redisMock)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/health", nil)

	h.Check(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("expected status 'ok', got %s", resp.Status)
	}
	if resp.Services.MongoDB != "ok" || resp.Services.Redis != "ok" {
		t.Errorf("expected both services 'ok', got %+v", resp.Services)
	}
}

func TestHealthHandler_MongoFailing(t *testing.T) {
	mongoMock := &mockPinger{err: errors.New("connection refused")}
	redisMock := &mockPinger{err: nil}

	h := NewHandler(mongoMock, redisMock)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/health", nil)

	h.Check(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", w.Code)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "degraded" {
		t.Errorf("expected status 'degraded', got %s", resp.Status)
	}
	if resp.Services.MongoDB != "unavailable" {
		t.Errorf("expected mongodb 'unavailable', got %s", resp.Services.MongoDB)
	}
	if resp.Services.Redis != "ok" {
		t.Errorf("expected redis 'ok', got %s", resp.Services.Redis)
	}
}

func TestHealthHandler_RedisFailing(t *testing.T) {
	mongoMock := &mockPinger{err: nil}
	redisMock := &mockPinger{err: errors.New("redis timeout")}

	h := NewHandler(mongoMock, redisMock)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/health", nil)

	h.Check(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", w.Code)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "degraded" {
		t.Errorf("expected status 'degraded', got %s", resp.Status)
	}
	if resp.Services.MongoDB != "ok" {
		t.Errorf("expected mongodb 'ok', got %s", resp.Services.MongoDB)
	}
	if resp.Services.Redis != "unavailable" {
		t.Errorf("expected redis 'unavailable', got %s", resp.Services.Redis)
	}
}
