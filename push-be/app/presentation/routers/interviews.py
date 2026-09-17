from __future__ import annotations

from fastapi import APIRouter, Request

from app.domain import entities as ent
from app.presentation import schemas as s
from app.presentation.deps import (
    bind_json, current_user, data, optional_query_time, optional_query_uuid,
    page_body, page_request, param_id, parse_time_field,
)
from app.presentation.routers.jobs import _ai

router = APIRouter()


def _deps(request: Request):
    return request.app.state.deps


def _company_sources(items: list[s.CompanySourceReq] | None):
    if items is None:
        return None
    return [ent.CompanySource(
        source_url=c.source_url, source_text=c.source_text,
        accessed_at=parse_time_field(c.accessed_at,
                                     "companySources.accessedAt"))
            for c in items]


@router.get("/interviews")
async def list_interviews(request: Request):
    d = _deps(request)
    p = await d.interviews.list(
        current_user(request).id,
        optional_query_uuid(request, "applicationId"),
        optional_query_time(request, "from"),
        optional_query_time(request, "to"), page_request(request))
    return page_body(p)


@router.post("/interviews", status_code=201)
async def create_interview(request: Request):
    d = _deps(request)
    req = await bind_json(request, s.CreateInterviewReq)
    v = await d.interviews.create(
        current_user(request).id, req.application_id, req.title,
        parse_time_field(req.scheduled_at, "scheduledAt"),
        req.duration_minutes, req.evidence_ids,
        _company_sources(req.company_sources) or [], req.notes,
        req.time_zone)
    return data(201, v)


@router.get("/interviews/{id}")
async def get_interview(id: str, request: Request):
    d = _deps(request)
    v = await d.interviews.get(current_user(request).id, param_id(id, "id"))
    return data(200, v)


@router.patch("/interviews/{id}")
async def patch_interview(id: str, request: Request):
    d = _deps(request)
    req = await bind_json(request, s.PatchInterviewReq)
    scheduled = (parse_time_field(req.scheduled_at, "scheduledAt")
                 if req.scheduled_at is not None else None)
    v = await d.interviews.patch(
        current_user(request).id, param_id(id, "id"), req.expected_revision,
        req.title, scheduled, _company_sources(req.company_sources),
        req.notes, req.reflection)
    return data(200, v)


@router.post("/interviews/{id}/prepare", status_code=202)
async def prepare_interview(id: str, request: Request):
    d = _deps(request)
    req = await bind_json(request, s.PrepareReq)
    op = await d.interviews.prepare(current_user(request).id,
                                    param_id(id, "id"),
                                    req.expected_revision, _ai(req.ai))
    return data(202, op)
