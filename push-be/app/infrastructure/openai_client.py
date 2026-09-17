from __future__ import annotations
import httpx


class OpenAIClient:
    def __init__(self, base_url: str = "https://api.openai.com/v1"):
        self._client = httpx.AsyncClient(timeout=60)
        self.base_url = base_url

    async def chat(self, api_key: str, model: str, system: str,
                   user: str) -> dict:
        messages = ([{"role": "system", "content": system}] if system else []) + [
            {"role": "user", "content": user}]
        resp = await self._client.post(
            self.base_url + "/chat/completions",
            json={"model": model, "messages": messages},
            headers={"Authorization": "Bearer " + api_key})
        body = resp.json()
        if resp.status_code != 200:
            msg = (body.get("error") or {}).get("message") or \
                f"openai status {resp.status_code}"
            raise ValueError(msg)
        if not body.get("choices"):
            raise ValueError("openai returned no choices")
        return {
            "text": body["choices"][0]["message"]["content"],
            "input_tokens": (body.get("usage") or {}).get("prompt_tokens", 0),
            "output_tokens": (body.get("usage") or {}).get("completion_tokens", 0),
        }

    async def aclose(self):
        await self._client.aclose()
