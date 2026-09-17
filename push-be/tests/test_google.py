from __future__ import annotations

import base64
import hashlib
import json
from datetime import datetime, timedelta, timezone
from uuid import uuid4

import pytest

from app.db import NotFoundError
from app.domain import entities as ent
from app.domain.errors import DomainError
from app.domain.services.google import GoogleService
from tests.stubs import FakeDB

GOOGLE_CALLBACK_URI = "push://integrations/google/callback"


def _now():
    return datetime.now(timezone.utc)


class StubGoogleStore:
    def __init__(self, integration=None, code=None):
        self.integration = integration
        self.code = code
        self.messages = []

    async def save_integration_code(self, c):
        self.code = c

    async def get_integration_code(self, code):
        if self.code is None or self.code.code != code:
            raise NotFoundError()
        return self.code

    async def mark_integration_code_used(self, code, at):
        self.code.used_at = at

    async def put_google_integration(self, g):
        self.integration = g

    async def get_google_integration(self, user_id):
        if self.integration is None or self.integration.user_id != user_id:
            raise NotFoundError()
        return self.integration

    async def delete_google_integration(self, user_id):
        self.integration = None

    async def delete_google_data(self, user_id):
        self.messages = []


class StubSessionStore:
    def __init__(self):
        self.states = {}

    async def save_oauth_state(self, st):
        self.states[st.state] = st

    async def get_oauth_state(self, state):
        if state not in self.states:
            raise NotFoundError()
        return self.states[state]

    async def delete_oauth_state(self, state):
        self.states.pop(state, None)


class StubTokenCipher:
    def encrypt(self, plaintext, user_id, purpose):
        return b"ct:" + plaintext.encode(), b"n"

    def decrypt(self, ct, nonce, user_id, purpose):
        return ct[3:].decode()


class StubGoogleClient:
    def __init__(self, tokens=None, messages=None, events=None):
        self.tokens = tokens
        self.messages = messages or {}
        self.events = events or []

    async def exchange_code(self, code, redirect_uri):
        return self.tokens

    async def refresh_access_token(self, rt):
        return ent.GoogleTokens(access_token="fresh", expires_in=3600)


def _svc(store=None, sessions=None, gapi=None):
    svc = GoogleService(FakeDB(), StubTokenCipher(), gapi or StubGoogleClient(),
                        "cid", "csec",
                        "https://accounts.google.com/o/oauth2/v2/auth", True)
    svc.store = store or StubGoogleStore()
    svc.sessions = sessions or StubSessionStore()
    return svc


def _verifier():
    verifier = "test-verifier-123"
    challenge = base64.urlsafe_b64encode(
        hashlib.sha256(verifier.encode()).digest()).rstrip(b"=").decode()
    return verifier, challenge


async def test_connect_validates_input():
    svc = _svc()
    with pytest.raises(DomainError) as e:
        await svc.connect(uuid4(), "", "S256", GOOGLE_CALLBACK_URI,
                          "https://cb")
    assert e.value.code == "VALIDATION_ERROR"
    with pytest.raises(DomainError) as e:
        await svc.connect(uuid4(), "ch", "plain", GOOGLE_CALLBACK_URI,
                          "https://cb")
    assert e.value.code == "VALIDATION_ERROR"
    with pytest.raises(DomainError) as e:
        await svc.connect(uuid4(), "ch", "S256", "https://evil.example",
                          "https://cb")
    assert e.value.code == "VALIDATION_ERROR"


async def test_connect_builds_authorization_url():
    sessions = StubSessionStore()
    svc = _svc(sessions=sessions)
    url, state, _ = await svc.connect(
        uuid4(), "ch", "S256", GOOGLE_CALLBACK_URI,
        "https://host/api/v1/integrations/google/callback")
    assert url and state
    st = sessions.states[state]
    assert st.purpose == "GOOGLE"


async def test_complete_pkce_mismatch():
    user_id = uuid4()
    _, challenge = _verifier()
    payload = json.dumps({"accessToken": "at"})
    ct, nonce = StubTokenCipher().encrypt(payload, user_id, "google")
    bundle = {"ciphertext": base64.b64encode(ct).decode(),
              "nonce": base64.b64encode(nonce).decode()}
    store = StubGoogleStore(code=ent.IntegrationCode(
        code="ic1", user_id=user_id, code_challenge=challenge,
        payload=bundle, expires_at=_now() + timedelta(minutes=1),
        created_at=_now()))
    with pytest.raises(DomainError) as e:
        await _svc(store=store).complete(user_id, "ic1", "wrong-verifier")
    assert e.value.code == "UNAUTHENTICATED"


async def test_complete_user_mismatch():
    verifier, challenge = _verifier()
    payload = json.dumps({"accessToken": "at"})
    ct, nonce = StubTokenCipher().encrypt(payload, uuid4(), "google")
    bundle = {"ciphertext": base64.b64encode(ct).decode(),
              "nonce": base64.b64encode(nonce).decode()}
    store = StubGoogleStore(code=ent.IntegrationCode(
        code="ic1", user_id=uuid4(), code_challenge=challenge,
        payload=bundle, expires_at=_now() + timedelta(minutes=1),
        created_at=_now()))
    with pytest.raises(DomainError) as e:
        await _svc(store=store).complete(uuid4(), "ic1", verifier)
    assert e.value.code == "UNAUTHENTICATED"


async def test_sync_requires_connection():
    with pytest.raises(DomainError) as e:
        await _svc().sync(uuid4())
    assert e.value.code == "INTEGRATION_REQUIRED"
