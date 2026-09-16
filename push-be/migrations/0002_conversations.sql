CREATE TABLE conversations (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    revision bigint NOT NULL DEFAULT 1,
    application_id uuid REFERENCES applications(id),
    title text NOT NULL DEFAULT '',
    pinned boolean NOT NULL DEFAULT false,
    archived boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_conversations_list ON conversations (user_id, created_at DESC, id DESC);

CREATE TABLE messages (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    conversation_id uuid NOT NULL REFERENCES conversations(id),
    role text NOT NULL,
    text text NOT NULL,
    attachments jsonb NOT NULL DEFAULT '[]',
    operation_id uuid,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_messages_list ON messages (conversation_id, created_at DESC, id DESC);
