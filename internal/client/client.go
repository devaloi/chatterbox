package client

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/devaloi/chatterbox/internal/domain"
	"github.com/devaloi/chatterbox/internal/hub"
)

const (
	// writeWait is the time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// pongWait is the time allowed to read the next pong message from the peer.
	// If no pong is received within this window, the connection is considered dead.
	pongWait = 60 * time.Second

	// pingPeriod is the interval for sending pings to the peer. Must be less than
	// pongWait so that a missed pong is detected before the next ping is due.
	pingPeriod = (pongWait * 9) / 10

	// maxMessageSize is the maximum message size allowed from peer (bytes).
	maxMessageSize = 4096

	// sendBufferSize is the channel buffer for outgoing messages per client.
	sendBufferSize = 256
)

// Client is a WebSocket client connected to the hub.
type Client struct {
	hub      *hub.Hub
	conn     *websocket.Conn
	send     chan []byte
	done     chan struct{} // closed on disconnect to signal Send to stop
	username string
	rooms    map[string]bool
	mu       sync.RWMutex // protects rooms map
	closeOnce sync.Once
}

// New creates a new Client.
func New(h *hub.Hub, conn *websocket.Conn, username string) *Client {
	return &Client{
		hub:      h,
		conn:     conn,
		send:     make(chan []byte, sendBufferSize),
		done:     make(chan struct{}),
		username: username,
		rooms:    make(map[string]bool),
	}
}

// Username returns the client's username.
func (c *Client) Username() string {
	return c.username
}

// Send queues a message to be sent to the WebSocket client.
// Safe to call concurrently; returns silently if the client is disconnected.
func (c *Client) Send(data []byte) {
	select {
	case c.send <- data:
	case <-c.done:
		// Client disconnected, drop message.
	default:
		// Client send buffer full, drop message.
		log.Printf("client %s: send buffer full, dropping message", c.username)
	}
}

// ReadPump reads messages from the WebSocket connection and routes them to the hub.
// Each client runs one ReadPump goroutine. It unregisters from all rooms and
// closes the send channel on disconnect to unblock WritePump.
func (c *Client) ReadPump() {
	defer func() {
		// Signal Send() to stop accepting messages.
		c.closeOnce.Do(func() { close(c.done) })

		// Unregister from all rooms on disconnect.
		c.mu.RLock()
		rooms := make([]string, 0, len(c.rooms))
		for room := range c.rooms {
			rooms = append(rooms, room)
		}
		c.mu.RUnlock()

		for _, room := range rooms {
			c.hub.Unregister(c, room)
		}
		// Close send channel to unblock WritePump, preventing goroutine leak.
		close(c.send)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("client %s: read error: %v", c.username, err)
			}
			return
		}
		c.handleMessage(data)
	}
}

// WritePump writes messages from the send channel to the WebSocket connection.
// Each client runs one WritePump goroutine. It exits when the send channel is
// closed (by ReadPump on disconnect) or a write error occurs.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) handleMessage(data []byte) {
	var msg domain.Message
	if err := json.Unmarshal(data, &msg); err != nil {
		errMsg := domain.ErrorMessage{Type: domain.MsgError, Message: "invalid JSON"}
		d, e := domain.Encode(errMsg)
		if e != nil {
			log.Printf("client %s: encode error: %v", c.username, e)
			return
		}
		c.Send(d)
		return
	}

	switch msg.Type {
	case domain.MsgJoin:
		if msg.Room == "" {
			c.sendError("room name required")
			return
		}
		// Prevent joining the same room twice.
		c.mu.Lock()
		if c.rooms[msg.Room] {
			c.mu.Unlock()
			return
		}
		c.rooms[msg.Room] = true
		c.mu.Unlock()
		c.hub.Register(c, msg.Room)

	case domain.MsgLeave:
		if msg.Room == "" {
			c.sendError("room name required")
			return
		}
		c.mu.Lock()
		delete(c.rooms, msg.Room)
		c.mu.Unlock()
		c.hub.Unregister(c, msg.Room)

	case domain.MsgChat:
		if msg.Room == "" || msg.Text == "" {
			c.sendError("room and text required")
			return
		}
		c.mu.RLock()
		inRoom := c.rooms[msg.Room]
		c.mu.RUnlock()
		if !inRoom {
			c.sendError("not in room")
			return
		}
		msg.User = c.username
		msg.Timestamp = time.Now().UTC()
		c.hub.RouteMessage(msg, c)

	default:
		c.sendError("unknown message type: " + msg.Type)
	}
}

func (c *Client) sendError(message string) {
	errMsg := domain.ErrorMessage{Type: domain.MsgError, Message: message}
	data, err := domain.Encode(errMsg)
	if err != nil {
		log.Printf("client %s: encode error: %v", c.username, err)
		return
	}
	c.Send(data)
}
