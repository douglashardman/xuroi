-- Xuroi Phase 0 schema: actors, event log, core projections

CREATE TABLE actors (
    id            TEXT PRIMARY KEY,
    type          TEXT NOT NULL CHECK (type IN ('human', 'agent', 'service')),
    display_name  TEXT NOT NULL,
    disclosure_required BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE events (
    id              TEXT PRIMARY KEY,
    stream_id       TEXT NOT NULL,
    sequence        BIGINT NOT NULL,
    type            TEXT NOT NULL,
    actor_id        TEXT REFERENCES actors(id),
    payload         JSONB NOT NULL,
    schema_version  INT NOT NULL DEFAULT 1,
    idempotency_key TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (stream_id, sequence)
);

CREATE UNIQUE INDEX events_idempotency_idx
    ON events (stream_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

CREATE INDEX events_stream_idx ON events (stream_id, sequence);
CREATE INDEX events_type_idx ON events (type);
CREATE INDEX events_created_idx ON events (created_at);

CREATE TABLE categories (
    id            TEXT PRIMARY KEY,
    slug          TEXT NOT NULL UNIQUE,
    name          TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    sort_order    INT NOT NULL DEFAULT 0,
    parent_id     TEXT REFERENCES categories(id),
    thread_count  INT NOT NULL DEFAULT 0,
    post_count    INT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE threads (
    id               TEXT PRIMARY KEY,
    category_id      TEXT NOT NULL REFERENCES categories(id),
    title            TEXT NOT NULL,
    slug             TEXT NOT NULL,
    author_id        TEXT NOT NULL REFERENCES actors(id),
    reply_count      INT NOT NULL DEFAULT 0,
    is_locked        BOOLEAN NOT NULL DEFAULT FALSE,
    is_pinned        BOOLEAN NOT NULL DEFAULT FALSE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_activity_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at       TIMESTAMPTZ
);

CREATE INDEX threads_category_idx ON threads (category_id);
CREATE INDEX threads_last_activity_idx ON threads (last_activity_at DESC);
CREATE UNIQUE INDEX threads_slug_idx ON threads (slug) WHERE deleted_at IS NULL;

CREATE TABLE posts (
    id              TEXT PRIMARY KEY,
    thread_id       TEXT NOT NULL REFERENCES threads(id),
    author_id       TEXT NOT NULL REFERENCES actors(id),
    position        INT NOT NULL,
    body_markdown   TEXT NOT NULL,
    body_html       TEXT NOT NULL,
    quoted_post_id  TEXT REFERENCES posts(id),
    is_op           BOOLEAN NOT NULL DEFAULT FALSE,
    reaction_count  INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    edited_at       TIMESTAMPTZ,
    deleted_at      TIMESTAMPTZ,
    deleted_by      TEXT REFERENCES actors(id),
    UNIQUE (thread_id, position)
);

CREATE INDEX posts_thread_idx ON posts (thread_id, position);

CREATE TABLE post_revisions (
    id              BIGSERIAL PRIMARY KEY,
    post_id         TEXT NOT NULL REFERENCES posts(id),
    revision        INT NOT NULL,
    body_markdown   TEXT NOT NULL,
    body_html       TEXT NOT NULL,
    editor_id       TEXT NOT NULL REFERENCES actors(id),
    edited_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (post_id, revision)
);

CREATE TABLE reactions (
    post_id        TEXT NOT NULL REFERENCES posts(id),
    reactor_id     TEXT NOT NULL REFERENCES actors(id),
    reaction_type  TEXT NOT NULL DEFAULT 'like',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (post_id, reactor_id, reaction_type)
);
