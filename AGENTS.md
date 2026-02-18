# AGENTS.md

## Project Overview

Godzilla is a deliberately over-engineered microservice playground on Kubernetes. It implements a session-based API gateway with real-time chat, built across multiple languages as a learning exercise.

## Architecture

```
Client → Nginx/OpenResty (gateway, :80) → Ruby/Sinatra (accounts, :8081)
                                        → Node.js/Fastify (chat, :3000) → Go/gRPC (presence, :50051)
Backing stores: Redis (:6379), MongoDB (:27017)
```

Gateway authenticates via Lua session lookup in Redis. All services run in a single k8s namespace (`godzilla`).

## Services

| Service | Language | Path | Purpose |
|---------|----------|------|---------|
| gateway | Lua/OpenResty | `services/gateway/` | Reverse proxy, auth, rate limiting |
| accounts | Ruby/Sinatra | `services/accounts/` | User CRUD, session management |
| chat | Node.js/Fastify | `services/chat/` | Socket.io real-time chat |
| presence | Go/gRPC | `services/presence/` | Online/typing status tracking |

## Key Files

- `justfile` — Top-level orchestration (build, deploy, test, clean)
- `proto/presence.proto` — gRPC contract for presence service
- `k8s/*.yaml` — Kubernetes manifests (one per service + redis/mongodb)
- `tests/test_auth_flow.py` — Integration tests (pytest, managed via uv)

## Build & Test

```bash
just deploy            # Build all images + deploy to minikube
just test              # Run unit + integration tests
just unit-test         # Unit tests only (per-service)
just integration-test  # pytest integration tests
just port-forward      # Forward localhost:8080 → gateway
just status            # Show k8s pods/services
just clean             # Delete namespace
```

Each service has its own `justfile` with `build`, `deploy`, and `test` targets, invoked via `just <service> <target>`.

## Conventions

- Each service is self-contained with its own Dockerfile, justfile, README, and tests
- Proto definitions live in `proto/`, generated Go code in `services/presence/gen/`
- Integration tests use `uv` for Python dependency management
- No `.env` files in repo — secrets are k8s-managed
