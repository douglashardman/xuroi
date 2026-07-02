CREATE TABLE auth_tokens (
    id          TEXT PRIMARY KEY,
    purpose     TEXT NOT NULL CHECK (purpose IN ('password_reset', 'magic_link')),
    token_hash  TEXT NOT NULL UNIQUE,
    actor_id    TEXT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
    email       TEXT NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX auth_tokens_actor_purpose_idx ON auth_tokens (actor_id, purpose);
CREATE INDEX auth_tokens_expires_idx ON auth_tokens (expires_at);