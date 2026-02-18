package domain

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMessageEncodeDecode(t *testing.T) {
	t.Parallel()
	now := time.Now().Truncate(time.Second)
	original := Message{
		Type:      MsgChat,
		Room:      "general",
		User:      "alice",
		Text:      "hello world",
		Timestamp: now,
	}

	data, err := Encode(original)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	decoded, err := DecodeMessage(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if decoded.Type != original.Type {
		t.Errorf("type: got %q, want %q", decoded.Type, original.Type)
	}
	if decoded.Room != original.Room {
		t.Errorf("room: got %q, want %q", decoded.Room, original.Room)
	}
	if decoded.User != original.User {
		t.Errorf("user: got %q, want %q", decoded.User, original.User)
	}
	if decoded.Text != original.Text {
		t.Errorf("text: got %q, want %q", decoded.Text, original.Text)
	}
}

func TestHistoryMessageEncode(t *testing.T) {
	t.Parallel()
	hm := HistoryMessage{
		Type: MsgHistory,
		Room: "general",
		Messages: []Message{
			{Type: MsgChat, Room: "general", User: "alice", Text: "hi"},
		},
	}
	data, err := Encode(hm)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := raw["messages"]; !ok {
		t.Error("expected messages field in history message")
	}
}

func TestPresenceMessageEncode(t *testing.T) {
	t.Parallel()
	pm := PresenceMessage{
		Type:  MsgPresence,
		Room:  "general",
		Users: []string{"alice", "bob"},
	}
	data, err := Encode(pm)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := raw["users"]; !ok {
		t.Error("expected users field in presence message")
	}
}

func TestErrorMessageEncode(t *testing.T) {
	t.Parallel()
	em := ErrorMessage{
		Type:    MsgError,
		Message: "bad request",
	}
	data, err := Encode(em)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var decoded ErrorMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Message != "bad request" {
		t.Errorf("message: got %q, want %q", decoded.Message, "bad request")
	}
}

func TestDecodeInvalidJSON(t *testing.T) {
	t.Parallel()
	_, err := DecodeMessage([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestMessageTypes(t *testing.T) {
	t.Parallel()
	types := []string{MsgChat, MsgJoin, MsgLeave, MsgSystem, MsgHistory, MsgPresence, MsgError}
	expected := []string{"chat", "join", "leave", "system", "history", "presence", "error"}
	for i, typ := range types {
		if typ != expected[i] {
			t.Errorf("type %d: got %q, want %q", i, typ, expected[i])
		}
	}
}
