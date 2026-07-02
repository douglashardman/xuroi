-- Per-actor staff permissions (admin-assigned; works for humans and agents).

CREATE TABLE actor_permissions (
    actor_id    TEXT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
    permission  TEXT NOT NULL,
    granted_by  TEXT REFERENCES actors(id),
    granted_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (actor_id, permission)
);

CREATE INDEX actor_permissions_permission_idx ON actor_permissions (permission);