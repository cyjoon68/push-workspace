from __future__ import annotations
from datetime import datetime, timezone
from typing import Optional
from uuid import UUID, uuid4

from app.db import DB, NotFoundError
from app.domain import entities as ent
from app.domain.errors import (
    internal, map_revision_err, not_found, revision_conflict, validation_field,
)
from app.domain.services.ai import AIGate
from app.domain.validators import code_point_len
from app.graph.chat import build_chat_graph
from app.infrastructure.store_ai import AiUsageStore
from app.infrastructure.store_applications import ApplicationStore
from app.infrastructure.store_conversations import ConversationStore
from app.infrastructure.store_documents import DocumentStore
from app.infrastructure.store_evidence import EvidenceStore
from app.infrastructure.store_operations import OperationStore


def _now() -> datetime:
    return datetime.now(timezone.utc)


class ConversationService:
    def __init__(self, db: DB, ai: AIGate, checkpointer=None):
        self.db = db
        self.conversations = ConversationStore(db)
        self.applications = ApplicationStore(db)
        self.documents = DocumentStore(db)
        self.evidence = EvidenceStore(db)
        self.ops = OperationStore(db)
        self.ai = ai
        self.usage = AiUsageStore(db)
        self.graph = build_chat_graph(self, checkpointer)

    async def list(self, user_id: UUID, application_id: Optional[UUID], page):
        try:
            return await self.conversations.list(user_id, application_id, page)
        except Exception:
            raise internal()

    async def get(self, user_id: UUID, id_: UUID) -> ent.Conversation:
        try:
            return await self.conversations.get(user_id, id_)
        except NotFoundError:
            raise not_found()
        except Exception:
            raise internal()

    async def create(self, user_id: UUID, application_id: Optional[UUID],
                     title: str) -> ent.Conversation:
        if len(title) > 300:
            raise validation_field("title", "title must be at most 300 characters")
        if application_id is not None:
            try:
                await self.applications.get(user_id, application_id)
            except NotFoundError:
                raise not_found()
            except Exception:
                raise internal()
        now = _now()
        c = ent.Conversation(
            id=uuid4(), user_id=user_id, revision=1, application_id=application_id,
            title=title, created_at=now, updated_at=now)
        try:
            await self.conversations.create(c)
        except Exception:
            raise internal()
        return c

    async def patch(self, user_id: UUID, id_: UUID, expected: int,
                    title: Optional[str], pinned: Optional[bool]) -> ent.Conversation:
        out = None

        async def work():
            nonlocal out
            try:
                v = await self.conversations.get(user_id, id_)
            except NotFoundError:
                raise not_found()
            if v.revision != expected:
                raise revision_conflict(v.revision)
            if title is not None:
                if len(title) > 300:
                    raise validation_field(
                        "title", "title must be at most 300 characters")
                v.title = title
            if pinned is not None:
                v.pinned = pinned
            v.updated_at = _now()
            try:
                await self.conversations.update(v, expected)
            except Exception as err:
                raise map_revision_err(err)
            v.revision = expected + 1
            out = v

        await self.db.do(work)
        return out

    async def archive(self, user_id: UUID, id_: UUID,
                      expected: int) -> ent.Conversation:
        out = None

        async def work():
            nonlocal out
            try:
                v = await self.conversations.get(user_id, id_)
            except NotFoundError:
                raise not_found()
            if v.revision != expected:
                raise revision_conflict(v.revision)
            v.archived = True
            v.updated_at = _now()
            try:
                await self.conversations.update(v, expected)
            except Exception as err:
                raise map_revision_err(err)
            v.revision = expected + 1
            out = v

        await self.db.do(work)
        return out

    async def list_messages(self, user_id: UUID, conversation_id: UUID, page):
        await self.get(user_id, conversation_id)
        try:
            return await self.conversations.list_messages(user_id, conversation_id, page)
        except Exception:
            raise internal()

    async def post_message(self, user_id: UUID, conversation_id: UUID, text: str,
                           context: dict, ai: Optional[ent.AiOptions],
                           access_mode: str) -> ent.Operation:
        if not text or code_point_len(text) > 20000:
            raise validation_field("text", "text must be 1-20000 characters")
        if not ent.valid_access_mode(access_mode):
            raise validation_field(
                "accessMode", "must be SUGGEST or CONFIRM_ACTIONS")
        result = await self.graph.ainvoke(
            {
                "user_id": user_id,
                "conversation_id": conversation_id,
                "text": text,
                "context": context or {},
                "ai": ai.model_dump(by_alias=True) if ai else None,
                "access_mode": access_mode,
            },
            config={"configurable": {"thread_id": str(conversation_id)}},
        )
        return result["operation"]
