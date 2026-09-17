from __future__ import annotations
import base64
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Callable, Generic, Optional, TypeVar
from uuid import UUID

DEFAULT_LIMIT = 50
MAX_LIMIT = 100

T = TypeVar("T")


@dataclass
class Cursor:
    created_at: datetime
    id: UUID

    def encode(self) -> str:
        ts = self.created_at.astimezone(timezone.utc)
        frac = ""
        if ts.microsecond:
            frac = f".{ts.microsecond:06d}".rstrip("0")
        raw = ts.strftime("%Y-%m-%dT%H:%M:%S") + frac + "Z|" + str(self.id)
        return base64.urlsafe_b64encode(raw.encode()).rstrip(b"=").decode()


def decode_cursor(s: str) -> Cursor:
    try:
        raw = base64.urlsafe_b64decode(s + "=" * (-len(s) % 4))
        parts = raw.decode().split("|", 1)
        if len(parts) != 2:
            raise ValueError("bad cursor")
        return Cursor(created_at=datetime.fromisoformat(parts[0]), id=UUID(parts[1]))
    except Exception:
        raise ValueError("bad cursor")


@dataclass
class PageRequest:
    limit: int = 0
    cursor: Optional[Cursor] = None

    def effective_limit(self) -> int:
        if self.limit <= 0:
            return DEFAULT_LIMIT
        return min(self.limit, MAX_LIMIT)


@dataclass
class Page(Generic[T]):
    items: list[T] = field(default_factory=list)
    next_cursor: Optional[str] = None
    has_more: bool = False


def new_page(items: list[T], limit: int, cursor_of: Callable[[T], Cursor]) -> Page[T]:
    p: Page[T] = Page(items=items or [])
    if len(p.items) > limit:
        p.items = p.items[:limit]
        p.has_more = True
        p.next_cursor = cursor_of(p.items[-1]).encode()
    return p
