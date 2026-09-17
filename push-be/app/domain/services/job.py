from __future__ import annotations
import re
from datetime import datetime, timezone
from typing import Optional
from uuid import UUID, uuid4

from app.db import DB, NotFoundError
from app.domain import entities as ent
from app.domain.errors import (
    internal, map_revision_err, not_found, revision_conflict, validation_field,
)
from app.domain.services.ai import AIGate
from app.domain.services.approval import ApprovalService
from app.domain.validators import validate_https_url
from app.infrastructure.store_applications import ApplicationStore
from app.infrastructure.store_evidence import EvidenceStore
from app.infrastructure.store_jobs import GapAnalysisStore, JobStore
from app.infrastructure.store_operations import OperationStore
from app.jsonutil import to_jsonable


def _now() -> datetime:
    return datetime.now(timezone.utc)


_WORD_RE = re.compile(r"[a-z0-9+#가-힣]+")


def significant_words(s: str) -> list[str]:
    return [w for w in _WORD_RE.findall(s) if len(w) >= 3]


def match_evidence(requirement: str, pool: list[ent.CareerEvidence]) -> list[str]:
    ids = []
    req = requirement.lower()
    words = significant_words(req)
    for e in pool:
        hit = any(sk and sk.lower() in req for sk in e.skills)
        if not hit:
            src = e.source_text.lower()
            hit = any(w in src for w in words)
        if hit:
            ids.append(str(e.id))
    return ids


def run_gap_analysis(j: ent.JobPosting, app_id: UUID,
                     pool: list[ent.CareerEvidence], method: str) -> ent.GapAnalysis:
    matched, missing = [], []
    for req in j.requirements:
        ids = match_evidence(req, pool)
        if ids:
            matched.append(ent.RequirementMatch(requirement=req, evidence_ids=ids))
        else:
            missing.append(req)
    preferred_missing = [p for p in j.preferred if not match_evidence(p, pool)]
    fit = len(matched) * 100 // len(j.requirements) if j.requirements else None
    return ent.GapAnalysis(
        id=uuid4(), user_id=j.user_id, application_id=app_id, job_id=j.id,
        job_revision=j.revision, evidence_ids=[e.id for e in pool],
        matched=matched, missing=missing, preferred_missing=preferred_missing,
        risks=j.risks or [], fit_score=fit, method=method, created_at=_now())


def validate_job_input(company: str, title: str, kind: str,
                       source_url: Optional[str], source_text: str) -> None:
    if not 1 <= len(company) <= 200:
        raise validation_field("company", "company must be 1-200 characters")
    if not 1 <= len(title) <= 300:
        raise validation_field("title", "title must be 1-300 characters")
    if not ent.valid_job_source_kind(kind):
        raise validation_field("sourceKind", "must be URL, TEXT or DOM")
    if kind != "TEXT" and not source_url:
        raise validation_field("sourceUrl", "required for URL/DOM sources")
    if source_url:
        try:
            validate_https_url(source_url)
        except ValueError:
            raise validation_field("sourceUrl", "must be an http(s) URL without credentials")
    if not 1 <= len(source_text) <= 200000:
        raise validation_field(
            "sourceText", "collected source text is required (1-200000 characters)")


