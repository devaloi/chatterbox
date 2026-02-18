# Build chatterbox — Go WebSocket Chat Server

You are building a **portfolio project** for a Senior AI Engineer's public GitHub. It must be impressive, clean, and production-grade. Read these docs before writing any code:

1. **`G03-go-websocket-chat.md`** — Complete project spec: architecture, phases, hub/room design, commit plan. This is your primary blueprint. Follow it phase by phase.
2. **`github-portfolio.md`** — Portfolio goals and Definition of Done (Level 1 + Level 2). Understand the quality bar.
3. **`github-portfolio-checklist.md`** — Pre-publish checklist. Every item must pass before you're done.

---

## Instructions

### Read first, build second
Read all three docs completely before writing a single line of code. Understand the hub pattern, room management, WebSocket client lifecycle, and message persistence.

### Follow the phases in order
The project spec has 5 phases. Do them in order:
1. **Foundation** — project setup, core types (Message, Room, Client), config
2. **Hub + Rooms** — central hub for client management, named rooms, join/leave/broadcast
3. **WebSocket Client** — gorilla/websocket upgrade, read/write pumps, ping/pong heartbeat, clean disconnect
4. **HTTP + Persistence** — REST endpoints for room list/history, SQLite message persistence, single-file HTML frontend
5. **Polish** — comprehensive tests, race condition checks, refactor, README

### Commit frequently
Follow the commit plan in the spec. Use **conventional commits**. Each commit should be a logical unit.

### Quality non-negotiables
- **Hub pattern.** Central hub goroutine manages all clients. Channels for register/unregister/broadcast. No mutexes on the hot path — channel-based concurrency.
- **gorilla/websocket.** Proper upgrade, read/write pumps in separate goroutines, ping/pong keepalive, configurable timeouts.
- **Named rooms.** Clients join rooms. Messages broadcast only to room members. Room creation/deletion is dynamic.
- **Clean disconnect handling.** Client disconnect detected via read error or pong timeout. Hub notified. Room membership cleaned up. No goroutine leaks.
- **Message history.** SQLite persistence. REST endpoint to fetch room history with pagination.
- **Race-free.** `go test -race` must pass. The hub pattern should make this natural, but verify.
- **Single-file HTML frontend.** One HTML file with embedded CSS/JS. Connects via WebSocket. Shows rooms, messages, user list. Minimal but functional.
- **Lint clean.** `golangci-lint run` and `go vet` must pass.
- **No Docker.** Just `go build` and `go run`. Open browser to test.

### What NOT to do
- Don't use mutexes for client management. Use channels and the hub pattern.
- Don't use any WebSocket library other than gorilla/websocket.
- Don't skip the ping/pong heartbeat. Dead connections must be detected.
- Don't make the frontend complex. One HTML file. Vanilla JS. No React, no build step.
- Don't leave `// TODO` or `// FIXME` comments anywhere.
- Don't commit the SQLite database file.

---

## GitHub Username

The GitHub username is **devaloi**. For Go module paths, use `github.com/devaloi/chatterbox`. All internal imports must use this module path.

## Start

Read the three docs. Then begin Phase 1 from `G03-go-websocket-chat.md`.
