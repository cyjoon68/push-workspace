from __future__ import annotations
from datetime import datetime, timezone

import httpx

from app.domain.entities import StripeCheckout


class StripeClient:
    def __init__(self, secret: str):
        self._secret = secret
        self._client = httpx.AsyncClient(timeout=20)

    async def _post(self, path: str, form: dict) -> dict:
        resp = await self._client.post(
            "https://api.stripe.com" + path, data=form,
            auth=(self._secret, ""))
        if resp.status_code != 200:
            msg = ""
            try:
                msg = (resp.json().get("error") or {}).get("message", "")
            except Exception:
                pass
            raise ValueError(f"stripe {path} failed: status {resp.status_code} {msg}")
        return resp.json()

    async def create_checkout_session(self, price_id: str, customer_id: str,
                                      user_id: str, plan_id: str,
                                      success_url: str,
                                      cancel_url: str) -> StripeCheckout:
        form = {
            "mode": "subscription",
            "line_items[0][price]": price_id,
            "line_items[0][quantity]": "1",
            "success_url": success_url,
            "cancel_url": cancel_url,
            "client_reference_id": user_id,
            "metadata[planId]": plan_id,
        }
        if customer_id:
            form["customer"] = customer_id
        body = await self._post("/v1/checkout/sessions", form)
        if not body.get("url"):
            raise ValueError("stripe checkout session missing url")
        out = StripeCheckout(url=body["url"])
        if body.get("expires_at"):
            out.expires_at = datetime.fromtimestamp(body["expires_at"], timezone.utc)
        return out

    async def create_portal_session(self, customer_id: str,
                                    return_url: str) -> str:
        body = await self._post("/v1/billing_portal/sessions",
                                {"customer": customer_id, "return_url": return_url})
        if not body.get("url"):
            raise ValueError("stripe portal session missing url")
        return body["url"]

    async def aclose(self):
        await self._client.aclose()
