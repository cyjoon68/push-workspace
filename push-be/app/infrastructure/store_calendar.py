from __future__ import annotations
from datetime import datetime
from typing import Optional
from uuid import UUID

from app.domain.entities import CalendarEvent, InterviewSession
from app.domain.pagination import Cursor, Page, new_page
from app.jsonutil import dump
from app.infrastructure.store_common import Conds, Store, list_page, to_model

_INTERVIEW_COLS = ("id, user_id, revision, application_id, title, scheduled_at, "
                   "duration_minutes, event_id, evidence_ids, company_sources, notes, "
                   "reflection, created_at, updated_at")
_EVENT_COLS = ("id, user_id, revision, application_id, type, title, starts_at, ends_at, "
               "time_zone, source, external_id, notes, created_at, updated_at")


def _interview(row) -> InterviewSession:
    v = dict(row)
    v["evidence_ids"] = v.get("evidence_ids") or []
    v["company_sources"] = v.get("company_sources") or []
    return InterviewSession(**v)


class InterviewStore(Store):
    async def create(self, v: InterviewSession) -> None:
        await self.q().execute(
            f"INSERT INTO interviews ({_INTERVIEW_COLS}) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)",
            v.id, v.user_id, v.revision, v.application_id, v.title, v.scheduled_at,
            v.duration_minutes, v.event_id, dump(v.evidence_ids), dump(v.company_sources),
            v.notes, v.reflection, v.created_at, v.updated_at)

    async def get(self, user_id: UUID, id_: UUID) -> InterviewSession:
        return _interview(await self.one(
            f"SELECT {_INTERVIEW_COLS} FROM interviews WHERE id = $1 AND user_id = $2",
            id_, user_id))

    async def list(self, user_id: UUID, application_id: Optional[UUID],
                   from_: Optional[datetime], to: Optional[datetime], page) -> Page:
        c = Conds()
        c.add("user_id = {}", user_id)
        if application_id:
            c.add("application_id = {}", application_id)
        if from_:
            c.add("scheduled_at >= {}", from_)
        if to:
            c.add("scheduled_at <= {}", to)
        c.cursor(page)
        sql, args = c.query(_INTERVIEW_COLS, "interviews", page.effective_limit())
        rows = await self.q().fetch(sql, *args)
        return new_page([_interview(r) for r in rows], page.effective_limit(),
                        lambda v: Cursor(v.created_at, v.id))

    async def update(self, v: InterviewSession, expected: int) -> None:
        await self.guard_update(
            "interviews", v.id, v.user_id,
            "UPDATE interviews SET revision = revision + 1, title = $3, scheduled_at = $4, "
            "duration_minutes = $5, event_id = $6, evidence_ids = $7, company_sources = $8, "
            "notes = $9, reflection = $10, updated_at = $11 "
            "WHERE id = $1 AND user_id = $2 AND revision = $12",
            v.id, v.user_id, v.title, v.scheduled_at, v.duration_minutes, v.event_id,
            dump(v.evidence_ids), dump(v.company_sources), v.notes, v.reflection,
            v.updated_at, expected)

    async def create_calendar_event(self, e: CalendarEvent) -> None:
        await self.q().execute(
            f"INSERT INTO calendar_events ({_EVENT_COLS}) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)",
            e.id, e.user_id, e.revision, e.application_id, e.type, e.title, e.starts_at,
            e.ends_at, e.time_zone, e.source, e.external_id, e.notes, e.created_at,
            e.updated_at)

    async def update_calendar_event(self, id_: UUID, title: str, starts_at, ends_at) -> None:
        await self.q().execute(
            "UPDATE calendar_events SET title = coalesce(nullif($2,''), title), "
            "starts_at = $3, ends_at = $4, updated_at = now() WHERE id = $1",
            id_, title, starts_at, ends_at)

    async def count_by_application(self, user_id: UUID, application_id: UUID) -> int:
        return await self.q().fetchval(
            "SELECT count(*) FROM interviews WHERE user_id = $1 AND application_id = $2",
            user_id, application_id)


