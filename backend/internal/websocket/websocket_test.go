package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"pulsepoll/backend/internal/database/redis"
	"pulsepoll/backend/internal/polls"
	"pulsepoll/backend/internal/votes"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// mockPollFinder provides in-memory poll metadata for testing.
type mockPollFinder struct {
	polls map[string]*polls.Poll
}

func (m *mockPollFinder) FindByID(ctx context.Context, id string) (*polls.Poll, error) {
	p, exists := m.polls[id]
	if !exists || p.IsDeleted {
		return nil, polls.ErrPollNotFound
	}
	return p, nil
}

// mockVoteResultsProvider provides in-memory results snapshots for testing.
type mockVoteResultsProvider struct {
	mu      sync.Mutex
	results map[string]*votes.PollResultsResponse
}

func (m *mockVoteResultsProvider) GetResults(ctx context.Context, pollID string) (*votes.PollResultsResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	res, exists := m.results[pollID]
	if !exists {
		return &votes.PollResultsResponse{
			PollID:     pollID,
			TotalVotes: 0,
			Results:    []votes.OptionResult{},
		}, nil
	}
	return res, nil
}

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestEnvironment(t *testing.T) (*redis.Client, *HubManager, *mockPollFinder, *mockVoteResultsProvider, *Handler, *polls.Poll) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6380"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	redisClient, err := redis.Connect(ctx, redisURL)
	if err != nil {
		t.Skipf("Skipping websocket live tests (Redis unavailable): %v", err)
	}

	pollID := bson.NewObjectID().Hex()
	testPoll := &polls.Poll{
		ID:        bson.NewObjectID(),
		CreatorID: "creator_test",
		Question:  "Best Programming Language?",
		Options: []polls.PollOption{
			{ID: "opt_go", Text: "Go"},
			{ID: "opt_rust", Text: "Rust"},
		},
		Status:    polls.PollStatusActive,
		IsDeleted: false,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	testPoll.ID, _ = bson.ObjectIDFromHex(pollID)

	pollFinder := &mockPollFinder{
		polls: map[string]*polls.Poll{
			pollID: testPoll,
		},
	}

	voteResults := &mockVoteResultsProvider{
		results: map[string]*votes.PollResultsResponse{
			pollID: {
				PollID:     pollID,
				TotalVotes: 10,
				Results: []votes.OptionResult{
					{OptionID: "opt_go", Text: "Go", Votes: 6, Percentage: 60.0},
					{OptionID: "opt_rust", Text: "Rust", Votes: 4, Percentage: 40.0},
				},
			},
		},
	}

	hubManager := NewHubManager(redisClient.Raw())
	handler := NewHandler(hubManager, pollFinder, voteResults, []string{"*"})

	return redisClient, hubManager, pollFinder, voteResults, handler, testPoll
}

func TestWebSocket_Handler_Validation(t *testing.T) {
	_, hubManager, pollFinder, _, handler, _ := setupTestEnvironment(t)
	defer hubManager.Shutdown()

	r := gin.New()
	r.GET("/api/polls/:id/ws", handler.ServeWS)

	t.Run("rejects invalid ObjectID format with HTTP 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/polls/invalid-id/ws", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
		var errResp struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &errResp)
		if errResp.Error.Code != "INVALID_POLL_ID" {
			t.Errorf("expected INVALID_POLL_ID error code, got %v", errResp.Error.Code)
		}
	})

	t.Run("rejects non-existent poll with HTTP 404", func(t *testing.T) {
		nonExistentID := bson.NewObjectID().Hex()
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/polls/%s/ws", nonExistentID), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404 Not Found, got %d", w.Code)
		}
	})

	t.Run("rejects soft-deleted poll with HTTP 404", func(t *testing.T) {
		deletedID := bson.NewObjectID().Hex()
		objID, _ := bson.ObjectIDFromHex(deletedID)
		pollFinder.polls[deletedID] = &polls.Poll{
			ID:        objID,
			Question:  "Deleted?",
			IsDeleted: true,
			Status:    polls.PollStatusActive,
		}

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/polls/%s/ws", deletedID), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404 Not Found for soft-deleted poll, got %d", w.Code)
		}
	})
}

