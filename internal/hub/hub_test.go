package hub

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/devaloi/chatterbox/internal/domain"
	"github.com/devaloi/chatterbox/internal/testutil"
)

func TestHubCreateRoom(t *testing.T) {
	t.Parallel()
	s := testutil.NewMockStore()
	h := New(s, 100, 50)
	go h.Run()
	defer h.Stop()

	c := testutil.NewMockClient("alice")
	h.Register(c, "general")
	time.Sleep(100 * time.Millisecond)

	rooms := h.ListRooms()
	if len(rooms) != 1 {
		t.Fatalf("expected 1 room, got %d", len(rooms))
	}
	if rooms[0].Name != "general" {
		t.Errorf("expected room 'general', got %q", rooms[0].Name)
	}
}

func TestHubRoomInfo(t *testing.T) {
	t.Parallel()
	s := testutil.NewMockStore()
	h := New(s, 100, 50)
	go h.Run()
	defer h.Stop()

	c := testutil.NewMockClient("alice")
	h.Register(c, "general")
	time.Sleep(100 * time.Millisecond)

	info := h.RoomInfo("general")
	if info == nil {
		t.Fatal("expected room info, got nil")
	}
	if info.UserCount != 1 {
		t.Errorf("expected 1 user, got %d", info.UserCount)
	}

	if h.RoomInfo("nonexistent") != nil {
		t.Error("expected nil for nonexistent room")
	}
}

func TestHubRouteMessage(t *testing.T) {
	t.Parallel()
	s := testutil.NewMockStore()
	h := New(s, 100, 50)
	go h.Run()
	defer h.Stop()

	c1 := testutil.NewMockClient("alice")
	c2 := testutil.NewMockClient("bob")
	h.Register(c1, "general")
	h.Register(c2, "general")
	time.Sleep(100 * time.Millisecond)

	msg := domain.Message{
		Type:      domain.MsgChat,
		Room:      "general",
		User:      "alice",
		Text:      "hello",
		Timestamp: time.Now(),
	}
	h.RouteMessage(msg, c1)
	time.Sleep(100 * time.Millisecond)

	// Both clients should receive the message.
	for _, c := range []*testutil.MockClient{c1, c2} {
		msgs := c.GetMessages()
		found := false
		for _, m := range msgs {
			var decoded domain.Message
			if err := json.Unmarshal(m, &decoded); err == nil && decoded.Text == "hello" {
				found = true
			}
		}
		if !found {
			t.Errorf("client %s did not receive message", c.Name)
		}
	}

	// Message should be persisted.
	history, _ := s.History("general", 50)
	if len(history) != 1 {
		t.Errorf("expected 1 stored message, got %d", len(history))
	}
}

func TestHubAutoCleanup(t *testing.T) {
	t.Parallel()
	s := testutil.NewMockStore()
	h := New(s, 100, 50)
	go h.Run()
	defer h.Stop()

	c := testutil.NewMockClient("alice")
	h.Register(c, "temp")
	time.Sleep(100 * time.Millisecond)

	if len(h.ListRooms()) != 1 {
		t.Fatal("expected 1 room")
	}

	h.Unregister(c, "temp")
	time.Sleep(100 * time.Millisecond)

	if len(h.ListRooms()) != 0 {
		t.Error("expected room to be auto-deleted")
	}
}

func TestHubMaxRooms(t *testing.T) {
	t.Parallel()
	s := testutil.NewMockStore()
	h := New(s, 2, 50)
	go h.Run()
	defer h.Stop()

	c1 := testutil.NewMockClient("alice")
	c2 := testutil.NewMockClient("bob")
	c3 := testutil.NewMockClient("charlie")

	h.Register(c1, "room1")
	h.Register(c2, "room2")
	time.Sleep(100 * time.Millisecond)

	h.Register(c3, "room3")
	time.Sleep(100 * time.Millisecond)

	if len(h.ListRooms()) != 2 {
		t.Errorf("expected 2 rooms (max), got %d", len(h.ListRooms()))
	}

	// c3 should have received an error.
	msgs := c3.GetMessages()
	found := false
	for _, m := range msgs {
		var em domain.ErrorMessage
		if err := json.Unmarshal(m, &em); err == nil && em.Type == domain.MsgError {
			found = true
		}
	}
	if !found {
		t.Error("expected error message for max rooms")
	}
}
