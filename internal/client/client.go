package client

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"

	"github.com/devaloi/chatterbox/internal/domain"
	"github.com/devaloi/chatterbox/internal/hub"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 4096
)

// Client is a WebSocket client connected to the hub.
type Client struct {
	hub      *hub.Hub
	conn     *websocket.Conn
	send     chan []byte
	username string
	rooms    map[string]bool
}

// New creates a new Client.
func New(h *hub.Hub, conn *websocket.Conn, username string) *Client {
	return &Client{
		hub:      h,
		conn:     conn,
		send:     make(chan []byte, 256),
		username: username,
		rooms:    make(map[string]bool),
	}
}

// Username returns the client's username.
func (c *Client) Username() string {
	return c.username
}

// Send queues a message to be sent to the WebSocket client.
func (c *Client) Send(data []byte) {
	select {
	case c.send <- data:
	default:
		// Client send buffer full, drop message.
		log.Printf("client %s: send buffer full, dropping message", c.username)
	}
}

// ReadPump reads messages from the WebSocket connection and routes them to the hub.
func (c *Client) ReadPump() {
	defer func() {
		// Unregister from all rooms on disconnect.
		for room := range c.rooms {
			c.hub.Unregister(c, room)
		}
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
		if d, e := domain.Encode(errMsg); e == nil {
			c.Send(d)
		}
		return
	}

	switch msg.Type {
	case domain.MsgJoin:
		if msg.Room == "" {
			c.sendError("room name required")
			return
		}
		c.rooms[msg.Room] = true
		c.hub.Register(c, msg.Room)

	case domain.MsgLeave:
		if msg.Room == "" {
			c.sendError("room name required")
			return
		}
		delete(c.rooms, msg.Room)
		c.hub.Unregister(c, msg.Room)

	case domain.MsgChat:
		if msg.Room == "" || msg.Text == "" {
			c.sendError("room and text required")
			return
		}
		if !c.rooms[msg.Room] {
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
	if data, err := domain.Encode(errMsg); err == nil {
		c.Send(data)
	}
}
