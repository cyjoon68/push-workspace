from __future__ import annotations
from datetime import datetime, timedelta
from typing import Any, ClassVar, Optional
from uuid import UUID

from pydantic import BaseModel, ConfigDict, Field, model_serializer
from pydantic.alias_generators import to_camel

ACCESS_TOKEN_TTL_SECONDS = 900
REFRESH_TOKEN_TTL = timedelta(days=30)
OAUTH_STATE_TTL = timedelta(minutes=10)
EXCHANGE_CODE_TTL = timedelta(seconds=60)
INTEGRATION_CODE_TTL = timedelta(seconds=60)
AUTH_CALLBACK_URI = "push://auth/callback"
GOOGLE_CALLBACK_URI = "push://integrations/google/callback"
OAUTH_PURPOSE_LOGIN = "LOGIN"
OAUTH_PURPOSE_GOOGLE = "GOOGLE"

DEFAULT_APPROVAL_TTL = timedelta(hours=24)

PLAN_FREE = "FREE"
PLAN_PRO = "PRO"
PLAN_ULTRA = "ULTRA"
SUB_NONE = "NONE"
SUB_ACTIVE = "ACTIVE"
SUB_CANCELED = "CANCELED"
LEDGER_PURCHASE = "PURCHASE"
LEDGER_REFUND = "REFUND"
PLAN_CREDITS_MICRO = {PLAN_PRO: 2_000_000, PLAN_ULTRA: 10_000_000}

GOOGLE_SCOPE_GMAIL = "https://www.googleapis.com/auth/gmail.readonly"
GOOGLE_SCOPE_CALENDAR = "https://www.googleapis.com/auth/calendar.events"

EVENT_STAGE_CHANGED = "STAGE_CHANGED"
EVENT_IMPORTED = "IMPORTED"
EVENT_SUBMISSION = "SUBMISSION"
EVENT_DOCUMENT_FINAL = "DOCUMENT_FINALIZED"
EVENT_APPROVAL = "APPROVAL"
EVENT_INTERVIEW = "INTERVIEW"


class Model(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)


class OmitModel(Model):
    _omit_none: ClassVar[frozenset] = frozenset()

    @model_serializer(mode="wrap")
    def _ser(self, handler):
        d = handler(self)
        aliases = {n: (f.alias or n) for n, f in type(self).model_fields.items()}
        drop = {aliases[n] for n in self._omit_none} | set(self._omit_none)
        return {k: v for k, v in d.items() if v is not None or k not in drop}


class User(Model):
    id: UUID
    provider: str = Field(exclude=True, default="")
    provider_subject: str = Field(exclude=True, default="")
    display_name: str
    locale: str = "ko"
    plan: str = Field(exclude=True, default="FREE")
    subscription_status: str = Field(exclude=True, default="NONE")
    stripe_customer_id: Optional[str] = Field(exclude=True, default=None)
    period_ends_at: Optional[datetime] = Field(exclude=True, default=None)
    created_at: datetime


class OAuthState(BaseModel):
    state: str
    provider: str
    purpose: str = "LOGIN"
    user_id: Optional[UUID] = None
    code_challenge: str
    redirect_uri: str
    expires_at: datetime
    created_at: datetime


class ExchangeCode(BaseModel):
    code: str
    user_id: UUID
    code_challenge: str
    expires_at: datetime
    used_at: Optional[datetime] = None
    created_at: datetime


class AccessToken(BaseModel):
    id: UUID
    user_id: UUID
    token_hash: str
    refresh_token_id: UUID
    expires_at: datetime
    created_at: datetime


class RefreshToken(BaseModel):
    id: UUID
    user_id: UUID
    token_hash: str
    expires_at: datetime
    revoked_at: Optional[datetime] = None
    created_at: datetime


class Session(BaseModel):
    access_token: str
    refresh_token: str
    expires_in: int
    user: User


def valid_oauth_provider(p: str) -> bool:
    return p in ("google", "github")


