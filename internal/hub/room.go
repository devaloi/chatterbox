package hub

import (
	"sync"

	"github.com/devaloi/chatterbox/internal/domain"
	"github.com/devaloi/chatterbox/internal/store"
)

// Client is the interface that hub/room expects from a WebSocket client.
type Client interface {
	Username() string
	Send(data []byte)
}

// Room manages a set of clients and broadcasts messages to them.
type Room struct {
	name      string
	clients   map[Client]bool
	mu        sync.RWMutex
	broadcast chan []byte
	store     store.Store
	history   int
	quit      chan struct{}
}

// NewRoom creates a new room with the given name and message store.
func NewRoom(name string, s store.Store, historyLimit int) *Room {
	return &Room{
		name:      name,
		clients:   make(map[Client]bool),
		broadcast: make(chan []byte, 256),
		store:     s,
		history:   historyLimit,
		quit:      make(chan struct{}),
	}
}

// Run starts the room's broadcast loop. Should be called as a goroutine.
func (r *Room) Run() {
	for {
		select {
		case msg := <-r.broadcast:
			r.mu.RLock()
			for c := range r.clients {
				c.Send(msg)
			}
			r.mu.RUnlock()
		case <-r.quit:
			return
		}
	}
}

// Stop signals the room's broadcast loop to exit.
func (r *Room) Stop() {
	close(r.quit)
}

// Join adds a client to the room and sends history + presence.
func (r *Room) Join(c Client) {
	r.mu.Lock()
	r.clients[c] = true
	r.mu.Unlock()

	// Send message history to the joining client.
	if r.store != nil {
		msgs, err := r.store.History(r.name, r.history)
		if err == nil && len(msgs) > 0 {
			hm := domain.HistoryMessage{
				Type:     domain.MsgHistory,
				Room:     r.name,
				Messages: msgs,
			}
			if data, err := domain.Encode(hm); err == nil {
				c.Send(data)
			}
		}
	}

	// Broadcast join notification.
	joinMsg := domain.Message{Type: domain.MsgJoin, Room: r.name, User: c.Username()}
	if data, err := domain.Encode(joinMsg); err == nil {
		r.broadcast <- data
	}

	// Send presence to the joining client.
	r.sendPresence(c)
}

// Leave removes a client from the room and broadcasts a leave notification.
func (r *Room) Leave(c Client) {
	r.mu.Lock()
	delete(r.clients, c)
	r.mu.Unlock()

	leaveMsg := domain.Message{Type: domain.MsgLeave, Room: r.name, User: c.Username()}
	if data, err := domain.Encode(leaveMsg); err == nil {
		r.broadcast <- data
	}
}

// Broadcast sends a raw JSON message to all clients in the room.
func (r *Room) Broadcast(data []byte) {
	r.broadcast <- data
}

// ClientCount returns the number of connected clients.
func (r *Room) ClientCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.clients)
}

// Name returns the room name.
func (r *Room) Name() string {
	return r.name
}

// Users returns a list of usernames in the room.
func (r *Room) Users() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	users := make([]string, 0, len(r.clients))
	for c := range r.clients {
		users = append(users, c.Username())
	}
	return users
}

func (r *Room) sendPresence(c Client) {
	pm := domain.PresenceMessage{
		Type:  domain.MsgPresence,
		Room:  r.name,
		Users: r.Users(),
	}
	if data, err := domain.Encode(pm); err == nil {
		c.Send(data)
	}
}
