CREATE TABLE actor_emails (
    actor_id    TEXT PRIMARY KEY REFERENCES actors(id) ON DELETE CASCADE,
    email       TEXT NOT NULL UNIQUE,
    verified    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    id          TEXT PRIMARY KEY,
    actor_id    TEXT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX sessions_actor_idx ON sessions (actor_id);
CREATE INDEX sessions_expires_idx ON sessions (expires_at);