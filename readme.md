# Lamport Timestamp Implementation in Go

A thread-safe implementation of Leslie Lamport's logical clock algorithm for distributed systems, featuring an HTTP API for testing and demonstration.

## What are Lamport Timestamps?

Lamport timestamps are a fundamental concept in distributed systems that solve the problem of **event ordering** when physical clocks cannot be trusted. Introduced by Leslie Lamport in 1978, they provide a way to establish a partial ordering of events across distributed processes without relying on synchronized physical clocks.

### The Core Problem

In distributed systems, determining the order of events is challenging because:
- Physical clocks on different machines drift and are never perfectly synchronized
- Network delays are unpredictable
- Events may appear to happen "simultaneously" but still need to be ordered

### The Solution

Lamport timestamps use **logical clocks** - simple counters that follow two rules:

1. **Local Event**: When a process performs a local event, increment its logical clock
2. **Message Passing**: When receiving a message with timestamp T, set local clock to `max(local_clock, T) + 1`

This ensures that if event A causally precedes event B, then `timestamp(A) < timestamp(B)`.

## Why Lamport Timestamps Matter in Computer Science

### 1. **Foundation of Distributed Systems Theory**
- Established the concept of **logical time** vs physical time
- Introduced the "happens-before" relationship (→)
- Laid groundwork for understanding causality in concurrent systems

### 2. **Practical Applications**
- **Database Systems**: Ensuring consistent transaction ordering across replicas
- **Version Control**: Git uses similar concepts for commit ordering
- **Distributed Consensus**: Building blocks for algorithms like Raft and Paxos
- **Event Sourcing**: Ordering events in distributed event stores
- **Blockchain**: Establishing transaction order without central authority

### 3. **Academic Significance**
- One of the most cited papers in computer science
- Essential for understanding distributed systems correctness
- Bridge between theoretical computer science and practical systems

## Use Cases

### 1. **Distributed Database Replication**
```
Node A: UPDATE user SET balance=100 (timestamp: 5)
Node B: UPDATE user SET balance=200 (timestamp: 3)
Result: Apply Node A's update last (higher timestamp)
```

### 2. **Microservices Event Ordering**
```
Order Service: "Order created" (timestamp: 10)
Payment Service: "Payment processed" (timestamp: 15)
Shipping Service: "Package shipped" (timestamp: 16)
```

### 3. **Collaborative Editing Systems**
- Ensure document edits are applied in consistent order across all clients
- Resolve conflicts when multiple users edit simultaneously

### 4. **Distributed Logging and Monitoring**
- Order log entries from multiple services for debugging
- Maintain causal relationships between related events

## Features of This Implementation

- ✅ **Thread-safe**: Safe for concurrent access using mutexes
- ✅ **HTTP API**: Easy testing and integration
- ✅ **Event logging**: Track all events with both logical and wall-clock time
- ✅ **Message simulation**: Test distributed scenarios on single instance
- ✅ **Production-ready**: Proper error handling and logging

## Quick Start

```bash
# Start the server
go run main.go

# Create local events
curl -X POST "http://localhost:8080/event?message=User login"

# Simulate receiving external message
curl -X POST "http://localhost:8080/message?timestamp=10&message=External event"

# View all events with timestamps
curl http://localhost:8080/events
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/event?message=<msg>` | Create a local event |
| `POST` | `/message?timestamp=<ts>&message=<msg>` | Process received message |
| `GET` | `/events` | List all events with timestamps |
| `GET` | `/time` | Get current Lamport timestamp |

## Example Output

```json
{
  "current_timestamp": 15,
  "events": [
    {
      "id": "init",
      "message": "Server started",
      "lamport_timestamp": 1,
      "wall_time": "2024-01-01T10:00:00Z"
    },
    {
      "id": "event-123",
      "message": "User login",
      "lamport_timestamp": 2,
      "wall_time": "2024-01-01T10:01:00Z"
    },
    {
      "id": "msg-11",
      "message": "Processed: External event",
      "lamport_timestamp": 11,
      "wall_time": "2024-01-01T10:02:00Z"
    }
  ]
}
```

## Educational Value

This implementation demonstrates:

1. **Logical vs Physical Time**: How systems can order events without synchronized clocks
2. **Causality**: Understanding which events could have influenced others
3. **Concurrent Programming**: Thread-safe design patterns in Go
4. **Distributed Systems Concepts**: Foundation for more complex algorithms

## Further Reading

- **Original Paper**: "Time, Clocks, and the Ordering of Events in a Distributed System" by Leslie Lamport (1978)
- **Vector Clocks**: Extension that captures more precise causality information
- **Distributed Systems**: Concepts like consensus, CAP theorem, and eventual consistency

## Contributing

This is an educational implementation. Consider extending it with:
- Vector clocks for better causality tracking
- Persistence layer for event storage
- Multiple node simulation
- Performance benchmarks

---

*"The concept of one event happening before another in a distributed system is examined, and is shown to define a partial ordering of the events."* - Leslie Lamport