from __future__ import annotations

from datetime import datetime, timezone
from uuid import uuid4

import pytest

from app.domain.pagination import (
    Cursor, PageRequest, decode_cursor, new_page,
)

TS = datetime(2025, 1, 2, 3, 4, 5, 6, tzinfo=timezone.utc)


def test_cursor_roundtrip():
    c = Cursor(created_at=TS, id=uuid4())
    dec = decode_cursor(c.encode())
    assert dec.created_at == c.created_at
    assert dec.id == c.id
    with pytest.raises(Exception):
        decode_cursor("!!!notbase64!!!")


def test_new_page():
    items = [1, 2, 3]
    p = new_page(items, 2, lambda i: Cursor(created_at=TS, id=uuid4()))
    assert p.has_more and p.next_cursor and len(p.items) == 2
    p = new_page(items[:1], 2, lambda i: Cursor(created_at=TS, id=uuid4()))
    assert not p.has_more and p.next_cursor is None


def test_effective_limit():
    assert PageRequest().effective_limit() == 50
    assert PageRequest(limit=500).effective_limit() == 100
    assert PageRequest(limit=7).effective_limit() == 7
