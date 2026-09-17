from __future__ import annotations

import json

from fastapi import APIRouter, Request
from fastapi.responses import StreamingResponse

from app.domain import entities as ent
from app.jsonutil import to_jsonable
from app.presentation import schemas as s
from app.presentation.deps import bind_json, current_user, data, param_id

router = APIRouter()


def _deps(request: Request):
    return request.app.state.deps


@router.get("/operations/{id}")
async def get_operation(id: str, request: Request):
    d = _deps(request)
    op = await d.operations.get(current_user(request).id, param_id(id, "id"))
    return data(200, op)


@router.post("/operations/{id}/input")
async def operation_input(id: str, request: Request):
    d = _deps(request)
    req = await bind_json(request, s.OpInputReq)
    op = await d.operations.submit_input(current_user(request).id,
                                         param_id(id, "id"), req.fields,
                                         req.source_id)
    return data(200, op)


@router.post("/operations/{id}/cancel")
async def cancel_operation(id: str, request: Request):
    d = _deps(request)
    op = await d.operations.cancel(current_user(request).id,
                                   param_id(id, "id"))
    return data(200, op)


@router.get("/operations/{id}/events")
async def operation_events(id: str, request: Request):
    d = _deps(request)
    op = await d.operations.get(current_user(request).id, param_id(id, "id"))
    seq = 0

    def emit(kind: str, payload) -> str:
        nonlocal seq
        seq += 1
        b = json.dumps({"operationId": str(op.id), "sequence": seq,
                        "payload": to_jsonable(payload)})
        return f"id: {op.id}:{seq}\nevent: {kind}\ndata: {b}\n\n"

    async def stream():
        yield emit("progress", {"status": op.status, "progress": op.progress,
                                "step": op.status})
        if op.status == ent.OP_SUCCEEDED and op.result is not None:
            yield emit("result", op.result.value)
        elif op.status == ent.OP_FAILED:
            yield emit("error", op.error)
        elif op.status == ent.OP_CANCELLED:
            yield emit("error", ent.OperationError(
                code="CANCELLED", message="operation cancelled",
                retryable=False))

    return StreamingResponse(
        stream(), media_type="text/event-stream",
        headers={"Cache-Control": "no-cache", "Connection": "keep-alive"})
