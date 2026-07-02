-- Thread read tracking + debounced reply notification queue

CREATE TABLE thread_reads (
    actor_id      TEXT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
    thread_id     TEXT NOT NULL REFERENCES threads(id) ON DELETE CASCADE,
    last_read_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (actor_id, thread_id)
);

CREATE INDEX thread_reads_thread_idx ON thread_reads (thread_id);

CREATE TABLE email_preferences (
    actor_id                 TEXT PRIMARY KEY REFERENCES actors(id) ON DELETE CASCADE,
    thread_replies_enabled   BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- One pending digest per participant per thread (debounced before send)
CREATE TABLE email_notification_queue (
    id            TEXT PRIMARY KEY,
    actor_id      TEXT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
    thread_id     TEXT NOT NULL REFERENCES threads(id) ON DELETE CASCADE,
    last_post_id  TEXT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    scheduled_at  TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (actor_id, thread_id)
);

CREATE INDEX email_notification_queue_scheduled_idx
    ON email_notification_queue (scheduled_at);