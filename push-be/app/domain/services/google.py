from __future__ import annotations
import base64
import json
from datetime import datetime, timedelta, timezone
from typing import Optional
from urllib.parse import urlencode
from uuid import UUID, uuid4

from app.db import DB, NotFoundError
from app.domain import entities as ent
from app.domain.errors import (
    DomainError, integration_required, internal, not_configured, not_found,
    provider_error, unauthenticated, validation_field,
)
from app.infrastructure.crypto import random_token, verify_pkce
from app.infrastructure.google_client import SyncTokenInvalid
from app.infrastructure.store_auth import SessionStore
from app.infrastructure.store_calendar import CalendarEventStore
from app.infrastructure.store_integrations import IntegrationStore
from app.infrastructure.store_applications import ApplicationStore
from app.infrastructure.store_operations import OperationStore
from app.jsonutil import to_jsonable

GOOGLE_CIPHER_PURPOSE = "google"
GOOGLE_SCOPE_GMAIL = "https://www.googleapis.com/auth/gmail.readonly"
GOOGLE_SCOPE_CALENDAR = "https://www.googleapis.com/auth/calendar.events"
GOOGLE_CALLBACK_URI = "push://integrations/google/callback"
OAUTH_STATE_TTL = timedelta(minutes=10)
INTEGRATION_CODE_TTL = timedelta(seconds=60)


def _now() -> datetime:
    return datetime.now(timezone.utc)


