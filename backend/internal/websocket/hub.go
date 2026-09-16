package websocket

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// HubRemover specifies the interface to remove an inactive hub from the central manager.
type HubRemover interface {
	RemoveHub(pollID string, hub *PollHub)
}

// PollHub maintains the set of active connected clients for a single poll
// and subscribes dynamically to Redis Pub/Sub channel poll:{pollID}:updates.
type PollHub struct {
	pollID     string
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	done       chan struct{}
	stopOnce   sync.Once

	rdb     *redis.Client
	pubsub  *redis.PubSub
	remover HubRemover

	mu sync.Mutex
}

// NewPollHub creates and initializes a PollHub for a specific poll,
// establishing the Redis Pub/Sub subscription before returning to eliminate race conditions.
func NewPollHub(ctx context.Context, pollID string, rdb *redis.Client, remover HubRemover) (*PollHub, error) {
	channel := fmt.Sprintf("poll:%s:updates", pollID)

	// Establish Redis Pub/Sub subscription dynamically
	pubsub := rdb.Subscribe(ctx, channel)

	// Block until subscription confirmation is received from Redis
	subCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if _, err := pubsub.Receive(subCtx); err != nil {
		_ = pubsub.Close()
		return nil, fmt.Errorf("failed to subscribe to Redis channel %s: %w", channel, err)
	}

	slog.Info("Redis Pub/Sub subscription started for poll",
		slog.String("poll_id", pollID),
		slog.String("channel", channel),
	)

	hub := &PollHub{
		pollID:     pollID,
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 256),
		done:       make(chan struct{}),
		rdb:        rdb,
		pubsub:     pubsub,
		remover:    remover,
	}

	return hub, nil
}

// IsClosed checks whether the hub has terminated.
func (h *PollHub) IsClosed() bool {
	select {
	case <-h.done:
		return true
	default:
		return false
	}
}

// Register registers a new client to the hub.
func (h *PollHub) Register(c *Client) {
	select {
	case <-h.done:
		// Hub is closing; discard and close client send
		close(c.send)
	case h.register <- c:
	}
}

// Unregister schedules a client to be unregistered from the hub.
func (h *PollHub) Unregister(c *Client) {
	select {
	case <-h.done:
	case h.unregister <- c:
	}
}

// Broadcast schedules a raw byte message for broadcast to all connected clients.
func (h *PollHub) Broadcast(msg []byte) {
	select {
	case <-h.done:
	case h.broadcast <- msg:
	}
}

// Stop terminates the hub, unregistering all clients and closing the Redis subscription.
func (h *PollHub) Stop() {
	h.stopOnce.Do(func() {
		close(h.done)
		if h.pubsub != nil {
			_ = h.pubsub.Close()
			slog.Info("Redis Pub/Sub subscription stopped for poll",
				slog.String("poll_id", h.pollID),
			)
		}
		if h.remover != nil {
			h.remover.RemoveHub(h.pollID, h)
		}
	})
}

// listenRedis reads incoming events from the Redis Pub/Sub channel and forwards them to the broadcast queue.
func (h *PollHub) listenRedis() {
	ch := h.pubsub.Channel()
	for {
		select {
		case <-h.done:
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			slog.Info("Received Redis Pub/Sub event on channel",
				slog.String("channel", msg.Channel),
				slog.String("poll_id", h.pollID),
			)
			h.Broadcast([]byte(msg.Payload))
		}
	}
}

// Run executes the hub event loop. It manages client lifecycle and terminates
// when the last client disconnects or when Stop is called.
func (h *PollHub) Run() {
	// Start background listener for Redis Pub/Sub channel
	go h.listenRedis()

	defer func() {
		h.Stop()
		// Clean up all remaining client send channels safely
		for client := range h.clients {
			close(client.send)
		}
		h.clients = make(map[*Client]bool)
	}()

	for {
		select {
		case <-h.done:
			return

		case client := <-h.register:
			h.clients[client] = true
			slog.Info("WebSocket client connected",
				slog.String("poll_id", h.pollID),
				slog.Int("total_clients", len(h.clients)),
			)

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				slog.Info("WebSocket client disconnected",
					slog.String("poll_id", h.pollID),
					slog.Int("remaining_clients", len(h.clients)),
				)
			}
			// When the last client disconnects, stop subscription and hub
			if len(h.clients) == 0 {
				slog.Info("Last WebSocket client disconnected; shutting down hub",
					slog.String("poll_id", h.pollID),
				)
				return
			}

		case msg := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- msg:
				default:
					// Drop slow or blocked client whose buffer is full without blocking other clients
					slog.Warn("WebSocket client buffer full; dropping slow client to avoid stalling",
						slog.String("poll_id", h.pollID),
					)
					delete(h.clients, client)
					close(client.send)
				}
			}
		}
	}
}
