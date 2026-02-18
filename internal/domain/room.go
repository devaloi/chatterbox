package domain

// Room represents a chat room.
type Room struct {
	Name       string `json:"name"`
	UserCount  int    `json:"user_count"`
	MessageCount int  `json:"message_count,omitempty"`
}
