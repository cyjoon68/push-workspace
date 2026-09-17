from __future__ import annotations
from datetime import datetime, timezone
from urllib.parse import quote

import httpx

from app.domain.entities import GoogleEvent, GoogleMessageMeta, GoogleTokens


class SyncTokenInvalid(Exception):
    pass


def _parse_google_time(dt: str, d: str) -> datetime:
    if dt:
        try:
            return datetime.fromisoformat(dt.replace("Z", "+00:00"))
        except ValueError:
            pass
    if d:
        try:
            return datetime.strptime(d, "%Y-%m-%d").replace(tzinfo=timezone.utc)
        except ValueError:
            pass
    return datetime.fromtimestamp(0, timezone.utc)


class GoogleClient:
    def __init__(self, client_id: str, client_secret: str):
        self._client = httpx.AsyncClient(timeout=20)
        self._client_id = client_id
        self._client_secret = client_secret

    async def _token_request(self, form: dict) -> GoogleTokens:
        resp = await self._client.post(
            "https://oauth2.googleapis.com/token", data=form)
        body = resp.json()
        if resp.status_code != 200 or not body.get("access_token"):
            raise ValueError(
                f"google token request failed: status {resp.status_code} "
                f"{body.get('error', '')}")
        return GoogleTokens(
            access_token=body["access_token"],
            refresh_token=body.get("refresh_token", ""),
            expires_in=body.get("expires_in", 0),
            scope=body.get("scope", ""))

    async def exchange_code(self, code: str, redirect_uri: str) -> GoogleTokens:
        return await self._token_request({
            "client_id": self._client_id, "client_secret": self._client_secret,
            "code": code, "redirect_uri": redirect_uri,
            "grant_type": "authorization_code"})

    async def refresh_access_token(self, refresh_token: str) -> GoogleTokens:
        return await self._token_request({
            "client_id": self._client_id, "client_secret": self._client_secret,
            "refresh_token": refresh_token, "grant_type": "refresh_token"})

    async def revoke_token(self, token: str) -> None:
        await self._client.post(
            "https://oauth2.googleapis.com/revoke?token=" + quote(token, safe=""))

    async def _get_json(self, access_token: str, url: str):
        resp = await self._client.get(
            url, headers={"Authorization": "Bearer " + access_token})
        try:
            body = resp.json()
        except Exception:
            body = {}
        return resp.status_code, body

    async def list_message_ids(self, access_token: str, page_token: str):
        url = ("https://gmail.googleapis.com/gmail/v1/users/me/messages"
               "?maxResults=50")
        if page_token:
            url += "&pageToken=" + quote(page_token, safe="")
        status, body = await self._get_json(access_token, url)
        if status != 200:
            raise ValueError(f"gmail list failed: status {status}")
        ids = [m["id"] for m in body.get("messages", [])]
        return ids, body.get("nextPageToken", ""), str(body.get("historyId", ""))

    async def get_message(self, access_token: str, id_: str) -> GoogleMessageMeta:
        url = ("https://gmail.googleapis.com/gmail/v1/users/me/messages/" +
               quote(id_, safe="") +
               "?format=metadata&metadataHeaders=From"
               "&metadataHeaders=Subject&metadataHeaders=Date")
        status, body = await self._get_json(access_token, url)
        if status != 200:
            raise ValueError(f"gmail get failed: status {status}")
        m = GoogleMessageMeta(
            id=body.get("id", ""), thread_id=body.get("threadId", ""),
            snippet=body.get("snippet", ""))
        for h in (body.get("payload") or {}).get("headers", []):
            name = (h.get("name") or "").lower()
            if name == "from":
                m.from_ = h.get("value", "")
            elif name == "subject":
                m.subject = h.get("value", "")
        try:
            m.received_at = datetime.fromtimestamp(
                int(body["internalDate"]) / 1000, timezone.utc)
        except (KeyError, ValueError):
            pass
        return m

    async def list_events(self, access_token: str, sync_token: str):
        url = ("https://www.googleapis.com/calendar/v3/calendars/primary/events"
               "?maxResults=100")
        if sync_token:
            url += "&syncToken=" + quote(sync_token, safe="")
        else:
            since = (datetime.now(timezone.utc).timestamp()
                     - 90 * 24 * 3600)
            tmin = datetime.fromtimestamp(since, timezone.utc).strftime(
                "%Y-%m-%dT%H:%M:%SZ")
            url += "&singleEvents=true&timeMin=" + quote(tmin, safe="")
        status, body = await self._get_json(access_token, url)
        if status == 410:
            raise SyncTokenInvalid()
        if status != 200:
            raise ValueError(f"calendar list failed: status {status}")
        events = []
        for it in body.get("items", []):
            events.append(GoogleEvent(
                id=it.get("id", ""), status=it.get("status", ""),
                summary=it.get("summary", ""),
                time_zone=(it.get("start") or {}).get("timeZone", ""),
                starts_at=_parse_google_time(
                    (it.get("start") or {}).get("dateTime", ""),
                    (it.get("start") or {}).get("date", "")),
                ends_at=_parse_google_time(
                    (it.get("end") or {}).get("dateTime", ""),
                    (it.get("end") or {}).get("date", ""))))
        return events, body.get("nextSyncToken", "")

    async def aclose(self):
        await self._client.aclose()
