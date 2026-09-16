package votes

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"pulsepoll/backend/internal/polls"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// mockPollFinder implements PollFinder for testing.
type mockPollFinder struct {
	polls map[string]*polls.Poll
}

func newMockPollFinder() *mockPollFinder {
	return &mockPollFinder{polls: make(map[string]*polls.Poll)}
}

func (m *mockPollFinder) FindByID(ctx context.Context, id string) (*polls.Poll, error) {
	p, ok := m.polls[id]
	if !ok || p.IsDeleted {
		return nil, polls.ErrPollNotFound
	}
	copyP := *p
	return &copyP, nil
}

// mockVoteRepository implements VoteRepository in-memory.
type mockVoteRepository struct {
	mu    sync.Mutex
	votes map[string]*Vote // key: pollID + ":" + voterID
}

func newMockVoteRepository() *mockVoteRepository {
	return &mockVoteRepository{votes: make(map[string]*Vote)}
}

func (m *mockVoteRepository) Create(ctx context.Context, vote *Vote) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := vote.PollID + ":" + vote.VoterID
	if _, exists := m.votes[key]; exists {
		return ErrDuplicateVote
	}
	if vote.ID.IsZero() {
		vote.ID = bson.NewObjectID()
	}
	m.votes[key] = vote
	return nil
}

func (m *mockVoteRepository) HasVoted(ctx context.Context, pollID, voterID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := pollID + ":" + voterID
	_, exists := m.votes[key]
	return exists, nil
}

func (m *mockVoteRepository) GetVoteCountsByPoll(ctx context.Context, pollID string) (map[string]int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	counts := make(map[string]int64)
	for _, v := range m.votes {
		if v.PollID == pollID {
			counts[v.OptionID]++
		}
	}
	return counts, nil
}

func (m *mockVoteRepository) CountByPollID(ctx context.Context, pollID string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var count int64
	for _, v := range m.votes {
		if v.PollID == pollID {
			count++
		}
	}
	return count, nil
}

func (m *mockVoteRepository) EnsureIndexes(ctx context.Context) error {
	return nil
}

// mockVoteCounterStore implements VoteCounterStore in-memory.
type mockVoteCounterStore struct {
	mu               sync.Mutex
	counts           map[string]map[string]int64 // pollID -> (optionID -> count)
	publishedUpdates []*VoteUpdateEvent
	publishedClosed  []*PollClosedEvent
	simulateErr      bool
	simulatePubErr   bool
}

func newMockVoteCounterStore() *mockVoteCounterStore {
	return &mockVoteCounterStore{
		counts: make(map[string]map[string]int64),
	}
}

func (m *mockVoteCounterStore) IncrementVote(ctx context.Context, pollID, optionID string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.simulateErr {
		return 0, errors.New("redis connection failure")
	}

	if _, ok := m.counts[pollID]; !ok {
		m.counts[pollID] = make(map[string]int64)
	}
	m.counts[pollID][optionID]++
	return m.counts[pollID][optionID], nil
}

func (m *mockVoteCounterStore) GetOptionCounts(ctx context.Context, pollID string, optionIDs []string) (map[string]int64, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.simulateErr {
		return nil, 0, errors.New("redis connection failure")
	}

	pollMap, exists := m.counts[pollID]
	counts := make(map[string]int64, len(optionIDs))
	var total int64 = 0

	for _, optID := range optionIDs {
		c := int64(0)
		if exists {
			c = pollMap[optID]
		}
		counts[optID] = c
		total += c
	}
	return counts, total, nil
}

func (m *mockVoteCounterStore) InitializeCounters(ctx context.Context, pollID string, optionIDs []string, initialCounts map[string]int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.counts[pollID]; !ok {
		m.counts[pollID] = make(map[string]int64)
		for _, optID := range optionIDs {
			m.counts[pollID][optID] = initialCounts[optID]
		}
	}
	return nil
}

func (m *mockVoteCounterStore) HasCounters(ctx context.Context, pollID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, exists := m.counts[pollID]
	return exists, nil
}

func (m *mockVoteCounterStore) ResetCounters(ctx context.Context, pollID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.counts, pollID)
	return nil
}

