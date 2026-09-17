from __future__ import annotations

from datetime import datetime, timezone
from uuid import uuid4

import pytest

from app.domain import entities as ent
from app.domain.errors import DomainError
from app.domain.services.conversation import ConversationService
from tests.stubs import (FakeDB, StubApplicationStore, StubConversationStore,
                         StubDocumentStore, StubEvidenceStore,
                         StubOperationStore)


def _now():
    return datetime.now(timezone.utc)


def _conv(user_id=None, application_id=None, archived=False, revision=1):
    return ent.Conversation(
        id=uuid4(), user_id=user_id or uuid4(), revision=revision,
        application_id=application_id, archived=archived,
        created_at=_now(), updated_at=_now())


def _svc(convs, apps=None, docs=None, ev=None, ops=None, gate=None,
         usage=None):
    svc = ConversationService(FakeDB(), gate, None)
    svc.conversations = convs
    svc.applications = apps or StubApplicationStore()
    svc.documents = docs or StubDocumentStore()
    svc.evidence = ev or StubEvidenceStore()
    svc.ops = ops or StubOperationStore()
    if usage is not None:
        svc.usage = usage
    return svc


async def test_create_requires_application():
    svc = _svc(StubConversationStore())
    with pytest.raises(DomainError) as e:
        await svc.create(uuid4(), uuid4(), "")
    assert e.value.code == "NOT_FOUND"


async def test_create_without_application():
    store = StubConversationStore()
    c = await _svc(store).create(uuid4(), None, "hello")
    assert c.revision == 1 and c.title == "hello" and c.application_id is None
    assert len(store.created) == 1


async def test_patch_conflict():
    user_id = uuid4()
    conv = _conv(user_id=user_id, revision=3)
    with pytest.raises(DomainError) as e:
        await _svc(StubConversationStore(conv=conv)).patch(
            user_id, conv.id, 1, title=None, pinned=True)
    assert e.value.code == "REVISION_CONFLICT"


async def test_post_message_rejects_bad_access_mode():
    user_id = uuid4()
    conv = _conv(user_id=user_id)
    with pytest.raises(DomainError) as e:
        await _svc(StubConversationStore(conv=conv)).post_message(
            user_id, conv.id, "hi", {}, None, "YOLO")
    assert e.value.code == "VALIDATION_ERROR"


async def test_post_message_archived_conversation():
    user_id = uuid4()
    conv = _conv(user_id=user_id, archived=True)
    with pytest.raises(DomainError) as e:
        await _svc(StubConversationStore(conv=conv)).post_message(
            user_id, conv.id, "hi", {}, None, "SUGGEST")
    assert e.value.code == "INVALID_TRANSITION"


async def test_post_message_evidence_ownership():
    user_id = uuid4()
    conv = _conv(user_id=user_id)
    with pytest.raises(DomainError) as e:
        await _svc(StubConversationStore(conv=conv)).post_message(
            user_id, conv.id, "hi", {"evidenceIds": [str(uuid4())]}, None,
            "SUGGEST")
    assert e.value.code == "VALIDATION_ERROR"


async def test_post_message_document_without_version():
    user_id = uuid4()
    app_id = uuid4()
    conv = _conv(user_id=user_id, application_id=app_id)
    doc = ent.Document(id=uuid4(), user_id=user_id, application_id=app_id,
                       title="resume", kind="RESUME", template="CLASSIC",
                       created_at=_now(), updated_at=_now())
    svc = _svc(StubConversationStore(conv=conv),
               docs=StubDocumentStore(doc=doc))
    with pytest.raises(DomainError) as e:
        await svc.post_message(user_id, conv.id, "hi",
                               {"documentId": str(doc.id)}, None, "SUGGEST")
    assert e.value.code == "VALIDATION_ERROR"


async def test_post_message_document_scope_mismatch():
    user_id = uuid4()
    app_id, other = uuid4(), uuid4()
    conv = _conv(user_id=user_id, application_id=app_id)
    version_id = uuid4()
    doc = ent.Document(id=uuid4(), user_id=user_id, application_id=other,
                       title="resume", kind="RESUME", template="CLASSIC",
                       latest_version_id=version_id,
                       created_at=_now(), updated_at=_now())
    svc = _svc(StubConversationStore(conv=conv),
               docs=StubDocumentStore(doc=doc))
    with pytest.raises(DomainError) as e:
        await svc.post_message(user_id, conv.id, "hi",
                               {"documentId": str(doc.id)}, None, "SUGGEST")
    assert e.value.code == "VALIDATION_ERROR"


async def test_post_message_success_creates_operation():
    user_id = uuid4()
    app_id = uuid4()
    conv = _conv(user_id=user_id, application_id=app_id)
    version_id = uuid4()
    doc = ent.Document(id=uuid4(), user_id=user_id, application_id=app_id,
                       title="resume", kind="RESUME", template="CLASSIC",
                       latest_version_id=version_id,
                       created_at=_now(), updated_at=_now())
    version = ent.DocumentVersion(
        id=version_id, user_id=user_id, document_id=doc.id,
        application_id=app_id, number=1, content={}, created_at=_now())
    ev_id = uuid4()
    ev = StubEvidenceStore({ev_id: ent.CareerEvidence(
        id=ev_id, user_id=user_id, revision=1, kind="CAREER", title="ev",
        source_text="x", created_at=_now(), updated_at=_now())})
    convs = StubConversationStore(conv=conv)
    ops = StubOperationStore()
    svc = _svc(convs, docs=StubDocumentStore(doc=doc, version=version),
               ev=ev, ops=ops)
    op = await svc.post_message(
        user_id, conv.id, "도와줘",
        {"documentId": str(doc.id), "evidenceIds": [str(ev_id)]},
        None, "CONFIRM_ACTIONS")
    assert op.status == ent.OP_SUCCEEDED and op.type == ent.OP_CHAT_MESSAGE
    assert len(convs.messages) == 2
    user_msg, ai_msg = convs.messages
    assert user_msg.role == "USER" and ai_msg.role == "ASSISTANT"
    assert user_msg.operation_id == op.id
    assert len(user_msg.attachments) == 2
    assert (user_msg.attachments[0].type == "DOCUMENT_VERSION"
            and user_msg.attachments[0].id == version_id)
    assert len(ops.created) == 1


async def test_archive():
    user_id = uuid4()
    conv = _conv(user_id=user_id)
    out = await _svc(StubConversationStore(conv=conv)).archive(
        user_id, conv.id, 1)
    assert out.archived and out.revision == 2
