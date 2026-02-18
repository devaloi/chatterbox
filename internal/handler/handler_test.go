package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/devaloi/chatterbox/internal/hub"
	"github.com/devaloi/chatterbox/internal/testutil"
)

func TestHealth(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	Health()(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("expected ok, got %s", body["status"])
	}
}

func TestListRoomsEmpty(t *testing.T) {
	t.Parallel()
	s := testutil.NewMockStore()
	h := hub.New(s, 100, 50)
	go h.Run()
	defer h.Stop()

	req := httptest.NewRequest(http.MethodGet, "/api/rooms", nil)
	w := httptest.NewRecorder()
	ListRooms(h)(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRoomInfoNotFound(t *testing.T) {
	t.Parallel()
	s := testutil.NewMockStore()
	h := hub.New(s, 100, 50)
	go h.Run()
	defer h.Stop()

	req := httptest.NewRequest(http.MethodGet, "/api/rooms/nonexistent", nil)
	w := httptest.NewRecorder()
	RoomInfo(h)(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestWSUpgradeNoUser(t *testing.T) {
	t.Parallel()
	s := testutil.NewMockStore()
	h := hub.New(s, 100, 50)
	go h.Run()
	defer h.Stop()

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	w := httptest.NewRecorder()
	ServeWS(h)(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestWSUpgradeSuccess(t *testing.T) {
	t.Parallel()
	s := testutil.NewMockStore()
	h := hub.New(s, 100, 50)
	go h.Run()
	defer h.Stop()

	server := httptest.NewServer(ServeWS(h))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "?user=alice"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Send a join message.
	conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"join","room":"general"}`))
	time.Sleep(200 * time.Millisecond)

	// Should receive join + presence.
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var msg map[string]interface{}
	json.Unmarshal(data, &msg)
	if msg["type"] != "join" && msg["type"] != "presence" {
		t.Errorf("unexpected first message type: %v", msg["type"])
	}
}
