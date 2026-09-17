from __future__ import annotations

from fastapi import APIRouter, Request

from app.domain.errors import validation
from app.presentation import schemas as s
from app.presentation.deps import (
    bind_json, current_user, data, page_body, page_request,
)

router = APIRouter()


def _deps(request: Request):
    return request.app.state.deps


@router.get("/billing")
async def billing_status(request: Request):
    d = _deps(request)
    st = await d.billing.status(current_user(request))
    return data(200, st)


@router.post("/billing/checkout")
async def billing_checkout(request: Request):
    d = _deps(request)
    req = await bind_json(request, s.CheckoutReq)
    sess = await d.billing.checkout(current_user(request), req.plan_id)
    return data(200, {"url": sess.url, "expiresAt": sess.expires_at})


@router.post("/billing/portal")
async def billing_portal(request: Request):
    d = _deps(request)
    url = await d.billing.portal(current_user(request))
    return data(200, {"url": url})


@router.get("/billing/ledger")
async def billing_ledger(request: Request):
    d = _deps(request)
    p = await d.billing.ledger(current_user(request).id,
                               page_request(request))
    return page_body(p)


@router.post("/billing/webhook")
async def billing_webhook(request: Request):
    d = _deps(request)
    body = await request.body()
    if not body:
        raise validation("unreadable body")
    await d.billing.handle_webhook(
        body, request.headers.get("stripe-signature", ""))
    return data(200, {"received": True})
