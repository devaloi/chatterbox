package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/devaloi/chatterbox/internal/domain"
	"github.com/devaloi/chatterbox/internal/handler"
	"github.com/devaloi/chatterbox/internal/hub"
	"github.com/devaloi/chatterbox/internal/store"
)

func setupServer(t *testing.T) (*httptest.Server, *hub.Hub, *store.SQLiteStore) {
	t.Helper()
	s, err := store.NewSQLite(":memory:")
	if err != nil {
		t.Fatalf("store: %v", err)
	}

	h := hub.New(s, 100, 50)
	go h.Run()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handler.ServeWS(h))
	mux.HandleFunc("/health", handler.Health())
	mux.HandleFunc("/api/rooms", handler.ListRooms(h))
	mux.HandleFunc("/api/rooms/", handler.RoomInfo(h))

	server := httptest.NewServer(mux)
	return server, h, s
}

func dialWS(t *testing.T, serverURL, user string) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(serverURL, "http") + "/ws?user=" + user
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial %s: %v", user, err)
	}
	return conn
}

func readUntilType(t *testing.T, conn *websocket.Conn, msgType string, maxReads int) map[string]interface{} {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for i := 0; i < maxReads; i++ {
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read while looking for %s: %v", msgType, err)
		}
		var msg map[string]interface{}
		json.Unmarshal(data, &msg)
		if msg["type"] == msgType {
			return msg
		}
	}
	t.Fatalf("did not find message type %s in %d reads", msgType, maxReads)
	return nil
}

func TestMultiClientBroadcast(t *testing.T) {
	t.Parallel()
	server, h, s := setupServer(t)
	defer server.Close()
	defer h.Stop()
	defer s.Close()

	alice := dialWS(t, server.URL, "alice")
	defer alice.Close()
	bob := dialWS(t, server.URL, "bob")
	defer bob.Close()
	charlie := dialWS(t, server.URL, "charlie")
	defer charlie.Close()

	// All join "general".
	for _, c := range []*websocket.Conn{alice, bob, charlie} {
		c.WriteMessage(websocket.TextMessage, []byte(`{"type":"join","room":"general"}`))
	}
	time.Sleep(300 * time.Millisecond)

	// Alice sends a message.
	alice.WriteMessage(websocket.TextMessage, []byte(`{"type":"chat","room":"general","text":"hello all"}`))

	// Bob and Charlie should receive it.
	for _, c := range []*websocket.Conn{bob, charlie} {
		msg := readUntilType(t, c, "chat", 10)
		if msg["text"] != "hello all" {
			t.Errorf("expected 'hello all', got %v", msg["text"])
		}
	}
}

func TestPresenceUpdates(t *testing.T) {
	t.Parallel()
	server, h, s := setupServer(t)
	defer server.Close()
	defer h.Stop()
	defer s.Close()

	alice := dialWS(t, server.URL, "alice")
	defer alice.Close()

	alice.WriteMessage(websocket.TextMessage, []byte(`{"type":"join","room":"general"}`))
	pm := readUntilType(t, alice, "presence", 5)
	users := pm["users"].([]interface{})
	if len(users) != 1 {
		t.Errorf("expected 1 user in presence, got %d", len(users))
	}
}

func TestHistoryOnJoin(t *testing.T) {
	t.Parallel()
	server, h, s := setupServer(t)
	defer server.Close()
	defer h.Stop()
	defer s.Close()

	// Pre-populate messages.
	for i := 0; i < 5; i++ {
		s.Save(domain.Message{
			Type: domain.MsgChat, Room: "general", User: "system",
			Text: "old msg", Timestamp: time.Now().UTC(),
		})
	}

	alice := dialWS(t, server.URL, "alice")
	defer alice.Close()

	alice.WriteMessage(websocket.TextMessage, []byte(`{"type":"join","room":"general"}`))
	hm := readUntilType(t, alice, "history", 5)
	msgs := hm["messages"].([]interface{})
	if len(msgs) != 5 {
		t.Errorf("expected 5 history messages, got %d", len(msgs))
	}
}

func TestDisconnectBroadcastsLeave(t *testing.T) {
	t.Parallel()
	server, h, s := setupServer(t)
	defer server.Close()
	defer h.Stop()
	defer s.Close()

	alice := dialWS(t, server.URL, "alice")
	defer alice.Close()
	bob := dialWS(t, server.URL, "bob")

	alice.WriteMessage(websocket.TextMessage, []byte(`{"type":"join","room":"general"}`))
	time.Sleep(100 * time.Millisecond)
	bob.WriteMessage(websocket.TextMessage, []byte(`{"type":"join","room":"general"}`))
	time.Sleep(200 * time.Millisecond)

	// Bob disconnects.
	bob.Close()
	time.Sleep(300 * time.Millisecond)

	msg := readUntilType(t, alice, "leave", 10)
	if msg["user"] != "bob" {
		t.Errorf("expected leave from bob, got %v", msg["user"])
	}
}

func TestRESTRoomList(t *testing.T) {
	t.Parallel()
	server, h, s := setupServer(t)
	defer server.Close()
	defer h.Stop()
	defer s.Close()

	alice := dialWS(t, server.URL, "alice")
	defer alice.Close()
	alice.WriteMessage(websocket.TextMessage, []byte(`{"type":"join","room":"general"}`))
	time.Sleep(200 * time.Millisecond)

	resp, err := http.Get(server.URL + "/api/rooms")
	if err != nil {
		t.Fatalf("get rooms: %v", err)
	}
	defer resp.Body.Close()

	var rooms []domain.Room
	json.NewDecoder(resp.Body).Decode(&rooms)
	if len(rooms) != 1 {
		t.Fatalf("expected 1 room, got %d", len(rooms))
	}
	if rooms[0].Name != "general" {
		t.Errorf("expected room 'general', got %q", rooms[0].Name)
	}
	if rooms[0].UserCount != 1 {
		t.Errorf("expected 1 user, got %d", rooms[0].UserCount)
	}
}

func TestHealthEndpoint(t *testing.T) {
	t.Parallel()
	server, h, s := setupServer(t)
	defer server.Close()
	defer h.Stop()
	defer s.Close()

	resp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("get health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("expected ok, got %s", body["status"])
	}
}

func TestMultipleRooms(t *testing.T) {
	t.Parallel()
	server, h, s := setupServer(t)
	defer server.Close()
	defer h.Stop()
	defer s.Close()

	alice := dialWS(t, server.URL, "alice")
	defer alice.Close()
	bob := dialWS(t, server.URL, "bob")
	defer bob.Close()

	alice.WriteMessage(websocket.TextMessage, []byte(`{"type":"join","room":"room1"}`))
	bob.WriteMessage(websocket.TextMessage, []byte(`{"type":"join","room":"room2"}`))
	time.Sleep(200 * time.Millisecond)

	alice.WriteMessage(websocket.TextMessage, []byte(`{"type":"chat","room":"room1","text":"only for room1"}`))
	time.Sleep(200 * time.Millisecond)

	// Bob should NOT receive room1 messages.
	bob.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	for {
		_, data, err := bob.ReadMessage()
		if err != nil {
			break
		}
		var msg map[string]interface{}
		json.Unmarshal(data, &msg)
		if msg["type"] == "chat" && msg["text"] == "only for room1" {
			t.Error("bob in room2 should not receive room1 message")
		}
	}
}
