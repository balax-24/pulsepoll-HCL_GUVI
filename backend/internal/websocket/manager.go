package websocket

import (
	"context"
	"log/slog"
	"sync"

	"github.com/redis/go-redis/v9"
)

// HubManager manages the collection of per-poll hubs with concurrency-safe
// lifecycle management and dynamic Redis Pub/Sub provisioning.
type HubManager struct {
	rdb  *redis.Client
	hubs map[string]*PollHub
	mu   sync.RWMutex
}

// NewHubManager constructs a central HubManager with the application's Redis client.
func NewHubManager(rdb *redis.Client) *HubManager {
	return &HubManager{
		rdb:  rdb,
		hubs: make(map[string]*PollHub),
	}
}

// GetOrCreateHub retrieves an existing active hub for the given poll ID,
// or instantiates and starts a new one with a verified Redis Pub/Sub subscription.
func (m *HubManager) GetOrCreateHub(ctx context.Context, pollID string) (*PollHub, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if active hub exists
	if hub, exists := m.hubs[pollID]; exists && !hub.IsClosed() {
		return hub, nil
	}

	// Instantiate new poll-specific hub and establish Redis Pub/Sub subscription
	hub, err := NewPollHub(ctx, pollID, m.rdb, m)
	if err != nil {
		return nil, err
	}

	m.hubs[pollID] = hub
	go hub.Run()

	slog.Info("Initialized new PollHub with active Redis Pub/Sub",
		slog.String("poll_id", pollID),
		slog.Int("total_active_hubs", len(m.hubs)),
	)

	return hub, nil
}

// RemoveHub unregisters an inactive hub from the registry.
// Only removes if the mapped hub matches the given instance, preventing race conditions with newer hubs.
func (m *HubManager) RemoveHub(pollID string, hub *PollHub) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if current, exists := m.hubs[pollID]; exists && current == hub {
		delete(m.hubs, pollID)
		slog.Info("Removed inactive PollHub from registry",
			slog.String("poll_id", pollID),
			slog.Int("remaining_active_hubs", len(m.hubs)),
		)
	}
}

// ActiveHubCount returns the number of currently active poll hubs.
func (m *HubManager) ActiveHubCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.hubs)
}

// Shutdown stops all active hubs and cleans up Redis subscriptions.
func (m *HubManager) Shutdown() {
	m.mu.Lock()
	activeHubs := make([]*PollHub, 0, len(m.hubs))
	for _, hub := range m.hubs {
		activeHubs = append(activeHubs, hub)
	}
	m.hubs = make(map[string]*PollHub)
	m.mu.Unlock()

	for _, hub := range activeHubs {
		hub.Stop()
	}

	slog.Info("All PollHubs shut down successfully")
}
