from __future__ import annotations
from datetime import datetime, timezone
from typing import Optional
from uuid import UUID, uuid4

from app.db import DB, NotFoundError
from app.domain import entities as ent
from app.domain.errors import (
    DomainError, integration_required, internal, not_configured, not_found,
    provider_error, validation_field,
)
from app.domain.validators import code_point_len
from app.infrastructure.store_ai import AiKeyStore, AiUsageStore
from app.infrastructure.store_applications import ApplicationStore
from app.infrastructure.store_operations import OperationStore


def _now() -> datetime:
    return datetime.now(timezone.utc)


def validate_ai_options(ai: Optional[ent.AiOptions]) -> None:
    if ai is None:
        return
    if not ent.valid_ai_provider(ai.provider):
        raise validation_field("ai.provider", "unsupported provider")
    if not ai.model:
        raise validation_field("ai.model", "model is required")
    if not ent.valid_credential_mode(ai.credential_mode):
        raise validation_field("ai.credentialMode", "unsupported credentialMode")
    if ai.credential_mode == "MANAGED" and ai.provider != "OPENAI":
        raise validation_field("ai.credentialMode", "MANAGED is only supported for OPENAI")
    if not ent.valid_effort(ai.effort):
        raise validation_field("ai.effort", "unsupported effort")


class AIGate:
    def __init__(self, db: DB, managed_key: str, cipher, chat):
        self.keys = AiKeyStore(db)
        self.managed_key = managed_key
        self.cipher = cipher
        self.chat = chat
        self.byok_enabled = cipher is not None

    async def check(self, user_id: UUID, ai: Optional[ent.AiOptions]) -> None:
        if ai is None:
            return
        validate_ai_options(ai)
        if ai.credential_mode == "MANAGED":
            if not self.managed_key:
                raise not_configured("managed AI is not configured")
            return
        if not self.byok_enabled or not await self.keys.has(user_id, ai.provider):
            raise integration_required("no BYOK key configured for " + ai.provider)

    async def resolve_key(self, user_id: UUID, ai: ent.AiOptions) -> str:
        if ai.credential_mode == "MANAGED":
            return self.managed_key
        try:
            k = await self.keys.get(user_id, ai.provider)
        except NotFoundError:
            raise integration_required("no BYOK key configured for " + ai.provider)
        if self.cipher is None:
            raise not_configured("BYOK encryption is not configured")
        try:
            return self.cipher.decrypt(k.ciphertext, k.nonce, user_id, ai.provider)
        except Exception:
            raise integration_required("BYOK key could not be decrypted")

    async def complete(self, user_id: UUID, ai: ent.AiOptions,
                       system: str, user: str) -> ent.AICompletion:
        await self.check(user_id, ai)
        if ai.provider != "OPENAI" or self.chat is None:
            raise not_configured("AI provider " + ai.provider + " is not supported")
        key = await self.resolve_key(user_id, ai)
        try:
            return await self.chat.chat(key, ai.model, system, user)
        except Exception:
            raise provider_error("AI provider request failed")


async def record_usage(usage: AiUsageStore, user_id: UUID, op_id: UUID,
                       ai: ent.AiOptions, c: ent.AICompletion) -> None:
    await usage.create(ent.AiUsage(
        id=uuid4(), user_id=user_id, operation_id=op_id, provider=ai.provider,
        model=ai.model, managed=ai.credential_mode == "MANAGED",
        input_tokens=c.input_tokens, output_tokens=c.output_tokens,
        cost_micro_credits=ent.ai_cost_micro_credits(
            ai.provider, ai.model, c.input_tokens, c.output_tokens),
        status=ent.USAGE_SETTLED, created_at=_now()))


class AIService:
    def __init__(self, db: DB, gate: AIGate):
        self.db = db
        self.gate = gate
        self.ops = OperationStore(db)
        self.usage = AiUsageStore(db)
        self.applications = ApplicationStore(db)

    async def generate(self, user_id: UUID, ai: Optional[ent.AiOptions], prompt: str,
                       application_id: UUID, evidence_ids: list[UUID]) -> ent.Operation:
        if ai is None:
            raise validation_field("ai", "required")
        if not prompt or code_point_len(prompt) > 20000:
            raise validation_field("prompt", "prompt must be 1-20000 characters")
        try:
            await self.applications.get(user_id, application_id)
        except NotFoundError:
            raise not_found()
        except Exception:
            raise internal()
        c = await self.gate.complete(user_id, ai, "", prompt)
        op = None

        async def work():
            nonlocal op
            now = _now()
            op_id = uuid4()
            op = ent.Operation(
                id=op_id, user_id=user_id, type=ent.OP_AI_GENERATE,
                application_id=application_id, status=ent.OP_SUCCEEDED,
                result=ent.OperationResult(kind=ent.OP_AI_GENERATE, value={
                    "text": c.text, "citations": [],
                    "usage": {"inputTokens": c.input_tokens,
                              "outputTokens": c.output_tokens}}),
                created_at=now, updated_at=now)
            await self.ops.create(op)
            await record_usage(self.usage, user_id, op_id, ai, c)

        try:
            await self.db.do(work)
        except DomainError:
            raise
        except Exception:
            raise internal()
        return op

    async def list_usage(self, user_id: UUID, from_, to, page):
        try:
            return await self.usage.list(user_id, from_, to, page)
        except Exception:
            raise internal()
