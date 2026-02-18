package handler

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/devaloi/chatterbox/internal/client"
	"github.com/devaloi/chatterbox/internal/hub"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// ServeWS handles WebSocket upgrade requests.
func ServeWS(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.URL.Query().Get("user")
		if user == "" {
			http.Error(w, `{"error":"user query param required"}`, http.StatusBadRequest)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("ws upgrade error: %v", err)
			return
		}

		c := client.New(h, conn, user)
		go c.ReadPump()
		go c.WritePump()
	}
}
