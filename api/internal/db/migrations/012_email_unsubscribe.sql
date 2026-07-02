CREATE TABLE email_thread_mutes (
    actor_id   TEXT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
    thread_id  TEXT NOT NULL REFERENCES threads(id) ON DELETE CASCADE,
    muted_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (actor_id, thread_id)
);

CREATE INDEX email_thread_mutes_thread_idx ON email_thread_mutes (thread_id);

ALTER TABLE auth_tokens ADD COLUMN IF NOT EXISTS thread_id TEXT REFERENCES threads(id) ON DELETE CASCADE;

ALTER TABLE auth_tokens DROP CONSTRAINT IF EXISTS auth_tokens_purpose_check;
ALTER TABLE auth_tokens ADD CONSTRAINT auth_tokens_purpose_check
    CHECK (purpose IN ('password_reset', 'magic_link', 'thread_unsubscribe'));