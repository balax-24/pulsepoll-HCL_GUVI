package votes

import (
	"context"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
)

// VoteCounterStore specifies operations for managing live atomic vote counts in Redis.
type VoteCounterStore interface {
	IncrementVote(ctx context.Context, pollID, optionID string) (int64, error)
	GetOptionCounts(ctx context.Context, pollID string, optionIDs []string) (map[string]int64, int64, error)
	InitializeCounters(ctx context.Context, pollID string, optionIDs []string, initialCounts map[string]int64) error
	HasCounters(ctx context.Context, pollID string) (bool, error)
	ResetCounters(ctx context.Context, pollID string) error
}

// RedisVoteRepository implements VoteCounterStore backed by Redis Hash structures.
type RedisVoteRepository struct {
	client *redis.Client
}

// NewRedisVoteRepository constructs a Redis-backed vote counter repository.
func NewRedisVoteRepository(client *redis.Client) *RedisVoteRepository {
	return &RedisVoteRepository{
		client: client,
	}
}

// pollVotesKey constructs the Redis Hash key for a specific poll's live vote tally.
// Format: poll:{pollID}:votes
func pollVotesKey(pollID string) string {
	return fmt.Sprintf("poll:%s:votes", pollID)
}

// IncrementVote performs an atomic HINCRBY on the poll's vote hash.
func (r *RedisVoteRepository) IncrementVote(ctx context.Context, pollID, optionID string) (int64, error) {
	key := pollVotesKey(pollID)
	newVal, err := r.client.HIncrBy(ctx, key, optionID, 1).Result()
	if err != nil {
		return 0, fmt.Errorf("redis HINCRBY failed for poll %s, option %s: %w", pollID, optionID, err)
	}
	return newVal, nil
}

// GetOptionCounts retrieves all option counts from Redis for a poll using HGETALL.
// Options with zero votes are included with 0.
func (r *RedisVoteRepository) GetOptionCounts(ctx context.Context, pollID string, optionIDs []string) (map[string]int64, int64, error) {
	key := pollVotesKey(pollID)
	rawMap, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, 0, fmt.Errorf("redis HGETALL failed for poll %s: %w", pollID, err)
	}

	counts := make(map[string]int64, len(optionIDs))
	var totalVotes int64 = 0

	for _, optID := range optionIDs {
		valStr, exists := rawMap[optID]
		if !exists {
			counts[optID] = 0
			continue
		}
		c, parseErr := strconv.ParseInt(valStr, 10, 64)
		if parseErr != nil {
			c = 0
		}
		counts[optID] = c
		totalVotes += c
	}

	return counts, totalVotes, nil
}

// InitializeCounters idempotently populates a poll's vote hash in Redis if not already present.
// It sets all option IDs to their initial durable count (or 0) using HSETNX to avoid race-condition overwrites.
func (r *RedisVoteRepository) InitializeCounters(ctx context.Context, pollID string, optionIDs []string, initialCounts map[string]int64) error {
	key := pollVotesKey(pollID)

	pipe := r.client.Pipeline()
	for _, optID := range optionIDs {
		initVal := int64(0)
		if initialCounts != nil {
			if v, ok := initialCounts[optID]; ok {
				initVal = v
			}
		}
		pipe.HSetNX(ctx, key, optID, initVal)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize redis vote counters for poll %s: %w", pollID, err)
	}

	return nil
}

// HasCounters checks whether a Redis vote hash exists for the given poll ID.
func (r *RedisVoteRepository) HasCounters(ctx context.Context, pollID string) (bool, error) {
	key := pollVotesKey(pollID)
	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("redis EXISTS check failed for poll %s: %w", pollID, err)
	}
	return exists > 0, nil
}

// ResetCounters deletes the Redis vote hash for a given poll (useful in testing or poll purge).
func (r *RedisVoteRepository) ResetCounters(ctx context.Context, pollID string) error {
	key := pollVotesKey(pollID)
	return r.client.Del(ctx, key).Err()
}
