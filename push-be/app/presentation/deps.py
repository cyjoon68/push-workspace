from __future__ import annotations

import json
from datetime import datetime
from typing import Optional
from uuid import UUID

from fastapi import Request
from fastapi.responses import JSONResponse
from pydantic import BaseModel, ValidationError

from app.domain import entities as ent
from app.domain.errors import unauthenticated, validation, validation_field
from app.domain.pagination import PageRequest, decode_cursor
from app.jsonutil import to_jsonable

MAX_LIMIT = 100


def current_user(request: Request) -> ent.User:
    u = getattr(request.state, "user", None)
    if u is None:
        raise unauthenticated("missing bearer token")
    return u


async def bind_json(request: Request, model: type[BaseModel]):
    body = await request.body()
    if not body:
        raise validation("request body required")
    try:
        raw = json.loads(body)
    except json.JSONDecodeError as e:
        raise validation("invalid request body: " + str(e))
    try:
        return model.model_validate(raw)
    except ValidationError as e:
        raise validation("invalid request body: " + str(e))


def param_id(value: str, name: str) -> UUID:
    try:
        return UUID(value)
    except (ValueError, AttributeError):
        raise validation_field(name, "must be a UUID")


def page_request(request: Request) -> PageRequest:
    p = PageRequest()
    limit = request.query_params.get("limit")
    if limit:
        try:
            n = int(limit)
        except ValueError:
            raise validation_field("limit", "must be a positive integer")
        if n < 1:
            raise validation_field("limit", "must be a positive integer")
        p.limit = min(n, MAX_LIMIT)
    cur = request.query_params.get("cursor")
    if cur:
        try:
            p.cursor = decode_cursor(cur)
        except Exception:
            raise validation_field("cursor", "invalid cursor")
    return p


def optional_query_uuid(request: Request, name: str) -> Optional[UUID]:
    v = request.query_params.get(name)
    if not v:
        return None
    try:
        return UUID(v)
    except ValueError:
        raise validation_field(name, "must be a UUID")


def optional_query_time(request: Request, name: str) -> Optional[datetime]:
    v = request.query_params.get(name)
    if not v:
        return None
    try:
        return parse_rfc3339(v)
    except ValueError:
        raise validation_field(name, "must be RFC3339")


def parse_rfc3339(s: str) -> datetime:
    dt = datetime.fromisoformat(s.replace("Z", "+00:00"))
    return dt


def data(status: int, v) -> JSONResponse:
    return JSONResponse(content={"data": to_jsonable(v)},
                        status_code=status)


def page_body(p) -> JSONResponse:
    return JSONResponse(content={
        "data": to_jsonable(p.items),
        "page": {"nextCursor": p.next_cursor, "hasMore": p.has_more}},
        status_code=200)


def parse_deadline(value: Optional[str], present: bool):
    if not present:
        return None, False
    if value is None:
        return None, True
    try:
        dt = datetime.strptime(value, "%Y-%m-%d")
    except (ValueError, TypeError):
        raise validation_field("deadline", "must be YYYY-MM-DD")
    if dt.strftime("%Y-%m-%d") != value:
        raise validation_field("deadline", "must be YYYY-MM-DD")
    return dt, True


def parse_time_field(value: Optional[str], field: str) -> datetime:
    if not isinstance(value, str):
        raise validation_field(field, "must be RFC3339")
    try:
        return parse_rfc3339(value)
    except ValueError:
        raise validation_field(field, "must be RFC3339")