func TestWebSocket_LiveBroadcasting(t *testing.T) {
	redisClient, hubManager, _, _, handler, testPoll := setupTestEnvironment(t)
	defer func() {
		hubManager.Shutdown()
		_ = redisClient.Close()
	}()

	r := gin.New()
	r.GET("/api/polls/:id/ws", handler.ServeWS)

	server := httptest.NewServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + fmt.Sprintf("/api/polls/%s/ws", testPoll.ID.Hex())

	t.Run("single client receives initial results_snapshot and live vote_update", func(t *testing.T) {
		dialer := websocket.Dialer{HandshakeTimeout: 3 * time.Second}
		conn, resp, err := dialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("WebSocket connection failed: %v (status: %v)", err, resp)
		}
		defer conn.Close()

		// 1. First message must be results_snapshot
		_, snapshotMsg, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("failed to read snapshot: %v", err)
		}

		var snapshot votes.ResultsSnapshotEvent
		if err := json.Unmarshal(snapshotMsg, &snapshot); err != nil {
			t.Fatalf("failed to parse snapshot JSON: %v", err)
		}
		if snapshot.Type != votes.EventTypeResultsSnapshot {
			t.Errorf("expected snapshot event type, got %s", snapshot.Type)
		}
		if snapshot.PollID != testPoll.ID.Hex() {
			t.Errorf("expected poll ID %s, got %s", testPoll.ID.Hex(), snapshot.PollID)
		}
		if snapshot.TotalVotes != 10 {
			t.Errorf("expected 10 total votes in snapshot, got %d", snapshot.TotalVotes)
		}

		// 2. Publish a VoteUpdateEvent to Redis channel poll:{pollID}:updates
		updateEvent := votes.VoteUpdateEvent{
			Type:       votes.EventTypeVoteUpdate,
			PollID:     testPoll.ID.Hex(),
			OptionID:   "opt_go",
			Count:      7,
			TotalVotes: 11,
			Timestamp:  time.Now().UTC(),
		}
		eventPayload, _ := json.Marshal(updateEvent)
		channel := fmt.Sprintf("poll:%s:updates", testPoll.ID.Hex())

		ctx := context.Background()
		if err := redisClient.Raw().Publish(ctx, channel, eventPayload).Err(); err != nil {
			t.Fatalf("failed to publish to Redis: %v", err)
		}

		// 3. Client receives vote_update event without page refresh
		conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		_, updateMsg, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("failed to read live vote update: %v", err)
		}

		var receivedUpdate votes.VoteUpdateEvent
		if err := json.Unmarshal(updateMsg, &receivedUpdate); err != nil {
			t.Fatalf("failed to parse vote update: %v", err)
		}
		if receivedUpdate.Type != votes.EventTypeVoteUpdate {
			t.Errorf("expected vote_update type, got %s", receivedUpdate.Type)
		}
		if receivedUpdate.OptionID != "opt_go" || receivedUpdate.Count != 7 || receivedUpdate.TotalVotes != 11 {
			t.Errorf("mismatched update contents: %+v", receivedUpdate)
		}
	})

	t.Run("multiple clients receive identical live updates", func(t *testing.T) {
		dialer := websocket.Dialer{HandshakeTimeout: 3 * time.Second}

		conn1, _, err := dialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("client 1 dial failed: %v", err)
		}
		defer conn1.Close()

		conn2, _, err := dialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("client 2 dial failed: %v", err)
		}
		defer conn2.Close()

		// Both consume their initial snapshots
		_, _, _ = conn1.ReadMessage()
		_, _, _ = conn2.ReadMessage()

		// Publish live vote update
		updateEvent := votes.VoteUpdateEvent{
			Type:       votes.EventTypeVoteUpdate,
			PollID:     testPoll.ID.Hex(),
			OptionID:   "opt_rust",
			Count:      5,
			TotalVotes: 12,
			Timestamp:  time.Now().UTC(),
		}
		eventPayload, _ := json.Marshal(updateEvent)
		channel := fmt.Sprintf("poll:%s:updates", testPoll.ID.Hex())

		ctx := context.Background()
		_ = redisClient.Raw().Publish(ctx, channel, eventPayload).Err()

		// Verify both received update
		conn1.SetReadDeadline(time.Now().Add(3 * time.Second))
		conn2.SetReadDeadline(time.Now().Add(3 * time.Second))

		_, msg1, err1 := conn1.ReadMessage()
		if err1 != nil {
			t.Fatalf("client 1 read error: %v", err1)
		}
		_, msg2, err2 := conn2.ReadMessage()
		if err2 != nil {
			t.Fatalf("client 2 read error: %v", err2)
		}

		var u1, u2 votes.VoteUpdateEvent
		_ = json.Unmarshal(msg1, &u1)
		_ = json.Unmarshal(msg2, &u2)

		if u1.Count != 5 || u2.Count != 5 {
			t.Errorf("expected both clients to receive count 5, got u1=%d, u2=%d", u1.Count, u2.Count)
		}
	})

	t.Run("disconnecting client does not affect remaining clients", func(t *testing.T) {
		dialer := websocket.Dialer{HandshakeTimeout: 3 * time.Second}

		conn1, _, _ := dialer.Dial(wsURL, nil)
		conn2, _, _ := dialer.Dial(wsURL, nil)

		// Drain initial snapshots
		_, _, _ = conn1.ReadMessage()
		_, _, _ = conn2.ReadMessage()

		// Disconnect client 1 explicitly
		_ = conn1.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "leaving"))
		_ = conn1.Close()

		// Allow hub to process unregister
		time.Sleep(50 * time.Millisecond)

		// Publish update
		updateEvent := votes.VoteUpdateEvent{
			Type:       votes.EventTypeVoteUpdate,
			PollID:     testPoll.ID.Hex(),
			OptionID:   "opt_go",
			Count:      8,
			TotalVotes: 13,
			Timestamp:  time.Now().UTC(),
		}
		eventPayload, _ := json.Marshal(updateEvent)
		channel := fmt.Sprintf("poll:%s:updates", testPoll.ID.Hex())

		_ = redisClient.Raw().Publish(context.Background(), channel, eventPayload).Err()

		// Client 2 must still receive the event
		conn2.SetReadDeadline(time.Now().Add(3 * time.Second))
		_, msg2, err := conn2.ReadMessage()
		if err != nil {
			t.Fatalf("remaining client failed to receive update: %v", err)
		}

		var u2 votes.VoteUpdateEvent
		_ = json.Unmarshal(msg2, &u2)
		if u2.Count != 8 {
			t.Errorf("expected count 8 on remaining client, got %d", u2.Count)
		}
		_ = conn2.Close()
	})
}

