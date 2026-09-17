from __future__ import annotations
from urllib.parse import urlparse

from app.domain.entities import Block
from app.domain.errors import validation, validation_field

ALLOWED_NODE_TYPES = {"doc", "paragraph", "heading", "bulletList", "orderedList",
                      "listItem", "text"}
BLOCK_NODE_TYPES = {"paragraph", "heading", "listItem"}


def _validate_url(raw: str) -> None:
    u = urlparse(raw)
    if u.scheme not in ("http", "https"):
        raise validation("url must be http(s)")
    if u.username or u.password:
        raise validation("url must not contain credentials")
    if not u.hostname:
        raise validation("url must have a host")


def node_text(n: dict) -> str:
    out = [n.get("text") or ""]
    for c in n.get("content") or []:
        out.append(node_text(c))
    return "".join(out)


def _walk(n: dict, extracted: dict, order: list) -> None:
    t = n.get("type")
    if t not in ALLOWED_NODE_TYPES:
        raise validation_field("content", "unsupported node type " + str(t))
    for m in n.get("marks") or []:
        if m.get("type") != "link":
            raise validation_field("content", "unsupported mark type " + str(m.get("type")))
        href = (m.get("attrs") or {}).get("href")
        if not isinstance(href, str) or not href:
            raise validation_field("content", "link mark requires href")
        _validate_url(href)
    if t in BLOCK_NODE_TYPES:
        attrs = n.get("attrs")
        if not isinstance(attrs, dict) or not attrs.get("blockId"):
            raise validation_field(
                "content", f"block node of type {t} requires attrs.blockId")
        bid = attrs["blockId"]
        if bid in extracted:
            raise validation_field("content", "duplicate blockId " + str(bid))
        extracted[bid] = node_text(n)
        order.append(bid)
    for c in n.get("content") or []:
        _walk(c, extracted, order)


def validate_document_content(content, blocks: list[Block]) -> None:
    if not isinstance(content, dict):
        raise validation("content must be valid TipTap JSON")
    if content.get("type") != "doc":
        raise validation_field("content", "root node must be doc")
    extracted: dict = {}
    order: list = []
    _walk(content, extracted, order)
    seen: set = set()
    for b in blocks:
        if not b.id:
            raise validation_field("blocks", "block id is required")
        if b.id in seen:
            raise validation_field("blocks", "duplicate block id " + b.id)
        seen.add(b.id)
        if b.id not in extracted:
            raise validation_field(
                "blocks", "block " + b.id + " has no matching content node")
        if extracted[b.id] != b.text:
            raise validation_field(
                "blocks", "block " + b.id + " text does not match content node text")
    for bid in order:
        if bid not in seen:
            raise validation_field(
                "content", "content node " + bid + " has no matching block")


def extract_block_text(content) -> dict:
    if not isinstance(content, dict):
        raise validation("content must be valid TipTap JSON")
    extracted: dict = {}
    order: list = []
    _walk(content, extracted, order)
    return extracted


def code_point_slice(s: str, start: int, end: int):
    if start < 0 or end < 0 or start >= end:
        return None
    if end > len(s) or start >= len(s):
        return None
    return s[start:end]


def build_excerpt_content(entries: list) -> tuple[dict, list[Block]]:
    nodes, blocks = [], []
    for i, e in enumerate(entries, 1):
        bid = f"block-{i}"
        nodes.append({"type": "paragraph", "attrs": {"blockId": bid},
                      "content": [{"type": "text", "text": e.text}]})
        blocks.append(Block(
            id=bid, text=e.text,
            evidence_refs=[{"evidenceId": e.evidence_id, "start": e.start, "end": e.end}],
            claim_status="SUPPORTED"))
    return {"type": "doc", "content": nodes}, blocks


def replace_in_node(n: dict, old: str, new: str, state: list) -> None:
    if state[0]:
        return
    if n.get("type") == "text" and old in (n.get("text") or ""):
        n["text"] = n["text"].replace(old, new, 1)
        state[0] = True
        return
    for c in n.get("content") or []:
        replace_in_node(c, old, new, state)
