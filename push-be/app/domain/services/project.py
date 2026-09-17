from __future__ import annotations
import posixpath
from datetime import datetime, timezone
from typing import Optional
from uuid import UUID, uuid4

from app.db import DB, NotFoundError
from app.domain import entities as ent
from app.domain.errors import (
    approval_stale, internal, invalid_transition, map_revision_err, not_found,
    revision_conflict, validation_field,
)
from app.domain.services.ai import AIGate
from app.domain.services.approval import ApprovalService, hash_json
from app.domain.validators import hash_bytes, validate_https_url
from app.infrastructure.store_applications import ApplicationStore
from app.infrastructure.store_jobs import GapAnalysisStore
from app.infrastructure.store_operations import OperationStore
from app.infrastructure.store_projects import ProjectStore
from app.jsonutil import to_jsonable

RUN_APPROVAL_REQUIRED = "APPROVAL_REQUIRED"
RUN_RUNNING = "RUNNING"
RUN_VERIFYING = "VERIFYING"
RUN_VERIFIED = "VERIFIED"
RUN_FAILED = "FAILED"
PEV_PENDING = "PENDING"
PEV_VERIFIED = "VERIFIED"


def _now() -> datetime:
    return datetime.now(timezone.utc)


def build_blueprints(user_id: UUID, application_id: UUID, gap_analysis_id: UUID,
                     a: ent.GapAnalysis, now: datetime) -> list[ent.ProjectBlueprint]:
    gaps = list(a.missing) + list(a.preferred_missing) or ["general engineering practice"]
    out = []
    for i in range(4):
        skill = gaps[i % len(gaps)]
        out.append(ent.ProjectBlueprint(
            id=uuid4(), user_id=user_id, revision=1, application_id=application_id,
            gap_analysis_id=gap_analysis_id,
            title=f"Gap project {i + 1}: {skill}", skills=[skill],
            problem="Missing or unproven requirement: " + skill,
            solution="Build a small project that exercises " + skill,
            tasks=[ent.BlueprintTask(
                id="task-1", title="Implement core feature using " + skill,
                description="Create a minimal but working implementation",
                acceptance=["runs locally", "tests pass"])],
            completion_criteria=["committed to a remote repository",
                                 "tests pass in CI"],
            metrics=[ent.BlueprintMetric(name="test_coverage", unit="percent",
                                         measurement="coverage report", target=None)],
            estimated_effort=ent.EffortEstimate(min_hours=4, max_hours=16),
            state=ent.BLUEPRINT_DRAFT, created_at=now, updated_at=now))
    return out


def valid_manifest_path(p: str) -> bool:
    if not p or p.startswith("/") or p.startswith("\\"):
        return False
    clean = posixpath.normpath(p)
    if clean != p or clean.startswith("..") or "/../" in clean:
        return False
    return True


def build_manifest(b: ent.ProjectBlueprint) -> dict:
    readme = ("# " + b.title + "\n\n## Problem\n" + b.problem +
              "\n\n## Solution\n" + b.solution + "\n")
    tasks = ""
    for t in b.tasks:
        tasks += "## " + t.title + "\n" + t.description + "\n\nAcceptance:\n"
        for a in t.acceptance:
            tasks += "- " + a + "\n"
        tasks += "\n"
    files = []
    for path_, content in (("README.md", readme), ("docs/tasks.md", tasks)):
        if not valid_manifest_path(path_):
            raise internal()
        files.append({"path": path_, "encoding": "utf8", "content": content,
                      "sha256": hash_bytes(content.encode())})
    return {"projectId": b.id, "blueprintRevision": b.revision, "files": files}


def run_payload_hash(provider: str, dir_: str, executable: str,
                     args: list[str], prompt: str) -> str:
    return hash_json({"provider": provider, "workingDirectory": dir_,
                      "executable": executable, "arguments": args, "prompt": prompt})


