package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"pulsepoll/backend/internal/polls"
	"pulsepoll/backend/internal/response"
	"pulsepoll/backend/internal/votes"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// PollFinder provides read access to poll metadata for pre-validation.
type PollFinder interface {
	FindByID(ctx context.Context, id string) (*polls.Poll, error)
}

// VoteResultsProvider provides live aggregated results snapshot.
type VoteResultsProvider interface {
	GetResults(ctx context.Context, pollID string) (*votes.PollResultsResponse, error)
}

// Handler handles WebSocket upgrades and connection lifecycles for poll viewers.
type Handler struct {
	hubManager     *HubManager
	pollFinder     PollFinder
	voteService    VoteResultsProvider
	upgrader       websocket.Upgrader
	allowedOrigins []string
}

// NewHandler constructs a new WebSocket Handler.
func NewHandler(
	hubManager *HubManager,
	pollFinder PollFinder,
	voteService VoteResultsProvider,
	allowedOrigins []string,
) *Handler {
	h := &Handler{
		hubManager:     hubManager,
		pollFinder:     pollFinder,
		voteService:    voteService,
		allowedOrigins: allowedOrigins,
	}

	h.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			// Allow all in development or wildcard
			if len(allowedOrigins) == 0 || (len(allowedOrigins) == 1 && allowedOrigins[0] == "*") {
				return true
			}
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true // Direct clients, curl, tests
			}
			for _, allowed := range allowedOrigins {
				if strings.EqualFold(origin, allowed) {
					return true
				}
			}
			return false
		},
	}

	return h
}

// ServeWS handles GET /api/polls/:id/ws.
// It pre-validates poll existence and status before WebSocket upgrade.
// Upon successful connection, it guarantees immediate delivery of a results_snapshot,
// preventing missed-update race conditions before continuous Pub/Sub streaming starts.
func (h *Handler) ServeWS(c *gin.Context) {
	pollID := strings.TrimSpace(c.Param("id"))
	if pollID == "" {
		response.Error(c, http.StatusBadRequest, "INVALID_POLL_ID", "Poll ID is required")
		return
	}

	if _, err := bson.ObjectIDFromHex(pollID); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_POLL_ID", "Invalid poll ID format")
		return
	}

	// 1. Pre-validate poll existence and deletion status
	poll, err := h.pollFinder.FindByID(c.Request.Context(), pollID)
	if err != nil {
		if errors.Is(err, polls.ErrPollNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "Poll not found")
			return
		}
		slog.Error("Failed to look up poll before WebSocket upgrade",
			slog.String("poll_id", pollID),
			slog.String("error", err.Error()),
		)
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve poll")
		return
	}

	// 2. Ensure Redis Pub/Sub subscription exists before upgrading / taking snapshot
	hub, err := h.hubManager.GetOrCreateHub(c.Request.Context(), pollID)
	if err != nil {
		slog.Error("Failed to initialize WebSocket poll hub",
			slog.String("poll_id", pollID),
			slog.String("error", err.Error()),
		)
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to initialize realtime hub")
		return
	}

	// 3. Retrieve initial live results snapshot from Redis/MongoDB
	results, err := h.voteService.GetResults(c.Request.Context(), pollID)
	if err != nil {
		slog.Error("Failed to retrieve initial results snapshot",
			slog.String("poll_id", pollID),
			slog.String("error", err.Error()),
		)
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve results snapshot")
		return
	}

	// 4. Upgrade HTTP connection to WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.Warn("WebSocket upgrade rejected or failed",
			slog.String("poll_id", pollID),
			slog.String("error", err.Error()),
		)
		return
	}

	client := NewClient(hub, conn, pollID)

	// 5. Register client with hub
	hub.Register(client)

	// 6. Build and enqueue initial snapshot event into client's outbound buffer
	snapshotOptions := make([]votes.SnapshotOption, len(results.Results))
	for i, r := range results.Results {
		snapshotOptions[i] = votes.SnapshotOption{
			OptionID:   r.OptionID,
			Text:       r.Text,
			Count:      r.Votes,
			Votes:      r.Votes,
			Percentage: r.Percentage,
		}
	}

	snapshot := votes.ResultsSnapshotEvent{
		Type:       votes.EventTypeResultsSnapshot,
		PollID:     pollID,
		Options:    snapshotOptions,
		TotalVotes: results.TotalVotes,
	}

	snapshotBytes, err := json.Marshal(snapshot)
	if err == nil {
		client.send <- snapshotBytes
	}

	// 7. If poll is closed, enqueue poll_closed event so client is informed
	if poll.Status == polls.PollStatusClosed {
		closedEvent := votes.PollClosedEvent{
			Type:      votes.EventTypePollClosed,
			PollID:    pollID,
			Timestamp: time.Now().UTC(),
		}
		if closedBytes, err := json.Marshal(closedEvent); err == nil {
			client.send <- closedBytes
		}
	}

	// 8. Launch concurrent pump goroutines
	go client.writePump()
	go client.readPump()

	slog.Info("WebSocket connection established successfully",
		slog.String("poll_id", pollID),
	)
}
