from __future__ import annotations
from datetime import datetime
from typing import Optional
from uuid import UUID

from app.db import NotFoundError
from app.domain.entities import (
    GoogleIntegration, GoogleMessage, IntegrationCode, LedgerEntry,
)
from app.domain.pagination import Cursor, Page, new_page
from app.jsonutil import dump
from app.infrastructure.store_common import Conds, Store, to_model

_GI_COLS = ("user_id, access_ciphertext, access_nonce, refresh_ciphertext, refresh_nonce, "
            "scopes, token_expires_at, gmail_history_id, calendar_sync_token, "
            "last_synced_at, created_at, updated_at")
_GM_COLS = ("id, user_id, external_id, thread_id, application_id, sender, subject, "
            "snippet, received_at, created_at")


class IntegrationStore(Store):
    async def save_integration_code(self, c: IntegrationCode) -> None:
        await self.q().execute(
            "INSERT INTO integration_codes (code, user_id, code_challenge, payload, "
            "expires_at, used_at, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)",
            c.code, c.user_id, c.code_challenge, dump(c.payload), c.expires_at,
            c.used_at, c.created_at)

    async def get_integration_code(self, code: str) -> IntegrationCode:
        return to_model(IntegrationCode, await self.one(
            "SELECT code, user_id, code_challenge, payload, expires_at, used_at, created_at "
            "FROM integration_codes WHERE code = $1", code))

    async def mark_integration_code_used(self, code: str, at: datetime) -> None:
        tag = await self.q().execute(
            "UPDATE integration_codes SET used_at = $2 WHERE code = $1 AND used_at IS NULL",
            code, at)
        if tag.split()[-1] == "0":
            raise NotFoundError()

    async def put_google_integration(self, g: GoogleIntegration) -> None:
        await self.q().execute(
            f"INSERT INTO google_integrations ({_GI_COLS}) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) "
            "ON CONFLICT (user_id) DO UPDATE SET access_ciphertext = $2, access_nonce = $3, "
            "refresh_ciphertext = $4, refresh_nonce = $5, scopes = $6, token_expires_at = $7, "
            "gmail_history_id = $8, calendar_sync_token = $9, last_synced_at = $10, "
            "updated_at = $12",
            g.user_id, bytes(g.access_ciphertext), bytes(g.access_nonce),
            bytes(g.refresh_ciphertext) if g.refresh_ciphertext else None,
            bytes(g.refresh_nonce) if g.refresh_nonce else None, dump(g.scopes),
            g.token_expires_at, g.gmail_history_id, g.calendar_sync_token, g.last_synced_at,
            g.created_at, g.updated_at)

    async def get_google_integration(self, user_id: UUID) -> GoogleIntegration:
        return to_model(GoogleIntegration, await self.one(
            f"SELECT {_GI_COLS} FROM google_integrations WHERE user_id = $1", user_id))

    async def delete_google_integration(self, user_id: UUID) -> None:
        tag = await self.q().execute(
            "DELETE FROM google_integrations WHERE user_id = $1", user_id)
        if tag.split()[-1] == "0":
            raise NotFoundError()

    async def upsert_google_message(self, m: GoogleMessage) -> None:
        await self.q().execute(
            f"INSERT INTO google_messages ({_GM_COLS}) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) "
            "ON CONFLICT (user_id, external_id) DO UPDATE SET thread_id = $4, sender = $6, "
            "subject = $7, snippet = $8, received_at = $9",
            m.id, m.user_id, m.external_id, m.thread_id, m.application_id, m.sender,
            m.subject, m.snippet, m.received_at, m.created_at)

    async def get_google_message(self, user_id: UUID, id_: UUID) -> GoogleMessage:
        return to_model(GoogleMessage, await self.one(
            f"SELECT {_GM_COLS} FROM google_messages WHERE id = $1 AND user_id = $2",
            id_, user_id))

    async def list_google_messages(self, user_id: UUID, application_id: Optional[UUID],
                                   page) -> Page:
        c = Conds()
        c.add("user_id = {}", user_id)
        if application_id:
            c.add("application_id = {}", application_id)
        c.cursor(page)
        sql, args = c.query(_GM_COLS, "google_messages", page.effective_limit())
        rows = await self.q().fetch(sql, *args)
        return new_page([to_model(GoogleMessage, r) for r in rows], page.effective_limit(),
                        lambda m: Cursor(m.created_at, m.id))

    async def link_google_message(self, user_id: UUID, id_: UUID, application_id: UUID) -> None:
        tag = await self.q().execute(
            "UPDATE google_messages SET application_id = $3 WHERE id = $1 AND user_id = $2",
            id_, user_id, application_id)
        if tag.split()[-1] == "0":
            raise NotFoundError()

    async def delete_google_data(self, user_id: UUID) -> None:
        await self.q().execute("DELETE FROM google_messages WHERE user_id = $1", user_id)
        await self.q().execute(
            "DELETE FROM calendar_events WHERE user_id = $1 AND source = 'GOOGLE'", user_id)
        await self.q().execute("DELETE FROM google_integrations WHERE user_id = $1", user_id)


class BillingStore(Store):
    async def record_stripe_event(self, event_id: str, typ: str, at: datetime) -> bool:
        tag = await self.q().execute(
            "INSERT INTO stripe_events (event_id, type, processed_at) VALUES ($1,$2,$3) "
            "ON CONFLICT (event_id) DO NOTHING", event_id, typ, at)
        return tag.split()[-1] == "1"

    async def update_subscription(self, user_id: UUID, plan: str, status: str,
                                  customer_id: Optional[str], period_ends_at) -> None:
        await self.q().execute(
            "UPDATE users SET plan=$2, subscription_status=$3, "
            "stripe_customer_id=COALESCE($4, stripe_customer_id), "
            "period_ends_at=$5 WHERE id=$1",
            user_id, plan, status, customer_id, period_ends_at)

    async def update_subscription_by_customer(self, customer_id: str, status: str,
                                              period_ends_at) -> None:
        await self.q().execute(
            "UPDATE users SET subscription_status=$2, period_ends_at=$3 "
            "WHERE stripe_customer_id=$1", customer_id, status, period_ends_at)

    async def user_id_by_stripe_customer(self, customer_id: str) -> UUID:
        return await self.q().fetchval(
            "SELECT id FROM users WHERE stripe_customer_id=$1", customer_id)

    async def append_ledger(self, e: LedgerEntry) -> None:
        await self.q().execute(
            "INSERT INTO billing_ledger (id, user_id, type, amount_micro_credits, "
            "balance_after, reference_id, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)",
            e.id, e.user_id, e.type, e.amount_micro_credits, e.balance_after,
            e.reference_id, e.created_at)

    async def list_ledger(self, user_id: UUID, page) -> Page:
        c = Conds()
        c.add("user_id = {}", user_id)
        c.cursor(page)
        sql, args = c.query(
            "id, user_id, type, amount_micro_credits, balance_after, reference_id, created_at",
            "billing_ledger", page.effective_limit())
        rows = await self.q().fetch(sql, *args)
        return new_page([to_model(LedgerEntry, r) for r in rows], page.effective_limit(),
                        lambda e: Cursor(e.created_at, e.id))

    async def last_balance(self, user_id: UUID) -> int:
        n = await self.q().fetchval(
            "SELECT balance_after FROM billing_ledger WHERE user_id = $1 "
            "ORDER BY created_at DESC, id DESC LIMIT 1", user_id)
        return n or 0
