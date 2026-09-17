from __future__ import annotations

from fastapi import APIRouter, Request
from fastapi.responses import RedirectResponse, Response

from app.domain.errors import feature_disabled, not_configured
from app.presentation import schemas as s
from app.presentation.deps import (
    bind_json, current_user, data, optional_query_time, optional_query_uuid,
    page_body, page_request, param_id,
)

router = APIRouter()

JOB_SITE_CHECKLIST = [
    "Open the job posting in the built-in browser",
    "Review the extracted fields against the posting",
    "Upload finalized documents manually",
    "Confirm submission in the site UI",
    "Record the submission via POST /applications/:id/submission-drafts",
]


def _deps(request: Request):
    return request.app.state.deps


@router.get("/integrations/google")
async def google_status(request: Request):
    d = _deps(request)
    st = await d.google.status(current_user(request).id)
    return data(200, st)


@router.post("/integrations/google/connect")
async def google_connect(request: Request):
    d = _deps(request)
    req = await bind_json(request, s.GoogleConnectReq)
    if not d.cfg.google_configured:
        raise not_configured("google integration is not configured")
    callback_url = (str(request.base_url).rstrip("/")
                    + "/api/v1/integrations/google/callback")
    url, state, expires_at = await d.google.connect(
        current_user(request).id, req.code_challenge,
        req.code_challenge_method, req.redirect_uri, callback_url)
    return data(200, {"authorizationUrl": url, "state": state,
                      "expiresAt": expires_at})


@router.get("/integrations/google/callback")
async def google_callback(request: Request):
    d = _deps(request)
    if not d.cfg.google_configured:
        raise not_configured("google integration is not configured")
    link = await d.google.handle_callback(
        request.query_params.get("code", ""),
        request.query_params.get("state", ""))
    return RedirectResponse(link, status_code=302)


@router.post("/integrations/google/complete")
async def google_complete(request: Request):
    d = _deps(request)
    req = await bind_json(request, s.GoogleCompleteReq)
    if not d.cfg.google_configured:
        raise not_configured("google integration is not configured")
    st = await d.google.complete(current_user(request).id,
                                 req.integration_code, req.code_verifier)
    return data(200, st)


@router.delete("/integrations/google", status_code=204)
async def google_disconnect(request: Request):
    d = _deps(request)
    await d.google.disconnect(current_user(request).id)
    return Response(status_code=204)


@router.post("/integrations/google/sync", status_code=202)
async def google_sync(request: Request):
    d = _deps(request)
    if not d.cfg.google_configured:
        raise not_configured("google integration is not configured")
    op = await d.google.sync(current_user(request).id)
    return data(202, op)


@router.get("/integrations/google/messages")
async def google_messages(request: Request):
    d = _deps(request)
    if not d.cfg.google_configured:
        raise not_configured("google integration is not configured")
    if not d.cfg.gmail_beta:
        raise feature_disabled("gmail sync is not enabled")
    p = await d.google.list_messages(
        current_user(request).id,
        optional_query_uuid(request, "applicationId"), page_request(request))
    return page_body(p)


@router.get("/integrations/google/events")
async def google_events(request: Request):
    d = _deps(request)
    if not d.cfg.google_configured:
        raise not_configured("google integration is not configured")
    p = await d.google.list_events(
        current_user(request).id, optional_query_time(request, "from"),
        optional_query_time(request, "to"), page_request(request))
    return page_body(p)


@router.post("/integrations/google/messages/{id}/link")
async def google_link_message(id: str, request: Request):
    d = _deps(request)
    req = await bind_json(request, s.LinkMessageReq)
    if not d.cfg.google_configured:
        raise not_configured("google integration is not configured")
    m = await d.google.link_message(current_user(request).id,
                                    param_id(id, "id"), req.application_id)
    return data(200, m)


@router.get("/integrations/job-sites")
async def job_sites(request: Request):
    d = _deps(request)
    sites = [{"provider": p,
              "enabled": d.cfg.job_site_adapters.get(p, False),
              "permissionVerified": False, "mode": "MANUAL_CHECKLIST",
              "supportedFields": [], "checklist": JOB_SITE_CHECKLIST}
             for p in ("WANTED", "JUMPIT", "JOBKOREA")]
    return data(200, sites)
