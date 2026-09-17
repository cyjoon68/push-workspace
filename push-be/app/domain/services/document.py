from __future__ import annotations
import copy
import json
from dataclasses import dataclass
from datetime import datetime, timezone
from typing import Optional
from uuid import UUID, uuid4

from app.db import DB, NotFoundError
from app.domain import content as dc
from app.domain import entities as ent
from app.domain.errors import (
    approval_stale, document_not_finalized, internal,
    invalid_transition, map_revision_err, not_found, revision_conflict,
    unsupported_claim, validation_field,
)
from app.domain.services.ai import AIGate
from app.domain.services.approval import ApprovalService, hash_json
from app.domain.content import code_point_slice
from app.domain.validators import code_point_len, hash_bytes
from app.infrastructure.store_applications import ApplicationStore
from app.infrastructure.store_documents import DocumentStore
from app.infrastructure.store_evidence import EvidenceStore
from app.infrastructure.store_operations import OperationStore
from app.jsonutil import to_jsonable


def _now() -> datetime:
    return datetime.now(timezone.utc)


EXPORT_PDF = "PDF"
EXPORT_DOCX = "DOCX"
EXPORT_READY_TO_RENDER = "READY_TO_RENDER"
EXPORT_SUCCEEDED = "SUCCEEDED"
EXPORT_FAILED = "FAILED"


def validate_document_input(title: str, kind: str, template: str) -> None:
    if not 1 <= len(title) <= 300:
        raise validation_field("title", "title must be 1-300 characters")
    if not ent.valid_document_kind(kind):
        raise validation_field("kind", "must be RESUME, PORTFOLIO or COVER_LETTER")
    if not ent.valid_document_template(template):
        raise validation_field("template", "must be CLASSIC, MODERN or COMPACT")


def compute_quality(blocks: list[ent.Block], method: str) -> ent.Quality:
    issues = []
    unsupported = 0
    for b in blocks:
        if b.claim_status == ent.CLAIM_UNSUPPORTED:
            unsupported += 1
            issues.append(ent.QualityIssue(
                code="UNSUPPORTED_CLAIM", severity="ERROR", block_id=b.id,
                message="block has no evidence references"))
        elif b.claim_status == ent.CLAIM_NEEDS_REVIEW:
            issues.append(ent.QualityIssue(
                code="NEEDS_REVIEW", severity="WARN", block_id=b.id,
                message="evidence span does not literally appear in block text"))
    fidelity = (len(blocks) - unsupported) * 100 // len(blocks) if blocks else None
    return ent.Quality(evidence_fidelity=fidelity, method=method, issues=issues)


@dataclass
class ExcerptEntry:
    evidence_id: UUID
    text: str
    start: int
    end: int


