# Godzilla - A Microservice Playground on Kubernetes

A deliberately over-engineered session-based API gateway on Kubernetes, built with OpenResty (Lua), Ruby, Node.js, Go gRPC, MongoDB and Redis. This project exists primarily as a learning exercise to explore microservice patterns, container orchestration, and multi-language service integration. Built with AI assistance.

```ascii
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣠⣤⡀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣠⣶⣿⣿⣿⡟⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣾⣿⣿⣿⣿⣿⣀⣀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠠⣦⣄⣤⣤⣶⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠇⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⣠⣤⣠⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡟⠁⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⣀⣀⣙⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡟⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢈⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠁⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢠⣤⣄⣀⣻⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣏⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠐⠿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣤⣤⣼⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡟⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣧⡀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣬⣝⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣄⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣬⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠋⠻⢿⣿⣿⣷⣄⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣷⣄⠀⠈⢻⣿⣿⡄⠀
⠀⠀⠀⠀⠀⠀⠀⠀⣀⣤⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣷⡤⣼⣿⣿⣷⠀
⠀⠀⠀⠀⠀⢀⣴⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣆⠉⠉⠁⠀
⠀⠀⠀⢀⣾⣿⣿⣿⣍⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣧⠀⠀⠀
⠀⠀⣴⣿⣿⣿⣿⣼⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡆⠀⠀
⢀⣠⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡿⠟⠛⠛⣻⣿⣿⣿⣿⣿⣿⣿⠇⠀⠀
⠸⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠏⠀⠀⠀⠀⠀⢹⣿⣿⣿⣿⣿⣿⡟⠀⠀⠀
⠘⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡿⠏⠁⠀⠀⠀⠀⠀⠀⠈⣿⣿⣿⣿⣿⣿⡇⠀⠀⠀
⠀⠻⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡿⠟⠋⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣿⣿⣿⣿⣿⣿⣧⣄⠀⠀
⠀⠀⠈⠻⢿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣿⣿⣿⣿⣿⣿⣿⣿⣷⡄
⠀⠀⠀⠀⠀⠀⠀⠸⣿⣿⣿⣿⣿⣿⣿⡿⠆⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠉⠉⠉⠉⠉⠉⠉⠉⠁
```

## Architecture

```mermaid
graph TD
    Client([Client]) -->|HTTP / WebSocket| Gateway

    subgraph K8s Namespace: godzilla
        Gateway["Nginx / OpenResty<br>:80 (NodePort 30009)"]

        Gateway -->|/user/register<br>/user/login<br>/user/logout| UserAPI["User API (Ruby/Sinatra)<br>:8081"]
        Gateway -->|/socket.io/| Chat["Chat Service (Node.js/Fastify)<br>:3000"]
        Gateway -.->|Lua auth: session lookup| Redis

        UserAPI --> MongoDB["MongoDB<br>:27017"]
        UserAPI --> Redis["Redis<br>:6379"]
        Chat -->|gRPC| Presence["Presence Service (Go/gRPC)<br>:50051"]
    end

    style Gateway fill:#2d6a4f,color:#fff
    style UserAPI fill:#1b4332,color:#fff
    style Chat fill:#1b4332,color:#fff
    style Presence fill:#0d6efd,color:#fff
    style MongoDB fill:#6c757d,color:#fff
    style Redis fill:#6c757d,color:#fff
```

**Nginx/OpenResty** reverse-proxies all traffic and authenticates protected endpoints by checking sessions directly in Redis via Lua.

**Ruby/Sinatra** handles user CRUD and session management, storing users in MongoDB and sessions in Redis.

**Node.js/Fastify** serves real-time WebSocket chat via Socket.io.

## Auth Flow

```mermaid
sequenceDiagram
    participant C as Client
    participant N as Nginx Gateway
    participant U as User API
    participant M as MongoDB
    participant R as Redis

    C->>N: POST /user/login (JSON)
    N->>U: Proxy request
    U->>M: Find user by username
    U->>U: Verify password (bcrypt)
    U->>R: Store session (24h TTL)
    U-->>C: Set httpOnly session cookie

    C->>N: POST /user/logout
    N->>U: Proxy request
    U->>R: Delete session
    U-->>C: 200 Logged out
```

## Prerequisites

- [Minikube](https://minikube.sigs.k8s.io/docs/start/) + [kubectl](https://kubernetes.io/docs/tasks/tools/) + [Docker](https://docs.docker.com/get-docker/)
- [HTTPie](https://httpie.io/) (optional, for manual testing)
- [uv](https://docs.astral.sh/uv/) (for integration tests)

## Quick Start

```bash
minikube start
just deploy        # Build images + deploy all services to 'godzilla' namespace
just port-forward  # Forward localhost:8080 → nginx gateway
```

## API

All endpoints go through the Nginx gateway at `localhost:8080`.

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/user/register` | No | Create a user (JSON `username` + `password`) |
| POST | `/user/login` | No | Authenticate, receive session cookie (rate-limited: 10 req/s) |
| POST | `/user/logout` | Session | Delete session, clear cookie |
| WS | `/socket.io/` | Session | Real-time chat (Socket.io) |

### Example

```bash
# Register + login
http POST :8080/user/register username=alice password=secret123
http --session=/tmp/alice.json POST :8080/user/login username=alice password=secret123

# Logout
http --session=/tmp/alice.json POST :8080/user/logout
```

### WebSocket Events
[See](https://github.com/adrienschuler/godzilla/tree/main/services/chat#socket-events)

### CLI Client
[See](https://github.com/adrienschuler/godzilla/tree/main/services/chat#cli-chat-client)

## Project Structure

```text
services/
  accounts/              # Ruby/Sinatra — user CRUD, session auth, bcrypt passwords
  gateway/               # OpenResty — reverse proxy, auth_request, rate limiting
  chat/                  # Node.js/Fastify — Socket.io real-time chat
  presence/              # Go/gRPC — User presence tracking (online/typing)
k8s/
  redis.yaml             # Redis deployment + ClusterIP service
  accounts.yaml          # Accounts deployment + ClusterIP service
  gateway.yaml           # Gateway deployment + NodePort service (30009)
  chat.yaml              # Chat deployment + ClusterIP service
  presence.yaml          # Presence deployment + ClusterIP service
tests/                   # pytest integration tests (register → login → access → logout)
justfile                 # Orchestration (see below)
```

## Just Targets

| Target | Description |
|--------|-------------|
| `just deploy` | Build all images and deploy everything |
| `just test` | Run integration tests |
| `just port-forward` | Forward `localhost:8080` → gateway |
| `just status` | Show pods and services |
| `just gateway rebuild` | Rebuild + redeploy gateway |
| `just accounts rebuild` | Rebuild + redeploy accounts service |
| `just chat rebuild` | Rebuild + redeploy chat service |
| `just clean` | Delete the `godzilla` namespace |
