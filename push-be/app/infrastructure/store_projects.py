from __future__ import annotations
from typing import Optional
from uuid import UUID

from app.domain.entities import CliRun, ProjectBlueprint, ProjectEvidence
from app.domain.pagination import Cursor, Page, new_page
from app.jsonutil import dump
from app.infrastructure.store_common import Conds, Store

_BP_COLS = ("id, user_id, revision, application_id, gap_analysis_id, title, skills, problem, "
            "solution, tasks, completion_criteria, metrics, estimated_effort, state, "
            "created_at, updated_at")
_RUN_COLS = ("id, user_id, revision, project_id, application_id, provider, working_directory, "
             "executable, arguments, prompt, payload_hash, state, launch_status, approval_id, "
             "device_id, detected_version, started_at, finished_at, failure_reason, exit_code, "
             "commit_sha, stdout_hash, stderr_hash, created_at, updated_at")
_PEV_COLS = ("id, user_id, revision, project_id, run_id, commit_url, commit_sha, test_results, "
             "metrics, summary, status, verification_method, verified_at, career_evidence_id, "
             "created_at, updated_at")


def _blueprint(row) -> ProjectBlueprint:
    b = dict(row)
    for k in ("skills", "tasks", "completion_criteria", "metrics"):
        b[k] = b.get(k) or []
    b["estimated_effort"] = b.get("estimated_effort") or {}
    return ProjectBlueprint(**b)


def _run(row) -> CliRun:
    r = dict(row)
    r["arguments"] = r.get("arguments") or []
    return CliRun(**r)


def _pev(row) -> ProjectEvidence:
    e = dict(row)
    e["test_results"] = e.get("test_results") or {}
    e["metrics"] = e.get("metrics") or []
    return ProjectEvidence(**e)


