package votes

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"pulsepoll/backend/internal/database/mongodb"
	"pulsepoll/backend/internal/database/redis"
	"pulsepoll/backend/internal/polls"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestVoting_Concurrency(t *testing.T) {
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6380"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	testDBName := "pulsepoll_test_concurrency"
	mongoClient, err := mongodb.Connect(ctx, mongoURI, testDBName)
	if err != nil {
		t.Skipf("skipping concurrency test (MongoDB unavailable): %v", err)
	}
	defer func() {
		_ = mongoClient.Database().Drop(context.Background())
		_ = mongoClient.Close(context.Background())
	}()

	redisClient, err := redis.Connect(ctx, redisURL)
	if err != nil {
		t.Skipf("skipping concurrency test (Redis unavailable): %v", err)
	}
	defer func() {
		_ = redisClient.Close()
	}()

	// Initialize repositories
	pollRepo := polls.NewMongoPollRepository(mongoClient.Database())
	if err := pollRepo.EnsureIndexes(ctx); err != nil {
		t.Fatalf("failed to ensure poll indexes: %v", err)
	}

	voteRepo := NewMongoVoteRepository(mongoClient.Database())
	if err := voteRepo.EnsureIndexes(ctx); err != nil {
		t.Fatalf("failed to ensure vote indexes: %v", err)
	}

	redisVoteRepo := NewRedisVoteRepository(redisClient.Raw())

	// Initialize Service
	svc := NewService(pollRepo, voteRepo, redisVoteRepo)

	// Create test poll
	opt1ID := bson.NewObjectID().Hex()
	opt2ID := bson.NewObjectID().Hex()
	opt3ID := bson.NewObjectID().Hex()

	testPoll := &polls.Poll{
		ID:        bson.NewObjectID(),
		CreatorID: "creator_concurrency_test",
		Question:  "High-Concurrency Performance Test Poll",
		Options: []polls.PollOption{
			{ID: opt1ID, Text: "Option 1 (Target 25)"},
			{ID: opt2ID, Text: "Option 2 (Target 15)"},
			{ID: opt3ID, Text: "Option 3 (Target 10)"},
		},
		Status:    polls.PollStatusActive,
		IsDeleted: false,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := pollRepo.Create(ctx, testPoll); err != nil {
		t.Fatalf("failed to create test poll: %v", err)
	}
	pollID := testPoll.ID.Hex()

	// Ensure Redis counters are cleaned up before starting
	_ = redisVoteRepo.ResetCounters(ctx, pollID)

	// Total voters: 50
	// 25 vote for opt1ID
	// 15 vote for opt2ID
	// 10 vote for opt3ID
	// In addition, 10 duplicate vote attempts will be launched concurrently by voters 1..10
	type voteTask struct {
		voterID  string
		optionID string
	}

	var tasks []voteTask
	for i := 1; i <= 25; i++ {
		tasks = append(tasks, voteTask{voterID: fmt.Sprintf("voter_%03d", i), optionID: opt1ID})
	}
	for i := 26; i <= 40; i++ {
		tasks = append(tasks, voteTask{voterID: fmt.Sprintf("voter_%03d", i), optionID: opt2ID})
	}
	for i := 41; i <= 50; i++ {
		tasks = append(tasks, voteTask{voterID: fmt.Sprintf("voter_%03d", i), optionID: opt3ID})
	}

	// Duplicate tasks: voters 1..10 try to vote a second time concurrently on opt1ID
	for i := 1; i <= 10; i++ {
		tasks = append(tasks, voteTask{voterID: fmt.Sprintf("voter_%03d", i), optionID: opt1ID})
	}

	var wg sync.WaitGroup
	var acceptedVotes int64
	var duplicateRejections int64
	var unexpectedErrors int64

	// Concurrency barrier: start all goroutines simultaneously
	startBarrier := make(chan struct{})

	for _, task := range tasks {
		wg.Add(1)
		go func(t voteTask) {
			defer wg.Done()
			<-startBarrier // wait for synchronized trigger

			_, err := svc.CastVote(context.Background(), pollID, t.optionID, t.voterID)
			if err == nil {
				atomic.AddInt64(&acceptedVotes, 1)
			} else if errors.Is(err, ErrDuplicateVote) {
				atomic.AddInt64(&duplicateRejections, 1)
			} else {
				atomic.AddInt64(&unexpectedErrors, 1)
			}
		}(task)
	}

	// Fire all goroutines
	close(startBarrier)
	wg.Wait()

	// Assertions
	if acceptedVotes != 50 {
		t.Fatalf("expected exactly 50 accepted votes, got %d", acceptedVotes)
	}
	if duplicateRejections != 10 {
		t.Fatalf("expected exactly 10 duplicate rejections, got %d", duplicateRejections)
	}
	if unexpectedErrors != 0 {
		t.Fatalf("expected 0 unexpected errors, got %d", unexpectedErrors)
	}

	// 1. Verify MongoDB votes collection has exactly 50 documents
	mongoCount, err := voteRepo.CountByPollID(ctx, pollID)
	if err != nil {
		t.Fatalf("failed to count mongo votes: %v", err)
	}
	if mongoCount != 50 {
		t.Errorf("expected 50 votes in MongoDB, got %d", mongoCount)
	}

	// 2. Verify Redis live counts
	results, err := svc.GetResults(ctx, pollID)
	if err != nil {
		t.Fatalf("failed to retrieve live results: %v", err)
	}
	if results.TotalVotes != 50 {
		t.Errorf("expected Redis total_votes 50, got %d", results.TotalVotes)
	}

	optionCounts := make(map[string]int64)
	for _, opt := range results.Results {
		optionCounts[opt.OptionID] = opt.Votes
	}

	if optionCounts[opt1ID] != 25 {
		t.Errorf("expected Option 1 count = 25, got %d", optionCounts[opt1ID])
	}
	if optionCounts[opt2ID] != 15 {
		t.Errorf("expected Option 2 count = 15, got %d", optionCounts[opt2ID])
	}
	if optionCounts[opt3ID] != 10 {
		t.Errorf("expected Option 3 count = 10, got %d", optionCounts[opt3ID])
	}

	// 3. Verify MongoDB poll document does NOT contain any vote counters
	var rawPoll bson.M
	err = mongoClient.Database().Collection("polls").FindOne(ctx, bson.M{"_id": testPoll.ID}).Decode(&rawPoll)
	if err != nil {
		t.Fatalf("failed to query raw poll: %v", err)
	}

	for _, forbiddenField := range []string{"vote_count", "votes", "total_votes", "counts"} {
		if _, exists := rawPoll[forbiddenField]; exists {
			t.Errorf("MongoDB poll document incorrectly contains vote counter field: '%s'", forbiddenField)
		}
	}

	// Clean up Redis test key
	_ = redisVoteRepo.ResetCounters(ctx, pollID)
}
