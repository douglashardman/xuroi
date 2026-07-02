-- Email bans + queue retry metadata (I7, E8).

CREATE TABLE email_bans (
    id           TEXT PRIMARY KEY,
    email        TEXT NOT NULL,
    actor_id     TEXT REFERENCES actors(id) ON DELETE SET NULL,
    reason       TEXT NOT NULL DEFAULT '',
    banned_until TIMESTAMPTZ,
    banned_by    TEXT REFERENCES actors(id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX email_bans_email_lower_uniq ON email_bans (lower(email));
CREATE INDEX email_bans_active_idx ON email_bans (lower(email))
    WHERE banned_until IS NULL OR banned_until > now();

ALTER TABLE email_mention_queue
    ADD COLUMN IF NOT EXISTS attempts INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS last_error TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS next_retry_at TIMESTAMPTZ;

ALTER TABLE email_notification_queue
    ADD COLUMN IF NOT EXISTS attempts INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS last_error TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS next_retry_at TIMESTAMPTZ;