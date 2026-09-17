from __future__ import annotations

from types import SimpleNamespace
from uuid import uuid4

import pytest

from app.domain.content import (
    build_excerpt_content, code_point_slice, validate_document_content,
)
from app.domain.entities import Block
from app.domain.errors import DomainError
from app.domain.validators import code_point_len, validate_https_url


def para(block_id: str, text: str) -> dict:
    return {"type": "paragraph", "attrs": {"blockId": block_id},
            "content": [{"type": "text", "text": text}]}


def test_validate_document_content_valid():
    content = {"type": "doc",
               "content": [para("b1", "hello"), para("b2", "world")]}
    blocks = [Block(id="b1", text="hello", claim_status="SUPPORTED"),
              Block(id="b2", text="world", claim_status="SUPPORTED")]
    validate_document_content(content, blocks)


def test_root_not_doc():
    with pytest.raises(DomainError) as e:
        validate_document_content({"type": "paragraph"}, [])
    assert e.value.code == "VALIDATION_ERROR"


def test_unsupported_node():
    content = {"type": "doc", "content": [{"type": "script"}]}
    with pytest.raises(DomainError):
        validate_document_content(content, [])


def test_block_mismatch():
    content = {"type": "doc", "content": [para("b1", "hello")]}
    with pytest.raises(DomainError):
        validate_document_content(content, [Block(id="b1", text="different")])
    with pytest.raises(DomainError):
        validate_document_content(content, [Block(id="b9", text="hello")])
    with pytest.raises(DomainError):
        validate_document_content(content, [])


def test_duplicate_block_id():
    content = {"type": "doc",
               "content": [para("b1", "a"), para("b1", "b")]}
    blocks = [Block(id="b1", text="a"), Block(id="b1", text="b")]
    with pytest.raises(DomainError):
        validate_document_content(content, blocks)


def test_link_mark():
    bad = {"type": "doc", "content": [{
        "type": "paragraph", "attrs": {"blockId": "b1"},
        "content": [{"type": "text", "text": "x", "marks": [
            {"type": "link",
             "attrs": {"href": "javascript:alert(1)"}}]}]}]}
    with pytest.raises(DomainError):
        validate_document_content(bad, [Block(id="b1", text="x")])


def test_validate_https_url():
    for u in ("https://example.com/x", "http://a.b/path?q=1"):
        validate_https_url(u)
    for u in ("ftp://x", "https://user:pw@x/", "notaurl", ""):
        with pytest.raises(ValueError):
            validate_https_url(u)


def test_code_point_slice():
    assert code_point_slice("héllo wörld", 0, 5) == "héllo"
    assert code_point_slice("abc", 2, 2) is None
    assert code_point_len("한국어") == 3


def test_build_excerpt_content_roundtrip():
    eid = uuid4()
    content, blocks = build_excerpt_content(
        [SimpleNamespace(evidence_id=eid, text="built a service",
                         start=0, end=15)])
    validate_document_content(content, blocks)
    assert blocks[0].evidence_refs[0].evidence_id == eid
    assert blocks[0].claim_status == "SUPPORTED"
