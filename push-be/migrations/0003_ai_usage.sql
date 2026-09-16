CREATE TABLE ai_usage (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    operation_id uuid REFERENCES operations(id),
    provider text NOT NULL,
    model text NOT NULL,
    managed boolean NOT NULL,
    input_tokens int NOT NULL,
    output_tokens int NOT NULL,
    cost_micro_credits bigint NOT NULL DEFAULT 0,
    status text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_ai_usage_list ON ai_usage (user_id, created_at DESC, id DESC);
