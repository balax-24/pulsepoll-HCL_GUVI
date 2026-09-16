package polls

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// PollRepository specifies persistent storage operations for Poll entities.
type PollRepository interface {
	Create(ctx context.Context, poll *Poll) error
	FindByID(ctx context.Context, id string) (*Poll, error)
	FindByCreatorID(ctx context.Context, creatorID string, limit, offset int64) ([]*Poll, int64, error)
	Update(ctx context.Context, poll *Poll) error
	UpdateStatus(ctx context.Context, id string, status PollStatus) (*Poll, error)
	SoftDelete(ctx context.Context, id string) error
	EnsureIndexes(ctx context.Context) error
}

// MongoPollRepository implements PollRepository using MongoDB.
type MongoPollRepository struct {
	collection *mongo.Collection
}

// NewMongoPollRepository creates a new MongoPollRepository initialized on the 'polls' collection.
func NewMongoPollRepository(db *mongo.Database) *MongoPollRepository {
	return &MongoPollRepository{
		collection: db.Collection("polls"),
	}
}

// EnsureIndexes creates performance indexes on polls:
// 1. (creator_id, created_at DESC): Composite index optimizing creator poll retrieval with temporal sorting.
// 2. (created_at DESC): Single-field index for global timeline queries.
// 3. (status): Filtering polls by active/closed lifecycle status.
// 4. (is_deleted): Index to accelerate filtering out soft-deleted records.
func (r *MongoPollRepository) EnsureIndexes(ctx context.Context) error {
	models := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "creator_id", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().SetName("idx_poll_creator_created"),
		},
		{
			Keys: bson.D{
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().SetName("idx_poll_created_at"),
		},
		{
			Keys: bson.D{
				{Key: "status", Value: 1},
			},
			Options: options.Index().SetName("idx_poll_status"),
		},
		{
			Keys: bson.D{
				{Key: "is_deleted", Value: 1},
			},
			Options: options.Index().SetName("idx_poll_is_deleted"),
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, models)
	if err != nil {
		return fmt.Errorf("failed to create poll collection indexes: %w", err)
	}
	return nil
}

// Create persists a new Poll document into MongoDB.
func (r *MongoPollRepository) Create(ctx context.Context, poll *Poll) error {
	if poll.ID.IsZero() {
		poll.ID = bson.NewObjectID()
	}
	now := time.Now().UTC()
	if poll.CreatedAt.IsZero() {
		poll.CreatedAt = now
	}
	if poll.UpdatedAt.IsZero() {
		poll.UpdatedAt = now
	}

	_, err := r.collection.InsertOne(ctx, poll)
	if err != nil {
		return fmt.Errorf("failed to insert poll document: %w", err)
	}
	return nil
}

// FindByID retrieves an active (non-deleted) poll document by its hexadecimal string ID.
func (r *MongoPollRepository) FindByID(ctx context.Context, id string) (*Poll, error) {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrPollNotFound
	}

	filter := bson.M{
		"_id":        objID,
		"is_deleted": false,
	}

	var poll Poll
	err = r.collection.FindOne(ctx, filter).Decode(&poll)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrPollNotFound
		}
		return nil, fmt.Errorf("failed to query poll by id: %w", err)
	}

	return &poll, nil
}

// FindByCreatorID retrieves paginated active polls created by a specific user, sorted newest first.
func (r *MongoPollRepository) FindByCreatorID(ctx context.Context, creatorID string, limit, offset int64) ([]*Poll, int64, error) {
	filter := bson.M{
		"creator_id": creatorID,
		"is_deleted": false,
	}

	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count creator polls: %w", err)
	}

	findOpts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(offset).
		SetLimit(limit)

	cursor, err := r.collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list creator polls: %w", err)
	}
	defer cursor.Close(ctx)

	var polls []*Poll
	if err := cursor.All(ctx, &polls); err != nil {
		return nil, 0, fmt.Errorf("failed to decode creator polls: %w", err)
	}
	if polls == nil {
		polls = []*Poll{}
	}

	return polls, total, nil
}

// Update writes modifications to question, options, and updatedAt fields of an active poll.
func (r *MongoPollRepository) Update(ctx context.Context, poll *Poll) error {
	filter := bson.M{
		"_id":        poll.ID,
		"is_deleted": false,
	}

	update := bson.M{
		"$set": bson.M{
			"question":   poll.Question,
			"options":    poll.Options,
			"updated_at": poll.UpdatedAt,
		},
	}

	res, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update poll document: %w", err)
	}
	if res.MatchedCount == 0 {
		return ErrPollNotFound
	}
	return nil
}

// UpdateStatus changes the status of an active poll (e.g. active -> closed).
func (r *MongoPollRepository) UpdateStatus(ctx context.Context, id string, status PollStatus) (*Poll, error) {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrPollNotFound
	}

	filter := bson.M{
		"_id":        objID,
		"is_deleted": false,
	}

	now := time.Now().UTC()
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": now,
		},
	}

	res, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, fmt.Errorf("failed to update poll status: %w", err)
	}
	if res.MatchedCount == 0 {
		return nil, ErrPollNotFound
	}

	return r.FindByID(ctx, id)
}

// SoftDelete marks a poll as deleted without removing its document, preserving future vote audit logs.
func (r *MongoPollRepository) SoftDelete(ctx context.Context, id string) error {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return ErrPollNotFound
	}

	filter := bson.M{
		"_id":        objID,
		"is_deleted": false,
	}

	now := time.Now().UTC()
	update := bson.M{
		"$set": bson.M{
			"is_deleted": true,
			"deleted_at": now,
			"updated_at": now,
		},
	}

	res, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to soft delete poll: %w", err)
	}
	if res.MatchedCount == 0 {
		return ErrPollNotFound
	}

	return nil
}
