from __future__ import annotations
from datetime import datetime
from typing import Optional
from uuid import UUID

from app.domain.entities import Approval, Operation
from app.domain.pagination import Cursor, Page
from app.jsonutil import dump
from app.infrastructure.store_common import Conds, Store, list_page, to_model

_OP_COLS = ("id, user_id, type, application_id, status, progress, result, error, "
            "input_request, pending_payload, created_at, updated_at")
_APPR_COLS = ("id, user_id, revision, kind, application_id, target_id, target_revision, "
              "payload_hash, status, expires_at, decided_at, consumed_at, created_at, updated_at")


def _op(row) -> Operation:
    o = dict(row)
    return Operation(**o)


class OperationStore(Store):
    async def create(self, o: Operation) -> None:
        await self.q().execute(
            f"INSERT INTO operations ({_OP_COLS}) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)",
            o.id, o.user_id, o.type, o.application_id, o.status, o.progress,
            dump(o.result) if o.result else None, dump(o.error) if o.error else None,
            dump(o.input_request) if o.input_request else None,
            dump(o.pending_payload) if o.pending_payload is not None else None,
            o.created_at, o.updated_at)

    async def get(self, user_id: UUID, id_: UUID) -> Operation:
        return _op(await self.one(
            f"SELECT {_OP_COLS} FROM operations WHERE id = $1 AND user_id = $2", id_, user_id))

    async def update(self, o: Operation) -> None:
        await self.q().execute(
            "UPDATE operations SET status = $2, progress = $3, result = $4, error = $5, "
            "input_request = $6, pending_payload = $7, updated_at = $8 WHERE id = $1",
            o.id, o.status, o.progress, dump(o.result) if o.result else None,
            dump(o.error) if o.error else None,
            dump(o.input_request) if o.input_request else None,
            dump(o.pending_payload) if o.pending_payload is not None else None,
            o.updated_at)


class ApprovalStore(Store):
    async def create(self, a: Approval) -> None:
        await self.q().execute(
            f"INSERT INTO approvals ({_APPR_COLS}) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)",
            a.id, a.user_id, a.revision, a.kind, a.application_id, a.target_id,
            a.target_revision, a.payload_hash, a.status, a.expires_at, a.decided_at,
            a.consumed_at, a.created_at, a.updated_at)

    async def get(self, user_id: UUID, id_: UUID) -> Approval:
        return to_model(Approval, await self.one(
            f"SELECT {_APPR_COLS} FROM approvals WHERE id = $1 AND user_id = $2", id_, user_id))

    async def list(self, user_id: UUID, application_id: Optional[UUID],
                   status: Optional[str], page) -> Page:
        c = Conds()
        c.add("user_id = {}", user_id)
        if application_id:
            c.add("application_id = {}", application_id)
        if status:
            c.add("status = {}", status)
        c.cursor(page)
        sql, args = c.query(_APPR_COLS, "approvals", page.effective_limit())
        rows = await self.q().fetch(sql, *args)
        return list_page(rows, page, Approval, lambda a: Cursor(a.created_at, a.id))

    async def update(self, a: Approval, expected: int) -> None:
        await self.guard_update(
            "approvals", a.id, a.user_id,
            "UPDATE approvals SET revision = revision + 1, status = $3, expires_at = $4, "
            "decided_at = $5, consumed_at = $6, updated_at = $7 "
            "WHERE id = $1 AND user_id = $2 AND revision = $8",
            a.id, a.user_id, a.status, a.expires_at, a.decided_at, a.consumed_at,
            a.updated_at, expected)

    async def find_active(self, user_id: UUID, kind: str, application_id: UUID,
                          target_id: UUID, now: datetime) -> Approval:
        return to_model(Approval, await self.one(
            f"SELECT {_APPR_COLS} FROM approvals WHERE user_id = $1 AND kind = $2 AND "
            "application_id = $3 AND target_id = $4 AND status IN ('APPROVED','PENDING') "
            "AND (expires_at IS NULL OR expires_at > $5) "
            "ORDER BY created_at DESC LIMIT 1",
            user_id, kind, application_id, target_id, now))