class IdempotencyRecord(BaseModel):
    id: UUID
    user_id: UUID
    key: UUID
    method: str
    path: str
    request_hash: str
    response_status: Optional[int] = None
    response_body: Optional[bytes] = None
    created_at: datetime


EVIDENCE_KINDS = {"RESUME", "GITHUB", "CAREER", "EDUCATION", "SKILL", "PROJECT"}
VERIFICATION_USER_PROVIDED = "USER_PROVIDED"
VERIFICATION_PENDING = "PENDING"
VERIFICATION_VERIFIED = "VERIFIED"
VERIFICATION_REJECTED = "REJECTED"


def valid_evidence_kind(k: str) -> bool:
    return k in EVIDENCE_KINDS


class SourceLocation(Model):
    start: int
    end: int
    unit: str


class Provenance(Model):
    source_id: Optional[UUID] = None
    project_evidence_id: Optional[UUID] = None
    content_hash: str = ""
    source_location: Optional[SourceLocation] = None


class CareerEvidence(Model):
    id: UUID
    user_id: UUID = Field(exclude=True)
    revision: int = 1
    kind: str
    title: str
    source_text: str
    source_url: Optional[str] = None
    skills: list[str] = []
    verification_status: str = "USER_PROVIDED"
    provenance: Provenance = Field(default_factory=Provenance)
    supersedes_id: Optional[UUID] = None
    archived: bool = False
    created_at: datetime
    updated_at: datetime


class Source(BaseModel):
    id: UUID
    user_id: UUID
    file_name: str
    mime_type: str
    size: int
    sha256: str
    path: str
    status: str = "STORED"
    created_at: datetime


JOB_SOURCE_KINDS = {"URL", "TEXT", "DOM"}
METHOD_RULE_BASED = "RULE_BASED"
METHOD_AI_ASSISTED = "AI_ASSISTED"


def valid_job_source_kind(k: str) -> bool:
    return k in JOB_SOURCE_KINDS


class JobPosting(Model):
    id: UUID
    user_id: UUID = Field(exclude=True)
    revision: int = 1
    company: str
    title: str
    source_kind: str
    source_url: Optional[str] = None
    source_text: str
    requirements: list[str] = []
    preferred: list[str] = []
    keywords: list[str] = []
    risks: list[str] = []
    deadline: Optional[datetime] = None
    language: str = ""
    archived: bool = False
    created_at: datetime
    updated_at: datetime


class RequirementMatch(Model):
    requirement: str
    evidence_ids: list[str]


class GapAnalysis(Model):
    id: UUID
    user_id: UUID = Field(exclude=True)
    application_id: UUID
    job_id: UUID
    job_revision: int
    evidence_ids: list[UUID] = []
    matched: list[RequirementMatch] = []
    missing: list[str] = []
    preferred_missing: list[str] = []
    risks: list[str] = []
    fit_score: Optional[int] = None
    method: str
    stale: bool = False
    created_at: datetime


STAGE_ORDER = ["DISCOVERED", "PREPARING", "READY", "APPLIED", "SCREENING", "INTERVIEW", "OFFER", "ACCEPTED"]
STAGE_REJECTED = "REJECTED"
STAGE_WITHDRAWN = "WITHDRAWN"
STAGE_APPLIED = "APPLIED"
TERMINAL_STAGES = {"ACCEPTED", "REJECTED", "WITHDRAWN"}


def _stage_index(s: str) -> int:
    try:
        return STAGE_ORDER.index(s)
    except ValueError:
        return -1


def valid_stage(s: str) -> bool:
    return _stage_index(s) >= 0 or s in ("REJECTED", "WITHDRAWN")


def terminal_stage(s: str) -> bool:
    return s in TERMINAL_STAGES


def can_patch_stage(frm: str, to: str) -> bool:
    if frm == to:
        return True
    if terminal_stage(frm):
        return False
    if to in ("REJECTED", "WITHDRAWN"):
        return True
    if to == "APPLIED":
        return False
    return _stage_index(to) == _stage_index(frm) + 1


