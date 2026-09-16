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
