package testutil

import (
	"sync"

	"github.com/devaloi/chatterbox/internal/domain"
)

// MockClient implements hub.Client for testing.
type MockClient struct {
	Name     string
	messages [][]byte
	mu       sync.Mutex
}

// NewMockClient creates a new MockClient with the given name.
func NewMockClient(name string) *MockClient {
	return &MockClient{Name: name}
}

// Username returns the mock client's name.
func (m *MockClient) Username() string { return m.Name }

// Send records a message sent to the mock client.
func (m *MockClient) Send(data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]byte, len(data))
	copy(cp, data)
	m.messages = append(m.messages, cp)
}

// GetMessages returns a copy of all messages received by the mock client.
func (m *MockClient) GetMessages() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([][]byte, len(m.messages))
	copy(cp, m.messages)
	return cp
}

// MockStore implements store.Store for testing.
type MockStore struct {
	mu       sync.Mutex
	messages map[string][]domain.Message
}

// NewMockStore creates a new MockStore.
func NewMockStore() *MockStore {
	return &MockStore{messages: make(map[string][]domain.Message)}
}

// Save persists a message in the mock store.
func (s *MockStore) Save(msg domain.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages[msg.Room] = append(s.messages[msg.Room], msg)
	return nil
}

// History returns stored messages for a room.
func (s *MockStore) History(room string, limit int) ([]domain.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	msgs := s.messages[room]
	if len(msgs) > limit {
		msgs = msgs[len(msgs)-limit:]
	}
	return msgs, nil
}

// Close is a no-op for the mock store.
func (s *MockStore) Close() error { return nil }