def next_stage(s: str) -> str | None:
    i = _stage_index(s)
    if i < 0 or i + 1 >= len(STAGE_ORDER):
        return None
    return STAGE_ORDER[i + 1]


def stage_after_applied(s: str) -> bool:
    return s in ("SCREENING", "INTERVIEW", "OFFER")


class Application(Model):
    id: UUID
    user_id: UUID = Field(exclude=True)
    revision: int = 1
    job_id: UUID
    company: str
    title: str
    stage: str = "DISCOVERED"
    notes: str = ""
    applied_at: Optional[datetime] = None
    next_action_at: Optional[datetime] = None
    imported: bool = False
    created_at: datetime
    updated_at: datetime


class ApplicationEvent(Model):
    id: UUID
    user_id: UUID = Field(exclude=True)
    application_id: UUID
    type: str
    payload: dict = {}
    created_at: datetime


class SubmissionDraft(Model):
    id: UUID
    user_id: UUID = Field(exclude=True)
    application_id: UUID
    application_revision: int
    mode: str
    adapter: Optional[str] = None
    document_version_ids: list[UUID] = []
    confirmed_submitted: bool = False
    payload_hash: str
    created_at: datetime


class Submission(Model):
    id: UUID
    user_id: UUID = Field(exclude=True)
    application_id: UUID
    mode: str
    adapter: Optional[str] = None
    status: str
    document_version_ids: list[UUID] = []
    approval_id: Optional[UUID] = None
    receipt_url: Optional[str] = None
    error_code: Optional[str] = None
    created_at: datetime
    updated_at: datetime


DOC_KINDS = {"RESUME", "PORTFOLIO", "COVER_LETTER"}
DOC_TEMPLATES = {"CLASSIC", "MODERN", "COMPACT"}
DOC_DRAFT = "DRAFT"
DOC_FINALIZED = "FINALIZED"
DOC_ARCHIVED = "ARCHIVED"
CLAIM_SUPPORTED = "SUPPORTED"
CLAIM_NEEDS_REVIEW = "NEEDS_REVIEW"
CLAIM_UNSUPPORTED = "UNSUPPORTED"
REVISION_ACTIONS = {"REWRITE", "SHORTEN", "EMPHASIZE_METRICS", "CHANGE_TONE", "TAILOR_TO_JOB"}


def valid_document_kind(k: str) -> bool:
    return k in DOC_KINDS


def valid_document_template(t: str) -> bool:
    return t in DOC_TEMPLATES


def valid_revision_action(a: str) -> bool:
    return a in REVISION_ACTIONS


class Document(Model):
    id: UUID
    user_id: UUID = Field(exclude=True)
    revision: int = 1
    application_id: UUID
    title: str
    kind: str
    template: str
    language: str = ""
    status: str = "DRAFT"
    latest_version_id: Optional[UUID] = None
    finalized_version_id: Optional[UUID] = None
    created_at: datetime
    updated_at: datetime


class EvidenceRef(Model):
    evidence_id: UUID
    start: int
    end: int


class Block(Model):
    id: str
    text: str
    evidence_refs: list[EvidenceRef] = []
    claim_status: str = ""


class QualityIssue(OmitModel):
    _omit_none: ClassVar[frozenset] = frozenset({"block_id"})
    code: str
    severity: str
    block_id: Optional[str] = None
    message: str


class Quality(Model):
    job_fit: Optional[int] = None
    evidence_fidelity: Optional[int] = None
    readability: Optional[int] = None
    ats: Optional[int] = None
    method: str = "RULE_BASED"
    issues: list[QualityIssue] = []


class DocumentVersion(Model):
    id: UUID
    user_id: UUID = Field(exclude=True)
    document_id: UUID
    application_id: UUID
    number: int
    content: dict
    blocks: list[Block] = []
    change_note: str = ""
    quality: Quality = Field(default_factory=Quality)
    created_at: datetime


class Selection(Model):
    from_: int = Field(alias="from")
    to: int
    text: str


