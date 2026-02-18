package hub

import (
	"log"
	"sync"

	"github.com/devaloi/chatterbox/internal/domain"
	"github.com/devaloi/chatterbox/internal/store"
)

// RegisterRequest asks the hub to register a client.
type RegisterRequest struct {
	Client Client
	Room   string
}

// UnregisterRequest asks the hub to unregister a client from a room.
type UnregisterRequest struct {
	Client Client
	Room   string
}

// MessageRequest routes a message through the hub.
type MessageRequest struct {
	Message domain.Message
	Sender  Client
}

// Hub manages all rooms and routes messages between clients.
type Hub struct {
	rooms      map[string]*Room
	mu         sync.RWMutex
	register   chan RegisterRequest
	unregister chan UnregisterRequest
	message    chan MessageRequest
	store      store.Store
	maxRooms   int
	maxHistory int
	quit       chan struct{}
}

// New creates a new Hub.
func New(s store.Store, maxRooms, maxHistory int) *Hub {
	return &Hub{
		rooms:      make(map[string]*Room),
		register:   make(chan RegisterRequest, 256),
		unregister: make(chan UnregisterRequest, 256),
		message:    make(chan MessageRequest, 256),
		store:      s,
		maxRooms:   maxRooms,
		maxHistory: maxHistory,
		quit:       make(chan struct{}),
	}
}

// Run starts the hub's main event loop. Should be called as a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case req := <-h.register:
			h.handleRegister(req)
		case req := <-h.unregister:
			h.handleUnregister(req)
		case req := <-h.message:
			h.handleMessage(req)
		case <-h.quit:
			return
		}
	}
}

// Stop signals the hub's event loop to exit and stops all rooms.
func (h *Hub) Stop() {
	close(h.quit)
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, r := range h.rooms {
		r.Stop()
	}
}

// Register queues a client registration request.
func (h *Hub) Register(client Client, room string) {
	h.register <- RegisterRequest{Client: client, Room: room}
}

// Unregister queues a client unregistration request.
func (h *Hub) Unregister(client Client, room string) {
	h.unregister <- UnregisterRequest{Client: client, Room: room}
}

// RouteMessage queues a message for routing.
func (h *Hub) RouteMessage(msg domain.Message, sender Client) {
	h.message <- MessageRequest{Message: msg, Sender: sender}
}

// ListRooms returns info about all active rooms.
func (h *Hub) ListRooms() []domain.Room {
	h.mu.RLock()
	defer h.mu.RUnlock()
	rooms := make([]domain.Room, 0, len(h.rooms))
	for _, r := range h.rooms {
		rooms = append(rooms, domain.Room{
			Name:      r.Name(),
			UserCount: r.ClientCount(),
		})
	}
	return rooms
}

// RoomInfo returns details about a specific room, or nil if not found.
func (h *Hub) RoomInfo(name string) *domain.Room {
	h.mu.RLock()
	defer h.mu.RUnlock()
	r, ok := h.rooms[name]
	if !ok {
		return nil
	}
	return &domain.Room{
		Name:      r.Name(),
		UserCount: r.ClientCount(),
	}
}

func (h *Hub) handleRegister(req RegisterRequest) {
	h.mu.Lock()
	r, ok := h.rooms[req.Room]
	if !ok {
		if len(h.rooms) >= h.maxRooms {
			h.mu.Unlock()
			errMsg := domain.ErrorMessage{Type: domain.MsgError, Message: "max rooms reached"}
			if data, err := domain.Encode(errMsg); err == nil {
				req.Client.Send(data)
			}
			return
		}
		r = NewRoom(req.Room, h.store, h.maxHistory)
		h.rooms[req.Room] = r
		go r.Run()
		log.Printf("room created: %s", req.Room)
	}
	h.mu.Unlock()
	r.Join(req.Client)
}

func (h *Hub) handleUnregister(req UnregisterRequest) {
	h.mu.Lock()
	r, ok := h.rooms[req.Room]
	if !ok {
		h.mu.Unlock()
		return
	}
	h.mu.Unlock()

	r.Leave(req.Client)

	// Auto-cleanup empty rooms.
	if r.ClientCount() == 0 {
		h.mu.Lock()
		// Double-check after acquiring write lock.
		if r.ClientCount() == 0 {
			r.Stop()
			delete(h.rooms, req.Room)
			log.Printf("room deleted: %s", req.Room)
		}
		h.mu.Unlock()
	}
}

func (h *Hub) handleMessage(req MessageRequest) {
	h.mu.RLock()
	r, ok := h.rooms[req.Message.Room]
	h.mu.RUnlock()
	if !ok {
		errMsg := domain.ErrorMessage{Type: domain.MsgError, Message: "room not found"}
		if data, err := domain.Encode(errMsg); err == nil {
			req.Sender.Send(data)
		}
		return
	}

	// Persist the message.
	if h.store != nil {
		if err := h.store.Save(req.Message); err != nil {
			log.Printf("store save error: %v", err)
		}
	}

	if data, err := domain.Encode(req.Message); err == nil {
		r.Broadcast(data)
	}
}
