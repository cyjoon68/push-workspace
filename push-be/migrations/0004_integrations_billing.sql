ALTER TABLE users ADD COLUMN plan text NOT NULL DEFAULT 'FREE';
ALTER TABLE users ADD COLUMN subscription_status text NOT NULL DEFAULT 'NONE';
ALTER TABLE users ADD COLUMN stripe_customer_id text;
ALTER TABLE users ADD COLUMN period_ends_at timestamptz;

ALTER TABLE oauth_states ADD COLUMN user_id uuid;
ALTER TABLE oauth_states ADD COLUMN purpose text NOT NULL DEFAULT 'LOGIN';

CREATE TABLE integration_codes (
    code text PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    code_challenge text NOT NULL,
    payload jsonb NOT NULL,
    expires_at timestamptz NOT NULL,
    used_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE google_integrations (
    user_id uuid PRIMARY KEY REFERENCES users(id),
    access_ciphertext bytea NOT NULL,
    access_nonce bytea NOT NULL,
    refresh_ciphertext bytea,
    refresh_nonce bytea,
    scopes jsonb NOT NULL DEFAULT '[]',
    token_expires_at timestamptz,
    gmail_history_id text,
    calendar_sync_token text,
    last_synced_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE google_messages (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    external_id text NOT NULL,
    thread_id text NOT NULL DEFAULT '',
    application_id uuid REFERENCES applications(id),
    sender text NOT NULL DEFAULT '',
    subject text NOT NULL DEFAULT '',
    snippet text NOT NULL DEFAULT '',
    received_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, external_id)
);
CREATE INDEX idx_google_messages_list ON google_messages (user_id, created_at DESC, id DESC);

CREATE TABLE billing_ledger (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    type text NOT NULL,
    amount_micro_credits bigint NOT NULL,
    balance_after bigint NOT NULL,
    reference_id text,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_billing_ledger ON billing_ledger (user_id, created_at DESC, id DESC);

CREATE TABLE stripe_events (
    event_id text PRIMARY KEY,
    type text NOT NULL,
    processed_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_calendar_external ON calendar_events (user_id, external_id) WHERE external_id IS NOT NULL;
