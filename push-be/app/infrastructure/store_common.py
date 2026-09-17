from __future__ import annotations
from typing import Any

import asyncpg

from app.db import DB, NotFoundError
from app.domain.pagination import Page, PageRequest, new_page


class Store:
    def __init__(self, db: DB):
        self.d = db

    def q(self):
        return self.d.q()

    async def one(self, sql: str, *args) -> asyncpg.Record:
        row = await self.q().fetchrow(sql, *args)
        if row is None:
            raise NotFoundError()
        return row

    async def guard_update(self, table: str, id_, user_id, sql: str, *args) -> None:
        tag = await self.q().execute(sql, *args)
        if tag.split()[-1] == "0":
            await self.d.revision_guard(table, id_, user_id)


def to_model(cls, row) -> Any:
    return cls(**dict(row))


def list_page(rows, page: PageRequest, cls, cursor_of) -> Page:
    items = [to_model(cls, r) for r in rows]
    return new_page(items, page.effective_limit(), cursor_of)


class Conds:
    def __init__(self):
        self.where: list[str] = []
        self.args: list[Any] = []

    def add(self, clause: str, *vals):
        self.args.extend(vals)
        n = len(self.args)
        self.where.append(clause.format(*["$" + str(i) for i in range(n - len(vals) + 1, n + 1)]))

    def cursor(self, page: PageRequest):
        if page.cursor is not None:
            self.args.extend([page.cursor.created_at, page.cursor.id])
            n = len(self.args)
            self.where.append(f"(created_at, id) < (${n - 1}, ${n})")

    def query(self, cols: str, table: str, limit: int) -> tuple[str, list[Any]]:
        self.args.append(limit + 1)
        return (f"SELECT {cols} FROM {table} WHERE {' AND '.join(self.where)} "
                f"ORDER BY created_at DESC, id DESC LIMIT ${len(self.args)}"), self.args


def m(cls):
    def conv(row):
        return cls(**dict(row))
    return conv
