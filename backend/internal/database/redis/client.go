package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client wraps the go-redis client.
type Client struct {
	client *redis.Client
}

// Connect parses the Redis URL, verifies the connection with a Ping, and returns the Client.
func Connect(ctx context.Context, redisURL string) (*Client, error) {
	if redisURL == "" {
		return nil, fmt.Errorf("redis URL cannot be empty")
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	opts.DialTimeout = 5 * time.Second
	opts.ReadTimeout = 3 * time.Second
	opts.WriteTimeout = 3 * time.Second

	rdb := redis.NewClient(opts)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := rdb.Ping(pingCtx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("failed to ping redis at %s: %w", opts.Addr, err)
	}

	return &Client{
		client: rdb,
	}, nil
}

// Ping checks if Redis is responsive.
func (c *Client) Ping(ctx context.Context) error {
	if c.client == nil {
		return fmt.Errorf("redis client is not initialized")
	}
	return c.client.Ping(ctx).Err()
}

// Close gracefully closes the Redis client connection pool.
func (c *Client) Close() error {
	if c.client == nil {
		return nil
	}
	return c.client.Close()
}

// Raw returns the underlying *redis.Client for executing commands and Pub/Sub.
func (c *Client) Raw() *redis.Client {
	return c.client
}