class GoogleService:
    def __init__(self, db: DB, cipher, gapi, client_id: str,
                 client_secret: str, auth_url: str, gmail_beta: bool):
        self.db = db
        self.store = IntegrationStore(db)
        self.sessions = SessionStore(db)
        self.calendar = CalendarEventStore(db)
        self.applications = ApplicationStore(db)
        self.ops = OperationStore(db)
        self.cipher = cipher
        self.gapi = gapi
        self.client_id = client_id
        self.client_secret = client_secret
        self.auth_url = auth_url
        self.gmail_beta = gmail_beta

    @property
    def configured(self) -> bool:
        return bool(self.client_id and self.client_secret)

    def _check(self) -> None:
        if not self.configured:
            raise not_configured("google integration is not configured")
        if self.cipher is None:
            raise not_configured("token encryption is not configured")

    async def status(self, user_id: UUID) -> ent.GoogleStatus:
        st = ent.GoogleStatus(
            enabled=self.configured, connected=False,
            gmail_status="DISCONNECTED" if self.gmail_beta else "DISABLED",
            calendar_status="DISCONNECTED")
        try:
            gi = await self.store.get_google_integration(user_id)
        except NotFoundError:
            return st
        except Exception:
            raise internal()
        st.connected = True
        st.scopes = gi.scopes
        st.last_synced_at = gi.last_synced_at
        if self.gmail_beta and gi.has_scope(GOOGLE_SCOPE_GMAIL):
            st.gmail_status = "CONNECTED"
        if gi.has_scope(GOOGLE_SCOPE_CALENDAR):
            st.calendar_status = "CONNECTED"
        return st

    async def connect(self, user_id: UUID, code_challenge: str, method: str,
                      redirect_uri: str, callback_url: str):
        self._check()
        if method != "S256":
            raise validation_field("codeChallengeMethod", "must be S256")
        if not code_challenge:
            raise validation_field("codeChallenge", "required")
        if redirect_uri != GOOGLE_CALLBACK_URI:
            raise validation_field("redirectUri", "not allowed")
        state = random_token()
        now = _now()
        try:
            await self.sessions.save_oauth_state(ent.OAuthState(
                state=state, provider="google", purpose="GOOGLE",
                user_id=user_id, code_challenge=code_challenge,
                redirect_uri=callback_url,
                expires_at=now + OAUTH_STATE_TTL, created_at=now))
        except Exception:
            raise internal()
        q = urlencode({
            "client_id": self.client_id,
            "redirect_uri": callback_url,
            "response_type": "code",
            "state": state,
            "scope": GOOGLE_SCOPE_GMAIL + " " + GOOGLE_SCOPE_CALENDAR,
            "access_type": "offline",
            "prompt": "consent"})
        return self.auth_url + "?" + q, state, now + OAUTH_STATE_TTL

    async def handle_callback(self, code: str, state: str) -> str:
        self._check()
        try:
            rec = await self.sessions.get_oauth_state(state)
        except NotFoundError:
            raise unauthenticated("invalid state")
        except Exception:
            raise internal()
        now = _now()
        await self.sessions.delete_oauth_state(state)
        if (now > rec.expires_at or rec.purpose != "GOOGLE"
                or rec.user_id is None):
            raise unauthenticated("state expired")
        try:
            tokens = await self.gapi.exchange_code(code, rec.redirect_uri)
        except Exception:
            raise provider_error("provider token exchange failed")
        payload = json.dumps({
            "accessToken": tokens.access_token,
            "refreshToken": tokens.refresh_token,
            "scope": tokens.scope,
            "expiresIn": tokens.expires_in})
        try:
            ct, nonce = self.cipher.encrypt(
                payload, rec.user_id, GOOGLE_CIPHER_PURPOSE)
        except Exception:
            raise internal()
        bundle = {
            "ciphertext": base64.b64encode(ct).decode(),
            "nonce": base64.b64encode(nonce).decode()}
        integration_code = random_token()
        try:
            await self.store.save_integration_code(ent.IntegrationCode(
                code=integration_code, user_id=rec.user_id,
                code_challenge=rec.code_challenge, payload=bundle,
                expires_at=now + INTEGRATION_CODE_TTL, created_at=now))
        except Exception:
            raise internal()
        return GOOGLE_CALLBACK_URI + "?code=" + integration_code

    async def complete(self, user_id: UUID, code: str,
                       code_verifier: str) -> ent.GoogleStatus:
        self._check()
        try:
            rec = await self.store.get_integration_code(code)
        except NotFoundError:
            raise unauthenticated("invalid code")
        except Exception:
            raise internal()
        now = _now()
        if rec.used_at is not None or now > rec.expires_at:
            raise unauthenticated("code expired or already used")
        if rec.user_id != user_id:
            raise unauthenticated("code belongs to a different account")
        if not code_verifier or not verify_pkce(code_verifier, rec.code_challenge):
            raise unauthenticated("pkce verification failed")
        try:
            ct = base64.b64decode(rec.payload["ciphertext"])
            nonce = base64.b64decode(rec.payload["nonce"])
            raw = self.cipher.decrypt(
                ct, nonce, user_id, GOOGLE_CIPHER_PURPOSE)
            payload = json.loads(raw)
        except Exception:
            raise internal()
        try:
            act, an = self.cipher.encrypt(
                payload["accessToken"], user_id, GOOGLE_CIPHER_PURPOSE)
        except Exception:
            raise internal()
        gi = ent.GoogleIntegration(
            user_id=user_id, access_ciphertext=act, access_nonce=an,
            scopes=payload.get("scope", "").split(),
            created_at=now, updated_at=now)
        if payload.get("expiresIn"):
            gi.token_expires_at = now + timedelta(
                seconds=payload["expiresIn"])
        if payload.get("refreshToken"):
            rct, rn = self.cipher.encrypt(
                payload["refreshToken"], user_id, GOOGLE_CIPHER_PURPOSE)
            gi.refresh_ciphertext = rct
            gi.refresh_nonce = rn

        async def work():
            try:
                await self.store.mark_integration_code_used(code, now)
            except Exception:
                raise unauthenticated("code expired or already used")
            await self.store.put_google_integration(gi)

        try:
            await self.db.do(work)
        except DomainError:
            raise
        except Exception:
            raise internal()
        return await self.status(user_id)

    async def _access_token(self, gi: ent.GoogleIntegration) -> str:
        try:
            at = self.cipher.decrypt(
                gi.access_ciphertext, gi.access_nonce, gi.user_id,
                GOOGLE_CIPHER_PURPOSE)
        except Exception:
            raise internal()
        expired = (gi.token_expires_at is not None
                   and _now() > gi.token_expires_at - timedelta(minutes=1))
        if not expired or not gi.refresh_ciphertext:
            return at
        try:
            rt = self.cipher.decrypt(
                gi.refresh_ciphertext, gi.refresh_nonce, gi.user_id,
                GOOGLE_CIPHER_PURPOSE)
            tokens = await self.gapi.refresh_access_token(rt)
        except Exception:
            raise provider_error("google token refresh failed")
        ct, nonce = self.cipher.encrypt(
            tokens.access_token, gi.user_id, GOOGLE_CIPHER_PURPOSE)
        gi.access_ciphertext = ct
        gi.access_nonce = nonce
        if tokens.expires_in > 0:
            gi.token_expires_at = _now() + timedelta(
                seconds=tokens.expires_in)
        try:
            await self.store.put_google_integration(gi)
        except Exception:
            raise internal()
        return tokens.access_token

    async def disconnect(self, user_id: UUID) -> None:
        try:
            gi = await self.store.get_google_integration(user_id)
        except NotFoundError:
            return
        except Exception:
            raise internal()
        try:
            at = self.cipher.decrypt(
                gi.access_ciphertext, gi.access_nonce, user_id,
                GOOGLE_CIPHER_PURPOSE)
            await self.gapi.revoke_token(at)
        except Exception:
            if gi.refresh_ciphertext:
                try:
                    rt = self.cipher.decrypt(
                        gi.refresh_ciphertext, gi.refresh_nonce, user_id,
                        GOOGLE_CIPHER_PURPOSE)
                    await self.gapi.revoke_token(rt)
                except Exception:
                    pass

        async def work():
            await self.store.delete_google_data(user_id)
            await self.store.delete_google_integration(user_id)

        try:
            await self.db.do(work)
        except Exception:
            raise internal()

    async def sync(self, user_id: UUID) -> ent.Operation:
        try:
            gi = await self.store.get_google_integration(user_id)
        except NotFoundError:
            raise integration_required("google account is not connected")
        except Exception:
            raise internal()
        now = _now()
        op = ent.Operation(
            id=uuid4(), user_id=user_id, type=ent.OP_GOOGLE_SYNC,
            status=ent.OP_RUNNING, created_at=now, updated_at=now)
        try:
            await self.ops.create(op)
        except Exception:
            raise internal()

        async def fail(code: str, msg: str):
            op.status = ent.OP_FAILED
            op.error = ent.OperationError(code=code, message=msg,
                                          retryable=True)
            op.updated_at = _now()
            try:
                await self.ops.update(op)
            except Exception:
                pass
            raise provider_error(msg)

        try:
            token = await self._access_token(gi)
        except DomainError:
            await fail("PROVIDER_ERROR", "google access token unavailable")

        messages = []
        events = []
        history_id = gi.gmail_history_id
        sync_token = gi.calendar_sync_token
        if self.gmail_beta and gi.has_scope(GOOGLE_SCOPE_GMAIL):
            page_token = ""
            for _ in range(5):
                try:
                    ids, nxt, hid = await self.gapi.list_message_ids(
                        token, page_token)
                except Exception:
                    await fail("PROVIDER_ERROR", "gmail message list failed")
                for mid in ids:
                    try:
                        m = await self.gapi.get_message(token, mid)
                    except Exception:
                        await fail("PROVIDER_ERROR", "gmail message fetch failed")
                    messages.append(m)
                if hid and hid != "0":
                    history_id = hid
                if not nxt:
                    break
                page_token = nxt
        if gi.has_scope(GOOGLE_SCOPE_CALENDAR):
            try:
                events, nst = await self.gapi.list_events(token, sync_token)
            except SyncTokenInvalid:
                try:
                    events, nst = await self.gapi.list_events(token, "")
                except Exception:
                    await fail("PROVIDER_ERROR", "calendar event list failed")
            except Exception:
                await fail("PROVIDER_ERROR", "calendar event list failed")
            if nst:
                sync_token = nst
        gi.gmail_history_id = history_id
        gi.calendar_sync_token = sync_token
        gi.last_synced_at = now
        gi.updated_at = now

        async def work():
            for m in messages:
                await self.store.upsert_google_message(ent.GoogleMessage(
                    id=uuid4(), user_id=user_id, external_id=m.id,
                    thread_id=m.thread_id, sender=m.from_,
                    subject=m.subject, snippet=m.snippet,
                    received_at=m.received_at, created_at=now))
            for ev in events:
                if ev.status == "cancelled":
                    await self.calendar.delete_external(user_id, ev.id)
                    continue
                tz = ev.time_zone or "UTC"
                ends = ev.ends_at
                if ends <= ev.starts_at:
                    ends = ev.starts_at + timedelta(hours=1)
                await self.calendar.upsert_external(ent.CalendarEvent(
                    id=uuid4(), user_id=user_id, revision=1,
                    type="CUSTOM", title=ev.summary,
                    starts_at=ev.starts_at, ends_at=ends, time_zone=tz,
                    source="GOOGLE", external_id=ev.id,
                    created_at=now, updated_at=now))
            await self.store.put_google_integration(gi)
            op.status = ent.OP_SUCCEEDED
            op.result = ent.OperationResult(
                kind=ent.OP_GOOGLE_SYNC,
                value={"messages": len(messages), "events": len(events),
                       "lastSyncedAt": to_jsonable(now)})
            op.updated_at = now
            await self.ops.update(op)

        try:
            await self.db.do(work)
        except Exception:
            await fail("INTERNAL", "google sync persistence failed")
        return op

    async def _require_connection(self, user_id: UUID) -> None:
        try:
            await self.store.get_google_integration(user_id)
        except NotFoundError:
            raise integration_required("google account is not connected")
        except Exception:
            raise internal()

    async def list_messages(self, user_id: UUID,
                            application_id: Optional[UUID], page):
        await self._require_connection(user_id)
        try:
            return await self.store.list_google_messages(
                user_id, application_id, page)
        except Exception:
            raise internal()

    async def list_events(self, user_id: UUID, from_: Optional[datetime],
                          to: Optional[datetime], page):
        await self._require_connection(user_id)
        try:
            return await self.calendar.list(
                user_id, None, from_, to, "GOOGLE", page)
        except Exception:
            raise internal()

    async def link_message(self, user_id: UUID, message_id: UUID,
                           application_id: UUID) -> ent.GoogleMessage:
        await self._require_connection(user_id)
        try:
            m = await self.store.get_google_message(user_id, message_id)
        except NotFoundError:
            raise not_found()
        except Exception:
            raise internal()
        try:
            await self.applications.get(user_id, application_id)
        except NotFoundError:
            raise not_found()
        except Exception:
            raise internal()
        try:
            await self.store.link_google_message(user_id, message_id, application_id)
        except Exception:
            raise internal()
        m.application_id = application_id
        return m
