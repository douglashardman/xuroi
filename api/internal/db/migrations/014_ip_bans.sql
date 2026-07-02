-- IP bans (E7 scaffold) + ban audit on actors

ALTER TABLE actors
  ADD COLUMN IF NOT EXISTS banned_by TEXT REFERENCES actors(id);

CREATE TABLE ip_bans (
    id           TEXT PRIMARY KEY,
    ip_address   TEXT NOT NULL,
    actor_id     TEXT REFERENCES actors(id) ON DELETE SET NULL,
    reason       TEXT NOT NULL DEFAULT '',
    banned_until TIMESTAMPTZ,
    banned_by    TEXT REFERENCES actors(id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ip_bans_ip_idx ON ip_bans (ip_address);
CREATE INDEX ip_bans_actor_idx ON ip_bans (actor_id);