class DocumentService:
    def __init__(self, db: DB, approvals: ApprovalService, ai: AIGate):
        self.db = db
        self.documents = DocumentStore(db)
        self.applications = ApplicationStore(db)
        self.evidence = EvidenceStore(db)
        self.approvals = approvals
        self.ops = OperationStore(db)
        self.ai = ai

    async def list(self, user_id: UUID, application_id: Optional[UUID],
                   kind: Optional[str], page):
        try:
            return await self.documents.list(user_id, application_id, kind, page)
        except Exception:
            raise internal()

    async def get(self, user_id: UUID, id_: UUID) -> ent.Document:
        try:
            return await self.documents.get(user_id, id_)
        except NotFoundError:
            raise not_found()
        except Exception:
            raise internal()

    async def create(self, user_id: UUID, application_id: UUID, title: str,
                     kind: str, template: str, language: str) -> ent.Document:
        try:
            await self.applications.get(user_id, application_id)
        except NotFoundError:
            raise not_found()
        except Exception:
            raise internal()
        validate_document_input(title, kind, template)
        now = _now()
        d = ent.Document(
            id=uuid4(), user_id=user_id, revision=1, application_id=application_id,
            title=title, kind=kind, template=template, language=language,
            status=ent.DOC_DRAFT, created_at=now, updated_at=now)
        try:
            await self.documents.create(d)
        except Exception:
            raise internal()
        return d

    async def _get_mutable(self, user_id: UUID, id_: UUID) -> ent.Document:
        try:
            d = await self.documents.get(user_id, id_)
        except NotFoundError:
            raise not_found()
        except Exception:
            raise internal()
        if d.status == ent.DOC_ARCHIVED:
            raise invalid_transition("document is archived")
        return d

    async def patch(self, user_id: UUID, id_: UUID, expected: int,
                    title: Optional[str], template: Optional[str],
                    language: Optional[str]) -> ent.Document:
        out = None

        async def work():
            nonlocal out
            d = await self._get_mutable(user_id, id_)
            if d.revision != expected:
                raise revision_conflict(d.revision)
            if title is not None:
                if not 1 <= len(title) <= 300:
                    raise validation_field("title", "title must be 1-300 characters")
                d.title = title
            if template is not None:
                if not ent.valid_document_template(template):
                    raise validation_field(
                        "template", "must be CLASSIC, MODERN or COMPACT")
                if template != d.template:
                    d.template = template
                    d.finalized_version_id = None
                    if d.status == ent.DOC_FINALIZED:
                        d.status = ent.DOC_DRAFT
            if language is not None:
                d.language = language
            d.updated_at = _now()
            try:
                await self.documents.update(d, expected)
            except Exception as err:
                raise map_revision_err(err)
            d.revision = expected + 1
            out = d

        await self.db.do(work)
        return out

    async def _verify_blocks(self, user_id: UUID, application_id: UUID,
                             blocks: list[ent.Block]) -> dict:
        loaded: dict = {}
        for b in blocks:
            if not b.evidence_refs:
                b.claim_status = ent.CLAIM_UNSUPPORTED
                continue
            status = ent.CLAIM_SUPPORTED
            for ref in b.evidence_refs:
                e = loaded.get(ref.evidence_id)
                if e is None:
                    try:
                        e = await self.evidence.get(user_id, ref.evidence_id)
                    except Exception:
                        raise validation_field(
                            "blocks", "evidence " + str(ref.evidence_id) + " not found")
                    loaded[ref.evidence_id] = e
                if e.archived:
                    raise validation_field(
                        "blocks", "evidence " + str(ref.evidence_id) + " is archived")
                span = code_point_slice(e.source_text, ref.start, ref.end)
                if span is None:
                    raise validation_field(
                        "blocks",
                        "evidenceRef range out of bounds for " + str(ref.evidence_id))
                if not span or span not in b.text:
                    status = ent.CLAIM_NEEDS_REVIEW
            b.claim_status = status
        return loaded

    async def create_version(self, user_id: UUID, doc_id: UUID, expected: int,
                             content: dict, blocks: list[ent.Block],
                             change_note: str) -> tuple[ent.Document, ent.DocumentVersion]:
        out_doc = out_ver = None

        async def work():
            nonlocal out_doc, out_ver
            d = await self._get_mutable(user_id, doc_id)
            if d.revision != expected:
                raise revision_conflict(d.revision)
            dc.validate_document_content(content, blocks)
            new_blocks = copy.deepcopy(blocks)
            await self._verify_blocks(user_id, d.application_id, new_blocks)
            num = await self.documents.next_version_number(doc_id)
            now = _now()
            v = ent.DocumentVersion(
                id=uuid4(), user_id=user_id, document_id=doc_id,
                application_id=d.application_id, number=num, content=content,
                blocks=new_blocks, change_note=change_note,
                quality=compute_quality(new_blocks, ent.METHOD_RULE_BASED),
                created_at=now)
            await self.documents.create_version(v)
            d.latest_version_id = v.id
            d.finalized_version_id = None
            if d.status == ent.DOC_FINALIZED:
                d.status = ent.DOC_DRAFT
            d.updated_at = now
            try:
                await self.documents.update(d, expected)
            except Exception as err:
                raise map_revision_err(err)
            d.revision = expected + 1
            out_doc, out_ver = d, v

        await self.db.do(work)
        return out_doc, out_ver

    async def list_versions(self, user_id: UUID, doc_id: UUID, page):
        await self.get(user_id, doc_id)
        try:
            return await self.documents.list_versions(user_id, doc_id, page)
        except Exception:
            raise internal()

    async def get_version(self, user_id: UUID, doc_id: UUID,
                          version_id: UUID) -> ent.DocumentVersion:
        await self.get(user_id, doc_id)
        try:
            return await self.documents.get_version(user_id, doc_id, version_id)
        except NotFoundError:
            raise not_found()
        except Exception:
            raise internal()

    async def generate(self, user_id: UUID, doc_id: UUID, expected: int,
                       evidence_ids: list[UUID], analysis_id: Optional[UUID],
                       ai: Optional[ent.AiOptions],
                       language: Optional[str]) -> ent.Operation:
        await self.ai.check(user_id, ai)
        op = None

        async def work():
            nonlocal op
            d = await self._get_mutable(user_id, doc_id)
            if d.revision != expected:
                raise revision_conflict(d.revision)
            if not evidence_ids:
                raise validation_field(
                    "evidenceIds", "at least one evidence is required")
            entries = []
            for eid in evidence_ids:
                try:
                    e = await self.evidence.get(user_id, eid)
                except Exception:
                    raise validation_field(
                        "evidenceIds", "evidence " + str(eid) + " not found")
                if e.archived:
                    raise validation_field(
                        "evidenceIds", "evidence " + str(eid) + " is archived")
                await self.approvals.require_evidence_use(
                    user_id, d.application_id, eid)
                text = e.source_text
                if code_point_len(text) > 500:
                    text = code_point_slice(text, 0, 500)
                entries.append(ExcerptEntry(
                    evidence_id=eid, text=text, start=0, end=code_point_len(text)))
            if language is not None:
                d.language = language
            num = await self.documents.next_version_number(doc_id)
            now = _now()
            content, blocks = dc.build_excerpt_content(entries)
            v = ent.DocumentVersion(
                id=uuid4(), user_id=user_id, document_id=doc_id,
                application_id=d.application_id, number=num, content=content,
                blocks=blocks, change_note="generated",
                quality=compute_quality(blocks, ent.METHOD_RULE_BASED),
                created_at=now)
            await self.documents.create_version(v)
            d.latest_version_id = v.id
            d.finalized_version_id = None
            if d.status == ent.DOC_FINALIZED:
                d.status = ent.DOC_DRAFT
            d.updated_at = now
            try:
                await self.documents.update(d, expected)
            except Exception as err:
                raise map_revision_err(err)
            d.revision = expected + 1
            op = ent.Operation(
                id=uuid4(), user_id=user_id, type=ent.OP_DOCUMENT_GENERATE,
                application_id=d.application_id, status=ent.OP_SUCCEEDED,
                result=ent.OperationResult(
                    kind=ent.OP_DOCUMENT_GENERATE,
                    value={"document": to_jsonable(d), "version": to_jsonable(v)}),
                created_at=now, updated_at=now)
            await self.ops.create(op)

        await self.db.do(work)
        return op

    async def create_revision(self, user_id: UUID, doc_id: UUID, expected: int,
                              version_id: UUID, selection: ent.Selection,
                              action: str, instruction: str,
                              ai: Optional[ent.AiOptions]) -> ent.Operation:
        await self.ai.check(user_id, ai)
        if not ent.valid_revision_action(action):
            raise validation_field("action", "unsupported action")
        if (selection.from_ < 0 or selection.to <= selection.from_
                or not selection.text):
            raise validation_field(
                "selection", "requires from < to and matching text")
        op = None

        async def work():
            nonlocal op
            d = await self._get_mutable(user_id, doc_id)
            if d.revision != expected:
                raise revision_conflict(d.revision)
            try:
                v = await self.documents.get_version(user_id, doc_id, version_id)
            except Exception:
                raise not_found()
            if not any(selection.text in b.text for b in v.blocks):
                raise validation_field(
                    "selection",
                    "selection text does not match the version content")
            now = _now()
            replacement = selection.text
            if action == "SHORTEN" and len(replacement) > 1:
                replacement = replacement[: len(replacement) // 2]
            p = ent.RevisionProposal(
                id=uuid4(), user_id=user_id, document_id=doc_id,
                source_version_id=v.id, source_revision=d.revision,
                selection=selection, replacement=replacement, evidence_refs=[],
                claim_status=ent.CLAIM_NEEDS_REVIEW, created_at=now)
            await self.documents.create_proposal(p)
            op = ent.Operation(
                id=uuid4(), user_id=user_id, type=ent.OP_DOCUMENT_REVISE,
                application_id=d.application_id, status=ent.OP_SUCCEEDED,
                result=ent.OperationResult(kind=ent.OP_DOCUMENT_REVISE,
                                           value=to_jsonable(p)),
                created_at=now, updated_at=now)
            await self.ops.create(op)

        await self.db.do(work)
        return op

    async def apply_revision(self, user_id: UUID, doc_id: UUID, proposal_id: UUID,
                             expected: int) -> tuple[ent.Document, ent.DocumentVersion]:
        out_doc = out_ver = None

        async def work():
            nonlocal out_doc, out_ver
            d = await self._get_mutable(user_id, doc_id)
            if d.revision != expected:
                raise revision_conflict(d.revision)
            try:
                p = await self.documents.get_proposal(user_id, doc_id, proposal_id)
            except Exception:
                raise not_found()
            if p.applied_at is not None:
                raise invalid_transition("proposal already applied")
            if (d.latest_version_id is None
                    or d.latest_version_id != p.source_version_id
                    or p.source_revision != d.revision):
                raise approval_stale(
                    "document changed since the proposal was created")
            try:
                src = await self.documents.get_version(
                    user_id, doc_id, p.source_version_id)
            except Exception:
                raise internal()
            new_content = copy.deepcopy(src.content)
            applied = [False]
            dc.replace_in_node(new_content, p.selection.text, p.replacement, applied)
            new_blocks = copy.deepcopy(src.blocks)
            for b in new_blocks:
                if p.selection.text in b.text:
                    b.text = b.text.replace(p.selection.text, p.replacement, 1)
            if not applied[0]:
                raise approval_stale(
                    "selection text no longer present in the version")
            dc.validate_document_content(new_content, new_blocks)
            await self._verify_blocks(user_id, d.application_id, new_blocks)
            num = await self.documents.next_version_number(doc_id)
            now = _now()
            v = ent.DocumentVersion(
                id=uuid4(), user_id=user_id, document_id=doc_id,
                application_id=d.application_id, number=num, content=new_content,
                blocks=new_blocks, change_note="applied proposal " + str(p.id),
                quality=compute_quality(new_blocks, ent.METHOD_RULE_BASED),
                created_at=now)
            await self.documents.create_version(v)
            await self.documents.mark_proposal_applied(p.id, now)
            d.latest_version_id = v.id
            d.finalized_version_id = None
            if d.status == ent.DOC_FINALIZED:
                d.status = ent.DOC_DRAFT
            d.updated_at = now
            try:
                await self.documents.update(d, expected)
            except Exception as err:
                raise map_revision_err(err)
            d.revision = expected + 1
            out_doc, out_ver = d, v

        await self.db.do(work)
        return out_doc, out_ver

    async def review(self, user_id: UUID, doc_id: UUID,
                     version_id: UUID) -> ent.DocumentVersion:
        v = await self.get_version(user_id, doc_id, version_id)
        blocks = copy.deepcopy(v.blocks)
        d = await self.get(user_id, doc_id)
        await self._verify_blocks(user_id, d.application_id, blocks)
        v.blocks = blocks
        v.quality = compute_quality(blocks, ent.METHOD_RULE_BASED)
        return v

    def _version_hash(self, v: ent.DocumentVersion) -> str:
        return hash_json({"kind": ent.APPROVAL_DOCUMENT_FINALIZE, "versionId": v.id,
                          "content": v.content, "blocks": v.blocks})

    async def finalize(self, user_id: UUID, doc_id: UUID, expected: int,
                       version_id: UUID, approval_id: UUID) -> ent.Document:
        out = None

        async def work():
            nonlocal out
            d = await self._get_mutable(user_id, doc_id)
            if d.revision != expected:
                raise revision_conflict(d.revision)
            try:
                v = await self.documents.get_version(user_id, doc_id, version_id)
            except Exception:
                raise not_found()
            blocks = copy.deepcopy(v.blocks)
            refs = await self._verify_blocks(user_id, d.application_id, blocks)
            for b in blocks:
                if b.claim_status != ent.CLAIM_SUPPORTED:
                    raise unsupported_claim(
                        "block " + b.id + " is " + b.claim_status)
            for eid in refs:
                await self.approvals.require_evidence_use(
                    user_id, d.application_id, eid)
            v.blocks = blocks
            await self.approvals.consume(
                user_id, approval_id, ent.APPROVAL_DOCUMENT_FINALIZE,
                d.application_id, version_id, self._version_hash(v))
            now = _now()
            d.status = ent.DOC_FINALIZED
            d.finalized_version_id = version_id
            d.updated_at = now
            try:
                await self.documents.update(d, expected)
            except Exception as err:
                raise map_revision_err(err)
            d.revision = expected + 1
            await self.applications.add_event(ent.ApplicationEvent(
                id=uuid4(), user_id=user_id, application_id=d.application_id,
                type=ent.EVENT_DOCUMENT_FINAL,
                payload={"documentId": d.id, "versionId": version_id},
                created_at=now))
            out = d

        await self.db.do(work)
        return out

    async def archive(self, user_id: UUID, doc_id: UUID,
                      expected: int) -> ent.Document:
        out = None

        async def work():
            nonlocal out
            try:
                d = await self.documents.get(user_id, doc_id)
            except NotFoundError:
                raise not_found()
            if d.revision != expected:
                raise revision_conflict(d.revision)
            if d.status == ent.DOC_ARCHIVED:
                out = d
                return
            d.status = ent.DOC_ARCHIVED
            d.updated_at = _now()
            try:
                await self.documents.update(d, expected)
            except Exception as err:
                raise map_revision_err(err)
            d.revision = expected + 1
            out = d

        await self.db.do(work)
        return out

    async def create_export(self, user_id: UUID, doc_id: UUID, version_id: UUID,
                            format_: str, renderer_version: str) -> ent.DocumentExport:
        if format_ not in (EXPORT_PDF, EXPORT_DOCX):
            raise validation_field("format", "must be PDF or DOCX")
        if not renderer_version:
            raise validation_field("rendererVersion", "required")
        out = None

        async def work():
            nonlocal out
            try:
                d = await self.documents.get(user_id, doc_id)
            except NotFoundError:
                raise not_found()
            try:
                v = await self.documents.get_version(user_id, doc_id, version_id)
            except Exception:
                raise not_found()
            if d.finalized_version_id is None or d.finalized_version_id != v.id:
                raise document_not_finalized(
                    "only the finalized version can be exported")
            now = _now()
            e = ent.DocumentExport(
                id=uuid4(), user_id=user_id, document_id=doc_id, version_id=v.id,
                format=format_, template=d.template, language=d.language,
                content=v.content, blocks=v.blocks,
                content_hash=hash_bytes(json.dumps(
                    v.content, separators=(",", ":"), sort_keys=True).encode()),
                status=EXPORT_READY_TO_RENDER, renderer_version=renderer_version,
                created_at=now, updated_at=now)
            await self.documents.create_export(e)
            out = e

        await self.db.do(work)
        return out

    async def get_export(self, user_id: UUID, doc_id: UUID,
                         export_id: UUID) -> ent.DocumentExport:
        await self.get(user_id, doc_id)
        try:
            return await self.documents.get_export(user_id, doc_id, export_id)
        except NotFoundError:
            raise not_found()
        except Exception:
            raise internal()

    async def list_exports(self, user_id: UUID, doc_id: UUID, page):
        await self.get(user_id, doc_id)
        try:
            return await self.documents.list_exports(user_id, doc_id, page)
        except Exception:
            raise internal()

    async def record_export_result(self, user_id: UUID, doc_id: UUID, export_id: UUID,
                                   sha256: str, byte_length: int,
                                   page_count: Optional[int],
                                   validation_: ent.ExportValidation, status: str,
                                   error_code: Optional[str]) -> ent.DocumentExport:
        if status not in (EXPORT_SUCCEEDED, EXPORT_FAILED):
            raise validation_field("status", "must be SUCCEEDED or FAILED")
        if not sha256 or byte_length < 0:
            raise validation_field(
                "sha256", "sha256 and non-negative byteLength are required")
        out = None

        async def work():
            nonlocal out
            try:
                e = await self.documents.get_export(user_id, doc_id, export_id)
            except NotFoundError:
                raise not_found()
            if e.status != EXPORT_READY_TO_RENDER:
                raise invalid_transition("export result already recorded")
            e.status = status
            e.result = ent.ExportResult(
                sha256=sha256, byte_length=byte_length, page_count=page_count,
                validation=validation_, error_code=error_code)
            e.updated_at = _now()
            await self.documents.update_export_result(e)
            out = e

        await self.db.do(work)
        return out
