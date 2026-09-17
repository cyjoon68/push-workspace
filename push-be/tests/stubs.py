from __future__ import annotations

from typing import Optional
from uuid import UUID

from app.db import NotFoundError
from app.domain import entities as ent


class FakeDB:
    async def do(self, fn):
        return await fn()

    def q(self):
        raise NotImplementedError


class StubApplicationStore:
    def __init__(self, app: Optional[ent.Application] = None,
                 update_err: Optional[Exception] = None):
        self.app = app
        self.update_err = update_err
        self.events = []
        self.update_calls = 0

    async def create(self, a):
        return None

    async def get(self, user_id: UUID, id_: UUID):
        if self.app is None or self.app.id != id_ or self.app.user_id != user_id:
            raise NotFoundError()
        return self.app

    async def update(self, a, expected: int):
        self.update_calls += 1
        if self.update_err is not None:
            raise self.update_err

    async def add_event(self, e):
        self.events.append(e)

    async def list_events(self, user_id, application_id, page):
        return self.events


class StubJobStore:
    def __init__(self, job=None):
        self.job = job

    async def get(self, user_id: UUID, id_: UUID):
        if self.job is None or self.job.id != id_ or self.job.user_id != user_id:
            raise NotFoundError()
        return self.job


class StubConversationStore:
    def __init__(self, conv: Optional[ent.Conversation] = None,
                 update_err: Optional[Exception] = None):
        self.conv = conv
        self.messages = []
        self.created = []
        self.update_err = update_err

    async def create(self, c):
        self.created.append(c)

    async def get(self, user_id: UUID, id_: UUID):
        if self.conv is None or self.conv.id != id_ or self.conv.user_id != user_id:
            raise NotFoundError()
        return self.conv

    async def update(self, c, expected: int):
        if self.update_err is not None:
            raise self.update_err

    async def create_message(self, m):
        self.messages.append(m)

    async def list_messages(self, user_id, conversation_id, page):
        return self.messages


class StubDocumentStore:
    def __init__(self, doc=None, version=None):
        self.doc = doc
        self.version = version

    async def get(self, user_id: UUID, id_: UUID):
        if self.doc is None or self.doc.id != id_ or self.doc.user_id != user_id:
            raise NotFoundError()
        return self.doc

    async def get_version(self, user_id: UUID, document_id: UUID, version_id: UUID):
        if (self.version is None or self.version.id != version_id
                or (document_id != UUID(int=0)
                    and self.version.document_id != document_id)):
            raise NotFoundError()
        return self.version

    async def get_version_by_id(self, user_id: UUID, version_id: UUID):
        if (self.version is None or self.version.id != version_id
                or self.version.user_id != user_id):
            raise NotFoundError()
        return self.version


class StubEvidenceStore:
    def __init__(self, items: Optional[dict] = None):
        self.items = items or {}

    async def get(self, user_id: UUID, id_: UUID):
        e = self.items.get(id_)
        if e is None or e.user_id != user_id:
            raise NotFoundError()
        return e


class StubOperationStore:
    def __init__(self):
        self.created = []

    async def create(self, o):
        self.created.append(o)


class StubChatCompleter:
    def __init__(self, text: str = "", in_tokens: int = 0,
                 out_tokens: int = 0, err: Optional[Exception] = None):
        self.text = text
        self.in_tokens = in_tokens
        self.out_tokens = out_tokens
        self.err = err
        self.got = {}

    async def chat(self, api_key: str, model: str, system: str, user: str):
        if self.err is not None:
            raise self.err
        self.got = {"key": api_key, "model": model, "system": system,
                    "user": user}
        return ent.AICompletion(text=self.text, input_tokens=self.in_tokens,
                                output_tokens=self.out_tokens)


class StubCipher:
    def __init__(self, plaintext: str = "", err: Optional[Exception] = None):
        self.plaintext = plaintext
        self.err = err

    def decrypt(self, ciphertext: bytes, nonce: bytes, user_id: UUID,
                purpose: str) -> str:
        if self.err is not None:
            raise self.err
        return self.plaintext

    def encrypt(self, plaintext: str, user_id: UUID, purpose: str):
        return b"ct:" + plaintext.encode(), b"n"


class StubAiKeyStore:
    def __init__(self, key: Optional[ent.AiKey] = None):
        self.key = key

    async def get(self, user_id: UUID, provider: str):
        if (self.key is None or self.key.provider != provider
                or self.key.user_id != user_id):
            raise NotFoundError()
        return self.key

    async def has(self, user_id: UUID, provider: str) -> bool:
        return (self.key is not None and self.key.provider == provider
                and self.key.user_id == user_id)


class StubAiUsageStore:
    def __init__(self, items: Optional[list] = None):
        self.items = items or []

    async def create(self, u):
        self.items.append(u)

    async def list(self, user_id: UUID, from_, to, page):
        from app.domain.pagination import new_page
        items = [u for u in self.items if u.user_id == user_id]
        return new_page(items, len(items) + 1, lambda u: None)
