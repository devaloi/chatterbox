package store

import (
	"testing"
	"time"

	"github.com/devaloi/chatterbox/internal/domain"
)

func TestSQLiteSaveAndHistory(t *testing.T) {
	t.Parallel()
	s, err := NewSQLite(":memory:")
	if err != nil {
		t.Fatalf("new sqlite: %v", err)
	}
	defer s.Close()

	now := time.Now().UTC()
	msgs := []domain.Message{
		{Type: domain.MsgChat, Room: "general", User: "alice", Text: "msg1", Timestamp: now.Add(-2 * time.Second)},
		{Type: domain.MsgChat, Room: "general", User: "bob", Text: "msg2", Timestamp: now.Add(-1 * time.Second)},
		{Type: domain.MsgChat, Room: "general", User: "alice", Text: "msg3", Timestamp: now},
	}
	for _, m := range msgs {
		if err := s.Save(m); err != nil {
			t.Fatalf("save: %v", err)
		}
	}

	history, err := s.History("general", 50)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(history) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(history))
	}
	// Should be oldest-first.
	if history[0].Text != "msg1" {
		t.Errorf("expected msg1 first, got %s", history[0].Text)
	}
	if history[2].Text != "msg3" {
		t.Errorf("expected msg3 last, got %s", history[2].Text)
	}
}

func TestSQLiteHistoryLimit(t *testing.T) {
	t.Parallel()
	s, err := NewSQLite(":memory:")
	if err != nil {
		t.Fatalf("new sqlite: %v", err)
	}
	defer s.Close()

	now := time.Now().UTC()
	for i := 0; i < 10; i++ {
		s.Save(domain.Message{
			Type: domain.MsgChat, Room: "general", User: "alice",
			Text: "msg", Timestamp: now.Add(time.Duration(i) * time.Second),
		})
	}

	history, err := s.History("general", 5)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(history) != 5 {
		t.Errorf("expected 5 messages, got %d", len(history))
	}
}

func TestSQLiteRoomIsolation(t *testing.T) {
	t.Parallel()
	s, err := NewSQLite(":memory:")
	if err != nil {
		t.Fatalf("new sqlite: %v", err)
	}
	defer s.Close()

	s.Save(domain.Message{Type: domain.MsgChat, Room: "room1", User: "alice", Text: "hi", Timestamp: time.Now()})
	s.Save(domain.Message{Type: domain.MsgChat, Room: "room2", User: "bob", Text: "hi", Timestamp: time.Now()})

	h1, _ := s.History("room1", 50)
	h2, _ := s.History("room2", 50)

	if len(h1) != 1 || len(h2) != 1 {
		t.Errorf("expected 1 message per room, got room1=%d room2=%d", len(h1), len(h2))
	}
}

func TestSQLiteEmptyHistory(t *testing.T) {
	t.Parallel()
	s, err := NewSQLite(":memory:")
	if err != nil {
		t.Fatalf("new sqlite: %v", err)
	}
	defer s.Close()

	history, err := s.History("empty", 50)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(history) != 0 {
		t.Errorf("expected 0 messages, got %d", len(history))
	}
}
