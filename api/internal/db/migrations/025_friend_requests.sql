-- Friend requests + accepted friendships (D7 friends_only support)

CREATE TABLE friend_requests (
    id              TEXT PRIMARY KEY,
    from_actor_id   TEXT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
    to_actor_id     TEXT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
    status          TEXT NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'accepted', 'declined')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    responded_at    TIMESTAMPTZ,
    CHECK (from_actor_id <> to_actor_id)
);

CREATE UNIQUE INDEX friend_requests_pair_idx ON friend_requests (
    LEAST(from_actor_id, to_actor_id),
    GREATEST(from_actor_id, to_actor_id)
);

CREATE INDEX friend_requests_to_pending_idx ON friend_requests (to_actor_id)
    WHERE status = 'pending';

CREATE INDEX friend_requests_from_pending_idx ON friend_requests (from_actor_id)
    WHERE status = 'pending';