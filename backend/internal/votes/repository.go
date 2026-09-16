package votes

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	// ErrDuplicateVote indicates that the voter has already cast a vote on this poll.
	ErrDuplicateVote = errors.New("duplicate vote: voter has already cast a vote on this poll")
)

// VoteRepository defines persistent storage operations for Vote audit records.
type VoteRepository interface {
	Create(ctx context.Context, vote *Vote) error
	HasVoted(ctx context.Context, pollID, voterID string) (bool, error)
	GetVoteCountsByPoll(ctx context.Context, pollID string) (map[string]int64, error)
	CountByPollID(ctx context.Context, pollID string) (int64, error)
	EnsureIndexes(ctx context.Context) error
}

// MongoVoteRepository implements VoteRepository backed by MongoDB.
type MongoVoteRepository struct {
	collection *mongo.Collection
}

// NewMongoVoteRepository constructs a repository instance pointing to the 'votes' collection.
func NewMongoVoteRepository(db *mongo.Database) *MongoVoteRepository {
	return &MongoVoteRepository{
		collection: db.Collection("votes"),
	}
}

// EnsureIndexes creates database-level constraints and performance indexes:
// 1. (poll_id, voter_id) [UNIQUE]: Guarantees concurrency-safe single vote per participant.
// 2. (poll_id): Facilitates rapid lookups and aggregation per poll.
// 3. (poll_id, option_id): Accelerates per-option query aggregations.
func (r *MongoVoteRepository) EnsureIndexes(ctx context.Context) error {
	models := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "poll_id", Value: 1},
				{Key: "voter_id", Value: 1},
			},
			Options: options.Index().
				SetUnique(true).
				SetName("uniq_poll_voter"),
		},
		{
			Keys: bson.D{
				{Key: "poll_id", Value: 1},
			},
			Options: options.Index().SetName("idx_vote_poll_id"),
		},
		{
			Keys: bson.D{
				{Key: "poll_id", Value: 1},
				{Key: "option_id", Value: 1},
			},
			Options: options.Index().SetName("idx_vote_poll_option"),
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, models)
	if err != nil {
		return fmt.Errorf("failed to create vote collection indexes: %w", err)
	}
	return nil
}

// Create inserts a new vote audit document into MongoDB.
// If the unique (poll_id, voter_id) constraint is violated, returns ErrDuplicateVote.
func (r *MongoVoteRepository) Create(ctx context.Context, vote *Vote) error {
	if vote.ID.IsZero() {
		vote.ID = bson.NewObjectID()
	}
	if vote.CreatedAt.IsZero() {
		vote.CreatedAt = time.Now().UTC()
	}

	_, err := r.collection.InsertOne(ctx, vote)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrDuplicateVote
		}
		return fmt.Errorf("failed to insert vote: %w", err)
	}

	return nil
}

// HasVoted queries whether a vote record already exists for the given (poll_id, voter_id) pair.
func (r *MongoVoteRepository) HasVoted(ctx context.Context, pollID, voterID string) (bool, error) {
	filter := bson.M{
		"poll_id":  pollID,
		"voter_id": voterID,
	}

	count, err := r.collection.CountDocuments(ctx, filter, options.Count().SetLimit(1))
	if err != nil {
		return false, fmt.Errorf("failed to check existing vote: %w", err)
	}

	return count > 0, nil
}

// GetVoteCountsByPoll aggregates total votes per option directly from durable MongoDB storage.
// This is used for Redis initialization and crash reconciliation.
func (r *MongoVoteRepository) GetVoteCountsByPoll(ctx context.Context, pollID string) (map[string]int64, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"poll_id": pollID}}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$option_id",
			"count": bson.M{"$sum": 1},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate vote counts: %w", err)
	}
	defer cursor.Close(ctx)

	type aggResult struct {
		OptionID string `bson:"_id"`
		Count    int64  `bson:"count"`
	}

	counts := make(map[string]int64)
	for cursor.Next(ctx) {
		var item aggResult
		if err := cursor.Decode(&item); err != nil {
			return nil, fmt.Errorf("failed to decode aggregation result: %w", err)
		}
		counts[item.OptionID] = item.Count
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error in vote aggregation: %w", err)
	}

	return counts, nil
}

// CountByPollID counts total persisted votes for a given poll.
func (r *MongoVoteRepository) CountByPollID(ctx context.Context, pollID string) (int64, error) {
	filter := bson.M{"poll_id": pollID}
	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count votes for poll: %w", err)
	}
	return count, nil
}
