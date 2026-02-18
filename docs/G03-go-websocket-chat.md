# G03: chatterbox — Go WebSocket Chat Server

**Catalog ID:** G03 | **Size:** M | **Language:** Go
**Repo name:** `chatterbox`
**One-liner:** A real-time chat server in Go using WebSockets — named rooms, user presence, message history, and broadcast with gorilla/websocket.

---

## Why This Stands Out

- **Concurrency showcase** — goroutines, channels, sync primitives for real-time fan-out
- **Hub pattern** — central message broker managing client connections per room
- **Named rooms** — join/leave/list with per-room broadcast, not just global chat
- **User presence** — who's online in each room, join/leave notifications
- **Message persistence** — SQLite history, retrieve last N messages on join
- **Clean separation** — transport (WebSocket) decoupled from domain (rooms, messages)
- **Load-testable** — includes simple Go client for concurrent connection testing

---

## Architecture

```
chatterbox/
├── cmd/
│   └── server/
│       └── main.go              # Entry point: wire hub, start HTTP + WS server
├── internal/
│   ├── config/
│   │   └── config.go            # Env config: port, db path, max rooms, max history
│   ├── domain/
│   │   ├── message.go           # Message types (chat, join, leave, system)
│   │   ├── room.go              # Room entity
│   │   └── user.go              # User/client identity
│   ├── hub/
│   │   ├── hub.go               # Central hub: manage rooms, route messages
│   │   ├── hub_test.go
│   │   ├── room.go              # Room: client set, broadcast loop, history
│   │   └── room_test.go
│   ├── client/
│   │   ├── client.go            # WebSocket client: read/write pumps
│   │   └── client_test.go
│   ├── handler/
│   │   ├── ws.go                # WebSocket upgrade handler
│   │   ├── api.go               # REST: list rooms, room info, health
│   │   └── handler_test.go
│   ├── store/
│   │   ├── store.go             # Message store interface
│   │   ├── sqlite.go            # SQLite message persistence
│   │   └── sqlite_test.go
│   └── middleware/
│       ├── logging.go
│       └── cors.go
├── tools/
│   └── loadtest/
│       └── main.go              # Concurrent WebSocket client for load testing
├── static/
│   └── index.html               # Minimal browser chat client (vanilla JS)
├── go.mod
├── go.sum
├── Makefile
├── .env.example
├── .gitignore
├── .golangci.yml
├── LICENSE
└── README.md
```

---

## WebSocket Protocol

### Client → Server Messages

| Type | Payload | Description |
|------|---------|-------------|
| `join` | `{ room: "general" }` | Join a room |
| `leave` | `{ room: "general" }` | Leave a room |
| `chat` | `{ room: "general", text: "hello" }` | Send message to room |

### Server → Client Messages

| Type | Payload | Description |
|------|---------|-------------|
| `chat` | `{ room, user, text, timestamp }` | Chat message |
| `join` | `{ room, user }` | User joined notification |
| `leave` | `{ room, user }` | User left notification |
| `history` | `{ room, messages: [...] }` | Last N messages on join |
| `presence` | `{ room, users: [...] }` | Current room members |
| `error` | `{ message }` | Error (bad JSON, room not found, etc.) |

### REST Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/rooms` | List active rooms with user counts |
| GET | `/api/rooms/:name` | Room details (users, message count) |
| GET | `/health` | Server health |
| GET | `/ws?user=name` | WebSocket upgrade |

---

## Tech Stack

| Component | Choice |
|-----------|--------|
| Language | Go 1.22+ |
| WebSocket | gorilla/websocket |
| HTTP | stdlib net/http |
| Storage | SQLite (message history) |
| Frontend | Single HTML file with vanilla JS |
| Testing | stdlib + gorilla/websocket test helpers |
| Linting | golangci-lint |

---

## Phased Build Plan

### Phase 1: Foundation

**1.1 — Project setup + config**
- `go mod init github.com/devaloi/chatterbox`
- Directory structure, Makefile, deps (gorilla/websocket, mattn/go-sqlite3)
- Env config: PORT, DB_PATH, MAX_ROOMS, MAX_HISTORY

**1.2 — Domain types + message encoding**
- Message struct with Type, Room, User, Text, Timestamp
- JSON marshal/unmarshal
- Message type constants
- Tests: encode/decode round-trips

### Phase 2: Hub + Rooms

**2.1 — Room implementation**
- Room struct: name, clients map, broadcast channel
- `Run()` goroutine: listen on broadcast channel, fan-out to clients
- `Join(client)`, `Leave(client)`, `Broadcast(message)`
- Client set with mutex protection
- Tests: join/leave/broadcast, concurrent access

**2.2 — Hub (room manager)**
- Hub struct: rooms map, register/unregister channels
- `Run()` goroutine: handle room creation, client routing
- `GetOrCreateRoom(name)`, `ListRooms()`, `RoomInfo(name)`
- Auto-delete empty rooms after last client leaves
- Tests: create rooms, route messages, cleanup empty rooms

### Phase 3: WebSocket Client

**3.1 — Client read/write pumps**
- Client struct: hub, conn, room, user, send channel
- `ReadPump()` — read from WS, parse message, route to hub
- `WritePump()` — read from send channel, write to WS
- Ping/pong keepalive (configurable interval)
- Graceful close: drain send channel, close connection
- Tests: message flow, ping/pong, disconnect handling

### Phase 4: HTTP + Persistence

**4.1 — WebSocket handler + REST API**
- WS upgrade handler: extract user from query param, create client, register with hub
- REST: list rooms, room info, health check
- Tests: WS upgrade, REST endpoints

**4.2 — SQLite message store**
- Store interface: `Save(message)`, `History(room, limit) []Message`
- SQLite implementation: messages table (id, room, user, text, type, created_at)
- Load history on room join (last 50 messages)
- Tests: save/retrieve, history limit, room isolation

### Phase 5: Polish

**5.1 — Browser client**
- Single `index.html` with vanilla JS WebSocket client
- Room selector, message display, user list sidebar
- Minimal CSS, no build step

**5.2 — Load test tool**
- `tools/loadtest/main.go` — spawn N concurrent WebSocket clients
- Each client joins a room, sends messages, measures latency
- Report: connections/sec, message latency p50/p95/p99

**5.3 — Integration tests**
- Multiple clients join same room, messages broadcast correctly
- Client disconnect triggers presence update
- History loaded on join

**5.4 — README**
- Badges, install, quick start
- WebSocket protocol reference
- Architecture diagram (hub pattern)
- Screenshots of browser client
- Load test results

---

## Commit Plan

1. `chore: scaffold project with deps and config`
2. `feat: add domain types and message encoding`
3. `feat: add room with broadcast loop`
4. `feat: add hub (room manager)`
5. `feat: add WebSocket client with read/write pumps`
6. `feat: add HTTP handlers and WS upgrade`
7. `feat: add SQLite message persistence`
8. `feat: add browser chat client`
9. `feat: add load test tool`
10. `test: add integration tests`
11. `docs: add README with protocol reference`
