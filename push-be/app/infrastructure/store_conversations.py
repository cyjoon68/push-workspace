from __future__ import annotations
from typing import Optional
from uuid import UUID

from app.domain.entities import Conversation, Message
from app.domain.pagination import Cursor, Page, new_page
from app.jsonutil import dump
from app.infrastructure.store_common import Conds, Store, list_page, to_model

_CONV_COLS = ("id, user_id, revision, application_id, title, pinned, archived, "
              "created_at, updated_at")
_MSG_COLS = ("id, user_id, conversation_id, role, text, attachments, operation_id, created_at")


def _msg(row) -> Message:
    m = dict(row)
    m["attachments"] = m.get("attachments") or []
    return Message(**m)


class ConversationStore(Store):
    async def create(self, c: Conversation) -> None:
        await self.q().execute(
            f"INSERT INTO conversations ({_CONV_COLS}) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)",
            c.id, c.user_id, c.revision, c.application_id, c.title, c.pinned, c.archived,
            c.created_at, c.updated_at)

    async def get(self, user_id: UUID, id_: UUID) -> Conversation:
        return to_model(Conversation, await self.one(
            f"SELECT {_CONV_COLS} FROM conversations WHERE id = $1 AND user_id = $2",
            id_, user_id))

    async def list(self, user_id: UUID, application_id: Optional[UUID], page) -> Page:
        c = Conds()
        c.add("user_id = {}", user_id)
        c.add("archived = {}", False)
        if application_id:
            c.add("application_id = {}", application_id)
        c.cursor(page)
        sql, args = c.query(_CONV_COLS, "conversations", page.effective_limit())
        rows = await self.q().fetch(sql, *args)
        return list_page(rows, page, Conversation, lambda c: Cursor(c.created_at, c.id))

    async def update(self, c: Conversation, expected: int) -> None:
        await self.guard_update(
            "conversations", c.id, c.user_id,
            "UPDATE conversations SET revision = revision + 1, application_id = $3, "
            "title = $4, pinned = $5, archived = $6, updated_at = $7 "
            "WHERE id = $1 AND user_id = $2 AND revision = $8",
            c.id, c.user_id, c.application_id, c.title, c.pinned, c.archived,
            c.updated_at, expected)

    async def create_message(self, m: Message) -> None:
        await self.q().execute(
            f"INSERT INTO messages ({_MSG_COLS}) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)",
            m.id, m.user_id, m.conversation_id, m.role, m.text, dump(m.attachments),
            m.operation_id, m.created_at)

    async def list_messages(self, user_id: UUID, conversation_id: UUID, page) -> Page:
        c = Conds()
        c.add("user_id = {}", user_id)
        c.add("conversation_id = {}", conversation_id)
        c.cursor(page)
        sql, args = c.query(_MSG_COLS, "messages", page.effective_limit())
        rows = await self.q().fetch(sql, *args)
        return new_page([_msg(r) for r in rows], page.effective_limit(),
                        lambda m: Cursor(m.created_at, m.id))
