from __future__ import annotations
import hashlib
from urllib.parse import urlparse


def validate_https_url(url: str) -> None:
    u = urlparse(url)
    if u.scheme not in ("http", "https") or not u.hostname or u.username or u.password:
        raise ValueError("must be an http(s) URL without credentials")


def hash_bytes(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def code_point_len(s: str) -> int:
    return len(s)


def truncate_runes(s: str, n: int) -> str:
    return s[:n]
