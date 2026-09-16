# push-be

Push backend. Go + Echo v4 + pgx/v5 + PostgreSQL 16.

## Requirements

- Go 1.26.5+
- Docker (PostgreSQL 16 via docker-compose)

## Run

```sh
cp .env.example .env
docker compose up -d postgres
go run ./cmd/server
```

The server applies `migrations/*.sql` at startup and listens on `:8080`.
API base: `http://localhost:8080/api/v1`. `GET /healthz` is unauthenticated.

## Development auth

With `APP_ENV=development` and `DEV_AUTH_TOKEN` set, the dev user
(`DEV_USER_ID`, seeded at startup) is authenticated via:

```
Authorization: Bearer $DEV_AUTH_TOKEN
```

Development auth is never active outside `APP_ENV=development`.

## Test

```sh
go test ./...
```

Tests use in-memory stubs; no running PostgreSQL required.

## Layout

- `cmd/server` — composition root
- `internal/presentation` — Echo handlers, DTOs, middleware
- `internal/domain` — entities, use cases, port interfaces, state machines
- `internal/infra` — pgx repositories, config, crypto, OAuth client
- `migrations` — numbered SQL files applied in order at startup
