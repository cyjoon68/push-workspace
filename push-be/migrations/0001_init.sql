CREATE TABLE users (
    id uuid PRIMARY KEY,
    provider text NOT NULL,
    provider_subject text NOT NULL DEFAULT '',
    display_name text NOT NULL,
    locale text NOT NULL DEFAULT 'ko',
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider, provider_subject)
);

CREATE TABLE oauth_states (
    state text PRIMARY KEY,
    provider text NOT NULL,
    code_challenge text NOT NULL,
    redirect_uri text NOT NULL,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE exchange_codes (
    code text PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    code_challenge text NOT NULL,
    expires_at timestamptz NOT NULL,
    used_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE refresh_tokens (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    token_hash text NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE access_tokens (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    token_hash text NOT NULL UNIQUE,
    refresh_token_id uuid NOT NULL REFERENCES refresh_tokens(id),
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE idempotency_keys (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    method text NOT NULL,
    path text NOT NULL,
    key uuid NOT NULL,
    request_hash text NOT NULL,
    response_status int,
    response_body bytea,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, method, path, key)
);
CREATE INDEX idx_idem_created ON idempotency_keys (created_at);

CREATE TABLE sources (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    file_name text NOT NULL,
    mime_type text NOT NULL,
    size bigint NOT NULL,
    sha256 text NOT NULL,
    path text NOT NULL,
    status text NOT NULL DEFAULT 'STORED',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE career_evidence (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    revision bigint NOT NULL DEFAULT 1,
    kind text NOT NULL,
    title text NOT NULL,
    source_text text NOT NULL,
    source_url text,
    skills jsonb NOT NULL DEFAULT '[]',
    verification_status text NOT NULL DEFAULT 'USER_PROVIDED',
    provenance jsonb NOT NULL DEFAULT '{}',
    supersedes_id uuid,
    archived boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_evidence_list ON career_evidence (user_id, created_at DESC, id DESC);

CREATE TABLE jobs (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    revision bigint NOT NULL DEFAULT 1,
    company text NOT NULL,
    title text NOT NULL,
    source_kind text NOT NULL,
    source_url text,
    source_text text NOT NULL,
    requirements jsonb NOT NULL DEFAULT '[]',
    preferred jsonb NOT NULL DEFAULT '[]',
    keywords jsonb NOT NULL DEFAULT '[]',
    risks jsonb NOT NULL DEFAULT '[]',
    deadline date,
    language text NOT NULL DEFAULT '',
    archived boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_jobs_list ON jobs (user_id, created_at DESC, id DESC);

CREATE TABLE applications (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    revision bigint NOT NULL DEFAULT 1,
    job_id uuid NOT NULL REFERENCES jobs(id),
    company text NOT NULL,
    title text NOT NULL,
    stage text NOT NULL DEFAULT 'DISCOVERED',
    notes text NOT NULL DEFAULT '',
    applied_at timestamptz,
    next_action_at timestamptz,
    imported boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_applications_list ON applications (user_id, created_at DESC, id DESC);

CREATE TABLE application_events (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    application_id uuid NOT NULL REFERENCES applications(id),
    type text NOT NULL,
    payload jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_app_events ON application_events (application_id, created_at DESC, id DESC);

CREATE TABLE gap_analyses (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    application_id uuid NOT NULL REFERENCES applications(id),
    job_id uuid NOT NULL REFERENCES jobs(id),
    job_revision bigint NOT NULL,
    evidence_ids jsonb NOT NULL DEFAULT '[]',
    matched jsonb NOT NULL DEFAULT '[]',
    missing jsonb NOT NULL DEFAULT '[]',
    preferred_missing jsonb NOT NULL DEFAULT '[]',
    risks jsonb NOT NULL DEFAULT '[]',
    fit_score int,
    method text NOT NULL,
    stale boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_analyses_job ON gap_analyses (job_id, created_at DESC, id DESC);

CREATE TABLE documents (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    revision bigint NOT NULL DEFAULT 1,
    application_id uuid NOT NULL REFERENCES applications(id),
    title text NOT NULL,
    kind text NOT NULL,
    template text NOT NULL,
    language text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'DRAFT',
    latest_version_id uuid,
    finalized_version_id uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_documents_list ON documents (user_id, created_at DESC, id DESC);

CREATE TABLE document_versions (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    document_id uuid NOT NULL REFERENCES documents(id),
    application_id uuid NOT NULL REFERENCES applications(id),
    number int NOT NULL,
    content jsonb NOT NULL,
    blocks jsonb NOT NULL DEFAULT '[]',
    change_note text NOT NULL DEFAULT '',
    quality jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (document_id, number)
);

CREATE TABLE revision_proposals (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    document_id uuid NOT NULL REFERENCES documents(id),
    source_version_id uuid NOT NULL REFERENCES document_versions(id),
    source_revision bigint NOT NULL,
    selection jsonb NOT NULL,
    replacement text NOT NULL,
    evidence_refs jsonb NOT NULL DEFAULT '[]',
    claim_status text NOT NULL,
    applied_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE document_exports (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    document_id uuid NOT NULL REFERENCES documents(id),
    version_id uuid NOT NULL REFERENCES document_versions(id),
    format text NOT NULL,
    template text NOT NULL,
    language text NOT NULL DEFAULT '',
    content jsonb NOT NULL,
    blocks jsonb NOT NULL DEFAULT '[]',
    content_hash text NOT NULL,
    status text NOT NULL DEFAULT 'READY_TO_RENDER',
    renderer_version text NOT NULL,
    result jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE submission_drafts (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    application_id uuid NOT NULL REFERENCES applications(id),
    application_revision bigint NOT NULL,
    mode text NOT NULL,
    adapter text,
    document_version_ids jsonb NOT NULL DEFAULT '[]',
    confirmed_submitted boolean NOT NULL DEFAULT false,
    payload_hash text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE submissions (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    application_id uuid NOT NULL REFERENCES applications(id),
    mode text NOT NULL,
    adapter text,
    status text NOT NULL,
    document_version_ids jsonb NOT NULL DEFAULT '[]',
    approval_id uuid,
    receipt_url text,
    error_code text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE project_blueprints (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    revision bigint NOT NULL DEFAULT 1,
    application_id uuid NOT NULL REFERENCES applications(id),
    gap_analysis_id uuid NOT NULL REFERENCES gap_analyses(id),
    title text NOT NULL,
    skills jsonb NOT NULL DEFAULT '[]',
    problem text NOT NULL DEFAULT '',
    solution text NOT NULL DEFAULT '',
    tasks jsonb NOT NULL DEFAULT '[]',
    completion_criteria jsonb NOT NULL DEFAULT '[]',
    metrics jsonb NOT NULL DEFAULT '[]',
    estimated_effort jsonb NOT NULL DEFAULT '{}',
    state text NOT NULL DEFAULT 'DRAFT',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE cli_runs (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    revision bigint NOT NULL DEFAULT 1,
    project_id uuid NOT NULL REFERENCES project_blueprints(id),
    application_id uuid NOT NULL REFERENCES applications(id),
    provider text NOT NULL,
    working_directory text NOT NULL,
    executable text NOT NULL,
    arguments jsonb NOT NULL DEFAULT '[]',
    prompt text NOT NULL,
    payload_hash text NOT NULL,
    state text NOT NULL,
    launch_status text NOT NULL DEFAULT 'NOT_CLAIMED',
    approval_id uuid,
    device_id text,
    detected_version text,
    started_at timestamptz,
    finished_at timestamptz,
    failure_reason text,
    exit_code int,
    commit_sha text,
    stdout_hash text,
    stderr_hash text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE project_evidence (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    revision bigint NOT NULL DEFAULT 1,
    project_id uuid NOT NULL REFERENCES project_blueprints(id),
    run_id uuid NOT NULL REFERENCES cli_runs(id),
    commit_url text NOT NULL,
    commit_sha text,
    test_results jsonb NOT NULL DEFAULT '{}',
    metrics jsonb NOT NULL DEFAULT '[]',
    summary text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'PENDING',
    verification_method text,
    verified_at timestamptz,
    career_evidence_id uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE interviews (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    revision bigint NOT NULL DEFAULT 1,
    application_id uuid NOT NULL REFERENCES applications(id),
    title text NOT NULL,
    scheduled_at timestamptz NOT NULL,
    duration_minutes int,
    event_id uuid,
    evidence_ids jsonb NOT NULL DEFAULT '[]',
    company_sources jsonb NOT NULL DEFAULT '[]',
    notes text NOT NULL DEFAULT '',
    reflection text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE calendar_events (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    revision bigint NOT NULL DEFAULT 1,
    application_id uuid REFERENCES applications(id),
    type text NOT NULL,
    title text NOT NULL,
    starts_at timestamptz NOT NULL,
    ends_at timestamptz NOT NULL,
    time_zone text NOT NULL DEFAULT 'UTC',
    source text NOT NULL DEFAULT 'LOCAL',
    external_id text,
    notes text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE approvals (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    revision bigint NOT NULL DEFAULT 1,
    kind text NOT NULL,
    application_id uuid NOT NULL REFERENCES applications(id),
    target_id uuid NOT NULL,
    target_revision bigint,
    payload_hash text NOT NULL,
    status text NOT NULL DEFAULT 'PENDING',
    expires_at timestamptz,
    decided_at timestamptz,
    consumed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_approvals_list ON approvals (user_id, created_at DESC, id DESC);
CREATE INDEX idx_approvals_target ON approvals (user_id, kind, application_id, target_id, status);

CREATE TABLE operations (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    type text NOT NULL,
    application_id uuid,
    status text NOT NULL,
    progress int,
    result jsonb,
    error jsonb,
    input_request jsonb,
    pending_payload jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE ai_keys (
    user_id uuid NOT NULL REFERENCES users(id),
    provider text NOT NULL,
    last_four text NOT NULL,
    ciphertext bytea NOT NULL,
    nonce bytea NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, provider)
);
