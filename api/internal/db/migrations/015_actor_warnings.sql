-- Member warnings: 8h visible overlay; 3 lifetime → auto 7-day ban

CREATE TABLE actor_warnings (
    id          TEXT PRIMARY KEY,
    actor_id    TEXT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
    message     TEXT NOT NULL,
    warned_by   TEXT NOT NULL REFERENCES actors(id),
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX actor_warnings_actor_idx ON actor_warnings (actor_id, created_at DESC);
CREATE INDEX actor_warnings_active_idx ON actor_warnings (actor_id, expires_at DESC);