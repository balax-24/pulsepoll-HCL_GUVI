package polls

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// mockPollRepository implements PollRepository with an in-memory thread-safe map.
type mockPollRepository struct {
	polls map[string]*Poll
}

func newMockPollRepository() *mockPollRepository {
	return &mockPollRepository{
		polls: make(map[string]*Poll),
	}
}

func (m *mockPollRepository) Create(ctx context.Context, poll *Poll) error {
	if poll.ID.IsZero() {
		poll.ID = bson.NewObjectID()
	}
	idHex := poll.ID.Hex()
	m.polls[idHex] = poll
	return nil
}

func (m *mockPollRepository) FindByID(ctx context.Context, id string) (*Poll, error) {
	poll, exists := m.polls[id]
	if !exists || poll.IsDeleted {
		return nil, ErrPollNotFound
	}
	// Return a shallow copy
	copyPoll := *poll
	return &copyPoll, nil
}

func (m *mockPollRepository) FindByCreatorID(ctx context.Context, creatorID string, limit, offset int64) ([]*Poll, int64, error) {
	var results []*Poll
	for _, poll := range m.polls {
		if poll.CreatorID == creatorID && !poll.IsDeleted {
			copyPoll := *poll
			results = append(results, &copyPoll)
		}
	}
	total := int64(len(results))

	// Simple pagination slicing
	start := offset
	if start > total {
		return []*Poll{}, total, nil
	}
	end := start + limit
	if end > total {
		end = total
	}

	return results[start:end], total, nil
}

func (m *mockPollRepository) Update(ctx context.Context, poll *Poll) error {
	idHex := poll.ID.Hex()
	existing, exists := m.polls[idHex]
	if !exists || existing.IsDeleted {
		return ErrPollNotFound
	}
	m.polls[idHex] = poll
	return nil
}

func (m *mockPollRepository) UpdateStatus(ctx context.Context, id string, status PollStatus) (*Poll, error) {
	poll, exists := m.polls[id]
	if !exists || poll.IsDeleted {
		return nil, ErrPollNotFound
	}
	poll.Status = status
	poll.UpdatedAt = time.Now().UTC()
	return poll, nil
}

func (m *mockPollRepository) SoftDelete(ctx context.Context, id string) error {
	poll, exists := m.polls[id]
	if !exists || poll.IsDeleted {
		return ErrPollNotFound
	}
	now := time.Now().UTC()
	poll.IsDeleted = true
	poll.DeletedAt = &now
	poll.UpdatedAt = now
	return nil
}

func (m *mockPollRepository) EnsureIndexes(ctx context.Context) error {
	return nil
}

// ---------------------- Service Tests ----------------------

