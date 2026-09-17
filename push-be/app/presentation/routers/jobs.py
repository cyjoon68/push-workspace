from __future__ import annotations

from fastapi import APIRouter, Request

from app.domain import entities as ent
from app.presentation import schemas as s
from app.presentation.deps import (
    bind_json, current_user, data, page_body, page_request, param_id,
    parse_deadline,
)

router = APIRouter()


def _deps(request: Request):
    return request.app.state.deps


def _ai(req: s.AiOptionsReq | None) -> ent.AiOptions | None:
    if req is None:
        return None
    return ent.AiOptions(provider=req.provider, model=req.model,
                         credential_mode=req.credential_mode, effort=req.effort)


def job_dto(j: ent.JobPosting) -> dict:
    return {"id": j.id, "revision": j.revision, "company": j.company,
            "title": j.title, "sourceKind": j.source_kind,
            "sourceUrl": j.source_url, "sourceText": j.source_text,
            "requirements": j.requirements, "preferred": j.preferred,
            "keywords": j.keywords, "risks": j.risks,
            "deadline": j.deadline.date() if j.deadline else None,
            "language": j.language, "createdAt": j.created_at,
            "updatedAt": j.updated_at}


@router.get("/jobs")
async def list_jobs(request: Request):
    d = _deps(request)
    page = page_request(request)
    archived = None
    if "archived" in request.query_params:
        archived = request.query_params["archived"] == "true"
    p = await d.jobs.list(current_user(request).id, archived,
                          request.query_params.get("query", ""), page)
    from app.jsonutil import to_jsonable
    from fastapi.responses import JSONResponse
    return JSONResponse(content={
        "data": [to_jsonable(job_dto(j)) for j in p.items],
        "page": {"nextCursor": p.next_cursor, "hasMore": p.has_more}},
        status_code=200)


@router.post("/jobs", status_code=201)
async def create_job(request: Request):
    d = _deps(request)
    req = await bind_json(request, s.CreateJobReq)
    deadline, _ = parse_deadline(req.deadline, "deadline" in req.model_fields_set)
    j = await d.jobs.create(current_user(request).id, req.company, req.title,
                            req.source_kind, req.source_url, req.source_text,
                            req.requirements, req.preferred, deadline,
                            req.language)
    return data(201, job_dto(j))


@router.get("/jobs/{id}")
async def get_job(id: str, request: Request):
    d = _deps(request)
    j = await d.jobs.get(current_user(request).id, param_id(id, "id"))
    return data(200, job_dto(j))


@router.patch("/jobs/{id}")
async def patch_job(id: str, request: Request):
    d = _deps(request)
    req = await bind_json(request, s.PatchJobReq)
    deadline, deadline_set = parse_deadline(
        req.deadline, "deadline" in req.model_fields_set)
    j = await d.jobs.patch(current_user(request).id, param_id(id, "id"),
                           req.expected_revision, req.company, req.title,
                           req.requirements, req.preferred, deadline,
                           deadline_set and deadline is None)
    return data(200, job_dto(j))


@router.post("/jobs/{id}/analyze", status_code=202)
async def analyze_job(id: str, request: Request):
    d = _deps(request)
    req = await bind_json(request, s.AnalyzeReq)
    op = await d.jobs.analyze(current_user(request).id, param_id(id, "id"),
                              req.application_id, req.expected_revision,
                              req.evidence_ids, _ai(req.ai))
    return data(202, op)


@router.get("/jobs/{id}/analyses")
async def list_analyses(id: str, request: Request):
    d = _deps(request)
    p = await d.jobs.list_analyses(current_user(request).id,
                                   param_id(id, "id"), page_request(request))
    return page_body(p)
