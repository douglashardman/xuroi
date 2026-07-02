ALTER TABLE post_reports
    ADD COLUMN IF NOT EXISTS resolved_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS resolved_by TEXT REFERENCES actors(id);

CREATE INDEX IF NOT EXISTS idx_post_reports_open
    ON post_reports (created_at DESC)
    WHERE resolved_at IS NULL;