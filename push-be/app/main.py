from __future__ import annotations

from contextlib import asynccontextmanager
from dataclasses import dataclass, field
from pathlib import Path

from fastapi import FastAPI
from fastapi.exceptions import RequestValidationError
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse
from starlette.exceptions import HTTPException

from app.config import Config, load_config
from app.db import create_pool, run_migrations, DB
from app.domain.errors import DomainError
from app.infrastructure.crypto import new_key_cipher
from app.infrastructure.google_client import GoogleClient
from app.infrastructure.oauth import OAuthClient, provider_configs
from app.infrastructure.openai_client import OpenAIClient
from app.infrastructure.stripe_client import StripeClient
from app.infrastructure.store_ai import AiKeyStore
from app.infrastructure.store_auth import IdempotencyStore, UserStore
from app.infrastructure.store_evidence import SourceStore
from app.domain.services.ai import AIGate, AIService
from app.domain.services.application import ApplicationService
from app.domain.services.approval import ApprovalService
from app.domain.services.auth import AuthService
from app.domain.services.billing import BillingService
from app.domain.services.calendar import CalendarService
from app.domain.services.conversation import ConversationService
from app.domain.services.document import DocumentService
from app.domain.services.evidence import EvidenceService
from app.domain.services.google import GoogleService
from app.domain.services.interview import InterviewService
from app.domain.services.job import JobService
from app.domain.services.operation import OperationService
from app.domain.services.project import ProjectService
from app.presentation.errors import (
    BodyLimitMiddleware, RequestIDMiddleware, domain_error_handler,
    http_error_handler, unhandled_error_handler, validation_error_handler,
)
from app.presentation.middleware import ApiMiddleware
from app.presentation.routers import (
    ai, applications, approvals, auth, billing, calendar, conversations,
    documents, evidence, integrations, interviews, jobs, operations, projects,
)


@dataclass
class ConfigView:
    google_configured: bool = False
    gmail_beta: bool = False
    managed_ai: bool = False
    byok_enabled: bool = False
    stripe_configured: bool = False
    job_site_adapters: dict = field(default_factory=dict)


@dataclass
class Deps:
    auth: AuthService
    evidence: EvidenceService
    jobs: JobService
    applications: ApplicationService
    documents: DocumentService
    projects: ProjectService
    interviews: InterviewService
    calendar: CalendarService
    conversations: ConversationService
    approvals: ApprovalService
    operations: OperationService
    ai: AIService
    google: GoogleService
    billing: BillingService
    sources: SourceStore
    ai_keys: AiKeyStore
    idem: IdempotencyStore
    cfg: ConfigView
    storage_dir: str
    master_key: str


def _healthz():
    return JSONResponse(content={"data": {"status": "ok"}}, status_code=200)


