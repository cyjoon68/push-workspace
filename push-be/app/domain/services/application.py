from __future__ import annotations
from datetime import datetime, timezone
from typing import Optional
from uuid import UUID, uuid4

from app.db import DB, NotFoundError
from app.domain import entities as ent
from app.domain.errors import (
    document_not_finalized, feature_disabled, internal, invalid_transition,
    map_revision_err, not_found, revision_conflict, validation_field,
    approval_stale,
)
from app.domain.services.approval import ApprovalService, hash_json
from app.infrastructure.store_applications import ApplicationStore, SubmissionStore
from app.infrastructure.store_calendar import InterviewStore
from app.infrastructure.store_documents import DocumentStore
from app.infrastructure.store_evidence import EvidenceStore
from app.infrastructure.store_jobs import GapAnalysisStore, JobStore

SUBMISSION_MANUAL = "MANUAL_RECORD"
SUBMISSION_ADAPTER = "ADAPTER"
SUBMISSION_SUCCEEDED = "SUCCEEDED"


def _now() -> datetime:
    return datetime.now(timezone.utc)


class ApplicationService:
    def __init__(self, db: DB, approvals: ApprovalService, adapters: dict):
        self.db = db
        self.applications = ApplicationStore(db)
        self.jobs = JobStore(db)
        self.documents = DocumentStore(db)
        self.analyses = GapAnalysisStore(db)
        self.evidence = EvidenceStore(db)
        self.interviews = InterviewStore(db)
        self.submissions = SubmissionStore(db)
        self.approvals = approvals
        self.adapters = adapters or {}

    async def list(self, user_id: UUID, stage: Optional[str], query: str, page):
        try:
            return await self.applications.list(user_id, stage, query, page)
        except Exception:
            raise internal()

    async def get(self, user_id: UUID, id_: UUID) -> ent.Application:
        try:
            return await self.applications.get(user_id, id_)
        except NotFoundError:
            raise not_found()
        except Exception:
            raise internal()

    async def create(self, user_id: UUID, job_id: UUID, notes: str) -> ent.Application:
        try:
            j = await self.jobs.get(user_id, job_id)
        except NotFoundError:
            raise not_found()
        except Exception:
            raise internal()
        now = _now()
        a = ent.Application(
            id=uuid4(), user_id=user_id, revision=1, job_id=job_id, company=j.company,
            title=j.title, stage="DISCOVERED", notes=notes, created_at=now, updated_at=now)
        try:
            await self.applications.create(a)
        except Exception:
            raise internal()
        return a

    async def patch(self, user_id: UUID, id_: UUID, expected: int,
                    stage: Optional[str], notes: Optional[str],
                    next_action_at, clear_next_action: bool) -> ent.Application:
        out = None

        async def work():
            nonlocal out
            try:
                a = await self.applications.get(user_id, id_)
            except NotFoundError:
                raise not_found()
            if a.revision != expected:
                raise revision_conflict(a.revision)
            now = _now()
            if stage is not None:
                to = stage
                if not ent.valid_stage(to):
                    raise validation_field("stage", "unsupported stage")
                if to == ent.STAGE_APPLIED and a.stage != ent.STAGE_APPLIED:
                    raise invalid_transition(
                        "APPLIED can only be reached through submissions")
                if not ent.can_patch_stage(a.stage, to):
                    raise invalid_transition(
                        f"cannot move from {a.stage} to {to}")
                if to != a.stage:
                    from_ = a.stage
                    a.stage = to
                    if to == ent.STAGE_APPLIED:
                        a.applied_at = now
                    await self.applications.add_event(ent.ApplicationEvent(
                        id=uuid4(), user_id=user_id, application_id=a.id,
                        type=ent.EVENT_STAGE_CHANGED,
                        payload={"from": from_, "to": to}, created_at=now))
            if notes is not None:
                a.notes = notes
            if clear_next_action:
                a.next_action_at = None
            elif next_action_at is not None:
                a.next_action_at = next_action_at
            a.updated_at = now
            try:
                await self.applications.update(a, expected)
            except Exception as err:
                raise map_revision_err(err)
            a.revision = expected + 1
            out = a

        await self.db.do(work)
        return out

    async def import_(self, user_id: UUID, job_id: UUID, stage: str,
                      applied_at: datetime, notes: str,
                      confirmed: bool) -> ent.Application:
        if not confirmed:
            raise validation_field(
                "confirmed", "must be true to record a past application")
        if not ent.valid_stage(stage) or stage == "DISCOVERED":
            raise validation_field("stage", "unsupported stage for import")
        try:
            j = await self.jobs.get(user_id, job_id)
        except NotFoundError:
            raise not_found()
        except Exception:
            raise internal()
        now = _now()
        a = ent.Application(
            id=uuid4(), user_id=user_id, revision=1, job_id=job_id, company=j.company,
            title=j.title, stage=stage, notes=notes, applied_at=applied_at,
            imported=True, created_at=now, updated_at=now)

        async def work():
            await self.applications.create(a)
            await self.applications.add_event(ent.ApplicationEvent(
                id=uuid4(), user_id=user_id, application_id=a.id,
                type=ent.EVENT_IMPORTED,
                payload={"stage": stage, "appliedAt": applied_at}, created_at=now))

        try:
            await self.db.do(work)
        except Exception:
            raise internal()
        return a

    async def timeline(self, user_id: UUID, id_: UUID, page):
        await self.get(user_id, id_)
        try:
            return await self.applications.list_events(user_id, id_, page)
        except Exception:
            raise internal()

    async def checklist(self, user_id: UUID, id_: UUID) -> dict:
        a = await self.get(user_id, id_)
        evidence_count = await self.evidence.count(user_id)
        analysis_count = await self.analyses.count_by_application(user_id, a.id)
        doc_count = await self.documents.count_finalized_by_application(user_id, a.id)
        interview_count = await self.interviews.count_by_application(user_id, a.id)
        applied = (a.applied_at is not None or ent.terminal_stage(a.stage)
                   or a.stage == ent.STAGE_APPLIED or ent.stage_after_applied(a.stage))
        return {
            "items": [
                {"key": "evidence", "label": "Career evidence collected",
                 "completed": evidence_count > 0},
                {"key": "analysis", "label": "Gap analysis completed",
                 "completed": analysis_count > 0},
                {"key": "document_finalized", "label": "Document finalized",
                 "completed": doc_count > 0},
                {"key": "applied", "label": "Application submitted",
                 "completed": applied},
                {"key": "interview_prep", "label": "Interview session scheduled",
                 "completed": interview_count > 0},
            ],
            "automationEnabled": False,
        }

    async def create_draft(self, user_id: UUID, app_id: UUID, expected: int,
                           mode: str, adapter: Optional[str],
                           document_version_ids: list[UUID],
                           confirmed_submitted: bool) -> ent.SubmissionDraft:
        out = None

        async def work():
            nonlocal out
            try:
                a = await self.applications.get(user_id, app_id)
            except NotFoundError:
                raise not_found()
            if a.revision != expected:
                raise revision_conflict(a.revision)
            if mode not in (SUBMISSION_MANUAL, SUBMISSION_ADAPTER):
                raise validation_field("mode", "must be MANUAL_RECORD or ADAPTER")
            if mode == SUBMISSION_ADAPTER:
                if not adapter or not self.adapters.get(adapter):
                    raise feature_disabled(
                        "adapter submission is not enabled for this site")
            elif adapter is not None:
                raise validation_field(
                    "adapter", "adapter is only valid with ADAPTER mode")
            if not document_version_ids:
                raise validation_field(
                    "documentVersionIds", "at least one document version is required")
            nil = UUID(int=0)
            for vid in document_version_ids:
                try:
                    v = await self.documents.get_version(user_id, nil, vid)
                except Exception:
                    raise validation_field(
                        "documentVersionIds", "version " + str(vid) + " not found")
                if v.application_id != app_id:
                    raise validation_field(
                        "documentVersionIds",
                        "version " + str(vid) + " belongs to another application")
                try:
                    doc = await self.documents.get(user_id, v.document_id)
                except Exception:
                    raise internal()
                if doc.finalized_version_id is None or doc.finalized_version_id != vid:
                    raise document_not_finalized(
                        "document version " + str(vid) + " is not finalized")
            d = ent.SubmissionDraft(
                id=uuid4(), user_id=user_id, application_id=app_id,
                application_revision=a.revision, mode=mode, adapter=adapter,
                document_version_ids=document_version_ids,
                confirmed_submitted=confirmed_submitted,
                payload_hash=hash_json({
                    "applicationId": app_id, "applicationRevision": a.revision,
                    "mode": mode, "adapter": adapter,
                    "documentVersionIds": document_version_ids}),
                created_at=_now())
            await self.submissions.create_draft(d)
            out = d

        await self.db.do(work)
        return out

    async def submit(self, user_id: UUID, app_id: UUID, expected: int,
                     draft_id: UUID, approval_id: UUID) -> ent.Submission:
        out = None

        async def work():
            nonlocal out
            try:
                a = await self.applications.get(user_id, app_id)
            except NotFoundError:
                raise not_found()
            if a.revision != expected:
                raise revision_conflict(a.revision)
            try:
                d = await self.submissions.get_draft(user_id, draft_id)
            except Exception:
                raise validation_field("draftId", "draft not found")
            if d.application_id != app_id:
                raise validation_field(
                    "draftId", "draft belongs to another application")
            if d.application_revision != a.revision:
                raise approval_stale(
                    "application changed since the draft was created; "
                    "create a new draft and approval")
            await self.approvals.consume(
                user_id, approval_id, ent.APPROVAL_APPLICATION_SUBMIT,
                app_id, d.id, d.payload_hash)
            if d.mode == SUBMISSION_MANUAL and not d.confirmed_submitted:
                raise validation_field(
                    "confirmedSubmitted",
                    "manual submission requires confirming the external submission completed")
            now = _now()
            sub = ent.Submission(
                id=uuid4(), user_id=user_id, application_id=app_id,
                mode=d.mode, adapter=d.adapter, status=SUBMISSION_SUCCEEDED,
                document_version_ids=d.document_version_ids, approval_id=approval_id,
                created_at=now, updated_at=now)
            await self.submissions.create(sub)
            if ent.terminal_stage(a.stage):
                raise invalid_transition("application is closed")
            if (a.stage != ent.STAGE_APPLIED
                    and not ent.stage_after_applied(a.stage)):
                from_ = a.stage
                a.stage = ent.STAGE_APPLIED
                a.applied_at = now
                a.updated_at = now
                try:
                    await self.applications.update(a, expected)
                except Exception as err:
                    raise map_revision_err(err)
                a.revision = expected + 1
                await self.applications.add_event(ent.ApplicationEvent(
                    id=uuid4(), user_id=user_id, application_id=a.id,
                    type=ent.EVENT_STAGE_CHANGED,
                    payload={"from": from_, "to": ent.STAGE_APPLIED}, created_at=now))
            await self.applications.add_event(ent.ApplicationEvent(
                id=uuid4(), user_id=user_id, application_id=a.id,
                type=ent.EVENT_SUBMISSION,
                payload={"submissionId": sub.id, "mode": d.mode, "status": sub.status},
                created_at=now))
            out = sub

        await self.db.do(work)
        return out

    async def list_submissions(self, user_id: UUID, app_id: UUID, page):
        await self.get(user_id, app_id)
        try:
            return await self.submissions.list(user_id, app_id, page)
        except Exception:
            raise internal()
