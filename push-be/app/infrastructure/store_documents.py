from __future__ import annotations
from datetime import datetime
from typing import Optional
from uuid import UUID

from app.domain.entities import Document, DocumentExport, DocumentVersion, RevisionProposal
from app.domain.pagination import Cursor, Page
from app.jsonutil import dump
from app.infrastructure.store_common import Conds, Store, list_page, to_model

_DOC_COLS = ("id, user_id, revision, application_id, title, kind, template, language, status, "
             "latest_version_id, finalized_version_id, created_at, updated_at")
_VERSION_COLS = ("id, user_id, document_id, application_id, number, content, blocks, "
                 "change_note, quality, created_at")
_PROPOSAL_COLS = ("id, user_id, document_id, source_version_id, source_revision, selection, "
                  "replacement, evidence_refs, claim_status, applied_at, created_at")
_EXPORT_COLS = ("id, user_id, document_id, version_id, format, template, language, content, "
                "blocks, content_hash, status, renderer_version, result, created_at, updated_at")


def _version(row) -> DocumentVersion:
    v = dict(row)
    v["blocks"] = v.get("blocks") or []
    v["quality"] = v.get("quality") or {}
    return DocumentVersion(**v)


def _export(row) -> DocumentExport:
    e = dict(row)
    e["blocks"] = e.get("blocks") or []
    return DocumentExport(**e)


class DocumentStore(Store):
    async def create(self, d: Document) -> None:
        await self.q().execute(
            f"INSERT INTO documents ({_DOC_COLS}) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)",
            d.id, d.user_id, d.revision, d.application_id, d.title, d.kind, d.template,
            d.language, d.status, d.latest_version_id, d.finalized_version_id,
            d.created_at, d.updated_at)

    async def get(self, user_id: UUID, id_: UUID) -> Document:
        return to_model(Document, await self.one(
            f"SELECT {_DOC_COLS} FROM documents WHERE id = $1 AND user_id = $2", id_, user_id))

    async def list(self, user_id: UUID, application_id: Optional[UUID], kind: Optional[str], page) -> Page:
        c = Conds()
        c.add("user_id = {}", user_id)
        if application_id:
            c.add("application_id = {}", application_id)
        if kind:
            c.add("kind = {}", kind)
        c.cursor(page)
        sql, args = c.query(_DOC_COLS, "documents", page.effective_limit())
        rows = await self.q().fetch(sql, *args)
        return list_page(rows, page, Document, lambda d: Cursor(d.created_at, d.id))

    async def update(self, d: Document, expected: int) -> None:
        await self.guard_update(
            "documents", d.id, d.user_id,
            "UPDATE documents SET revision = revision + 1, title = $3, template = $4, "
            "language = $5, status = $6, latest_version_id = $7, finalized_version_id = $8, "
            "updated_at = $9 WHERE id = $1 AND user_id = $2 AND revision = $10",
            d.id, d.user_id, d.title, d.template, d.language, d.status,
            d.latest_version_id, d.finalized_version_id, d.updated_at, expected)

    async def create_version(self, v: DocumentVersion) -> None:
        await self.q().execute(
            f"INSERT INTO document_versions ({_VERSION_COLS}) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)",
            v.id, v.user_id, v.document_id, v.application_id, v.number, dump(v.content),
            dump(v.blocks), v.change_note, dump(v.quality), v.created_at)

    async def get_version(self, user_id: UUID, document_id: UUID, version_id: UUID) -> DocumentVersion:
        sql = (f"SELECT {_VERSION_COLS} FROM document_versions "
               "WHERE id = $1 AND user_id = $2")
        args: list = [version_id, user_id]
        if document_id != UUID(int=0):
            sql += " AND document_id = $3"
            args.append(document_id)
        return _version(await self.one(sql, *args))

    async def get_version_by_id(self, user_id: UUID, version_id: UUID) -> DocumentVersion:
        return _version(await self.one(
            f"SELECT {_VERSION_COLS} FROM document_versions WHERE id = $1 AND user_id = $2",
            version_id, user_id))

    async def list_versions(self, user_id: UUID, document_id: UUID, page) -> Page:
        c = Conds()
        c.add("user_id = {}", user_id)
        c.add("document_id = {}", document_id)
        c.cursor(page)
        sql, args = c.query(_VERSION_COLS, "document_versions", page.effective_limit())
        rows = await self.q().fetch(sql, *args)
        from app.domain.pagination import new_page
        return new_page([_version(r) for r in rows], page.effective_limit(),
                        lambda v: Cursor(v.created_at, v.id))

    async def next_version_number(self, document_id: UUID) -> int:
        n = await self.q().fetchval(
            "SELECT COALESCE(MAX(number), 0) FROM document_versions WHERE document_id = $1",
            document_id)
        return n + 1

    async def create_proposal(self, p: RevisionProposal) -> None:
        await self.q().execute(
            f"INSERT INTO revision_proposals ({_PROPOSAL_COLS}) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)",
            p.id, p.user_id, p.document_id, p.source_version_id, p.source_revision,
            dump(p.selection), p.replacement, dump(p.evidence_refs), p.claim_status,
            p.applied_at, p.created_at)

    async def get_proposal(self, user_id: UUID, document_id: UUID, proposal_id: UUID) -> RevisionProposal:
        return to_model(RevisionProposal, await self.one(
            f"SELECT {_PROPOSAL_COLS} FROM revision_proposals WHERE id = $1 AND user_id = $2 "
            "AND document_id = $3", proposal_id, user_id, document_id))

    async def mark_proposal_applied(self, id_: UUID, at: datetime) -> None:
        await self.q().execute(
            "UPDATE revision_proposals SET applied_at = $2 WHERE id = $1", id_, at)

    async def create_export(self, e: DocumentExport) -> None:
        await self.q().execute(
            f"INSERT INTO document_exports ({_EXPORT_COLS}) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)",
            e.id, e.user_id, e.document_id, e.version_id, e.format, e.template, e.language,
            dump(e.content), dump(e.blocks), e.content_hash, e.status, e.renderer_version,
            dump(e.result) if e.result else None, e.created_at, e.updated_at)

    async def get_export(self, user_id: UUID, document_id: UUID, export_id: UUID) -> DocumentExport:
        return _export(await self.one(
            f"SELECT {_EXPORT_COLS} FROM document_exports WHERE id = $1 AND user_id = $2 "
            "AND document_id = $3", export_id, user_id, document_id))

    async def list_exports(self, user_id: UUID, document_id: UUID, page) -> Page:
        c = Conds()
        c.add("user_id = {}", user_id)
        c.add("document_id = {}", document_id)
        c.cursor(page)
        sql, args = c.query(_EXPORT_COLS, "document_exports", page.effective_limit())
        rows = await self.q().fetch(sql, *args)
        from app.domain.pagination import new_page
        return new_page([_export(r) for r in rows], page.effective_limit(),
                        lambda e: Cursor(e.created_at, e.id))

    async def update_export_result(self, e: DocumentExport) -> None:
        await self.q().execute(
            "UPDATE document_exports SET status = $2, result = $3, updated_at = $4 WHERE id = $1",
            e.id, e.status, dump(e.result) if e.result else None, e.updated_at)

    async def count_finalized_by_application(self, user_id: UUID, application_id: UUID) -> int:
        return await self.q().fetchval(
            "SELECT count(*) FROM documents WHERE user_id = $1 AND application_id = $2 "
            "AND status = 'FINALIZED'", user_id, application_id)
