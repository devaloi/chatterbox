package main

import (
	"log"
	"net/http"

	"github.com/devaloi/chatterbox/internal/config"
	"github.com/devaloi/chatterbox/internal/handler"
	"github.com/devaloi/chatterbox/internal/hub"
	"github.com/devaloi/chatterbox/internal/middleware"
	"github.com/devaloi/chatterbox/internal/store"
)

func main() {
	cfg := config.Load()

	s, err := store.NewSQLite(cfg.DBPath)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer s.Close()

	h := hub.New(s, cfg.MaxRooms, cfg.MaxHistory)
	go h.Run()
	defer h.Stop()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handler.Health())
	mux.HandleFunc("/api/rooms", handler.ListRooms(h))
	mux.HandleFunc("/api/rooms/", handler.RoomInfo(h))
	mux.HandleFunc("/ws", handler.ServeWS(h))
	mux.Handle("/", http.FileServer(http.Dir("static")))

	wrapped := middleware.Logging(middleware.CORS(mux))

	addr := ":" + cfg.Port
	log.Printf("chatterbox listening on %s", addr)
	if err := http.ListenAndServe(addr, wrapped); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
