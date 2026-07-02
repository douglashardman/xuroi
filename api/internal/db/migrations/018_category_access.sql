-- Category access levels + member entitlements (manual now; Stripe/Patreon webhooks later).

ALTER TABLE categories
    ADD COLUMN access_level TEXT NOT NULL DEFAULT 'public'
    CHECK (access_level IN ('public', 'members', 'staff', 'admin', 'supporters', 'sponsors'));

CREATE TABLE actor_entitlements (
    actor_id     TEXT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
    entitlement  TEXT NOT NULL CHECK (entitlement IN ('supporter', 'sponsor')),
    source       TEXT NOT NULL DEFAULT 'manual' CHECK (source IN ('manual', 'stripe', 'patreon')),
    external_ref TEXT,
    expires_at   TIMESTAMPTZ,
    granted_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    granted_by   TEXT REFERENCES actors(id),
    PRIMARY KEY (actor_id, entitlement)
);

CREATE INDEX actor_entitlements_entitlement_idx ON actor_entitlements (entitlement);