func TestService_CreatePoll(t *testing.T) {
	repo := newMockPollRepository()
	svc := NewService(repo)
	ctx := context.Background()

	t.Run("valid poll creation", func(t *testing.T) {
		req := CreatePollRequest{
			Question: "What is your favorite Go framework?",
			Options:  []string{"Gin", "Chi", "Fiber"},
		}
		creatorID := "user_123"

		resp, err := svc.CreatePoll(ctx, creatorID, req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if resp.ID == "" {
			t.Errorf("expected non-empty poll ID")
		}
		if resp.CreatorID != creatorID {
			t.Errorf("expected creatorID %s, got %s", creatorID, resp.CreatorID)
		}
		if resp.Question != req.Question {
			t.Errorf("expected question %s, got %s", req.Question, resp.Question)
		}
		if len(resp.Options) != 3 {
			t.Fatalf("expected 3 options, got %d", len(resp.Options))
		}
		if resp.Status != PollStatusActive {
			t.Errorf("expected status %s, got %s", PollStatusActive, resp.Status)
		}

		// Verify stable option IDs were generated
		seenOptIDs := make(map[string]bool)
		for _, opt := range resp.Options {
			if opt.ID == "" {
				t.Errorf("expected option ID to be non-empty")
			}
			if seenOptIDs[opt.ID] {
				t.Errorf("duplicate option ID detected: %s", opt.ID)
			}
			seenOptIDs[opt.ID] = true
		}
	})

	t.Run("unauthenticated create rejected", func(t *testing.T) {
		req := CreatePollRequest{
			Question: "What is your favorite Go framework?",
			Options:  []string{"Gin", "Chi"},
		}
		_, err := svc.CreatePoll(ctx, "", req)
		if !errors.Is(err, ErrForbidden) {
			t.Errorf("expected ErrForbidden, got %v", err)
		}
	})

	t.Run("invalid question length", func(t *testing.T) {
		testCases := []struct {
			name     string
			question string
		}{
			{"empty question", ""},
			{"too short question (<3 chars)", "Hi"},
			{"too long question (>255 chars)", strings.Repeat("A", 256)},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := CreatePollRequest{
					Question: tc.question,
					Options:  []string{"Option 1", "Option 2"},
				}
				_, err := svc.CreatePoll(ctx, "user_123", req)
				if !errors.Is(err, ErrInvalidInput) {
					t.Errorf("expected ErrInvalidInput, got %v", err)
				}
			})
		}
	})

	t.Run("too few options (<2)", func(t *testing.T) {
		req := CreatePollRequest{
			Question: "Single option test?",
			Options:  []string{"Only One"},
		}
		_, err := svc.CreatePoll(ctx, "user_123", req)
		if !errors.Is(err, ErrInvalidInput) {
			t.Errorf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("too many options (>10)", func(t *testing.T) {
		var options []string
		for i := 1; i <= 11; i++ {
			options = append(options, fmt.Sprintf("Option %d", i))
		}
		req := CreatePollRequest{
			Question: "Eleven options test?",
			Options:  options,
		}
		_, err := svc.CreatePoll(ctx, "user_123", req)
		if !errors.Is(err, ErrInvalidInput) {
			t.Errorf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("empty option text rejected", func(t *testing.T) {
		req := CreatePollRequest{
			Question: "Empty option test?",
			Options:  []string{"Valid", "   "},
		}
		_, err := svc.CreatePoll(ctx, "user_123", req)
		if !errors.Is(err, ErrInvalidInput) {
			t.Errorf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("duplicate option text rejected", func(t *testing.T) {
		req := CreatePollRequest{
			Question: "Duplicate option test?",
			Options:  []string{"Docker", "docker"}, // Case-insensitive duplicate
		}
		_, err := svc.CreatePoll(ctx, "user_123", req)
		if !errors.Is(err, ErrInvalidInput) {
			t.Errorf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("expiration date in past rejected", func(t *testing.T) {
		past := time.Now().UTC().Add(-1 * time.Hour)
		req := CreatePollRequest{
			Question:  "Expired poll test?",
			Options:   []string{"Yes", "No"},
			ExpiresAt: &past,
		}
		_, err := svc.CreatePoll(ctx, "user_123", req)
		if !errors.Is(err, ErrInvalidInput) {
			t.Errorf("expected ErrInvalidInput, got %v", err)
		}
	})
}

func TestService_GetPublicPoll(t *testing.T) {
	repo := newMockPollRepository()
	svc := NewService(repo)
	ctx := context.Background()

	created, err := svc.CreatePoll(ctx, "user_123", CreatePollRequest{
		Question: "Public poll test?",
		Options:  []string{"Choice A", "Choice B"},
	})
	if err != nil {
		t.Fatalf("failed to create poll: %v", err)
	}

	t.Run("public poll exists", func(t *testing.T) {
		publicResp, err := svc.GetPublicPoll(ctx, created.ID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if publicResp.ID != created.ID {
			t.Errorf("expected poll ID %s, got %s", created.ID, publicResp.ID)
		}
		if publicResp.Question != created.Question {
			t.Errorf("expected question %s, got %s", created.Question, publicResp.Question)
		}
		if len(publicResp.Options) != 2 {
			t.Errorf("expected 2 options, got %d", len(publicResp.Options))
		}
		if publicResp.Status != PollStatusActive {
			t.Errorf("expected status active, got %s", publicResp.Status)
		}
	})

	t.Run("public poll missing returns 404", func(t *testing.T) {
		_, err := svc.GetPublicPoll(ctx, "nonexistent_id")
		if !errors.Is(err, ErrPollNotFound) {
			t.Errorf("expected ErrPollNotFound, got %v", err)
		}
	})

	t.Run("closed poll is viewable publicly with closed status", func(t *testing.T) {
		_, err := svc.ClosePoll(ctx, "user_123", created.ID)
		if err != nil {
			t.Fatalf("failed to close poll: %v", err)
		}

		publicResp, err := svc.GetPublicPoll(ctx, created.ID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if publicResp.Status != PollStatusClosed {
			t.Errorf("expected status %s, got %s", PollStatusClosed, publicResp.Status)
		}
	})
}

func TestService_GetCreatorPolls(t *testing.T) {
	repo := newMockPollRepository()
	svc := NewService(repo)
	ctx := context.Background()

	userA := "user_A"
	userB := "user_B"

	// Create 2 polls for User A and 1 poll for User B
	_, _ = svc.CreatePoll(ctx, userA, CreatePollRequest{
		Question: "Poll 1 for User A",
		Options:  []string{"A", "B"},
	})
	_, _ = svc.CreatePoll(ctx, userA, CreatePollRequest{
		Question: "Poll 2 for User A",
		Options:  []string{"C", "D"},
	})
	_, _ = svc.CreatePoll(ctx, userB, CreatePollRequest{
		Question: "Poll 1 for User B",
		Options:  []string{"E", "F"},
	})

	t.Run("user A retrieves only their polls", func(t *testing.T) {
		resp, err := svc.GetCreatorPolls(ctx, userA, 1, 10)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if resp.TotalCount != 2 {
			t.Errorf("expected total count 2, got %d", resp.TotalCount)
		}
		for _, p := range resp.Polls {
			if p.CreatorID != userA {
				t.Errorf("expected creator ID %s, got %s", userA, p.CreatorID)
			}
		}
	})

	t.Run("user B retrieves only their polls", func(t *testing.T) {
		resp, err := svc.GetCreatorPolls(ctx, userB, 1, 10)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if resp.TotalCount != 1 {
			t.Errorf("expected total count 1, got %d", resp.TotalCount)
		}
		if resp.Polls[0].CreatorID != userB {
			t.Errorf("expected creator ID %s, got %s", userB, resp.Polls[0].CreatorID)
		}
	})

	t.Run("unauthenticated retrieval rejected", func(t *testing.T) {
		_, err := svc.GetCreatorPolls(ctx, "", 1, 10)
		if !errors.Is(err, ErrForbidden) {
			t.Errorf("expected ErrForbidden, got %v", err)
		}
	})
}

func TestService_UpdatePoll(t *testing.T) {
	repo := newMockPollRepository()
	svc := NewService(repo)
	ctx := context.Background()

	ownerID := "creator_owner"
	attackerID := "creator_attacker"

	poll, err := svc.CreatePoll(ctx, ownerID, CreatePollRequest{
		Question: "Original question?",
		Options:  []string{"Option 1", "Option 2"},
	})
	if err != nil {
		t.Fatalf("failed to create poll: %v", err)
	}

	opt1ID := poll.Options[0].ID
	opt2ID := poll.Options[1].ID

	t.Run("owner can update question and preserve option IDs", func(t *testing.T) {
		newQ := "Updated question?"
		resp, err := svc.UpdatePoll(ctx, ownerID, poll.ID, UpdatePollRequest{
			Question: &newQ,
			Options: []UpdateOptionRequest{
				{ID: opt1ID, Text: "Renamed Option 1"},
				{ID: opt2ID, Text: "Renamed Option 2"},
			},
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if resp.Question != newQ {
			t.Errorf("expected question %s, got %s", newQ, resp.Question)
		}
		if resp.Options[0].ID != opt1ID || resp.Options[0].Text != "Renamed Option 1" {
			t.Errorf("option 1 ID altered or text mismatch")
		}
		if resp.Options[1].ID != opt2ID || resp.Options[1].Text != "Renamed Option 2" {
			t.Errorf("option 2 ID altered or text mismatch")
		}
	})

	t.Run("non-owner receives ErrForbidden", func(t *testing.T) {
		newQ := "Hacked question?"
		_, err := svc.UpdatePoll(ctx, attackerID, poll.ID, UpdatePollRequest{
			Question: &newQ,
		})
		if !errors.Is(err, ErrForbidden) {
			t.Errorf("expected ErrForbidden, got %v", err)
		}
	})

	t.Run("nonexistent poll returns ErrPollNotFound", func(t *testing.T) {
		newQ := "Nonexistent poll question?"
		_, err := svc.UpdatePoll(ctx, ownerID, "nonexistent_id", UpdatePollRequest{
			Question: &newQ,
		})
		if !errors.Is(err, ErrPollNotFound) {
			t.Errorf("expected ErrPollNotFound, got %v", err)
		}
	})

	t.Run("invalid update rejected (empty question)", func(t *testing.T) {
		badQ := "  "
		_, err := svc.UpdatePoll(ctx, ownerID, poll.ID, UpdatePollRequest{
			Question: &badQ,
		})
		if !errors.Is(err, ErrInvalidInput) {
			t.Errorf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("invalid update rejected (duplicate option text)", func(t *testing.T) {
		_, err := svc.UpdatePoll(ctx, ownerID, poll.ID, UpdatePollRequest{
			Options: []UpdateOptionRequest{
				{ID: opt1ID, Text: "Same"},
				{ID: opt2ID, Text: "same"},
			},
		})
		if !errors.Is(err, ErrInvalidInput) {
			t.Errorf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("closed poll rejects updates", func(t *testing.T) {
		_, err := svc.ClosePoll(ctx, ownerID, poll.ID)
		if err != nil {
			t.Fatalf("failed to close poll: %v", err)
		}

		newQ := "Update after close?"
		_, err = svc.UpdatePoll(ctx, ownerID, poll.ID, UpdatePollRequest{
			Question: &newQ,
		})
		if !errors.Is(err, ErrPollClosed) {
			t.Errorf("expected ErrPollClosed, got %v", err)
		}
	})
}

func TestService_ClosePoll(t *testing.T) {
	repo := newMockPollRepository()
	svc := NewService(repo)
	ctx := context.Background()

	ownerID := "user_close_owner"
	attackerID := "user_close_attacker"

	poll, err := svc.CreatePoll(ctx, ownerID, CreatePollRequest{
		Question: "Close test question?",
		Options:  []string{"A", "B"},
	})
	if err != nil {
		t.Fatalf("failed to create poll: %v", err)
	}

	t.Run("non-owner cannot close poll", func(t *testing.T) {
		_, err := svc.ClosePoll(ctx, attackerID, poll.ID)
		if !errors.Is(err, ErrForbidden) {
			t.Errorf("expected ErrForbidden, got %v", err)
		}
	})

	t.Run("nonexistent poll returns 404", func(t *testing.T) {
		_, err := svc.ClosePoll(ctx, ownerID, "unknown_poll_id")
		if !errors.Is(err, ErrPollNotFound) {
			t.Errorf("expected ErrPollNotFound, got %v", err)
		}
	})

	t.Run("owner can close poll", func(t *testing.T) {
		resp, err := svc.ClosePoll(ctx, ownerID, poll.ID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if resp.Status != PollStatusClosed {
			t.Errorf("expected status %s, got %s", PollStatusClosed, resp.Status)
		}
	})

	t.Run("closing already closed poll is idempotent", func(t *testing.T) {
		resp, err := svc.ClosePoll(ctx, ownerID, poll.ID)
		if err != nil {
			t.Fatalf("expected no error on idempotent close, got %v", err)
		}
		if resp.Status != PollStatusClosed {
			t.Errorf("expected status %s, got %s", PollStatusClosed, resp.Status)
		}
	})
}

func TestService_DeletePoll(t *testing.T) {
	repo := newMockPollRepository()
	svc := NewService(repo)
	ctx := context.Background()

	ownerID := "user_del_owner"
	attackerID := "user_del_attacker"

	poll, err := svc.CreatePoll(ctx, ownerID, CreatePollRequest{
		Question: "Delete test question?",
		Options:  []string{"A", "B"},
	})
	if err != nil {
		t.Fatalf("failed to create poll: %v", err)
	}

	t.Run("non-owner cannot delete poll", func(t *testing.T) {
		err := svc.DeletePoll(ctx, attackerID, poll.ID)
		if !errors.Is(err, ErrForbidden) {
			t.Errorf("expected ErrForbidden, got %v", err)
		}
	})

	t.Run("nonexistent poll returns ErrPollNotFound", func(t *testing.T) {
		err := svc.DeletePoll(ctx, ownerID, "nonexistent_id")
		if !errors.Is(err, ErrPollNotFound) {
			t.Errorf("expected ErrPollNotFound, got %v", err)
		}
	})

	t.Run("owner can delete poll successfully", func(t *testing.T) {
		err := svc.DeletePoll(ctx, ownerID, poll.ID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Subsequent retrieval should return ErrPollNotFound (soft-deleted)
		_, err = svc.GetPublicPoll(ctx, poll.ID)
		if !errors.Is(err, ErrPollNotFound) {
			t.Errorf("expected ErrPollNotFound after deletion, got %v", err)
		}
	})
}
