package votes

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Vote represents the persistent vote audit record in MongoDB.
// It connects a voter session with an option in a specific poll without duplicating poll schemas.
type Vote struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	PollID    string        `bson:"poll_id" json:"poll_id"`
	OptionID  string        `bson:"option_id" json:"option_id"`
	VoterID   string        `bson:"voter_id" json:"voter_id"`
	CreatedAt time.Time     `bson:"created_at" json:"created_at"`
}

// CastVoteRequest represents the JSON request payload sent by audience members.
type CastVoteRequest struct {
	OptionID string `json:"option_id" binding:"required"`
}

// CastVoteResponse represents the acknowledgement payload returned after a successful vote.
type CastVoteResponse struct {
	PollID   string `json:"poll_id"`
	OptionID string `json:"option_id"`
	VoterID  string `json:"voter_id"`
	Message  string `json:"message"`
}

// OptionResult encapsulates live vote counts and calculated percentage for an option.
type OptionResult struct {
	OptionID   string  `json:"option_id"`
	Text       string  `json:"text"`
	Votes      int64   `json:"votes"`
	Percentage float64 `json:"percentage"`
}

// PollResultsResponse represents the full live results envelope returned by GET /api/polls/:id/results.
type PollResultsResponse struct {
	PollID     string         `json:"poll_id"`
	TotalVotes int64          `json:"total_votes"`
	Results    []OptionResult `json:"results"`
}

// Realtime event types for WebSocket and Redis Pub/Sub broadcasting.
const (
	EventTypeResultsSnapshot = "results_snapshot"
	EventTypeVoteUpdate      = "vote_update"
	EventTypePollClosed      = "poll_closed"
)

// VoteUpdateEvent represents the live event published to Redis Pub/Sub
// and forwarded over WebSockets when a vote is recorded.
// Strict privacy: Never contains voter IDs, creator info, or JWTs.
type VoteUpdateEvent struct {
	Type       string    `json:"type"`
	PollID     string    `json:"poll_id"`
	OptionID   string    `json:"option_id"`
	Count      int64     `json:"count"`
	TotalVotes int64     `json:"total_votes"`
	Timestamp  time.Time `json:"timestamp"`
}

// SnapshotOption represents an option within a results snapshot payload.
// Exposes both 'count' and 'votes' for complete client compatibility.
type SnapshotOption struct {
	OptionID   string  `json:"option_id"`
	Text       string  `json:"text"`
	Count      int64   `json:"count"`
	Votes      int64   `json:"votes"`
	Percentage float64 `json:"percentage"`
}

// ResultsSnapshotEvent represents the full results payload sent immediately
// upon WebSocket connection to prevent missed-update races.
type ResultsSnapshotEvent struct {
	Type       string           `json:"type"`
	PollID     string           `json:"poll_id"`
	Options    []SnapshotOption `json:"options"`
	TotalVotes int64            `json:"total_votes"`
}

// PollClosedEvent represents a notification that the poll has been closed.
type PollClosedEvent struct {
	Type      string    `json:"type"`
	PollID    string    `json:"poll_id"`
	Timestamp time.Time `json:"timestamp"`
}
