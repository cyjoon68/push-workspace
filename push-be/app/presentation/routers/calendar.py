from __future__ import annotations

from fastapi import APIRouter, Request
from fastapi.responses import Response

from app.presentation import schemas as s
from app.presentation.deps import (
    bind_json, current_user, data, optional_query_time, optional_query_uuid,
    page_body, page_request, param_id, parse_time_field,
)

router = APIRouter()


def _deps(request: Request):
    return request.app.state.deps


@router.get("/calendar/events")
async def list_calendar_events(request: Request):
    d = _deps(request)
    p = await d.calendar.list(
        current_user(request).id,
        optional_query_uuid(request, "applicationId"),
        optional_query_time(request, "from"),
        optional_query_time(request, "to"), None, page_request(request))
    return page_body(p)


@router.post("/calendar/events", status_code=201)
async def create_calendar_event(request: Request):
    d = _deps(request)
    req = await bind_json(request, s.CreateCalendarEventReq)
    e = await d.calendar.create(
        current_user(request).id, req.application_id, req.type, req.title,
        parse_time_field(req.starts_at, "startsAt"),
        parse_time_field(req.ends_at, "endsAt"), req.time_zone, req.notes)
    return data(201, e)


@router.patch("/calendar/events/{id}")
async def patch_calendar_event(id: str, request: Request):
    d = _deps(request)
    req = await bind_json(request, s.PatchCalendarEventReq)
    starts = (parse_time_field(req.starts_at, "startsAt")
              if req.starts_at is not None else None)
    ends = (parse_time_field(req.ends_at, "endsAt")
            if req.ends_at is not None else None)
    e = await d.calendar.patch(current_user(request).id, param_id(id, "id"),
                               req.expected_revision, req.title, starts, ends,
                               req.time_zone, req.notes)
    return data(200, e)


@router.delete("/calendar/events/{id}", status_code=204)
async def delete_calendar_event(id: str, request: Request):
    d = _deps(request)
    req = await bind_json(request, s.ExpectedRevisionReq)
    await d.calendar.delete(current_user(request).id, param_id(id, "id"),
                            req.expected_revision)
    return Response(status_code=204)
