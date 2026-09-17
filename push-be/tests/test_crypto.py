from __future__ import annotations

from uuid import uuid4

import pytest

from app.infrastructure.crypto import new_key_cipher


def test_key_cipher_roundtrip():
    c = new_key_cipher("test-master-key")
    uid = uuid4()
    ct, nonce = c.encrypt("secret-token", uid, "google")
    assert c.decrypt(ct, nonce, uid, "google") == "secret-token"


def test_key_cipher_wrong_purpose():
    c = new_key_cipher("test-master-key")
    uid = uuid4()
    ct, nonce = c.encrypt("secret-token", uid, "google")
    with pytest.raises(Exception):
        c.decrypt(ct, nonce, uid, "openai")
    with pytest.raises(Exception):
        c.decrypt(ct, nonce, uuid4(), "google")
