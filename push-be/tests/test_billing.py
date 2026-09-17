from __future__ import annotations

import hashlib
import hmac as hmac_mod
import json
from datetime import datetime, timedelta, timezone
from uuid import uuid4

import pytest

from app.domain import entities as ent
from app.domain.errors import DomainError
from app.domain.services.billing import BillingService, verify_stripe_signature
from tests.stubs import FakeDB


def _now():
    return datetime.now(timezone.utc)


def _sign(payload: bytes, secret: str, ts: datetime) -> str:
    t = int(ts.timestamp())
    mac = hmac_mod.new(secret.encode(), f"{t}.".encode() + payload,
                       hashlib.sha256).hexdigest()
    return f"t={t},v1={mac}"


def test_verify_stripe_signature():
    payload = b'{"id":"evt_1","type":"checkout.session.completed"}'
    verify_stripe_signature(payload, _sign(payload, "whsec_test", _now()),
                            "whsec_test", _now())


def test_verify_stripe_signature_bad():
    header = _sign(b'{"other":1}', "whsec_test", _now())
    with pytest.raises(ValueError):
        verify_stripe_signature(b"{}", header, "whsec_test", _now())


def test_verify_stripe_signature_stale():
    payload = b"{}"
    header = _sign(payload, "whsec_test", _now() - timedelta(minutes=10))
    with pytest.raises(ValueError):
        verify_stripe_signature(payload, header, "whsec_test", _now())


class StubBillingStore:
    def __init__(self, event_seen=False, balance=0):
        self.event_seen = event_seen
        self.balance = balance
        self.ledger = []
        self.sub_updates = []

    async def record_stripe_event(self, event_id, typ, at):
        return not self.event_seen

    async def update_subscription(self, user_id, plan, status, customer_id,
                                  period_ends_at):
        self.sub_updates.append((user_id, plan, status))

    async def update_subscription_by_customer(self, customer_id, status,
                                              period_ends_at):
        pass

    async def user_id_by_stripe_customer(self, customer_id):
        from app.db import NotFoundError
        raise NotFoundError()

    async def append_ledger(self, e):
        self.ledger.append(e)

    async def list_ledger(self, user_id, page):
        return self.ledger

    async def last_balance(self, user_id):
        return self.balance


class StubStripeGateway:
    def __init__(self, checkout=None, err=None):
        self.checkout = checkout
        self.err = err

    async def create_checkout_session(self, price_id, customer_id, user_id,
                                      plan_id, success_url, cancel_url):
        if self.err is not None:
            raise self.err
        return self.checkout

    async def create_portal_session(self, customer_id, return_url):
        return "https://billing.stripe.com/session/x"


def _svc(store=None, gw=None, prices=None, webhook_secret=""):
    svc = BillingService(FakeDB(), gw or StubStripeGateway(), True,
                         prices or {}, "https://app/ok", "https://app/cancel",
                         "https://app", webhook_secret)
    svc.billing = store or StubBillingStore()
    return svc


def _user():
    return ent.User(id=uuid4(), display_name="u", locale="en",
                    created_at=_now())


async def test_checkout_rejects_unknown_plan():
    svc = _svc(prices={ent.PLAN_ULTRA: "price_1"})
    with pytest.raises(DomainError) as e:
        await svc.checkout(_user(), "ENTERPRISE")
    assert e.value.code == "VALIDATION_ERROR"


async def test_checkout_not_configured_without_price():
    svc = _svc(prices={})
    with pytest.raises(DomainError) as e:
        await svc.checkout(_user(), ent.PLAN_ULTRA)
    assert e.value.code == "NOT_CONFIGURED"


async def test_checkout_returns_session_url():
    exp = _now() + timedelta(hours=1)
    gw = StubStripeGateway(ent.StripeCheckout(
        url="https://checkout.stripe.com/x", expires_at=exp))
    out = await _svc(gw=gw, prices={ent.PLAN_ULTRA: "price_1"}).checkout(
        _user(), ent.PLAN_ULTRA)
    assert out.url == "https://checkout.stripe.com/x"


async def test_webhook_rejects_bad_signature():
    svc = _svc(webhook_secret="whsec_test")
    with pytest.raises(DomainError) as e:
        await svc.handle_webhook(b"{}", "t=1,v1=bad")
    assert e.value.code == "VALIDATION_ERROR"


async def test_webhook_checkout_completed_grants_credits():
    user_id = uuid4()
    secret = "whsec_test"
    payload = json.dumps({
        "id": "evt_1", "type": "checkout.session.completed",
        "data": {"object": {
            "id": "cs_1", "client_reference_id": str(user_id),
            "customer": "cus_1", "metadata": {"planId": "ULTRA"}}},
    }).encode()
    store = StubBillingStore()
    svc = _svc(store=store, webhook_secret=secret)
    await svc.handle_webhook(payload, _sign(payload, secret, _now()))
    assert len(store.ledger) == 1
    assert store.ledger[0].amount_micro_credits == \
        ent.PLAN_CREDITS_MICRO[ent.PLAN_ULTRA]
