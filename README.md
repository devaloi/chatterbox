# 💬 chatterbox

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Build](https://img.shields.io/badge/build-passing-brightgreen)]()

A real-time WebSocket chat server in Go — named rooms, user presence, message history, and broadcast with gorilla/websocket.

## Features

- **Named rooms** — create, join, leave with per-room broadcast
- **User presence** — who's online in each room, join/leave notifications
- **Message history** — SQLite persistence, last 50 messages on join
- **Hub pattern** — central message broker with goroutine-per-room broadcast
- **REST API** — list rooms, room details, health check
- **Browser client** — vanilla JS, no build step
- **Load test tool** — concurrent WebSocket stress testing

## Architecture

```
┌────────────┐     ┌─────────┐     ┌──────────────┐
│  Browser   │────▶│  HTTP   │────▶│   Handler    │
│  Client    │◀────│ Server  │◀────│  (WS + API)  │
└────────────┘     └─────────┘     └──────┬───────┘
                                          │
                                   ┌──────▼───────┐
                                   │     Hub      │
                                   │  (goroutine) │
                                   └──┬───┬───┬───┘
                                      │   │   │
                               ┌──────▼┐ ┌▼─┐ ┌▼──────┐
                               │ Room  │ │..│ │ Room   │
                               │  #1   │ │  │ │  #N   │
                               └───┬───┘ └──┘ └───┬───┘
                                   │               │
                              ┌────▼────┐    ┌────▼────┐
                              │ Clients │    │ Clients │
                              └─────────┘    └─────────┘
                                      │
                               ┌──────▼───────┐
                               │   SQLite     │
                               │   Store      │
                               └──────────────┘
```

Each **Client** has `ReadPump` and `WritePump` goroutines. The **Hub** goroutine routes register/unregister/message requests. Each **Room** has its own broadcast goroutine for fan-out.

## Quick Start

```bash
# Clone and build
git clone https://github.com/devaloi/chatterbox.git
cd chatterbox
make build

# Run (defaults: port 8080, SQLite chatterbox.db)
./bin/chatterbox

# Or with environment variables
PORT=3000 DB_PATH=chat.db MAX_ROOMS=50 ./bin/chatterbox
```

Open [http://localhost:8080](http://localhost:8080) in your browser.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `DB_PATH` | `chatterbox.db` | SQLite database path |
| `MAX_ROOMS` | `100` | Maximum concurrent rooms |
| `MAX_HISTORY` | `50` | Messages loaded on room join |

## WebSocket Protocol

### Connect

```
GET /ws?user=alice → 101 Switching Protocols
```

### Client → Server

```json
// Join a room
{"type": "join", "room": "general"}

// Send a message
{"type": "chat", "room": "general", "text": "Hello!"}

// Leave a room
{"type": "leave", "room": "general"}
```

### Server → Client

```json
// Chat message
{"type": "chat", "room": "general", "user": "alice", "text": "Hello!", "timestamp": "2026-01-15T10:30:00Z"}

// User joined
{"type": "join", "room": "general", "user": "bob"}

// User left
{"type": "leave", "room": "general", "user": "bob"}

// Message history (on join)
{"type": "history", "room": "general", "messages": [...]}

// Room presence
{"type": "presence", "room": "general", "users": ["alice", "bob"]}

// Error
{"type": "error", "message": "room not found"}
```

## REST API

```bash
# Health check
curl http://localhost:8080/health
# {"status":"ok"}

# List rooms
curl http://localhost:8080/api/rooms
# [{"name":"general","user_count":3}]

# Room details
curl http://localhost:8080/api/rooms/general
# {"name":"general","user_count":3}
```

## Testing with wscat

```bash
# Install wscat
npm install -g wscat

# Connect
wscat -c "ws://localhost:8080/ws?user=alice"

# Join a room
> {"type":"join","room":"general"}

# Send a message
> {"type":"chat","room":"general","text":"Hello from wscat!"}
```

## Development

```bash
make build    # Build binary to bin/chatterbox
make test     # Run tests with race detector
make vet      # Run go vet
make lint     # Run golangci-lint
make cover    # Generate coverage report
make clean    # Remove build artifacts
make run      # Build and run
```

## Load Testing

```bash
# Build and run the server
make run &

# Run load test (10 clients, 10 messages each)
go run tools/loadtest/main.go -clients 10 -messages 10

# Custom parameters
go run tools/loadtest/main.go \
  -url ws://localhost:8080/ws \
  -clients 100 \
  -messages 50 \
  -room loadtest
```

## Project Structure

```
chatterbox/
├── cmd/server/main.go          # Entry point
├── internal/
│   ├── config/                 # Environment configuration
│   ├── domain/                 # Message types, room, user
│   ├── hub/                    # Central hub + room management
│   ├── client/                 # WebSocket client (read/write pumps)
│   ├── handler/                # WS upgrade + REST API handlers
│   ├── store/                  # Message persistence (SQLite)
│   ├── middleware/              # Logging + CORS
│   └── integration/            # Integration tests
├── tools/loadtest/             # WebSocket load test tool
├── static/index.html           # Browser chat client
├── Makefile                    # Build/test/lint commands
└── docs/                       # Project specification
```

## License

[MIT](LICENSE) © 2026 Jason DeAloia
