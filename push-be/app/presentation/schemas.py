from __future__ import annotations

from typing import Annotated, Any, Optional
from uuid import UUID

from pydantic import BaseModel, BeforeValidator, ConfigDict, Field
from pydantic.alias_generators import to_camel


def _coerce_uuid(v: Any) -> UUID:
    return v if isinstance(v, UUID) else UUID(str(v))


UUIDT = Annotated[UUID, BeforeValidator(_coerce_uuid)]


class Req(BaseModel):
    model_config = ConfigDict(extra="forbid", alias_generator=to_camel,
                              populate_by_name=False, strict=True)


class ExchangeReq(Req):
    code: str
    code_verifier: str


class RefreshReq(Req):
    refresh_token: str


class LogoutReq(Req):
    refresh_token: str


class CreateEvidenceReq(Req):
    kind: str
    title: str
    source_text: str
    source_url: Optional[str] = None
    skills: list[str] = []
    supersedes_id: Optional[UUIDT] = None


class ImportEvidenceReq(Req):
    source_id: Optional[UUIDT] = None
    text: str
    source_url: Optional[str] = None
    content_hash: str
    format: str
    kind: str
    title: str


class ExpectedRevisionReq(Req):
    expected_revision: int


class CreateJobReq(Req):
    company: str
    title: str
    source_kind: str
    source_url: Optional[str] = None
    source_text: str
    requirements: list[str] = []
    preferred: list[str] = []
    deadline: Optional[str] = None
    language: str


class PatchJobReq(Req):
    expected_revision: int
    company: Optional[str] = None
    title: Optional[str] = None
    requirements: Optional[list[str]] = None
    preferred: Optional[list[str]] = None
    deadline: Optional[str] = None


class AiOptionsReq(Req):
    provider: str
    model: str
    credential_mode: str
    effort: str


class AnalyzeReq(Req):
    application_id: UUIDT
    expected_revision: int
    evidence_ids: list[UUIDT] = []
    ai: Optional[AiOptionsReq] = None


class CreateApplicationReq(Req):
    job_id: UUIDT
    notes: str = ""


class ImportApplicationReq(Req):
    job_id: UUIDT
    stage: str
    applied_at: str
    notes: str = ""
    confirmed: bool = False


class PatchApplicationReq(Req):
    expected_revision: int
    stage: Optional[str] = None
    notes: Optional[str] = None
    next_action_at: Optional[str] = None


class CreateDraftReq(Req):
    expected_revision: int
    mode: str
    adapter: Optional[str] = None
    document_version_ids: list[UUIDT] = []
    confirmed_submitted: bool = False


class SubmitReq(Req):
    expected_revision: int
    draft_id: UUIDT
    approval_id: UUIDT


class CreateDocumentReq(Req):
    application_id: UUIDT
    title: str
    kind: str
    template: str = ""
    language: str


class PatchDocumentReq(Req):
    expected_revision: int
    title: Optional[str] = None
    template: Optional[str] = None
    language: Optional[str] = None


class EvidenceRefReq(Req):
    evidence_id: UUIDT
    start: int
    end: int


class BlockInput(Req):
    id: str
    text: str
    evidence_refs: list[EvidenceRefReq] = []


class CreateVersionReq(Req):
    expected_revision: int
    content: dict[str, Any]
    blocks: list[BlockInput] = []
    change_note: str = ""


class GenerateDocReq(Req):
    expected_revision: int
    evidence_ids: list[UUIDT] = []
    analysis_id: Optional[UUIDT] = None
    ai: Optional[AiOptionsReq] = None
    language: Optional[str] = None


class SelectionReq(Req):
    model_config = ConfigDict(extra="forbid", populate_by_name=False,
                              strict=True)
    from_: int = Field(alias="from")
    to: int
    text: str


class ReviseReq(Req):
    expected_revision: int
    version_id: UUIDT
    selection: SelectionReq
    action: str
    instruction: str = ""
    ai: Optional[AiOptionsReq] = None


class ReviewReq(Req):
    version_id: UUIDT


class FinalizeReq(Req):
    expected_revision: int
    version_id: UUIDT
    approval_id: UUIDT


class CreateExportReq(Req):
    version_id: UUIDT
    format: str
    renderer_version: str


