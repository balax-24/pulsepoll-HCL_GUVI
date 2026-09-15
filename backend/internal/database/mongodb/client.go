package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Client encapsulates the MongoDB client and active database connection.
type Client struct {
	client   *mongo.Client
	database *mongo.Database
}

// Connect establishes a connection to MongoDB, validates connectivity with a Ping,
// and returns an initialized Client.
func Connect(ctx context.Context, uri, dbName string) (*Client, error) {
	if uri == "" {
		return nil, fmt.Errorf("mongodb URI cannot be empty")
	}
	if dbName == "" {
		return nil, fmt.Errorf("mongodb database name cannot be empty")
	}

	opts := options.Client().
		ApplyURI(uri).
		SetTimeout(10 * time.Second)

	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create mongodb client: %w", err)
	}

	// Verify connection with Ping using the supplied context (or a timeout context)
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("failed to ping mongodb at %s: %w", sanitizeURI(uri), err)
	}

	db := client.Database(dbName)

	return &Client{
		client:   client,
		database: db,
	}, nil
}

// Ping checks if MongoDB is responsive.
func (c *Client) Ping(ctx context.Context) error {
	if c.client == nil {
		return fmt.Errorf("mongodb client is not initialized")
	}
	return c.client.Ping(ctx, nil)
}

// Close disconnects the MongoDB client cleanly.
func (c *Client) Close(ctx context.Context) error {
	if c.client == nil {
		return nil
	}
	return c.client.Disconnect(ctx)
}

// Database returns the active *mongo.Database handle.
func (c *Client) Database() *mongo.Database {
	return c.database
}

// Collection returns a handle to a specific MongoDB collection.
func (c *Client) Collection(name string) *mongo.Collection {
	return c.database.Collection(name)
}

// sanitizeURI strips passwords from MongoDB connection strings for safe logging.
func sanitizeURI(uri string) string {
	// Simple redaction to avoid leaking credentials in logs
	parts := options.Client().ApplyURI(uri)
	if parts.Auth != nil {
		return fmt.Sprintf("mongodb://%s:***@...", parts.Auth.Username)
	}
	return uri
}
