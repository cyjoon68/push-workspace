from __future__ import annotations

from fastapi import APIRouter, Request
from fastapi.responses import RedirectResponse, Response

from app.domain import entities as ent
from app.presentation import schemas as s
from app.presentation.deps import bind_json, current_user, data

router = APIRouter()


def _deps(request: Request):
    return request.app.state.deps


def session_dto(sess: ent.Session) -> dict:
    return {
        "accessToken": sess.access_token,
        "refreshToken": sess.refresh_token,
        "expiresIn": sess.expires_in,
        "user": {"id": str(sess.user.id),
                 "displayName": sess.user.display_name,
                 "locale": sess.user.locale},
    }


@router.get("/auth/{provider}/start")
async def oauth_start(provider: str, request: Request):
    d = _deps(request)
    qp = request.query_params
    callback_url = (str(request.base_url).rstrip("/")
                    + "/api/v1/auth/" + provider + "/callback")
    url, state, expires_at = await d.auth.start_oauth(
        provider, qp.get("codeChallenge", ""),
        qp.get("codeChallengeMethod", ""), qp.get("redirectUri", ""),
        callback_url)
    return data(200, {"authorizationUrl": url, "state": state,
                      "expiresAt": expires_at})


@router.get("/auth/{provider}/callback")
async def oauth_callback(provider: str, request: Request):
    d = _deps(request)
    link = await d.auth.handle_callback(
        provider, request.query_params.get("code", ""),
        request.query_params.get("state", ""))
    return RedirectResponse(link, status_code=302)


@router.post("/auth/exchange")
async def auth_exchange(request: Request):
    d = _deps(request)
    req = await bind_json(request, s.ExchangeReq)
    sess = await d.auth.exchange(req.code, req.code_verifier)
    return data(200, session_dto(sess))


@router.post("/auth/refresh")
async def auth_refresh(request: Request):
    d = _deps(request)
    req = await bind_json(request, s.RefreshReq)
    sess = await d.auth.refresh(req.refresh_token)
    return data(200, session_dto(sess))


@router.post("/auth/logout")
async def auth_logout(request: Request):
    d = _deps(request)
    req = await bind_json(request, s.LogoutReq)
    await d.auth.logout(req.refresh_token)
    return Response(status_code=204)


@router.get("/auth/me")
async def auth_me(request: Request):
    u = current_user(request)
    return data(200, {"id": u.id, "displayName": u.display_name,
                      "locale": u.locale, "createdAt": u.created_at})
