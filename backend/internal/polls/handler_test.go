package polls

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"pulsepoll/backend/internal/auth"
	"pulsepoll/backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupPollHTTPRouter() (*gin.Engine, *Handler, *auth.JWTManager, *mockPollRepository) {
	repo := newMockPollRepository()
	jwtMgr := auth.NewJWTManager("test-jwt-secret-key-12345678901234", 24)
	svc := NewService(repo)
	h := NewHandler(svc)

	r := gin.New()
	api := r.Group("/api")
	{
		api.GET("/my/polls", middleware.Authenticate(jwtMgr), h.GetMyPolls)

		pollsGroup := api.Group("/polls")
		{
			pollsGroup.GET("/:id", h.GetPublic)

			protected := pollsGroup.Group("")
			protected.Use(middleware.Authenticate(jwtMgr))
			{
				protected.POST("", h.Create)
				protected.PATCH("/:id", h.Update)
				protected.PATCH("/:id/status", h.UpdateStatus)
				protected.POST("/:id/close", h.Close)
				protected.DELETE("/:id", h.Delete)
			}
		}
	}

	return r, h, jwtMgr, repo
}

func TestHandler_CreatePoll(t *testing.T) {
	r, _, jwtMgr, _ := setupPollHTTPRouter()

	token, _ := jwtMgr.Generate("user_100", "user100@example.com")

	t.Run("authenticated create poll success", func(t *testing.T) {
		body := CreatePollRequest{
			Question: "Which database do you prefer?",
			Options:  []string{"PostgreSQL", "MongoDB", "Redis"},
		}
		jsonBody, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/polls", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]PollResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		poll := resp["poll"]
		if poll.ID == "" {
			t.Errorf("expected poll ID")
		}
		if poll.CreatorID != "user_100" {
			t.Errorf("expected creator ID 'user_100', got '%s'", poll.CreatorID)
		}
		if len(poll.Options) != 3 {
			t.Errorf("expected 3 options, got %d", len(poll.Options))
		}
	})

	t.Run("unauthenticated create rejected", func(t *testing.T) {
		body := CreatePollRequest{
			Question: "Unauthenticated poll?",
			Options:  []string{"A", "B"},
		}
		jsonBody, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/polls", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid payload rejected with 400", func(t *testing.T) {
		body := CreatePollRequest{
			Question: "Too few options?",
			Options:  []string{"Only One"},
		}
		jsonBody, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/polls", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestHandler_GetPublicPoll(t *testing.T) {
	r, _, jwtMgr, _ := setupPollHTTPRouter()

	token, _ := jwtMgr.Generate("user_200", "user200@example.com")

	// Create poll
	body := CreatePollRequest{
		Question: "Public polling?",
		Options:  []string{"Yes", "No"},
	}
	jsonBody, _ := json.Marshal(body)
	wCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/polls", bytes.NewBuffer(jsonBody))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	r.ServeHTTP(wCreate, reqCreate)

	var createResp map[string]PollResponse
	_ = json.Unmarshal(wCreate.Body.Bytes(), &createResp)
	pollID := createResp["poll"].ID

	t.Run("public poll accessible without authentication", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/polls/%s", pollID), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]PublicPollResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode public poll: %v", err)
		}
		if resp["poll"].ID != pollID {
			t.Errorf("expected poll ID %s, got %s", pollID, resp["poll"].ID)
		}
	})

	t.Run("nonexistent poll returns 404", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/polls/nonexistent_id", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404 Not Found, got %d", w.Code)
		}
	})
}

