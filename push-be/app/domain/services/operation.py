from __future__ import annotations
from datetime import datetime, timezone
from typing import Optional
from uuid import UUID

from app.db import DB, NotFoundError
from app.domain import entities as ent
from app.domain.errors import internal, invalid_transition, not_found
from app.infrastructure.store_operations import OperationStore


def _now() -> datetime:
    return datetime.now(timezone.utc)


class OperationService:
    def __init__(self, db: DB, evidence_svc):
        self.db = db
        self.ops = OperationStore(db)
        self.evidence = evidence_svc

    async def get(self, user_id: UUID, id_: UUID) -> ent.Operation:
        try:
            return await self.ops.get(user_id, id_)
        except NotFoundError:
            raise not_found()
        except Exception:
            raise internal()

    async def submit_input(self, user_id: UUID, id_: UUID, fields: dict,
                           source_id: Optional[UUID]) -> ent.Operation:
        out = None

        async def work():
            nonlocal out
            try:
                op = await self.ops.get(user_id, id_)
            except NotFoundError:
                raise not_found()
            if ent.terminal_operation_status(op.status):
                raise invalid_transition("operation already finished")
            if op.status != ent.OP_NEEDS_INPUT or op.input_request is None:
                raise invalid_transition("operation does not await input")
            if op.type == ent.OP_EVIDENCE_IMPORT:
                await self.evidence.complete_import_input(op, fields)
            else:
                raise invalid_transition("operation does not accept input")
            op.input_request = None
            op.updated_at = _now()
            if op.status == ent.OP_NEEDS_INPUT:
                op.status = ent.OP_QUEUED
            await self.ops.update(op)
            out = op

        await self.db.do(work)
        return out

    async def cancel(self, user_id: UUID, id_: UUID) -> ent.Operation:
        out = None

        async def work():
            nonlocal out
            try:
                op = await self.ops.get(user_id, id_)
            except NotFoundError:
                raise not_found()
            if ent.terminal_operation_status(op.status):
                out = op
                return
            op.status = ent.OP_CANCELLED
            op.updated_at = _now()
            await self.ops.update(op)
            out = op

        await self.db.do(work)
        return out
