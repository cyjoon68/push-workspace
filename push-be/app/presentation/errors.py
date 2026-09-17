from __future__ import annotations

import json
import uuid

from fastapi import Request
from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse
from starlette.exceptions import HTTPException
from starlette.types import Message, Receive, Scope, Send

from app.domain.errors import DomainError


def error_body(code: str, message: str, request_id: str,
               details=None) -> dict:
    return {"error": {"code": code, "message": message,
                      "requestId": request_id, "details": details}}


def request_id_of(request: Request) -> str:
    return getattr(request.state, "request_id", "")


async def domain_error_handler(request: Request, exc: DomainError):
    return JSONResponse(
        status_code=exc.status,
        content=error_body(exc.code, exc.message, request_id_of(request),
                           exc.details))


async def validation_error_handler(request: Request,
                                   exc: RequestValidationError):
    return JSONResponse(
        status_code=400,
        content=error_body("VALIDATION_ERROR", "invalid request",
                           request_id_of(request)))


async def http_error_handler(request: Request, exc: HTTPException):
    code = "VALIDATION_ERROR"
    if exc.status_code in (404, 405):
        code = "NOT_FOUND"
    elif exc.status_code == 401:
        code = "UNAUTHENTICATED"
    import http as _http
    message = _http.HTTPStatus(exc.status_code).phrase \
        if exc.status_code in _http.HTTPStatus._value2member_map_ \
        else "Error"
    return JSONResponse(
        status_code=exc.status_code,
        content=error_body(code, message, request_id_of(request)))


async def unhandled_error_handler(request: Request, exc: Exception):
    return JSONResponse(
        status_code=500,
        content=error_body("INTERNAL", "internal error",
                           request_id_of(request)))


class RequestIDMiddleware:
    def __init__(self, app):
        self.app = app

    async def __call__(self, scope: Scope, receive: Receive, send: Send):
        if scope["type"] != "http":
            await self.app(scope, receive, send)
            return
        headers = dict(scope.get("headers") or [])
        rid = headers.get(b"x-request-id", b"").decode(errors="replace")
        if not rid or len(rid) > 128:
            rid = "req_" + str(uuid.uuid4())
        scope.setdefault("state", {})["request_id"] = rid

        async def send_with_rid(message: Message):
            if message["type"] == "http.response.start":
                hdrs = list(message.get("headers") or [])
                hdrs.append((b"x-request-id", rid.encode()))
                message["headers"] = hdrs
            await send(message)

        await self.app(scope, receive, send_with_rid)


class BodyLimitMiddleware:
    def __init__(self, app, limit: int):
        self.app = app
        self.limit = limit

    async def __call__(self, scope: Scope, receive: Receive, send: Send):
        if scope["type"] != "http":
            await self.app(scope, receive, send)
            return
        headers = dict(scope.get("headers") or [])
        try:
            content_length = int(headers.get(b"content-length", b"0"))
        except ValueError:
            content_length = 0
        if content_length > self.limit:
            await self._too_large(scope, send)
            return
        received = 0
        overflow = False

        async def counting_receive() -> Message:
            nonlocal received, overflow
            message = await receive()
            if message["type"] == "http.request":
                received += len(message.get("body") or b"")
                if received > self.limit:
                    overflow = True
                    return {"type": "http.disconnect"}
            return message

        async def guarded_send(message: Message):
            if overflow and message["type"] == "http.response.start":
                message["status"] = 413
                message["headers"] = [(b"content-type", b"application/json")]
            elif overflow and message["type"] == "http.response.body":
                message["body"] = json.dumps(error_body(
                    "PAYLOAD_TOO_LARGE", "request body too large",
                    (scope.get("state") or {}).get("request_id", ""))).encode()
                message["more_body"] = False
            await send(message)

        await self.app(scope, counting_receive, guarded_send)

    async def _too_large(self, scope: Scope, send: Send):
        body = json.dumps(error_body(
            "PAYLOAD_TOO_LARGE", "request body too large",
            (scope.get("state") or {}).get("request_id", ""))).encode()
        await send({"type": "http.response.start", "status": 413,
                    "headers": [(b"content-type", b"application/json"),
                                (b"content-length", str(len(body)).encode())]})
        await send({"type": "http.response.body", "body": body})
