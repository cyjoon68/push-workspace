from __future__ import annotations
import hashlib
import hmac as hmac_mod
import json
from datetime import datetime, timezone
from uuid import UUID, uuid4

from app.db import DB
from app.domain import entities as ent
from app.domain.errors import (
    internal, invalid_transition, not_configured, provider_error,
    validation, validation_field,
)
from app.infrastructure.store_integrations import BillingStore

STRIPE_WEBHOOK_TOLERANCE = 300


def _now() -> datetime:
    return datetime.now(timezone.utc)


def verify_stripe_signature(payload: bytes, header: str, secret: str,
                            now: datetime) -> None:
    ts = 0
    sigs = []
    for part in header.split(","):
        kv = part.strip().split("=", 1)
        if len(kv) != 2:
            continue
        if kv[0] == "t":
            try:
                ts = int(kv[1])
            except ValueError:
                raise ValueError("bad timestamp")
        elif kv[0] == "v1":
            sigs.append(kv[1])
    if not ts or not sigs:
        raise ValueError("missing signature")
    d = int(now.timestamp()) - ts
    if abs(d) > STRIPE_WEBHOOK_TOLERANCE:
        raise ValueError("stale timestamp")
    expected = hmac_mod.new(
        secret.encode(), f"{ts}.".encode() + payload, hashlib.sha256).hexdigest()
    if not any(hmac_mod.compare_digest(sig, expected) for sig in sigs):
        raise ValueError("signature mismatch")


class BillingService:
    def __init__(self, db: DB, stripe, configured: bool, prices: dict,
                 success_url: str, cancel_url: str, return_url: str,
                 webhook_secret: str):
        self.db = db
        self.billing = BillingStore(db)
        self.stripe = stripe
        self.configured = configured
        self.prices = prices
        self.success_url = success_url
        self.cancel_url = cancel_url
        self.return_url = return_url
        self.webhook_secret = webhook_secret

    async def status(self, u: ent.User) -> ent.BillingStatus:
        try:
            bal = await self.billing.last_balance(u.id)
        except Exception:
            raise internal()
        st = ent.BillingStatus(
            subscription_status=u.subscription_status or ent.SUB_NONE,
            plan=u.plan or None, period_ends_at=u.period_ends_at,
            balance_micro_credits=bal)
        return st

    async def checkout(self, u: ent.User, plan_id: str) -> ent.StripeCheckout:
        if not self.configured:
            raise not_configured("billing is not configured")
        if plan_id == ent.PLAN_FREE or not ent.valid_plan(plan_id):
            raise validation_field("planId", "unsupported plan")
        price = self.prices.get(plan_id, "")
        if not price:
            raise not_configured("plan price is not configured")
        try:
            return await self.stripe.create_checkout_session(
                price, u.stripe_customer_id or "", str(u.id), plan_id,
                self.success_url, self.cancel_url)
        except Exception:
            raise provider_error("stripe checkout session failed")

    async def portal(self, u: ent.User) -> str:
        if not self.configured:
            raise not_configured("billing is not configured")
        if not u.stripe_customer_id:
            raise invalid_transition("no billing account")
        try:
            return await self.stripe.create_portal_session(
                u.stripe_customer_id, self.return_url)
        except Exception:
            raise provider_error("stripe portal session failed")

    async def ledger(self, user_id: UUID, page):
        try:
            return await self.billing.list_ledger(user_id, page)
        except Exception:
            raise internal()

    async def handle_webhook(self, payload: bytes, sig_header: str) -> None:
        if not self.configured or not self.webhook_secret:
            raise not_configured("billing is not configured")
        try:
            verify_stripe_signature(payload, sig_header, self.webhook_secret,
                                    datetime.now())
        except ValueError:
            raise validation("invalid webhook signature")
        try:
            event = json.loads(payload)
        except Exception:
            event = {}
        if not event.get("id"):
            raise validation("invalid webhook payload")
        try:
            fresh = await self.billing.record_stripe_event(
                event["id"], event.get("type", ""), _now())
        except Exception:
            raise internal()
        if not fresh:
            return
        obj = (event.get("data") or {}).get("object") or {}
        if event["type"] == "checkout.session.completed":
            await self._checkout_completed(obj)
        elif event["type"] in ("customer.subscription.updated",
                               "customer.subscription.deleted"):
            await self._subscription_changed(event["type"], obj)

    async def _checkout_completed(self, obj: dict) -> None:
        try:
            user_id = UUID(obj.get("client_reference_id") or "")
        except ValueError:
            raise validation("invalid session reference")
        plan_id = (obj.get("metadata") or {}).get("planId", "")
        if not ent.valid_plan(plan_id) or plan_id == ent.PLAN_FREE:
            raise validation("unsupported plan")
        credits = ent.PLAN_CREDITS_MICRO[plan_id]
        session_id = obj.get("id", "")
        customer = obj.get("customer", "")

        async def work():
            await self.billing.update_subscription(
                user_id, plan_id, ent.SUB_ACTIVE, customer, None)
            bal = await self.billing.last_balance(user_id)
            await self.billing.append_ledger(ent.LedgerEntry(
                id=uuid4(), user_id=user_id, type=ent.LEDGER_PURCHASE,
                amount_micro_credits=credits, balance_after=bal + credits,
                reference_id=session_id, created_at=_now()))

        await self.db.do(work)

    async def _subscription_changed(self, typ: str, obj: dict) -> None:
        customer = obj.get("customer", "")
        if not customer:
            raise validation("invalid subscription payload")
        period_end = None
        if obj.get("current_period_end"):
            period_end = datetime.fromtimestamp(
                obj["current_period_end"], timezone.utc)
        status = ent.SUB_ACTIVE
        if typ == "customer.subscription.deleted" or obj.get("status") == "canceled":
            status = ent.SUB_CANCELED
        try:
            await self.billing.update_subscription_by_customer(
                customer, status, period_end)
        except Exception:
            raise internal()
