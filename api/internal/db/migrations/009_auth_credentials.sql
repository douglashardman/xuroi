CREATE TABLE actor_passwords (
    actor_id       TEXT PRIMARY KEY REFERENCES actors(id) ON DELETE CASCADE,
    password_hash  TEXT NOT NULL,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE webauthn_credentials (
    id             TEXT PRIMARY KEY,
    actor_id       TEXT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
    credential_id  BYTEA NOT NULL UNIQUE,
    credential_json JSONB NOT NULL,
    device_name    TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at   TIMESTAMPTZ
);

CREATE INDEX webauthn_credentials_actor_idx ON webauthn_credentials (actor_id);

CREATE TABLE webauthn_sessions (
    id            TEXT PRIMARY KEY,
    kind          TEXT NOT NULL CHECK (kind IN ('signup', 'registration', 'login')),
    actor_id      TEXT REFERENCES actors(id) ON DELETE CASCADE,
    email         TEXT,
    display_name  TEXT,
    session_data  JSONB NOT NULL,
    expires_at    TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX webauthn_sessions_expires_idx ON webauthn_sessions (expires_at);