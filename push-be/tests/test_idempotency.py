from __future__ import annotations

from datetime import datetime, timezone
from types import SimpleNamespace
from uuid import UUID, uuid4

from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse
from httpx import ASGITransport, AsyncClient

from app.db import NotFoundError
from app.domain.entities import User
from app.presentation.middleware import ApiMiddleware

USER = User(id=UUID("00000000-0000-4000-8000-000000000001"),
            display_name="Dev User", locale="ko",
            created_at=datetime.now(timezone.utc))


class StubAuth:
    async def resolve_access_token(self, token: str):
        if token == "good":
            return USER
        from app.domain.errors import unauthenticated
        raise unauthenticated("invalid access token")


class StubIdem:
    def __init__(self):
        self.records = {}

    def _key(self, user_id, method, path, key):
        return f"{user_id}|{method}|{path}|{key}"

    async def insert_pending(self, rec) -> bool:
        k = self._key(rec.user_id, rec.method, rec.path, rec.key)
        if k in self.records:
            return False
        self.records[k] = rec
        return True

    async def get(self, user_id, method, path, key):
        rec = self.records.get(self._key(user_id, method, path, key))
        if rec is None:
            raise NotFoundError()
        return rec

    async def complete(self, id_, status, body):
        for rec in self.records.values():
            if rec.id == id_:
                rec.response_status = status
                rec.response_body = body
                return
        raise NotFoundError()

    async def delete_pending(self, id_):
        for k, rec in list(self.records.items()):
            if rec.id == id_:
                del self.records[k]
                return
        raise NotFoundError()


def make_app():
    app = FastAPI()
    calls = {"n": 0}

    @app.post("/api/v1/things")
    async def things(request: Request):
        calls["n"] += 1
        return JSONResponse(status_code=201, content={"data": {"n": calls["n"]}})

    app.state.deps = SimpleNamespace(auth=StubAuth(), idem=StubIdem())
    app.add_middleware(ApiMiddleware)
    return app, calls


def post(client_args):
    return None


async def do_post(app, key: str, body: str):
    headers = {"Authorization": "Bearer good", "Content-Type": "application/json"}
    if key:
        headers["Idempotency-Key"] = key
    async with AsyncClient(transport=ASGITransport(app=app),
                           base_url="http://t") as c:
        return await c.post("/api/v1/things", content=body, headers=headers)


async def test_key_required():
    app, calls = make_app()
    r = await do_post(app, "", '{"a":1}')
    assert r.status_code == 400
    assert r.json()["error"]["code"] == "VALIDATION_ERROR"
    assert calls["n"] == 0


async def test_key_must_be_uuid():
    app, calls = make_app()
    r = await do_post(app, "not-a-uuid", '{"a":1}')
    assert r.status_code == 400
    assert calls["n"] == 0


async def test_replay_returns_stored():
    app, calls = make_app()
    key = str(uuid4())
    first = await do_post(app, key, '{"a":1}')
    assert first.status_code == 201 and calls["n"] == 1
    second = await do_post(app, key, '{"a":1}')
    assert second.status_code == 201 and calls["n"] == 1
    assert first.content == second.content


async def test_conflict_on_different_body():
    app, calls = make_app()
    key = str(uuid4())
    await do_post(app, key, '{"a":1}')
    r = await do_post(app, key, '{"a":2}')
    assert r.status_code == 409
    assert r.json()["error"]["code"] == "IDEMPOTENCY_CONFLICT"
    assert calls["n"] == 1


async def test_different_key_runs_again():
    app, calls = make_app()
    await do_post(app, str(uuid4()), '{"a":1}')
    await do_post(app, str(uuid4()), '{"a":1}')
    assert calls["n"] == 2


async def test_get_skips():
    app, calls = make_app()

    @app.get("/api/v1/things")
    async def get_things():
        calls["n"] += 1
        return JSONResponse(status_code=200, content={})

    async with AsyncClient(transport=ASGITransport(app=app),
                           base_url="http://t") as c:
        r = await c.get("/api/v1/things",
                        headers={"Authorization": "Bearer good"})
    assert r.status_code == 200 and calls["n"] == 1


async def test_unauthenticated():
    app, calls = make_app()
    async with AsyncClient(transport=ASGITransport(app=app),
                           base_url="http://t") as c:
        r = await c.post("/api/v1/things", content="{}")
        r2 = await c.post("/api/v1/things", content="{}",
                          headers={"Authorization": "Bearer bad",
                                   "Idempotency-Key": str(uuid4())})
    assert r.status_code == 401
    assert r2.status_code == 401
    assert calls["n"] == 0
