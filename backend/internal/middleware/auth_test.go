package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pulsepoll/backend/internal/auth"
	"pulsepoll/backend/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestAuthRouter(jwtMgr *auth.JWTManager) *gin.Engine {
	r := gin.New()
	r.GET("/protected", Authenticate(jwtMgr), func(c *gin.Context) {
		userID, _ := GetUserID(c)
		email, _ := GetUserEmail(c)
		c.JSON(http.StatusOK, gin.H{
			"user_id": userID,
			"email":   email,
		})
	})
	return r
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	jwtMgr := auth.NewJWTManager("test-secret-key-1234567890123456", 24)
	r := setupTestAuthRouter(jwtMgr)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
	}

	var errResp response.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if errResp.Error.Code != "UNAUTHORIZED" {
		t.Errorf("expected error code UNAUTHORIZED, got %s", errResp.Error.Code)
	}
}

func TestAuthMiddleware_MalformedHeader(t *testing.T) {
	jwtMgr := auth.NewJWTManager("test-secret-key-1234567890123456", 24)
	r := setupTestAuthRouter(jwtMgr)

	malformedHeaders := []string{
		"Basic abc123xyz",
		"Bearer",
		"Bearer  ",
		"InvalidFormat",
	}

	for _, headerVal := range malformedHeaders {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", headerVal)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("header %q: expected 401, got %d", headerVal, w.Code)
		}
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	jwtMgr := auth.NewJWTManager("test-secret-key-1234567890123456", 24)
	r := setupTestAuthRouter(jwtMgr)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.jwt.token")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	secret := []byte("test-secret-key-1234567890123456")
	now := time.Now().Add(-2 * time.Hour)

	claims := auth.CustomClaims{
		UserID: "user-123",
		Email:  "test@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-123",
			Issuer:    "pulsepoll",
			IssuedAt:  jwt.NewNumericDate(now.Add(-1 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString(secret)

	jwtMgr := auth.NewJWTManager(string(secret), 24)
	r := setupTestAuthRouter(jwtMgr)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for expired token, got %d", w.Code)
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	jwtMgr := auth.NewJWTManager("test-secret-key-1234567890123456", 24)
	r := setupTestAuthRouter(jwtMgr)

	token, err := jwtMgr.Generate("user-999", "valid@example.com")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["user_id"] != "user-999" {
		t.Errorf("expected user_id user-999, got %s", resp["user_id"])
	}
	if resp["email"] != "valid@example.com" {
		t.Errorf("expected email valid@example.com, got %s", resp["email"])
	}
}
