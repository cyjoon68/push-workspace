# push-be FastAPI + LangGraph rewrite spec

Task: rewrite the push-be backend from Go to FastAPI (Python). Branch `feat/fastapi-rewrite` already exists off develop — `git switch feat/fastapi-rewrite` first. All work inside `push-be/`.

## Contract

`docs/api.md` is the source of truth. Every route, DTO field, error code, state transition, operation type, and status code must match exactly. `docs/prd.md` is product context. The old Go code in `push-be/` is reference for behavior details — read it lazily, only the files you need.

## Scope

- Replace push-be Go implementation with a Python FastAPI service at the same path `push-be/`.
- Same API surface: all REST routes, request/response JSON, cursor pagination, idempotency keys, revision-conflict semantics, 202 async Operations, auth (`DEV_AUTH_TOKEN` dev mode + OAuth flows), evidence/documents/conversations/operations/interviews/calendar/jobs/applications/projects endpoints, AI/BYOK usage accounting, billing/Stripe webhooks, Google integrations.
- Reuse `push-be/migrations/*.sql` unchanged (PostgreSQL). SQLAlchemy async or asyncpg + raw migrations runner matching existing files.
- Layered structure: presentation (routers/schemas), domain (entities/use cases/ports), infrastructure (repositories, external clients). Dependencies point inward.
- Arrow-style compact code, no comments.

## LangGraph (the point of this rewrite)

- The AI chat workflow (`PostMessage` in `internal/domain/usecase/conversation.go`) becomes a LangGraph `StateGraph` (Python `langgraph` package):
  - Node: resolve message context (document version attachment, evidence attachments).
  - Node: generate assistant reply via the AI gate (BYOK key decryption or managed provider).
  - Node: persist user+assistant messages, Operation, AiUsage.
  - Checkpointer so interrupted conversations resume.
- Keep stub fallback when no AI options (old `stubReply` behavior).
- AI completions flow through the same credential gate: BYOK AES-encrypted keys, MANAGED mode, provider/model routing, usage recording with micro-credit cost.

## Verify

- Port the Go test suite to pytest (domain unit tests + async API tests). Local PostgreSQL via `push-be/docker-compose.yml` db service.
- `docker compose config --quiet` and `docker compose build` must pass.
- Update `push-be/README.md` and `push-be/.env.example` for the Python stack.

## Commits

Conventional Commits, lowercase type + required scope (e.g., `refactor(push-be): ...`). Commit by work unit on `feat/fastapi-rewrite`. Do NOT push, do NOT open a PR. Do NOT delete Go files until the FastAPI service passes its tests — then remove Go sources in a final cleanup commit.
