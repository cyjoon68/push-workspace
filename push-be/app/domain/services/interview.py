from __future__ import annotations
from datetime import datetime, timedelta, timezone
from typing import Optional
from uuid import UUID, uuid4

from app.db import DB, NotFoundError
from app.domain import entities as ent
from app.domain.errors import (
    internal, map_revision_err, not_found, revision_conflict, validation_field,
)
from app.domain.services.ai import AIGate
from app.domain.services.calendar import _valid_tz
from app.domain.content import code_point_slice
from app.domain.validators import code_point_len, validate_https_url
from app.infrastructure.store_applications import ApplicationStore
from app.infrastructure.store_calendar import InterviewStore
from app.infrastructure.store_evidence import EvidenceStore
from app.infrastructure.store_jobs import JobStore
from app.infrastructure.store_operations import OperationStore
from app.jsonutil import _ts


def _now() -> datetime:
    return datetime.now(timezone.utc)


def validate_company_sources(sources: list[ent.CompanySource], now: datetime) -> None:
    if len(sources) > 10:
        raise validation_field("companySources", "at most 10 sources")
    for cs in sources:
        try:
            validate_https_url(cs.source_url)
        except ValueError:
            raise validation_field("companySources", "sourceUrl must be http(s)")
        if not 1 <= code_point_len(cs.source_text) <= 20000:
            raise validation_field(
                "companySources", "sourceText must be 1-20000 characters")
        if cs.accessed_at > now:
            raise validation_field(
                "companySources", "accessedAt cannot be in the future")


