package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// LamportClock represents a Lamport logical clock
type LamportClock struct {
	timestamp int64
	mutex     sync.RWMutex
}

// NewLamportClock creates a new Lamport clock initialized to 0
func NewLamportClock() *LamportClock {
	return &LamportClock{
		timestamp: 0,
	}
}

// Tick increments the logical clock for a local event
func (lc *LamportClock) Tick() int64 {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()

	lc.timestamp++
	return lc.timestamp
}

// Update updates the clock when receiving a message with a timestamp
// This implements the Lamport algorithm: max(local_time, received_time) + 1
func (lc *LamportClock) Update(receivedTimestamp int64) int64 {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()

	if receivedTimestamp > lc.timestamp {
		lc.timestamp = receivedTimestamp
	}
	lc.timestamp++
	return lc.timestamp
}

// GetTime returns the current logical time (read-only)
func (lc *LamportClock) GetTime() int64 {
	lc.mutex.RLock()
	defer lc.mutex.RUnlock()
	return lc.timestamp
}

// Event represents a timestamped event
type Event struct {
	ID        string    `json:"id"`
	Message   string    `json:"message"`
	Timestamp int64     `json:"lamport_timestamp"`
	WallTime  time.Time `json:"wall_time"`
}

// Server holds the Lamport clock and event log
type Server struct {
	clock  *LamportClock
	events []Event
	mutex  sync.RWMutex
}

// NewServer creates a new server with a Lamport clock
func NewServer() *Server {
	return &Server{
		clock:  NewLamportClock(),
		events: make([]Event, 0),
	}
}

// logEvent creates and logs an event with Lamport timestamp
func (s *Server) logEvent(id, message string) Event {
	timestamp := s.clock.Tick()

	event := Event{
		ID:        id,
		Message:   message,
		Timestamp: timestamp,
		WallTime:  time.Now(),
	}

	s.mutex.Lock()
	s.events = append(s.events, event)
	s.mutex.Unlock()

	log.Printf("Event logged: %s (Lamport: %d)", message, timestamp)
	return event
}

// processMessage simulates processing a message from another node
func (s *Server) processMessage(receivedTimestamp int64, message string) Event {
	// Update our clock based on received timestamp
	newTimestamp := s.clock.Update(receivedTimestamp)

	event := Event{
		ID:        fmt.Sprintf("msg-%d", newTimestamp),
		Message:   fmt.Sprintf("Processed: %s", message),
		Timestamp: newTimestamp,
		WallTime:  time.Now(),
	}

	s.mutex.Lock()
	s.events = append(s.events, event)
	s.mutex.Unlock()

	log.Printf("Message processed: %s (Received: %d, New: %d)",
		message, receivedTimestamp, newTimestamp)
	return event
}

// HTTP Handlers

func (s *Server) handleCreateEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	message := r.URL.Query().Get("message")
	if message == "" {
		message = "Local event"
	}

	eventID := fmt.Sprintf("event-%d", time.Now().UnixNano())
	event := s.logEvent(eventID, message)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}

func (s *Server) handleReceiveMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	timestampStr := r.URL.Query().Get("timestamp")
	message := r.URL.Query().Get("message")

	if timestampStr == "" || message == "" {
		http.Error(w, "Missing timestamp or message parameter", http.StatusBadRequest)
		return
	}

	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid timestamp", http.StatusBadRequest)
		return
	}

	event := s.processMessage(timestamp, message)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}

func (s *Server) handleGetEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mutex.RLock()
	events := make([]Event, len(s.events))
	copy(events, s.events)
	s.mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"current_timestamp": s.clock.GetTime(),
		"events":            events,
		"event_count":       len(events),
	})
}

func (s *Server) handleGetTime(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"lamport_timestamp": s.clock.GetTime(),
		"wall_time":         time.Now(),
	})
}

func main() {
	server := NewServer()

	// Set up HTTP routes
	http.HandleFunc("/event", server.handleCreateEvent)
	http.HandleFunc("/message", server.handleReceiveMessage)
	http.HandleFunc("/events", server.handleGetEvents)
	http.HandleFunc("/time", server.handleGetTime)

	// Welcome endpoint
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `Lamport Timestamp Server

Available endpoints:
- POST /event?message=<msg>     : Create a local event
- POST /message?timestamp=<ts>&message=<msg> : Process received message
- GET  /events                  : Get all events with timestamps
- GET  /time                    : Get current Lamport timestamp

Example usage:
curl -X POST "http://localhost:8080/event?message=User login"
curl -X POST "http://localhost:8080/message?timestamp=5&message=External event"
curl http://localhost:8080/events
`)
	})

	// Start server
	port := ":8080"
	log.Printf("Starting Lamport timestamp server on port %s", port)
	log.Printf("Visit http://localhost%s for usage instructions", port)

	// Log initial state
	server.logEvent("init", "Server started")

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
