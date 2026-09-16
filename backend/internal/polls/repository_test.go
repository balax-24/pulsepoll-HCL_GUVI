package polls

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"pulsepoll/backend/internal/database/mongodb"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestMongoPollRepository_Integration(t *testing.T) {
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	testDBName := "pulsepoll_test_repo"
	client, err := mongodb.Connect(ctx, mongoURI, testDBName)
	if err != nil {
		t.Skipf("skipping MongoDB integration tests (database unavailable): %v", err)
	}
	defer func() {
		_ = client.Database().Drop(context.Background())
		_ = client.Close(context.Background())
	}()

	repo := NewMongoPollRepository(client.Database())

	// 1. Ensure Indexes
	if err := repo.EnsureIndexes(ctx); err != nil {
		t.Fatalf("failed to ensure indexes: %v", err)
	}

	creatorID := "user_repo_test_1"
	now := time.Now().UTC().Truncate(time.Millisecond)

	poll := &Poll{
		ID:        bson.NewObjectID(),
		CreatorID: creatorID,
		Question:  "What is your primary backend language?",
		Options: []PollOption{
			{ID: bson.NewObjectID().Hex(), Text: "Go"},
			{ID: bson.NewObjectID().Hex(), Text: "Rust"},
			{ID: bson.NewObjectID().Hex(), Text: "TypeScript"},
		},
		Status:    PollStatusActive,
		IsDeleted: false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// 2. Create Poll
	if err := repo.Create(ctx, poll); err != nil {
		t.Fatalf("failed to create poll: %v", err)
	}

	// 3. FindByID
	found, err := repo.FindByID(ctx, poll.ID.Hex())
	if err != nil {
		t.Fatalf("failed to find poll by ID: %v", err)
	}
	if found.Question != poll.Question {
		t.Errorf("expected question %s, got %s", poll.Question, found.Question)
	}
	if len(found.Options) != 3 {
		t.Errorf("expected 3 options, got %d", len(found.Options))
	}
	if found.Status != PollStatusActive {
		t.Errorf("expected status active, got %s", found.Status)
	}

	// 4. FindByCreatorID with pagination
	polls, total, err := repo.FindByCreatorID(ctx, creatorID, 10, 0)
	if err != nil {
		t.Fatalf("failed to find creator polls: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
	if len(polls) != 1 || polls[0].ID != poll.ID {
		t.Errorf("expected matching poll in list")
	}

	// 5. Update Poll
	poll.Question = "Updated Question Text?"
	poll.Options[0].Text = "Golang"
	poll.UpdatedAt = time.Now().UTC()
	if err := repo.Update(ctx, poll); err != nil {
		t.Fatalf("failed to update poll: %v", err)
	}

	updated, err := repo.FindByID(ctx, poll.ID.Hex())
	if err != nil {
		t.Fatalf("failed to find updated poll: %v", err)
	}
	if updated.Question != "Updated Question Text?" {
		t.Errorf("expected updated question, got %s", updated.Question)
	}
	if updated.Options[0].Text != "Golang" {
		t.Errorf("expected updated option text 'Golang', got %s", updated.Options[0].Text)
	}

	// 6. UpdateStatus
	statusUpdated, err := repo.UpdateStatus(ctx, poll.ID.Hex(), PollStatusClosed)
	if err != nil {
		t.Fatalf("failed to update status: %v", err)
	}
	if statusUpdated.Status != PollStatusClosed {
		t.Errorf("expected status closed, got %s", statusUpdated.Status)
	}

	// 7. SoftDelete
	if err := repo.SoftDelete(ctx, poll.ID.Hex()); err != nil {
		t.Fatalf("failed to soft delete poll: %v", err)
	}

	// Verify FindByID returns ErrPollNotFound for soft-deleted poll
	_, err = repo.FindByID(ctx, poll.ID.Hex())
	if !errors.Is(err, ErrPollNotFound) {
		t.Errorf("expected ErrPollNotFound after soft delete, got %v", err)
	}

	// Verify FindByCreatorID does not return soft-deleted poll
	pollsAfterDelete, totalAfterDelete, err := repo.FindByCreatorID(ctx, creatorID, 10, 0)
	if err != nil {
		t.Fatalf("failed to find creator polls after delete: %v", err)
	}
	if totalAfterDelete != 0 || len(pollsAfterDelete) != 0 {
		t.Errorf("expected 0 creator polls after soft delete, got total %d", totalAfterDelete)
	}
}