class InterviewService:
    def __init__(self, db: DB, ai: AIGate):
        self.db = db
        self.interviews = InterviewStore(db)
        self.applications = ApplicationStore(db)
        self.jobs = JobStore(db)
        self.evidence = EvidenceStore(db)
        self.ops = OperationStore(db)
        self.ai = ai

    async def list(self, user_id: UUID, application_id: Optional[UUID],
                   from_, to, page):
        try:
            return await self.interviews.list(user_id, application_id, from_, to, page)
        except Exception:
            raise internal()

    async def get(self, user_id: UUID, id_: UUID) -> ent.InterviewSession:
        try:
            return await self.interviews.get(user_id, id_)
        except NotFoundError:
            raise not_found()
        except Exception:
            raise internal()

    async def create(self, user_id: UUID, application_id: UUID, title: str,
                     scheduled_at: datetime, duration_minutes: Optional[int],
                     evidence_ids: list[UUID], company_sources: list[ent.CompanySource],
                     notes: str, time_zone: str) -> ent.InterviewSession:
        if not 1 <= len(title) <= 300:
            raise validation_field("title", "title must be 1-300 characters")
        if scheduled_at is None:
            raise validation_field("scheduledAt", "required")
        now = _now()
        validate_company_sources(company_sources or [], now)
        out = None

        async def work():
            nonlocal out
            try:
                app = await self.applications.get(user_id, application_id)
            except NotFoundError:
                raise not_found()
            for eid in evidence_ids or []:
                try:
                    await self.evidence.get(user_id, eid)
                except Exception:
                    raise validation_field(
                        "evidenceIds", "evidence " + str(eid) + " not found")
            tz = time_zone or "UTC"
            if not _valid_tz(tz):
                raise validation_field("timeZone", "invalid IANA time zone")
            ends = scheduled_at
            if duration_minutes is not None:
                ends = scheduled_at + timedelta(minutes=duration_minutes)
            event = ent.CalendarEvent(
                id=uuid4(), user_id=user_id, revision=1, application_id=app.id,
                type="INTERVIEW", title=title, starts_at=scheduled_at, ends_at=ends,
                time_zone=tz, source=ent.EVENT_SOURCE_LOCAL, notes=notes,
                created_at=now, updated_at=now)
            await self.interviews.create_calendar_event(event)
            session = ent.InterviewSession(
                id=uuid4(), user_id=user_id, revision=1, application_id=application_id,
                title=title, scheduled_at=scheduled_at,
                duration_minutes=duration_minutes, event_id=event.id,
                evidence_ids=evidence_ids or [],
                company_sources=company_sources or [], notes=notes,
                created_at=now, updated_at=now)
            await self.interviews.create(session)
            await self.applications.add_event(ent.ApplicationEvent(
                id=uuid4(), user_id=user_id, application_id=app.id,
                type=ent.EVENT_INTERVIEW,
                payload={"interviewId": session.id, "scheduledAt": scheduled_at},
                created_at=now))
            out = session

        await self.db.do(work)
        return out

    async def patch(self, user_id: UUID, id_: UUID, expected: int,
                    title: Optional[str], scheduled_at, company_sources,
                    notes: Optional[str], reflection: Optional[str]) -> ent.InterviewSession:
        out = None

        async def work():
            nonlocal out
            try:
                v = await self.interviews.get(user_id, id_)
            except NotFoundError:
                raise not_found()
            if v.revision != expected:
                raise revision_conflict(v.revision)
            now = _now()
            if title is not None:
                if not 1 <= len(title) <= 300:
                    raise validation_field("title", "title must be 1-300 characters")
                v.title = title
            if scheduled_at is not None:
                v.scheduled_at = scheduled_at
            if company_sources is not None:
                validate_company_sources(company_sources, now)
                v.company_sources = company_sources
            if notes is not None:
                v.notes = notes
            if reflection is not None:
                v.reflection = reflection
            v.updated_at = now
            try:
                await self.interviews.update(v, expected)
            except Exception as err:
                raise map_revision_err(err)
            v.revision = expected + 1
            if v.event_id is not None and scheduled_at is not None:
                ends = v.scheduled_at
                if v.duration_minutes is not None:
                    ends = v.scheduled_at + timedelta(minutes=v.duration_minutes)
                await self.interviews.update_calendar_event(
                    v.event_id, "", v.scheduled_at, ends)
            out = v

        await self.db.do(work)
        return out

    async def prepare(self, user_id: UUID, id_: UUID, expected: int,
                      ai: Optional[ent.AiOptions]) -> ent.Operation:
        await self.ai.check(user_id, ai)
        op = None

        async def work():
            nonlocal op
            try:
                v = await self.interviews.get(user_id, id_)
            except NotFoundError:
                raise not_found()
            if v.revision != expected:
                raise revision_conflict(v.revision)
            try:
                app = await self.applications.get(user_id, v.application_id)
                job = await self.jobs.get(user_id, app.job_id)
            except Exception:
                raise internal()
            eids = [str(e) for e in v.evidence_ids]
            questions = [
                {"question": "Describe your experience with: " + req,
                 "requirement": req, "evidenceIds": eids}
                for req in job.requirements
            ]
            star_answers = []
            for eid in v.evidence_ids:
                e = await self.evidence.get(user_id, eid)
                star_answers.append({
                    "evidenceIds": [str(eid)], "situation": e.title,
                    "task": "", "action": "", "result": "",
                    "needsInput": ["task", "action", "result"]})
            research = []
            for cs in v.company_sources:
                claim = cs.source_text
                if code_point_len(claim) > 200:
                    claim = code_point_slice(claim, 0, 200)
                research.append({
                    "claim": claim, "sourceUrl": cs.source_url,
                    "accessedAt": _ts(cs.accessed_at),
                    "verificationStatus": ent.VERIFICATION_USER_PROVIDED})
            now = _now()
            op = ent.Operation(
                id=uuid4(), user_id=user_id, type=ent.OP_INTERVIEW_PREPARE,
                application_id=v.application_id, status=ent.OP_SUCCEEDED,
                result=ent.OperationResult(
                    kind=ent.OP_INTERVIEW_PREPARE,
                    value={"questions": questions, "starAnswers": star_answers,
                           "research": research}),
                created_at=now, updated_at=now)
            await self.ops.create(op)

        await self.db.do(work)
        return op
