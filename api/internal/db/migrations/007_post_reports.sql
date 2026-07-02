CREATE TABLE IF NOT EXISTS post_reports (
    id          TEXT PRIMARY KEY,
    post_id     TEXT NOT NULL REFERENCES posts(id),
    thread_id   TEXT NOT NULL REFERENCES threads(id),
    reporter_id TEXT NOT NULL REFERENCES actors(id),
    reason      TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (post_id, reporter_id)
);

CREATE INDEX IF NOT EXISTS idx_post_reports_thread ON post_reports(thread_id, created_at DESC);