class CalendarEventStore(Store):
    async def create(self, e: CalendarEvent) -> None:
        await self.q().execute(
            f"INSERT INTO calendar_events ({_EVENT_COLS}) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)",
            e.id, e.user_id, e.revision, e.application_id, e.type, e.title, e.starts_at,
            e.ends_at, e.time_zone, e.source, e.external_id, e.notes, e.created_at, e.updated_at)

    async def get(self, user_id: UUID, id_: UUID) -> CalendarEvent:
        return to_model(CalendarEvent, await self.one(
            f"SELECT {_EVENT_COLS} FROM calendar_events WHERE id = $1 AND user_id = $2",
            id_, user_id))

    async def list(self, user_id: UUID, application_id: Optional[UUID],
                   from_: Optional[datetime], to: Optional[datetime],
                   source: Optional[str], page) -> Page:
        c = Conds()
        c.add("user_id = {}", user_id)
        if application_id:
            c.add("application_id = {}", application_id)
        if from_:
            c.add("starts_at >= {}", from_)
        if to:
            c.add("starts_at <= {}", to)
        if source:
            c.add("source = {}", source)
        c.cursor(page)
        sql, args = c.query(_EVENT_COLS, "calendar_events", page.effective_limit())
        rows = await self.q().fetch(sql, *args)
        return list_page(rows, page, CalendarEvent, lambda e: Cursor(e.created_at, e.id))

    async def patch(self, e: CalendarEvent, expected: int) -> None:
        await self.guard_update(
            "calendar_events", e.id, e.user_id,
            "UPDATE calendar_events SET revision = revision + 1, title = $3, starts_at = $4, "
            "ends_at = $5, time_zone = $6, notes = $7, updated_at = $8 "
            "WHERE id = $1 AND user_id = $2 AND revision = $9",
            e.id, e.user_id, e.title, e.starts_at, e.ends_at, e.time_zone, e.notes,
            e.updated_at, expected)

    async def update(self, e: CalendarEvent) -> None:
        await self.q().execute(
            "UPDATE calendar_events SET revision = revision + 1, application_id = $3, "
            "type = $4, title = $5, starts_at = $6, ends_at = $7, time_zone = $8, source = $9, "
            "external_id = $10, notes = $11, updated_at = $12 WHERE id = $1 AND user_id = $2",
            e.id, e.user_id, e.application_id, e.type, e.title, e.starts_at, e.ends_at,
            e.time_zone, e.source, e.external_id, e.notes, e.updated_at)

    async def delete(self, user_id: UUID, id_: UUID, expected: int) -> None:
        tag = await self.q().execute(
            "DELETE FROM calendar_events WHERE id = $1 AND user_id = $2 AND revision = $3",
            id_, user_id, expected)
        if tag.split()[-1] == "0":
            await self.d.revision_guard("calendar_events", id_, user_id)

    async def upsert_external(self, e: CalendarEvent) -> None:
        await self.q().execute(
            f"INSERT INTO calendar_events ({_EVENT_COLS}) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) "
            "ON CONFLICT (user_id, external_id) WHERE external_id IS NOT NULL DO UPDATE SET "
            "title = $6, starts_at = $7, ends_at = $8, time_zone = $9, notes = $12, "
            "revision = calendar_events.revision + 1, updated_at = $14",
            e.id, e.user_id, e.revision, e.application_id, e.type, e.title, e.starts_at,
            e.ends_at, e.time_zone, e.source, e.external_id, e.notes, e.created_at,
            e.updated_at)

    async def delete_external(self, user_id: UUID, external_id: str) -> None:
        await self.q().execute(
            "DELETE FROM calendar_events WHERE user_id = $1 AND external_id = $2 "
            "AND source = 'GOOGLE'", user_id, external_id)
