package store

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite"

	"github.com/devaloi/chatterbox/internal/domain"
)

// SQLiteStore implements Store using SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLite opens or creates a SQLite database at the given path.
// Use ":memory:" for an in-memory database.
func NewSQLite(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	// Enable WAL mode for better concurrent read performance.
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, err
	}

	if err := createTables(db); err != nil {
		db.Close()
		return nil, err
	}

	return &SQLiteStore{db: db}, nil
}

func createTables(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			room TEXT NOT NULL,
			user TEXT NOT NULL,
			text TEXT NOT NULL,
			type TEXT NOT NULL,
			created_at DATETIME NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_messages_room_created ON messages(room, created_at);
	`)
	return err
}

// Save persists a message to the database.
func (s *SQLiteStore) Save(msg domain.Message) error {
	ts := msg.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	_, err := s.db.Exec(
		"INSERT INTO messages (room, user, text, type, created_at) VALUES (?, ?, ?, ?, ?)",
		msg.Room, msg.User, msg.Text, msg.Type, ts,
	)
	return err
}

// History returns the last `limit` messages for a room, oldest first.
func (s *SQLiteStore) History(room string, limit int) ([]domain.Message, error) {
	rows, err := s.db.Query(`
		SELECT room, user, text, type, created_at FROM messages
		WHERE room = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, room, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []domain.Message
	for rows.Next() {
		var m domain.Message
		if err := rows.Scan(&m.Room, &m.User, &m.Text, &m.Type, &m.Timestamp); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Reverse to oldest-first order.
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
