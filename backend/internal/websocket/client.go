package websocket

import (
	"log/slog"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// writeWait is the maximum duration allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// pongWait is the maximum duration to wait for the next pong message from the peer.
	pongWait = 60 * time.Second

	// pingPeriod is the interval to send ping messages to the peer (must be less than pongWait).
	pingPeriod = (pongWait * 9) / 10

	// maxMessageSize is the maximum message size allowed from peer (in bytes).
	maxMessageSize = 512

	// sendBufferSize is the capacity of the client outbound message buffer.
	sendBufferSize = 256
)

// Client represents a single active WebSocket connection associated with a specific poll.
type Client struct {
	hub    *PollHub
	conn   *websocket.Conn
	send   chan []byte
	pollID string
}

// NewClient constructs a Client instance.
func NewClient(hub *PollHub, conn *websocket.Conn, pollID string) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, sendBufferSize),
		pollID: pollID,
	}
}

// readPump pumps messages from the WebSocket connection to the hub.
// The backend uses a server-push model; incoming messages are discarded or checked for close frames.
// When read fails or peer disconnects, it cleanly unregisters the client from the hub.
func (c *Client) readPump() {
	defer func() {
		c.hub.Unregister(c)
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				slog.Debug("WebSocket peer disconnected", slog.String("poll_id", c.pollID), slog.String("error", err.Error()))
			}
			break
		}
	}
}

// writePump pumps messages from the hub's send channel to the WebSocket connection.
// It manages heartbeat pings and gracefully handles channel closures.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
