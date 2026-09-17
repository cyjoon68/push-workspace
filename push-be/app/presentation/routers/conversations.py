from __future__ import annotations

from fastapi import APIRouter, Request

from app.presentation import schemas as s
from app.presentation.deps import (
    bind_json, current_user, data, optional_query_uuid, page_body,
    page_request, param_id,
)
from app.presentation.routers.jobs import _ai

router = APIRouter()


def _deps(request: Request):
    return request.app.state.deps


@router.get("/conversations")
async def list_conversations(request: Request):
    d = _deps(request)
    p = await d.conversations.list(
        current_user(request).id,
        optional_query_uuid(request, "applicationId"), page_request(request))
    return page_body(p)


@router.post("/conversations", status_code=201)
async def create_conversation(request: Request):
    d = _deps(request)
    req = await bind_json(request, s.CreateConversationReq)
    v = await d.conversations.create(current_user(request).id,
                                     req.application_id, req.title)
    return data(201, v)


@router.patch("/conversations/{id}")
async def patch_conversation(id: str, request: Request):
    d = _deps(request)
    req = await bind_json(request, s.PatchConversationReq)
    v = await d.conversations.patch(current_user(request).id,
                                    param_id(id, "id"), req.expected_revision,
                                    req.title, req.pinned)
    return data(200, v)


@router.get("/conversations/{id}/messages")
async def list_messages(id: str, request: Request):
    d = _deps(request)
    p = await d.conversations.list_messages(current_user(request).id,
                                            param_id(id, "id"),
                                            page_request(request))
    return page_body(p)


@router.post("/conversations/{id}/messages", status_code=202)
async def post_message(id: str, request: Request):
    d = _deps(request)
    req = await bind_json(request, s.PostMessageReq)
    context = (req.context.model_dump(by_alias=True) if req.context else {})
    op = await d.conversations.post_message(current_user(request).id,
                                            param_id(id, "id"), req.text,
                                            context, _ai(req.ai),
                                            req.access_mode)
    return data(202, op)


@router.post("/conversations/{id}/archive")
async def archive_conversation(id: str, request: Request):
    d = _deps(request)
    req = await bind_json(request, s.ExpectedRevisionReq)
    v = await d.conversations.archive(current_user(request).id,
                                      param_id(id, "id"),
                                      req.expected_revision)
    return data(200, v)