class ProjectStore(Store):
    async def create_blueprint(self, b: ProjectBlueprint) -> None:
        await self.q().execute(
            f"INSERT INTO project_blueprints ({_BP_COLS}) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)",
            b.id, b.user_id, b.revision, b.application_id, b.gap_analysis_id, b.title,
            dump(b.skills), b.problem, b.solution, dump(b.tasks), dump(b.completion_criteria),
            dump(b.metrics), dump(b.estimated_effort), b.state, b.created_at, b.updated_at)

    async def get_blueprint(self, user_id: UUID, id_: UUID) -> ProjectBlueprint:
        return _blueprint(await self.one(
            f"SELECT {_BP_COLS} FROM project_blueprints WHERE id = $1 AND user_id = $2",
            id_, user_id))

    async def list_blueprints(self, user_id: UUID, application_id: Optional[UUID], page) -> Page:
        c = Conds()
        c.add("user_id = {}", user_id)
        if application_id:
            c.add("application_id = {}", application_id)
        c.cursor(page)
        sql, args = c.query(_BP_COLS, "project_blueprints", page.effective_limit())
        rows = await self.q().fetch(sql, *args)
        return new_page([_blueprint(r) for r in rows], page.effective_limit(),
                        lambda b: Cursor(b.created_at, b.id))

    async def update_blueprint(self, b: ProjectBlueprint, expected: int) -> None:
        await self.guard_update(
            "project_blueprints", b.id, b.user_id,
            "UPDATE project_blueprints SET revision = revision + 1, title = $3, skills = $4, "
            "problem = $5, solution = $6, tasks = $7, completion_criteria = $8, metrics = $9, "
            "estimated_effort = $10, state = $11, updated_at = $12 "
            "WHERE id = $1 AND user_id = $2 AND revision = $13",
            b.id, b.user_id, b.title, dump(b.skills), b.problem, b.solution, dump(b.tasks),
            dump(b.completion_criteria), dump(b.metrics), dump(b.estimated_effort), b.state,
            b.updated_at, expected)

    async def create_run(self, run: CliRun) -> None:
        await self.q().execute(
            f"INSERT INTO cli_runs ({_RUN_COLS}) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25)",
            run.id, run.user_id, run.revision, run.project_id, run.application_id, run.provider,
            run.working_directory, run.executable, dump(run.arguments), run.prompt,
            run.payload_hash, run.state, run.launch_status, run.approval_id, run.device_id,
            run.detected_version, run.started_at, run.finished_at, run.failure_reason,
            run.exit_code, run.commit_sha, run.stdout_hash, run.stderr_hash,
            run.created_at, run.updated_at)

    async def get_run(self, user_id: UUID, project_id: UUID, run_id: UUID) -> CliRun:
        sql = f"SELECT {_RUN_COLS} FROM cli_runs WHERE id = $1 AND user_id = $2"
        args: list = [run_id, user_id]
        if project_id != UUID(int=0):
            sql += " AND project_id = $3"
            args.append(project_id)
        return _run(await self.one(sql, *args))

    async def get_run_by_id(self, user_id: UUID, run_id: UUID) -> CliRun:
        return _run(await self.one(
            f"SELECT {_RUN_COLS} FROM cli_runs WHERE id = $1 AND user_id = $2",
            run_id, user_id))

    async def list_runs(self, user_id: UUID, project_id: UUID, page) -> Page:
        c = Conds()
        c.add("user_id = {}", user_id)
        c.add("project_id = {}", project_id)
        c.cursor(page)
        sql, args = c.query(_RUN_COLS, "cli_runs", page.effective_limit())
        rows = await self.q().fetch(sql, *args)
        return new_page([_run(r) for r in rows], page.effective_limit(),
                        lambda r: Cursor(r.created_at, r.id))

    async def update_run(self, run: CliRun, expected: int) -> None:
        await self.guard_update(
            "cli_runs", run.id, run.user_id,
            "UPDATE cli_runs SET revision = revision + 1, state = $3, launch_status = $4, "
            "approval_id = $5, device_id = $6, detected_version = $7, started_at = $8, "
            "finished_at = $9, failure_reason = $10, exit_code = $11, commit_sha = $12, "
            "stdout_hash = $13, stderr_hash = $14, updated_at = $15 "
            "WHERE id = $1 AND user_id = $2 AND revision = $16",
            run.id, run.user_id, run.state, run.launch_status, run.approval_id, run.device_id,
            run.detected_version, run.started_at, run.finished_at, run.failure_reason,
            run.exit_code, run.commit_sha, run.stdout_hash, run.stderr_hash,
            run.updated_at, expected)

    async def create_evidence(self, e: ProjectEvidence) -> None:
        await self.q().execute(
            f"INSERT INTO project_evidence ({_PEV_COLS}) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)",
            e.id, e.user_id, e.revision, e.project_id, e.run_id, e.commit_url, e.commit_sha,
            dump(e.test_results), dump(e.metrics), e.summary, e.status, e.verification_method,
            e.verified_at, e.career_evidence_id, e.created_at, e.updated_at)

    async def get_evidence(self, user_id: UUID, project_id: UUID, evidence_id: UUID) -> ProjectEvidence:
        return _pev(await self.one(
            f"SELECT {_PEV_COLS} FROM project_evidence WHERE id = $1 AND user_id = $2 "
            "AND project_id = $3", evidence_id, user_id, project_id))

    async def list_evidence(self, user_id: UUID, project_id: UUID, page) -> Page:
        c = Conds()
        c.add("user_id = {}", user_id)
        c.add("project_id = {}", project_id)
        c.cursor(page)
        sql, args = c.query(_PEV_COLS, "project_evidence", page.effective_limit())
        rows = await self.q().fetch(sql, *args)
        return new_page([_pev(r) for r in rows], page.effective_limit(),
                        lambda e: Cursor(e.created_at, e.id))

    async def update_evidence(self, e: ProjectEvidence, expected: int) -> None:
        await self.guard_update(
            "project_evidence", e.id, e.user_id,
            "UPDATE project_evidence SET revision = revision + 1, commit_url = $3, "
            "commit_sha = $4, test_results = $5, metrics = $6, summary = $7, status = $8, "
            "verification_method = $9, verified_at = $10, career_evidence_id = $11, "
            "updated_at = $12 WHERE id = $1 AND user_id = $2 AND revision = $13",
            e.id, e.user_id, e.commit_url, e.commit_sha, dump(e.test_results), dump(e.metrics),
            e.summary, e.status, e.verification_method, e.verified_at, e.career_evidence_id,
            e.updated_at, expected)
