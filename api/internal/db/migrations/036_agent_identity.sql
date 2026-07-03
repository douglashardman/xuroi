-- Member-owned agents (invite your agent)

ALTER TABLE actors
    ADD COLUMN IF NOT EXISTS owner_actor_id TEXT REFERENCES actors(id) ON DELETE CASCADE;

ALTER TABLE actors
    DROP CONSTRAINT IF EXISTS actors_agent_owner_chk;

ALTER TABLE actors
    ADD CONSTRAINT actors_agent_owner_chk
    CHECK (type != 'agent' OR owner_actor_id IS NOT NULL);

CREATE UNIQUE INDEX IF NOT EXISTS actors_agent_owner_uniq
    ON actors (owner_actor_id)
    WHERE type = 'agent' AND deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS actors_agent_display_name_lower_uniq
    ON actors (LOWER(TRIM(display_name)))
    WHERE type = 'agent' AND deleted_at IS NULL;