class RevisionProposal(Model):
    id: UUID
    user_id: UUID = Field(exclude=True)
    document_id: UUID
    source_version_id: UUID
    source_revision: int
    selection: Selection
    replacement: str
    evidence_refs: list[EvidenceRef] = []
    claim_status: str = ""
    applied_at: Optional[datetime] = None
    created_at: datetime


class ExportValidation(Model):
    korean_text: bool
    links: bool
    ats_text: bool


class ExportResult(OmitModel):
    _omit_none: ClassVar[frozenset] = frozenset({"page_count", "error_code"})
    sha256: str
    byte_length: int
    page_count: Optional[int] = None
    validation: ExportValidation
    error_code: Optional[str] = None


class DocumentExport(Model):
    id: UUID
    user_id: UUID = Field(exclude=True)
    document_id: UUID
    version_id: UUID
    format: str
    template: str
    language: str = ""
    content: dict
    blocks: list[Block] = []
    content_hash: str
    status: str = "READY_TO_RENDER"
    renderer_version: str
    result: Optional[ExportResult] = None
    created_at: datetime
    updated_at: datetime


CLI_PROVIDERS = {"CODEX", "CLAUDE_CODE", "GROK_BUILD"}
CLI_EXECUTABLES = {"CODEX": "codex", "CLAUDE_CODE": "claude", "GROK_BUILD": "grok"}
BLUEPRINT_DRAFT = "DRAFT"
BLUEPRINT_SELECTED = "SELECTED"
BLUEPRINT_IN_PROGRESS = "IN_PROGRESS"
BLUEPRINT_VERIFIED = "VERIFIED"
BLUEPRINT_ARCHIVED = "ARCHIVED"
RUN_DRAFT = "DRAFT"
RUN_APPROVAL_REQUIRED = "APPROVAL_REQUIRED"
RUN_RUNNING = "RUNNING"
RUN_VERIFYING = "VERIFYING"
RUN_VERIFIED = "VERIFIED"
RUN_FAILED = "FAILED"
LAUNCH_NOT_CLAIMED = "NOT_CLAIMED"
LAUNCH_CLAIMED = "CLAIMED"
LAUNCH_STARTED = "STARTED"
LAUNCH_UNKNOWN = "UNKNOWN"
LAUNCH_FINISHED = "FINISHED"


def valid_cli_provider(p: str) -> bool:
    return p in CLI_PROVIDERS


def valid_reported_launch_status(s: str) -> bool:
    return s in (LAUNCH_CLAIMED, LAUNCH_STARTED, LAUNCH_UNKNOWN)


def can_transition_launch(frm: str, to: str) -> bool:
    if frm == to:
        return True
    if frm == LAUNCH_NOT_CLAIMED:
        return to == LAUNCH_CLAIMED
    if frm == LAUNCH_CLAIMED:
        return to in (LAUNCH_STARTED, LAUNCH_UNKNOWN)
    if frm == LAUNCH_STARTED:
        return to == LAUNCH_UNKNOWN
    return False


class BlueprintTask(Model):
    id: str
    title: str
    description: str
    acceptance: list[str] = []


class BlueprintMetric(Model):
    name: str
    unit: str
    measurement: str
    target: Optional[float] = None


class EffortEstimate(Model):
    min_hours: int
    max_hours: int


class ProjectBlueprint(Model):
    id: UUID
    user_id: UUID = Field(exclude=True)
    revision: int = 1
    application_id: UUID
    gap_analysis_id: UUID
    title: str
    skills: list[str] = []
    problem: str = ""
    solution: str = ""
    tasks: list[BlueprintTask] = []
    completion_criteria: list[str] = []
    metrics: list[BlueprintMetric] = []
    estimated_effort: EffortEstimate = Field(default_factory=lambda: EffortEstimate(min_hours=0, max_hours=0))
    state: str = "DRAFT"
    created_at: datetime
    updated_at: datetime


class ProcessInfo(Model):
    pid: int
    started_at: datetime


