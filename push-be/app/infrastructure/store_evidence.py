from __future__ import annotations
from typing import Optional
from uuid import UUID

from app.db import NotFoundError
from app.domain.entities import CareerEvidence, Source
from app.domain.pagination import Cursor, Page
from app.jsonutil import dump
from app.infrastructure.store_common import Conds, Store, list_page, to_model

_EVIDENCE_COLS = ("id, user_id, revision, kind, title, source_text, source_url, skills, "
                  "verification_status, provenance, supersedes_id, archived, created_at, updated_at")
_SOURCE_COLS = ("id, user_id, file_name, mime_type, size, sha256, path, status, created_at")


class EvidenceStore(Store):
    async def create(self, e: CareerEvidence) -> None:
        await self.q().execute(
            f"INSERT INTO career_evidence ({_EVIDENCE_COLS}) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)",
            e.id, e.user_id, e.revision, e.kind, e.title, e.source_text, e.source_url,
            dump(e.skills), e.verification_status, dump(e.provenance), e.supersedes_id,
            e.archived, e.created_at, e.updated_at)

    async def get(self, user_id: UUID, id_: UUID) -> CareerEvidence:
        return to_model(CareerEvidence, await self.one(
            f"SELECT {_EVIDENCE_COLS} FROM career_evidence WHERE id = $1 AND user_id = $2",
            id_, user_id))

    async def list(self, user_id: UUID, kind: Optional[str], query: str, page) -> Page:
        c = Conds()
        c.add("user_id = {}", user_id)
        if kind:
            c.add("kind = {}", kind)
        if query:
            c.add("(title ILIKE {} OR source_text ILIKE {})", "%" + query + "%", "%" + query + "%")
        c.cursor(page)
        sql, args = c.query(_EVIDENCE_COLS, "career_evidence", page.effective_limit())
        rows = await self.q().fetch(sql, *args)
        return list_page(rows, page, CareerEvidence, lambda e: Cursor(e.created_at, e.id))

    async def update(self, e: CareerEvidence, expected: int) -> None:
        await self.guard_update(
            "career_evidence", e.id, e.user_id,
            "UPDATE career_evidence SET revision = revision + 1, kind = $3, title = $4, "
            "source_text = $5, source_url = $6, skills = $7, verification_status = $8, "
            "provenance = $9, supersedes_id = $10, archived = $11, updated_at = $12 "
            "WHERE id = $1 AND user_id = $2 AND revision = $13",
            e.id, e.user_id, e.kind, e.title, e.source_text, e.source_url, dump(e.skills),
            e.verification_status, dump(e.provenance), e.supersedes_id, e.archived,
            e.updated_at, expected)

    async def count(self, user_id: UUID) -> int:
        return await self.q().fetchval(
            "SELECT count(*) FROM career_evidence WHERE user_id = $1 AND NOT archived", user_id)

    async def list_by_ids(self, user_id: UUID, ids: list[UUID]) -> list[CareerEvidence]:
        rows = await self.q().fetch(
            f"SELECT {_EVIDENCE_COLS} FROM career_evidence WHERE user_id = $1 AND id = ANY($2)",
            user_id, ids)
        return [to_model(CareerEvidence, r) for r in rows]


class SourceStore(Store):
    async def create(self, s: Source) -> None:
        await self.q().execute(
            f"INSERT INTO sources ({_SOURCE_COLS}) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)",
            s.id, s.user_id, s.file_name, s.mime_type, s.size, s.sha256, s.path,
            s.status, s.created_at)

    async def get(self, user_id: UUID, id_: UUID) -> Source:
        return to_model(Source, await self.one(
            f"SELECT {_SOURCE_COLS} FROM sources WHERE id = $1 AND user_id = $2", id_, user_id))

    async def delete(self, user_id: UUID, id_: UUID) -> None:
        tag = await self.q().execute(
            "DELETE FROM sources WHERE id = $1 AND user_id = $2", id_, user_id)
        if tag.split()[-1] == "0":
            raise NotFoundError()

    async def referenced(self, user_id: UUID, id_: UUID) -> bool:
        n = await self.q().fetchval(
            "SELECT count(*) FROM career_evidence WHERE user_id = $1 AND provenance->>'sourceId' = $2",
            user_id, str(id_))
        return n > 0
