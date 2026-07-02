-- Thread intelligence projections (worker-generated summaries)

CREATE TABLE thread_intelligence (
    thread_id     TEXT PRIMARY KEY REFERENCES threads(id) ON DELETE CASCADE,
    summary       TEXT NOT NULL,
    model_version TEXT NOT NULL,
    post_count    INT NOT NULL DEFAULT 0,
    generated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX thread_intelligence_updated_idx ON thread_intelligence (updated_at DESC);