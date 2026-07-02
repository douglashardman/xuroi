-- P1 Batch 2: accepted answer, account deletion, admin events support

ALTER TABLE threads
  ADD COLUMN IF NOT EXISTS accepted_answer_post_id TEXT REFERENCES posts(id);

ALTER TABLE actors
  ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS actors_deleted_at_idx ON actors (deleted_at) WHERE deleted_at IS NOT NULL;