func TestHandler_GetMyPolls(t *testing.T) {
	r, _, jwtMgr, _ := setupPollHTTPRouter()

	userA := "user_A"
	userB := "user_B"
	tokenA, _ := jwtMgr.Generate(userA, "a@example.com")
	tokenB, _ := jwtMgr.Generate(userB, "b@example.com")

	// User A creates 2 polls
	for i := 1; i <= 2; i++ {
		b, _ := json.Marshal(CreatePollRequest{
			Question: fmt.Sprintf("Poll %d for User A", i),
			Options:  []string{"Option 1", "Option 2"},
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/polls", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenA))
		r.ServeHTTP(w, req)
	}

	t.Run("creator retrieves own polls", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/my/polls", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenA))
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var listResp ListPollsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &listResp); err != nil {
			t.Fatalf("failed to decode list response: %v", err)
		}
		if listResp.TotalCount != 2 {
			t.Errorf("expected 2 polls, got %d", listResp.TotalCount)
		}
	})

	t.Run("other user has empty list", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/my/polls", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenB))
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}

		var listResp ListPollsResponse
		_ = json.Unmarshal(w.Body.Bytes(), &listResp)
		if listResp.TotalCount != 0 {
			t.Errorf("expected 0 polls for user B, got %d", listResp.TotalCount)
		}
	})

	t.Run("unauthenticated call to /api/my/polls returns 401", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/my/polls", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("excessive page query parameter is capped at 1000", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/my/polls?page=999999999&limit=10", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenA))
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var listResp ListPollsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &listResp); err != nil {
			t.Fatalf("failed to decode list response: %v", err)
		}
		if listResp.Page != 1000 {
			t.Errorf("expected page capped to 1000, got %d", listResp.Page)
		}
	})
}

func TestHandler_Ownership_Authorization(t *testing.T) {
	r, _, jwtMgr, _ := setupPollHTTPRouter()

	userOwner := "owner_id"
	userAttacker := "attacker_id"
	tokenOwner, _ := jwtMgr.Generate(userOwner, "owner@example.com")
	tokenAttacker, _ := jwtMgr.Generate(userAttacker, "attacker@example.com")

	// Create poll as userOwner
	createBody, _ := json.Marshal(CreatePollRequest{
		Question: "Ownership test poll?",
		Options:  []string{"Option 1", "Option 2"},
	})
	wCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/polls", bytes.NewBuffer(createBody))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenOwner))
	r.ServeHTTP(wCreate, reqCreate)

	var createResp map[string]PollResponse
	_ = json.Unmarshal(wCreate.Body.Bytes(), &createResp)
	pollID := createResp["poll"].ID

	// 1. Attacker attempts PATCH -> 403 Forbidden
	t.Run("non-owner cannot PATCH poll", func(t *testing.T) {
		newQ := "Hacked Question?"
		patchBody, _ := json.Marshal(UpdatePollRequest{Question: &newQ})
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/polls/%s", pollID), bytes.NewBuffer(patchBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenAttacker))
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden on PATCH by attacker, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 2. Attacker attempts CLOSE via status -> 403 Forbidden
	t.Run("non-owner cannot close poll via PATCH status", func(t *testing.T) {
		statusBody, _ := json.Marshal(UpdateStatusRequest{Status: PollStatusClosed})
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/polls/%s/status", pollID), bytes.NewBuffer(statusBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenAttacker))
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden on PATCH status by attacker, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 3. Attacker attempts CLOSE via /close -> 403 Forbidden
	t.Run("non-owner cannot close poll via POST close", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/polls/%s/close", pollID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenAttacker))
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden on POST close by attacker, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 4. Attacker attempts DELETE -> 403 Forbidden
	t.Run("non-owner cannot DELETE poll", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/polls/%s", pollID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenAttacker))
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden on DELETE by attacker, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 5. Owner PATCH succeeds
	t.Run("owner can PATCH poll", func(t *testing.T) {
		newQ := "Legitimate owner updated question?"
		patchBody, _ := json.Marshal(UpdatePollRequest{Question: &newQ})
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/polls/%s", pollID), bytes.NewBuffer(patchBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenOwner))
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK on PATCH by owner, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 6. Owner CLOSE succeeds
	t.Run("owner can CLOSE poll", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/polls/%s/close", pollID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenOwner))
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK on CLOSE by owner, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]PollResponse
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["poll"].Status != PollStatusClosed {
			t.Errorf("expected status closed, got %s", resp["poll"].Status)
		}
	})

	// 7. Owner DELETE succeeds
	t.Run("owner can DELETE poll", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/polls/%s", pollID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenOwner))
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK on DELETE by owner, got %d: %s", w.Code, w.Body.String())
		}

		// Subsequent GET returns 404
		wGet := httptest.NewRecorder()
		reqGet := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/polls/%s", pollID), nil)
		r.ServeHTTP(wGet, reqGet)
		if wGet.Code != http.StatusNotFound {
			t.Fatalf("expected 404 Not Found after deletion, got %d", wGet.Code)
		}
	})
}
