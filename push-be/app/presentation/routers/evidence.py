from __future__ import annotations

import hashlib
import os
from datetime import datetime, timezone
from uuid import uuid4

from fastapi import APIRouter, Request, UploadFile
from fastapi.responses import FileResponse, Response

from app.domain import entities as ent
from app.domain.errors import (
    internal, invalid_transition, not_found, payload_too_large, validation_field,
)
from app.presentation import schemas as s
from app.presentation.deps import (
    bind_json, current_user, data, page_body, page_request, param_id,
)

router = APIRouter()

ALLOWED_SOURCE_MIME = {
    "application/pdf",
    "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
    "text/plain",
    "text/markdown",
    "application/octet-stream",
}
MAX_SOURCE_BYTES = 20 << 20


def _deps(request: Request):
    return request.app.state.deps


def source_dto(src: ent.Source) -> dict:
    return {"id": src.id, "fileName": src.file_name, "mimeType": src.mime_type,
            "size": src.size, "sha256": src.sha256, "status": src.status,
            "createdAt": src.created_at}


@router.get("/career-evidence")
async def list_evidence(request: Request):
    d = _deps(request)
    page = page_request(request)
    kind = request.query_params.get("kind") or None
    if kind is not None and not ent.valid_evidence_kind(kind):
        raise validation_field("kind", "unsupported kind")
    p = await d.evidence.list(current_user(request).id, kind,
                              request.query_params.get("query", ""), page)
    return page_body(p)


@router.post("/career-evidence", status_code=201)
async def create_evidence(request: Request):
    d = _deps(request)
    req = await bind_json(request, s.CreateEvidenceReq)
    e = await d.evidence.create(current_user(request).id, req.kind, req.title,
                                req.source_text, req.source_url, req.skills,
                                req.supersedes_id)
    return data(201, e)


@router.get("/career-evidence/{id}")
async def get_evidence(id: str, request: Request):
    d = _deps(request)
    e = await d.evidence.get(current_user(request).id, param_id(id, "id"))
    return data(200, e)


@router.post("/career-evidence/import", status_code=202)
async def import_evidence(request: Request):
    d = _deps(request)
    req = await bind_json(request, s.ImportEvidenceReq)
    op = await d.evidence.import_(current_user(request).id, req.source_id,
                                  req.text, req.source_url, req.content_hash,
                                  req.format)
    return data(202, op)


@router.post("/career-evidence/{id}/archive")
async def archive_evidence(id: str, request: Request):
    d = _deps(request)
    req = await bind_json(request, s.ExpectedRevisionReq)
    e = await d.evidence.archive(current_user(request).id,
                                 param_id(id, "id"), req.expected_revision)
    return data(200, e)


@router.post("/sources", status_code=201)
async def create_source(request: Request):
    d = _deps(request)
    user = current_user(request)
    form = await request.form()
    kind = form.get("kind")
    if not kind or not isinstance(kind, str):
        raise validation_field("kind", "required")
    file = form.get("file")
    if not isinstance(file, UploadFile):
        raise validation_field("file", "required")
    if file.size is not None and file.size > MAX_SOURCE_BYTES:
        raise payload_too_large()
    buf = await file.read(MAX_SOURCE_BYTES + 1)
    if len(buf) > MAX_SOURCE_BYTES:
        raise payload_too_large()
    mime = file.content_type or "application/octet-stream"
    if mime not in ALLOWED_SOURCE_MIME:
        raise validation_field("file", "unsupported file type " + mime)
    sha = hashlib.sha256(buf).hexdigest()
    src_id = uuid4()
    path = os.path.join(d.storage_dir, str(user.id), str(src_id))
    os.makedirs(os.path.dirname(path), mode=0o700, exist_ok=True)
    with open(path, "wb") as f:
        f.write(buf)
    os.chmod(path, 0o600)
    src = ent.Source(id=src_id, user_id=user.id,
                     file_name=file.filename or "", mime_type=mime,
                     size=len(buf), sha256=sha, path=path, status="STORED",
                     created_at=datetime.now(timezone.utc))
    try:
        await d.sources.create(src)
    except Exception:
        raise internal()
    return data(201, {"id": src.id, "fileName": src.file_name,
                      "mimeType": src.mime_type, "size": src.size,
                      "sha256": src.sha256, "status": src.status})


@router.get("/sources/{id}")
async def get_source(id: str, request: Request):
    d = _deps(request)
    try:
        src = await d.sources.get(current_user(request).id,
                                  param_id(id, "id"))
    except Exception:
        raise not_found()
    return data(200, source_dto(src))


@router.get("/sources/{id}/content")
async def get_source_content(id: str, request: Request):
    d = _deps(request)
    try:
        src = await d.sources.get(current_user(request).id,
                                  param_id(id, "id"))
    except Exception:
        raise not_found()
    return FileResponse(src.path)


@router.delete("/sources/{id}", status_code=204)
async def delete_source(id: str, request: Request):
    d = _deps(request)
    user_id = current_user(request).id
    sid = param_id(id, "id")
    try:
        ref = await d.sources.referenced(user_id, sid)
    except Exception:
        raise internal()
    if ref:
        raise invalid_transition("source is referenced by evidence")
    try:
        await d.sources.delete(user_id, sid)
    except Exception:
        raise not_found()
    return Response(status_code=204)
