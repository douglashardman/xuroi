-- In-app notifications + @mention tracking

CREATE TABLE notifications (
    id            TEXT PRIMARY KEY,
    actor_id      TEXT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
    type          TEXT NOT NULL,
    from_actor_id TEXT REFERENCES actors(id) ON DELETE SET NULL,
    post_id       TEXT REFERENCES posts(id) ON DELETE CASCADE,
    thread_id     TEXT REFERENCES threads(id) ON DELETE CASCADE,
    title         TEXT NOT NULL DEFAULT '',
    body          TEXT NOT NULL DEFAULT '',
    url           TEXT NOT NULL DEFAULT '',
    read_at       TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX notifications_actor_created_idx
    ON notifications (actor_id, created_at DESC);

CREATE INDEX notifications_actor_unread_idx
    ON notifications (actor_id, created_at DESC)
    WHERE read_at IS NULL;

CREATE TABLE post_mentions (
    post_id   TEXT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    actor_id  TEXT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
    PRIMARY KEY (post_id, actor_id)
);

CREATE INDEX post_mentions_actor_idx ON post_mentions (actor_id);

ALTER TABLE email_preferences
    ADD COLUMN IF NOT EXISTS mentions_enabled BOOLEAN NOT NULL DEFAULT TRUE;

CREATE TABLE email_mention_queue (
    id           TEXT PRIMARY KEY,
    actor_id     TEXT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
    post_id      TEXT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    scheduled_at TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (actor_id, post_id)
);

CREATE INDEX email_mention_queue_scheduled_idx
    ON email_mention_queue (scheduled_at);