func TestWebSocket_ConcurrentBroadcasts_RaceSafety(t *testing.T) {
	redisClient, hubManager, _, _, handler, testPoll := setupTestEnvironment(t)
	defer func() {
		hubManager.Shutdown()
		_ = redisClient.Close()
	}()

	r := gin.New()
	r.GET("/api/polls/:id/ws", handler.ServeWS)

	server := httptest.NewServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + fmt.Sprintf("/api/polls/%s/ws", testPoll.ID.Hex())

	const numClients = 5
	const numMessages = 20

	conns := make([]*websocket.Conn, numClients)
	dialer := websocket.Dialer{HandshakeTimeout: 3 * time.Second}

	for i := 0; i < numClients; i++ {
		c, _, err := dialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("client %d failed to connect: %v", i, err)
		}
		conns[i] = c
		defer c.Close()
		// Drain snapshot
		_, _, _ = c.ReadMessage()
	}

	channel := fmt.Sprintf("poll:%s:updates", testPoll.ID.Hex())

	// Concurrently publish messages from multiple goroutines
	var wg sync.WaitGroup
	for m := 1; m <= numMessages; m++ {
		wg.Add(1)
		go func(msgIndex int) {
			defer wg.Done()
			update := votes.VoteUpdateEvent{
				Type:       votes.EventTypeVoteUpdate,
				PollID:     testPoll.ID.Hex(),
				OptionID:   "opt_go",
				Count:      int64(msgIndex),
				TotalVotes: int64(msgIndex),
				Timestamp:  time.Now().UTC(),
			}
			bytes, _ := json.Marshal(update)
			_ = redisClient.Raw().Publish(context.Background(), channel, bytes).Err()
		}(m)
	}

	wg.Wait()

	// Read messages on all clients with deadline
	for i, conn := range conns {
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		receivedCount := 0
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
			receivedCount++
			if receivedCount >= numMessages {
				break
			}
		}
		if receivedCount == 0 {
			t.Errorf("client %d received zero broadcast messages", i)
		}
	}
}
