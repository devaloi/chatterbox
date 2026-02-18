package hub

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/devaloi/chatterbox/internal/domain"
)

// mockClient implements the Client interface for testing.
type mockClient struct {
	name     string
	messages [][]byte
	mu       sync.Mutex
}

func newMockClient(name string) *mockClient {
	return &mockClient{name: name}
}

func (m *mockClient) Username() string { return m.name }

func (m *mockClient) Send(data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]byte, len(data))
	copy(cp, data)
	m.messages = append(m.messages, cp)
}

func (m *mockClient) getMessages() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([][]byte, len(m.messages))
	copy(cp, m.messages)
	return cp
}

// mockStore implements store.Store for testing.
type mockStore struct {
	mu       sync.Mutex
	messages map[string][]domain.Message
}

func newMockStore() *mockStore {
	return &mockStore{messages: make(map[string][]domain.Message)}
}

func (s *mockStore) Save(msg domain.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages[msg.Room] = append(s.messages[msg.Room], msg)
	return nil
}

func (s *mockStore) History(room string, limit int) ([]domain.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	msgs := s.messages[room]
	if len(msgs) > limit {
		msgs = msgs[len(msgs)-limit:]
	}
	return msgs, nil
}

func (s *mockStore) Close() error { return nil }

func TestRoomJoinLeave(t *testing.T) {
	t.Parallel()
	s := newMockStore()
	r := NewRoom("test", s, 50)
	go r.Run()
	defer r.Stop()

	c1 := newMockClient("alice")
	c2 := newMockClient("bob")

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
	s := newMockStore()
	r := NewRoom("test", s, 50)
	go r.Run()
	defer r.Stop()

	c1 := newMockClient("alice")
	c2 := newMockClient("bob")

	r.Join(c1)
	r.Join(c2)
	time.Sleep(50 * time.Millisecond)

	msg := domain.Message{Type: domain.MsgChat, Room: "test", User: "alice", Text: "hello"}
	data, _ := domain.Encode(msg)
	r.Broadcast(data)
	time.Sleep(50 * time.Millisecond)

	// Both clients should have received the broadcast.
	for _, c := range []*mockClient{c1, c2} {
		msgs := c.getMessages()
		found := false
		for _, m := range msgs {
			var decoded domain.Message
			if err := json.Unmarshal(m, &decoded); err == nil && decoded.Type == domain.MsgChat && decoded.Text == "hello" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("client %s did not receive broadcast", c.name)
		}
	}
}

func TestRoomUsers(t *testing.T) {
	t.Parallel()
	r := NewRoom("test", nil, 50)
	go r.Run()
	defer r.Stop()

	c1 := newMockClient("alice")
	c2 := newMockClient("bob")

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
	s := newMockStore()
	// Pre-populate store with messages.
	for i := 0; i < 5; i++ {
		s.Save(domain.Message{Type: domain.MsgChat, Room: "test", User: "system", Text: "msg"})
	}

	r := NewRoom("test", s, 50)
	go r.Run()
	defer r.Stop()

	c := newMockClient("alice")
	r.Join(c)
	time.Sleep(50 * time.Millisecond)

	msgs := c.getMessages()
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
