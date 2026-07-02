-- Actor bios, thread view counts, view dedup for members

ALTER TABLE actors
    ADD COLUMN IF NOT EXISTS bio TEXT NOT NULL DEFAULT '';

ALTER TABLE threads
    ADD COLUMN IF NOT EXISTS view_count INT NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS thread_view_dedup (
    actor_id          TEXT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
    thread_id         TEXT NOT NULL REFERENCES threads(id) ON DELETE CASCADE,
    last_counted_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (actor_id, thread_id)
);

CREATE INDEX IF NOT EXISTS thread_view_dedup_counted_idx
    ON thread_view_dedup (last_counted_at);