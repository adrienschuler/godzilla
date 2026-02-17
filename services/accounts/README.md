# Accounts Service

Sinatra-based HTTP API for user registration, authentication, and session management.

## Stack

- **Sinatra** — HTTP routing
- **MongoDB** — user storage (unique username index)
- **Redis** — session storage (24h TTL)
- **BCrypt** — password hashing (cost 12)
- **Puma** — production web server

## Routes

| Method | Path             | Description                              |
|--------|------------------|------------------------------------------|
| GET    | `/healthz`       | Health check (pings Redis + MongoDB)     |
| POST   | `/user/register` | Register a new user                      |
| POST   | `/user/login`    | Login, sets `auth_token` session cookie  |
| POST   | `/user/logout`   | Logout, clears session                   |

## Environment Variables

| Variable                      | Default                     |
|-------------------------------|-----------------------------|
| `PORT`                        | `8081`                      |
| `MONGO_URI`                   | `mongodb://localhost:27017` |
| `MONGO_DB`                    | `godzilla`                      |
| `REDIS_SERVICE_SERVICE_HOST`  | `127.0.0.1`                |
| `REDIS_SERVICE_SERVICE_PORT`  | `6379`                      |

## Prerequisites

The accounts service requires a modern Ruby (4.0+). macOS ships with an older system Ruby that cannot compile native extensions like `bcrypt`.

Install Ruby via Homebrew:

```sh
brew install ruby
```

Then add it to your PATH (or add to `~/.zshrc` for persistence):

```sh
export PATH="/opt/homebrew/opt/ruby/bin:$PATH"
```

## Run locally

```sh
bundle config set --local path 'vendor/bundle'
bundle install
bundle exec puma -p 8081
```

## Build & deploy (Minikube)

```sh
just accounts build
just accounts deploy
```
