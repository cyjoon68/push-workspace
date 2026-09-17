from __future__ import annotations
from datetime import datetime, timezone
from typing import Optional
from uuid import UUID, uuid4

from app.db import DB, NotFoundError
from app.domain import entities as ent
from app.domain.errors import (
    internal, map_revision_err, not_found, validation_field, integration_required,
)
from app.domain.validators import code_point_len, hash_bytes, validate_https_url
from app.infrastructure.store_evidence import EvidenceStore
from app.infrastructure.store_operations import OperationStore
from app.jsonutil import to_jsonable


def _now() -> datetime:
    return datetime.now(timezone.utc)


IMPORT_FORMATS = {"TEXT", "MARKDOWN", "PDF", "DOCX", "GITHUB"}


def validate_evidence_input(kind: str, title: str, source_text: str,
                            source_url: Optional[str], skills: list[str],
                            supersedes_id: Optional[UUID]) -> None:
    if not ent.valid_evidence_kind(kind):
        raise validation_field("kind", "unsupported kind")
    if not 1 <= len(title) <= 200:
        raise validation_field("title", "title must be 1-200 characters")
    if not 1 <= len(source_text) <= 100000:
        raise validation_field("sourceText", "sourceText must be 1-100000 characters")
    if len(skills) > 100:
        raise validation_field("skills", "at most 100 skills")
    if any(not 1 <= len(s) <= 100 for s in skills):
        raise validation_field("skills", "each skill must be 1-100 characters")
    if source_url is not None:
        try:
            validate_https_url(source_url)
        except ValueError:
            raise validation_field("sourceUrl", "must be an http(s) URL without credentials")


class EvidenceService:
    def __init__(self, db: DB):
        self.db = db
        self.evidence = EvidenceStore(db)
        self.ops = OperationStore(db)

    async def list(self, user_id: UUID, kind: Optional[str], query: str, page) -> ent.Page:
        try:
            return await self.evidence.list(user_id, kind, query, page)
        except Exception:
            raise internal()

    async def get(self, user_id: UUID, id_: UUID) -> ent.CareerEvidence:
        try:
            return await self.evidence.get(user_id, id_)
        except NotFoundError:
            raise not_found()
        except Exception:
            raise internal()

    async def create(self, user_id: UUID, kind: str, title: str, source_text: str,
                     source_url: Optional[str], skills: list[str],
                     supersedes_id: Optional[UUID]) -> ent.CareerEvidence:
        validate_evidence_input(kind, title, source_text, source_url, skills or [], supersedes_id)
        now = _now()
        e = ent.CareerEvidence(
            id=uuid4(), user_id=user_id, revision=1, kind=kind, title=title,
            source_text=source_text, source_url=source_url, skills=skills or [],
            verification_status=ent.VERIFICATION_USER_PROVIDED,
            provenance=ent.Provenance(content_hash=hash_bytes(source_text.encode())),
            supersedes_id=supersedes_id, created_at=now, updated_at=now)
        if supersedes_id is not None:
            try:
                await self.evidence.get(user_id, supersedes_id)
            except NotFoundError:
                raise validation_field("supersedesId", "referenced evidence not found")
        try:
            await self.evidence.create(e)
        except Exception:
            raise internal()
        return e

    async def archive(self, user_id: UUID, id_: UUID, expected: int) -> ent.CareerEvidence:
        out = None

        async def work():
            nonlocal out
            try:
                e = await self.evidence.get(user_id, id_)
            except NotFoundError:
                raise not_found()
            if e.archived:
                out = e
                return
            e.archived = True
            e.updated_at = _now()
            try:
                await self.evidence.update(e, expected)
            except Exception as err:
                raise map_revision_err(err)
            e.revision = expected + 1
            out = e

        await self.db.do(work)
        return out

    async def import_(self, user_id: UUID, source_id: Optional[UUID], text: str,
                      source_url: Optional[str], content_hash: str,
                      format_: str) -> ent.Operation:
        if format_ not in IMPORT_FORMATS:
            raise validation_field("format", "unsupported format")
        if format_ == "GITHUB":
            raise integration_required("GitHub collection requires a connected integration")
        if source_url is not None:
            try:
                validate_https_url(source_url)
            except ValueError:
                raise validation_field("sourceUrl", "must be an http(s) URL without credentials")
        if source_id is not None:
            raise validation_field(
                "sourceId", "stored sources are not supported by this build; pass extracted text")
        now = _now()
        op = ent.Operation(
            id=uuid4(), user_id=user_id, type=ent.OP_EVIDENCE_IMPORT, status="",
            pending_payload={"text": text, "sourceUrl": source_url,
                             "contentHash": content_hash, "format": format_},
            created_at=now, updated_at=now)
        await self._progress_import(op)

        async def work():
            await self._persist_import_result(op)
            await self.ops.create(op)

        try:
            await self.db.do(work)
        except Exception:
            raise internal()
        return op

    async def _persist_import_result(self, op: ent.Operation) -> None:
        if op.status != ent.OP_SUCCEEDED or op.result is None:
            return
        ev_list = (op.result.value or {}).get("evidence") or []
        if not ev_list:
            return
        created = ent.CareerEvidence(**ev_list[0])
        await self.evidence.create(created)
        op.result = ent.OperationResult(
            kind=ent.OP_EVIDENCE_IMPORT,
            value={"evidence": [to_jsonable(created)], "warnings": []})

    async def _progress_import(self, op: ent.Operation) -> None:
        p = op.pending_payload or {}
        missing = []
        if not p.get("text"):
            missing.append(ent.InputField(name="text", label="Extracted source text", type="TEXT"))
        if not ent.valid_evidence_kind(p.get("kind") or ""):
            missing.append(ent.InputField(
                name="kind", label="Evidence kind (RESUME|CAREER|EDUCATION|SKILL|PROJECT)",
                type="TEXT"))
        if not p.get("title"):
            missing.append(ent.InputField(name="title", label="Evidence title", type="TEXT"))
        now = _now()
        if missing:
            op.status = ent.OP_NEEDS_INPUT
            op.input_request = ent.InputRequest(
                code="IMPORT_DETAILS_REQUIRED",
                message="text, kind and title are required to create evidence",
                fields=missing)
            op.updated_at = now
            return
        evidence = ent.CareerEvidence(
            id=uuid4(), user_id=op.user_id, revision=1, kind=p["kind"], title=p["title"],
            source_text=p["text"], source_url=p.get("sourceUrl"), skills=[],
            verification_status=ent.VERIFICATION_USER_PROVIDED,
            provenance=ent.Provenance(
                content_hash=p.get("contentHash") or hash_bytes(p["text"].encode()),
                source_location=ent.SourceLocation(
                    start=0, end=code_point_len(p["text"]), unit="CODE_POINT")),
            created_at=now, updated_at=now)
        op.result = ent.OperationResult(
            kind=ent.OP_EVIDENCE_IMPORT,
            value={"evidence": [to_jsonable(evidence)], "warnings": []})
        op.status = ent.OP_SUCCEEDED
        op.updated_at = now

    async def complete_import_input(self, op: ent.Operation, fields: dict) -> None:
        p = op.pending_payload if isinstance(op.pending_payload, dict) else {}
        allowed = {f.name for f in (op.input_request.fields if op.input_request else [])}
        for k, v in fields.items():
            if k not in allowed:
                raise validation_field(f"fields.{k}", "field not requested")
            p[k] = v
        op.pending_payload = p
        await self._progress_import(op)
        await self._persist_import_result(op)
