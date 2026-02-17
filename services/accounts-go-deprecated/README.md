# Accounts

User registration and session management service in Go (stdlib `net/http`), backed by Redis and MongoDB.

## Endpoints

| Method | Path             | Description                          |
|--------|------------------|--------------------------------------|
| GET    | `/healthz`       | Health check (Redis + MongoDB)       |
| POST   | `/user/register` | Register (JSON: `username`+`password`) |
| POST   | `/user/login`    | Login, sets session cookie           |
| POST   | `/user/logout`   | Logout, clears session               |
| GET    | `/user/me`       | Current user (requires auth header)  |

## Run locally

```sh
go run ./cmd
```

Expects Redis on `localhost:6379` and MongoDB on `localhost:27017` (configurable via env vars).

## Development (Minikube)

```sh
just accounts rebuild
```
