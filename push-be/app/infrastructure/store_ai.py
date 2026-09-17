from __future__ import annotations
from datetime import datetime
from typing import Optional
from uuid import UUID

from app.db import NotFoundError
from app.domain.entities import AiKey, AiUsage
from app.domain.pagination import Cursor, Page, new_page
from app.infrastructure.store_common import Store, to_model

_USAGE_COLS = ("id, user_id, operation_id, provider, model, managed, input_tokens, "
               "output_tokens, cost_micro_credits, status, created_at")


class AiKeyStore(Store):
    async def put(self, k: AiKey) -> None:
        await self.q().execute(
            "INSERT INTO ai_keys (user_id, provider, last_four, ciphertext, nonce, updated_at) "
            "VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (user_id, provider) DO UPDATE SET "
            "last_four = $3, ciphertext = $4, nonce = $5, updated_at = $6",
            k.user_id, k.provider, k.last_four, bytes(k.ciphertext), bytes(k.nonce),
            k.updated_at)

    async def get(self, user_id: UUID, provider: str) -> AiKey:
        return to_model(AiKey, await self.one(
            "SELECT user_id, provider, last_four, ciphertext, nonce, updated_at "
            "FROM ai_keys WHERE user_id = $1 AND provider = $2", user_id, provider))

    async def list(self, user_id: UUID) -> list[AiKey]:
        rows = await self.q().fetch(
            "SELECT user_id, provider, last_four, ciphertext, nonce, updated_at "
            "FROM ai_keys WHERE user_id = $1 ORDER BY provider", user_id)
        return [to_model(AiKey, r) for r in rows]

    async def delete(self, user_id: UUID, provider: str) -> None:
        tag = await self.q().execute(
            "DELETE FROM ai_keys WHERE user_id = $1 AND provider = $2", user_id, provider)
        if tag.split()[-1] == "0":
            raise NotFoundError()

    async def has(self, user_id: UUID, provider: str) -> bool:
        return await self.q().fetchval(
            "SELECT EXISTS(SELECT 1 FROM ai_keys WHERE user_id = $1 AND provider = $2)",
            user_id, provider)


class AiUsageStore(Store):
    async def create(self, u: AiUsage) -> None:
        await self.q().execute(
            f"INSERT INTO ai_usage ({_USAGE_COLS}) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)",
            u.id, u.user_id, u.operation_id, u.provider, u.model, u.managed,
            u.input_tokens, u.output_tokens, u.cost_micro_credits, u.status, u.created_at)

    async def list(self, user_id: UUID, from_: Optional[datetime],
                   to: Optional[datetime], page) -> Page:
        q = (f"SELECT {_USAGE_COLS} FROM ai_usage WHERE user_id = $1")
        args: list = [user_id]
        n = 1
        if from_:
            n += 1
            q += f" AND created_at >= ${n}"
            args.append(from_)
        if to:
            n += 1
            q += f" AND created_at <= ${n}"
            args.append(to)
        if page.cursor is not None:
            n += 1
            q += f" AND (created_at, id) < (${n}"
            n += 1
            q += f", ${n})"
            args.extend([page.cursor.created_at, page.cursor.id])
        q += " ORDER BY created_at DESC, id DESC"
        n += 1
        q += f" LIMIT ${n}"
        args.append(page.effective_limit() + 1)
        rows = await self.q().fetch(q, *args)
        return new_page([to_model(AiUsage, r) for r in rows], page.effective_limit(),
                        lambda u: Cursor(u.created_at, u.id))