class CliRun(Model):
    id: UUID
    user_id: UUID = Field(exclude=True)
    revision: int = 1
    project_id: UUID
    application_id: UUID
    provider: str
    working_directory: str
    executable: str
    arguments: list[str] = []
    prompt: str
    payload_hash: str
    state: str
    launch_status: str = "NOT_CLAIMED"
    approval_id: Optional[UUID] = None
    device_id: Optional[str] = None
    detected_version: Optional[str] = None
    started_at: Optional[datetime] = None
    finished_at: Optional[datetime] = None
    failure_reason: Optional[str] = None
    exit_code: Optional[int] = None
    commit_sha: Optional[str] = None
    stdout_hash: Optional[str] = None
    stderr_hash: Optional[str] = None
    created_at: datetime
    updated_at: datetime


class TestResults(Model):
    test_command: str = ""
    test_output: str = ""
    exit_code: int = 0


class MetricEntry(Model):
    name: str
    value: float
    unit: str


class ProjectEvidence(Model):
    id: UUID
    user_id: UUID = Field(exclude=True)
    revision: int = 1
    project_id: UUID
    run_id: UUID
    commit_url: str
    commit_sha: Optional[str] = None
    test_results: TestResults = Field(default_factory=TestResults)
    metrics: list[MetricEntry] = []
    summary: str = ""
    status: str = "PENDING"
    verification_method: Optional[str] = None
    verified_at: Optional[datetime] = None
    career_evidence_id: Optional[UUID] = None
    created_at: datetime
    updated_at: datetime


class CompanySource(Model):
    source_url: str
    source_text: str
    accessed_at: datetime


class InterviewSession(Model):
    id: UUID
    user_id: UUID = Field(exclude=True)
    revision: int = 1
    application_id: UUID
    title: str
    scheduled_at: datetime
    duration_minutes: Optional[int] = None
    event_id: Optional[UUID] = None
    evidence_ids: list[UUID] = []
    company_sources: list[CompanySource] = []
    notes: str = ""
    reflection: str = ""
    created_at: datetime
    updated_at: datetime


CAL_EVENT_TYPES = {"INTERVIEW", "DEADLINE", "FOLLOW_UP", "CUSTOM"}
EVENT_SOURCE_LOCAL = "LOCAL"
EVENT_SOURCE_GOOGLE = "GOOGLE"


def valid_event_type(t: str) -> bool:
    return t in CAL_EVENT_TYPES


class CalendarEvent(Model):
    id: UUID
    user_id: UUID = Field(exclude=True)
    revision: int = 1
    application_id: Optional[UUID] = None
    type: str
    title: str
    starts_at: datetime
    ends_at: datetime
    time_zone: str = "UTC"
    source: str = "LOCAL"
    external_id: Optional[str] = None
    notes: str = ""
    created_at: datetime
    updated_at: datetime


class Conversation(Model):
    id: UUID
    user_id: UUID = Field(exclude=True)
    revision: int = 1
    application_id: Optional[UUID] = None
    title: str = ""
    pinned: bool = False
    archived: bool = Field(default=False, exclude=True)
    created_at: datetime
    updated_at: datetime


class MessageAttachment(OmitModel):
    _omit_none: ClassVar[frozenset] = frozenset({"document_id", "title"})
    type: str
    id: UUID
    document_id: Optional[UUID] = None
    title: Optional[str] = None


class Message(Model):
    id: UUID
    user_id: UUID = Field(exclude=True)
    conversation_id: UUID
    role: str
    text: str
    attachments: list[MessageAttachment] = []
    operation_id: Optional[UUID] = None
    created_at: datetime


ACCESS_MODES = {"SUGGEST", "CONFIRM_ACTIONS"}


def valid_access_mode(m: str) -> bool:
    return m in ACCESS_MODES


