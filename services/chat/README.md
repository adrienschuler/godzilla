# Chat

Real-time WebSocket chat server built with Fastify and Socket.io. Authenticated users can send and receive messages broadcast to all connected clients.

## How It Works

1. Client connects via Socket.io at `/socket.io/` with a username (from `auth.username` or `X-Authenticated-User` header)
2. Server emits a `welcome` event to the newly connected client
3. Incoming `message` events (with a `{ text }` payload) are broadcast to all other connected users with sender info and timestamp
4. Unauthenticated connections are rejected

## Endpoints

| Endpoint | Protocol | Description |
|---|---|---|
| `/socket.io/` | WebSocket | Real-time chat |
| `/health` | HTTP GET | Returns `{ status: 'ok', service: 'chat' }` |

## Socket Events

| Event | Direction | Payload |
|---|---|---|
| `welcome` | Server -> Client | `{ message, timestamp }` |
| `message` | Client -> Server | `{ text }` |
| `message` | Server -> Client | `{ from, data: { text }, timestamp }` |

## Files

- `src/server.js` — Fastify server with Socket.io integration, auth middleware, and message broadcasting
- `src/session.js` — Parses session files to extract auth cookies and usernames from JWTs
- `bin/chat-cli.js` — Interactive CLI chat client that authenticates via a saved session file
- `test/session.test.js` — Tests for session parsing
- `Dockerfile` — Builds on `node:20-alpine`, production dependencies only

## Development

```sh
# Run tests
npm test

# Lint
npm run lint

# Format
npm run format

# Rebuild and deploy to Minikube
just rebuild
```

## CLI Chat Client

Connect to the chat server from the terminal using a saved session file:

```sh
npm run chat-cli -- <session-file>
```

Set `SERVER_URL` to override the default endpoint (`http://127.0.0.1:8080`).
