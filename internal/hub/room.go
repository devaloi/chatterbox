package hub

import (
	"log"
	"sync"

	"github.com/devaloi/chatterbox/internal/domain"
	"github.com/devaloi/chatterbox/internal/store"
)

// roomBroadcastBuffer is the channel buffer size for room broadcast messages.
const roomBroadcastBuffer = 256

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
	stopOnce  sync.Once
}

// NewRoom creates a new room with the given name and message store.
func NewRoom(name string, s store.Store, historyLimit int) *Room {
	return &Room{
		name:      name,
		clients:   make(map[Client]bool),
		broadcast: make(chan []byte, roomBroadcastBuffer),
		store:     s,
		history:   historyLimit,
		quit:      make(chan struct{}),
	}
}

// Run starts the room's broadcast loop. Should be called as a goroutine.
// Uses panic recovery so one room crash doesn't bring down the whole server.
func (r *Room) Run() {
	defer func() {
		if rv := recover(); rv != nil {
			log.Printf("room %s: recovered from panic: %v", r.name, rv)
		}
	}()

	for {
		select {
		case msg := <-r.broadcast:
			// Copy client list under lock, then send outside lock to avoid
			// holding the read lock while calling into client Send methods
			// (which may block or acquire their own locks).
			r.mu.RLock()
			clients := make([]Client, 0, len(r.clients))
			for c := range r.clients {
				clients = append(clients, c)
			}
			r.mu.RUnlock()

			for _, c := range clients {
				c.Send(msg)
			}
		case <-r.quit:
			return
		}
	}
}

// Stop signals the room's broadcast loop to exit.
// Safe to call multiple times; only the first call takes effect.
func (r *Room) Stop() {
	r.stopOnce.Do(func() {
		close(r.quit)
	})
}

// Join adds a client to the room and sends history + presence.
func (r *Room) Join(c Client) {
	r.mu.Lock()
	r.clients[c] = true
	r.mu.Unlock()

	// Send message history to the joining client.
	if r.store != nil {
		msgs, err := r.store.History(r.name, r.history)
		if err != nil {
			log.Printf("room %s: history error: %v", r.name, err)
		} else if len(msgs) > 0 {
			hm := domain.HistoryMessage{
				Type:     domain.MsgHistory,
				Room:     r.name,
				Messages: msgs,
			}
			data, err := domain.Encode(hm)
			if err != nil {
				log.Printf("room %s: encode history error: %v", r.name, err)
			} else {
				c.Send(data)
			}
		}
	}

	// Broadcast join notification.
	joinMsg := domain.Message{Type: domain.MsgJoin, Room: r.name, User: c.Username()}
	data, err := domain.Encode(joinMsg)
	if err != nil {
		log.Printf("room %s: encode join error: %v", r.name, err)
	} else {
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
	data, err := domain.Encode(leaveMsg)
	if err != nil {
		log.Printf("room %s: encode leave error: %v", r.name, err)
	} else {
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
	data, err := domain.Encode(pm)
	if err != nil {
		log.Printf("room %s: encode presence error: %v", r.name, err)
		return
	}
	c.Send(data)
}
