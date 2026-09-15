package auth

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
	// ErrUserNotFound is returned when a user does not exist in the database.
	ErrUserNotFound = errors.New("user not found")

	// ErrDuplicateEmail is returned when an email is already registered.
	ErrDuplicateEmail = errors.New("email already registered")
)

// UserRepository defines persistence operations for User documents.
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id string) (*User, error)
	EnsureIndexes(ctx context.Context) error
}

// MongoUserRepository implements UserRepository backed by MongoDB.
type MongoUserRepository struct {
	collection *mongo.Collection
}

// NewMongoUserRepository creates a new repository instance pointing to the 'users' collection.
func NewMongoUserRepository(db *mongo.Database) *MongoUserRepository {
	return &MongoUserRepository{
		collection: db.Collection("users"),
	}
}

// EnsureIndexes creates a unique index on the 'email' field to guarantee uniqueness at the database level.
func (r *MongoUserRepository) EnsureIndexes(ctx context.Context) error {
	indexModel := mongo.IndexModel{
		Keys: bson.D{{Key: "email", Value: 1}},
		Options: options.Index().
			SetUnique(true).
			SetName("uniq_user_email"),
	}

	_, err := r.collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return fmt.Errorf("failed to create unique email index: %w", err)
	}
	return nil
}

// Create inserts a new user record. If the email is already in use, returns ErrDuplicateEmail.
func (r *MongoUserRepository) Create(ctx context.Context, user *User) error {
	if user.ID.IsZero() {
		user.ID = bson.NewObjectID()
	}
	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now

	_, err := r.collection.InsertOne(ctx, user)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrDuplicateEmail
		}
		return fmt.Errorf("failed to insert user: %w", err)
	}
	return nil
}

// FindByEmail searches for a user by normalized email.
func (r *MongoUserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	filter := bson.M{"email": email}

	var user User
	err := r.collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}
	return &user, nil
}

// FindByID searches for a user by their ObjectID string.
func (r *MongoUserRepository) FindByID(ctx context.Context, id string) (*User, error) {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrUserNotFound
	}

	filter := bson.M{"_id": objID}

	var user User
	err = r.collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user by ID: %w", err)
	}
	return &user, nil
}
