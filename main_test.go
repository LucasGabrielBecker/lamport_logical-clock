package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// Test basic LamportClock functionality
func TestLamportClockTick(t *testing.T) {
	clock := NewLamportClock()

	// Initial state should be 0
	if clock.GetTime() != 0 {
		t.Errorf("Expected initial timestamp to be 0, got %d", clock.GetTime())
	}

	// First tick should return 1
	timestamp1 := clock.Tick()
	if timestamp1 != 1 {
		t.Errorf("Expected first tick to return 1, got %d", timestamp1)
	}

	// Second tick should return 2
	timestamp2 := clock.Tick()
	if timestamp2 != 2 {
		t.Errorf("Expected second tick to return 2, got %d", timestamp2)
	}

	// GetTime should return current value
	if clock.GetTime() != 2 {
		t.Errorf("Expected GetTime to return 2, got %d", clock.GetTime())
	}
}

func TestLamportClockUpdate(t *testing.T) {
	clock := NewLamportClock()

	// Test case 1: received timestamp is higher
	// Local: 0, Received: 5 -> should become 6
	newTime := clock.Update(5)
	if newTime != 6 {
		t.Errorf("Expected update with higher timestamp to return 6, got %d", newTime)
	}

	// Test case 2: received timestamp is lower
	// Local: 6, Received: 3 -> should become 7
	newTime = clock.Update(3)
	if newTime != 7 {
		t.Errorf("Expected update with lower timestamp to return 7, got %d", newTime)
	}

	// Test case 3: received timestamp equals local
	// Local: 7, Received: 7 -> should become 8
	newTime = clock.Update(7)
	if newTime != 8 {
		t.Errorf("Expected update with equal timestamp to return 8, got %d", newTime)
	}
}

func TestLamportClockConcurrency(t *testing.T) {
	clock := NewLamportClock()
	numGoroutines := 100
	ticksPerGoroutine := 10

	var wg sync.WaitGroup
	timestamps := make([]int64, numGoroutines*ticksPerGoroutine)
	var mu sync.Mutex
	index := 0

	// Launch multiple goroutines that tick the clock
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < ticksPerGoroutine; j++ {
				ts := clock.Tick()
				mu.Lock()
				timestamps[index] = ts
				index++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Check that all timestamps are unique and positive
	timestampSet := make(map[int64]bool)
	for _, ts := range timestamps {
		if ts <= 0 {
			t.Errorf("Found non-positive timestamp: %d", ts)
		}
		if timestampSet[ts] {
			t.Errorf("Found duplicate timestamp: %d", ts)
		}
		timestampSet[ts] = true
	}

	// Final timestamp should equal total number of ticks
	expectedFinal := int64(numGoroutines * ticksPerGoroutine)
	if clock.GetTime() != expectedFinal {
		t.Errorf("Expected final timestamp to be %d, got %d", expectedFinal, clock.GetTime())
	}
}

func TestServerEventCreation(t *testing.T) {
	server := NewServer()

	// Create an event
	event := server.logEvent("test-1", "Test event")

	if event.ID != "test-1" {
		t.Errorf("Expected event ID to be 'test-1', got '%s'", event.ID)
	}

	if event.Message != "Test event" {
		t.Errorf("Expected event message to be 'Test event', got '%s'", event.Message)
	}

	if event.Timestamp != 1 {
		t.Errorf("Expected first event timestamp to be 1, got %d", event.Timestamp)
	}

	// Create another event
	event2 := server.logEvent("test-2", "Second event")
	if event2.Timestamp != 2 {
		t.Errorf("Expected second event timestamp to be 2, got %d", event2.Timestamp)
	}

	// Check events are stored
	server.mutex.RLock()
	eventCount := len(server.events)
	server.mutex.RUnlock()

	if eventCount != 2 {
		t.Errorf("Expected 2 events to be stored, got %d", eventCount)
	}
}

func TestServerMessageProcessing(t *testing.T) {
	server := NewServer()

	// Create a local event first (timestamp will be 1)
	server.logEvent("local", "Local event")

	// Process message with higher timestamp
	event := server.processMessage(5, "External message")

	// Should update to max(1, 5) + 1 = 6
	if event.Timestamp != 6 {
		t.Errorf("Expected message processing to result in timestamp 6, got %d", event.Timestamp)
	}

	// Process another message with lower timestamp
	event2 := server.processMessage(3, "Another message")

	// Should update to max(6, 3) + 1 = 7
	if event2.Timestamp != 7 {
		t.Errorf("Expected second message processing to result in timestamp 7, got %d", event2.Timestamp)
	}
}

// HTTP Handler Tests

func TestCreateEventHandler(t *testing.T) {
	server := NewServer()

	// Test successful event creation
	req := httptest.NewRequest("POST", "/event?message=test_message", nil)
	w := httptest.NewRecorder()

	server.handleCreateEvent(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}

	var response Event
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Message != "test_message" {
		t.Errorf("Expected message 'Test message', got '%s'", response.Message)
	}

	if response.Timestamp != 1 {
		t.Errorf("Expected timestamp 1, got %d", response.Timestamp)
	}

	// Test with default message
	req2 := httptest.NewRequest("POST", "/event", nil)
	w2 := httptest.NewRecorder()

	server.handleCreateEvent(w2, req2)

	var response2 Event
	json.NewDecoder(w2.Body).Decode(&response2)

	if response2.Message != "Local event" {
		t.Errorf("Expected default message 'Local event', got '%s'", response2.Message)
	}

	// Test wrong method
	req3 := httptest.NewRequest("GET", "/event", nil)
	w3 := httptest.NewRecorder()

	server.handleCreateEvent(w3, req3)

	if w3.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status MethodNotAllowed, got %d", w3.Code)
	}
}

