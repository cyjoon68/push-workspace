from __future__ import annotations

from fastapi import APIRouter, Request

from app.domain.errors import validation_field
from app.presentation import schemas as s
from app.presentation.deps import (
    bind_json, current_user, data, optional_query_uuid, page_body,
    page_request, param_id,
)

router = APIRouter()

APPROVAL_STATUSES = {"PENDING", "APPROVED", "DENIED", "EXPIRED", "CONSUMED"}


def _deps(request: Request):
    return request.app.state.deps


@router.get("/approvals")
async def list_approvals(request: Request):
    d = _deps(request)
    status = request.query_params.get("status") or None
    if status is not None and status not in APPROVAL_STATUSES:
        raise validation_field("status", "unsupported status")
    p = await d.approvals.list(
        current_user(request).id,
        optional_query_uuid(request, "applicationId"), status,
        page_request(request))
    return page_body(p)


@router.post("/approvals", status_code=201)
async def create_approval(request: Request):
    d = _deps(request)
    req = await bind_json(request, s.CreateApprovalReq)
    a = await d.approvals.create(current_user(request).id, req.kind,
                                 req.application_id, req.target_id,
                                 req.target_revision)
    return data(201, a)


@router.get("/approvals/{id}")
async def get_approval(id: str, request: Request):
    d = _deps(request)
    a, target = await d.approvals.get(current_user(request).id,
                                      param_id(id, "id"))
    return data(200, {"approval": a, "target": target})


@router.post("/approvals/{id}/decision")
async def decide_approval(id: str, request: Request):
    d = _deps(request)
    req = await bind_json(request, s.DecisionReq)
    a = await d.approvals.decide(current_user(request).id, param_id(id, "id"),
                                 req.expected_revision, req.decision)
    return data(200, a)
