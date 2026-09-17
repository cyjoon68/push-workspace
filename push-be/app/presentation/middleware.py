from __future__ import annotations

import hashlib
import json
import re
from datetime import datetime, timezone
from uuid import UUID, uuid4

from starlette.types import Message, Receive, Scope, Send

from app.domain.entities import IdempotencyRecord
from app.domain.errors import DomainError
from app.presentation.errors import error_body

_PUBLIC_EXACT = {
    "/api/v1/healthz",
    "/api/v1/auth/exchange",
    "/api/v1/auth/refresh",
    "/api/v1/auth/logout",
    "/api/v1/integrations/google/callback",
    "/api/v1/billing/webhook",
}
_PUBLIC_PATTERN = re.compile(r"^/api/v1/auth/[^/]+/(start|callback)$")
_IDEMPOTENCY_EXEMPT = _PUBLIC_EXACT


def _is_public(path: str) -> bool:
    return path in _PUBLIC_EXACT or bool(_PUBLIC_PATTERN.match(path))


class ApiMiddleware:
    def __init__(self, app):
        self.app = app

    async def __call__(self, scope: Scope, receive: Receive, send: Send):
        if scope["type"] != "http":
            await self.app(scope, receive, send)
            return
        deps = getattr(scope["app"].state, "deps", None)
        if deps is None:
            await self.app(scope, receive, send)
            return
        self.auth = deps.auth
        self.idem = deps.idem
        path = scope.get("path", "")
        if not path.startswith("/api/v1") or _is_public(path):
            await self.app(scope, receive, send)
            return
        rid = (scope.get("state") or {}).get("request_id", "")
        headers = dict(scope.get("headers") or [])
        authz = headers.get(b"authorization", b"").decode(errors="replace")
        if not authz.startswith("Bearer "):
            await self._error(send, 401, "UNAUTHENTICATED",
                              "missing bearer token", rid)
            return
        try:
            user = await self.auth.resolve_access_token(authz[7:])
        except DomainError as e:
            await self._error(send, e.status, e.code, e.message, rid, e.details)
            return
        except Exception:
            await self._error(send, 500, "INTERNAL", "internal error", rid)
            return
        scope.setdefault("state", {})["user"] = user

        if scope["method"] != "POST" or path in _IDEMPOTENCY_EXEMPT:
            await self.app(scope, receive, send)
            return
        await self._idempotent(scope, receive, send, user, path, rid)

    async def _idempotent(self, scope: Scope, receive: Receive, send: Send,
                          user, path: str, rid: str):
        headers = dict(scope.get("headers") or [])
        key_raw = headers.get(b"idempotency-key", b"").decode(errors="replace")
        try:
            key = UUID(key_raw)
        except (ValueError, AttributeError):
            await self._error(send, 400, "VALIDATION_ERROR",
                              "must be a UUID", rid,
                              {"field": "Idempotency-Key"})
            return
        body = b""
        while True:
            message = await receive()
            if message["type"] == "http.request":
                body += message.get("body") or b""
                if not message.get("more_body"):
                    break
            elif message["type"] == "http.disconnect":
                return
        request_hash = hashlib.sha256(body).hexdigest()
        rec = IdempotencyRecord(
            id=uuid4(), user_id=user.id, key=key, method="POST", path=path,
            request_hash=request_hash,
            created_at=datetime.now(timezone.utc))
        try:
            inserted = await self.idem.insert_pending(rec)
        except Exception:
            await self._error(send, 500, "INTERNAL", "internal error", rid)
            return
        if not inserted:
            try:
                existing = await self.idem.get(user.id, "POST", path, key)
            except Exception:
                await self._error(send, 500, "INTERNAL", "internal error", rid)
                return
            if (existing.request_hash != request_hash
                    or existing.response_status is None):
                await self._error(send, 409, "IDEMPOTENCY_CONFLICT",
                                  "idempotency key conflict", rid)
                return
            payload = existing.response_body or b""
            await send({"type": "http.response.start",
                        "status": existing.response_status,
                        "headers": [(b"content-type",
                                     b"application/json; charset=utf-8")]})
            await send({"type": "http.response.body", "body": payload})
            return

        sent = False

        async def replay_receive() -> Message:
            nonlocal sent
            if sent:
                return {"type": "http.disconnect"}
            sent = True
            return {"type": "http.request", "body": body, "more_body": False}

        status = 500
        captured = bytearray()
        start_message: dict = {}
        handler_failed = False

        async def capture_send(message: Message):
            nonlocal status
            if message["type"] == "http.response.start":
                start_message.update(message)
                status = message["status"]
            elif message["type"] == "http.response.body":
                captured.extend(message.get("body") or b"")

        try:
            await self.app(scope, replay_receive, capture_send)
        except Exception:
            handler_failed = True
        if handler_failed or not (200 <= status < 300):
            try:
                await self.idem.delete_pending(rec.id)
            except Exception:
                pass
        else:
            try:
                await self.idem.complete(rec.id, status, bytes(captured))
            except Exception:
                pass
        if start_message:
            await send(dict(start_message))
        await send({"type": "http.response.body", "body": bytes(captured)})

    async def _error(self, send: Send, status: int, code: str, message: str,
                     rid: str, details=None):
        body = json.dumps(error_body(code, message, rid, details)).encode()
        await send({"type": "http.response.start", "status": status,
                    "headers": [(b"content-type",
                                 b"application/json; charset=utf-8"),
                                (b"content-length", str(len(body)).encode())]})
        await send({"type": "http.response.body", "body": body})