APPROVAL_KINDS = {"EVIDENCE_USE", "DOCUMENT_FINALIZE", "APPLICATION_SUBMIT", "CLI_EXECUTE"}
APPROVAL_EVIDENCE_USE = "EVIDENCE_USE"
APPROVAL_DOCUMENT_FINALIZE = "DOCUMENT_FINALIZE"
APPROVAL_APPLICATION_SUBMIT = "APPLICATION_SUBMIT"
APPROVAL_CLI_EXECUTE = "CLI_EXECUTE"
APPROVAL_PENDING = "PENDING"
APPROVAL_APPROVED = "APPROVED"
APPROVAL_DENIED = "DENIED"
APPROVAL_EXPIRED = "EXPIRED"
APPROVAL_CONSUMED = "CONSUMED"


def valid_approval_kind(k: str) -> bool:
    return k in APPROVAL_KINDS


def one_shot_approval(k: str) -> bool:
    return k != APPROVAL_EVIDENCE_USE


class Approval(Model):
    id: UUID
    user_id: UUID = Field(exclude=True)
    revision: int = 1
    kind: str
    application_id: UUID
    target_id: UUID
    target_revision: Optional[int] = None
    payload_hash: str
    status: str = "PENDING"
    expires_at: Optional[datetime] = None
    decided_at: Optional[datetime] = None
    consumed_at: Optional[datetime] = None
    created_at: datetime
    updated_at: datetime

    def expired(self, now: datetime) -> bool:
        return self.expires_at is not None and now > self.expires_at


OP_EVIDENCE_IMPORT = "EVIDENCE_IMPORT"
OP_JOB_ANALYSIS = "JOB_ANALYSIS"
OP_DOCUMENT_GENERATE = "DOCUMENT_GENERATE"
OP_DOCUMENT_REVISE = "DOCUMENT_REVISE"
OP_PROJECT_BLUEPRINTS = "PROJECT_BLUEPRINTS"
OP_PROJECT_VERIFY = "PROJECT_VERIFY"
OP_INTERVIEW_PREPARE = "INTERVIEW_PREPARE"
OP_AI_GENERATE = "AI_GENERATE"
OP_CHAT_MESSAGE = "CHAT_MESSAGE"
OP_GOOGLE_SYNC = "GOOGLE_SYNC"
OP_APPLICATION_SUBMIT = "APPLICATION_SUBMIT"
OP_QUEUED = "QUEUED"
OP_RUNNING = "RUNNING"
OP_NEEDS_INPUT = "NEEDS_INPUT"
OP_SUCCEEDED = "SUCCEEDED"
OP_FAILED = "FAILED"
OP_CANCELLED = "CANCELLED"
TERMINAL_OP_STATUSES = {OP_SUCCEEDED, OP_FAILED, OP_CANCELLED}


def terminal_operation_status(s: str) -> bool:
    return s in TERMINAL_OP_STATUSES


class InputField(Model):
    name: str
    label: str
    type: str


class InputRequest(Model):
    code: str
    message: str
    fields: list[InputField]


class OperationError(Model):
    code: str
    message: str
    retryable: bool


class OperationResult(Model):
    kind: str
    value: Any


class Operation(Model):
    id: UUID
    user_id: UUID = Field(exclude=True)
    type: str
    application_id: Optional[UUID] = None
    status: str
    progress: Optional[int] = None
    result: Optional[OperationResult] = None
    error: Optional[OperationError] = None
    input_request: Optional[InputRequest] = None
    pending_payload: Any = Field(default=None, exclude=True)
    created_at: datetime
    updated_at: datetime


class AiKey(BaseModel):
    user_id: UUID
    provider: str
    last_four: str
    ciphertext: bytes
    nonce: bytes
    updated_at: datetime


AI_PROVIDERS = {"OPENAI", "CLAUDE", "GEMINI", "GROK"}
CREDENTIAL_MODES = {"MANAGED", "BYOK"}
EFFORTS = {"LOW", "MEDIUM", "HIGH"}


def valid_ai_provider(p: str) -> bool:
    return p in AI_PROVIDERS


def valid_credential_mode(m: str) -> bool:
    return m in CREDENTIAL_MODES


def valid_effort(e: str) -> bool:
    return e in EFFORTS


