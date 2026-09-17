from __future__ import annotations
from datetime import datetime
from typing import Optional
from uuid import UUID

from app.db import unique_violation
from app.domain.entities import (
    AccessToken, ExchangeCode, IdempotencyRecord, OAuthState, RefreshToken, User,
)
from app.infrastructure.store_common import Store, to_model

_USER_COLS = ("id, provider, provider_subject, display_name, locale, plan, "
              "subscription_status, stripe_customer_id, period_ends_at, created_at")


class UserStore(Store):
    async def create(self, u: User) -> None:
        await self.q().execute(
            "INSERT INTO users (id, provider, provider_subject, display_name, locale, created_at) "
            "VALUES ($1,$2,$3,$4,$5,$6)",
            u.id, u.provider, u.provider_subject, u.display_name, u.locale, u.created_at)

    async def get(self, id_: UUID) -> User:
        return to_model(User, await self.one(f"SELECT {_USER_COLS} FROM users WHERE id = $1", id_))

    async def get_by_provider(self, provider: str, subject: str) -> User:
        return to_model(User, await self.one(
            f"SELECT {_USER_COLS} FROM users WHERE provider = $1 AND provider_subject = $2",
            provider, subject))

    async def upsert_dev_user(self, id_: UUID) -> None:
        await self.q().execute(
            "INSERT INTO users (id, provider, provider_subject, display_name, locale, created_at) "
            "VALUES ($1,'dev','dev','Dev User','ko',now()) ON CONFLICT (id) DO NOTHING", id_)

    async def set_billing(self, user_id: UUID, plan: str, status: str,
                          customer_id: Optional[str], period_ends_at) -> None:
        await self.q().execute(
            "UPDATE users SET plan=$2, subscription_status=$3, stripe_customer_id=$4, "
            "period_ends_at=$5 WHERE id=$1",
            user_id, plan, status, customer_id, period_ends_at)

    async def set_stripe_customer(self, user_id: UUID, customer_id: str) -> None:
        await self.q().execute(
            "UPDATE users SET stripe_customer_id=$2 WHERE id=$1", user_id, customer_id)


class SessionStore(Store):
    async def save_oauth_state(self, s: OAuthState) -> None:
        await self.q().execute(
            "INSERT INTO oauth_states (state, provider, code_challenge, redirect_uri, "
            "expires_at, created_at, user_id, purpose) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)",
            s.state, s.provider, s.code_challenge, s.redirect_uri, s.expires_at,
            s.created_at, s.user_id, s.purpose or "LOGIN")

    async def get_oauth_state(self, state: str) -> OAuthState:
        return to_model(OAuthState, await self.one(
            "SELECT state, provider, code_challenge, redirect_uri, expires_at, created_at, "
            "user_id, purpose FROM oauth_states WHERE state = $1", state))

    async def delete_oauth_state(self, state: str) -> None:
        await self.q().execute("DELETE FROM oauth_states WHERE state = $1", state)

    async def save_exchange_code(self, c: ExchangeCode) -> None:
        await self.q().execute(
            "INSERT INTO exchange_codes (code, user_id, code_challenge, expires_at, used_at, "
            "created_at) VALUES ($1,$2,$3,$4,$5,$6)",
            c.code, c.user_id, c.code_challenge, c.expires_at, c.used_at, c.created_at)

    async def get_exchange_code(self, code: str) -> ExchangeCode:
        return to_model(ExchangeCode, await self.one(
            "SELECT code, user_id, code_challenge, expires_at, used_at, created_at "
            "FROM exchange_codes WHERE code = $1", code))

    async def mark_exchange_code_used(self, code: str, at: datetime) -> None:
        tag = await self.q().execute(
            "UPDATE exchange_codes SET used_at = $2 WHERE code = $1 AND used_at IS NULL", code, at)
        if tag.split()[-1] == "0":
            from app.db import NotFoundError
            raise NotFoundError()

    async def save_access_token(self, t: AccessToken) -> None:
        await self.q().execute(
            "INSERT INTO access_tokens (id, user_id, token_hash, refresh_token_id, expires_at, "
            "created_at) VALUES ($1,$2,$3,$4,$5,$6)",
            t.id, t.user_id, t.token_hash, t.refresh_token_id, t.expires_at, t.created_at)

    async def get_access_token(self, token_hash: str) -> AccessToken:
        return to_model(AccessToken, await self.one(
            "SELECT id, user_id, token_hash, refresh_token_id, expires_at, created_at "
            "FROM access_tokens WHERE token_hash = $1", token_hash))

    async def save_refresh_token(self, t: RefreshToken) -> None:
        await self.q().execute(
            "INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, revoked_at, "
            "created_at) VALUES ($1,$2,$3,$4,$5,$6)",
            t.id, t.user_id, t.token_hash, t.expires_at, t.revoked_at, t.created_at)

    async def get_refresh_token(self, token_hash: str) -> RefreshToken:
        return to_model(RefreshToken, await self.one(
            "SELECT id, user_id, token_hash, expires_at, revoked_at, created_at "
            "FROM refresh_tokens WHERE token_hash = $1", token_hash))

    async def revoke_refresh_token(self, id_: UUID, at: datetime) -> None:
        await self.q().execute(
            "UPDATE refresh_tokens SET revoked_at = $2 WHERE id = $1 AND revoked_at IS NULL",
            id_, at)

    async def revoke_access_tokens_for_refresh(self, refresh_id: UUID, at: datetime) -> None:
        await self.q().execute(
            "UPDATE access_tokens SET expires_at = $2 WHERE refresh_token_id = $1",
            refresh_id, at)


class IdempotencyStore(Store):
    async def insert_pending(self, rec: IdempotencyRecord) -> bool:
        try:
            await self.q().execute(
                "INSERT INTO idempotency_keys (id, user_id, method, path, key, request_hash, "
                "created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)",
                rec.id, rec.user_id, rec.method, rec.path, rec.key, rec.request_hash,
                rec.created_at)
            return True
        except Exception as err:
            if unique_violation(err):
                return False
            raise

    async def get(self, user_id: UUID, method: str, path: str, key: UUID) -> IdempotencyRecord:
        return to_model(IdempotencyRecord, await self.one(
            "SELECT id, user_id, key, method, path, request_hash, response_status, "
            "response_body, created_at FROM idempotency_keys "
            "WHERE user_id = $1 AND method = $2 AND path = $3 AND key = $4",
            user_id, method, path, key))

    async def complete(self, id_: UUID, status: int, body: bytes) -> None:
        await self.q().execute(
            "UPDATE idempotency_keys SET response_status = $2, response_body = $3 WHERE id = $1",
            id_, status, body)

    async def delete_pending(self, id_: UUID) -> None:
        await self.q().execute(
            "DELETE FROM idempotency_keys WHERE id = $1 AND response_status IS NULL", id_)
