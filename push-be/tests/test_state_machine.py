from __future__ import annotations

from app.domain import entities as ent


def test_can_patch_stage():
    cases = [
        ("DISCOVERED", "PREPARING", True),
        ("PREPARING", "READY", True),
        ("READY", "APPLIED", False),
        ("APPLIED", "SCREENING", True),
        ("SCREENING", "INTERVIEW", True),
        ("INTERVIEW", "OFFER", True),
        ("OFFER", "ACCEPTED", True),
        ("DISCOVERED", "DISCOVERED", True),
        ("DISCOVERED", "INTERVIEW", False),
        ("DISCOVERED", "APPLIED", False),
        ("PREPARING", "DISCOVERED", False),
        ("DISCOVERED", "REJECTED", True),
        ("INTERVIEW", "WITHDRAWN", True),
        ("ACCEPTED", "REJECTED", False),
        ("REJECTED", "WITHDRAWN", False),
        ("WITHDRAWN", "DISCOVERED", False),
    ]
    for frm, to, want in cases:
        assert ent.can_patch_stage(frm, to) == want, (frm, to)


def test_valid_stage():
    for s in ("DISCOVERED", "PREPARING", "READY", "APPLIED", "SCREENING",
              "INTERVIEW", "OFFER", "ACCEPTED", "REJECTED", "WITHDRAWN"):
        assert ent.valid_stage(s)
    assert not ent.valid_stage("BOGUS")


def test_terminal_stage():
    for s in ("ACCEPTED", "REJECTED", "WITHDRAWN"):
        assert ent.terminal_stage(s)
    assert not ent.terminal_stage("INTERVIEW")


def test_next_stage():
    assert ent.next_stage("OFFER") == "ACCEPTED"
    assert ent.next_stage("ACCEPTED") is None
    assert ent.next_stage("REJECTED") is None


def test_can_transition_launch():
    cases = [
        ("NOT_CLAIMED", "CLAIMED", True),
        ("NOT_CLAIMED", "STARTED", False),
        ("CLAIMED", "STARTED", True),
        ("CLAIMED", "UNKNOWN", True),
        ("STARTED", "UNKNOWN", True),
        ("STARTED", "FINISHED", False),
        ("UNKNOWN", "STARTED", False),
        ("FINISHED", "FINISHED", True),
    ]
    for frm, to, want in cases:
        assert ent.can_transition_launch(frm, to) == want, (frm, to)


def test_valid_reported_launch_status():
    for s in ("CLAIMED", "STARTED", "UNKNOWN"):
        assert ent.valid_reported_launch_status(s)
    for s in ("NOT_CLAIMED", "FINISHED"):
        assert not ent.valid_reported_launch_status(s)
