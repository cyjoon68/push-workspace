from __future__ import annotations
import os
from dataclasses import dataclass, field
from uuid import UUID


def env(key: str, default: str = "") -> str:
    return os.environ.get(key) or default


@dataclass
class OAuthProviderConfig:
    client_id: str = ""
    client_secret: str = ""


@dataclass
class Config:
    port: str = "8080"
    app_env: str = "development"
    database_url: str = ""
    dev_auth_token: str = ""
    dev_user_id: UUID = UUID("00000000-0000-4000-8000-000000000001")
    google: OAuthProviderConfig = field(default_factory=OAuthProviderConfig)
    github: OAuthProviderConfig = field(default_factory=OAuthProviderConfig)
    managed_ai_key: str = ""
    byok_master_key: str = ""
    stripe_secret: str = ""
    stripe_webhook_secret: str = ""
    stripe_price_pro: str = ""
    stripe_price_ultra: str = ""
    stripe_success_url: str = "push://billing/success"
    stripe_cancel_url: str = "push://billing/cancel"
    stripe_portal_url: str = "push://billing/portal"
    storage_dir: str = "./data/sources"
    gmail_beta: bool = False
    job_site_adapters: dict = field(default_factory=dict)


def load_config() -> Config:
    try:
        dev_user = UUID(env("DEV_USER_ID", "00000000-0000-4000-8000-000000000001"))
    except ValueError:
        dev_user = UUID("00000000-0000-4000-8000-000000000001")
    db_url = os.environ.get("DATABASE_URL") or (
        "postgres://"
        + env("PGUSER", "push") + ":" + env("PGPASSWORD", "push")
        + "@" + env("PGHOST", "localhost") + ":" + env("PGPORT", "5432")
        + "/" + env("PGDATABASE", "push") + "?sslmode=disable"
    )
    return Config(
        port=env("PORT") or env("LISTEN_ADDR", "8080").lstrip(":"),
        app_env=env("APP_ENV", "development"),
        database_url=db_url,
        dev_auth_token=os.environ.get("DEV_AUTH_TOKEN", ""),
        dev_user_id=dev_user,
        google=OAuthProviderConfig(env("GOOGLE_CLIENT_ID"), env("GOOGLE_CLIENT_SECRET")),
        github=OAuthProviderConfig(env("GITHUB_CLIENT_ID"), env("GITHUB_CLIENT_SECRET")),
        managed_ai_key=os.environ.get("OPENAI_API_KEY", ""),
        byok_master_key=os.environ.get("BYOK_MASTER_KEY", ""),
        stripe_secret=os.environ.get("STRIPE_SECRET_KEY", ""),
        stripe_webhook_secret=os.environ.get("STRIPE_WEBHOOK_SECRET", ""),
        stripe_price_pro=os.environ.get("STRIPE_PRICE_PRO", ""),
        stripe_price_ultra=os.environ.get("STRIPE_PRICE_ULTRA", ""),
        stripe_success_url=env("STRIPE_SUCCESS_URL", "push://billing/success"),
        stripe_cancel_url=env("STRIPE_CANCEL_URL", "push://billing/cancel"),
        stripe_portal_url=env("STRIPE_PORTAL_RETURN_URL", "push://billing/portal"),
        storage_dir=env("STORAGE_DIR", "./data/sources"),
        gmail_beta=os.environ.get("GOOGLE_GMAIL_BETA_ENABLED") == "true",
        job_site_adapters={},
    )
