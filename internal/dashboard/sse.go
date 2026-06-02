package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// SSEEvent represents a server-sent event
type SSEEvent struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// SSEBroker manages SSE client connections and broadcasts events
type SSEBroker struct {
	clients   map[chan []byte]struct{}
	mu        sync.RWMutex
	subscribe chan chan []byte
	unsubscribe chan chan []byte
	broadcast chan []byte
}

// NewSSEBroker creates a new SSE broker
func NewSSEBroker() *SSEBroker {
	b := &SSEBroker{
		clients:     make(map[chan []byte]struct{}),
		subscribe:   make(chan chan []byte, 10),
		unsubscribe: make(chan chan []byte, 10),
		broadcast:   make(chan []byte, 256),
	}
	go b.run()
	return b
}

// run is the main event loop for the broker
func (b *SSEBroker) run() {
	// Heartbeat to keep connections alive
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case client := <-b.subscribe:
			b.mu.Lock()
			b.clients[client] = struct{}{}
			b.mu.Unlock()

		case client := <-b.unsubscribe:
			b.mu.Lock()
			if _, ok := b.clients[client]; ok {
				delete(b.clients, client)
				close(client)
			}
			b.mu.Unlock()

		case msg := <-b.broadcast:
			b.mu.RLock()
			for client := range b.clients {
				select {
				case client <- msg:
				default:
					// Client is slow, skip
				}
			}
			b.mu.RUnlock()

		case <-ticker.C:
			// Send heartbeat
			b.sendRaw([]byte(": heartbeat\n\n"))
		}
	}
}

// Publish sends an event to all connected clients
func (b *SSEBroker) Publish(eventType string, data interface{}) {
	event := SSEEvent{Type: eventType, Data: data}
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}

	msg := fmt.Sprintf("data: %s\n\n", payload)
	b.broadcast <- []byte(msg)
}

// sendRaw broadcasts raw bytes (for heartbeats)
func (b *SSEBroker) sendRaw(data []byte) {
	b.mu.RLock()
	for client := range b.clients {
		select {
		case client <- data:
		default:
		}
	}
	b.mu.RUnlock()
}

// ClientCount returns the number of connected SSE clients
func (b *SSEBroker) ClientCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.clients)
}

// ServeHTTP handles SSE connections from the dashboard
// GET /events
func (b *SSEBroker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check for SSE support
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering

	// Register client
	client := make(chan []byte, 32)
	b.subscribe <- client

	// Send initial connection event
	fmt.Fprintf(w, "data: {\"type\":\"connected\",\"data\":{}}\n\n")
	flusher.Flush()

	// Cleanup on disconnect
	defer func() {
		b.unsubscribe <- client
	}()

	// Stream events
	for {
		select {
		case msg, ok := <-client:
			if !ok {
				return
			}
			w.Write(msg)
			flusher.Flush()

		case <-r.Context().Done():
			return
		}
	}
}

// ─── Event Types ───────────────────────────────────────────────────────────

// RequestEvent is sent when a request completes
type RequestEvent struct {
	ID           int64   `json:"id"`
	Provider     string  `json:"provider"`
	ModelAsked   string  `json:"model_asked"`
	ModelUsed    string  `json:"model_used"`
	Complexity   string  `json:"complexity"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	CostUSD      float64 `json:"cost_usd"`
	LatencyMS    int64   `json:"latency_ms"`
	Status       int     `json:"status"`
	Timestamp    string  `json:"timestamp"`
}

// StatsEvent is sent periodically with updated stats
type StatsEvent struct {
	TotalRequests int     `json:"total_requests"`
	TotalCostUSD  float64 `json:"total_cost_usd"`
	TotalTokens   int     `json:"total_tokens"`
	ForecastUSD   float64 `json:"forecast_usd"`
	ActiveClients int     `json:"active_clients"`
}

// ProviderStatusEvent is sent when provider health changes
type ProviderStatusEvent struct {
	Provider string `json:"provider"`
	Healthy  bool   `json:"healthy"`
	Latency  int64  `json:"latency_ms"`
}
