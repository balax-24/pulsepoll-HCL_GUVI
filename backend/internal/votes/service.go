package votes

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"pulsepoll/backend/internal/polls"

	"go.mongodb.org/mongo-driver/v2/bson"
)

var (
	// ErrPollNotFound indicates the target poll does not exist or was soft-deleted.
	ErrPollNotFound = errors.New("poll not found")

	// ErrPollClosed indicates voting cannot occur because the poll has been closed.
	ErrPollClosed = errors.New("this poll is closed and no longer accepting votes")

	// ErrPollExpired indicates voting cannot occur because the poll's deadline has passed.
	ErrPollExpired = errors.New("this poll has expired and is no longer accepting votes")

	// ErrInvalidOption indicates the selected option ID does not belong to the target poll.
	ErrInvalidOption = errors.New("option does not belong to this poll")

	// ErrInvalidInput indicates missing or invalid fields in the vote submission.
	ErrInvalidInput = errors.New("invalid vote payload")
)

// PollFinder abstracts reading poll metadata for vote validation.
type PollFinder interface {
	FindByID(ctx context.Context, id string) (*polls.Poll, error)
}

// Service coordinates vote validation, persistent audit logging in MongoDB,
// and atomic live counter updates in Redis.
type Service struct {
	pollRepo  PollFinder
	voteRepo  VoteRepository
	redisRepo VoteCounterStore
}

// NewService constructs a vote Service.
func NewService(pollRepo PollFinder, voteRepo VoteRepository, redisRepo VoteCounterStore) *Service {
	return &Service{
		pollRepo:  pollRepo,
		voteRepo:  voteRepo,
		redisRepo: redisRepo,
	}
}

