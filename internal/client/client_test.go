package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/devaloi/chatterbox/internal/domain"
	"github.com/devaloi/chatterbox/internal/hub"
)

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

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func setupTestServer(h *hub.Hub) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		username := r.URL.Query().Get("user")
		if username == "" {
			username = "test"
		}
		c := New(h, conn, username)
		go c.ReadPump()
		go c.WritePump()
	}))
}

func dialWS(t *testing.T, url string, user string) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(url, "http") + "?user=" + user
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	return conn
}

func readMessage(t *testing.T, conn *websocket.Conn) map[string]interface{} {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var msg map[string]interface{}
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return msg
}

func TestClientJoinAndChat(t *testing.T) {
	t.Parallel()
	s := newMockStore()
	h := hub.New(s, 100, 50)
	go h.Run()
	defer h.Stop()

	server := setupTestServer(h)
	defer server.Close()

	conn1 := dialWS(t, server.URL, "alice")
	defer conn1.Close()

	// Alice joins "general".
	joinMsg := `{"type":"join","room":"general"}`
	conn1.WriteMessage(websocket.TextMessage, []byte(joinMsg))

	// Read join notification and presence.
	var gotJoin, gotPresence bool
	for i := 0; i < 2; i++ {
		msg := readMessage(t, conn1)
		switch msg["type"] {
		case "join":
			gotJoin = true
		case "presence":
			gotPresence = true
		}
	}
	if !gotJoin {
		t.Error("expected join notification")
	}
	if !gotPresence {
		t.Error("expected presence message")
	}

	// Alice sends a chat message.
	chatMsg := `{"type":"chat","room":"general","text":"hello"}`
	conn1.WriteMessage(websocket.TextMessage, []byte(chatMsg))

	msg := readMessage(t, conn1)
	if msg["type"] != "chat" || msg["text"] != "hello" {
		t.Errorf("unexpected message: %v", msg)
	}
}

func TestClientBroadcast(t *testing.T) {
	t.Parallel()
	s := newMockStore()
	h := hub.New(s, 100, 50)
	go h.Run()
	defer h.Stop()

	server := setupTestServer(h)
	defer server.Close()

	conn1 := dialWS(t, server.URL, "alice")
	defer conn1.Close()
	conn2 := dialWS(t, server.URL, "bob")
	defer conn2.Close()

	// Both join "general".
	conn1.WriteMessage(websocket.TextMessage, []byte(`{"type":"join","room":"general"}`))
	time.Sleep(100 * time.Millisecond)
	conn2.WriteMessage(websocket.TextMessage, []byte(`{"type":"join","room":"general"}`))
	time.Sleep(200 * time.Millisecond)

	// Alice sends a message.
	conn1.WriteMessage(websocket.TextMessage, []byte(`{"type":"chat","room":"general","text":"hi everyone"}`))

	// Bob reads messages until he finds the chat message.
	found := false
	conn2.SetReadDeadline(time.Now().Add(3 * time.Second))
	for i := 0; i < 10; i++ {
		_, data, err := conn2.ReadMessage()
		if err != nil {
			break
		}
		var msg map[string]interface{}
		if err := json.Unmarshal(data, &msg); err == nil {
			if msg["type"] == "chat" && msg["text"] == "hi everyone" {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("bob did not receive broadcast chat message")
	}
}

func TestClientInvalidJSON(t *testing.T) {
	t.Parallel()
	s := newMockStore()
	h := hub.New(s, 100, 50)
	go h.Run()
	defer h.Stop()

	server := setupTestServer(h)
	defer server.Close()

	conn := dialWS(t, server.URL, "alice")
	defer conn.Close()

	conn.WriteMessage(websocket.TextMessage, []byte("not json"))
	msg := readMessage(t, conn)
	if msg["type"] != "error" {
		t.Errorf("expected error, got: %v", msg)
	}
}

func TestClientChatNotInRoom(t *testing.T) {
	t.Parallel()
	s := newMockStore()
	h := hub.New(s, 100, 50)
	go h.Run()
	defer h.Stop()

	server := setupTestServer(h)
	defer server.Close()

	conn := dialWS(t, server.URL, "alice")
	defer conn.Close()

	conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"chat","room":"general","text":"hi"}`))
	msg := readMessage(t, conn)
	if msg["type"] != "error" {
		t.Errorf("expected error for chat without join, got: %v", msg)
	}
}
