from __future__ import annotations
from typing import Optional
from uuid import UUID

from app.domain.entities import Application, ApplicationEvent, Submission, SubmissionDraft
from app.domain.pagination import Cursor, Page
from app.jsonutil import dump
from app.infrastructure.store_common import Conds, Store, list_page, to_model

_APP_COLS = ("id, user_id, revision, job_id, company, title, stage, notes, "
             "applied_at, next_action_at, imported, created_at, updated_at")
_EVENT_COLS = "id, user_id, application_id, type, payload, created_at"
_DRAFT_COLS = ("id, user_id, application_id, application_revision, mode, adapter, "
               "document_version_ids, confirmed_submitted, payload_hash, created_at")
_SUB_COLS = ("id, user_id, application_id, mode, adapter, status, document_version_ids, "
             "approval_id, receipt_url, error_code, created_at, updated_at")


class ApplicationStore(Store):
    async def create(self, a: Application) -> None:
        await self.q().execute(
            f"INSERT INTO applications ({_APP_COLS}) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)",
            a.id, a.user_id, a.revision, a.job_id, a.company, a.title, a.stage, a.notes,
            a.applied_at, a.next_action_at, a.imported, a.created_at, a.updated_at)

    async def get(self, user_id: UUID, id_: UUID) -> Application:
        return to_model(Application, await self.one(
            f"SELECT {_APP_COLS} FROM applications WHERE id = $1 AND user_id = $2", id_, user_id))

    async def list(self, user_id: UUID, stage: Optional[str], query: str, page) -> Page:
        c = Conds()
        c.add("user_id = {}", user_id)
        if stage:
            c.add("stage = {}", stage)
        if query:
            c.add("(company ILIKE {} OR title ILIKE {})", "%" + query + "%", "%" + query + "%")
        c.cursor(page)
        sql, args = c.query(_APP_COLS, "applications", page.effective_limit())
        rows = await self.q().fetch(sql, *args)
        return list_page(rows, page, Application, lambda a: Cursor(a.created_at, a.id))

    async def update(self, a: Application, expected: int) -> None:
        await self.guard_update(
            "applications", a.id, a.user_id,
            "UPDATE applications SET revision = revision + 1, stage = $3, notes = $4, "
            "applied_at = $5, next_action_at = $6, imported = $7, updated_at = $8 "
            "WHERE id = $1 AND user_id = $2 AND revision = $9",
            a.id, a.user_id, a.stage, a.notes, a.applied_at, a.next_action_at, a.imported,
            a.updated_at, expected)

    async def add_event(self, e: ApplicationEvent) -> None:
        await self.q().execute(
            "INSERT INTO application_events (id, user_id, application_id, type, payload, "
            "created_at) VALUES ($1,$2,$3,$4,$5,$6)",
            e.id, e.user_id, e.application_id, e.type, dump(e.payload), e.created_at)

    async def list_events(self, user_id: UUID, application_id: UUID, page) -> Page:
        c = Conds()
        c.add("user_id = {}", user_id)
        c.add("application_id = {}", application_id)
        c.cursor(page)
        sql, args = c.query(_EVENT_COLS, "application_events", page.effective_limit())
        rows = await self.q().fetch(sql, *args)
        return list_page(rows, page, ApplicationEvent, lambda e: Cursor(e.created_at, e.id))


class SubmissionStore(Store):
    async def create_draft(self, d: SubmissionDraft) -> None:
        await self.q().execute(
            f"INSERT INTO submission_drafts ({_DRAFT_COLS}) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)",
            d.id, d.user_id, d.application_id, d.application_revision, d.mode, d.adapter,
            dump(d.document_version_ids), d.confirmed_submitted, d.payload_hash, d.created_at)

    async def get_draft(self, user_id: UUID, id_: UUID) -> SubmissionDraft:
        return to_model(SubmissionDraft, await self.one(
            f"SELECT {_DRAFT_COLS} FROM submission_drafts WHERE id = $1 AND user_id = $2",
            id_, user_id))

    async def create(self, s: Submission) -> None:
        await self.q().execute(
            f"INSERT INTO submissions ({_SUB_COLS}) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)",
            s.id, s.user_id, s.application_id, s.mode, s.adapter, s.status,
            dump(s.document_version_ids), s.approval_id, s.receipt_url, s.error_code,
            s.created_at, s.updated_at)

    async def list(self, user_id: UUID, application_id: UUID, page) -> Page:
        c = Conds()
        c.add("user_id = {}", user_id)
        c.add("application_id = {}", application_id)
        c.cursor(page)
        sql, args = c.query(_SUB_COLS, "submissions", page.effective_limit())
        rows = await self.q().fetch(sql, *args)
        return list_page(rows, page, Submission, lambda s: Cursor(s.created_at, s.id))
