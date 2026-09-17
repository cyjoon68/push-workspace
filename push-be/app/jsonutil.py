from __future__ import annotations
from datetime import date, datetime, timezone
from uuid import UUID

from pydantic import BaseModel


def _ts(v: datetime) -> str:
    if v.tzinfo is None:
        v = v.replace(tzinfo=timezone.utc)
    v = v.astimezone(timezone.utc)
    frac = f".{v.microsecond:06d}" if v.microsecond else ""
    return v.strftime("%Y-%m-%dT%H:%M:%S") + frac + "Z"


def to_jsonable(v):
    if isinstance(v, BaseModel):
        return _j(v.model_dump(by_alias=True, mode="python"))
    return _j(v)


def dump(v):
    out = to_jsonable(v)
    return out if out is not None else {}


def _j(v):
    if isinstance(v, BaseModel):
        return _j(v.model_dump(by_alias=True, mode="python"))
    if isinstance(v, datetime):
        return _ts(v)
    if isinstance(v, date):
        return v.isoformat()
    if isinstance(v, UUID):
        return str(v)
    if isinstance(v, dict):
        return {k: _j(x) for k, x in v.items()}
    if isinstance(v, (list, tuple)):
        return [_j(x) for x in v]
    return v
