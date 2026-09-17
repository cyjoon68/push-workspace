# push-be

Push backend. Python 3.13 + FastAPI + asyncpg + LangGraph + PostgreSQL 16.

## Requirements

- Python 3.13+
- Docker (PostgreSQL 16 via docker-compose)

## Run

```sh
cp .env.example .env
docker compose up -d postgres
pip install -e .
uvicorn app.main:app --host 0.0.0.0 --port 8080
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
pytest
```

Tests use in-memory stubs; no running PostgreSQL required.

## Layout

- `app/main.py` — composition root (app factory, lifespan, wiring)
- `app/presentation` — FastAPI routers, request schemas, middleware
- `app/domain` — entities, services, errors, state machines
- `app/graph` — LangGraph chat workflow with Postgres checkpointing
- `app/infrastructure` — asyncpg stores, crypto, OAuth/AI/Stripe/Google clients
- `migrations` — numbered SQL files applied in order at startup
