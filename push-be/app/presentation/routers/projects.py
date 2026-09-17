from __future__ import annotations

from fastapi import APIRouter, Request

from app.domain import entities as ent
from app.presentation import schemas as s
from app.presentation.deps import (
    bind_json, current_user, data, optional_query_uuid, page_body,
    page_request, param_id, parse_time_field,
)
from app.presentation.routers.jobs import _ai

router = APIRouter()


def _deps(request: Request):
    return request.app.state.deps


def _process(p: s.ProcessInfoReq | None) -> ent.ProcessInfo | None:
    if p is None:
        return None
    return ent.ProcessInfo(pid=p.pid,
                           started_at=parse_time_field(p.started_at,
                                                       "process.startedAt"))


@router.get("/projects")
async def list_projects(request: Request):
    d = _deps(request)
    p = await d.projects.list_blueprints(
        current_user(request).id, optional_query_uuid(request, "applicationId"),
        page_request(request))
    return page_body(p)


@router.post("/projects/blueprints", status_code=202)
async def generate_blueprints(request: Request):
    d = _deps(request)
    req = await bind_json(request, s.BlueprintsReq)
    op = await d.projects.generate_blueprints(
        current_user(request).id, req.application_id, req.gap_analysis_id,
        _ai(req.ai))
    return data(202, op)


@router.get("/projects/{id}")
async def get_project(id: str, request: Request):
    d = _deps(request)
    b = await d.projects.get_blueprint(current_user(request).id,
                                       param_id(id, "id"))
    return data(200, b)


@router.post("/projects/{id}/select")
async def select_project(id: str, request: Request):
    d = _deps(request)
    req = await bind_json(request, s.ExpectedRevisionReq)
    b, manifest = await d.projects.select(current_user(request).id,
                                          param_id(id, "id"),
                                          req.expected_revision)
    return data(200, {"blueprint": b, "manifest": manifest})


@router.get("/projects/{id}/runs")
async def list_runs(id: str, request: Request):
    d = _deps(request)
    p = await d.projects.list_runs(current_user(request).id,
                                   param_id(id, "id"), page_request(request))
    return page_body(p)


@router.post("/projects/{id}/runs", status_code=201)
async def create_run(id: str, request: Request):
    d = _deps(request)
    req = await bind_json(request, s.CreateRunReq)
    r = await d.projects.create_run(current_user(request).id,
                                    param_id(id, "id"), req.provider,
                                    req.working_directory, req.prompt)
    return data(201, r)


def _run_ids(id: str, runId: str):
    return param_id(id, "id"), param_id(runId, "runId")


@router.post("/projects/{id}/runs/{runId}/start")
async def start_run(id: str, runId: str, request: Request):
    d = _deps(request)
    req = await bind_json(request, s.StartRunReq)
    pid, rid = _run_ids(id, runId)
    r = await d.projects.start_run(current_user(request).id, pid, rid,
                                   req.expected_revision, req.approval_id,
                                   req.detected_version, req.device_id)
    return data(200, r)


@router.post("/projects/{id}/runs/{runId}/launch")
async def launch_run(id: str, runId: str, request: Request):
    d = _deps(request)
    req = await bind_json(request, s.LaunchReq)
    pid, rid = _run_ids(id, runId)
    r = await d.projects.report_launch(current_user(request).id, pid, rid,
                                       req.expected_revision, req.device_id,
                                       req.payload_hash, req.launch_status,
                                       _process(req.process))
    return data(200, r)


@router.post("/projects/{id}/runs/{runId}/recover")
async def recover_run(id: str, runId: str, request: Request):
    d = _deps(request)
    req = await bind_json(request, s.RecoverReq)
    pid, rid = _run_ids(id, runId)
    r = await d.projects.recover(current_user(request).id, pid, rid,
                                 req.expected_revision, req.device_id,
                                 req.decision, _process(req.process),
                                 req.failure_reason)
    return data(200, r)


@router.post("/projects/{id}/runs/{runId}/result")
async def result_run(id: str, runId: str, request: Request):
    d = _deps(request)
    req = await bind_json(request, s.RunResultReq)
    pid, rid = _run_ids(id, runId)
    r = await d.projects.report_result(current_user(request).id, pid, rid,
                                       req.expected_revision, req.exit_code,
                                       req.commit_sha, req.stdout_hash,
                                       req.stderr_hash)
    return data(200, r)


@router.post("/projects/{id}/evidence", status_code=201)
async def create_project_evidence(id: str, request: Request):
    d = _deps(request)
    req = await bind_json(request, s.CreateProjectEvidenceReq)
    e = await d.projects.create_evidence(
        current_user(request).id, param_id(id, "id"), req.run_id,
        req.commit_url, req.test_command, req.test_output, req.exit_code,
        [ent.MetricEntry(name=m.name, value=m.value, unit=m.unit)
         for m in req.metrics], req.summary)
    return data(201, e)


@router.post("/projects/{id}/evidence/{evidenceId}/verify", status_code=202)
async def verify_project_evidence(id: str, evidenceId: str, request: Request):
    d = _deps(request)
    req = await bind_json(request, s.ExpectedRevisionReq)
    op = await d.projects.verify_evidence(current_user(request).id,
                                          param_id(id, "id"),
                                          param_id(evidenceId, "evidenceId"),
                                          req.expected_revision)
    return data(202, op)


@router.get("/projects/{id}/evidence")
async def list_project_evidence(id: str, request: Request):
    d = _deps(request)
    p = await d.projects.list_evidence(current_user(request).id,
                                       param_id(id, "id"),
                                       page_request(request))
    return page_body(p)
