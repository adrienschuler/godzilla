# Gateway

OpenResty-based API gateway that handles route protection and reverse proxying to backend services.

## Architecture

The gateway sits in front of all backend services and provides:

- **Route protection** — Protected routes (`/socket.io/`) validate sessions directly in Redis via Lua; the authenticated username is forwarded via `X-Authenticated-User` header
- **Reverse proxying** — Routes requests to `accounts` and `chat` upstream services
- **Rate limiting** — Login endpoint is rate-limited to 10 requests/seconds per IP
- **JSON error responses** — All nginx-generated errors return structured JSON

## Routes

| Route | Auth | Backend | Description |
|---|---|---|---|
| `POST /user/login` | No | accounts | Authenticate and create session (rate-limited) |
| `POST /user/logout` | No | accounts | Destroy session |
| `/user/register` | No | accounts | Create new user |
| `/socket.io/` | Yes | chat | WebSocket (Socket.io) |
| `/healthz` | No | Lua (inline) | Health check |

## Files

- `nginx.conf` — Server config, upstream definitions, route declarations, and auth subrequest
- `gateway.lua` — Lua utility module (Redis connection helpers, JSON response helpers)
- `Dockerfile` — Builds on `openresty/openresty:alpine`, installs `lua-resty-session`

## Development

Config and Lua code are baked into the Docker image. After any change:

```sh
just gateway rebuild
```