func (m *mockVoteCounterStore) PublishVoteUpdate(ctx context.Context, pollID string, event *VoteUpdateEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.simulateErr || m.simulatePubErr {
		return errors.New("redis publish failure")
	}
	m.publishedUpdates = append(m.publishedUpdates, event)
	return nil
}

func (m *mockVoteCounterStore) PublishPollClosed(ctx context.Context, pollID string, event *PollClosedEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.simulateErr || m.simulatePubErr {
		return errors.New("redis publish failure")
	}
	m.publishedClosed = append(m.publishedClosed, event)
	return nil
}

// ---------------------- Service Unit Tests ----------------------

func setupServiceTestFixture() (*Service, *mockPollFinder, *mockVoteRepository, *mockVoteCounterStore, *polls.Poll) {
	pollFinder := newMockPollFinder()
	voteRepo := newMockVoteRepository()
	redisRepo := newMockVoteCounterStore()

	testPoll := &polls.Poll{
		ID:        bson.NewObjectID(),
		CreatorID: "creator_1",
		Question:  "Best programming language?",
		Options: []polls.PollOption{
			{ID: "opt_go", Text: "Go"},
			{ID: "opt_rust", Text: "Rust"},
			{ID: "opt_python", Text: "Python"},
		},
		Status:    polls.PollStatusActive,
		IsDeleted: false,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	pollFinder.polls[testPoll.ID.Hex()] = testPoll

	svc := NewService(pollFinder, voteRepo, redisRepo)
	return svc, pollFinder, voteRepo, redisRepo, testPoll
}

func TestService_CastVote_Validation(t *testing.T) {
	svc, pollFinder, _, _, testPoll := setupServiceTestFixture()
	ctx := context.Background()
	pollID := testPoll.ID.Hex()

	t.Run("valid vote succeeds", func(t *testing.T) {
		resp, err := svc.CastVote(ctx, pollID, "opt_go", "voter_1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if resp.PollID != pollID || resp.OptionID != "opt_go" || resp.VoterID != "voter_1" {
			t.Errorf("unexpected response content: %+v", resp)
		}
	})

	t.Run("nonexistent poll returns ErrPollNotFound", func(t *testing.T) {
		_, err := svc.CastVote(ctx, "nonexistent_id", "opt_go", "voter_2")
		if !errors.Is(err, ErrPollNotFound) {
			t.Errorf("expected ErrPollNotFound, got %v", err)
		}
	})

	t.Run("soft-deleted poll returns ErrPollNotFound", func(t *testing.T) {
		delPoll := &polls.Poll{
			ID:        bson.NewObjectID(),
			Question:  "Deleted poll?",
			Options:   []polls.PollOption{{ID: "opt_1", Text: "A"}},
			Status:    polls.PollStatusActive,
			IsDeleted: true,
		}
		pollFinder.polls[delPoll.ID.Hex()] = delPoll

		_, err := svc.CastVote(ctx, delPoll.ID.Hex(), "opt_1", "voter_3")
		if !errors.Is(err, ErrPollNotFound) {
			t.Errorf("expected ErrPollNotFound, got %v", err)
		}
	})

	t.Run("closed poll rejects voting with ErrPollClosed", func(t *testing.T) {
		closedPoll := &polls.Poll{
			ID:        bson.NewObjectID(),
			Question:  "Closed poll?",
			Options:   []polls.PollOption{{ID: "opt_closed_1", Text: "A"}},
			Status:    polls.PollStatusClosed,
			IsDeleted: false,
		}
		pollFinder.polls[closedPoll.ID.Hex()] = closedPoll

		_, err := svc.CastVote(ctx, closedPoll.ID.Hex(), "opt_closed_1", "voter_4")
		if !errors.Is(err, ErrPollClosed) {
			t.Errorf("expected ErrPollClosed, got %v", err)
		}
	})

	t.Run("expired poll rejects voting with ErrPollExpired", func(t *testing.T) {
		past := time.Now().UTC().Add(-2 * time.Hour)
		expiredPoll := &polls.Poll{
			ID:        bson.NewObjectID(),
			Question:  "Expired poll?",
			Options:   []polls.PollOption{{ID: "opt_exp_1", Text: "A"}},
			Status:    polls.PollStatusActive,
			IsDeleted: false,
			ExpiresAt: &past,
		}
		pollFinder.polls[expiredPoll.ID.Hex()] = expiredPoll

		_, err := svc.CastVote(ctx, expiredPoll.ID.Hex(), "opt_exp_1", "voter_5")
		if !errors.Is(err, ErrPollExpired) {
			t.Errorf("expected ErrPollExpired, got %v", err)
		}
	})

	t.Run("nonexistent option rejected with ErrInvalidOption", func(t *testing.T) {
		_, err := svc.CastVote(ctx, pollID, "invalid_option_id", "voter_6")
		if !errors.Is(err, ErrInvalidOption) {
			t.Errorf("expected ErrInvalidOption, got %v", err)
		}
	})

	t.Run("empty voter ID rejected with ErrInvalidInput", func(t *testing.T) {
		_, err := svc.CastVote(ctx, pollID, "opt_go", "")
		if !errors.Is(err, ErrInvalidInput) {
			t.Errorf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("empty option ID rejected with ErrInvalidInput", func(t *testing.T) {
		_, err := svc.CastVote(ctx, pollID, "", "voter_7")
		if !errors.Is(err, ErrInvalidInput) {
			t.Errorf("expected ErrInvalidInput, got %v", err)
		}
	})
}

func TestService_DuplicateVoting(t *testing.T) {
	svc, _, _, _, testPoll := setupServiceTestFixture()
	ctx := context.Background()
	pollID := testPoll.ID.Hex()

	voterID := "voter_duplicate_test"

	// First vote should succeed
	_, err := svc.CastVote(ctx, pollID, "opt_go", voterID)
	if err != nil {
		t.Fatalf("first vote failed: %v", err)
	}

	// Second vote from same voter should be rejected
	_, err = svc.CastVote(ctx, pollID, "opt_rust", voterID)
	if !errors.Is(err, ErrDuplicateVote) {
		t.Errorf("expected ErrDuplicateVote for second vote, got %v", err)
	}
}

func TestService_GetResults(t *testing.T) {
	svc, _, _, redisRepo, testPoll := setupServiceTestFixture()
	ctx := context.Background()
	pollID := testPoll.ID.Hex()

	t.Run("zero votes handles safely without divide by zero", func(t *testing.T) {
		res, err := svc.GetResults(ctx, pollID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if res.TotalVotes != 0 {
			t.Errorf("expected total votes 0, got %d", res.TotalVotes)
		}
		if len(res.Results) != 3 {
			t.Fatalf("expected 3 options, got %d", len(res.Results))
		}
		for _, opt := range res.Results {
			if opt.Votes != 0 || opt.Percentage != 0.0 {
				t.Errorf("expected 0 votes and 0.0%% for %s, got %d and %f", opt.OptionID, opt.Votes, opt.Percentage)
			}
		}
	})

	t.Run("accurate tallies and percentages with mixed votes", func(t *testing.T) {
		// Cast votes: 2 for Go, 1 for Rust, 0 for Python (Total = 3)
		_, _ = svc.CastVote(ctx, pollID, "opt_go", "voter_a")
		_, _ = svc.CastVote(ctx, pollID, "opt_go", "voter_b")
		_, _ = svc.CastVote(ctx, pollID, "opt_rust", "voter_c")

		res, err := svc.GetResults(ctx, pollID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if res.TotalVotes != 3 {
			t.Errorf("expected total votes 3, got %d", res.TotalVotes)
		}

		// Check options
		optionMap := make(map[string]OptionResult)
		for _, r := range res.Results {
			optionMap[r.OptionID] = r
		}

		goRes := optionMap["opt_go"]
		if goRes.Votes != 2 || goRes.Percentage != 66.67 {
			t.Errorf("expected Go: 2 votes, 66.67%%, got %d, %f", goRes.Votes, goRes.Percentage)
		}

		rustRes := optionMap["opt_rust"]
		if rustRes.Votes != 1 || rustRes.Percentage != 33.33 {
			t.Errorf("expected Rust: 1 vote, 33.33%%, got %d, %f", rustRes.Votes, rustRes.Percentage)
		}

		pyRes := optionMap["opt_python"]
		if pyRes.Votes != 0 || pyRes.Percentage != 0.0 {
			t.Errorf("expected Python: 0 votes, 0.0%%, got %d, %f", pyRes.Votes, pyRes.Percentage)
		}
	})

	t.Run("fallback to durable MongoDB when Redis is unavailable", func(t *testing.T) {
		redisRepo.simulateErr = true
		res, err := svc.GetResults(ctx, pollID)
		if err != nil {
			t.Fatalf("expected successful fallback, got %v", err)
		}
		if res.TotalVotes != 3 {
			t.Errorf("expected 3 total votes on fallback, got %d", res.TotalVotes)
		}
		redisRepo.simulateErr = false
	})

	t.Run("vote publication triggers VoteUpdateEvent with correct counters", func(t *testing.T) {
		svcNew, _, _, redisNew, testPollNew := setupServiceTestFixture()
		pollIDNew := testPollNew.ID.Hex()

		resp, err := svcNew.CastVote(ctx, pollIDNew, "opt_go", "voter_live_1")
		if err != nil {
			t.Fatalf("expected vote success, got %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil response")
		}

		redisNew.mu.Lock()
		defer redisNew.mu.Unlock()

		if len(redisNew.publishedUpdates) != 1 {
			t.Fatalf("expected 1 published event, got %d", len(redisNew.publishedUpdates))
		}
		event := redisNew.publishedUpdates[0]
		if event.Type != EventTypeVoteUpdate {
			t.Errorf("expected event type %s, got %s", EventTypeVoteUpdate, event.Type)
		}
		if event.PollID != pollIDNew {
			t.Errorf("expected pollID %s, got %s", pollIDNew, event.PollID)
		}
		if event.OptionID != "opt_go" {
			t.Errorf("expected optionID opt_go, got %s", event.OptionID)
		}
		if event.Count != 1 {
			t.Errorf("expected count 1, got %d", event.Count)
		}
		if event.TotalVotes != 1 {
			t.Errorf("expected total votes 1, got %d", event.TotalVotes)
		}
		if event.Timestamp.IsZero() {
			t.Error("expected non-zero event timestamp")
		}
	})

	t.Run("failed PubSub does not fail the vote (best-effort delivery)", func(t *testing.T) {
		svcNew, _, _, redisNew, testPollNew := setupServiceTestFixture()
		pollIDNew := testPollNew.ID.Hex()

		// Simulate Redis Publish failure specifically
		redisNew.simulatePubErr = true

		resp, err := svcNew.CastVote(ctx, pollIDNew, "opt_go", "voter_resilient")
		if err != nil {
			t.Fatalf("expected vote to succeed even if PubSub delivery fails, got error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil response")
		}

		// Result should still be stored in Redis and MongoDB
		results, err := svcNew.GetResults(ctx, pollIDNew)
		if err != nil {
			t.Fatalf("expected results to be retrievable, got %v", err)
		}
		if results.TotalVotes != 1 {
			t.Errorf("expected 1 total vote, got %d", results.TotalVotes)
		}
	})

	t.Run("duplicate vote is rejected and never publishes to PubSub", func(t *testing.T) {
		svcNew, _, _, redisNew, testPollNew := setupServiceTestFixture()
		pollIDNew := testPollNew.ID.Hex()

		// First vote succeeds
		_, err := svcNew.CastVote(ctx, pollIDNew, "opt_go", "voter_dup_test")
		if err != nil {
			t.Fatalf("first vote failed: %v", err)
		}

		redisNew.mu.Lock()
		initialPubCount := len(redisNew.publishedUpdates)
		redisNew.mu.Unlock()

		// Duplicate vote attempt
		_, err = svcNew.CastVote(ctx, pollIDNew, "opt_go", "voter_dup_test")
		if !errors.Is(err, ErrDuplicateVote) {
			t.Fatalf("expected ErrDuplicateVote, got %v", err)
		}

		redisNew.mu.Lock()
		defer redisNew.mu.Unlock()
		if len(redisNew.publishedUpdates) != initialPubCount {
			t.Errorf("duplicate vote published an event! expected count %d, got %d", initialPubCount, len(redisNew.publishedUpdates))
		}
	})
}
