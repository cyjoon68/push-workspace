from __future__ import annotations
from datetime import datetime, timedelta, timezone
from typing import Optional
from uuid import UUID, uuid4
from zoneinfo import ZoneInfo, ZoneInfoNotFoundError

from app.db import DB, NotFoundError
from app.domain import entities as ent
from app.domain.errors import (
    internal, map_revision_err, not_found, revision_conflict, validation,
    validation_field,
)
from app.infrastructure.store_applications import ApplicationStore
from app.infrastructure.store_calendar import CalendarEventStore

MAX_RANGE_DAYS = 366


def _now() -> datetime:
    return datetime.now(timezone.utc)


def _valid_tz(tz: str) -> bool:
    try:
        ZoneInfo(tz)
        return True
    except (ZoneInfoNotFoundError, ValueError, KeyError):
        return False


def validate_event_input(type_: str, title: str, starts_at: datetime,
                         ends_at: datetime, tz: str) -> None:
    if not ent.valid_event_type(type_):
        raise validation_field("type", "unsupported event type")
    if not 1 <= len(title) <= 300:
        raise validation_field("title", "title must be 1-300 characters")
    if starts_at is None:
        raise validation_field("startsAt", "required")
    if ends_at <= starts_at:
        raise validation_field("endsAt", "must be after startsAt")
    if tz and not _valid_tz(tz):
        raise validation_field("timeZone", "invalid IANA time zone")


class CalendarService:
    def __init__(self, db: DB):
        self.db = db
        self.events = CalendarEventStore(db)
        self.applications = ApplicationStore(db)

    async def list(self, user_id: UUID, application_id: Optional[UUID],
                   from_: Optional[datetime], to: Optional[datetime],
                   source: Optional[str], page):
        if from_ is None or to is None:
            raise validation("from and to are required")
        if to <= from_:
            raise validation_field("to", "must be after from")
        if to - from_ > timedelta(days=MAX_RANGE_DAYS):
            raise validation_field("to", "range must be at most 366 days")
        try:
            return await self.events.list(user_id, application_id, from_, to,
                                          source, page)
        except Exception:
            raise internal()

    async def create(self, user_id: UUID, application_id: Optional[UUID],
                     type_: str, title: str, starts_at: datetime, ends_at: datetime,
                     time_zone: str, notes: str) -> ent.CalendarEvent:
        validate_event_input(type_, title, starts_at, ends_at, time_zone)
        if application_id is not None:
            try:
                await self.applications.get(user_id, application_id)
            except NotFoundError:
                raise not_found()
            except Exception:
                raise internal()
        now = _now()
        e = ent.CalendarEvent(
            id=uuid4(), user_id=user_id, revision=1, application_id=application_id,
            type=type_, title=title, starts_at=starts_at, ends_at=ends_at,
            time_zone=time_zone or "UTC", source=ent.EVENT_SOURCE_LOCAL,
            notes=notes, created_at=now, updated_at=now)
        try:
            await self.events.create(e)
        except Exception:
            raise internal()
        return e

    async def patch(self, user_id: UUID, id_: UUID, expected: int,
                    title: Optional[str], starts_at, ends_at,
                    time_zone: Optional[str], notes: Optional[str]) -> ent.CalendarEvent:
        try:
            e = await self.events.get(user_id, id_)
        except NotFoundError:
            raise not_found()
        except Exception:
            raise internal()
        if e.revision != expected:
            raise revision_conflict(e.revision)
        if title is not None:
            if not 1 <= len(title) <= 300:
                raise validation_field("title", "title must be 1-300 characters")
            e.title = title
        if starts_at is not None:
            e.starts_at = starts_at
        if ends_at is not None:
            e.ends_at = ends_at
        if e.ends_at <= e.starts_at:
            raise validation_field("endsAt", "must be after startsAt")
        if time_zone is not None:
            if not _valid_tz(time_zone):
                raise validation_field("timeZone", "invalid IANA time zone")
            e.time_zone = time_zone
        if notes is not None:
            e.notes = notes
        e.updated_at = _now()
        try:
            await self.events.patch(e, expected)
        except Exception as err:
            raise map_revision_err(err)
        e.revision = expected + 1
        return e

    async def delete(self, user_id: UUID, id_: UUID, expected: int) -> None:
        try:
            await self.events.delete(user_id, id_, expected)
        except NotFoundError:
            raise not_found()
        except Exception as err:
            raise map_revision_err(err)