// CastVote validates and executes a vote submission:
// 1. Validates poll presence and accessibility.
// 2. Enforces active lifecycle status and expiration deadlines.
// 3. Verifies that the option belongs to the poll.
// 4. Performs an application-level duplicate check.
// 5. Persists the vote to MongoDB (with unique compound index as the concurrency barrier).
// 6. Atomically increments the Redis live counter using HINCRBY.
func (s *Service) CastVote(ctx context.Context, pollID, optionID, voterID string) (*CastVoteResponse, error) {
	pollID = strings.TrimSpace(pollID)
	optionID = strings.TrimSpace(optionID)
	voterID = strings.TrimSpace(voterID)

	if pollID == "" || optionID == "" || voterID == "" {
		return nil, ErrInvalidInput
	}

	// 1. Retrieve poll
	poll, err := s.pollRepo.FindByID(ctx, pollID)
	if err != nil {
		if errors.Is(err, polls.ErrPollNotFound) {
			return nil, ErrPollNotFound
		}
		return nil, fmt.Errorf("failed to retrieve poll: %w", err)
	}

	// 2. Reject closed poll
	if poll.Status == polls.PollStatusClosed {
		return nil, ErrPollClosed
	}

	// 3. Reject expired poll
	if poll.ExpiresAt != nil && time.Now().UTC().After(*poll.ExpiresAt) {
		return nil, ErrPollExpired
	}

	// 4. Validate that option belongs to this poll
	optionFound := false
	for _, opt := range poll.Options {
		if opt.ID == optionID {
			optionFound = true
			break
		}
	}
	if !optionFound {
		return nil, ErrInvalidOption
	}

	// 5. Fast application-level duplicate check
	hasVoted, err := s.voteRepo.HasVoted(ctx, pollID, voterID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify voter status: %w", err)
	}
	if hasVoted {
		return nil, ErrDuplicateVote
	}

	// 6. Ensure Redis hash counters are initialized for all poll options
	_ = s.ensureRedisInitialized(ctx, poll)

	// 7. Persist vote audit document to MongoDB
	// The unique compound index on (poll_id, voter_id) serves as the definitive race-condition barrier
	vote := &Vote{
		ID:        bson.NewObjectID(),
		PollID:    pollID,
		OptionID:  optionID,
		VoterID:   voterID,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.voteRepo.Create(ctx, vote); err != nil {
		if errors.Is(err, ErrDuplicateVote) {
			return nil, ErrDuplicateVote
		}
		return nil, fmt.Errorf("failed to record vote in durable storage: %w", err)
	}

	// 8. Atomically increment Redis live counter (HINCRBY)
	// Guaranteed execution only after successful MongoDB persistence
	if err := s.incrementRedisWithRetry(ctx, pollID, optionID, 3); err != nil {
		slog.Error("Failed to increment Redis live counter after vote persistence",
			slog.String("poll_id", pollID),
			slog.String("option_id", optionID),
			slog.String("error", err.Error()),
		)
	}

	return &CastVoteResponse{
		PollID:   pollID,
		OptionID: optionID,
		VoterID:  voterID,
		Message:  "Vote recorded successfully",
	}, nil
}

// GetResults retrieves live vote tallies and calculates percentages:
// It queries the Redis live counters where available, falling back to MongoDB if Redis is unreachable.
func (s *Service) GetResults(ctx context.Context, pollID string) (*PollResultsResponse, error) {
	pollID = strings.TrimSpace(pollID)
	if pollID == "" {
		return nil, ErrPollNotFound
	}

	poll, err := s.pollRepo.FindByID(ctx, pollID)
	if err != nil {
		if errors.Is(err, polls.ErrPollNotFound) {
			return nil, ErrPollNotFound
		}
		return nil, fmt.Errorf("failed to retrieve poll: %w", err)
	}

	optionIDs := make([]string, len(poll.Options))
	for i, opt := range poll.Options {
		optionIDs[i] = opt.ID
	}

	// Ensure Redis counters are populated
	_ = s.ensureRedisInitialized(ctx, poll)

	// Fetch live counts from Redis
	counts, totalVotes, err := s.redisRepo.GetOptionCounts(ctx, pollID, optionIDs)
	if err != nil {
		slog.Warn("Redis unavailable for results query; falling back to durable MongoDB counts",
			slog.String("poll_id", pollID),
			slog.String("error", err.Error()),
		)
		counts, err = s.voteRepo.GetVoteCountsByPoll(ctx, pollID)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve durable fallback vote counts: %w", err)
		}
		totalVotes = 0
		for _, v := range counts {
			totalVotes += v
		}
	}

	// Build result objects with zero-safe percentage computation
	results := make([]OptionResult, len(poll.Options))
	for i, opt := range poll.Options {
		v := counts[opt.ID]
		var pct float64 = 0.0
		if totalVotes > 0 {
			pct = math.Round((float64(v)/float64(totalVotes))*10000) / 100.0
		}
		results[i] = OptionResult{
			OptionID:   opt.ID,
			Text:       opt.Text,
			Votes:      v,
			Percentage: pct,
		}
	}

	return &PollResultsResponse{
		PollID:     pollID,
		TotalVotes: totalVotes,
		Results:    results,
	}, nil
}

// ensureRedisInitialized verifies if Redis contains live counters for the poll.
// If not (e.g. after Redis restart or upon first access), it populates the hash with
// durable counts from MongoDB using HSETNX to avoid race-condition overwrites.
func (s *Service) ensureRedisInitialized(ctx context.Context, poll *polls.Poll) error {
	pollID := poll.ID.Hex()
	hasCounters, err := s.redisRepo.HasCounters(ctx, pollID)
	if err == nil && hasCounters {
		return nil
	}

	// Fetch current durable counts from MongoDB
	durableCounts, err := s.voteRepo.GetVoteCountsByPoll(ctx, pollID)
	if err != nil {
		durableCounts = make(map[string]int64)
	}

	optionIDs := make([]string, len(poll.Options))
	for i, opt := range poll.Options {
		optionIDs[i] = opt.ID
	}

	return s.redisRepo.InitializeCounters(ctx, pollID, optionIDs, durableCounts)
}

// incrementRedisWithRetry attempts atomic HINCRBY with exponential backoff retries.
func (s *Service) incrementRedisWithRetry(ctx context.Context, pollID, optionID string, maxRetries int) error {
	var err error
	backoff := 10 * time.Millisecond

	for attempt := 1; attempt <= maxRetries; attempt++ {
		_, err = s.redisRepo.IncrementVote(ctx, pollID, optionID)
		if err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			backoff *= 2
		}
	}

	return err
}
