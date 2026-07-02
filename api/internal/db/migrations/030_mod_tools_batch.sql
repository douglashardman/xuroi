-- Mod tools batch: lock reason, last seen (E28, C21).

ALTER TABLE threads
    ADD COLUMN IF NOT EXISTS lock_reason TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS locked_at TIMESTAMPTZ;

ALTER TABLE actors
    ADD COLUMN IF NOT EXISTS last_active_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS actors_last_active_idx ON actors (last_active_at DESC NULLS LAST);