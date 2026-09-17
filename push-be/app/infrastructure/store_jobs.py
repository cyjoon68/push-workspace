from __future__ import annotations
from typing import Optional
from uuid import UUID

from app.domain.entities import GapAnalysis, JobPosting
from app.domain.pagination import Cursor, Page
from app.jsonutil import dump
from app.infrastructure.store_common import Conds, Store

_JOB_COLS = ("id, user_id, revision, company, title, source_kind, source_url, source_text, "
             "requirements, preferred, keywords, risks, deadline, language, archived, "
             "created_at, updated_at")
_ANALYSIS_COLS = ("id, user_id, application_id, job_id, job_revision, evidence_ids, matched, "
                  "missing, preferred_missing, risks, fit_score, method, stale, created_at")


def _job(row) -> JobPosting:
    j = dict(row)
    for k in ("requirements", "preferred", "keywords", "risks"):
        j[k] = j.get(k) or []
    return JobPosting(**j)


class JobStore(Store):
    async def create(self, j: JobPosting) -> None:
        await self.q().execute(
            f"INSERT INTO jobs ({_JOB_COLS}) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)",
            j.id, j.user_id, j.revision, j.company, j.title, j.source_kind, j.source_url,
            j.source_text, dump(j.requirements), dump(j.preferred), dump(j.keywords),
            dump(j.risks), j.deadline.date() if j.deadline else None, j.language, j.archived, j.created_at, j.updated_at)

    async def get(self, user_id: UUID, id_: UUID) -> JobPosting:
        return _job(await self.one(
            f"SELECT {_JOB_COLS} FROM jobs WHERE id = $1 AND user_id = $2", id_, user_id))

    async def list(self, user_id: UUID, archived: Optional[bool], query: str, page) -> Page:
        c = Conds()
        c.add("user_id = {}", user_id)
        if archived is not None:
            c.add("archived = {}", archived)
        if query:
            c.add("(company ILIKE {} OR title ILIKE {})", "%" + query + "%", "%" + query + "%")
        c.cursor(page)
        sql, args = c.query(_JOB_COLS, "jobs", page.effective_limit())
        rows = await self.q().fetch(sql, *args)
        items = [_job(r) for r in rows]
        from app.domain.pagination import new_page
        return new_page(items, page.effective_limit(), lambda j: Cursor(j.created_at, j.id))

    async def update(self, j: JobPosting, expected: int) -> None:
        await self.guard_update(
            "jobs", j.id, j.user_id,
            "UPDATE jobs SET revision = revision + 1, company = $3, title = $4, "
            "source_kind = $5, source_url = $6, source_text = $7, requirements = $8, "
            "preferred = $9, keywords = $10, risks = $11, deadline = $12, language = $13, "
            "archived = $14, updated_at = $15 "
            "WHERE id = $1 AND user_id = $2 AND revision = $16",
            j.id, j.user_id, j.company, j.title, j.source_kind, j.source_url, j.source_text,
            dump(j.requirements), dump(j.preferred), dump(j.keywords), dump(j.risks),
            j.deadline.date() if j.deadline else None, j.language, j.archived, j.updated_at, expected)


def _analysis(row) -> GapAnalysis:
    g = dict(row)
    for k in ("evidence_ids", "matched", "missing", "preferred_missing", "risks"):
        g[k] = g.get(k) or []
    return GapAnalysis(**g)


class GapAnalysisStore(Store):
    async def create(self, g: GapAnalysis) -> None:
        await self.q().execute(
            f"INSERT INTO gap_analyses ({_ANALYSIS_COLS}) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)",
            g.id, g.user_id, g.application_id, g.job_id, g.job_revision,
            dump(g.evidence_ids), dump(g.matched), dump(g.missing), dump(g.preferred_missing),
            dump(g.risks), g.fit_score, g.method, g.stale, g.created_at)

    async def get(self, user_id: UUID, id_: UUID) -> GapAnalysis:
        return _analysis(await self.one(
            f"SELECT {_ANALYSIS_COLS} FROM gap_analyses WHERE id = $1 AND user_id = $2",
            id_, user_id))

    async def list_by_job(self, user_id: UUID, job_id: UUID, page) -> Page:
        c = Conds()
        c.add("user_id = {}", user_id)
        c.add("job_id = {}", job_id)
        c.cursor(page)
        sql, args = c.query(_ANALYSIS_COLS, "gap_analyses", page.effective_limit())
        rows = await self.q().fetch(sql, *args)
        items = [_analysis(r) for r in rows]
        from app.domain.pagination import new_page
        return new_page(items, page.effective_limit(), lambda g: Cursor(g.created_at, g.id))

    async def mark_stale_for_job(self, user_id: UUID, job_id: UUID, except_id: UUID) -> None:
        await self.q().execute(
            "UPDATE gap_analyses SET stale = true WHERE user_id = $1 AND job_id = $2 AND id <> $3",
            user_id, job_id, except_id)

    async def count_by_application(self, user_id: UUID, application_id: UUID) -> int:
        return await self.q().fetchval(
            "SELECT count(*) FROM gap_analyses WHERE user_id = $1 AND application_id = $2",
            user_id, application_id)