class AiOptions(Model):
    provider: str
    model: str
    credential_mode: str
    effort: str


class AIModelInfo(BaseModel):
    provider: str
    model: str
    label: str


OPENAI_MODELS = [
    AIModelInfo(provider="OPENAI", model="gpt-4o-mini", label="GPT-4o mini"),
    AIModelInfo(provider="OPENAI", model="gpt-4o", label="GPT-4o"),
]

_OPENAI_MICRO_PER_1M = {
    "gpt-4o-mini": (150_000, 600_000),
    "gpt-4o": (2_500_000, 10_000_000),
}


def ai_cost_micro_credits(provider: str, model: str, in_tokens: int, out_tokens: int) -> int:
    if provider != "OPENAI":
        return 0
    rates = _OPENAI_MICRO_PER_1M.get(model, _OPENAI_MICRO_PER_1M["gpt-4o-mini"])
    return (in_tokens * rates[0] + out_tokens * rates[1]) // 1_000_000


USAGE_RESERVED = "RESERVED"
USAGE_SETTLED = "SETTLED"
USAGE_RELEASED = "RELEASED"


class AiUsage(Model):
    id: UUID
    user_id: UUID = Field(exclude=True)
    operation_id: Optional[UUID] = None
    provider: str
    model: str
    managed: bool
    input_tokens: int
    output_tokens: int
    cost_micro_credits: int
    status: str
    created_at: datetime


class AICompletion(BaseModel):
    text: str
    input_tokens: int
    output_tokens: int


class IntegrationCode(BaseModel):
    code: str
    user_id: UUID
    code_challenge: str
    payload: dict
    expires_at: datetime
    used_at: Optional[datetime] = None
    created_at: datetime


class GoogleIntegration(BaseModel):
    user_id: UUID
    access_ciphertext: bytes = b""
    access_nonce: bytes = b""
    refresh_ciphertext: Optional[bytes] = None
    refresh_nonce: Optional[bytes] = None
    scopes: list[str] = []
    token_expires_at: Optional[datetime] = None
    gmail_history_id: Optional[str] = None
    calendar_sync_token: Optional[str] = None
    last_synced_at: Optional[datetime] = None
    created_at: datetime
    updated_at: datetime

    def has_scope(self, scope: str) -> bool:
        return scope in self.scopes


class GoogleMessage(Model):
    id: UUID
    user_id: UUID = Field(exclude=True)
    external_id: str = Field(exclude=True, default="")
    thread_id: str = ""
    application_id: Optional[UUID] = None
    sender: str = ""
    subject: str = ""
    snippet: str = ""
    received_at: Optional[datetime] = None
    created_at: datetime


class GoogleTokens(BaseModel):
    access_token: str
    refresh_token: str = ""
    expires_in: int = 0
    scope: str = ""


class GoogleMessageMeta(BaseModel):
    id: str
    thread_id: str = ""
    from_: str = ""
    subject: str = ""
    snippet: str = ""
    received_at: Optional[datetime] = None


class GoogleEvent(BaseModel):
    id: str
    status: str = ""
    summary: str = ""
    starts_at: datetime
    ends_at: datetime
    time_zone: str = ""


class GoogleStatus(Model):
    enabled: bool
    connected: bool
    scopes: list[str] = []
    gmail_status: str = ""
    calendar_status: str = ""
    last_synced_at: Optional[datetime] = None


class LedgerEntry(Model):
    id: UUID
    user_id: UUID = Field(exclude=True)
    type: str
    amount_micro_credits: int
    balance_after: int
    reference_id: Optional[str] = None
    created_at: datetime


class BillingStatus(Model):
    subscription_status: str
    plan: Optional[str] = None
    period_ends_at: Optional[datetime] = None
    balance_micro_credits: int
    reserved_micro_credits: int = 0


class StripeCheckout(BaseModel):
    url: str
    expires_at: Optional[datetime] = None


def valid_plan(p: str) -> bool:
    return p in (PLAN_FREE, PLAN_PRO, PLAN_ULTRA)
