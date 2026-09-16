package votes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pulsepoll/backend/internal/polls"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupVotesHTTPRouter() (*gin.Engine, *mockPollFinder, *mockVoteRepository, *mockVoteCounterStore, *polls.Poll) {
	pollFinder := newMockPollFinder()
	voteRepo := newMockVoteRepository()
	redisRepo := newMockVoteCounterStore()

	testPoll := &polls.Poll{
		ID:        bson.NewObjectID(),
		CreatorID: "creator_1",
		Question:  "Favorite Editor?",
		Options: []polls.PollOption{
			{ID: "opt_vscode", Text: "VS Code"},
			{ID: "opt_neovim", Text: "Neovim"},
		},
		Status:    polls.PollStatusActive,
		IsDeleted: false,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	pollFinder.polls[testPoll.ID.Hex()] = testPoll

	svc := NewService(pollFinder, voteRepo, redisRepo)
	h := NewHandler(svc)

	r := gin.New()
	api := r.Group("/api/polls")
	{
		api.POST("/:id/vote", h.CastVote)
		api.GET("/:id/results", h.GetResults)
	}

	return r, pollFinder, voteRepo, redisRepo, testPoll
}

func TestHandler_CastVote(t *testing.T) {
	r, _, _, _, testPoll := setupVotesHTTPRouter()
	pollID := testPoll.ID.Hex()

	t.Run("successful vote generates session cookie and records vote", func(t *testing.T) {
		body := CastVoteRequest{OptionID: "opt_vscode"}
		jsonBody, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/polls/%s/vote", pollID), bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		// Verify cookie was returned
		cookie := w.Header().Get("Set-Cookie")
		if cookie == "" {
			t.Errorf("expected Set-Cookie header for anonymous voter session")
		}

		voterHeader := w.Header().Get(VoterHeaderName)
		if voterHeader == "" {
			t.Errorf("expected %s header in response", VoterHeaderName)
		}

		var resp CastVoteResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.OptionID != "opt_vscode" {
			t.Errorf("expected option_id 'opt_vscode', got '%s'", resp.OptionID)
		}
	})

	t.Run("duplicate vote from same voter rejected with 409 Conflict", func(t *testing.T) {
		voterID := "explicit_voter_123"
		body := CastVoteRequest{OptionID: "opt_vscode"}
		jsonBody, _ := json.Marshal(body)

		// First vote
		w1 := httptest.NewRecorder()
		req1 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/polls/%s/vote", pollID), bytes.NewBuffer(jsonBody))
		req1.Header.Set("Content-Type", "application/json")
		req1.Header.Set(VoterHeaderName, voterID)
		r.ServeHTTP(w1, req1)
		if w1.Code != http.StatusOK {
			t.Fatalf("expected 200 OK on first vote, got %d", w1.Code)
		}

		// Second vote with same voter ID
		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/polls/%s/vote", pollID), bytes.NewBuffer(jsonBody))
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set(VoterHeaderName, voterID)
		r.ServeHTTP(w2, req2)

		if w2.Code != http.StatusConflict {
			t.Fatalf("expected 409 Conflict on duplicate vote, got %d: %s", w2.Code, w2.Body.String())
		}
	})

	t.Run("vote for nonexistent option rejected with 400 Bad Request", func(t *testing.T) {
		body := CastVoteRequest{OptionID: "unknown_option"}
		jsonBody, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/polls/%s/vote", pollID), bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("vote on nonexistent poll returns 404 Not Found", func(t *testing.T) {
		body := CastVoteRequest{OptionID: "opt_vscode"}
		jsonBody, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/polls/unknown_id/vote", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404 Not Found, got %d", w.Code)
		}
	})
}

func TestHandler_GetResults(t *testing.T) {
	r, _, _, _, testPoll := setupVotesHTTPRouter()
	pollID := testPoll.ID.Hex()

	t.Run("get results returns valid schema and zero tallies initially", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/polls/%s/results", pollID), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var res PollResultsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
			t.Fatalf("failed to decode results response: %v", err)
		}

		if res.PollID != pollID {
			t.Errorf("expected poll_id %s, got %s", pollID, res.PollID)
		}
		if res.TotalVotes != 0 {
			t.Errorf("expected total_votes 0, got %d", res.TotalVotes)
		}
		if len(res.Results) != 2 {
			t.Fatalf("expected 2 options, got %d", len(res.Results))
		}
	})
}