class ExportValidationReq(Req):
    korean_text: bool = False
    links: bool = False
    ats_text: bool = False


class ExportResultReq(Req):
    sha256: str
    byte_length: int
    page_count: Optional[int] = None
    validation: ExportValidationReq = Field(default_factory=ExportValidationReq)
    status: str
    error_code: Optional[str] = None


class BlueprintsReq(Req):
    application_id: UUIDT
    gap_analysis_id: UUIDT
    ai: Optional[AiOptionsReq] = None


class CreateRunReq(Req):
    provider: str
    working_directory: str
    prompt: str


class StartRunReq(Req):
    expected_revision: int
    approval_id: UUIDT
    detected_version: str
    device_id: str


class ProcessInfoReq(Req):
    pid: int
    started_at: str


class LaunchReq(Req):
    expected_revision: int
    device_id: str
    payload_hash: str
    launch_status: str
    process: Optional[ProcessInfoReq] = None


class RecoverReq(Req):
    expected_revision: int
    device_id: str
    decision: str
    process: Optional[ProcessInfoReq] = None
    failure_reason: Optional[str] = None


class RunResultReq(Req):
    expected_revision: int
    exit_code: int
    commit_sha: Optional[str] = None
    stdout_hash: str
    stderr_hash: str


class MetricEntryReq(Req):
    name: str
    value: float
    unit: str


class CreateProjectEvidenceReq(Req):
    run_id: UUIDT
    commit_url: str
    test_command: str
    test_output: str
    exit_code: int
    metrics: list[MetricEntryReq] = []
    summary: str


class CompanySourceReq(Req):
    source_url: str
    source_text: str
    accessed_at: str


class CreateInterviewReq(Req):
    application_id: UUIDT
    title: str
    scheduled_at: str
    duration_minutes: Optional[int] = None
    evidence_ids: list[UUIDT] = []
    company_sources: list[CompanySourceReq] = []
    notes: str = ""
    time_zone: str


class PatchInterviewReq(Req):
    expected_revision: int
    title: Optional[str] = None
    scheduled_at: Optional[str] = None
    company_sources: Optional[list[CompanySourceReq]] = None
    notes: Optional[str] = None
    reflection: Optional[str] = None


class PrepareReq(Req):
    expected_revision: int
    ai: Optional[AiOptionsReq] = None


class CreateApprovalReq(Req):
    kind: str
    application_id: UUIDT
    target_id: UUIDT
    target_revision: Optional[int] = None


class DecisionReq(Req):
    expected_revision: int
    decision: str


class OpInputReq(Req):
    fields: dict[str, str] = {}
    source_id: Optional[UUIDT] = None


class PutAiKeyReq(Req):
    key: str


class AiGenerateReq(Req):
    ai: Optional[AiOptionsReq] = None
    prompt: str
    application_id: UUIDT
    evidence_ids: list[UUIDT] = []


class GoogleConnectReq(Req):
    code_challenge: str
    code_challenge_method: str
    redirect_uri: str


class GoogleCompleteReq(Req):
    integration_code: str
    code_verifier: str


class LinkMessageReq(Req):
    application_id: UUIDT


class CheckoutReq(Req):
    plan_id: str


class CreateCalendarEventReq(Req):
    application_id: Optional[UUIDT] = None
    type: str
    title: str
    starts_at: str
    ends_at: str
    time_zone: str
    notes: str = ""


class PatchCalendarEventReq(Req):
    expected_revision: int
    title: Optional[str] = None
    starts_at: Optional[str] = None
    ends_at: Optional[str] = None
    time_zone: Optional[str] = None
    notes: Optional[str] = None


class CreateConversationReq(Req):
    application_id: Optional[UUIDT] = None
    title: str


class PatchConversationReq(Req):
    expected_revision: int
    title: Optional[str] = None
    pinned: Optional[bool] = None


class MessageContextReq(Req):
    document_id: Optional[UUIDT] = None
    version_id: Optional[UUIDT] = None
    evidence_ids: list[UUIDT] = []


class PostMessageReq(Req):
    text: str
    context: Optional[MessageContextReq] = None
    ai: Optional[AiOptionsReq] = None
    access_mode: str = ""
