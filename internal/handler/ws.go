package handler

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/devaloi/chatterbox/internal/client"
	"github.com/devaloi/chatterbox/internal/hub"
)

// WebSocket read/write buffer sizes (bytes).
const (
	wsReadBufferSize  = 1024
	wsWriteBufferSize = 1024
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  wsReadBufferSize,
	WriteBufferSize: wsWriteBufferSize,
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
