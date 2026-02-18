package hub

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/devaloi/chatterbox/internal/domain"
	"github.com/devaloi/chatterbox/internal/testutil"
)

func TestRoomJoinLeave(t *testing.T) {
	t.Parallel()
	s := testutil.NewMockStore()
	r := NewRoom("test", s, 50)
	go r.Run()
	defer r.Stop()

	c1 := testutil.NewMockClient("alice")
	c2 := testutil.NewMockClient("bob")

	r.Join(c1)
	time.Sleep(50 * time.Millisecond)

	if r.ClientCount() != 1 {
		t.Errorf("expected 1 client, got %d", r.ClientCount())
	}

	r.Join(c2)
	time.Sleep(50 * time.Millisecond)

	if r.ClientCount() != 2 {
		t.Errorf("expected 2 clients, got %d", r.ClientCount())
	}

	r.Leave(c1)
	time.Sleep(50 * time.Millisecond)

	if r.ClientCount() != 1 {
		t.Errorf("expected 1 client, got %d", r.ClientCount())
	}
}

func TestRoomBroadcast(t *testing.T) {
	t.Parallel()
	s := testutil.NewMockStore()
	r := NewRoom("test", s, 50)
	go r.Run()
	defer r.Stop()

	c1 := testutil.NewMockClient("alice")
	c2 := testutil.NewMockClient("bob")

	r.Join(c1)
	r.Join(c2)
	time.Sleep(50 * time.Millisecond)

	msg := domain.Message{Type: domain.MsgChat, Room: "test", User: "alice", Text: "hello"}
	data, _ := domain.Encode(msg)
	r.Broadcast(data)
	time.Sleep(50 * time.Millisecond)

	// Both clients should have received the broadcast.
	for _, c := range []*testutil.MockClient{c1, c2} {
		msgs := c.GetMessages()
		found := false
		for _, m := range msgs {
			var decoded domain.Message
			if err := json.Unmarshal(m, &decoded); err == nil && decoded.Type == domain.MsgChat && decoded.Text == "hello" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("client %s did not receive broadcast", c.Name)
		}
	}
}

func TestRoomUsers(t *testing.T) {
	t.Parallel()
	r := NewRoom("test", nil, 50)
	go r.Run()
	defer r.Stop()

	c1 := testutil.NewMockClient("alice")
	c2 := testutil.NewMockClient("bob")

	r.Join(c1)
	r.Join(c2)
	time.Sleep(50 * time.Millisecond)

	users := r.Users()
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestRoomHistoryOnJoin(t *testing.T) {
	t.Parallel()
	s := testutil.NewMockStore()
	// Pre-populate store with messages.
	for i := 0; i < 5; i++ {
		s.Save(domain.Message{Type: domain.MsgChat, Room: "test", User: "system", Text: "msg"})
	}

	r := NewRoom("test", s, 50)
	go r.Run()
	defer r.Stop()

	c := testutil.NewMockClient("alice")
	r.Join(c)
	time.Sleep(50 * time.Millisecond)

	msgs := c.GetMessages()
	foundHistory := false
	for _, m := range msgs {
		var hm domain.HistoryMessage
		if err := json.Unmarshal(m, &hm); err == nil && hm.Type == domain.MsgHistory {
			foundHistory = true
			if len(hm.Messages) != 5 {
				t.Errorf("expected 5 history messages, got %d", len(hm.Messages))
			}
		}
	}
	if !foundHistory {
		t.Error("expected history message on join")
	}
}
