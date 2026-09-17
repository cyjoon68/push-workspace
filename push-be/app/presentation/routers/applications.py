from __future__ import annotations

from fastapi import APIRouter, Request

from app.domain import entities as ent
from app.domain.errors import validation_field
from app.presentation import schemas as s
from app.presentation.deps import (
    bind_json, current_user, data, page_body, page_request, param_id,
    parse_time_field,
)

router = APIRouter()


def _deps(request: Request):
    return request.app.state.deps


@router.get("/applications")
async def list_applications(request: Request):
    d = _deps(request)
    page = page_request(request)
    stage = request.query_params.get("stage") or None
    if stage is not None and not ent.valid_stage(stage):
        raise validation_field("stage", "unsupported stage")
    p = await d.applications.list(current_user(request).id, stage,
                                  request.query_params.get("query", ""), page)
    return page_body(p)


@router.post("/applications", status_code=201)
async def create_application(request: Request):
    d = _deps(request)
    req = await bind_json(request, s.CreateApplicationReq)
    a = await d.applications.create(current_user(request).id, req.job_id,
                                    req.notes)
    return data(201, a)


@router.post("/applications/import", status_code=201)
async def import_application(request: Request):
    d = _deps(request)
    req = await bind_json(request, s.ImportApplicationReq)
    a = await d.applications.import_(
        current_user(request).id, req.job_id, req.stage,
        parse_time_field(req.applied_at, "appliedAt"), req.notes,
        req.confirmed)
    return data(201, a)


@router.get("/applications/{id}")
async def get_application(id: str, request: Request):
    d = _deps(request)
    a = await d.applications.get(current_user(request).id,
                                 param_id(id, "id"))
    return data(200, a)


@router.patch("/applications/{id}")
async def patch_application(id: str, request: Request):
    d = _deps(request)
    req = await bind_json(request, s.PatchApplicationReq)
    next_at = None
    clear_next = False
    if "next_action_at" in req.model_fields_set:
        if req.next_action_at is None:
            clear_next = True
        else:
            next_at = parse_time_field(req.next_action_at, "nextActionAt")
    a = await d.applications.patch(current_user(request).id,
                                   param_id(id, "id"), req.expected_revision,
                                   req.stage, req.notes, next_at, clear_next)
    return data(200, a)


@router.get("/applications/{id}/timeline")
async def application_timeline(id: str, request: Request):
    d = _deps(request)
    p = await d.applications.timeline(current_user(request).id,
                                      param_id(id, "id"),
                                      page_request(request))
    return page_body(p)


@router.get("/applications/{id}/checklist")
async def application_checklist(id: str, request: Request):
    d = _deps(request)
    cl = await d.applications.checklist(current_user(request).id,
                                        param_id(id, "id"))
    return data(200, cl)


@router.post("/applications/{id}/submission-drafts", status_code=201)
async def create_submission_draft(id: str, request: Request):
    d = _deps(request)
    req = await bind_json(request, s.CreateDraftReq)
    dr = await d.applications.create_draft(
        current_user(request).id, param_id(id, "id"), req.expected_revision,
        req.mode, req.adapter, req.document_version_ids,
        req.confirmed_submitted)
    return data(201, dr)


@router.post("/applications/{id}/submissions", status_code=201)
async def submit_application(id: str, request: Request):
    d = _deps(request)
    req = await bind_json(request, s.SubmitReq)
    sub = await d.applications.submit(current_user(request).id,
                                      param_id(id, "id"),
                                      req.expected_revision, req.draft_id,
                                      req.approval_id)
    return data(201, sub)


@router.get("/applications/{id}/submissions")
async def list_submissions(id: str, request: Request):
    d = _deps(request)
    p = await d.applications.list_submissions(current_user(request).id,
                                              param_id(id, "id"),
                                              page_request(request))
    return page_body(p)
