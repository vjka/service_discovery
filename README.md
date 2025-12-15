This project implements a simple distributed counter system in Go with:

  - Dynamic service discovery
  - Eventually consistent counter replication
  - Deduplication of increments
  - Retry handling with exponential backoff
  - Failure detection via heartbeats
  - Each node maintains its own counter and propagates increments to peers asynchronously.
    
## Architecture

```
service_discovery/
├── go.mod
├── go.sum
├── main.go
├── mocks/
│   └── service_discovery/
│       └── pkg/
│           ├── client/
│           │   └── mock_IClient.go
│           ├── peerStore/
│           │   └── mock_IPeerStore.go
│           └── service/
│               └── mock_IPeerService.go
└── pkg/
    ├── client/
    │   ├── client.go
    │   └── client_test.go
    ├── counter/
    │   └── counter.go
    ├── handler/
    │   ├── handler.go
    │   └── hanlder_test.go
    ├── peerStore/
    │   ├── peerStore.go
    │   └── peer_store_test.go
    └── service/
        ├── service.go
        └── service_test.go

```

| Component     | Responsibility                                                |
| ------------- | --------------------------------------------------------------|
| `PeerService` | Coordinates peer membership, counter updates, and retry logic |
| `PeerStore`   | Tracks active peers and last-seen timestamps                  |
| `Counter`     | Maintains counter value with deduplication                    |
| `Client`      | HTTP client for inter-node communication                      |
| `Handlers`    | HTTP API endpoints                                            |


## Design Decisions
### 1. Service Discovery

  - Nodes join the cluster via ```/nodes/join```
  - Joining node receives a list of known peers
  - Heartbeats (/nodes/heartbeat) update peer liveness
  - Periodic cleanup removes dead peers

#### Why:
  Simple, explicit discovery avoids complex consensus systems.

### 2. Counter & Deduplication

  - Each increment has a globally unique eventID.
  - ```applied := counter.Apply(eventID, 1)```
  - If an eventID was already applied → ignored
  - Prevents duplicate increments during retries or network issues

#### Why:
  Ensures idempotency and correctness under retries.

### 3. Increment Propagation

  - Increment is applied locally first
  - Propagated asynchronously to all peers
  - Failures are queued for retry

  ```go s.sendOrQueue(peer, eventID)```

#### Why:
  Low latency for the caller, eventual consistency for the cluster.

### 4. Retry Handling (Eventual Consistency)

  - Failed propagations are stored in Pending
  - Retry loop runs periodically
  - Exponential backoff per event
  - ```delay := 100ms * 2^(attempt-1)```
  - Max retry delay capped at 10 seconds
  #### Why:
  Handles transient failures and network partitions gracefully.

### 5. Failure Detection

  - Heartbeat every 2 seconds
  - Cleanup every 5 seconds
  - Peer removed if inactive > 6 seconds
  #### Why:
  Keeps peer list accurate without external coordination.

| Endpoint             | Method | Description         |
| -------------------- | ------ | ------------------- |
| `/nodes/join`        | POST   | Join cluster        |
| `/nodes`             | GET    | List peers          |
| `/nodes/heartbeat`   | POST   | Heartbeat           |
| `/counter/increment` | POST   | Increment counter   |
| `/counter/replicate` | POST   | Replicate increment |
| `/counter/count`     | GET    | Get counter value   |



## How to Run
### Start Node 1
```go run main.go --port=8080```

### Start Node 2 and Join Node 1
```go run main.go --port=8081 --peers=localhost:8080```

### Increment Counter
```curl -X POST http://localhost:8080/counter/increment```

### Get Counter Value
```curl http://localhost:8080/counter/count```

### Handling Network Partitions
#### How it Works
1.  Increments applied locally
2.  Failed peer updates queued
3.  Retry loop keeps attempting
4.  Once the network partition heals, pending updates are retried and applied successfully.
5.  Deduplication prevents double counting

### Result:
Cluster eventually converges to the same value.
