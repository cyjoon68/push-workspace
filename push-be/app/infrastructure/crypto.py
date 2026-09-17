from __future__ import annotations
import base64
import hashlib
import os
import secrets
from uuid import UUID

from cryptography.hazmat.primitives.ciphers.aead import AESGCM


def new_key_cipher(master: str) -> "KeyCipher":
    try:
        raw = base64.b64decode(master, validate=True) if master else b""
    except Exception:
        raw = b""
    if len(raw) != 32:
        if not master:
            raise ValueError("empty master key")
        raw = hashlib.sha256(master.encode()).digest()
    return KeyCipher(raw)


class KeyCipher:
    def __init__(self, key: bytes):
        self._aes = AESGCM(key)

    @staticmethod
    def _aad(user_id: UUID, provider: str) -> bytes:
        return f"{user_id}|{provider}|v1".encode()

    def encrypt(self, plaintext: str, user_id: UUID, provider: str) -> tuple[bytes, bytes]:
        nonce = os.urandom(12)
        return self._aes.encrypt(nonce, plaintext.encode(), self._aad(user_id, provider)), nonce

    def decrypt(self, ciphertext: bytes, nonce: bytes, user_id: UUID, provider: str) -> str:
        return self._aes.decrypt(nonce, ciphertext, self._aad(user_id, provider)).decode()


def random_token() -> str:
    return secrets.token_urlsafe(32)


def sha256_hex(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def token_hash(token: str) -> str:
    return sha256_hex(token.encode())


def sha256_b64(data: bytes) -> str:
    return base64.b64encode(hashlib.sha256(data).digest()).decode()


def verify_pkce(verifier: str, challenge: str) -> bool:
    if not verifier:
        return False
    digest = hashlib.sha256(verifier.encode()).digest()
    return secrets.compare_digest(
        base64.urlsafe_b64encode(digest).rstrip(b"=").decode(), challenge)
