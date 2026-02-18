package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/devaloi/chatterbox/internal/hub"
)

// Health returns a simple health check handler.
func Health() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}

// ListRooms returns all active rooms with user counts.
func ListRooms(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		rooms := h.ListRooms()
		json.NewEncoder(w).Encode(rooms)
	}
}

// RoomInfo returns details about a specific room.
func RoomInfo(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract room name from path: /api/rooms/{name}
		name := strings.TrimPrefix(r.URL.Path, "/api/rooms/")
		if name == "" {
			http.Error(w, `{"error":"room name required"}`, http.StatusBadRequest)
			return
		}

		info := h.RoomInfo(name)
		if info == nil {
			http.Error(w, `{"error":"room not found"}`, http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	}
}
