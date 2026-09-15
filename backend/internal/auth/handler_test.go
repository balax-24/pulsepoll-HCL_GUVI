package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"pulsepoll/backend/internal/response"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestHTTPRouter() (*gin.Engine, *Handler, *JWTManager) {
	repo := newMockUserRepository()
	jwtMgr := NewJWTManager("super-secret-key-12345678901234", 24)
	svc := NewService(repo, jwtMgr)
	h := NewHandler(svc)

	r := gin.New()
	api := r.Group("/api/auth")
	{
		api.POST("/register", h.Register)
		api.POST("/login", h.Login)
		api.GET("/me", func(c *gin.Context) {
			// Simulate auth middleware behavior for handler unit test
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authorization required")
				return
			}
			tokenStr := authHeader[len("Bearer "):]
			claims, err := jwtMgr.Validate(tokenStr)
			if err != nil {
				response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid token")
				return
			}
			c.Set("auth_user_id", claims.UserID)
			h.Me(c)
		})
	}

	return r, h, jwtMgr
}

func TestHandler_Register_Success(t *testing.T) {
	r, _, _ := setupTestHTTPRouter()

	body := RegisterRequest{
		Name:     "Test User",
		Email:    "test@example.com",
		Password: "Password1234!",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]UserResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["user"].Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", resp["user"].Email)
	}
}

func TestHandler_Register_DuplicateConflict(t *testing.T) {
	r, _, _ := setupTestHTTPRouter()

	body := RegisterRequest{
		Name:     "Test User",
		Email:    "dup@example.com",
		Password: "Password1234!",
	}
	jsonBody, _ := json.Marshal(body)

	// First registration
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(jsonBody))
	req1.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusCreated {
		t.Fatalf("first registration failed: %d", w1.Code)
	}

	// Duplicate registration
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(jsonBody))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d: %s", w2.Code, w2.Body.String())
	}

	var errResp response.ErrorResponse
	_ = json.Unmarshal(w2.Body.Bytes(), &errResp)
	if errResp.Error.Code != "EMAIL_EXISTS" {
		t.Errorf("expected error code EMAIL_EXISTS, got %s", errResp.Error.Code)
	}
}

func TestHandler_Login_And_Me(t *testing.T) {
	r, _, _ := setupTestHTTPRouter()

	// Register account
	regBody := RegisterRequest{
		Name:     "Login User",
		Email:    "login@example.com",
		Password: "SecretPassword123!",
	}
	jsonReg, _ := json.Marshal(regBody)
	wReg := httptest.NewRecorder()
	reqReg := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(jsonReg))
	reqReg.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(wReg, reqReg)

	// Login with correct credentials
	loginBody := LoginRequest{
		Email:    "login@example.com",
		Password: "SecretPassword123!",
	}
	jsonLogin, _ := json.Marshal(loginBody)
	wLogin := httptest.NewRecorder()
	reqLogin := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(jsonLogin))
	reqLogin.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(wLogin, reqLogin)

	if wLogin.Code != http.StatusOK {
		t.Fatalf("expected 200 OK login, got %d: %s", wLogin.Code, wLogin.Body.String())
	}

	var loginResp LoginResponse
	if err := json.Unmarshal(wLogin.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("failed to decode login response: %v", err)
	}

	if loginResp.Token == "" {
		t.Fatalf("expected JWT token, got empty string")
	}

	// Call /me with valid JWT token
	wMe := httptest.NewRecorder()
	reqMe := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	reqMe.Header.Set("Authorization", "Bearer "+loginResp.Token)
	r.ServeHTTP(wMe, reqMe)

	if wMe.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on /me, got %d: %s", wMe.Code, wMe.Body.String())
	}

	var meResp UserResponse
	if err := json.Unmarshal(wMe.Body.Bytes(), &meResp); err != nil {
		t.Fatalf("failed to decode /me response: %v", err)
	}

	if meResp.Email != "login@example.com" {
		t.Errorf("expected email login@example.com, got %s", meResp.Email)
	}

	// Call /me without JWT token -> 401
	wUnauth := httptest.NewRecorder()
	reqUnauth := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	r.ServeHTTP(wUnauth, reqUnauth)

	if wUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", wUnauth.Code)
	}
}
