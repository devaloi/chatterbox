package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	url := flag.String("url", "ws://localhost:8080/ws", "WebSocket server URL")
	clients := flag.Int("clients", 10, "Number of concurrent clients")
	room := flag.String("room", "loadtest", "Room to join")
	messages := flag.Int("messages", 10, "Messages per client")
	flag.Parse()

	log.Printf("Load test: %d clients, %d messages each, room=%s", *clients, *messages, *room)

	var (
		connected  int64
		sent       int64
		received   int64
		errors     int64
		latencies  []time.Duration
		latencyMu  sync.Mutex
		wg         sync.WaitGroup
	)

	start := time.Now()

	for i := 0; i < *clients; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			user := fmt.Sprintf("user_%d", id)
			wsURL := fmt.Sprintf("%s?user=%s", *url, user)
			conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if err != nil {
				atomic.AddInt64(&errors, 1)
				log.Printf("client %d: dial error: %v", id, err)
				return
			}
			defer conn.Close()
			atomic.AddInt64(&connected, 1)

			// Read goroutine.
			done := make(chan struct{})
			go func() {
				defer close(done)
				for {
					_, _, err := conn.ReadMessage()
					if err != nil {
						return
					}
					atomic.AddInt64(&received, 1)
				}
			}()

			// Join room.
			joinMsg, _ := json.Marshal(map[string]string{"type": "join", "room": *room})
			conn.WriteMessage(websocket.TextMessage, joinMsg)
			time.Sleep(100 * time.Millisecond)

			// Send messages.
			for j := 0; j < *messages; j++ {
				sendTime := time.Now()
				chatMsg, _ := json.Marshal(map[string]string{
					"type": "chat",
					"room": *room,
					"text": fmt.Sprintf("msg %d from %s", j, user),
				})
				if err := conn.WriteMessage(websocket.TextMessage, chatMsg); err != nil {
					atomic.AddInt64(&errors, 1)
					return
				}
				atomic.AddInt64(&sent, 1)
				lat := time.Since(sendTime)
				latencyMu.Lock()
				latencies = append(latencies, lat)
				latencyMu.Unlock()
				time.Sleep(10 * time.Millisecond)
			}

			// Wait a bit for remaining messages.
			time.Sleep(500 * time.Millisecond)
			conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	// Calculate percentiles.
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	fmt.Println("\n=== Load Test Results ===")
	fmt.Printf("Duration:    %s\n", elapsed.Round(time.Millisecond))
	fmt.Printf("Clients:     %d connected\n", connected)
	fmt.Printf("Sent:        %d messages\n", sent)
	fmt.Printf("Received:    %d messages\n", received)
	fmt.Printf("Errors:      %d\n", errors)
	if len(latencies) > 0 {
		fmt.Printf("Latency p50: %s\n", percentile(latencies, 50))
		fmt.Printf("Latency p95: %s\n", percentile(latencies, 95))
		fmt.Printf("Latency p99: %s\n", percentile(latencies, 99))
	}
	fmt.Printf("Throughput:  %.0f msgs/sec\n", float64(sent)/elapsed.Seconds())
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(p/100*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
