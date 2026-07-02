CREATE TABLE IF NOT EXISTS thread_reports (
    id          TEXT PRIMARY KEY,
    thread_id   TEXT NOT NULL REFERENCES threads(id),
    reporter_id TEXT NOT NULL REFERENCES actors(id),
    reason      TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ,
    resolved_by TEXT REFERENCES actors(id),
    UNIQUE (thread_id, reporter_id)
);

CREATE INDEX IF NOT EXISTS idx_thread_reports_thread
    ON thread_reports (thread_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_thread_reports_open
    ON thread_reports (created_at DESC)
    WHERE resolved_at IS NULL;