# Presence Service

gRPC service for real-time user presence tracking.

## API

```protobuf
// proto/presence.proto
service PresenceService {
  rpc UserConnected(UserRequest) returns (OnlineUsersResponse);
  rpc UserDisconnected(UserRequest) returns (Empty);
  rpc SetTyping(SetTypingRequest) returns (Empty);
  rpc GetOnlineUsers(Empty) returns (OnlineUsersResponse);
  rpc GetTypingUsers(Empty) returns (TypingUsersResponse);
}
```

## Usage

```bash
# Run locally (port 50051 by default)
go run ./cmd

# Run on custom port
PORT=50052 go run ./cmd

# Test
go test ./cmd

# Build Docker image
just build

# Deploy to Kubernetes
just deploy
```

**Environment Variables:**
- `PORT`: gRPC server port (default: 50051)

## Kubernetes

- Service: `presence-svc:50051`
- Port: 50051 (gRPC)
- Health: gRPC health checks

## Integration

Chat service → gRPC → Presence service for online/typing status.

**gRPC Clients:**
- Chat service uses `src/presence-client.js` to connect to this service
- Clients should implement the protobuf interface from `proto/presence.proto`

**Example gRPC Calls:**

```javascript
// Connect user
await presence.userConnected("alice"); // returns { usernames: ["alice", "bob"] }

// Set typing status
await presence.setTyping("alice", true);

// Get online users
const { usernames } = await presence.getOnlineUsers();

// Get typing users
const { usernames: typing } = await presence.getTypingUsers();
```