from __future__ import annotations
from datetime import datetime, timedelta, timezone
from typing import Optional
from urllib.parse import quote, urlencode
from uuid import UUID, uuid4

from app.db import DB, NotFoundError
from app.domain import entities as ent
from app.domain.errors import (
    internal, not_configured, not_found, provider_error,
    token_expired, unauthenticated, validation_field,
)
from app.infrastructure.crypto import random_token, token_hash, verify_pkce
from app.infrastructure.oauth import OAuthClient
from app.infrastructure.store_auth import SessionStore, UserStore


def _now() -> datetime:
    return datetime.now(timezone.utc)


class AuthService:
    def __init__(self, db: DB, oauth: OAuthClient, providers: dict,
                 app_env: str, dev_token: str, dev_user_id: UUID):
        self.db = db
        self.users = UserStore(db)
        self.sessions = SessionStore(db)
        self.oauth = oauth
        self.providers = providers
        self.app_env = app_env
        self.dev_token = dev_token
        self.dev_user_id = dev_user_id
        self.allowed_uris = {ent.AUTH_CALLBACK_URI}

    async def start_oauth(self, provider: str, code_challenge: str, method: str,
                          redirect_uri: str, provider_callback_url: str) -> tuple[str, str, datetime]:
        if not ent.valid_oauth_provider(provider):
            raise not_found()
        cfg = self.providers.get(provider)
        if cfg is None or not cfg.client_id or not cfg.client_secret:
            raise not_configured(f"oauth provider {provider} is not configured")
        if method != "S256":
            raise validation_field("codeChallengeMethod", "must be S256")
        if not code_challenge:
            raise validation_field("codeChallenge", "required")
        if redirect_uri not in self.allowed_uris:
            raise validation_field("redirectUri", "not allowed")
        state = random_token()
        now = _now()
        rec = ent.OAuthState(state=state, provider=provider, code_challenge=code_challenge,
                             redirect_uri=provider_callback_url,
                             expires_at=now + ent.OAUTH_STATE_TTL, created_at=now)
        await self.sessions.save_oauth_state(rec)
        q = {"client_id": cfg.client_id, "redirect_uri": provider_callback_url,
             "response_type": "code", "state": state}
        if provider == "google":
            q["scope"] = "openid email profile"
            q["access_type"] = "offline"
        else:
            q["scope"] = "read:user user:email"
        return cfg.auth_url + "?" + urlencode(q), state, rec.expires_at

    async def handle_callback(self, provider: str, code: str, state: str) -> str:
        if not ent.valid_oauth_provider(provider):
            raise not_found()
        cfg = self.providers.get(provider)
        if cfg is None or not cfg.client_id:
            raise not_configured(f"oauth provider {provider} is not configured")
        try:
            rec = await self.sessions.get_oauth_state(state)
        except NotFoundError:
            raise unauthenticated("invalid state")
        now = _now()
        await self.sessions.delete_oauth_state(state)
        if now > rec.expires_at or rec.provider != provider:
            raise unauthenticated("state expired")
        try:
            access_token = await self.oauth.exchange_code(cfg, code, rec.redirect_uri)
        except Exception:
            raise provider_error("provider token exchange failed")
        try:
            subject, display_name = await self.oauth.fetch_identity(cfg, access_token)
        except Exception:
            raise provider_error("provider identity fetch failed")

        exchange_code = ""

        async def work():
            nonlocal exchange_code
            try:
                user = await self.users.get_by_provider(provider, subject)
            except NotFoundError:
                user = ent.User(id=uuid4(), provider=provider, provider_subject=subject,
                                display_name=display_name, locale="ko", created_at=now)
                await self.users.create(user)
            exchange_code = random_token()
            await self.sessions.save_exchange_code(ent.ExchangeCode(
                code=exchange_code, user_id=user.id, code_challenge=rec.code_challenge,
                expires_at=now + ent.EXCHANGE_CODE_TTL, created_at=now))

        try:
            await self.db.do(work)
        except Exception:
            raise internal()
        return ent.AUTH_CALLBACK_URI + "?code=" + quote(exchange_code)

    async def _issue_session(self, user_id: UUID, now: datetime) -> ent.Session:
        access, refresh = random_token(), random_token()
        rt = ent.RefreshToken(id=uuid4(), user_id=user_id, token_hash=token_hash(refresh),
                              expires_at=now + ent.REFRESH_TOKEN_TTL, created_at=now)
        await self.sessions.save_refresh_token(rt)
        at = ent.AccessToken(
            id=uuid4(), user_id=user_id, token_hash=token_hash(access),
            refresh_token_id=rt.id,
            expires_at=now + timedelta(seconds=ent.ACCESS_TOKEN_TTL_SECONDS),
            created_at=now)
        await self.sessions.save_access_token(at)
        user = await self.users.get(user_id)
        return ent.Session(access_token=access, refresh_token=refresh,
                           expires_in=ent.ACCESS_TOKEN_TTL_SECONDS, user=user)

    async def exchange(self, code: str, code_verifier: str) -> ent.Session:
        try:
            rec = await self.sessions.get_exchange_code(code)
        except NotFoundError:
            raise unauthenticated("invalid code")
        now = _now()
        if rec.used_at is not None or now > rec.expires_at:
            raise unauthenticated("code expired or already used")
        if not code_verifier or not verify_pkce(code_verifier, rec.code_challenge):
            raise unauthenticated("pkce verification failed")

        session: Optional[ent.Session] = None

        async def work():
            nonlocal session
            await self.sessions.mark_exchange_code_used(code, now)
            session = await self._issue_session(rec.user_id, now)

        try:
            await self.db.do(work)
        except Exception:
            raise internal()
        return session

    async def refresh(self, refresh_token: str) -> ent.Session:
        try:
            rec = await self.sessions.get_refresh_token(token_hash(refresh_token))
        except NotFoundError:
            raise unauthenticated("invalid refresh token")
        now = _now()
        if rec.revoked_at is not None:
            raise unauthenticated("refresh token revoked")
        if now > rec.expires_at:
            raise token_expired()

        session: Optional[ent.Session] = None

        async def work():
            nonlocal session
            await self.sessions.revoke_refresh_token(rec.id, now)
            await self.sessions.revoke_access_tokens_for_refresh(rec.id, now)
            session = await self._issue_session(rec.user_id, now)

        try:
            await self.db.do(work)
        except Exception:
            raise internal()
        return session

    async def logout(self, refresh_token: str) -> None:
        if not refresh_token:
            return
        try:
            rec = await self.sessions.get_refresh_token(token_hash(refresh_token))
        except Exception:
            return
        now = _now()
        await self.sessions.revoke_refresh_token(rec.id, now)
        await self.sessions.revoke_access_tokens_for_refresh(rec.id, now)

    async def resolve_access_token(self, token: str) -> ent.User:
        if self.app_env == "development" and self.dev_token and token == self.dev_token:
            try:
                return await self.users.get(self.dev_user_id)
            except NotFoundError:
                raise unauthenticated("dev user not seeded")
        try:
            rec = await self.sessions.get_access_token(token_hash(token))
        except NotFoundError:
            raise unauthenticated("invalid access token")
        if _now() > rec.expires_at:
            raise token_expired()
        try:
            return await self.users.get(rec.user_id)
        except NotFoundError:
            raise unauthenticated("invalid access token")

    async def me(self, user_id: UUID) -> ent.User:
        try:
            return await self.users.get(user_id)
        except NotFoundError:
            raise not_found()