def create_app(cfg: Config | None = None) -> FastAPI:
    cfg = cfg or load_config()

    @asynccontextmanager
    async def lifespan(app: FastAPI):
        from langgraph.checkpoint.postgres.aio import AsyncPostgresSaver

        pool = await create_pool(cfg.database_url)
        db = DB(pool)
        migrations = Path(__file__).resolve().parent.parent / "migrations"
        if migrations.is_dir():
            await run_migrations(pool, str(migrations))
        if cfg.app_env == "development" and cfg.dev_auth_token:
            try:
                await UserStore(db).get(cfg.dev_user_id)
            except Exception:
                await UserStore(db).upsert_dev_user(cfg.dev_user_id)

        cipher = None
        if cfg.byok_master_key:
            cipher = new_key_cipher(cfg.byok_master_key)
        oauth = OAuthClient()
        openai = OpenAIClient()
        gapi = GoogleClient(cfg.google.client_id, cfg.google.client_secret)
        stripe = StripeClient(cfg.stripe_secret)
        managed_ai = bool(cfg.managed_ai_key)
        byok_enabled = cipher is not None

        gate = AIGate(db, cfg.managed_ai_key, cipher, openai)
        auths = AuthService(db, oauth, provider_configs(cfg), cfg.app_env,
                            cfg.dev_auth_token, cfg.dev_user_id)
        approvals = ApprovalService(db)
        evidence_svc = EvidenceService(db)
        applications = ApplicationService(db, approvals,
                                          cfg.job_site_adapters)
        jobs_svc = JobService(db, approvals, gate)
        documents = DocumentService(db, approvals, gate)
        projects = ProjectService(db, approvals, gate)
        interviews = InterviewService(db, gate)
        calendar = CalendarService(db)
        ai_svc = AIService(db, gate)
        operations = OperationService(db, evidence_svc)
        google = GoogleService(
            db, cipher, gapi, cfg.google.client_id, cfg.google.client_secret,
            "https://accounts.google.com/o/oauth2/v2/auth", cfg.gmail_beta)
        billing = BillingService(
            db, stripe, bool(cfg.stripe_secret),
            {"PRO": cfg.stripe_price_pro, "ULTRA": cfg.stripe_price_ultra},
            cfg.stripe_success_url, cfg.stripe_cancel_url,
            cfg.stripe_portal_url, cfg.stripe_webhook_secret)

        saver_ctx = AsyncPostgresSaver.from_conn_string(cfg.database_url)
        saver = await saver_ctx.__aenter__()
        try:
            await saver.setup()
            conversations = ConversationService(db, gate, saver)
            view = ConfigView(
                google_configured=bool(cfg.google.client_id
                                       and cfg.google.client_secret),
                gmail_beta=cfg.gmail_beta, managed_ai=managed_ai,
                byok_enabled=byok_enabled,
                stripe_configured=bool(cfg.stripe_secret),
                job_site_adapters=cfg.job_site_adapters)
            app.state.deps = Deps(
                auth=auths, evidence=evidence_svc, jobs=jobs_svc,
                applications=applications, documents=documents,
                projects=projects, interviews=interviews, calendar=calendar,
                conversations=conversations, approvals=approvals,
                operations=operations, ai=ai_svc, google=google,
                billing=billing, sources=SourceStore(db),
                ai_keys=AiKeyStore(db), idem=IdempotencyStore(db), cfg=view,
                storage_dir=cfg.storage_dir,
                master_key=cfg.byok_master_key)
            app.state.db = db
            yield
        finally:
            await saver_ctx.__aexit__(None, None, None)
            await oauth.aclose()
            await openai.aclose()
            await gapi.aclose()
            await stripe.aclose()
            await pool.close()

    app = FastAPI(lifespan=lifespan)
    app.add_exception_handler(DomainError, domain_error_handler)
    app.add_exception_handler(RequestValidationError, validation_error_handler)
    app.add_exception_handler(HTTPException, http_error_handler)
    app.add_exception_handler(Exception, unhandled_error_handler)

    app.add_middleware(ApiMiddleware)
    app.add_middleware(
        CORSMiddleware,
        allow_origins=[
            "http://localhost:5173", "http://127.0.0.1:5173",
            "tauri://localhost", "http://tauri.localhost",
            "https://tauri.localhost"],
        allow_headers=["Origin", "Content-Type", "Accept", "Authorization",
                       "Idempotency-Key", "X-Request-Id"],
        allow_methods=["GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"])
    app.add_middleware(BodyLimitMiddleware, limit=1 << 20)
    app.add_middleware(RequestIDMiddleware)

    for r in (auth.router, evidence.router, jobs.router, applications.router,
              documents.router, projects.router, interviews.router,
              conversations.router, calendar.router, approvals.router,
              operations.router, integrations.router, ai.router,
              billing.router):
        app.include_router(r, prefix="/api/v1")

    app.get("/healthz")(_healthz)
    app.get("/api/v1/healthz")(_healthz)
    return app


app = create_app()