func TestReceiveMessageHandler(t *testing.T) {
	server := NewServer()

	// Test successful message processing
	req := httptest.NewRequest("POST", "/message?timestamp=10&message=external_event", nil)
	w := httptest.NewRecorder()

	server.handleReceiveMessage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}

	var response Event
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Timestamp != 11 {
		t.Errorf("Expected timestamp 11 (max(0,10)+1), got %d", response.Timestamp)
	}

	// Test missing parameters
	req2 := httptest.NewRequest("POST", "/message?timestamp=5", nil)
	w2 := httptest.NewRecorder()

	server.handleReceiveMessage(w2, req2)

	if w2.Code != http.StatusBadRequest {
		t.Errorf("Expected status BadRequest for missing message, got %d", w2.Code)
	}

	// Test invalid timestamp
	req3 := httptest.NewRequest("POST", "/message?timestamp=invalid&message=test", nil)
	w3 := httptest.NewRecorder()

	server.handleReceiveMessage(w3, req3)

	if w3.Code != http.StatusBadRequest {
		t.Errorf("Expected status BadRequest for invalid timestamp, got %d", w3.Code)
	}
}

func TestGetEventsHandler(t *testing.T) {
	server := NewServer()

	// Add some events
	server.logEvent("event1", "First event")
	server.logEvent("event2", "Second event")

	req := httptest.NewRequest("GET", "/events", nil)
	w := httptest.NewRecorder()

	server.handleGetEvents(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	currentTimestamp := response["current_timestamp"].(float64)
	if currentTimestamp != 2 {
		t.Errorf("Expected current timestamp 2, got %f", currentTimestamp)
	}

	eventCount := response["event_count"].(float64)
	if eventCount != 2 {
		t.Errorf("Expected event count 2, got %f", eventCount)
	}

	events := response["events"].([]interface{})
	if len(events) != 2 {
		t.Errorf("Expected 2 events in response, got %d", len(events))
	}
}

func TestGetTimeHandler(t *testing.T) {
	server := NewServer()
	server.clock.Tick() // Make timestamp 1

	req := httptest.NewRequest("GET", "/time", nil)
	w := httptest.NewRecorder()

	server.handleGetTime(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	timestamp := response["lamport_timestamp"].(float64)
	if timestamp != 1 {
		t.Errorf("Expected lamport timestamp 1, got %f", timestamp)
	}

	// Check that wall_time is present
	if _, exists := response["wall_time"]; !exists {
		t.Error("Expected wall_time to be present in response")
	}
}

// Integration test simulating distributed scenario
func TestDistributedScenario(t *testing.T) {
	server := NewServer()

	// Simulate a distributed system scenario
	// Process 1 does some work
	event1 := server.logEvent("p1-1", "Process 1: Start transaction")  // ts: 1
	event2 := server.logEvent("p1-2", "Process 1: Read from database") // ts: 2

	// Process 2 sends a message with timestamp 5
	event3 := server.processMessage(5, "Process 2: Update notification") // ts: 6

	// Process 1 continues
	event4 := server.logEvent("p1-3", "Process 1: Complete transaction") // ts: 7

	// Process 3 sends message with older timestamp
	event5 := server.processMessage(4, "Process 3: Status check") // ts: 8

	// Verify timestamps maintain causality
	timestamps := []int64{event1.Timestamp, event2.Timestamp, event3.Timestamp, event4.Timestamp, event5.Timestamp}
	expected := []int64{1, 2, 6, 7, 8}

	for i, ts := range timestamps {
		if ts != expected[i] {
			t.Errorf("Event %d: expected timestamp %d, got %d", i+1, expected[i], ts)
		}
	}

	// Verify all timestamps are strictly increasing
	for i := 1; i < len(timestamps); i++ {
		if timestamps[i] <= timestamps[i-1] {
			t.Errorf("Timestamps not strictly increasing: %d followed by %d", timestamps[i-1], timestamps[i])
		}
	}

	// Final clock should be 8
	if server.clock.GetTime() != 8 {
		t.Errorf("Expected final clock value 8, got %d", server.clock.GetTime())
	}
}

// Benchmark tests
func BenchmarkLamportClockTick(b *testing.B) {
	clock := NewLamportClock()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clock.Tick()
	}
}

func BenchmarkLamportClockUpdate(b *testing.B) {
	clock := NewLamportClock()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clock.Update(int64(i))
	}
}

func BenchmarkConcurrentTicks(b *testing.B) {
	clock := NewLamportClock()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			clock.Tick()
		}
	})
}

// Test helper to verify Lamport timestamp properties
func TestLamportProperties(t *testing.T) {
	server := NewServer()

	// Property 1: If event A happens before event B in the same process,
	// then timestamp(A) < timestamp(B)
	eventA := server.logEvent("A", "Event A")
	time.Sleep(1 * time.Millisecond) // Ensure different wall clock times
	eventB := server.logEvent("B", "Event B")

	if eventA.Timestamp >= eventB.Timestamp {
		t.Errorf("Causality violation: Event A (%d) should have timestamp < Event B (%d)",
			eventA.Timestamp, eventB.Timestamp)
	}

	// Property 2: If message M is sent with timestamp T, and received causing event R,
	// then T < timestamp(R)
	sentTimestamp := int64(10)
	receivedEvent := server.processMessage(sentTimestamp, "Message M")

	if sentTimestamp >= receivedEvent.Timestamp {
		t.Errorf("Message causality violation: Sent timestamp (%d) should be < received timestamp (%d)",
			sentTimestamp, receivedEvent.Timestamp)
	}
}