class JobService:
    def __init__(self, db: DB, approvals: ApprovalService, ai: AIGate):
        self.db = db
        self.jobs = JobStore(db)
        self.analyses = GapAnalysisStore(db)
        self.applications = ApplicationStore(db)
        self.evidence = EvidenceStore(db)
        self.approvals = approvals
        self.ops = OperationStore(db)
        self.ai = ai

    async def list(self, user_id: UUID, archived: Optional[bool], query: str, page):
        try:
            return await self.jobs.list(user_id, archived, query, page)
        except Exception:
            raise internal()

    async def get(self, user_id: UUID, id_: UUID) -> ent.JobPosting:
        try:
            return await self.jobs.get(user_id, id_)
        except NotFoundError:
            raise not_found()
        except Exception:
            raise internal()

    async def create(self, user_id: UUID, company: str, title: str, source_kind: str,
                     source_url: Optional[str], source_text: str,
                     requirements: list[str], preferred: list[str],
                     deadline: Optional[datetime], language: str) -> ent.JobPosting:
        validate_job_input(company, title, source_kind, source_url, source_text)
        now = _now()
        j = ent.JobPosting(
            id=uuid4(), user_id=user_id, revision=1, company=company, title=title,
            source_kind=source_kind, source_url=source_url, source_text=source_text,
            requirements=requirements or [], preferred=preferred or [],
            keywords=[], risks=[], deadline=deadline, language=language,
            created_at=now, updated_at=now)
        try:
            await self.jobs.create(j)
        except Exception:
            raise internal()
        return j

    async def patch(self, user_id: UUID, id_: UUID, expected: int,
                    company: Optional[str], title: Optional[str],
                    requirements, preferred, deadline, clear_deadline: bool) -> ent.JobPosting:
        out = None

        async def work():
            nonlocal out
            try:
                j = await self.jobs.get(user_id, id_)
            except NotFoundError:
                raise not_found()
            if company is not None:
                if not 1 <= len(company) <= 200:
                    raise validation_field("company", "company must be 1-200 characters")
                j.company = company
            if title is not None:
                if not 1 <= len(title) <= 300:
                    raise validation_field("title", "title must be 1-300 characters")
                j.title = title
            if requirements is not None:
                j.requirements = requirements or []
            if preferred is not None:
                j.preferred = preferred or []
            if clear_deadline:
                j.deadline = None
            elif deadline is not None:
                j.deadline = deadline
            j.updated_at = _now()
            try:
                await self.jobs.update(j, expected)
            except Exception as err:
                raise map_revision_err(err)
            j.revision = expected + 1
            out = j

        await self.db.do(work)
        return out

    async def analyze(self, user_id: UUID, job_id: UUID, application_id: UUID,
                      expected: int, evidence_ids: list[UUID],
                      ai: Optional[ent.AiOptions]) -> ent.Operation:
        await self.ai.check(user_id, ai)
        op = None

        async def work():
            nonlocal op
            try:
                j = await self.jobs.get(user_id, job_id)
            except NotFoundError:
                raise not_found()
            if j.revision != expected:
                raise revision_conflict(j.revision)
            try:
                app = await self.applications.get(user_id, application_id)
            except NotFoundError:
                raise not_found()
            if app.job_id != job_id:
                raise validation_field(
                    "applicationId", "application does not belong to this job")
            pool = []
            for eid in evidence_ids:
                try:
                    e = await self.evidence.get(user_id, eid)
                except NotFoundError:
                    raise validation_field(
                        "evidenceIds", "evidence " + str(eid) + " not found")
                if e.archived:
                    raise validation_field(
                        "evidenceIds", "evidence " + str(eid) + " is archived")
                await self.approvals.require_evidence_use(user_id, app.id, eid)
                pool.append(e)
            method = ent.METHOD_AI_ASSISTED if ai else ent.METHOD_RULE_BASED
            analysis = run_gap_analysis(j, app.id, pool, method)
            await self.analyses.mark_stale_for_job(user_id, job_id, UUID(int=0))
            await self.analyses.create(analysis)
            now = _now()
            op = ent.Operation(
                id=uuid4(), user_id=user_id, type=ent.OP_JOB_ANALYSIS,
                application_id=app.id, status=ent.OP_SUCCEEDED,
                result=ent.OperationResult(kind=ent.OP_JOB_ANALYSIS,
                                           value=to_jsonable(analysis)),
                created_at=now, updated_at=now)
            await self.ops.create(op)

        await self.db.do(work)
        return op

    async def list_analyses(self, user_id: UUID, job_id: UUID, page):
        await self.get(user_id, job_id)
        try:
            return await self.analyses.list_by_job(user_id, job_id, page)
        except Exception:
            raise internal()
