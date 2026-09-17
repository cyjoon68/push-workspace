from __future__ import annotations

from datetime import datetime, timezone
from uuid import uuid4

import pytest

from app.domain import entities as ent
from app.domain.errors import DomainError
from app.domain.services.ai import AIGate, AIService
from tests.stubs import (FakeDB, StubAiKeyStore, StubAiUsageStore,
                         StubApplicationStore, StubChatCompleter, StubCipher,
                         StubConversationStore, StubOperationStore)
from tests.test_conversation import _conv, _svc as conv_svc


def _now():
    return datetime.now(timezone.utc)


def _managed():
    return ent.AiOptions(provider="OPENAI", model="gpt-4o-mini",
                         credential_mode="MANAGED", effort="LOW")


def _gate(managed_key="", cipher=None, chat=None, keys=None):
    g = AIGate(FakeDB(), managed_key, cipher, chat)
    if keys is not None:
        g.keys = keys
    return g


async def test_gate_complete_managed():
    chat = StubChatCompleter(text="hello", in_tokens=3, out_tokens=5)
    g = _gate(managed_key="sk-managed", chat=chat)
    c = await g.complete(uuid4(), _managed(), "sys", "hi")
    assert c.text == "hello" and c.input_tokens == 3 and c.output_tokens == 5
    assert chat.got == {"key": "sk-managed", "model": "gpt-4o-mini",
                       "system": "sys", "user": "hi"}


async def test_gate_complete_managed_not_configured():
    g = _gate(chat=StubChatCompleter())
    with pytest.raises(DomainError) as e:
        await g.complete(uuid4(), _managed(), "", "hi")
    assert e.value.code == "NOT_CONFIGURED"


async def test_gate_complete_byok_decrypts():
    user_id = uuid4()
    chat = StubChatCompleter(text="ok")
    keys = StubAiKeyStore(ent.AiKey(user_id=user_id, provider="OPENAI",
                                  last_four="user", ciphertext=b"1",
                                  nonce=b"2", updated_at=_now()))
    g = _gate(cipher=StubCipher(plaintext="sk-user"), chat=chat, keys=keys)
    ai = ent.AiOptions(provider="OPENAI", model="gpt-4o",
                       credential_mode="BYOK", effort="HIGH")
    c = await g.complete(user_id, ai, "", "yo")
    assert c.text == "ok" and chat.got["key"] == "sk-user"


async def test_gate_complete_byok_missing_key():
    g = _gate(cipher=StubCipher(), chat=StubChatCompleter(),
              keys=StubAiKeyStore())
    ai = ent.AiOptions(provider="OPENAI", model="gpt-4o",
                       credential_mode="BYOK", effort="LOW")
    with pytest.raises(DomainError) as e:
        await g.complete(uuid4(), ai, "", "hi")
    assert e.value.code == "INTEGRATION_REQUIRED"


async def test_gate_complete_unsupported_provider():
    user_id = uuid4()
    keys = StubAiKeyStore(ent.AiKey(user_id=user_id, provider="CLAUDE",
                                    last_four="k", ciphertext=b"1", nonce=b"2",
                                    updated_at=_now()))
    g = _gate(cipher=StubCipher(plaintext="k"), chat=StubChatCompleter(),
              keys=keys)
    ai = ent.AiOptions(provider="CLAUDE", model="claude-sonnet-4",
                       credential_mode="BYOK", effort="LOW")
    with pytest.raises(DomainError) as e:
        await g.complete(user_id, ai, "", "hi")
    assert e.value.code == "NOT_CONFIGURED"


async def test_gate_complete_provider_failure():
    chat = StubChatCompleter(err=RuntimeError("boom"))
    g = _gate(managed_key="sk", chat=chat)
    with pytest.raises(DomainError) as e:
        await g.complete(uuid4(), _managed(), "", "hi")
    assert e.value.code == "PROVIDER_ERROR"


def _ai_svc(gate, ops, usage, apps):
    svc = AIService(FakeDB(), gate)
    svc.ops = ops
    svc.usage = usage
    svc.applications = apps
    return svc


async def test_generate_creates_operation_and_usage():
    user_id, app_id = uuid4(), uuid4()
    ops, usage = StubOperationStore(), StubAiUsageStore()
    chat = StubChatCompleter(text="generated", in_tokens=10, out_tokens=20)
    gate = _gate(managed_key="sk", chat=chat)
    app = ent.Application(id=app_id, user_id=user_id, job_id=uuid4(),
                          company="c", title="t", created_at=_now(),
                          updated_at=_now())
    svc = _ai_svc(gate, ops, usage, StubApplicationStore(app=app))
    op = await svc.generate(user_id, _managed(), "write", app_id, [])
    assert op.type == ent.OP_AI_GENERATE and op.status == ent.OP_SUCCEEDED
    assert len(usage.items) == 1
    u = usage.items[0]
    assert (u.input_tokens == 10 and u.output_tokens == 20 and u.managed
            and u.provider == "OPENAI" and u.status == ent.USAGE_SETTLED)
    assert u.operation_id == op.id


async def test_generate_bad_application():
    svc = _ai_svc(AIGate(FakeDB(), "", None, None), StubOperationStore(),
                  StubAiUsageStore(), StubApplicationStore())
    with pytest.raises(DomainError) as e:
        await svc.generate(uuid4(), _managed(), "x", uuid4(), [])
    assert e.value.code == "NOT_FOUND"


async def test_list_usage():
    user_id = uuid4()
    usage = StubAiUsageStore([
        ent.AiUsage(id=uuid4(), user_id=user_id, provider="OPENAI",
                    model="m", managed=True, input_tokens=1, output_tokens=1,
                    cost_micro_credits=1, status="SETTLED",
                    created_at=_now()),
        ent.AiUsage(id=uuid4(), user_id=uuid4(), provider="OPENAI",
                    model="m", managed=True, input_tokens=1, output_tokens=1,
                    cost_micro_credits=1, status="SETTLED",
                    created_at=_now()),
    ])
    svc = _ai_svc(AIGate(FakeDB(), "", None, None), StubOperationStore(),
                  usage, StubApplicationStore())
    p = await svc.list_usage(user_id, None, None, None)
    assert len(p.items) == 1


async def test_post_message_ai_completion():
    user_id = uuid4()
    conv = _conv(user_id=user_id)
    convs = StubConversationStore(conv=conv)
    usage = StubAiUsageStore()
    chat = StubChatCompleter(text="real answer", in_tokens=7, out_tokens=9)
    gate = _gate(managed_key="sk", chat=chat)
    svc = conv_svc(convs, gate=gate, usage=usage)
    op = await svc.post_message(user_id, conv.id, "question", {},
                                _managed(), "SUGGEST")
    assert len(convs.messages) == 2
    assert convs.messages[1].text == "real answer"
    assert len(usage.items) == 1 and usage.items[0].operation_id == op.id


async def test_post_message_no_ai_keeps_stub():
    user_id = uuid4()
    conv = _conv(user_id=user_id)
    convs = StubConversationStore(conv=conv)
    chat = StubChatCompleter(text="should not be used")
    svc = conv_svc(convs, gate=_gate(managed_key="sk", chat=chat))
    await svc.post_message(user_id, conv.id, "hi", {}, None, "SUGGEST")
    assert convs.messages[1].text != "should not be used"
