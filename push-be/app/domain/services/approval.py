from __future__ import annotations
from datetime import datetime, timezone
from typing import Optional
from uuid import UUID, uuid4

from app.db import DB, NotFoundError
from app.domain import entities as ent
from app.domain.errors import (
    approval_required, approval_stale, internal, invalid_transition,
    map_revision_err, not_found, revision_conflict, validation_field,
)
from app.domain.validators import hash_bytes
from app.infrastructure.store_applications import ApplicationStore, SubmissionStore
from app.infrastructure.store_documents import DocumentStore
from app.infrastructure.store_evidence import EvidenceStore
from app.infrastructure.store_operations import ApprovalStore
from app.infrastructure.store_projects import ProjectStore
from app.jsonutil import to_jsonable
import json


def _now() -> datetime:
    return datetime.now(timezone.utc)


def hash_json(v) -> str:
    return hash_bytes(json.dumps(to_jsonable(v), separators=(",", ":"),
                                 sort_keys=True).encode())


class ApprovalService:
    def __init__(self, db: DB):
        self.db = db
        self.approvals = ApprovalStore(db)
        self.applications = ApplicationStore(db)
        self.evidence = EvidenceStore(db)
        self.documents = DocumentStore(db)
        self.submissions = SubmissionStore(db)
        self.projects = ProjectStore(db)

    async def list(self, user_id: UUID, application_id: Optional[UUID],
                   status: Optional[str], page):
        try:
            return await self.approvals.list(user_id, application_id, status, page)
        except Exception:
            raise internal()

    async def get(self, user_id: UUID, id_: UUID) -> tuple[ent.Approval, dict]:
        try:
            a = await self.approvals.get(user_id, id_)
        except NotFoundError:
            raise not_found()
        except Exception:
            raise internal()
        return a, await self._target_summary(user_id, a)

    async def _target_summary(self, user_id: UUID, a: ent.Approval) -> dict:
        try:
            if a.kind == ent.APPROVAL_EVIDENCE_USE:
                e = await self.evidence.get(user_id, a.target_id)
                return {"type": "EVIDENCE", "id": str(e.id), "title": e.title, "kind": e.kind}
            if a.kind == ent.APPROVAL_DOCUMENT_FINALIZE:
                v = await self.documents.get_version(user_id, UUID(int=0), a.target_id)
                return {"type": "DOCUMENT_VERSION", "id": str(v.id),
                        "documentId": str(v.document_id), "number": v.number}
            if a.kind == ent.APPROVAL_APPLICATION_SUBMIT:
                d = await self.submissions.get_draft(user_id, a.target_id)
                return {"type": "SUBMISSION_DRAFT", "id": str(d.id), "mode": d.mode,
                        "documentVersionIds": [str(x) for x in d.document_version_ids]}
            if a.kind == ent.APPROVAL_CLI_EXECUTE:
                r = await self.projects.get_run(user_id, UUID(int=0), a.target_id)
                return {"type": "CLI_RUN", "id": str(r.id), "provider": r.provider,
                        "workingDirectory": r.working_directory, "executable": r.executable,
                        "arguments": r.arguments, "prompt": r.prompt}
        except Exception:
            raise internal()
        return {}

    async def create(self, user_id: UUID, kind: str, application_id: UUID,
                     target_id: UUID, target_revision: Optional[int]) -> ent.Approval:
        if not ent.valid_approval_kind(kind):
            raise validation_field("kind", "unsupported kind")
        try:
            await self.applications.get(user_id, application_id)
        except NotFoundError:
            raise not_found()
        except Exception:
            raise internal()
        h = await self._hash_target(user_id, kind, application_id, target_id)
        now = _now()
        a = ent.Approval(
            id=uuid4(), user_id=user_id, revision=1, kind=kind,
            application_id=application_id, target_id=target_id,
            target_revision=target_revision, payload_hash=h,
            status=ent.APPROVAL_PENDING, expires_at=now + ent.DEFAULT_APPROVAL_TTL,
            created_at=now, updated_at=now)
        try:
            await self.approvals.create(a)
        except Exception:
            raise internal()
        return a

    async def _hash_target(self, user_id: UUID, kind: str, application_id: UUID,
                           target_id: UUID) -> str:
        nil = UUID(int=0)
        if kind == ent.APPROVAL_EVIDENCE_USE:
            try:
                e = await self.evidence.get(user_id, target_id)
            except Exception:
                raise not_found()
            if e.archived:
                raise invalid_transition("cannot approve archived evidence for new use")
            return hash_json({"kind": kind, "evidenceId": e.id,
                              "contentHash": e.provenance.content_hash})
        if kind == ent.APPROVAL_DOCUMENT_FINALIZE:
            try:
                v = await self.documents.get_version(user_id, nil, target_id)
            except Exception:
                raise not_found()
            if v.application_id != application_id:
                raise not_found()
            return hash_json({"kind": kind, "versionId": v.id,
                              "content": v.content, "blocks": v.blocks})
        if kind == ent.APPROVAL_APPLICATION_SUBMIT:
            try:
                d = await self.submissions.get_draft(user_id, target_id)
            except Exception:
                raise not_found()
            if d.application_id != application_id:
                raise not_found()
            return d.payload_hash
        if kind == ent.APPROVAL_CLI_EXECUTE:
            try:
                r = await self.projects.get_run(user_id, nil, target_id)
            except Exception:
                raise not_found()
            if r.application_id != application_id:
                raise not_found()
            return r.payload_hash
        raise validation_field("kind", "unsupported kind")

    async def decide(self, user_id: UUID, id_: UUID, expected: int,
                     decision: str) -> ent.Approval:
        if decision not in (ent.APPROVAL_APPROVED, ent.APPROVAL_DENIED):
            raise validation_field("decision", "must be APPROVED or DENIED")
        out = None

        async def work():
            nonlocal out
            try:
                a = await self.approvals.get(user_id, id_)
            except NotFoundError:
                raise not_found()
            if a.revision != expected:
                raise revision_conflict(a.revision)
            now = _now()
            if a.status == ent.APPROVAL_PENDING and a.expired(now):
                a.status = ent.APPROVAL_EXPIRED
                a.updated_at = now
                try:
                    await self.approvals.update(a, expected)
                except Exception as err:
                    raise map_revision_err(err)
                a.revision = expected + 1
                raise invalid_transition("approval expired")
            if a.status != ent.APPROVAL_PENDING:
                raise invalid_transition("approval already decided")
            a.status = decision
            a.decided_at = now
            a.updated_at = now
            try:
                await self.approvals.update(a, expected)
            except Exception as err:
                raise map_revision_err(err)
            a.revision = expected + 1
            out = a

        await self.db.do(work)
        return out

    async def consume(self, user_id: UUID, approval_id: UUID, kind: str,
                      application_id: UUID, target_id: UUID,
                      expected_hash: str) -> ent.Approval:
        try:
            a = await self.approvals.get(user_id, approval_id)
        except NotFoundError:
            raise approval_required("approval not found")
        except Exception:
            raise internal()
        if (a.kind != kind or a.application_id != application_id
                or a.target_id != target_id):
            raise approval_required("approval does not cover this target")
        now = _now()
        if a.expired(now):
            if a.status in (ent.APPROVAL_PENDING, ent.APPROVAL_APPROVED):
                a.status = ent.APPROVAL_EXPIRED
                a.updated_at = now
                try:
                    await self.approvals.update(a, a.revision)
                except Exception:
                    pass
            raise approval_required("approval expired")
        if a.status != ent.APPROVAL_APPROVED:
            raise approval_required("approval is not approved")
        if expected_hash and a.payload_hash != expected_hash:
            raise approval_stale("approved content has changed")
        if ent.one_shot_approval(kind):
            a.status = ent.APPROVAL_CONSUMED
            a.consumed_at = now
            a.updated_at = now
            try:
                await self.approvals.update(a, a.revision)
            except Exception as err:
                raise map_revision_err(err)
            a.revision += 1
        return a

    async def require_evidence_use(self, user_id: UUID, application_id: UUID,
                                   evidence_id: UUID) -> None:
        try:
            a = await self.approvals.find_active(
                user_id, ent.APPROVAL_EVIDENCE_USE, application_id, evidence_id, _now())
        except NotFoundError:
            raise approval_required(
                "EVIDENCE_USE approval required for evidence " + str(evidence_id))
        except Exception:
            raise internal()
        if a.status != ent.APPROVAL_APPROVED:
            raise approval_required(
                "EVIDENCE_USE approval required for evidence " + str(evidence_id))
