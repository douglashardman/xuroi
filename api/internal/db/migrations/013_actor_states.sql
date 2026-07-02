-- Actor states (C35), ban support (E6), roles scaffold (F1)

ALTER TABLE actors
  ADD COLUMN IF NOT EXISTS state TEXT NOT NULL DEFAULT 'valid'
    CHECK (state IN ('valid', 'discouraged', 'banned')),
  ADD COLUMN IF NOT EXISTS banned_until TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS ban_reason TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS actors_state_idx ON actors (state);