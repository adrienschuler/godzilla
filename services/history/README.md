# History Service

Stores and retrieves chat message history. Built with FastAPI, backed by MongoDB.

## API

All endpoints require the `X-Authenticated-User` header.

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/discussion` | List user's discussions (most recent first) |
| `GET` | `/discussion/{id}/messages` | Get messages with cursor-based pagination |
| `POST` | `/discussion/{id}/messages` | Add messages to a discussion |
| `GET` | `/health` | Health check |

## Examples

```sh
# list discussions
http :8000/discussion X-Authenticated-User:alice

# get messages (with pagination)
http :8000/discussion/{id}/messages X-Authenticated-User:alice limit==50

# add messages
http POST :8000/discussion/1234/messages X-Authenticated-User:alice '[0][text]=hello'
```

## Development

```sh
just test          # run tests
just build         # build docker image
just deploy        # deploy to minikube
just rebuild       # test + build + deploy
just logs          # tail pod logs
```