class ProjectService:
    def __init__(self, db: DB, approvals: ApprovalService, ai: AIGate):
        self.db = db
        self.projects = ProjectStore(db)
        self.applications = ApplicationStore(db)
        self.analyses = GapAnalysisStore(db)
        self.approvals = approvals
        self.ops = OperationStore(db)
        self.ai = ai

    async def list_blueprints(self, user_id: UUID, application_id: Optional[UUID], page):
        try:
            return await self.projects.list_blueprints(user_id, application_id, page)
        except Exception:
            raise internal()

    async def get_blueprint(self, user_id: UUID, id_: UUID) -> ent.ProjectBlueprint:
        try:
            return await self.projects.get_blueprint(user_id, id_)
        except NotFoundError:
            raise not_found()
        except Exception:
            raise internal()

    async def generate_blueprints(self, user_id: UUID, application_id: UUID,
                                  gap_analysis_id: UUID,
                                  ai: Optional[ent.AiOptions]) -> ent.Operation:
        await self.ai.check(user_id, ai)
        op = None

        async def work():
            nonlocal op
            try:
                app = await self.applications.get(user_id, application_id)
            except NotFoundError:
                raise not_found()
            try:
                analysis = await self.analyses.get(user_id, gap_analysis_id)
            except Exception:
                raise not_found()
            if analysis.application_id != application_id:
                raise validation_field(
                    "gapAnalysisId", "analysis belongs to another application")
            now = _now()
            blueprints = build_blueprints(
                user_id, application_id, gap_analysis_id, analysis, now)
            for b in blueprints:
                await self.projects.create_blueprint(b)
            op = ent.Operation(
                id=uuid4(), user_id=user_id, type=ent.OP_PROJECT_BLUEPRINTS,
                application_id=app.id, status=ent.OP_SUCCEEDED,
                result=ent.OperationResult(
                    kind=ent.OP_PROJECT_BLUEPRINTS,
                    value=[to_jsonable(b) for b in blueprints]),
                created_at=now, updated_at=now)
            await self.ops.create(op)

        await self.db.do(work)
        return op

    async def select(self, user_id: UUID, id_: UUID,
                     expected: int) -> tuple[ent.ProjectBlueprint, dict]:
        out = manifest = None

        async def work():
            nonlocal out, manifest
            try:
                b = await self.projects.get_blueprint(user_id, id_)
            except NotFoundError:
                raise not_found()
            if b.revision != expected:
                raise revision_conflict(b.revision)
            if b.state not in (ent.BLUEPRINT_DRAFT, ent.BLUEPRINT_SELECTED):
                raise invalid_transition(
                    "blueprint cannot be selected from " + b.state)
            b.state = ent.BLUEPRINT_SELECTED
            b.updated_at = _now()
            try:
                await self.projects.update_blueprint(b, expected)
            except Exception as err:
                raise map_revision_err(err)
            b.revision = expected + 1
            manifest = build_manifest(b)
            manifest["blueprintRevision"] = b.revision
            out = b

        await self.db.do(work)
        return out, manifest

    async def list_runs(self, user_id: UUID, project_id: UUID, page):
        await self.get_blueprint(user_id, project_id)
        try:
            return await self.projects.list_runs(user_id, project_id, page)
        except Exception:
            raise internal()

    async def get_run(self, user_id: UUID, project_id: UUID,
                      run_id: UUID) -> ent.CliRun:
        try:
            return await self.projects.get_run(user_id, project_id, run_id)
        except NotFoundError:
            raise not_found()
        except Exception:
            raise internal()

    async def create_run(self, user_id: UUID, project_id: UUID, provider: str,
                         working_directory: str, prompt: str) -> ent.CliRun:
        if not ent.valid_cli_provider(provider):
            raise validation_field("provider", "unsupported provider")
        if not working_directory or not posixpath.isabs(working_directory):
            raise validation_field("workingDirectory", "absolute path required")
        if not prompt:
            raise validation_field("prompt", "required")
        out = None

        async def work():
            nonlocal out
            try:
                b = await self.projects.get_blueprint(user_id, project_id)
            except NotFoundError:
                raise not_found()
            if b.state not in (ent.BLUEPRINT_SELECTED, ent.BLUEPRINT_IN_PROGRESS):
                raise invalid_transition(
                    "blueprint must be selected before creating a run")
            executable = ent.CLI_EXECUTABLES[provider]
            args: list[str] = []
            now = _now()
            r = ent.CliRun(
                id=uuid4(), user_id=user_id, revision=1, project_id=project_id,
                application_id=b.application_id, provider=provider,
                working_directory=working_directory, executable=executable,
                arguments=args, prompt=prompt,
                payload_hash=run_payload_hash(
                    provider, working_directory, executable, args, prompt),
                state=RUN_APPROVAL_REQUIRED, launch_status=ent.LAUNCH_NOT_CLAIMED,
                created_at=now, updated_at=now)
            await self.projects.create_run(r)
            if b.state == ent.BLUEPRINT_SELECTED:
                b.state = ent.BLUEPRINT_IN_PROGRESS
                b.updated_at = now
                try:
                    await self.projects.update_blueprint(b, b.revision)
                except Exception as err:
                    raise map_revision_err(err)
            out = r

        await self.db.do(work)
        return out

    async def start_run(self, user_id: UUID, project_id: UUID, run_id: UUID,
                        expected: int, approval_id: UUID,
                        detected_version: str, device_id: str) -> ent.CliRun:
        if not device_id:
            raise validation_field("deviceId", "required")
        if not detected_version:
            raise validation_field("detectedVersion", "required")
        out = None

        async def work():
            nonlocal out
            r = await self.get_run(user_id, project_id, run_id)
            if r.revision != expected:
                raise revision_conflict(r.revision)
            if r.state != RUN_APPROVAL_REQUIRED:
                raise invalid_transition("run cannot start from " + r.state)
            ap = await self.approvals.consume(
                user_id, approval_id, ent.APPROVAL_CLI_EXECUTE,
                r.application_id, r.id, r.payload_hash)
            if (ap.target_revision is not None
                    and ap.target_revision != r.revision):
                raise approval_stale(
                    "approval was issued for a different run revision")
            now = _now()
            r.state = RUN_RUNNING
            r.approval_id = approval_id
            r.device_id = device_id
            r.detected_version = detected_version
            r.started_at = now
            r.updated_at = now
            try:
                await self.projects.update_run(r, expected)
            except Exception as err:
                raise map_revision_err(err)
            r.revision = expected + 1
            out = r

        await self.db.do(work)
        return out

    async def report_launch(self, user_id: UUID, project_id: UUID, run_id: UUID,
                            expected: int, device_id: str, payload_hash: str,
                            launch_status: str,
                            process: Optional[ent.ProcessInfo]) -> ent.CliRun:
        if not ent.valid_reported_launch_status(launch_status):
            raise validation_field(
                "launchStatus", "must be CLAIMED, STARTED or UNKNOWN")
        out = None

        async def work():
            nonlocal out
            r = await self.get_run(user_id, project_id, run_id)
            if r.revision != expected:
                raise revision_conflict(r.revision)
            if r.state != RUN_RUNNING:
                raise invalid_transition("run is not running")
            if r.device_id is None or r.device_id != device_id:
                raise invalid_transition("run is bound to a different device")
            if payload_hash != r.payload_hash:
                raise approval_stale(
                    "payload hash does not match the approved run")
            if launch_status == ent.LAUNCH_STARTED and process is None:
                raise validation_field("process", "process info required for STARTED")
            if not ent.can_transition_launch(r.launch_status, launch_status):
                raise invalid_transition(
                    f"launchStatus cannot move from {r.launch_status} to {launch_status}")
            r.launch_status = launch_status
            r.updated_at = _now()
            try:
                await self.projects.update_run(r, expected)
            except Exception as err:
                raise map_revision_err(err)
            r.revision = expected + 1
            out = r

        await self.db.do(work)
        return out

    async def recover(self, user_id: UUID, project_id: UUID, run_id: UUID,
                      expected: int, device_id: str, decision: str,
                      process: Optional[ent.ProcessInfo],
                      failure_reason: Optional[str]) -> ent.CliRun:
        if decision not in ("REATTACH", "MARK_FAILED"):
            raise validation_field("decision", "must be REATTACH or MARK_FAILED")
        out = None

        async def work():
            nonlocal out
            r = await self.get_run(user_id, project_id, run_id)
            if r.revision != expected:
                raise revision_conflict(r.revision)
            if r.launch_status != ent.LAUNCH_UNKNOWN:
                raise invalid_transition(
                    "recover is only allowed from UNKNOWN launch status")
            if r.device_id is None or r.device_id != device_id:
                raise invalid_transition("run is bound to a different device")
            now = _now()
            if decision == "REATTACH":
                if process is None or process.pid <= 0:
                    raise validation_field(
                        "process", "process info required for REATTACH")
                r.launch_status = ent.LAUNCH_STARTED
            else:
                if not failure_reason:
                    raise validation_field(
                        "failureReason", "required for MARK_FAILED")
                r.launch_status = ent.LAUNCH_FINISHED
                r.state = RUN_FAILED
                r.failure_reason = failure_reason
                r.finished_at = now
            r.updated_at = now
            try:
                await self.projects.update_run(r, expected)
            except Exception as err:
                raise map_revision_err(err)
            r.revision = expected + 1
            out = r

        await self.db.do(work)
        return out

    async def report_result(self, user_id: UUID, project_id: UUID, run_id: UUID,
                            expected: int, exit_code: int,
                            commit_sha: Optional[str], stdout_hash: str,
                            stderr_hash: str) -> ent.CliRun:
        out = None

        async def work():
            nonlocal out
            r = await self.get_run(user_id, project_id, run_id)
            if r.revision != expected:
                raise revision_conflict(r.revision)
            if r.state != RUN_RUNNING or r.launch_status != ent.LAUNCH_STARTED:
                raise invalid_transition(
                    "result requires a RUNNING run with STARTED launch status")
            now = _now()
            r.launch_status = ent.LAUNCH_FINISHED
            r.exit_code = exit_code
            r.commit_sha = commit_sha
            r.stdout_hash = stdout_hash
            r.stderr_hash = stderr_hash
            r.finished_at = now
            if exit_code == 0:
                r.state = RUN_VERIFYING
            else:
                r.state = RUN_FAILED
                r.failure_reason = f"exit code {exit_code}"
            r.updated_at = now
            try:
                await self.projects.update_run(r, expected)
            except Exception as err:
                raise map_revision_err(err)
            r.revision = expected + 1
            out = r

        await self.db.do(work)
        return out

    async def create_evidence(self, user_id: UUID, project_id: UUID, run_id: UUID,
                              commit_url: str, test_command: str, test_output: str,
                              exit_code: int, metrics: list[ent.MetricEntry],
                              summary: str) -> ent.ProjectEvidence:
        try:
            validate_https_url(commit_url)
        except ValueError:
            raise validation_field("commitUrl", "must be an http(s) URL")
        out = None

        async def work():
            nonlocal out
            try:
                await self.projects.get_blueprint(user_id, project_id)
            except NotFoundError:
                raise not_found()
            try:
                r = await self.projects.get_run(user_id, project_id, run_id)
            except Exception:
                raise validation_field("runId", "run not found")
            if r.state not in (RUN_VERIFYING, RUN_VERIFIED):
                raise invalid_transition(
                    "run must be VERIFYING before evidence is submitted")
            now = _now()
            e = ent.ProjectEvidence(
                id=uuid4(), user_id=user_id, revision=1, project_id=project_id,
                run_id=run_id, commit_url=commit_url, commit_sha=r.commit_sha,
                test_results=ent.TestResults(
                    test_command=test_command, test_output=test_output,
                    exit_code=exit_code),
                metrics=metrics or [], summary=summary, status=PEV_PENDING,
                created_at=now, updated_at=now)
            await self.projects.create_evidence(e)
            out = e

        await self.db.do(work)
        return out

    async def list_evidence(self, user_id: UUID, project_id: UUID, page):
        await self.get_blueprint(user_id, project_id)
        try:
            return await self.projects.list_evidence(user_id, project_id, page)
        except Exception:
            raise internal()

    async def verify_evidence(self, user_id: UUID, project_id: UUID,
                              evidence_id: UUID, expected: int) -> ent.Operation:
        op = None

        async def work():
            nonlocal op
            try:
                e = await self.projects.get_evidence(user_id, project_id, evidence_id)
            except NotFoundError:
                raise not_found()
            if e.revision != expected:
                raise revision_conflict(e.revision)
            if e.status == PEV_VERIFIED:
                raise invalid_transition("evidence already verified")
            now = _now()
            e.verification_method = "UNAVAILABLE"
            e.updated_at = now
            try:
                await self.projects.update_evidence(e, expected)
            except Exception as err:
                raise map_revision_err(err)
            e.revision = expected + 1
            op = ent.Operation(
                id=uuid4(), user_id=user_id, type=ent.OP_PROJECT_VERIFY,
                status=ent.OP_SUCCEEDED,
                result=ent.OperationResult(kind=ent.OP_PROJECT_VERIFY,
                                           value=to_jsonable(e)),
                created_at=now, updated_at=now)
            try:
                b = await self.projects.get_blueprint(user_id, project_id)
                op.application_id = b.application_id
            except Exception:
                pass
            await self.ops.create(op)

        await self.db.do(work)
        return op
