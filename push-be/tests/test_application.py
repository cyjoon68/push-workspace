from __future__ import annotations

from datetime import datetime, timezone
from uuid import uuid4

import pytest

from app.db import ConflictError
from app.domain import entities as ent
from app.domain.errors import DomainError
from app.domain.services.application import ApplicationService
from tests.stubs import FakeDB, StubApplicationStore


def _now():
    return datetime.now(timezone.utc)


def _app(stage="DISCOVERED", revision=1, user_id=None):
    return ent.Application(
        id=uuid4(), user_id=user_id or uuid4(), revision=revision,
        job_id=uuid4(), company="Acme", title="Dev", stage=stage,
        created_at=_now(), updated_at=_now())


def _svc(store):
    svc = ApplicationService(FakeDB(), None, {})
    svc.applications = store
    return svc


def _code(err):
    assert isinstance(err, DomainError)
    return err.code


async def test_patch_invalid_transition():
    user_id = uuid4()
    a = _app(user_id=user_id)
    svc = _svc(StubApplicationStore(app=a))
    with pytest.raises(DomainError) as e:
        await svc.patch(user_id, a.id, 1, stage="SCREENING", notes=None,
                        next_action_at=None, clear_next_action=False)
    assert e.value.code == "INVALID_TRANSITION"
    with pytest.raises(DomainError) as e:
        await svc.patch(user_id, a.id, 1, stage="APPLIED", notes=None,
                        next_action_at=None, clear_next_action=False)
    assert e.value.code == "INVALID_TRANSITION"


async def test_patch_valid_transition_records_event():
    user_id = uuid4()
    a = _app(user_id=user_id)
    store = StubApplicationStore(app=a)
    out = await _svc(store).patch(user_id, a.id, 1, stage="PREPARING",
                                  notes=None, next_action_at=None,
                                  clear_next_action=False)
    assert out.stage == "PREPARING" and out.revision == 2
    assert len(store.events) == 1
    assert store.events[0].type == ent.EVENT_STAGE_CHANGED


async def test_patch_revision_conflict():
    user_id = uuid4()
    a = _app(user_id=user_id, revision=5)
    with pytest.raises(DomainError) as e:
        await _svc(StubApplicationStore(app=a)).patch(
            user_id, a.id, 3, stage=None, notes=None, next_action_at=None,
            clear_next_action=False)
    assert e.value.code == "REVISION_CONFLICT"
    assert e.value.details["currentRevision"] == 5


async def test_patch_revision_conflict_from_store():
    user_id = uuid4()
    a = _app(user_id=user_id, revision=2)
    store = StubApplicationStore(app=a, update_err=ConflictError(7))
    with pytest.raises(DomainError) as e:
        await _svc(store).patch(user_id, a.id, 2, stage="PREPARING",
                                notes=None, next_action_at=None,
                                clear_next_action=False)
    assert e.value.code == "REVISION_CONFLICT"
    assert e.value.details["currentRevision"] == 7


async def test_patch_ownership_isolation():
    a = _app()
    with pytest.raises(DomainError) as e:
        await _svc(StubApplicationStore(app=a)).patch(
            uuid4(), a.id, 1, stage=None, notes=None, next_action_at=None,
            clear_next_action=False)
    assert e.value.code == "NOT_FOUND"


async def test_import_requires_confirmation():
    svc = _svc(StubApplicationStore())
    with pytest.raises(DomainError) as e:
        await svc.import_(uuid4(), uuid4(), "APPLIED", _now(), "", False)
    assert e.value.code == "VALIDATION_ERROR"


async def test_import_rejects_discovered():
    svc = _svc(StubApplicationStore())
    with pytest.raises(DomainError) as e:
        await svc.import_(uuid4(), uuid4(), "DISCOVERED", _now(), "", True)
    assert e.value.code == "VALIDATION_ERROR"
