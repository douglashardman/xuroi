-- P1 Batch 4: BST prefixes, timezone, blocks, category watch.

ALTER TABLE threads
    ADD COLUMN IF NOT EXISTS title_prefix TEXT NOT NULL DEFAULT '';

ALTER TABLE actors
    ADD COLUMN IF NOT EXISTS timezone TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS actor_blocks (
    blocker_id TEXT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
    blocked_id TEXT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (blocker_id, blocked_id),
    CHECK (blocker_id <> blocked_id)
);

CREATE INDEX IF NOT EXISTS actor_blocks_blocker_idx ON actor_blocks (blocker_id);

CREATE TABLE IF NOT EXISTS category_email_watches (
    actor_id TEXT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
    category_id TEXT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (actor_id, category_id)
);

CREATE TABLE IF NOT EXISTS email_category_queue (
    id TEXT PRIMARY KEY,
    actor_id TEXT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
    category_id TEXT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    thread_id TEXT NOT NULL REFERENCES threads(id) ON DELETE CASCADE,
    scheduled_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    attempts INT NOT NULL DEFAULT 0,
    last_error TEXT NOT NULL DEFAULT '',
    next_retry_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS email_category_queue_due_idx
    ON email_category_queue (scheduled_at, next_retry_at);