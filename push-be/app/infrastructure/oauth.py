from __future__ import annotations
from dataclasses import dataclass

import httpx


@dataclass
class ProviderConfig:
    client_id: str
    client_secret: str
    auth_url: str
    token_url: str
    user_info_url: str


class OAuthClient:
    def __init__(self):
        self._client = httpx.AsyncClient(timeout=15)

    async def exchange_code(self, cfg: ProviderConfig, code: str, redirect_uri: str) -> str:
        resp = await self._client.post(cfg.token_url, data={
            "client_id": cfg.client_id,
            "client_secret": cfg.client_secret,
            "code": code,
            "redirect_uri": redirect_uri,
            "grant_type": "authorization_code",
        }, headers={"Accept": "application/json"})
        body = resp.json() if resp.content else {}
        if resp.status_code != 200 or not body.get("access_token"):
            raise ValueError(f"token exchange failed: status {resp.status_code} {body.get('error', '')}")
        return body["access_token"]

    async def fetch_identity(self, cfg: ProviderConfig, access_token: str) -> tuple[str, str]:
        resp = await self._client.get(cfg.user_info_url, headers={
            "Authorization": f"Bearer {access_token}",
            "Accept": "application/json",
        })
        if resp.status_code != 200:
            raise ValueError("identity fetch failed")
        body = resp.json()
        subject = body.get("sub") or (str(body["id"]) if body.get("id") else "")
        if not subject:
            raise ValueError("missing subject")
        name = body.get("name") or body.get("login") or body.get("email") or "user"
        return subject, name

    async def aclose(self) -> None:
        await self._client.aclose()


def provider_configs(cfg) -> dict[str, ProviderConfig]:
    return {
        "google": ProviderConfig(
            client_id=cfg.google.client_id, client_secret=cfg.google.client_secret,
            auth_url="https://accounts.google.com/o/oauth2/v2/auth",
            token_url="https://oauth2.googleapis.com/token",
            user_info_url="https://openidconnect.googleapis.com/v1/userinfo"),
        "github": ProviderConfig(
            client_id=cfg.github.client_id, client_secret=cfg.github.client_secret,
            auth_url="https://github.com/login/oauth/authorize",
            token_url="https://github.com/login/oauth/access_token",
            user_info_url="https://api.github.com/user"),
    }
