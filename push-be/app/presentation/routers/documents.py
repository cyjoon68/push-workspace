from __future__ import annotations

from fastapi import APIRouter, Request

from app.domain import entities as ent
from app.domain.errors import validation_field
from app.presentation import schemas as s
from app.presentation.deps import (
    bind_json, current_user, data, optional_query_uuid, page_body,
    page_request, param_id,
)

router = APIRouter()


def _deps(request: Request):
    return request.app.state.deps


def _ai(req: s.AiOptionsReq | None) -> ent.AiOptions | None:
    if req is None:
        return None
    return ent.AiOptions(provider=req.provider, model=req.model,
                         credential_mode=req.credential_mode, effort=req.effort)


@router.get("/documents")
async def list_documents(request: Request):
    d = _deps(request)
    page = page_request(request)
    application_id = optional_query_uuid(request, "applicationId")
    kind = request.query_params.get("kind") or None
    if kind is not None and not ent.valid_document_kind(kind):
        raise validation_field("kind", "unsupported kind")
    p = await d.documents.list(current_user(request).id, application_id,
                               kind, page)
    return page_body(p)


@router.post("/documents", status_code=201)
async def create_document(request: Request):
    d = _deps(request)
    req = await bind_json(request, s.CreateDocumentReq)
    doc = await d.documents.create(current_user(request).id,
                                   req.application_id, req.title, req.kind,
                                   req.template, req.language)
    return data(201, doc)


@router.get("/documents/{id}")
async def get_document(id: str, request: Request):
    d = _deps(request)
    doc = await d.documents.get(current_user(request).id, param_id(id, "id"))
    return data(200, doc)


@router.patch("/documents/{id}")
async def patch_document(id: str, request: Request):
    d = _deps(request)
    req = await bind_json(request, s.PatchDocumentReq)
    doc = await d.documents.patch(current_user(request).id, param_id(id, "id"),
                                  req.expected_revision, req.title,
                                  req.template, req.language)
    return data(200, doc)


@router.get("/documents/{id}/versions")
async def list_versions(id: str, request: Request):
    d = _deps(request)
    p = await d.documents.list_versions(current_user(request).id,
                                        param_id(id, "id"),
                                        page_request(request))
    return page_body(p)


@router.get("/documents/{id}/versions/{versionId}")
async def get_version(id: str, versionId: str, request: Request):
    d = _deps(request)
    v = await d.documents.get_version(current_user(request).id,
                                      param_id(id, "id"),
                                      param_id(versionId, "versionId"))
    return data(200, v)


@router.post("/documents/{id}/versions", status_code=201)
async def create_version(id: str, request: Request):
    d = _deps(request)
    req = await bind_json(request, s.CreateVersionReq)
    blocks = [ent.Block(id=b.id, text=b.text,
                        evidence_refs=[ent.EvidenceRef(
                            evidence_id=r.evidence_id, start=r.start,
                            end=r.end) for r in b.evidence_refs])
              for b in req.blocks]
    doc, v = await d.documents.create_version(
        current_user(request).id, param_id(id, "id"), req.expected_revision,
        req.content, blocks, req.change_note)
    return data(201, {"document": doc, "version": v})


@router.post("/documents/{id}/generate", status_code=202)
async def generate_document(id: str, request: Request):
    d = _deps(request)
    req = await bind_json(request, s.GenerateDocReq)
    op = await d.documents.generate(current_user(request).id,
                                    param_id(id, "id"), req.expected_revision,
                                    req.evidence_ids, req.analysis_id,
                                    _ai(req.ai), req.language)
    return data(202, op)


@router.post("/documents/{id}/revisions", status_code=202)
async def create_revision(id: str, request: Request):
    d = _deps(request)
    req = await bind_json(request, s.ReviseReq)
    sel = ent.Selection(from_=req.selection.from_, to=req.selection.to,
                        text=req.selection.text)
    op = await d.documents.create_revision(
        current_user(request).id, param_id(id, "id"), req.expected_revision,
        req.version_id, sel, req.action, req.instruction, _ai(req.ai))
    return data(202, op)


@router.post("/documents/{id}/revisions/{revisionId}/apply")
async def apply_revision(id: str, revisionId: str, request: Request):
    d = _deps(request)
    req = await bind_json(request, s.ExpectedRevisionReq)
    doc, v = await d.documents.apply_revision(
        current_user(request).id, param_id(id, "id"),
        param_id(revisionId, "revisionId"), req.expected_revision)
    return data(200, {"document": doc, "version": v})


@router.post("/documents/{id}/review")
async def review_document(id: str, request: Request):
    d = _deps(request)
    req = await bind_json(request, s.ReviewReq)
    v = await d.documents.review(current_user(request).id, param_id(id, "id"),
                                 req.version_id)
    return data(200, {"quality": v.quality, "blocks": v.blocks})


@router.post("/documents/{id}/finalize")
async def finalize_document(id: str, request: Request):
    d = _deps(request)
    req = await bind_json(request, s.FinalizeReq)
    doc = await d.documents.finalize(current_user(request).id,
                                     param_id(id, "id"),
                                     req.expected_revision, req.version_id,
                                     req.approval_id)
    return data(200, doc)


@router.post("/documents/{id}/archive")
async def archive_document(id: str, request: Request):
    d = _deps(request)
    req = await bind_json(request, s.ExpectedRevisionReq)
    doc = await d.documents.archive(current_user(request).id,
                                    param_id(id, "id"),
                                    req.expected_revision)
    return data(200, doc)


@router.post("/documents/{id}/exports", status_code=201)
async def create_export(id: str, request: Request):
    d = _deps(request)
    req = await bind_json(request, s.CreateExportReq)
    e = await d.documents.create_export(current_user(request).id,
                                        param_id(id, "id"), req.version_id,
                                        req.format, req.renderer_version)
    return data(201, e)


@router.post("/documents/{id}/exports/{exportId}/result")
async def record_export_result(id: str, exportId: str, request: Request):
    d = _deps(request)
    req = await bind_json(request, s.ExportResultReq)
    e = await d.documents.record_export_result(
        current_user(request).id, param_id(id, "id"),
        param_id(exportId, "exportId"), req.sha256, req.byte_length,
        req.page_count,
        ent.ExportValidation(korean_text=req.validation.korean_text,
                             links=req.validation.links,
                             ats_text=req.validation.ats_text),
        req.status, req.error_code)
    return data(200, e)


@router.get("/documents/{id}/exports")
async def list_exports(id: str, request: Request):
    d = _deps(request)
    p = await d.documents.list_exports(current_user(request).id,
                                       param_id(id, "id"),
                                       page_request(request))
    return page_body(p)
