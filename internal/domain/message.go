package domain

import (
	"encoding/json"
	"time"
)

// Message types.
const (
	MsgChat    = "chat"
	MsgJoin    = "join"
	MsgLeave   = "leave"
	MsgSystem  = "system"
	MsgHistory = "history"
	MsgPresence = "presence"
	MsgError   = "error"
)

// Message represents a chat protocol message.
type Message struct {
	Type      string    `json:"type"`
	Room      string    `json:"room,omitempty"`
	User      string    `json:"user,omitempty"`
	Text      string    `json:"text,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

// HistoryMessage is sent to a client upon joining a room.
type HistoryMessage struct {
	Type     string    `json:"type"`
	Room     string    `json:"room"`
	Messages []Message `json:"messages"`
}

// PresenceMessage lists current users in a room.
type PresenceMessage struct {
	Type  string   `json:"type"`
	Room  string   `json:"room"`
	Users []string `json:"users"`
}

// ErrorMessage reports an error to the client.
type ErrorMessage struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Encode serializes a value to JSON bytes.
func Encode(v any) ([]byte, error) {
	return json.Marshal(v)
}

// DecodeMessage deserializes JSON bytes into a Message.
func DecodeMessage(data []byte) (Message, error) {
	var m Message
	err := json.Unmarshal(data, &m)
	return m, err
}
