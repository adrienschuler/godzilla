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
# Run locally
go run ./cmd

# Test
go test ./cmd

# Build Docker image
just build
```

## Kubernetes

- Service: `presence-svc:50051`
- Port: 50051 (gRPC)
- Health: gRPC health checks

## Integration

Chat service → gRPC → Presence service for online/typing status.