from __future__ import annotations
import contextvars
import json
from pathlib import Path
from typing import Any, Awaitable, Callable

import asyncpg

_tx: contextvars.ContextVar[asyncpg.Connection | None] = contextvars.ContextVar("tx", default=None)


class NotFoundError(Exception):
    pass


class ConflictError(Exception):
    def __init__(self, current: int):
        super().__init__("revision conflict")
        self.current = current


async def _init_conn(conn: asyncpg.Connection) -> None:
    for t in ("json", "jsonb"):
        await conn.set_type_codec(t, encoder=json.dumps, decoder=json.loads, schema="pg_catalog")


class DB:
    def __init__(self, pool: asyncpg.Pool):
        self.pool = pool

    def q(self):
        tx = _tx.get()
        return tx if tx is not None else self.pool

    async def do(self, fn: Callable[[], Awaitable[Any]]) -> Any:
        async with self.pool.acquire() as conn:
            async with conn.transaction():
                token = _tx.set(conn)
                try:
                    return await fn()
                finally:
                    _tx.reset(token)

    async def revision_guard(self, table: str, id_, user_id) -> None:
        row = await self.q().fetchrow(
            f"SELECT revision FROM {table} WHERE id = $1 AND user_id = $2", id_, user_id)
        if row is None:
            raise NotFoundError()
        raise ConflictError(row["revision"])


def unique_violation(err: Exception) -> bool:
    return isinstance(err, asyncpg.UniqueViolationError)


async def run_migrations(pool: asyncpg.Pool, dir_: str) -> None:
    await pool.execute(
        "CREATE TABLE IF NOT EXISTS schema_migrations (name text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())")
    names = sorted(p.name for p in Path(dir_).iterdir() if p.suffix == ".sql" and p.is_file())
    for name in names:
        applied = await pool.fetchval("SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE name=$1)", name)
        if applied:
            continue
        sql = (Path(dir_) / name).read_text()
        async with pool.acquire() as conn:
            async with conn.transaction():
                await conn.execute(sql)
                await conn.execute("INSERT INTO schema_migrations (name) VALUES ($1)", name)


async def create_pool(database_url: str) -> asyncpg.Pool:
    url = database_url.replace("?sslmode=disable", "").replace("&sslmode=disable", "")
    return await asyncpg.create_pool(url, init=_init_conn)
