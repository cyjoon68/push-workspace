from __future__ import annotations
from typing import Any, Optional, TypedDict
from uuid import UUID, uuid4
from datetime import datetime, timezone

from langgraph.graph import END, START, StateGraph

from app.domain import entities as ent
from app.domain.errors import (
    invalid_transition, not_found, validation_field,
)
from app.db import NotFoundError
from app.jsonutil import to_jsonable


def _now() -> datetime:
    return datetime.now(timezone.utc)


def stub_reply(text: str) -> str:
    if len(text) > 80:
        text = text[:80]
    return f"(로컬 스텁) 메시지를 받았습니다: {text}"


class ChatState(TypedDict, total=False):
    user_id: Any
    conversation_id: Any
    text: str
    context: dict
    ai: Optional[dict]
    access_mode: str
    conversation: Any
    attachments: list
    completion: Any
    operation: Any


def build_chat_graph(svc, checkpointer=None):
    async def resolve_context(state: ChatState) -> dict:
        user_id = state["user_id"]
        try:
            conv = await svc.conversations.get(user_id, state["conversation_id"])
        except NotFoundError:
            raise not_found()
        if conv.archived:
            raise invalid_transition("conversation is archived")
        ctx = state.get("context") or {}
        attachments = []
        document_id = ctx.get("documentId")
        version_id = ctx.get("versionId")
        if document_id or version_id:
            doc = None
            version = None
            if document_id:
                try:
                    doc = await svc.documents.get(user_id, UUID(document_id))
                except Exception:
                    raise validation_field("context.documentId", "document not found")
                if (conv.application_id is not None
                        and doc.application_id != conv.application_id):
                    raise validation_field(
                        "context.documentId",
                        "document belongs to another application")
                if version_id:
                    try:
                        version = await svc.documents.get_version(
                            user_id, doc.id, UUID(version_id))
                    except Exception:
                        raise validation_field(
                            "context.versionId", "version not found in document")
                else:
                    if doc.latest_version_id is None:
                        raise validation_field(
                            "context.documentId", "document has no versions")
                    try:
                        version = await svc.documents.get_version(
                            user_id, doc.id, doc.latest_version_id)
                    except Exception:
                        from app.domain.errors import internal
                        raise internal()
            else:
                try:
                    version = await svc.documents.get_version_by_id(
                        user_id, UUID(version_id))
                except NotFoundError:
                    raise validation_field("context.versionId", "version not found")
                except Exception:
                    from app.domain.errors import internal
                    raise internal()
                try:
                    doc = await svc.documents.get(user_id, version.document_id)
                except Exception:
                    from app.domain.errors import internal
                    raise internal()
            if (conv.application_id is not None
                    and doc.application_id != conv.application_id):
                raise validation_field(
                    "context.documentId",
                    "document belongs to another application")
            attachments.append(ent.MessageAttachment(
                type="DOCUMENT_VERSION", id=version.id, document_id=doc.id,
                title=doc.title))
        for eid in ctx.get("evidenceIds") or []:
            try:
                e = await svc.evidence.get(user_id, UUID(eid))
            except Exception:
                raise validation_field(
                    "context.evidenceIds", "evidence " + str(eid) + " not found")
            attachments.append(ent.MessageAttachment(
                type="EVIDENCE", id=e.id, title=e.title))
        return {"conversation": conv, "attachments": attachments}

    async def generate_reply(state: ChatState) -> dict:
        ai = state.get("ai")
        if ai is None:
            return {"completion": None}
        opts = ent.AiOptions(**ai)
        completion = await svc.ai.complete(state["user_id"], opts, "", state["text"])
        return {"completion": completion}

    async def persist(state: ChatState) -> dict:
        user_id = state["user_id"]
        conv = state["conversation"]
        attachments = state["attachments"]
        completion = state.get("completion")
        now = _now()
        op_id = uuid4()
        user_msg = ent.Message(
            id=uuid4(), user_id=user_id, conversation_id=conv.id,
            role="USER", text=state["text"], attachments=attachments,
            operation_id=op_id, created_at=now)
        reply = stub_reply(state["text"]) if completion is None else completion.text
        assistant_msg = ent.Message(
            id=uuid4(), user_id=user_id, conversation_id=conv.id,
            role="ASSISTANT", text=reply, attachments=[],
            operation_id=op_id, created_at=now)
        op = ent.Operation(
            id=op_id, user_id=user_id, type=ent.OP_CHAT_MESSAGE,
            application_id=conv.application_id, status=ent.OP_SUCCEEDED,
            result=ent.OperationResult(kind=ent.OP_CHAT_MESSAGE, value={
                "userMessage": to_jsonable(user_msg),
                "assistantMessage": to_jsonable(assistant_msg),
                "approvalIds": []}),
            created_at=now, updated_at=now)

        async def work():
            await svc.conversations.create_message(user_msg)
            await svc.conversations.create_message(assistant_msg)
            await svc.ops.create(op)
            if completion is not None:
                from app.domain.services.ai import record_usage
                await record_usage(svc.usage, user_id, op_id,
                                   ent.AiOptions(**state["ai"]), completion)

        await svc.db.do(work)
        return {"operation": op}

    g = StateGraph(ChatState)
    g.add_node("resolve_context", resolve_context)
    g.add_node("generate_reply", generate_reply)
    g.add_node("persist", persist)
    g.add_edge(START, "resolve_context")
    g.add_edge("resolve_context", "generate_reply")
    g.add_edge("generate_reply", "persist")
    g.add_edge("persist", END)
    return g.compile(checkpointer=checkpointer)
