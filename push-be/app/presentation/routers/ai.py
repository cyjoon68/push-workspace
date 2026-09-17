from __future__ import annotations

from datetime import datetime, timezone

from fastapi import APIRouter, Request
from fastapi.responses import Response

from app.domain import entities as ent
from app.domain.errors import (
    internal, not_configured, not_found, validation_field,
)
from app.infrastructure.crypto import new_key_cipher
from app.presentation import schemas as s
from app.presentation.deps import (
    bind_json, current_user, data, optional_query_time, page_body,
    page_request,
)
from app.presentation.routers.jobs import _ai

router = APIRouter()


def _deps(request: Request):
    return request.app.state.deps


@router.get("/ai/models")
async def ai_models(request: Request):
    d = _deps(request)
    provider = request.query_params.get("provider", "")
    cred_mode = request.query_params.get("credentialMode", "")
    if provider and not ent.valid_ai_provider(provider):
        raise validation_field("provider", "unsupported provider")
    if cred_mode and not ent.valid_credential_mode(cred_mode):
        raise validation_field("credentialMode", "unsupported credentialMode")
    byok_openai = False
    if d.cfg.byok_enabled and d.ai_keys is not None:
        try:
            await d.ai_keys.get(current_user(request).id, "OPENAI")
            byok_openai = True
        except Exception:
            pass
    models = []
    for m in ent.OPENAI_MODELS:
        if provider and provider != m.provider:
            continue
        available = d.cfg.managed_ai or byok_openai
        if cred_mode == "MANAGED":
            available = d.cfg.managed_ai
        elif cred_mode == "BYOK":
            available = byok_openai
        models.append({"provider": m.provider, "model": m.model,
                       "label": m.label, "available": available,
                       "supportedEfforts": ["LOW", "MEDIUM", "HIGH"]})
    return data(200, models)


def _key_dto(k: ent.AiKey) -> dict:
    return {"provider": k.provider, "lastFour": k.last_four,
            "configured": True, "updatedAt": k.updated_at}


@router.get("/ai/keys")
async def ai_keys_list(request: Request):
    d = _deps(request)
    try:
        keys = await d.ai_keys.list(current_user(request).id)
    except Exception:
        raise internal()
    return data(200, [_key_dto(k) for k in keys])


@router.put("/ai/keys/{provider}")
async def ai_key_put(provider: str, request: Request):
    d = _deps(request)
    if not ent.valid_ai_provider(provider):
        raise validation_field("provider", "unsupported provider")
    if not d.cfg.byok_enabled:
        raise not_configured("BYOK encryption is not configured")
    req = await bind_json(request, s.PutAiKeyReq)
    if len(req.key) < 8:
        raise validation_field("key", "key is too short")
    try:
        cipher = new_key_cipher(d.master_key)
    except ValueError:
        raise not_configured("BYOK encryption is not configured")
    try:
        ct, nonce = cipher.encrypt(req.key, current_user(request).id, provider)
    except Exception:
        raise internal()
    k = ent.AiKey(user_id=current_user(request).id, provider=provider,
                  last_four=req.key[-4:], ciphertext=ct, nonce=nonce,
                  updated_at=datetime.now(timezone.utc))
    try:
        await d.ai_keys.put(k)
    except Exception:
        raise internal()
    return data(200, _key_dto(k))


@router.delete("/ai/keys/{provider}", status_code=204)
async def ai_key_delete(provider: str, request: Request):
    d = _deps(request)
    if not ent.valid_ai_provider(provider):
        raise validation_field("provider", "unsupported provider")
    try:
        await d.ai_keys.delete(current_user(request).id, provider)
    except Exception:
        raise not_found()
    return Response(status_code=204)


@router.post("/ai/keys/{provider}/test")
async def ai_key_test(provider: str, request: Request):
    d = _deps(request)
    if not ent.valid_ai_provider(provider):
        raise validation_field("provider", "unsupported provider")
    now = datetime.now(timezone.utc)
    try:
        await d.ai_keys.get(current_user(request).id, provider)
    except Exception:
        return data(200, {"valid": False, "checkedAt": now,
                          "errorCode": "KEY_MISSING"})
    return data(200, {"valid": False, "checkedAt": now,
                      "errorCode": "PROVIDER_UNREACHABLE"})


@router.post("/ai/generate", status_code=202)
async def ai_generate(request: Request):
    d = _deps(request)
    req = await bind_json(request, s.AiGenerateReq)
    op = await d.ai.generate(current_user(request).id, _ai(req.ai),
                             req.prompt, req.application_id,
                             req.evidence_ids)
    return data(202, op)


@router.get("/ai/usage")
async def ai_usage(request: Request):
    d = _deps(request)
    p = await d.ai.list_usage(current_user(request).id,
                              optional_query_time(request, "from"),
                              optional_query_time(request, "to"),
                              page_request(request))
    return page_body(p)
