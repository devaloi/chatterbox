package store

import "github.com/devaloi/chatterbox/internal/domain"

// Store defines the message persistence interface.
type Store interface {
	// Save persists a message.
	Save(msg domain.Message) error
	// History returns the last `limit` messages for a room, oldest first.
	History(room string, limit int) ([]domain.Message, error)
	// Close releases any resources held by the store.
	Close() error
}
